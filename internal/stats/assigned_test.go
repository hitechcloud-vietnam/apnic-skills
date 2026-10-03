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

func TestParseAssignedFull(t *testing.T) {
	result, err := ParseAssignedFull(strings.NewReader(testutil.SampleAssignedData))
	if err != nil {
		t.Fatalf("ParseAssignedFull() error: %v", err)
	}
	if result.Header.Version != "1" {
		t.Errorf("header version = %q, want 1", result.Header.Version)
	}
	if len(result.Summaries) != 2 {
		t.Errorf("summaries = %d, want 2", len(result.Summaries))
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestParseAssignedFullEdgeCases(t *testing.T) {
	data := `1|apnic|20260627|5|19850701|20260626|+1000
# comment

apnic|*|ipv4|*|100|summary
apnic|ae|asn||1||allocated||5
apnic|ae|ipv4||256||assigned||12
shortline
apnic|ae|ipv4||4||assigned||0
`
	result, err := ParseAssignedFull(strings.NewReader(data))
	if err != nil {
		t.Fatalf("ParseAssignedFull() error: %v", err)
	}
	// Only ipv4 entries should be parsed (asn entries skipped)
	// The entry with count 0 should still be included (parseIPv4Count returns error for 0, count stays 0)
	if len(result.Entries) != 2 {
		t.Errorf("entries = %d, want 2", len(result.Entries))
	}
}

func TestFetchAssignedEntries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleAssignedData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchAssignedEntries(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchAssignedEntries() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchAssignedEntriesByDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.ServeDated(w, r, testutil.SampleAssignedData)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchAssignedEntriesByDate(context.Background(), client, "20260627")
	if err != nil {
		t.Fatalf("FetchAssignedEntriesByDate() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchAssignedResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleAssignedData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchAssignedResult(context.Background(), client, "")
	if err != nil {
		t.Fatalf("FetchAssignedResult() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchAssignedResultHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchAssignedResult(context.Background(), client, "")
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}
