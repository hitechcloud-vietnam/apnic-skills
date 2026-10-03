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

func TestParseIPv6AssignedFull(t *testing.T) {
	result, err := ParseIPv6AssignedFull(strings.NewReader(testutil.SampleIPv6AssignedData))
	if err != nil {
		t.Fatalf("ParseIPv6AssignedFull() error: %v", err)
	}
	if result.Header.Version != "2" {
		t.Errorf("header version = %q, want 2", result.Header.Version)
	}
	if result.Header.Registry != "apnic" {
		t.Errorf("header registry = %q, want apnic", result.Header.Registry)
	}
	if len(result.Summaries) != 1 {
		t.Errorf("summaries = %d, want 1", len(result.Summaries))
	}
	if result.Summaries[0].Count != 7621 {
		t.Errorf("summary count = %d, want 7621", result.Summaries[0].Count)
	}
	if len(result.Entries) != 4 {
		t.Errorf("entries = %d, want 4", len(result.Entries))
	}

	first := result.Entries[0]
	if first.Country != "HK" {
		t.Errorf("first entry country = %q, want HK", first.Country)
	}
	if first.Start != "2001:7fa:0:1::" {
		t.Errorf("first entry start = %q, want 2001:7fa:0:1::", first.Start)
	}
	if first.Value != 64 {
		t.Errorf("first entry prefix = %d, want 64", first.Value)
	}
	if !first.Date.Equal(time.Date(2002, 1, 16, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("first entry date = %v, want 2002-01-16", first.Date)
	}
}

func TestParseIPv6AssignedFullEdgeCases(t *testing.T) {
	data := `2|apnic|20260629|7621|20020116|20260626|+1000
# comment line

apnic|*|ipv6|*|7621|summary
apnic|HK|ipv6|2001:7fa:0:1::|64|20020116
shortline
apnic|JP|ipv4|1.2.3.0|24|20200101
apnic|KR|ipv6|2001:7fa:0:2::|300|20020117
apnic|TW|ipv6|2001:7fa:1::
`
	result, err := ParseIPv6AssignedFull(strings.NewReader(data))
	if err != nil {
		t.Fatalf("ParseIPv6AssignedFull() error: %v", err)
	}
	// Only the HK entry survives: ipv4 row skipped, prefix 300 invalid (out of range),
	// and the TW row has too few fields (no date column → still 5 fields, < 6 required).
	if len(result.Entries) != 1 {
		t.Errorf("entries = %d, want 1 (only HK)", len(result.Entries))
	}
	if result.Entries[0].Country != "HK" {
		t.Errorf("entry country = %q, want HK", result.Entries[0].Country)
	}
}

func TestFetchIPv6AssignedEntries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleIPv6AssignedData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	entries, err := FetchIPv6AssignedEntries(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchIPv6AssignedEntries() error: %v", err)
	}
	if len(entries) != 4 {
		t.Errorf("entries = %d, want 4", len(entries))
	}
}

func TestFetchIPv6AssignedEntriesByDate(t *testing.T) {
	var requestedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedURL = r.URL.Path
		testutil.ServeDated(w, r, testutil.SampleIPv6AssignedData)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	entries, err := FetchIPv6AssignedEntriesByDate(context.Background(), client, "20260629")
	if err != nil {
		t.Fatalf("FetchIPv6AssignedEntriesByDate() error: %v", err)
	}
	if len(entries) != 4 {
		t.Errorf("entries = %d, want 4", len(entries))
	}
	if !strings.Contains(requestedURL, "20260629") {
		t.Errorf("requested URL %q does not contain date 20260629", requestedURL)
	}
	if !strings.Contains(requestedURL, "delegated-apnic-ipv6-assigned") {
		t.Errorf("requested URL %q does not contain delegated-apnic-ipv6-assigned", requestedURL)
	}
}

func TestFetchIPv6AssignedResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleIPv6AssignedData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchIPv6AssignedResult(context.Background(), client, "")
	if err != nil {
		t.Fatalf("FetchIPv6AssignedResult() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
	if len(result.Summaries) == 0 {
		t.Error("expected summaries")
	}
}

func TestFetchIPv6AssignedResultHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchIPv6AssignedResult(context.Background(), client, "")
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestFetchIPv6AssignedEntriesByDateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchIPv6AssignedEntriesByDate(context.Background(), client, "20260629")
	if err == nil {
		t.Error("expected error for HTTP 404 in FetchIPv6AssignedEntriesByDate")
	}
}

// TestFetchIPv6AssignedEntriesError covers the error branch of
// FetchIPv6AssignedEntries (ipv6_assigned.go:18) — when the underlying
// FetchIPv6AssignedResult returns an error (HTTP 500 here), Entries must
// propagate it as nil, err.
func TestFetchIPv6AssignedEntriesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	entries, err := FetchIPv6AssignedEntries(context.Background(), client)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if entries != nil {
		t.Errorf("expected nil entries on error, got %d", len(entries))
	}
}
