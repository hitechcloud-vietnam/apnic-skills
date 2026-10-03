package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestFetchText_RequestError covers the doHTTPRequest-error branch.
func TestFetchText_RequestError(t *testing.T) {
	c := NewClient(
		WithHTTPClient(&http.Client{Transport: dialErrRoundTripper{}}),
		WithStatsBaseURL("http://x/"), WithJitter(0, 0), WithCacheTTL(0),
		WithMaxConcurrentDownloads(0))
	if _, err := c.FetchText(context.Background(), "http://x/y"); err == nil {
		t.Error("expected request error")
	}
}

// TestFetchText_BadStatus covers the non-200 status branch.
func TestFetchText_BadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c := NewClient(WithHTTPClient(srv.Client()), WithStatsBaseURL(srv.URL+"/"),
		WithJitter(0, 0), WithCacheTTL(0), WithMaxConcurrentDownloads(0))
	if _, err := c.FetchText(context.Background(), srv.URL+"/x"); err == nil {
		t.Error("expected bad-status error")
	}
}

// TestFetchText_GzipInitError covers the gzip.NewReader-error branch: a .gz URL
// whose body is not valid gzip.
func TestFetchText_GzipInitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-gzip-at-all"))
	}))
	defer srv.Close()
	c := NewClient(WithHTTPClient(srv.Client()), WithStatsBaseURL(srv.URL+"/"),
		WithJitter(0, 0), WithCacheTTL(0), WithMaxConcurrentDownloads(0))
	if _, err := c.FetchText(context.Background(), srv.URL+"/x.gz"); err == nil {
		t.Error("expected gzip init error")
	}
}

// TestFetchText_ReadError covers the io.Copy-error branch via errorRoundTripper
// (200 + body that errors on read).
func TestFetchText_ReadError(t *testing.T) {
	c := NewClient(
		WithHTTPClient(&http.Client{Transport: errorRoundTripper{}}),
		WithStatsBaseURL("http://x/"), WithJitter(0, 0), WithCacheTTL(0),
		WithMaxConcurrentDownloads(0))
	if _, err := c.FetchText(context.Background(), "http://x/y"); err == nil {
		t.Error("expected read error")
	}
}

// TestFetchText_GzipDecompress covers the successful gzip decompression path of
// fetchText for a .gz URL with valid gzip content.
func TestFetchText_GzipDecompress(t *testing.T) {
	plain := "hello-gzip-fetchtext"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		w.Write(gzipBytes(plain))
	}))
	defer srv.Close()
	c := NewClient(WithHTTPClient(srv.Client()), WithStatsBaseURL(srv.URL+"/"),
		WithJitter(0, 0), WithCacheTTL(0), WithMaxConcurrentDownloads(0))
	got, err := c.FetchText(context.Background(), srv.URL+"/x.gz")
	if err != nil {
		t.Fatalf("fetchText: %v", err)
	}
	if got != plain {
		t.Errorf("got %q, want %q", got, plain)
	}
}
