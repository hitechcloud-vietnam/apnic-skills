// Package testutil provides shared test helpers and fixtures used by the
// subpackage test suites under internal/. It is a normal (non-_test) package
// so that any subpackage's _test.go files can import it.
//
// testutil deliberately imports no other package in this module (only stdlib)
// so that any subpackage — including internal/transport — can import it
// without forming an import cycle in tests.
package testutil

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ErrorReader is an io.Reader that always returns an error.
type ErrorReader struct{}

// Read implements io.Reader.
func (ErrorReader) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

// ErrorRoundTripper is an http.RoundTripper that returns a response with a body
// that errors on read. This triggers io.Copy / io.ReadAll errors.
type ErrorRoundTripper struct{}

// RoundTrip implements http.RoundTripper.
func (ErrorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(ErrorReader{}),
	}, nil
}

// DialErrRoundTripper is an http.RoundTripper whose RoundTrip always returns an
// error, exercising request-error branches.
type DialErrRoundTripper struct{}

// RoundTrip implements http.RoundTripper.
func (DialErrRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("dial: connection refused")
}

// CapturingHandler starts an httptest.Server that records the request headers
// of the last received request and responds with the given status and body.
// Returns the server and a pointer to the captured http.Header.
func CapturingHandler(t *testing.T, status int, body string) (*httptest.Server, *http.Header) {
	t.Helper()
	var captured http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	return srv, &captured
}

// MockWhoisServer starts a TCP server that mimics a Whois server: it reads one
// buffer of input then writes the fixed response and closes the connection.
// Returns the dial address and a cleanup func.
func MockWhoisServer(t *testing.T, response string) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				buf := make([]byte, 1024)
				conn.Read(buf)
				conn.Write([]byte(response))
			}()
		}
	}()
	return ln.Addr().String(), func() { ln.Close() }
}

// GzipBytes gzip-compresses the given data, for serving .gz responses in tests.
func GzipBytes(data string) []byte {
	var buf strings.Builder
	zw := gzip.NewWriter(&buf)
	zw.Write([]byte(data))
	zw.Close()
	return []byte(buf.String())
}

// ErrorConn is a net.Conn wrapper that can inject write and read errors. It is
// used by DialWithWriteError / DialWithReadError to exercise whois dial-error
// branches from subpackage tests.
type ErrorConn struct {
	net.Conn
	WriteErr error
	ReadErr  error
}

// Write implements net.Conn, injecting WriteErr when set.
func (c *ErrorConn) Write(b []byte) (int, error) {
	if c.WriteErr != nil {
		return 0, c.WriteErr
	}
	return c.Conn.Write(b)
}

// Read implements net.Conn, injecting ReadErr when set.
func (c *ErrorConn) Read(b []byte) (int, error) {
	if c.ReadErr != nil {
		return 0, c.ReadErr
	}
	return c.Conn.Read(b)
}

// DialWithWriteError returns a dial function that connects to addr but wraps
// the connection with a simulated write error. The returned function matches
// transport.DialFunc (plain stdlib signature, no module import needed).
func DialWithWriteError(addr string) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 5 * time.Second}
		conn, err := d.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return &ErrorConn{Conn: conn, WriteErr: fmt.Errorf("simulated write error")}, nil
	}
}

// DialWithReadError returns a dial function that connects to addr but wraps
// the connection with a simulated read error.
func DialWithReadError(addr string) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 5 * time.Second}
		conn, err := d.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return &ErrorConn{Conn: conn, ReadErr: fmt.Errorf("simulated read error")}, nil
	}
}

// ServeDated writes sample as either gzip-compressed (when the request URL
// ends in .gz, matching APNIC's dated-file layout) or plain text (for latest
// files).
func ServeDated(w http.ResponseWriter, r *http.Request, sample string) {
	if strings.HasSuffix(r.URL.Path, ".gz") {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(GzipBytes(sample))
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(sample))
}

// NewTestServer starts an httptest.Server with the given handler and returns
// it. Callers build their own client pointing at server.URL using the
// transport.NewClient options they need (the transport package cannot live
// here without creating a test import cycle).
func NewTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// AllStatsHandler returns an HTTP handler that serves all types of APNIC data
// based on URL path patterns. Dated (.gz) requests are served gzip-compressed.
func AllStatsHandler() http.HandlerFunc {
	pickSample := func(path string) (string, string) {
		switch {
		case strings.Contains(path, "delegated-extended"):
			return SampleExtendedData, "text/plain"
		case strings.Contains(path, "delegated-ipv6-assigned"):
			return SampleIPv6AssignedData, "text/plain"
		case strings.Contains(path, "assigned"):
			return SampleAssignedData, "text/plain"
		case strings.Contains(path, "legacy"):
			return SampleLegacyData, "text/plain"
		case strings.Contains(path, "delegated"):
			return SampleDelegatedData, "text/plain"
		case strings.Contains(path, "transfers"):
			return SampleTransfersJSON, "application/json"
		case strings.Contains(path, "changes"):
			return SampleChangesData, "application/json"
		default:
			return SampleRDAPNotFoundJSON, "application/rdap+json"
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		sample, contentType := pickSample(path)
		if strings.HasSuffix(path, ".gz") {
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(GzipBytes(sample))
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Write([]byte(sample))
	}
}

// RdapHandler returns an HTTP handler for RDAP requests.
func RdapHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rdap+json")
		path := r.URL.Path

		switch {
		case strings.Contains(path, "/ip/"):
			w.Write([]byte(SampleRDAPNetworkJSON))
		case strings.Contains(path, "/autnum/"):
			w.Write([]byte(SampleRDAPAutnumJSON))
		case strings.Contains(path, "/domain/"):
			w.Write([]byte(SampleRDAPDomainJSON))
		case strings.Contains(path, "/entity/"):
			w.Write([]byte(SampleRDAPEntityJSON))
		case strings.Contains(path, "/entities"):
			w.Write([]byte(SampleRDAPSearchJSON))
		case strings.Contains(path, "/domains"):
			w.Write([]byte(SampleRDAPDomainsSearchJSON))
		case strings.HasSuffix(path, "/help"):
			w.Write([]byte(SampleRDAPHelpJSON))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(SampleRDAPNotFoundJSON))
		}
	}
}

// CombinedHandler returns an HTTP handler that routes stats and RDAP requests.
func CombinedHandler() http.HandlerFunc {
	stats := AllStatsHandler()
	rdap := RdapHandler()

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/ip/") ||
			strings.HasPrefix(path, "/autnum/") ||
			strings.HasPrefix(path, "/domain/") ||
			strings.HasPrefix(path, "/domains") ||
			strings.HasPrefix(path, "/entity/") ||
			strings.HasPrefix(path, "/entities") ||
			strings.HasSuffix(path, "/help") {
			rdap(w, r)
		} else {
			stats(w, r)
		}
	}
}
