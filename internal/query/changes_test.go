package query

import (
	"context"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/testutil"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseChangesData(t *testing.T) {
	result, err := parseChangesData(testutil.SampleChangesData)
	if err != nil {
		t.Fatalf("parseChangesData() error: %v", err)
	}
	if len(result.Changes) != 3 {
		t.Fatalf("changes count = %d, want 3", len(result.Changes))
	}
}

func TestParseChangesMetadata(t *testing.T) {
	result, err := parseChangesData(testutil.SampleChangesData)
	if err != nil {
		t.Fatalf("parseChangesData() error: %v", err)
	}
	if result.Metadata.Count != 3 {
		t.Errorf("count = %d, want 3", result.Metadata.Count)
	}
	if result.Metadata.Version != "0.1" {
		t.Errorf("version = %q, want 0.1", result.Metadata.Version)
	}
	if result.Metadata.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestParseChangesRecords(t *testing.T) {
	result, err := parseChangesData(testutil.SampleChangesData)
	if err != nil {
		t.Fatalf("parseChangesData() error: %v", err)
	}

	c1 := result.Changes[0]
	if c1.Country != "IN" {
		t.Errorf("country = %q, want IN", c1.Country)
	}
	if c1.Custodian != "A91ED89F" {
		t.Errorf("custodian = %q, want A91ED89F", c1.Custodian)
	}
	if c1.Status != "allocated" {
		t.Errorf("status = %q, want allocated", c1.Status)
	}
	if c1.Type != "delegated" {
		t.Errorf("type = %q, want delegated", c1.Type)
	}
	if len(c1.Resources) != 1 || c1.Resources[0] != "160.236.32.0/23" {
		t.Errorf("resources = %v, want [160.236.32.0/23]", c1.Resources)
	}

	c2 := result.Changes[1]
	if c2.Type != "cc-changed" {
		t.Errorf("type = %q, want cc-changed", c2.Type)
	}
}

func TestParseChangesDataInvalidJSON(t *testing.T) {
	_, err := parseChangesData("invalid json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseChangesDataMalformedLine(t *testing.T) {
	data := `{"count":1,"version":"0.1"}

not valid json line`
	result, err := parseChangesData(data)
	if err != nil {
		t.Fatalf("parseChangesData() unexpected error: %v", err)
	}
	if len(result.Changes) != 0 {
		t.Errorf("changes count = %d, want 0 for malformed line", len(result.Changes))
	}
}

func TestParseChangesDataRFC3339Timestamp(t *testing.T) {
	data := `{"count":1,"version":"0.1","timestamp":"2026-06-26 15:23:38"}
{"cc":"AU","resources":["1.2.3.0/24"],"timestamp":"2026-06-25T22:27:30Z","type":"cc-changed"}`
	result, err := parseChangesData(data)
	if err != nil {
		t.Fatalf("parseChangesData() error: %v", err)
	}
	if len(result.Changes) != 1 {
		t.Fatalf("changes count = %d, want 1", len(result.Changes))
	}
	if result.Changes[0].Timestamp.IsZero() {
		t.Error("expected non-zero timestamp from RFC3339 format")
	}
}

func TestFetchChanges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(testutil.SampleChangesData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchChanges(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchChanges() error: %v", err)
	}
	if len(result.Changes) != 3 {
		t.Errorf("changes count = %d, want 3", len(result.Changes))
	}
}

func TestFetchChangesByDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(testutil.SampleChangesData))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchChangesByDate(context.Background(), client, "20260627")
	if err != nil {
		t.Fatalf("FetchChangesByDate() error: %v", err)
	}
	if len(result.Changes) != 3 {
		t.Errorf("changes count = %d, want 3", len(result.Changes))
	}
}

func TestFetchChangesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchChanges(context.Background(), client)
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestFetchChangesByDateHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchChangesByDate(context.Background(), client, "20260627")
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}
