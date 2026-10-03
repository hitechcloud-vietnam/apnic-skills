package apnic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/testutil"
)

// This file is a smoke-test layer for the root *Client wrapper methods in
// methods.go and the With* Option factories in reexport.go. The subpackage
// tests under internal/ already exercise the real parsing logic through the
// free functions (stats.FetchXxx / query.FetchXxx / history.FetchXxx); these
// tests only need to drive the thin forwarding wrappers so the root-package
// coverage reflects that the wrappers actually wire (root *Client) -> embedded
// *transport.Client -> subpackage function correctly.
//
// To do that without spinning up one server per endpoint, rootTestHandler
// routes by URL path shape to the testutil sample fixtures (and a handful of
// inline fixtures for the endpoints testutil does not carry: BGP thyme files,
// RRDP XML, REx JSON, telemetry JSON, IRR serial, transfers-all). Each test
// constructs a single httptest.Server with this handler and points every
// relevant With*BaseURL option at it.

const (
	rootBgpSummary = `Analysis Summary
----------------

BGP routing table entries examined:                             1059904
Total ASes present in the Internet Routing Table:                 78800
Average AS path length visible in the Internet Routing Table:       4.7
Number of addresses announced to Internet:                   3119677184
`

	rootBgpRawTable = `1.0.0.0/24	13335
1.1.1.0/24	13335
1.0.4.0/24	38803
`

	rootBgpBadPrefixes = `Prefixes longer than /24 announced
----------------------------------
Origin AS       Address
13335           1.1.1.128/25
38803           1.0.4.128/25
`

	rootBgpPerPrefixLength = `Number of prefixes by prefix length
-----------------------------------
/8:12  /16:90000  /24:400000  /48:12000
`

	rootBgpUsedAutnums = `13335 CLOUDFLARE - Cloudflare, Inc, US
38803 AS-APNICLABS-AS-AP - APNIC Labs, AU
`

	rootBgpSpar = `Prefixes from the Special Purpose Address Registry
---------------------------------------------------
Prefix	Origin AS	Description
0.0.0.0/8	0	"This network"
240.0.0.0/4	0	Reserved
`

	rootBgpSinglePfx = `Number of ASNs announcing fewer than 20 prefixes
------------------------------------------------
No.	Prefix count	ASN count	RIR
1	1	27539	Global
2	5	1234	APNIC
`

	rootRRDPNotification = `<notification xmlns="http://www.ripe.net/rpki/rrdp" version="1" session_id="8dad0cc8-0bc8-4021" serial="65148">
  <snapshot uri="/snapshot.xml" hash="479c1351cc5372febc3487abe80bad01ea04118a78f59100004c213f944022d9"/>
  <delta serial="65148" uri="/delta-65148.xml" hash="45ff4de1ac87c9b41009b5e71d7ff175adb01ce69af26bfbe5b7093a027cc0c5"/>
</notification>`

	rootRRDPSnapshot = `<snapshot version="1" session_id="8dad0cc8" serial="65148" xmlns="http://www.ripe.net/rpki/rrdp">
<publish uri="rsync://rpki.apnic.net/rep/A9110009/roa1.roa">AAAABASE64BODY1</publish>
<withdraw uri="rsync://rpki.apnic.net/rep/A9110009/old.roa"/>
</snapshot>`

	rootRExUserNetwork     = `{"ip":"1.1.1.1","prefix":"1.1.1.0/24","asn":13335,"economy":"AU"}`
	rootRExResources       = `{"items":[{"resource":"1.1.1.0/24","type":"ipv4","opaqueId":"abc","holderName":"APNIC-LABS","rir":"apnic","cc":"AU","delegationDate":"2011-08-11"}]}`
	rootRExHolder          = `{"opaqueId":"abc","registry":"apnic","holderName":"APNIC-LABS","asns":["AS13335"],"asnsCount":1,"ipv4":["1.1.1.0/24"],"ipv4_24Count":1.0}`
	rootRExHoldersCount    = `{"count":129665}`
	rootIRRCurrentSerial   = "12345"
	rootTransfersAllLatest = testutil.SampleTransfersAll
)

// rootTestHandler routes every endpoint shape onto one httptest.Server. .gz
// requests are served gzip-compressed (matching APNIC's archive layout); JSON
// and text are served plain. Unknown paths fall through to the combined
// stats/RDAP router so that delegated/extended/assigned/legacy/transfers/
// changes/RDAP requests reuse the shared fixtures.
func rootTestHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	combined := testutil.CombinedHandler()
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// RRDP: notification at /notification.xml, snapshot at any /snapshot*.xml.
		if strings.HasSuffix(path, "/notification.xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(rootRRDPNotification))
			return
		}
		if strings.HasSuffix(path, ".xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(rootRRDPSnapshot))
			return
		}

		// REx API: /v1/...
		switch path {
		case "/v1/user-network":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(rootRExUserNetwork))
			return
		case "/v1/resources":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(rootRExResources))
			return
		case "/v1/holder":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(rootRExHolder))
			return
		case "/v1/holders/unique-count":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(rootRExHoldersCount))
			return
		}

		// IRR current serial.
		if strings.HasSuffix(path, "/APNIC.CURRENTSERIAL") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(rootIRRCurrentSerial))
			return
		}

		// Telemetry JSON (latest + dated).
		if strings.Contains(path, "whois-rdap-stats") && strings.HasSuffix(path, ".json") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(testutil.SampleTelemetryJSON))
			return
		}

		// Transfers-all cumulative log (latest + dated), excluding the JSON
		// transfers_latest handled by the combined router below.
		if strings.Contains(path, "transfers-all") && strings.Contains(path, "transfer-all-apnic") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(rootTransfersAllLatest))
			return
		}

		// BGP thyme files: /<source>/<file>
		switch path {
		case "/current/data-summary", "/au/data-summary", "/hk/data-summary":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(rootBgpSummary))
			return
		case "/current/data-raw-table", "/au/data-raw-table", "/hk/data-raw-table":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(rootBgpRawTable))
			return
		case "/current/data-badpfx-nos", "/au/data-badpfx-nos", "/hk/data-badpfx-nos":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(rootBgpBadPrefixes))
			return
		case "/current/data-pfx-nos", "/au/data-pfx-nos", "/hk/data-pfx-nos":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(rootBgpPerPrefixLength))
			return
		case "/current/data-used-autnums", "/au/data-used-autnums", "/hk/data-used-autnums":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(rootBgpUsedAutnums))
			return
		case "/current/data-spar", "/au/data-spar", "/hk/data-spar":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(rootBgpSpar))
			return
		case "/current/data-singlepfx", "/au/data-singlepfx", "/hk/data-singlepfx":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(rootBgpSinglePfx))
			return
		}

		// IRR db dumps end in apnic.db.<type>.gz — serve gzip via ServeDated so
		// FetchText decompresses them. Delegate to the combined router for the
		// remaining stats/RDAP/transfers/changes paths.
		combined(w, r)
	}
}

// newRootTestClient builds a Client whose every base URL points at the test
// server, with jitter zeroed so tests run fast and deterministically.
func newRootTestClient(t *testing.T) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(rootTestHandler(t))
	client := NewClient(
		WithHTTPClient(srv.Client()),
		WithStatsBaseURL(srv.URL+"/"),
		WithRDAPBaseURL(srv.URL),
		WithFTPBaseURL(srv.URL+"/"),
		WithThymeBaseURL(srv.URL),
		WithRRDPBaseURL(srv.URL),
		WithRExBaseURL(srv.URL),
		WithJitter(0, 0),
		WithCacheTTL(0),
	)
	return client, srv
}

// --- stats: delegated -----------------------------------------------------

func TestRootFetchDelegatedEntries(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchDelegatedEntries(context.Background())
	if err != nil {
		t.Fatalf("FetchDelegatedEntries: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected delegated entries")
	}
}

func TestRootFetchDelegatedEntriesByDate(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchDelegatedEntriesByDate(context.Background(), "20260626")
	if err != nil {
		t.Fatalf("FetchDelegatedEntriesByDate: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected delegated entries by date")
	}
}

func TestRootFetchDelegatedResult(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchDelegatedResult(context.Background(), "20260626")
	if err != nil {
		t.Fatalf("FetchDelegatedResult: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil DelegatedResult")
	}
}

func TestRootFetchDelegatedResultByYear(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchDelegatedResultByYear(context.Background(), 2026)
	if err != nil {
		t.Fatalf("FetchDelegatedResultByYear: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil DelegatedResult")
	}
}

// --- stats: extended / assigned / ipv6 / legacy --------------------------

func TestRootFetchExtendedResult(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchExtendedResult(context.Background(), "20260626")
	if err != nil {
		t.Fatalf("FetchExtendedResult: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil ExtendedResult")
	}
}

func TestRootFetchAssignedResult(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchAssignedResult(context.Background(), "20260626")
	if err != nil {
		t.Fatalf("FetchAssignedResult: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil AssignedResult")
	}
}

func TestRootFetchIPv6AssignedResult(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchIPv6AssignedResult(context.Background(), "20260626")
	if err != nil {
		t.Fatalf("FetchIPv6AssignedResult: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil IPv6AssignedResult")
	}
}

func TestRootFetchLegacyResult(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchLegacyResult(context.Background(), "20260626")
	if err != nil {
		t.Fatalf("FetchLegacyResult: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil LegacyResult")
	}
}

// --- history -------------------------------------------------------------

func TestRootFetchHistoricalDelegated(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchHistoricalDelegated(context.Background(), "20260626")
	if err != nil {
		t.Fatalf("FetchHistoricalDelegated: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil DelegatedResult")
	}
}

func TestRootFetchHistoricalExtended(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchHistoricalExtended(context.Background(), "20260626")
	if err != nil {
		t.Fatalf("FetchHistoricalExtended: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil ExtendedResult")
	}
}

func TestRootFetchHistoricalAssigned(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchHistoricalAssigned(context.Background(), "20260626")
	if err != nil {
		t.Fatalf("FetchHistoricalAssigned: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil AssignedResult")
	}
}

func TestRootFetchHistoricalLegacy(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchHistoricalLegacy(context.Background(), "20260626")
	if err != nil {
		t.Fatalf("FetchHistoricalLegacy: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil LegacyResult")
	}
}

func TestRootFetchDelegatedByYear(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchDelegatedByYear(context.Background(), 2026)
	if err != nil {
		t.Fatalf("FetchDelegatedByYear: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil DelegatedResult")
	}
}

func TestRootFetchExtendedByYear(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchExtendedByYear(context.Background(), 2026)
	if err != nil {
		t.Fatalf("FetchExtendedByYear: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil ExtendedResult")
	}
}

// --- query: RDAP ---------------------------------------------------------

func TestRootRDAPLookupIP(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.RDAPLookupIP(context.Background(), "1.1.1.1")
	if err != nil {
		t.Fatalf("RDAPLookupIP: %v", err)
	}
	if got == nil || got.Handle == "" {
		t.Error("expected non-empty RDAPNetwork")
	}
}

func TestRootRDAPLookupCIDR(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.RDAPLookupCIDR(context.Background(), "1.1.1.0/24")
	if err != nil {
		t.Fatalf("RDAPLookupCIDR: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil RDAPNetwork")
	}
}

func TestRootRDAPLookupASN(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.RDAPLookupASN(context.Background(), 13335)
	if err != nil {
		t.Fatalf("RDAPLookupASN: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil RDAPAutnum")
	}
}

func TestRootRDAPLookupDomain(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.RDAPLookupDomain(context.Background(), "1.0.0.1.in-addr.arpa")
	if err != nil {
		t.Fatalf("RDAPLookupDomain: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil RDAPDomain")
	}
}

func TestRootRDAPLookupEntity(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.RDAPLookupEntity(context.Background(), "AIC3-AP")
	if err != nil {
		t.Fatalf("RDAPLookupEntity: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil RDAPEntity")
	}
}

func TestRootRDAPSearchEntities(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.RDAPSearchEntities(context.Background(), "fn", "APNIC")
	if err != nil {
		t.Fatalf("RDAPSearchEntities: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil RDAPSearchResult")
	}
}

func TestRootRDAPSearchDomains(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.RDAPSearchDomains(context.Background(), "1.in-addr.arpa")
	if err != nil {
		t.Fatalf("RDAPSearchDomains: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil RDAPDomainSearchResult")
	}
}

func TestRootRDAPHelp(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.RDAPHelp(context.Background())
	if err != nil {
		t.Fatalf("RDAPHelp: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil RDAPHelpInfo")
	}
}

// --- query: REx ----------------------------------------------------------

func TestRootFetchRExUserNetwork(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchRExUserNetwork(context.Background())
	if err != nil {
		t.Fatalf("FetchRExUserNetwork: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil RExUserNetwork")
	}
}

func TestRootFetchRExResources(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchRExResources(context.Background(), "ipv4")
	if err != nil {
		t.Fatalf("FetchRExResources: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil RExResourcesResult")
	}
}

func TestRootFetchRExHolder(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchRExHolder(context.Background(), "abc", "apnic")
	if err != nil {
		t.Fatalf("FetchRExHolder: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil RExHolder")
	}
}

func TestRootFetchRExHoldersUniqueCount(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchRExHoldersUniqueCount(context.Background())
	if err != nil {
		t.Fatalf("FetchRExHoldersUniqueCount: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil RExHoldersCount")
	}
}

// --- query: whois --------------------------------------------------------

func TestRootQueryWhois(t *testing.T) {
	addr, cleanup := testutil.MockWhoisServer(t, testutil.SampleWhoisResponse)
	defer cleanup()
	c := NewClient(WithWhoisServer(addr), WithWhoisTimeout(2*time.Second), WithJitter(0, 0))
	got, err := c.QueryWhois(context.Background(), "1.1.1.1")
	if err != nil {
		t.Fatalf("QueryWhois: %v", err)
	}
	if !strings.Contains(got, "inetnum") {
		t.Errorf("unexpected whois response: %q", got)
	}
}

func TestRootQueryWhoisIP(t *testing.T) {
	addr, cleanup := testutil.MockWhoisServer(t, testutil.SampleWhoisResponse)
	defer cleanup()
	c := NewClient(WithWhoisServer(addr), WithWhoisTimeout(2*time.Second), WithJitter(0, 0))
	got, err := c.QueryWhoisIP(context.Background(), "1.1.1.1")
	if err != nil {
		t.Fatalf("QueryWhoisIP: %v", err)
	}
	if got == nil || got.Country != "AU" {
		t.Errorf("unexpected WhoisInfo: %+v", got)
	}
}

func TestRootQueryWhoisASN(t *testing.T) {
	addr, cleanup := testutil.MockWhoisServer(t, testutil.SampleWhoisResponse)
	defer cleanup()
	c := NewClient(WithWhoisServer(addr), WithWhoisTimeout(2*time.Second), WithJitter(0, 0))
	got, err := c.QueryWhoisASN(context.Background(), 13335)
	if err != nil {
		t.Fatalf("QueryWhoisASN: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil WhoisInfo")
	}
}

// --- query: IRR ----------------------------------------------------------

func TestRootFetchIRRCurrentSerial(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchIRRCurrentSerial(context.Background())
	if err != nil {
		t.Fatalf("FetchIRRCurrentSerial: %v", err)
	}
	if got != 12345 {
		t.Errorf("serial = %d, want 12345", got)
	}
}

// --- query: RRDP ---------------------------------------------------------

func TestRootFetchRRDPNotification(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchRRDPNotification(context.Background())
	if err != nil {
		t.Fatalf("FetchRRDPNotification: %v", err)
	}
	if got == nil || got.Serial != 65148 {
		t.Errorf("unexpected RRDPNotification: %+v", got)
	}
}

func TestRootFetchRRDPSnapshot(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchRRDPSnapshot(context.Background(), srv.URL+"/snapshot.xml")
	if err != nil {
		t.Fatalf("FetchRRDPSnapshot: %v", err)
	}
	if got == nil || len(got.Published) == 0 {
		t.Errorf("unexpected RPKISnapshot: %+v", got)
	}
}

// --- query: BGP ----------------------------------------------------------

func TestRootFetchBGPSummary(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchBGPSummary(context.Background())
	if err != nil {
		t.Fatalf("FetchBGPSummary: %v", err)
	}
	if got == nil || len(got.Entries) == 0 {
		t.Error("expected non-empty BGPSummary")
	}
}

func TestRootFetchBGPRawTable(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchBGPRawTable(context.Background())
	if err != nil {
		t.Fatalf("FetchBGPRawTable: %v", err)
	}
	if got == nil || len(got.Routes) == 0 {
		t.Error("expected non-empty BGPRawTable")
	}
}

func TestRootFetchBGPASNMap(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchBGPASNMap(context.Background())
	if err != nil {
		t.Fatalf("FetchBGPASNMap: %v", err)
	}
	if got == nil || len(got.ASNs) == 0 {
		t.Error("expected non-empty BGPASNMap")
	}
}

func TestRootFetchBGPBadPrefixes(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchBGPBadPrefixes(context.Background(), "current")
	if err != nil {
		t.Fatalf("FetchBGPBadPrefixes: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil BGPBadPrefixes")
	}
}

func TestRootFetchBGPPerPrefixLength(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchBGPPerPrefixLength(context.Background(), "current")
	if err != nil {
		t.Fatalf("FetchBGPPerPrefixLength: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil BGPPerPrefixLength")
	}
}

func TestRootFetchBGPUsedAutnums(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchBGPUsedAutnums(context.Background(), "current")
	if err != nil {
		t.Fatalf("FetchBGPUsedAutnums: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil BGPUsedAutnums")
	}
}

func TestRootFetchBGPSparPrefixes(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchBGPSparPrefixes(context.Background(), "current")
	if err != nil {
		t.Fatalf("FetchBGPSparPrefixes: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil BGPSparPrefixes")
	}
}

func TestRootFetchBGPSinglePfx(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchBGPSinglePfx(context.Background(), "current")
	if err != nil {
		t.Fatalf("FetchBGPSinglePfx: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil BGPSinglePfx")
	}
}

// --- query: telemetry ----------------------------------------------------

func TestRootFetchTelemetry(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchTelemetry(context.Background(), "")
	if err != nil {
		t.Fatalf("FetchTelemetry: %v", err)
	}
	if got == nil || got.RDAP.TotalQueries == 0 {
		t.Errorf("unexpected telemetry: %+v", got)
	}
}

// --- query: transfers ----------------------------------------------------

func TestRootFetchTransfers(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchTransfers(context.Background())
	if err != nil {
		t.Fatalf("FetchTransfers: %v", err)
	}
	if got == nil || len(got.Transfers) == 0 {
		t.Error("expected non-empty TransfersResult")
	}
}

func TestRootFetchTransfersByYear(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchTransfersByYear(context.Background(), 2020)
	if err != nil {
		t.Fatalf("FetchTransfersByYear: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil TransfersResult")
	}
}

func TestRootFetchTransfersAll(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchTransfersAll(context.Background(), "")
	if err != nil {
		t.Fatalf("FetchTransfersAll: %v", err)
	}
	if got == nil || len(got.Records) == 0 {
		t.Error("expected non-empty TransfersAllResult")
	}
}

// --- query: changes ------------------------------------------------------

func TestRootFetchChanges(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchChanges(context.Background())
	if err != nil {
		t.Fatalf("FetchChanges: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil ChangesResult")
	}
}

func TestRootFetchChangesByDate(t *testing.T) {
	c, srv := newRootTestClient(t)
	defer srv.Close()
	got, err := c.FetchChangesByDate(context.Background(), "20260626")
	if err != nil {
		t.Fatalf("FetchChangesByDate: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil ChangesResult")
	}
}
