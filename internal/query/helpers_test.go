package query

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/testutil"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// newTestClient creates a *transport.Client pointing at a test server backed by
// the given handler, with jitter disabled and a 1-hour cache. It mirrors the
// transport package's same-named helper but lives here so the query subpackage
// tests (which cannot import transport's _test files) can build a client.
func newTestClient(handler http.HandlerFunc) (*transport.Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithRDAPBaseURL(server.URL),
		transport.WithCacheTTL(1*time.Hour),
		transport.WithJitter(0, 0),
	)
	return client, server
}

// combinedHandler re-exports testutil.CombinedHandler under the lowercase name
// that the query tests were written against.
func combinedHandler() http.HandlerFunc { return testutil.CombinedHandler() }

// mockWhoisServer is re-exported from testutil so query tests keep the
// lowercase name used before the package restructure.
func mockWhoisServer(t *testing.T, response string) (string, func()) {
	return testutil.MockWhoisServer(t, response)
}
