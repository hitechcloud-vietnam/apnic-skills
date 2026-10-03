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

func TestParseExtendedFull(t *testing.T) {
	result, err := ParseExtendedFull(strings.NewReader(testutil.SampleExtendedData))
	if err != nil {
		t.Fatalf("ParseExtendedFull() error: %v", err)
	}
	if result.Header.Version != "2.3" {
		t.Errorf("header version = %q, want 2.3", result.Header.Version)
	}
	if len(result.Summaries) != 3 {
		t.Errorf("summaries = %d, want 3", len(result.Summaries))
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
	for _, e := range result.Entries {
		if e.OpaqueID == "" {
			t.Error("expected non-empty OpaqueID in extended entry")
			break
		}
	}
	found := false
	for _, e := range result.Entries {
		if e.OpaqueID == "A91872ED" {
			found = true
			if e.Country != "AU" {
				t.Errorf("A91872ED country = %q, want AU", e.Country)
			}
		}
	}
	if !found {
		t.Error("expected to find entry with OpaqueID A91872ED")
	}
}

func TestParseExtendedFullFromString(t *testing.T) {
	result, err := ParseExtendedFullFromString(testutil.SampleExtendedData)
	if err != nil {
		t.Fatalf("ParseExtendedFullFromString() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestParseExtendedFullEdgeCases(t *testing.T) {
	// Test comment lines, empty lines, short lines, invalid values, unknown type
	data := `2.3|apnic|20260627|5|19850701|20260626|+1000
# comment

apnic|*|asn|*|100|summary
apnic|AU|ipv4|1.0.0.0|invalid|20110811|assigned|A91872ED
apnic|AU|unknown|1.0.1.0|256|20110811|allocated|A92E1062
shortline
apnic|CN|ipv4|1.0.2.0|256|20110414|allocated|A92E1062
`
	result, err := ParseExtendedFull(strings.NewReader(data))
	if err != nil {
		t.Fatalf("ParseExtendedFull() error: %v", err)
	}
	// Only CN ipv4 should be valid
	if len(result.Entries) != 1 {
		t.Errorf("entries = %d, want 1", len(result.Entries))
	}
}

func TestFetchExtendedEntries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleExtendedData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchExtendedEntries(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchExtendedEntries() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchExtendedEntriesByDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.ServeDated(w, r, testutil.SampleExtendedData)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchExtendedEntriesByDate(context.Background(), client, "20260627")
	if err != nil {
		t.Fatalf("FetchExtendedEntriesByDate() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchExtendedResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleExtendedData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchExtendedResult(context.Background(), client, "")
	if err != nil {
		t.Fatalf("FetchExtendedResult() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchExtendedResultByYear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.ServeDated(w, r, testutil.SampleExtendedData)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchExtendedResultByYear(context.Background(), client, 2026)
	if err != nil {
		t.Fatalf("FetchExtendedResultByYear() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchExtendedResultHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchExtendedResult(context.Background(), client, "")
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestFetchExtendedResultByYearHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchExtendedResultByYear(context.Background(), client, 2026)
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}
