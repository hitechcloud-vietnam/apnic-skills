package transport

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// downloadConfig holds chunked-download tunables. APNIC FTP throttles large
// files per-connection to ~8-18 KB/s, so downloading a multi-megabyte file over
// a single connection routinely exceeds the client timeout. Splitting the file
// into N parallel Range requests multiplies throughput because each connection
// is throttled independently.
//
// maxConcurrent caps how many Range requests run at once; it does NOT cap the
// number of ranges. A 50 MB IRR dump split into 2 MB chunks yields ~25 ranges
// that stream through maxConcurrent workers, keeping each individual request
// small enough to finish well inside the per-chunk timeout.
type downloadConfig struct {
	maxConcurrent int           // parallel Range requests in flight; <=1 disables chunking
	chunkSize     int64         // explicit bytes per chunk; 0 = use defaultTargetChunkSize
	targetChunk   int64         // default bytes per chunk when chunkSize==0 (default 2 MiB)
	timeout       time.Duration // per-chunk request timeout; 0 = inherit httpClient
	minSize       int64         // files smaller than this skip chunking
}

// defaultTargetChunkSize is the implicit chunk size when neither chunkSize nor
// targetChunk is set. At APNIC's ~22 KB/s per-connection throttle, 2 MiB takes
// ~90 s — comfortably under the recommended 5 min per-chunk timeout.
const defaultTargetChunkSize int64 = 2 * 1024 * 1024

// chunkRange is an inclusive [start, end] byte range within a file.
type chunkRange struct{ start, end int64 }

// FetchReader returns a streaming, decompressed io.Reader for the content at
// url. When the server advertises Accept-Ranges and the file is large enough,
// it downloads the file as parallel Range requests and merges them into the
// returned reader; otherwise it falls back to a single GET. The reader is
// gzip-decompressed when the URL ends in .gz or the response carries
// Content-Encoding: gzip.
//
// All HTTP requests (probe + each chunk) go through DoHTTPRequest, so stealth
// headers, rate-limiting and jitter apply uniformly.
func (c *Client) FetchReader(ctx context.Context, url string) (io.Reader, error) {
	if c.downloadCfg.maxConcurrent > 1 {
		r, err := c.downloadChunked(ctx, url)
		if err != errChunkingUnsupported {
			return r, err
		}
		// Server doesn't support Range or transport-gzip would break chunking —
		// fall through to single-connection streaming.
	}
	return c.singleStream(ctx, url)
}

// FetchTextStr returns the full decompressed body at url as a string. It is the
// string-oriented counterpart to FetchReader for parsers that consume a string.
func (c *Client) FetchTextStr(ctx context.Context, url string) (string, error) {
	r, err := c.FetchReader(ctx, url)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if _, err := io.Copy(&buf, r); err != nil {
		return "", fmt.Errorf("read response failed: %w", err)
	}
	return buf.String(), nil
}

// errChunkingUnsupported signals that the server does not support safe Range
// chunking (no Accept-Ranges, returned 200 to a Range request, or uses
// transport-layer gzip on a non-.gz URL). Callers fall back to single-stream.
var errChunkingUnsupported = fmt.Errorf("chunked download unsupported by server")

// downloadChunked probes the URL for Range support, splits the content into
// parallel chunks, and returns a merged (and gzip-decompressed if needed)
// io.Reader. Returns errChunkingUnsupported when the server cannot be chunked.
func (c *Client) downloadChunked(ctx context.Context, url string) (io.Reader, error) {
	total, supportsRange, gz, err := c.probeRange(ctx, url)
	if err != nil {
		return nil, err
	}
	if !supportsRange || total < c.downloadCfg.minSize {
		return nil, errChunkingUnsupported
	}
	// Transport-layer gzip on a non-.gz URL would be corrupted by Range cuts;
	// only .gz files (content is gzip, not transport encoding) are safe.
	if gz && !strings.HasSuffix(url, ".gz") {
		return nil, errChunkingUnsupported
	}

	ranges := planChunks(total, c.downloadCfg)
	if len(ranges) <= 1 {
		return nil, errChunkingUnsupported
	}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		data := make([][]byte, len(ranges))
		errs := make([]error, len(ranges))

		var wg sync.WaitGroup
		sem := make(chan struct{}, c.effectiveConcurrency(len(ranges)))
		for i, r := range ranges {
			wg.Add(1)
			go func(i int, r chunkRange) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				b, e := c.fetchChunkWithRetry(ctx, url, r, 2)
				data[i] = b
				errs[i] = e
			}(i, r)
		}
		wg.Wait()

		for i := range ranges {
			if errs[i] != nil {
				pw.CloseWithError(fmt.Errorf("chunk %d (%d-%d): %w", i, ranges[i].start, ranges[i].end, errs[i]))
				return
			}
			if _, e := pw.Write(data[i]); e != nil {
				return
			}
		}
	}()

	var reader io.Reader = pr
	if gz || strings.HasSuffix(url, ".gz") {
		gzr, e := gzip.NewReader(pr)
		if e != nil {
			pr.CloseWithError(e)
			return nil, fmt.Errorf("gzip init failed: %w", e)
		}
		reader = &gzipClosingReader{gz: gzr, closer: pr}
	}
	return reader, nil
}

// gzipClosingReader wraps a gzip.Reader so that closing the gzip reader also
// closes the underlying pipe reader, ensuring the writer goroutine is reaped
// when the consumer stops reading early.
type gzipClosingReader struct {
	gz     *gzip.Reader
	closer io.Closer
}

func (g *gzipClosingReader) Read(p []byte) (int, error) { return g.gz.Read(p) }
func (g *gzipClosingReader) Close() error {
	g.gz.Close()
	return g.closer.Close()
}

// effectiveConcurrency returns the in-flight worker cap for a chunk count.
// planChunks may produce up to 64 small ranges for a large file; workers run
// at most maxConcurrent (hard-capped at 16 to stay polite to APNIC), draining
// the remaining ranges as earlier ones finish.
func (c *Client) effectiveConcurrency(chunks int) int {
	n := c.downloadCfg.maxConcurrent
	if n < 1 {
		return 1
	}
	if n > 16 {
		n = 16
	}
	if n > chunks {
		n = chunks
	}
	return n
}

// probeRange issues a GET with Range: bytes=0-0 to learn the total content
// length, whether the server honors Range, and whether the body is gzip. A 206
// response with a Content-Range header means Range is supported; a 200 means it
// is not (the server ignored the Range header).
func (c *Client) probeRange(ctx context.Context, url string) (total int64, supportsRange bool, gzipped bool, err error) {
	hdr := http.Header{}
	hdr.Set("Range", "bytes=0-0")
	resp, err := c.DoHTTPRequest(ctx, "GET", url, "text/plain, */*", hdr)
	if err != nil {
		return 0, false, false, fmt.Errorf("range probe failed: %w", err)
	}
	defer resp.Body.Close()
	// Drain the 1-byte body so the connection can be reused.
	io.Copy(io.Discard, io.LimitReader(resp.Body, 64))

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return 0, false, false, fmt.Errorf("range probe status: %d", resp.StatusCode)
	}

	gzipped = strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") ||
		strings.HasSuffix(url, ".gz")

	if resp.StatusCode == http.StatusPartialContent {
		supportsRange = true
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			// Format: "bytes 0-0/TOTAL"
			if idx := strings.LastIndex(cr, "/"); idx >= 0 {
				if n, e := strconv.ParseInt(cr[idx+1:], 10, 64); e == nil {
					total = n
				}
			}
		}
	} else {
		// 200: server ignored Range. Total is Content-Length if present.
		supportsRange = false
		total = resp.ContentLength
	}
	return total, supportsRange, gzipped, nil
}

// fetchChunkWithRetry fetches one byte range, retrying on transient network
// errors or 5xx up to retries times. It returns the raw (still-compressed)
// chunk bytes. On a deadline-exceeded (a stuck/slow connection) it degrades to
// splitting the range into sub-ranges fetched in parallel, sidestepping the
// stalled connection.
func (c *Client) fetchChunkWithRetry(ctx context.Context, url string, r chunkRange, retries int) ([]byte, error) {
	b, err := c.fetchChunkRaw(ctx, url, r, retries)
	if err == nil {
		return b, nil
	}
	// Degrade on a deadline/stall: split the range in half and fetch each half
	// concurrently on fresh connections. This recovers from a single stuck
	// TCP connection without abandoning the whole download.
	if !isDeadlineError(err) || r.end-r.start+1 < 64*1024 {
		return nil, fmt.Errorf("fetch chunk %d-%d after %d retries: %w", r.start, r.end, retries, err)
	}
	mid := r.start + (r.end-r.start)/2
	halfA := chunkRange{start: r.start, end: mid}
	halfB := chunkRange{start: mid + 1, end: r.end}
	type res struct {
		b   []byte
		err error
	}
	rc := make(chan res, 2)
	for _, sub := range []chunkRange{halfA, halfB} {
		go func(s chunkRange) {
			sb, se := c.fetchChunkRaw(ctx, url, s, retries)
			rc <- res{sb, se}
		}(sub)
	}
	var parts [][]byte
	for i := 0; i < 2; i++ {
		res := <-rc
		if res.err != nil {
			return nil, fmt.Errorf("fetch chunk %d-%d (degraded split %d-%d): %w", r.start, r.end, halfA.start, halfB.end, res.err)
		}
		parts = append(parts, res.b)
	}
	return bytesJoin(parts...), nil
}

// bytesJoin concatenates byte slices. Used to reassemble degraded sub-chunks.
func bytesJoin(parts ...[]byte) []byte {
	var n int
	for _, p := range parts {
		n += len(p)
	}
	out := make([]byte, 0, n)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// isDeadlineError reports whether err is a context-deadline/timeout error,
// indicating a stalled connection rather than a hard failure.
func isDeadlineError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "context deadline exceeded") ||
		strings.Contains(s, "Client.Timeout") ||
		errors.Is(err, context.DeadlineExceeded)
}

// fetchChunkRaw performs a single range GET with up to `retries` retries on
// transient errors. It is the inner loop of fetchChunkWithRetry.
func (c *Client) fetchChunkRaw(ctx context.Context, url string, r chunkRange, retries int) ([]byte, error) {
	hdr := http.Header{}
	hdr.Set("Range", fmt.Sprintf("bytes=%d-%d", r.start, r.end))
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		chunkCtx := ctx
		var cancel context.CancelFunc
		if c.downloadCfg.timeout > 0 {
			chunkCtx, cancel = context.WithTimeout(ctx, c.downloadCfg.timeout)
		}
		// cancel must fire only AFTER the body is fully read; calling it
		// immediately after Do returns would abort the in-flight io.ReadAll
		// with "context canceled". Defer it to body-drain completion.
		resp, err := c.DoHTTPRequest(chunkCtx, "GET", url, "text/plain, */*", hdr)
		if err != nil {
			if cancel != nil {
				cancel()
			}
			lastErr = err
			continue
		}
		// A 200 here means the server ignored Range — chunking is unsafe.
		if resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			if cancel != nil {
				cancel()
			}
			return nil, errChunkingUnsupported
		}
		if resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			if cancel != nil {
				cancel()
			}
			lastErr = fmt.Errorf("chunk status %d", resp.StatusCode)
			if resp.StatusCode >= 500 {
				continue // retry server errors
			}
			return nil, lastErr // 4xx: don't retry
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if cancel != nil {
			cancel()
		}
		if err != nil {
			lastErr = err
			continue
		}
		return body, nil
	}
	return nil, lastErr
}

// planChunks splits a total byte length into contiguous inclusive ranges.
// chunkSize > 0 aligns ranges to chunkSize (last absorbs the remainder).
// Otherwise ranges are sized by targetChunk (default 2 MiB) so that each
// individual Range request is small enough to finish inside the per-chunk
// timeout under APNIC's per-connection throttle; maxConcurrent only governs
// how many run at once, not how many exist. The range count is at least 1 and
// at most 64; at least maxConcurrent ranges are produced (when total allows)
// so parallelism is not starved.
func planChunks(total int64, cfg downloadConfig) []chunkRange {
	if total <= 0 {
		return nil
	}
	per := cfg.chunkSize
	if per <= 0 {
		per = cfg.targetChunk
	}
	if per <= 0 {
		per = defaultTargetChunkSize
	}
	n := (total + per - 1) / per
	// Ensure at least maxConcurrent ranges (when total is large enough) so the
	// worker pool is fully utilized. The floor guards total >= mc, so n never
	// exceeds total here.
	if mc := int64(cfg.maxConcurrent); mc > 1 && n < mc && total >= mc {
		n = mc
	}
	if n > 64 {
		n = 64
	}
	base := total / n
	var ranges []chunkRange
	for i := int64(0); i < n; i++ {
		start := i * base
		end := start + base - 1
		if i == n-1 {
			end = total - 1
		}
		ranges = append(ranges, chunkRange{start: start, end: end})
	}
	return ranges
}

// singleStream performs a single GET and returns a streaming, gzip-decompressed
// reader. It is the fallback for servers that do not support Range and for
// files below the chunking threshold. Unlike fetchText it does not buffer the
// whole body into a string.
func (c *Client) singleStream(ctx context.Context, url string) (io.Reader, error) {
	resp, err := c.DoHTTPRequest(ctx, "GET", url, "text/plain")
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d for URL: %s", resp.StatusCode, url)
	}
	body := io.Reader(resp.Body)
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") || strings.HasSuffix(url, ".gz") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("gzip init failed: %w", err)
		}
		return &gzipClosingReader{gz: gz, closer: resp.Body}, nil
	}
	return body, nil
}
