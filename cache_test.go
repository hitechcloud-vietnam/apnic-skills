package apnic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/testutil"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// These tests cover the root-package caching layer (Get* methods in cache.go),
// which wraps the transport.Client cache with the stats/query fetch functions.
// They live in package apnic because Get* are root *Client methods; the cache
// field itself is unexported on transport.Client, so the WithCache cases seed
// the cache through the exported CacheSet accessor.

func TestGetDelegatedEntriesWithCache(t *testing.T) {
	client := NewClient(WithCacheTTL(1 * time.Hour))
	entries := []DelegatedEntry{
		{Country: "AU", Type: "ipv4", Start: "1.0.0.0", Value: 256},
		{Country: "CN", Type: "ipv4", Start: "1.0.1.0", Value: 256},
	}
	client.CacheSet(transport.CacheKeyDelegated, entries)

	result, err := client.GetDelegatedEntries(context.Background())
	if err != nil {
		t.Fatalf("GetDelegatedEntries() error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("cached entries count = %d, want 2", len(result))
	}
}

func TestGetDelegatedEntriesFetchPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleDelegatedData))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond), // cache expires immediately
	)

	result, err := client.GetDelegatedEntries(context.Background())
	if err != nil {
		t.Fatalf("GetDelegatedEntries() error: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected entries from fetch path")
	}
}

func TestGetDelegatedEntriesFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond),
	)

	_, err := client.GetDelegatedEntries(context.Background())
	if err == nil {
		t.Error("expected error for fetch failure in GetDelegatedEntries")
	}
}

func TestGetExtendedEntriesWithCache(t *testing.T) {
	client := NewClient(WithCacheTTL(1 * time.Hour))
	entries := []DelegatedExtendedEntry{
		{OpaqueID: "A1", Country: "AU"},
	}
	client.CacheSet(transport.CacheKeyExtended, entries)

	result, err := client.GetExtendedEntries(context.Background())
	if err != nil {
		t.Fatalf("GetExtendedEntries() error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("cached entries count = %d, want 1", len(result))
	}
}

func TestGetExtendedEntriesFetchPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleExtendedData))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond),
	)

	result, err := client.GetExtendedEntries(context.Background())
	if err != nil {
		t.Fatalf("GetExtendedEntries() error: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected entries from fetch path")
	}
}

func TestGetExtendedEntriesFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond),
	)

	_, err := client.GetExtendedEntries(context.Background())
	if err == nil {
		t.Error("expected error for fetch failure in GetExtendedEntries")
	}
}

func TestGetAssignedEntriesWithCache(t *testing.T) {
	client := NewClient(WithCacheTTL(1 * time.Hour))
	entries := []models.AssignedEntry{
		{Country: "ae", Type: "ipv4", Prefix: "4", Count: 1},
	}
	client.CacheSet(transport.CacheKeyAssigned, entries)

	result, err := client.GetAssignedEntries(context.Background())
	if err != nil {
		t.Fatalf("GetAssignedEntries() error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("cached entries count = %d, want 1", len(result))
	}
}

func TestGetAssignedEntriesFetchPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleAssignedData))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond),
	)

	result, err := client.GetAssignedEntries(context.Background())
	if err != nil {
		t.Fatalf("GetAssignedEntries() error: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected entries from fetch path")
	}
}

func TestGetAssignedEntriesFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond),
	)

	_, err := client.GetAssignedEntries(context.Background())
	if err == nil {
		t.Error("expected error for fetch failure in GetAssignedEntries")
	}
}

func TestGetIPv6AssignedEntriesWithCache(t *testing.T) {
	client := NewClient(WithCacheTTL(1 * time.Hour))
	entries := []models.IPv6AssignedEntry{
		{Country: "HK", Start: "2001:7fa:0:1::", Value: 64},
	}
	client.CacheSet(transport.CacheKeyIPv6Assigned, entries)

	result, err := client.GetIPv6AssignedEntries(context.Background())
	if err != nil {
		t.Fatalf("GetIPv6AssignedEntries() error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("cached entries count = %d, want 1", len(result))
	}
}

func TestGetIPv6AssignedEntriesFetchPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleIPv6AssignedData))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond),
	)

	result, err := client.GetIPv6AssignedEntries(context.Background())
	if err != nil {
		t.Fatalf("GetIPv6AssignedEntries() error: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected entries from fetch path")
	}
}

func TestGetIPv6AssignedEntriesFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond),
	)

	_, err := client.GetIPv6AssignedEntries(context.Background())
	if err == nil {
		t.Error("expected error for fetch failure in GetIPv6AssignedEntries")
	}
}

func TestGetLegacyEntriesWithCache(t *testing.T) {
	client := NewClient(WithCacheTTL(1 * time.Hour))
	entries := []models.LegacyEntry{
		{Start: "128.134.0.0", Value: 65536},
	}
	client.CacheSet(transport.CacheKeyLegacy, entries)

	result, err := client.GetLegacyEntries(context.Background())
	if err != nil {
		t.Fatalf("GetLegacyEntries() error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("cached entries count = %d, want 1", len(result))
	}
}

func TestGetLegacyEntriesFetchPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(testutil.SampleLegacyData))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond),
	)

	result, err := client.GetLegacyEntries(context.Background())
	if err != nil {
		t.Fatalf("GetLegacyEntries() error: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected entries from fetch path")
	}
}

func TestGetLegacyEntriesFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond),
	)

	_, err := client.GetLegacyEntries(context.Background())
	if err == nil {
		t.Error("expected error for fetch failure in GetLegacyEntries")
	}
}

func TestGetChangesWithCache(t *testing.T) {
	client := NewClient(WithCacheTTL(1 * time.Hour))
	data := &models.ChangesResult{
		Metadata: models.ChangesMetadata{Count: 1},
		Changes:  []models.ChangeRecord{{Country: "AU"}},
	}
	client.CacheSet(transport.CacheKeyChanges, data)

	result, err := client.GetChanges(context.Background())
	if err != nil {
		t.Fatalf("GetChanges() error: %v", err)
	}
	if len(result.Changes) != 1 {
		t.Errorf("changes count = %d, want 1", len(result.Changes))
	}
}

func TestGetChangesFetchPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(testutil.SampleChangesData))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond),
	)

	result, err := client.GetChanges(context.Background())
	if err != nil {
		t.Fatalf("GetChanges() error: %v", err)
	}
	if len(result.Changes) != 3 {
		t.Errorf("changes count = %d, want 3", len(result.Changes))
	}
}

func TestGetChangesFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond),
	)

	_, err := client.GetChanges(context.Background())
	if err == nil {
		t.Error("expected error for fetch failure in GetChanges")
	}
}

func TestGetTransfersWithCache(t *testing.T) {
	client := NewClient(WithCacheTTL(1 * time.Hour))
	data := &models.TransfersResult{
		Metadata:  models.TransfersMetadata{Producer: "APNIC"},
		Transfers: []models.TransferRecord{{Type: "RESOURCE_TRANSFER"}},
	}
	client.CacheSet(transport.CacheKeyTransfers, data)

	result, err := client.GetTransfers(context.Background())
	if err != nil {
		t.Fatalf("GetTransfers() error: %v", err)
	}
	if len(result.Transfers) != 1 {
		t.Errorf("transfers count = %d, want 1", len(result.Transfers))
	}
}

func TestGetTransfersFetchPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(testutil.SampleTransfersJSON))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond),
	)

	result, err := client.GetTransfers(context.Background())
	if err != nil {
		t.Fatalf("GetTransfers() error: %v", err)
	}
	if len(result.Transfers) != 2 {
		t.Errorf("transfers count = %d, want 2", len(result.Transfers))
	}
}

func TestGetTransfersFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Nanosecond),
	)

	_, err := client.GetTransfers(context.Background())
	if err == nil {
		t.Error("expected error for fetch failure in GetTransfers")
	}
}

func TestGetIRRDatabaseWithCache(t *testing.T) {
	client := NewClient(WithCacheTTL(1 * time.Hour))
	// Manually seed cache to verify the Get path returns cached data.
	db := &models.IRRDatabase{Type: "route", Objects: []models.IRRObject{{Type: "route", PrimaryKey: "1.0.0.0/24"}}}
	client.CacheSet(transport.CacheKeyIRR("route"), db)

	got, err := client.GetIRRDatabase(context.Background(), "route")
	if err != nil {
		t.Fatalf("GetIRRDatabase() error: %v", err)
	}
	if len(got.Objects) != 1 {
		t.Errorf("objects = %d, want 1", len(got.Objects))
	}
}

func TestGetIRRDatabaseFetchPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.ServeDated(w, r, testutil.SampleIRRDump)
	}))
	defer srv.Close()
	client := NewClient(
		WithHTTPClient(srv.Client()),
		WithFTPBaseURL(srv.URL+"/"),
		WithCacheTTL(0), // force fetch path
		WithJitter(0, 0),
	)
	got, err := client.GetIRRDatabase(context.Background(), "inetnum")
	if err != nil {
		t.Fatalf("GetIRRDatabase() error: %v", err)
	}
	if len(got.Objects) != 2 {
		t.Errorf("objects = %d, want 2", len(got.Objects))
	}
}

// TestGetIRRDatabaseFetchError covers the cache-miss + fetch-error branch of
// GetIRRDatabase (the FetchIRRDatabase error propagates).
func TestGetIRRDatabaseFetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	client := NewClient(
		WithHTTPClient(srv.Client()),
		WithFTPBaseURL(srv.URL+"/"),
		WithCacheTTL(0),
		WithJitter(0, 0),
		WithMaxConcurrentDownloads(0),
	)
	if _, err := client.GetIRRDatabase(context.Background(), "inetnum"); err == nil {
		t.Error("expected error from GetIRRDatabase when fetch fails")
	}
}
