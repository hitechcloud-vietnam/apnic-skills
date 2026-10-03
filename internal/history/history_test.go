package history

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/testutil"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

func TestFetchHistoricalDelegatedInvalidDate(t *testing.T) {
	client := transport.NewClient()
	_, err := FetchHistoricalDelegated(context.Background(), client, "invalid")
	if err == nil {
		t.Error("expected error for invalid date format")
	}
}

func TestFetchHistoricalExtendedInvalidDate(t *testing.T) {
	client := transport.NewClient()
	_, err := FetchHistoricalExtended(context.Background(), client, "2026")
	if err == nil {
		t.Error("expected error for invalid date format")
	}
}

func TestFetchHistoricalAssignedInvalidDate(t *testing.T) {
	client := transport.NewClient()
	_, err := FetchHistoricalAssigned(context.Background(), client, "abc")
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestFetchHistoricalLegacyInvalidDate(t *testing.T) {
	client := transport.NewClient()
	_, err := FetchHistoricalLegacy(context.Background(), client, "xyz")
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestFetchHistoricalDelegated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.ServeDated(w, r, testutil.SampleDelegatedData)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchHistoricalDelegated(context.Background(), client, "20260627")
	if err != nil {
		t.Fatalf("FetchHistoricalDelegated() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchHistoricalExtended(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.ServeDated(w, r, testutil.SampleExtendedData)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchHistoricalExtended(context.Background(), client, "20260627")
	if err != nil {
		t.Fatalf("FetchHistoricalExtended() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchHistoricalAssigned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.ServeDated(w, r, testutil.SampleAssignedData)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchHistoricalAssigned(context.Background(), client, "20260627")
	if err != nil {
		t.Fatalf("FetchHistoricalAssigned() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchHistoricalLegacy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.ServeDated(w, r, testutil.SampleLegacyData)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchHistoricalLegacy(context.Background(), client, "20260627")
	if err != nil {
		t.Fatalf("FetchHistoricalLegacy() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchDelegatedByYearInvalid(t *testing.T) {
	client := transport.NewClient()
	_, err := FetchDelegatedByYear(context.Background(), client, 2000)
	if err == nil {
		t.Error("expected error for invalid year")
	}
}

func TestFetchDelegatedByYear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.ServeDated(w, r, testutil.SampleDelegatedData)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchDelegatedByYear(context.Background(), client, 2026)
	if err != nil {
		t.Fatalf("FetchDelegatedByYear() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchDelegatedByYearHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchDelegatedByYear(context.Background(), client, 2026)
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestFetchExtendedByYearInvalid(t *testing.T) {
	client := transport.NewClient()
	_, err := FetchExtendedByYear(context.Background(), client, 1999)
	if err == nil {
		t.Error("expected error for invalid year")
	}
}

func TestFetchExtendedByYear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.ServeDated(w, r, testutil.SampleExtendedData)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchExtendedByYear(context.Background(), client, 2026)
	if err != nil {
		t.Fatalf("FetchExtendedByYear() error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries")
	}
}

func TestFetchExtendedByYearHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchExtendedByYear(context.Background(), client, 2026)
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestListAvailableYears(t *testing.T) {
	years := ListAvailableYears()
	if len(years) == 0 {
		t.Error("expected non-empty years list")
	}
	if years[0] != 2001 {
		t.Errorf("first year = %d, want 2001", years[0])
	}
}
