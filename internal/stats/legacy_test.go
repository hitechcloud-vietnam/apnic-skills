package stats

import (
	"context"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/testutil"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseLegacyFull(t *testing.T) {
	result, err := ParseLegacyFull(strings.NewReader(testutil.SampleLegacyData))
	if err != nil {
		t.Fatalf("ParseLegacyFull() error: %v", err)
	}
	if result.Header.Version != "1" {
		t.Errorf("header version = %q, want 1", result.Header.Version)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
	// Check that we have ipv4, ipv6, and asn entries
	ipv4Count, ipv6Count, asnCount := 0, 0, 0
	for _, e := range result.Entries {
		switch e.Type {
		case "ipv4":
			ipv4Count++
			if e.Value != 65536 {
				t.Errorf("ipv4 value = %d, want 65536", e.Value)
			}
		case "ipv6":
			ipv6Count++
			if e.Value != 32 {
				t.Errorf("ipv6 value = %d, want 32", e.Value)
			}
		case "asn":
			asnCount++
			if e.Value != 1 {
				t.Errorf("asn value = %d, want 1", e.Value)
			}
		}
	}
	if ipv4Count != 3 {
		t.Errorf("ipv4 count = %d, want 3", ipv4Count)
	}
	if ipv6Count != 1 {
		t.Errorf("ipv6 count = %d, want 1", ipv6Count)
	}
	if asnCount != 1 {
		t.Errorf("asn count = %d, want 1", asnCount)
	}
}

func TestParseLegacyFullEdgeCases(t *testing.T) {
	data := `1|apnic|20260627|3|19850701|20260626|+1000
# comment

apnic|*|ipv4|*|100|summary
apnic||ipv4|128.134.0.0|invalid|20040401|allocated
apnic||unknown|128.184.0.0|65536|20040401|allocated
shortline
apnic||ipv4|128.250.0.0|65536|20040401|allocated
`
	result, err := ParseLegacyFull(strings.NewReader(data))
	if err != nil {
		t.Fatalf("ParseLegacyFull() error: %v", err)
	}
	if len(result.Entries) != 1 {
		t.Errorf("entries = %d, want 1", len(result.Entries))
	}
}

func TestFetchLegacyEntries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleLegacyData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchLegacyEntries(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchLegacyEntries() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchLegacyEntriesByDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.ServeDated(w, r, testutil.SampleLegacyData)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchLegacyEntriesByDate(context.Background(), client, "20260627")
	if err != nil {
		t.Fatalf("FetchLegacyEntriesByDate() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchLegacyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleLegacyData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchLegacyResult(context.Background(), client, "")
	if err != nil {
		t.Fatalf("FetchLegacyResult() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchLegacyResultHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchLegacyResult(context.Background(), client, "")
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}
