package query

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

func TestParseTransfersData(t *testing.T) {
	result, err := parseTransfersData(testutil.SampleTransfersJSON)
	if err != nil {
		t.Fatalf("parseTransfersData() error: %v", err)
	}
	if len(result.Transfers) != 2 {
		t.Fatalf("transfers count = %d, want 2", len(result.Transfers))
	}

	t1 := result.Transfers[0]
	if t1.Type != "RESOURCE_TRANSFER" {
		t.Errorf("type = %q, want RESOURCE_TRANSFER", t1.Type)
	}
	if t1.SourceOrganization.Name != "Org A" {
		t.Errorf("source org = %q, want Org A", t1.SourceOrganization.Name)
	}
	if t1.RecipientOrganization.Name != "Org B" {
		t.Errorf("recipient org = %q, want Org B", t1.RecipientOrganization.Name)
	}
	if t1.IPv4Nets == nil || len(t1.IPv4Nets.TransferSet) != 1 {
		t.Fatal("expected IPv4 nets with 1 entry")
	}
	if t1.IPv4Nets.TransferSet[0].StartAddress != "1.2.3.0" {
		t.Errorf("start address = %q, want 1.2.3.0", t1.IPv4Nets.TransferSet[0].StartAddress)
	}
	if t1.IPv6Nets != nil {
		t.Error("expected nil IPv6 nets for first transfer")
	}
	if t1.ASNs != nil {
		t.Error("expected nil ASNs for first transfer")
	}

	t2 := result.Transfers[1]
	if t2.Type != "INTER_RIR_TRANSFER" {
		t.Errorf("type = %q, want INTER_RIR_TRANSFER", t2.Type)
	}
	if t2.IPv6Nets == nil || len(t2.IPv6Nets.TransferSet) != 1 {
		t.Fatal("expected IPv6 nets with 1 entry")
	}
	if t2.ASNs == nil || len(t2.ASNs.TransferSet) != 1 {
		t.Fatal("expected ASNs with 1 entry")
	}
	if t2.ASNs.TransferSet[0].StartASN != 64512 {
		t.Errorf("start ASN = %d, want 64512", t2.ASNs.TransferSet[0].StartASN)
	}
}

func TestParseTransfersMetadata(t *testing.T) {
	result, err := parseTransfersData(testutil.SampleTransfersJSON)
	if err != nil {
		t.Fatalf("parseTransfersData() error: %v", err)
	}
	if result.Metadata.Producer != "APNIC" {
		t.Errorf("producer = %q, want APNIC", result.Metadata.Producer)
	}
	if result.Metadata.StatsVersion != "4.0" {
		t.Errorf("stats version = %q, want 4.0", result.Metadata.StatsVersion)
	}
	if result.Metadata.ProductionDate.IsZero() {
		t.Error("expected non-zero production date")
	}
	if result.Metadata.StartDate.IsZero() {
		t.Error("expected non-zero start date")
	}
	if result.Metadata.EndDate.IsZero() {
		t.Error("expected non-zero end date")
	}
}

func TestParseTransfersDataInvalidJSON(t *testing.T) {
	_, err := parseTransfersData("invalid json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseTransfersWithNullNets(t *testing.T) {
	data := `{
  "version": {
    "producer": "APNIC",
    "production_date": "2026-06-25T18:00:04Z",
    "remarks": [],
    "UTC_offset": 10,
    "stats_version": "4.0",
    "records_interval": {
      "start_date": "2010-04-07T00:29:32Z",
      "end_date": "2026-06-25T18:00:04Z"
    }
  },
  "transfers": [
    {
      "transfer_date": "2020-01-15T00:00:00Z",
      "type": "RESOURCE_TRANSFER",
      "source_rir": "APNIC",
      "recipient_rir": "APNIC",
      "source_organization": null,
      "recipient_organization": null,
      "ip4nets": null,
      "ip6nets": null,
      "asns": null
    }
  ]
}`
	result, err := parseTransfersData(data)
	if err != nil {
		t.Fatalf("parseTransfersData() error: %v", err)
	}
	if len(result.Transfers) != 1 {
		t.Fatalf("transfers count = %d, want 1", len(result.Transfers))
	}
	t1 := result.Transfers[0]
	if t1.SourceOrganization.Name != "" {
		t.Error("expected empty source org")
	}
	if t1.IPv4Nets != nil {
		t.Error("expected nil IPv4 nets")
	}
}

func TestParseTransfersEmptyNetSets(t *testing.T) {
	data := `{
  "version": {
    "producer": "APNIC",
    "production_date": "2026-06-25T18:00:04Z",
    "remarks": [],
    "UTC_offset": 10,
    "stats_version": "4.0",
    "records_interval": {
      "start_date": "2010-04-07T00:29:32Z",
      "end_date": "2026-06-25T18:00:04Z"
    }
  },
  "transfers": [
    {
      "transfer_date": "not-a-date",
      "type": "RESOURCE_TRANSFER",
      "source_rir": "APNIC",
      "recipient_rir": "APNIC",
      "ip4nets": {"transfer_set": []},
      "ip6nets": {"transfer_set": []},
      "asns": {"transfer_set": []}
    }
  ]
}`
	result, err := parseTransfersData(data)
	if err != nil {
		t.Fatalf("parseTransfersData() error: %v", err)
	}
	if len(result.Transfers) != 1 {
		t.Fatalf("transfers count = %d, want 1", len(result.Transfers))
	}
	// Empty transfer sets should result in nil nets
	if result.Transfers[0].IPv4Nets != nil {
		t.Error("expected nil IPv4Nets for empty set")
	}
	if result.Transfers[0].IPv6Nets != nil {
		t.Error("expected nil IPv6Nets for empty set")
	}
	if result.Transfers[0].ASNs != nil {
		t.Error("expected nil ASNs for empty set")
	}
	// Invalid date should result in zero time
	if !result.Transfers[0].TransferDate.IsZero() {
		t.Error("expected zero transfer date for invalid format")
	}
}

func TestFetchTransfers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(testutil.SampleTransfersJSON))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchTransfers(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchTransfers() error: %v", err)
	}
	if len(result.Transfers) != 2 {
		t.Errorf("transfers count = %d, want 2", len(result.Transfers))
	}
}

func TestFetchTransfersByYear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(testutil.SampleTransfersJSON))
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	result, err := FetchTransfersByYear(context.Background(), client, 2026)
	if err != nil {
		t.Fatalf("FetchTransfersByYear() error: %v", err)
	}
	if len(result.Transfers) != 2 {
		t.Errorf("transfers count = %d, want 2", len(result.Transfers))
	}
}

func TestFetchTransfersHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchTransfers(context.Background(), client)
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestFetchTransfersByYearHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := transport.NewClient(
		transport.WithHTTPClient(server.Client()),
		transport.WithStatsBaseURL(server.URL+"/"),
		transport.WithCacheTTL(1*time.Hour),
	)

	_, err := FetchTransfersByYear(context.Background(), client, 2026)
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestParseTransfersAll(t *testing.T) {
	r, err := parseTransfersAll(testutil.SampleTransfersAll)
	if err != nil {
		t.Fatalf("parseTransfersAll() error: %v", err)
	}
	if len(r.Records) != 3 {
		t.Fatalf("records = %d, want 3", len(r.Records))
	}
	if r.Records[0].ResourceType != "asn" || r.Records[0].Resource != "45745" {
		t.Errorf("first record = %+v", r.Records[0])
	}
	if r.Records[0].TransferType != "M&A" {
		t.Errorf("transfer type = %q, want M&A", r.Records[0].TransferType)
	}
	if !r.Records[0].TransferDate.Equal(time.Date(2012, 6, 20, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("transfer date = %v", r.Records[0].TransferDate)
	}
	if r.Records[2].FromRIR != "ARIN" || r.Records[2].ToRIR != "APNIC" {
		t.Errorf("inter-rir record = %+v", r.Records[2])
	}
}

func TestParseTransfersAll_EmptyAndComments(t *testing.T) {
	r, err := parseTransfersAll("# only comments\n\n# more\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Records) != 0 {
		t.Errorf("expected 0 records, got %d", len(r.Records))
	}
}

// TestParseTransfersAll_ShortRowSkipped covers the len(parts)<11 skip branch:
// a malformed row with too few fields is skipped without error.
func TestParseTransfersAll_ShortRowSkipped(t *testing.T) {
	data := `resource_type|resource|from_organisation|from_economy|from_rir|previous_delegation_date|to_organisation|to_economy|to_rir|transfer_date|transfer_type
short|row|only|three
asn|45745|Gambit Group Pty Ltd|AU|APNIC|20090417|Bathurst One Pty Limited|AU|APNIC|20120620|M&A
`
	r, err := parseTransfersAll(data)
	if err != nil {
		t.Fatalf("parseTransfersAll() error: %v", err)
	}
	if len(r.Records) != 1 {
		t.Errorf("records = %d, want 1 (short row skipped)", len(r.Records))
	}
}

func TestFetchTransfersAll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".md5") {
			w.Write([]byte(testutil.SampleTransfersAllMD5))
			return
		}
		if strings.HasSuffix(r.URL.Path, ".asc") {
			w.Write([]byte("-----BEGIN PGP SIGNATURE-----\nmock\n-----END PGP SIGNATURE-----"))
			return
		}
		// transfer-all-apnic-latest and per-year archives.
		w.Write([]byte(testutil.SampleTransfersAll))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithFTPBaseURL(srv.URL+"/"), transport.WithJitter(0, 0))

	r, err := FetchTransfersAll(context.Background(), client, "")
	if err != nil {
		t.Fatalf("FetchTransfersAll() error: %v", err)
	}
	if len(r.Records) != 3 {
		t.Errorf("records = %d, want 3", len(r.Records))
	}

	// By date archive.
	ry, err := FetchTransfersAll(context.Background(), client, "20200615")
	if err != nil {
		t.Fatalf("FetchTransfersAll(date) error: %v", err)
	}
	if len(ry.Records) != 3 {
		t.Errorf("date records = %d, want 3", len(ry.Records))
	}

	md5, err := FetchTransfersAllMD5(context.Background(), client, "")
	if err != nil {
		t.Fatalf("FetchTransfersAllMD5() error: %v", err)
	}
	if md5 != "0123456789abcdef0123456789abcdef" {
		t.Errorf("md5 = %q", md5)
	}

	asc, err := FetchTransfersAllASC(context.Background(), client, "")
	if err != nil {
		t.Fatalf("FetchTransfersAllASC() error: %v", err)
	}
	if !strings.Contains(asc, "PGP SIGNATURE") {
		t.Errorf("asc = %q", asc)
	}
}

func TestFetchTransfersAllHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithFTPBaseURL(srv.URL+"/"), transport.WithJitter(0, 0))
	if _, err := FetchTransfersAll(context.Background(), client, ""); err == nil {
		t.Error("expected error on HTTP 500")
	}
	if _, err := FetchTransfersAllMD5(context.Background(), client, ""); err == nil {
		t.Error("expected error on HTTP 500 for md5")
	}
	if _, err := FetchTransfersAllASC(context.Background(), client, ""); err == nil {
		t.Error("expected error on HTTP 500 for asc")
	}
}
