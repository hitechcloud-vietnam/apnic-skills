package transport

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/testutil"
)

// init disables request jitter for the whole test binary so the suite runs fast.
// stealth_test.go re-enables it locally via t.Setenv where real jitter is needed.
func init() {
	_ = os.Setenv("APNIC_NO_JITTER", "1")
}

// md5Hash computes the MD5 hash of a string.
func md5Hash(data string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}

// errorReader / errorRoundTripper / mockWhoisServer are re-exported from the
// shared testutil package so existing transport tests keep the lowercase names.
type errorReader = testutil.ErrorReader

type errorRoundTripper = testutil.ErrorRoundTripper

// dialErrRoundTripper is re-exported from testutil (request-error RoundTripper).
type dialErrRoundTripper = testutil.DialErrRoundTripper

func mockWhoisServer(t *testing.T, response string) (string, func()) {
	return testutil.MockWhoisServer(t, response)
}

// capturingHandler records the request headers of the last received request.
// Re-exported from testutil so transport tests keep the lowercase name.
func capturingHandler(t *testing.T, status int, body string) (*httptest.Server, *http.Header) {
	return testutil.CapturingHandler(t, status, body)
}

// errorConn is a net.Conn wrapper that can inject write and read errors.
// This stays in the transport package because it is used only by the
// transport-private dial error helpers (withDialWhois / dialWith*).
type errorConn struct {
	net.Conn
	writeErr error
	readErr  error
}

func (c *errorConn) Write(b []byte) (int, error) {
	if c.writeErr != nil {
		return 0, c.writeErr
	}
	return c.Conn.Write(b)
}

func (c *errorConn) Read(b []byte) (int, error) {
	if c.readErr != nil {
		return 0, c.readErr
	}
	return c.Conn.Read(b)
}

// withDialWhois sets a custom dial function for whois connections (for testing).
func withDialWhois(fn dialFunc) Option {
	return func(c *Client) {
		c.dialWhois = fn
	}
}

// dialWithWriteError creates a dial function that connects to the given address
// but wraps the connection with a write error.
func dialWithWriteError(addr string) dialFunc {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 5 * time.Second}
		conn, err := d.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return &errorConn{Conn: conn, writeErr: fmt.Errorf("simulated write error")}, nil
	}
}

// dialWithReadError creates a dial function that connects to the given address
// but wraps the connection with a read error.
func dialWithReadError(addr string) dialFunc {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 5 * time.Second}
		conn, err := d.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return &errorConn{Conn: conn, readErr: fmt.Errorf("simulated read error")}, nil
	}
}

// mockWhoisServer starts a TCP server that mimics a Whois server.
// (Re-exported from testutil above as the lowercase alias.)

// newTestClient creates a Client with a mock HTTP server that intercepts
// both stats (FTP) and RDAP requests. Jitter/stealth pacing is disabled to keep
// tests fast; stealth headers are validated separately in stealth_test.go.
func newTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := NewClient(
		WithHTTPClient(server.Client()),
		WithRDAPBaseURL(server.URL),
		WithCacheTTL(1*time.Hour),
		WithJitter(0, 0),
	)
	return client, server
}

// allStatsHandler returns an HTTP handler that serves all types of APNIC data
// based on URL path patterns. Thin re-export of testutil.AllStatsHandler so
// existing transport tests keep using the lowercase name.
func allStatsHandler() http.HandlerFunc { return testutil.AllStatsHandler() }

// rdapHandler returns an HTTP handler for RDAP requests.
func rdapHandler() http.HandlerFunc { return testutil.RdapHandler() }

// combinedHandler returns an HTTP handler that routes stats and RDAP requests.
func combinedHandler() http.HandlerFunc { return testutil.CombinedHandler() }

// serveDated writes sample as gzip-compressed (.gz) or plain text. Re-export of
// testutil.ServeDated so transport tests keep the lowercase name.
func serveDated(w http.ResponseWriter, r *http.Request, sample string) {
	testutil.ServeDated(w, r, sample)
}

// gzipBytes gzip-compresses data. Re-export of testutil.GzipBytes.
func gzipBytes(data string) []byte { return testutil.GzipBytes(data) }

// stringReader creates an io.Reader from a string.
func stringReader(s string) io.Reader { return strings.NewReader(s) }

// Sample data fixtures are re-exported from the shared testutil package so that
// existing transport tests that reference the lowercase sample* names keep
// compiling. Other subpackages import testutil directly.
const (
	sampleDelegatedData         = testutil.SampleDelegatedData
	sampleExtendedData          = testutil.SampleExtendedData
	sampleAssignedData          = testutil.SampleAssignedData
	sampleIPv6AssignedData      = testutil.SampleIPv6AssignedData
	sampleLegacyData            = testutil.SampleLegacyData
	sampleTransfersJSON         = testutil.SampleTransfersJSON
	sampleTransfersAll          = testutil.SampleTransfersAll
	sampleTransfersAllMD5       = testutil.SampleTransfersAllMD5
	sampleTelemetryJSON         = testutil.SampleTelemetryJSON
	sampleTelemetryMD5          = testutil.SampleTelemetryMD5
	sampleChangesData           = testutil.SampleChangesData
	sampleRDAPNetworkJSON       = testutil.SampleRDAPNetworkJSON
	sampleRDAPAutnumJSON        = testutil.SampleRDAPAutnumJSON
	sampleRDAPDomainJSON        = testutil.SampleRDAPDomainJSON
	sampleRDAPEntityJSON        = testutil.SampleRDAPEntityJSON
	sampleRDAPSearchJSON        = testutil.SampleRDAPSearchJSON
	sampleRDAPDomainsSearchJSON = testutil.SampleRDAPDomainsSearchJSON
	sampleRDAPHelpJSON          = testutil.SampleRDAPHelpJSON
	sampleRDAPNotFoundJSON      = testutil.SampleRDAPNotFoundJSON
	sampleWhoisResponse         = testutil.SampleWhoisResponse
)
