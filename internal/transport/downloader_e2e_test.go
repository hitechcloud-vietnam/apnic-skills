//go:build e2e

// Real APNIC end-to-end tests for chunked download. These hit the live
// ftp.apnic.net / thyme.apnic.net endpoints and verify that large files which
// previously timed out under a single connection now download successfully via
// parallel Range requests.
//
// Run with:  go test -tags=e2e -run E2E -timeout 15m
package transport

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// e2eClient builds a client tuned for real APNIC downloads: a generous HTTP
// timeout for the single-connection fallback path, 4-way chunking (the default),
// and a per-chunk timeout long enough to survive APNIC's ~8-18 KB/s throttle.
func e2eClient() *Client {
	return NewClient(
		WithCacheTTL(0),
		WithHTTPClient(&http.Client{Timeout: 10 * time.Minute}),
		WithMaxConcurrentDownloads(4),
		WithDownloadTimeout(5*time.Minute),
	)
}

// TestE2E_DelegatedChunked downloads the latest delegated stats (a ~4MB file
// throttled to ~8-18 KB/s per connection) and asserts it parses to a non-empty
// entry set, demonstrating chunked download succeeds where a single connection
// would exceed a typical timeout.
func TestE2E_DelegatedChunked(t *testing.T) {
	c := e2eClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	start := time.Now()
	r, err := c.FetchDelegatedResult(ctx, "")
	if err != nil {
		t.Fatalf("FetchDelegatedResult: %v", err)
	}
	if len(r.Entries) == 0 {
		t.Fatal("expected non-empty delegated entries")
	}
	t.Logf("delegated: %d entries in %v", len(r.Entries), time.Since(start))
}

// TestE2E_ExtendedChunked downloads the latest extended delegated stats via
// chunked download.
func TestE2E_ExtendedChunked(t *testing.T) {
	c := e2eClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	start := time.Now()
	r, err := c.FetchExtendedResult(ctx, "")
	if err != nil {
		t.Fatalf("FetchExtendedResult: %v", err)
	}
	if len(r.Entries) == 0 {
		t.Fatal("expected non-empty extended entries")
	}
	t.Logf("extended: %d entries in %v", len(r.Entries), time.Since(start))
}

// TestE2E_IRRChunked downloads the inetnum IRR database dump (a multi-MB .gz
// RPSL file) via chunked download and asserts it parses to objects.
func TestE2E_IRRChunked(t *testing.T) {
	c := e2eClient()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	start := time.Now()
	r, err := c.FetchIRRDatabase(ctx, "inetnum")
	if err != nil {
		t.Fatalf("FetchIRRDatabase: %v", err)
	}
	if len(r.Objects) == 0 {
		t.Fatal("expected non-empty IRR objects")
	}
	t.Logf("irr inetnum: %d objects in %v", len(r.Objects), time.Since(start))
}
