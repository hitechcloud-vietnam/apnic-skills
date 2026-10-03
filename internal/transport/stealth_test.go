package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// capturingHandler is re-exported from testutil in test_helpers_test.go.

func TestApplyBrowserHeaders_StealthOn(t *testing.T) {
	c := NewClient()
	srv, hdr := capturingHandler(t, http.StatusOK, "ok")
	defer srv.Close()
	c.statsBaseURL = srv.URL + "/"
	if _, err := c.FetchText(context.Background(), srv.URL+"/x"); err != nil {
		t.Fatal(err)
	}
	if got := hdr.Get("User-Agent"); !strings.Contains(got, "Mozilla") {
		t.Errorf("stealth UA should look like a browser, got %q", got)
	}
	for _, want := range []string{"Accept-Language", "Accept-Encoding", "Sec-Fetch-Site", "Sec-Ch-Ua", "Upgrade-Insecure-Requests"} {
		if hdr.Get(want) == "" {
			t.Errorf("stealth should set %s header", want)
		}
	}
	if hdr.Get("Accept-Encoding") != "gzip" {
		t.Errorf("Accept-Encoding = %q, want gzip", hdr.Get("Accept-Encoding"))
	}
}

func TestApplyBrowserHeaders_StealthOff(t *testing.T) {
	c := NewClient(WithStealth(false))
	srv, hdr := capturingHandler(t, http.StatusOK, "ok")
	defer srv.Close()
	if _, err := c.FetchText(context.Background(), srv.URL+"/x"); err != nil {
		t.Fatal(err)
	}
	if got := hdr.Get("User-Agent"); got != "APNIC-Go-SDK/1.0 (security)" {
		t.Errorf("stealth off should use SDK UA, got %q", got)
	}
	// stealth off must NOT send browser-only headers
	for _, unwanted := range []string{"Sec-Fetch-Site", "Sec-Ch-Ua"} {
		if hdr.Get(unwanted) != "" {
			t.Errorf("stealth off should not send %s", unwanted)
		}
	}
}

func TestWithBrowserUserAgent(t *testing.T) {
	c := NewClient(WithBrowserUserAgent("MyBrowser/2.0"))
	srv, hdr := capturingHandler(t, http.StatusOK, "ok")
	defer srv.Close()
	if _, err := c.FetchText(context.Background(), srv.URL+"/x"); err != nil {
		t.Fatal(err)
	}
	if got := hdr.Get("User-Agent"); got != "MyBrowser/2.0" {
		t.Errorf("custom browser UA = %q, want MyBrowser/2.0", got)
	}
}

func TestWithUserAgent_StealthOffBackwardCompat(t *testing.T) {
	// WithUserAgent still applies when stealth is disabled.
	c := NewClient(WithStealth(false), WithUserAgent("legacy/1.0"))
	srv, hdr := capturingHandler(t, http.StatusOK, "ok")
	defer srv.Close()
	if _, err := c.FetchText(context.Background(), srv.URL+"/x"); err != nil {
		t.Fatal(err)
	}
	if got := hdr.Get("User-Agent"); got != "legacy/1.0" {
		t.Errorf("UA = %q, want legacy/1.0", got)
	}
}

func TestJitter_RangeAndDeterminism(t *testing.T) {
	// Re-enable jitter locally (init() disabled it for the suite).
	t.Setenv("APNIC_NO_JITTER", "")
	c := NewClient(WithJitter(5*time.Millisecond, 10*time.Millisecond))
	start := time.Now()
	c.Jitter(context.Background())
	elapsed := time.Since(start)
	if elapsed < 5*time.Millisecond {
		t.Errorf("jitter elapsed %v, want >= 5ms", elapsed)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("jitter elapsed %v, unexpectedly long", elapsed)
	}
}

func TestJitter_DisabledByZero(t *testing.T) {
	t.Setenv("APNIC_NO_JITTER", "")
	c := NewClient(WithJitter(0, 0))
	start := time.Now()
	c.Jitter(context.Background())
	if d := time.Since(start); d > 5*time.Millisecond {
		t.Errorf("zero jitter should be near-instant, took %v", d)
	}
}

func TestJitter_StealthOff(t *testing.T) {
	t.Setenv("APNIC_NO_JITTER", "")
	c := NewClient(WithStealth(false), WithJitter(50*time.Millisecond, 100*time.Millisecond))
	start := time.Now()
	c.Jitter(context.Background())
	if d := time.Since(start); d > 5*time.Millisecond {
		t.Errorf("stealth-off jitter should be near-instant, took %v", d)
	}
}

func TestJitter_ContextCancel(t *testing.T) {
	t.Setenv("APNIC_NO_JITTER", "")
	c := NewClient(WithJitter(10*time.Second, 20*time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	c.Jitter(ctx)
	if d := time.Since(start); d > 200*time.Millisecond {
		t.Errorf("cancelled jitter should return quickly, took %v", d)
	}
}

func TestJitter_SwappedMinMax(t *testing.T) {
	t.Setenv("APNIC_NO_JITTER", "")
	// max < min should be silently swapped, not panic.
	c := NewClient(WithJitter(10*time.Millisecond, 5*time.Millisecond))
	c.Jitter(context.Background())
	if c.jitterMin > c.jitterMax {
		t.Errorf("swapped min/max not corrected: min=%v max=%v", c.jitterMin, c.jitterMax)
	}
}

func TestRateLimit_Enforced(t *testing.T) {
	c := NewClient(WithRateLimit(2.0), WithJitter(0, 0)) // 2 req/s => 0.5s between tokens (burst 1)
	srv, _ := capturingHandler(t, http.StatusOK, "ok")
	defer srv.Close()

	start := time.Now()
	// First call consumes the burst token immediately; second must wait ~0.5s.
	if err := c.WaitRateLimit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := c.WaitRateLimit(context.Background()); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if elapsed < 300*time.Millisecond {
		t.Errorf("rate limiter should have delayed 2nd call, elapsed %v", elapsed)
	}
}

func TestRateLimit_Disabled(t *testing.T) {
	c := NewClient() // no rate limiter
	start := time.Now()
	if err := c.WaitRateLimit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 5*time.Millisecond {
		t.Errorf("disabled rate limit should be instant, took %v", d)
	}
}

func TestRateLimit_ContextCancel(t *testing.T) {
	c := NewClient(WithRateLimit(0.0001), WithJitter(0, 0)) // ~1 req per 2.7h
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	// First wait consumes the burst token (returns nil immediately); the second
	// must wait ~2.7h for the next token and should hit the ctx deadline.
	_ = c.WaitRateLimit(ctx)
	if err := c.WaitRateLimit(ctx); err == nil {
		t.Error("expected context-deadline error from rate limiter")
	}
}

func TestNewRateLimiter_Zero(t *testing.T) {
	if newRateLimiter(0) != nil {
		t.Error("newRateLimiter(0) should return nil")
	}
	if newRateLimiter(-1) != nil {
		t.Error("newRateLimiter(-1) should return nil")
	}
}

func TestRateLimiter_NilNoOp(t *testing.T) {
	var r *rateLimiter
	if err := r.wait(context.Background()); err != nil {
		t.Errorf("nil rateLimiter.wait should be no-op, got %v", err)
	}
}

// TestStealth_GzipNotDoubleDecompressed is the critical regression test: with
// stealth on, Accept-Encoding: gzip is sent explicitly. The server returns a
// gzip-compressed body with Content-Encoding: gzip. fetchText must decompress
// exactly once (the Content-Encoding branch), not twice (which would error).
func TestStealth_GzipNotDoubleDecompressed(t *testing.T) {
	c := NewClient() // stealth on by default
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "text/plain")
		w.Write(gzipBytes("hello-not-corrupted"))
	}))
	defer srv.Close()

	out, err := c.FetchText(context.Background(), srv.URL+"/x")
	if err != nil {
		t.Fatalf("fetchText with gzip response failed: %v", err)
	}
	if out != "hello-not-corrupted" {
		t.Errorf("decompressed body = %q, want hello-not-corrupted (double-decompress would have errored)", out)
	}
}

func TestDoHTTPRequest_ContextCancel(t *testing.T) {
	c := NewClient(WithJitter(0, 0))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := c.DoHTTPRequest(ctx, "GET", srv.URL, "text/plain"); err == nil {
		t.Error("expected error on cancelled context")
	}
}

// TestDoHTTPRequest_RateLimitError covers the waitRateLimit-error branch: with a
// rate limiter and an already-cancelled context, waitRateLimit returns ctx.Err
// before the HTTP call is made.
func TestDoHTTPRequest_RateLimitError(t *testing.T) {
	c := NewClient(WithJitter(0, 0), WithRateLimit(1.0))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the request so waitRateLimit fails immediately
	if _, err := c.DoHTTPRequest(ctx, "GET", srv.URL, "text/plain"); err == nil {
		t.Error("expected waitRateLimit error from cancelled context")
	}
}

// TestRandSource_Int63n_NonPositive covers the n<=0 guard of Int63n.
func TestRandSource_Int63n_NonPositive(t *testing.T) {
	c := NewClient()
	rs := c.rand
	if v := rs.Int63n(0); v != 0 {
		t.Errorf("Int63n(0) = %d, want 0", v)
	}
	if v := rs.Int63n(-5); v != 0 {
		t.Errorf("Int63n(-5) = %d, want 0", v)
	}
	// Positive n returns a value in [0, n).
	if v := rs.Int63n(100); v < 0 || v >= 100 {
		t.Errorf("Int63n(100) = %d, out of [0,100)", v)
	}
}
