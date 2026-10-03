package transport

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// FetchText performs an HTTP GET request and returns the response body as a string.
// If the response is gzip-compressed (either via Content-Encoding or a .gz URL
// suffix, as used by APNIC's archived historical files), it is transparently
// decompressed.
func (c *Client) FetchText(ctx context.Context, url string) (string, error) {
	resp, err := c.DoHTTPRequest(ctx, "GET", url, "text/plain")
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d for URL: %s", resp.StatusCode, url)
	}

	body := resp.Body
	// APNIC archives historical files as .gz with no Content-Encoding header, so
	// detect by URL suffix too. Both paths are decompressed transparently.
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") || strings.HasSuffix(url, ".gz") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("gzip init failed: %w", err)
		}
		defer gz.Close()
		body = gz
	}

	var buf strings.Builder
	if _, err := io.Copy(&buf, body); err != nil {
		return "", fmt.Errorf("read response failed: %w", err)
	}
	return buf.String(), nil
}
