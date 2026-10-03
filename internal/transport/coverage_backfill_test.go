package transport

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestBuildThymeURL covers BuildThymeURL including the empty-source default
// branch (source == "" -> "current") and the TrimRight behavior.
func TestBuildThymeURL(t *testing.T) {
	cases := []struct {
		name             string
		base, src, file  string
		want             string
	}{
		{"empty source defaults to current", "https://thyme.apnic.net/", "", "data-summary", "https://thyme.apnic.net/current/data-summary"},
		{"explicit au source", "https://thyme.apnic.net", "au", "data-raw-table", "https://thyme.apnic.net/au/data-raw-table"},
		{"hk source trailing slash trimmed", "https://thyme.apnic.net//", "hk", "data-spar", "https://thyme.apnic.net/hk/data-spar"},
	}
	for _, c := range cases {
		got := BuildThymeURL(c.base, c.src, c.file)
		if got != c.want {
			t.Errorf("%s: BuildThymeURL(%q,%q,%q) = %q, want %q", c.name, c.base, c.src, c.file, got, c.want)
		}
	}
}

// TestSourceOrDefault covers both branches: non-empty source returned as-is,
// empty source falls back to def.
func TestSourceOrDefault(t *testing.T) {
	if got := SourceOrDefault("au", "current"); got != "au" {
		t.Errorf("non-empty source: got %q, want au", got)
	}
	if got := SourceOrDefault("", "current"); got != "current" {
		t.Errorf("empty source fallback: got %q, want current", got)
	}
	if got := SourceOrDefault("", ""); got != "" {
		t.Errorf("both empty: got %q, want empty", got)
	}
}

// TestBuildRRDPNotificationURL covers the RRDP notification URL builder and
// its trailing-slash trimming.
func TestBuildRRDPNotificationURL(t *testing.T) {
	if got := BuildRRDPNotificationURL("https://rrdp.apnic.net"); got != "https://rrdp.apnic.net/notification.xml" {
		t.Errorf("no trailing slash: got %q", got)
	}
	if got := BuildRRDPNotificationURL("https://rrdp.apnic.net/"); got != "https://rrdp.apnic.net/notification.xml" {
		t.Errorf("trailing slash trimmed: got %q", got)
	}
	if got := BuildRRDPNotificationURL("https://rrdp.apnic.net///"); got != "https://rrdp.apnic.net/notification.xml" {
		t.Errorf("multi trailing slashes trimmed: got %q", got)
	}
}

// TestCacheGetSet_NonNilCache covers the non-nil cache path of the exported
// CacheGet/CacheSet methods (the cache type itself is unexported; existing
// cache_test.go exercises c.cache.get/set directly, leaving the Client method
// wrappers at 66.7%).
func TestCacheGetSet_NonNilCache(t *testing.T) {
	c := NewClient() // NewClient initializes a non-nil c.cache (30m TTL)

	// Miss on a fresh client cache.
	if _, ok := c.CacheGet("nope"); ok {
		t.Error("expected cache miss on CacheGet")
	}

	// Set then Get round-trip through the exported methods.
	c.CacheSet("k1", "v1")
	val, ok := c.CacheGet("k1")
	if !ok {
		t.Fatal("expected cache hit after CacheSet")
	}
	if val.(string) != "v1" {
		t.Errorf("CacheGet value = %v, want v1", val)
	}
}

// TestFetchTextStr_ReadError_Backfill covers the io.Copy-error branch
// (downloader.go:71) via errorRoundTripper, which returns 200 but a body that
// errors on read. A same-named test exists in downloader_test.go; this variant
// additionally asserts the error message substring.
func TestFetchTextStr_ReadError_Backfill(t *testing.T) {
	c := NewClient(WithHTTPClient(&http.Client{Transport: errorRoundTripper{}}))
	_, err := c.FetchTextStr(context.Background(), "http://x/y")
	if err == nil {
		t.Fatal("expected error from FetchTextStr when body read fails")
	}
	if !strings.Contains(err.Error(), "read response failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestFetchTextStr_OK covers the happy path of FetchTextStr (returns full body
// as string), ensuring the success return statement is exercised.
func TestFetchTextStr_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "hello-str")
	}))
	defer srv.Close()
	c := NewClient(
		WithHTTPClient(srv.Client()),
		WithMaxConcurrentDownloads(1),
	)
	got, err := c.FetchTextStr(context.Background(), srv.URL+"/x")
	if err != nil {
		t.Fatalf("FetchTextStr OK: %v", err)
	}
	if got != "hello-str" {
		t.Errorf("FetchTextStr body = %q, want hello-str", got)
	}
}

// TestFetchTextStr_FetchReaderError covers the FetchReader-error early-return
// branch of FetchTextStr (downloader.go:67). When FetchReader itself returns an
// error (here: a small non-gzip body at a .gz URL makes singleStream's
// gzip.NewReader fail), FetchTextStr must propagate it without touching io.Copy.
func TestFetchTextStr_FetchReaderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "not-actually-gzip")
	}))
	defer srv.Close()
	c := NewClient(
		WithHTTPClient(srv.Client()),
		WithMaxConcurrentDownloads(1), // singleStream path -> gzip init fails
	)
	_, err := c.FetchTextStr(context.Background(), srv.URL+"/data.gz")
	if err == nil {
		t.Fatal("expected error from FetchTextStr when FetchReader fails")
	}
	if !strings.Contains(err.Error(), "gzip init failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestSingleStream_GzipInitError covers singleStream's gzip-init-failed branch
// (downloader.go:400). WithMaxConcurrentDownloads(1) sets maxConcurrent<=1, so
// FetchReader skips downloadChunked entirely and calls singleStream directly
// (downloader.go:52: the `maxConcurrent > 1` guard is false). The .gz URL
// suffix then makes singleStream attempt gzip.NewReader on a non-gzip body,
// triggering the gzip init failed error.
func TestSingleStream_GzipInitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		// Small non-gzip body with a .gz URL -> gzip.NewReader fails.
		io.WriteString(w, "not-actually-gzip")
	}))
	defer srv.Close()
	c := NewClient(
		WithHTTPClient(srv.Client()),
		WithMaxConcurrentDownloads(1), // force singleStream path
	)
	r, err := c.FetchReader(context.Background(), srv.URL+"/data.gz")
	if err == nil {
		// If a reader came back, draining it must surface the gzip error.
		if r != nil {
			_, _ = io.ReadAll(r)
		}
		t.Fatal("expected gzip init error from singleStream on non-gzip .gz body")
	}
	if !strings.Contains(err.Error(), "gzip init failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestDoHTTPRequest_InvalidURL covers the NewRequestWithContext-error branch
// (stealth.go:118) by passing a URL with a control character (\x7f), which
// http.NewRequestWithContext rejects with "invalid control character in URL".
func TestDoHTTPRequest_InvalidURL(t *testing.T) {
	c := NewClient()
	_, err := c.DoHTTPRequest(context.Background(), "GET", "http://x/y\x7f", "text/plain")
	if err == nil {
		t.Fatal("expected error from DoHTTPRequest with invalid URL")
	}
	if !strings.Contains(err.Error(), "invalid control character") {
		t.Errorf("unexpected error: %v", err)
	}
}
