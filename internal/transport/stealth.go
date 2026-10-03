package transport

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// defaultBrowserUA is a mainstream Chrome User-Agent string used when stealth is
// enabled, so requests resemble a real browser rather than an SDK/bot.
const defaultBrowserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// rateLimiter wraps a token-bucket rate limiter so the Client can cap request
// frequency across all HTTP and whois outlets. A nil rateLimiter means no limit.
type rateLimiter struct {
	limiter *rate.Limiter
}

// newRateLimiter builds a rateLimiter allowing perSecond requests per second
// with a burst of 1. Returns nil if perSecond <= 0.
func newRateLimiter(perSecond float64) *rateLimiter {
	if perSecond <= 0 {
		return nil
	}
	return &rateLimiter{limiter: rate.NewLimiter(rate.Limit(perSecond), 1)}
}

// wait blocks until a token is available or ctx is cancelled. Safe to call on a
// nil receiver (no-op).
func (r *rateLimiter) wait(ctx context.Context) error {
	if r == nil || r.limiter == nil {
		return nil
	}
	return r.limiter.Wait(ctx)
}

// applyBrowserHeaders sets a full set of browser-like request headers on req
// when stealth is enabled, making the request indistinguishable from a real
// browser visit at the header level. When stealth is disabled, only User-Agent
// and Accept are set (backward-compatible with pre-stealth behavior).
//
// accept is the Accept header value appropriate to the caller (e.g.
// "text/plain" for stats, "application/rdap+json, application/json" for RDAP).
func (c *Client) applyBrowserHeaders(req *http.Request, accept string) {
	if !c.stealth {
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", accept)
		return
	}

	req.Header.Set("User-Agent", c.browserUA)
	req.Header.Set("Accept", accept)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	// Explicit Accept-Encoding: gzip. Note: Go's http.Transport only auto-handles
	// gzip when the header is NOT set by the caller; once we set it explicitly,
	// Transport leaves the response untouched and fetchText's existing
	// Content-Encoding branch decompresses it. The .gz URL-suffix branch is
	// unaffected. The decompression decision happens once at the fetchText entry,
	// so there is no double-decompression.
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="124", "Not.A/Brand";v="99"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Connection", "keep-alive")
}

// Jitter sleeps for a random duration in [jitterMin, jitterMax) when stealth is
// enabled and the interval is positive. It returns immediately if ctx is
// cancelled, so a long jitter never blocks a cancelled context.
//
// As a testing affordance, setting the APNIC_NO_JITTER environment variable to a
// non-empty value disables jitter entirely. This lets the test suite run fast
// without each test having to opt out. Production callers never set it.
func (c *Client) Jitter(ctx context.Context) {
	if !c.stealth || c.jitterMin <= 0 || c.jitterMax <= c.jitterMin {
		return
	}
	if v, _ := os.LookupEnv("APNIC_NO_JITTER"); v != "" {
		return
	}
	span := c.jitterMax - c.jitterMin
	d := c.jitterMin + time.Duration(c.rand.Int63n(int64(span)))
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

// WaitRateLimit blocks until the global rate limiter allows a request, or ctx is
// cancelled. No-op when no limiter is configured.
func (c *Client) WaitRateLimit(ctx context.Context) error {
	if c.rateLimiter == nil {
		return nil
	}
	return c.rateLimiter.wait(ctx)
}

// DoHTTPRequest is the unified HTTP execution outlet for the SDK. It builds the
// request, applies browser-mimicry headers, waits for the rate limiter and
// jitter, then performs the request. It does NOT read or decompress the body —
// callers (FetchText, doRDAPRequestAt) retain their own body handling.
//
// All HTTP traffic from the SDK goes through this method, so stealth/rate-limit
// behavior is applied consistently. The optional extra headers (e.g. Range) are
// applied after the browser headers so callers can inject request-specific
// headers without bypassing stealth.
func (c *Client) DoHTTPRequest(ctx context.Context, method, url, accept string, extra ...http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	c.applyBrowserHeaders(req, accept)
	for _, h := range extra {
		for k, vs := range h {
			for _, v := range vs {
				req.Header.Set(k, v)
			}
		}
	}
	if err := c.WaitRateLimit(ctx); err != nil {
		return nil, err
	}
	c.Jitter(ctx)
	return c.httpClient.Do(req)
}

// randSource is a small helper held on the Client so jitter is deterministic in
// tests (seeded per-Client) without polluting the global rand source.
type randSource struct {
	mu sync.Mutex
	r  *rand.Rand
}

func (rs *randSource) Int63n(n int64) int64 {
	if n <= 0 {
		return 0
	}
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.r.Int63n(n)
}
