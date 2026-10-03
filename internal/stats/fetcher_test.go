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

func TestParseDelegatedFull(t *testing.T) {
	result, err := ParseDelegatedFull(strings.NewReader(testutil.SampleDelegatedData))
	if err != nil {
		t.Fatalf("ParseDelegatedFull() error: %v", err)
	}
	if result.Header.Version != "2" {
		t.Errorf("header version = %q, want 2", result.Header.Version)
	}
	if len(result.Summaries) != 3 {
		t.Errorf("summaries count = %d, want 3", len(result.Summaries))
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries, got empty slice")
	}
	found := false
	for _, e := range result.Entries {
		if e.Country == "AU" && e.Type == "ipv4" && e.Start == "1.0.0.0" {
			found = true
			if e.Value != 256 {
				t.Errorf("AU ipv4 value = %d, want 256", e.Value)
			}
			if e.Status != "assigned" {
				t.Errorf("AU ipv4 status = %q, want assigned", e.Status)
			}
		}
	}
	if !found {
		t.Error("expected to find AU ipv4 entry")
	}
}

func TestParseDelegatedFullFromString(t *testing.T) {
	result, err := ParseDelegatedFullFromString(testutil.SampleDelegatedData)
	if err != nil {
		t.Fatalf("ParseDelegatedFullFromString() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestParseDelegatedData(t *testing.T) {
	result, err := ParseDelegatedData(strings.NewReader(testutil.SampleDelegatedData))
	if err != nil {
		t.Fatalf("ParseDelegatedData() error: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected entries")
	}
}

func TestParseDelegatedFullEdgeCases(t *testing.T) {
	// Test with comment lines, empty lines, short lines, invalid values, unknown types
	data := `2|apnic|20260627|5|19850701|20260626|+1000
# this is a comment

apnic|*|asn|*|100|summary
apnic|AU|ipv4|1.0.0.0|invalidcount|20110811|assigned
apnic|AU|unknown|1.0.2.0|256|20110811|allocated
short|line
apnic|CN|ipv4|1.0.3.0|256|20110414|allocated
`
	result, err := ParseDelegatedFull(strings.NewReader(data))
	if err != nil {
		t.Fatalf("ParseDelegatedFull() error: %v", err)
	}
	// Invalid count entry is skipped; unknown type entry is skipped;
	// short line is skipped; only CN ipv4 remains
	if len(result.Entries) != 1 {
		t.Errorf("entries = %d, want 1", len(result.Entries))
	}
}

func TestFetchDelegatedEntries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleDelegatedData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	entries, err := FetchDelegatedEntries(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchDelegatedEntries() error: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchDelegatedEntriesByDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.ServeDated(w, r, testutil.SampleDelegatedData)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	entries, err := FetchDelegatedEntriesByDate(context.Background(), client, "20260627")
	if err != nil {
		t.Fatalf("FetchDelegatedEntriesByDate() error: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchDelegatedResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleDelegatedData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchDelegatedResult(context.Background(), client, "")
	if err != nil {
		t.Fatalf("FetchDelegatedResult() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchDelegatedResultByYear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.ServeDated(w, r, testutil.SampleDelegatedData)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchDelegatedResultByYear(context.Background(), client, 2026)
	if err != nil {
		t.Fatalf("FetchDelegatedResultByYear() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchDelegatedResultHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchDelegatedResult(context.Background(), client, "")
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestFetchDelegatedCancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleDelegatedData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := FetchDelegatedEntries(ctx, client)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestFetchTextInvalidURL(t *testing.T) {
	client := transport.NewClient(transport.WithStatsBaseURL("http://[::1]:%invalid/"))
	_, err := FetchDelegatedEntries(context.Background(), client)
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestFetchDelegatedEntriesByDateHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchDelegatedEntriesByDate(context.Background(), client, "20260627")
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestFetchDelegatedResultByYearHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchDelegatedResultByYear(context.Background(), client, 2026)
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestParseDelegatedDataError(t *testing.T) {
	// ParseDelegatedData wraps ParseDelegatedFull; test with a reader that returns error
	_, err := ParseDelegatedData(testutil.ErrorReader{})
	if err == nil {
		t.Error("expected error from error reader")
	}
}

func TestFetchTextReadError(t *testing.T) {
	// Use a custom RoundTripper that returns a body that errors on read
	client := transport.NewClient(
		transport.WithHTTPClient(&http.Client{Transport: testutil.ErrorRoundTripper{}}),
		transport.WithStatsBaseURL("http://example.com/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchDelegatedEntries(context.Background(), client)
	if err == nil {
		t.Error("expected error for read failure")
	}
}

func TestFetchTextGzipInitError(t *testing.T) {
	// A .gz URL whose body is not valid gzip must surface a gzip init error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write([]byte("not-gzip-data"))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchDelegatedEntriesByDate(context.Background(), client, "20260627")
	if err == nil {
		t.Fatal("expected gzip init error")
	}
	if !strings.Contains(err.Error(), "gzip init failed") {
		t.Errorf("expected gzip init failed error, got: %v", err)
	}
}

func TestFetchTextContentEncodingGzip(t *testing.T) {
	// A non-.gz URL that carries Content-Encoding: gzip should also be decompressed.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(testutil.GzipBytes(testutil.SampleDelegatedData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	// Latest fetch (no .gz suffix) but server returns gzip via Content-Encoding.
	entries, err := FetchDelegatedEntries(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchDelegatedEntries() error: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected entries from Content-Encoding gzip response")
	}
}
