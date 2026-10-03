package query

import (
	"context"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/testutil"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchTelemetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".md5") {
			w.Write([]byte(testutil.SampleTelemetryMD5))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(testutil.SampleTelemetryJSON))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithFTPBaseURL(srv.URL+"/"), transport.WithJitter(0, 0))

	// Latest.
	tel, err := FetchTelemetry(context.Background(), client, "")
	if err != nil {
		t.Fatalf("FetchTelemetry() error: %v", err)
	}
	if tel.RDAP.TotalQueries != 3070925 {
		t.Errorf("total_queries = %d, want 3070925", tel.RDAP.TotalQueries)
	}
	if tel.RDAP.TotalASNs != 1737 {
		t.Errorf("total_asns = %d, want 1737", tel.RDAP.TotalASNs)
	}
	if tel.RDAP.QueryTypeDistribution["ip"] != 3030224 {
		t.Errorf("qtd[ip] = %d", tel.RDAP.QueryTypeDistribution["ip"])
	}
	if len(tel.RDAP.ASNs) != 1 {
		t.Fatalf("asns = %d, want 1", len(tel.RDAP.ASNs))
	}
	if tel.RDAP.ASNs[0].ASN != "45102" || tel.RDAP.ASNs[0].QueryCount != 2274463 {
		t.Errorf("asn[0] = %+v", tel.RDAP.ASNs[0])
	}

	// Archived snapshot by date.
	tel2, err := FetchTelemetry(context.Background(), client, "20260701")
	if err != nil {
		t.Fatalf("FetchTelemetry(date) error: %v", err)
	}
	if tel2.RDAP.TotalQueries != 3070925 {
		t.Errorf("archived total_queries = %d", tel2.RDAP.TotalQueries)
	}

	// MD5 sidecar.
	md5, err := FetchTelemetryMD5(context.Background(), client, "")
	if err != nil {
		t.Fatalf("FetchTelemetryMD5() error: %v", err)
	}
	if md5 != "0123456789abcdef0123456789abcdef" {
		t.Errorf("md5 = %q", md5)
	}
}

func TestFetchTelemetryInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithFTPBaseURL(srv.URL+"/"), transport.WithJitter(0, 0))
	if _, err := FetchTelemetry(context.Background(), client, ""); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFetchTelemetryHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithFTPBaseURL(srv.URL+"/"), transport.WithJitter(0, 0))
	if _, err := FetchTelemetry(context.Background(), client, ""); err == nil {
		t.Error("expected error on HTTP 500")
	}
	if _, err := FetchTelemetryMD5(context.Background(), client, ""); err == nil {
		t.Error("expected error on HTTP 500 for md5")
	}
}
