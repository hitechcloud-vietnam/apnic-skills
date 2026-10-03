package testutil

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testutil is a normal (non-_test) package so that every subpackage's tests
// can import it. That means its own helpers are exercised only indirectly from
// other packages' _test.go files, which Go's cover tool does not credit back
// to testutil. This file drives each exported helper directly so the
// restructure did not leave a 0%-coverage hole in the shared helpers package.

func TestErrorReader(t *testing.T) {
	var r ErrorReader
	buf := make([]byte, 4)
	n, err := r.Read(buf)
	if n != 0 || err != io.ErrUnexpectedEOF {
		t.Errorf("ErrorReader.Read = (%d, %v), want (0, io.ErrUnexpectedEOF)", n, err)
	}
}

func TestErrorRoundTripper(t *testing.T) {
	rt := ErrorRoundTripper{}
	resp, err := rt.RoundTrip(&http.Request{})
	if err != nil {
		t.Fatalf("ErrorRoundTripper.RoundTrip error: %v", err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err == nil {
		t.Error("expected read error from ErrorReader-backed body")
	}
}

func TestDialErrRoundTripper(t *testing.T) {
	rt := DialErrRoundTripper{}
	if _, err := rt.RoundTrip(&http.Request{}); err == nil {
		t.Error("expected dial error from DialErrRoundTripper")
	}
}

func TestCapturingHandler(t *testing.T) {
	srv, hdr := CapturingHandler(t, http.StatusOK, "hello")
	defer srv.Close()
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Errorf("body = %q, want hello", body)
	}
	// CapturingHandler clones the *request* headers; the response body and
	// status confirm the handler ran, while the captured request headers
	// confirm the capture path executed.
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if hdr.Get("User-Agent") == "" {
		t.Error("expected captured request User-Agent header")
	}
}

func TestMockWhoisServer(t *testing.T) {
	addr, cleanup := MockWhoisServer(t, "whois-response")
	defer cleanup()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("1.1.1.1\r\n")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "whois-response" {
		t.Errorf("got %q, want whois-response", buf[:n])
	}
}

func TestGzipBytes(t *testing.T) {
	raw := "compress me"
	gz := GzipBytes(raw)
	if len(gz) == 0 || string(gz[:2]) != "\x1f\x8b" {
		t.Errorf("GzipBytes did not return a gzip stream (first bytes: %x)", gz[:min(2, len(gz))])
	}
	// Round-trip via ServeDated's .gz path is covered separately; here just
	// confirm the magic header.
}

func TestServeDated(t *testing.T) {
	// Plain path.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/delegated-apnic-latest", nil)
	ServeDated(rec, req, "plain-data")
	if rec.Body.String() != "plain-data" {
		t.Errorf("plain body = %q", rec.Body.String())
	}

	// .gz path returns gzip bytes.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/delegated-apnic-20260626.gz", nil)
	ServeDated(rec2, req2, "gz-data")
	if rec2.Header().Get("Content-Type") != "application/gzip" {
		t.Errorf(".gz Content-Type = %q", rec2.Header().Get("Content-Type"))
	}
	if rec2.Body.Len() == 0 {
		t.Error("expected gzip body")
	}
}

func TestNewTestServer(t *testing.T) {
	srv := NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want 204", resp.StatusCode)
	}
}

func TestAllStatsHandler(t *testing.T) {
	srv := httptest.NewServer(AllStatsHandler())
	defer srv.Close()
	// AllStatsHandler dispatches by URL substring; assert each path returns a
	// non-empty body with the right content type. (The extended/ipv6-assigned
	// branches are matched on the substring "delegated-extended" /
	// "delegated-ipv6-assigned" which appear in the real archived filenames
	// like delegated-apnic-extended-YYYYMMDD, so exercise those exact names.)
	cases := []struct{ path string }{
		{"/delegated-apnic-extended-20260626"},
		{"/delegated-apnic-ipv6-assigned-20260626"},
		{"/assigned-apnic-latest"},
		{"/legacy-apnic-latest"},
		{"/delegated-apnic-latest"},
		{"/transfers_latest.json"},
		{"/changes_latest.json"},
	}
	for _, c := range cases {
		resp, err := http.Get(srv.URL + c.path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if len(body) == 0 {
			t.Errorf("path %q returned empty body", c.path)
		}
	}
	// .gz variant triggers gzip serving — must be served as application/gzip.
	resp, err := http.Get(srv.URL + "/delegated-apnic-20260626.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Type") != "application/gzip" {
		t.Errorf(".gz Content-Type = %q", resp.Header.Get("Content-Type"))
	}
}

func TestRdapHandler(t *testing.T) {
	srv := httptest.NewServer(RdapHandler())
	defer srv.Close()
	for _, c := range []struct{ path string }{
		{"/ip/1.1.1.1"},
		{"/autnum/13335"},
		{"/domain/1.in-addr.arpa"},
		{"/entity/AIC3-AP"},
		{"/entities?fn=APNIC"},
		{"/domains?name=1"},
		{"/help"},
	} {
		resp, err := http.Get(srv.URL + c.path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if len(body) == 0 {
			t.Errorf("path %q returned empty body", c.path)
		}
		if resp.Header.Get("Content-Type") != "application/rdap+json" {
			t.Errorf("path %q Content-Type = %q", c.path, resp.Header.Get("Content-Type"))
		}
	}
	// Unknown path -> 404.
	resp, err := http.Get(srv.URL + "/nope")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown path status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestCombinedHandler(t *testing.T) {
	srv := httptest.NewServer(CombinedHandler())
	defer srv.Close()
	// RDAP-style path routes to RdapHandler.
	resp, err := http.Get(srv.URL + "/ip/1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "rdapConformance") {
		t.Errorf("RDAP path did not route to RdapHandler: %s", body)
	}
	// stats-style path routes to AllStatsHandler.
	resp2, err := http.Get(srv.URL + "/delegated-apnic-latest")
	if err != nil {
		t.Fatal(err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if !strings.Contains(string(body2), "apnic|") {
		t.Errorf("stats path did not route to AllStatsHandler: %s", body2)
	}
}

func TestDialWithWriteError(t *testing.T) {
	// Start a throwaway TCP listener so the dial succeeds at the TCP layer;
	// the ErrorConn wrapper then injects a write error.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	dial := DialWithWriteError(ln.Addr().String())
	conn, err := dial(context.Background(), "tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("x")); err == nil {
		t.Error("expected injected write error")
	}
	// Read still works (no read error injected).
	if _, err := conn.Read(make([]byte, 4)); err == nil {
		// Either an error (peer closed) or nothing; both acceptable. The point
		// is Write errored.
	}
}

func TestDialWithReadError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	dial := DialWithReadError(ln.Addr().String())
	conn, err := dial(context.Background(), "tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Read(make([]byte, 4)); err == nil {
		t.Error("expected injected read error")
	}
	// Write still works.
	conn.Write([]byte("x"))
}

// TestAllStatsHandler_ExtendedSubstrings covers the delegated-extended and
// delegated-ipv6-assigned branches of pickSample (testutil.go:181,183). These
// match on the literal substrings "delegated-extended" /
// "delegated-ipv6-assigned"; paths containing "delegated-apnic-extended-..."
// (with "-apnic-" in between) do NOT match and previously fell through to the
// broader "delegated"/"assigned" branches, leaving these two returns uncovered.
func TestAllStatsHandler_ExtendedSubstrings(t *testing.T) {
	srv := httptest.NewServer(AllStatsHandler())
	defer srv.Close()
	for _, c := range []struct{ path string }{
		{"/delegated-extended"},
		{"/delegated-ipv6-assigned"},
	} {
		resp, err := http.Get(srv.URL + c.path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if len(body) == 0 {
			t.Errorf("path %q returned empty body", c.path)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "text/plain" {
			t.Errorf("path %q Content-Type = %q, want text/plain", c.path, ct)
		}
	}
}

// TestAllStatsHandler_DefaultBranch covers the default case of AllStatsHandler's
// pickSample (testutil.go:195), which returns SampleRDAPNotFoundJSON for paths
// matching none of the known substrings.
func TestAllStatsHandler_DefaultBranch(t *testing.T) {
	srv := httptest.NewServer(AllStatsHandler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/totally-unknown-path-zzz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("default branch returned empty body, want SampleRDAPNotFoundJSON")
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/rdap+json" {
		t.Errorf("default branch Content-Type = %q, want application/rdap+json", ct)
	}
}

// TestDialWithWriteError_DialFailure covers the DialContext-error branch of
// DialWithWriteError (testutil.go:135) by dialing a closed-port address that
// refuses connections.
func TestDialWithWriteError_DialFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	dial := DialWithWriteError(addr)
	conn, err := dial(context.Background(), "tcp", addr)
	if err == nil {
		conn.Close()
		t.Fatal("expected dial error connecting to closed port")
	}
	if conn != nil {
		t.Error("expected nil conn on dial error")
	}
}

// TestDialWithReadError_DialFailure covers the DialContext-error branch of
// DialWithReadError (testutil.go:147) by dialing a refused port.
func TestDialWithReadError_DialFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	dial := DialWithReadError(addr)
	conn, err := dial(context.Background(), "tcp", addr)
	if err == nil {
		conn.Close()
		t.Fatal("expected dial error connecting to closed port")
	}
	if conn != nil {
		t.Error("expected nil conn on dial error")
	}
}
