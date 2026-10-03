package query

import (
	"context"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sampleBGPSummary mirrors the real thyme data-summary format.
const sampleBGPSummary = `Analysis Summary
----------------

BGP routing table entries examined:                             1059904
    Prefixes after maximum aggregation (per Origin AS):          407882
    Deaggregation factor:                                          2.60
Total ASes present in the Internet Routing Table:                 78800
    Prefixes per ASN:                                             13.45
Average AS path length visible in the Internet Routing Table:       4.7
Number of addresses announced to Internet:                   3119677184
    Percentage of available address space announced:               84.3
`

const sampleBGPRawTable = `1.0.0.0/24	13335
1.0.4.0/24	38803
1.0.5.0/24	38803
1.0.6.0/24	38803
1.1.1.0/24	13335
`

func TestParseBGPSummary(t *testing.T) {
	s := parseBGPSummary(sampleBGPSummary)
	if len(s.Entries) == 0 {
		t.Fatal("expected non-empty entries")
	}
	// Title and dash lines must be skipped.
	for _, e := range s.Entries {
		if e.Key == "Analysis Summary" {
			t.Errorf("title line should be skipped: %+v", e)
		}
		if strings.HasPrefix(e.Key, "-") {
			t.Errorf("dash line should be skipped: %+v", e)
		}
	}
	// Find a known entry.
	var found bool
	for _, e := range s.Entries {
		if e.Key == "BGP routing table entries examined" && e.Value == "1059904" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find 'BGP routing table entries examined' entry: %+v", s.Entries)
	}
}

func TestParseBGPSummary_Empty(t *testing.T) {
	s := parseBGPSummary("")
	if len(s.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(s.Entries))
	}
}

func TestParseBGPRawTable(t *testing.T) {
	rt, err := parseBGPRawTable(sampleBGPRawTable)
	if err != nil {
		t.Fatalf("parseBGPRawTable() error: %v", err)
	}
	if len(rt.Routes) != 5 {
		t.Fatalf("routes = %d, want 5", len(rt.Routes))
	}
	if rt.Routes[0].Prefix != "1.0.0.0/24" || rt.Routes[0].ASN != "13335" {
		t.Errorf("first route = %+v", rt.Routes[0])
	}
	if rt.Routes[4].Prefix != "1.1.1.0/24" || rt.Routes[4].ASN != "13335" {
		t.Errorf("last route = %+v", rt.Routes[4])
	}
}

func TestParseBGPRawTable_Empty(t *testing.T) {
	rt, err := parseBGPRawTable("")
	if err != nil {
		t.Fatal(err)
	}
	if len(rt.Routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(rt.Routes))
	}
}

func TestParseBGPRawTable_WhitespaceFallbackAndJunk(t *testing.T) {
	// Lines with single space separator fall back to strings.Fields; lines that
	// don't yield exactly two fields are skipped.
	data := "1.0.0.0/24 13335\njunk line with too many fields\n1.1.1.0/24\t13335\n"
	rt, err := parseBGPRawTable(data)
	if err != nil {
		t.Fatalf("parseBGPRawTable() error: %v", err)
	}
	if len(rt.Routes) != 2 {
		t.Fatalf("routes = %d, want 2 (junk skipped)", len(rt.Routes))
	}
	if rt.Routes[0].ASN != "13335" {
		t.Errorf("first asn = %q", rt.Routes[0].ASN)
	}
}

func TestParseBGPSummary_NoColonSkipped(t *testing.T) {
	s := parseBGPSummary("plain line without colon\nkey: value\n")
	var found bool
	for _, e := range s.Entries {
		if e.Key == "key" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'key' entry, got %+v", s.Entries)
	}
	if len(s.Entries) != 1 {
		t.Errorf("expected 1 entry (colonless line skipped), got %d", len(s.Entries))
	}
}

func TestFetchBGPSummary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "data-summary") {
			w.Write([]byte(sampleBGPSummary))
			return
		}
		w.Write([]byte(sampleBGPRawTable))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL), transport.WithJitter(0, 0))

	s, err := FetchBGPSummary(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchBGPSummary() error: %v", err)
	}
	if len(s.Entries) == 0 {
		t.Error("expected non-empty entries")
	}
}

func TestFetchBGPRawTable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleBGPRawTable))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL), transport.WithJitter(0, 0))

	rt, err := FetchBGPRawTable(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchBGPRawTable() error: %v", err)
	}
	if len(rt.Routes) != 5 {
		t.Errorf("routes = %d, want 5", len(rt.Routes))
	}
}

func TestFetchBGPASNMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleBGPRawTable))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL), transport.WithJitter(0, 0))

	m, err := FetchBGPASNMap(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchBGPASNMap() error: %v", err)
	}
	// ASN 13335 originates 1.0.0.0/24 and 1.1.1.0/24.
	prefixes, ok := m.ASNs["13335"]
	if !ok {
		t.Fatal("expected ASN 13335 in map")
	}
	if len(prefixes) != 2 {
		t.Errorf("13335 prefixes = %v, want 2", prefixes)
	}
	// ASN 38803 originates three prefixes.
	if len(m.ASNs["38803"]) != 3 {
		t.Errorf("38803 prefixes = %v, want 3", m.ASNs["38803"])
	}
}

func TestFetchBGPHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL), transport.WithJitter(0, 0))
	if _, err := FetchBGPSummary(context.Background(), client); err == nil {
		t.Error("expected error on HTTP 500 for summary")
	}
	if _, err := FetchBGPRawTable(context.Background(), client); err == nil {
		t.Error("expected error on HTTP 500 for raw table")
	}
}

// TestParseBGPSummary_EmptyKey covers the key=="" skip branch (a line whose
// text before the first colon is empty, e.g. "  : value").
func TestParseBGPSummary_EmptyKey(t *testing.T) {
	s := parseBGPSummary("  : value\nreal: data\n")
	if len(s.Entries) != 1 {
		t.Fatalf("got %d entries, want 1 (empty key skipped)", len(s.Entries))
	}
	if s.Entries[0].Key != "real" {
		t.Errorf("entry key = %q, want %q", s.Entries[0].Key, "real")
	}
}

// TestParseBGPRawTable_BlankLineAndScannerErr covers the blank-line skip branch
// and the scanner.Err error path (a single line longer than the 4MB buffer).
func TestParseBGPRawTable_BlankLineAndScannerErr(t *testing.T) {
	// Blank line among valid rows.
	rt, err := parseBGPRawTable("1.2.3.0/24\t13335\n   \n10.0.0.0/8\t1239\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rt.Routes) != 2 {
		t.Errorf("got %d routes, want 2 (blank line skipped)", len(rt.Routes))
	}

	// Scanner error: a line exceeding the 4MB buffer limit.
	huge := strings.Repeat("x", 5*1024*1024)
	_, err = parseBGPRawTable(huge)
	if err == nil {
		t.Error("expected scanner error on >4MB line")
	}
}

// TestFetchBGPRawTable_ParseError covers FetchBGPRawTable's parse-error branch
// by serving a body whose single line exceeds 4MB.
func TestFetchBGPRawTable_ParseError(t *testing.T) {
	huge := strings.Repeat("x", 5*1024*1024) + "\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(huge))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL),
		transport.WithJitter(0, 0), transport.WithCacheTTL(0), transport.WithMaxConcurrentDownloads(0))
	if _, err := FetchBGPRawTable(context.Background(), client); err == nil {
		t.Error("expected parse error from oversized line")
	}
}

// TestFetchBGPASNMap_RawTableError covers FetchBGPASNMap's error branch when the
// underlying FetchBGPRawTable fails.
func TestFetchBGPASNMap_RawTableError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL),
		transport.WithJitter(0, 0), transport.WithCacheTTL(0), transport.WithMaxConcurrentDownloads(0))
	if _, err := FetchBGPASNMap(context.Background(), client); err == nil {
		t.Error("expected error from FetchBGPASNMap when raw table fails")
	}
}

const sampleBGPBadPrefixes = `Prefixes longer than /24 and their Origin AS (Global)
-----------------------------------------------------

Origin AS       Address
    10167       1.209.111.128/25
    12345       2.2.2.0/26
`

func TestParseBGPBadPrefixes(t *testing.T) {
	r := parseBGPBadPrefixes(sampleBGPBadPrefixes)
	if len(r.Prefixes) != 2 {
		t.Fatalf("prefixes = %d, want 2", len(r.Prefixes))
	}
	if r.Prefixes[0].OriginAS != "10167" || r.Prefixes[0].Address != "1.209.111.128/25" {
		t.Errorf("prefix[0] = %+v", r.Prefixes[0])
	}
	// Header lines must be skipped.
	for _, p := range r.Prefixes {
		if p.OriginAS == "Origin" {
			t.Error("column header was not skipped")
		}
	}
}

func TestParseBGPBadPrefixes_Empty(t *testing.T) {
	r := parseBGPBadPrefixes("")
	if len(r.Prefixes) != 0 {
		t.Errorf("expected 0 prefixes, got %d", len(r.Prefixes))
	}
}

// TestParseBGPBadPrefixes_HeaderSkip exercises the column-header skip branch
// (a two-field line whose first field is "Origin") and malformed lines.
func TestParseBGPBadPrefixes_HeaderSkip(t *testing.T) {
	// "Origin\tAddress" is exactly 2 fields and triggers the EqualFold skip.
	const in = "Origin\tAddress\n" +
		"onlyoneword\n" + // len(fields) != 2 -> skipped
		"1.2.3.0/24\n" + // 1 field -> skipped
		"999\t4.5.6.0/25\n" // valid
	r := parseBGPBadPrefixes(in)
	if len(r.Prefixes) != 1 {
		t.Fatalf("prefixes = %d, want 1", len(r.Prefixes))
	}
	if r.Prefixes[0].OriginAS != "999" {
		t.Errorf("unexpected prefix: %+v", r.Prefixes[0])
	}
}

const sampleBGPPerPrefixLength = `Number of prefixes announced per prefix length (Global)
-------------------------------------------------------

 /1:0        /2:0        /3:0        /4:0        /8:16       /9:14
 /10:39      /11:97      /12:299
`

func TestParseBGPPerPrefixLength(t *testing.T) {
	r := parseBGPPerPrefixLength(sampleBGPPerPrefixLength)
	if len(r.Counts) != 9 {
		t.Fatalf("counts = %d, want 9", len(r.Counts))
	}
	// Spot-check /8:16 and /12:299.
	var found8, found12 bool
	for _, c := range r.Counts {
		if c.Length == 8 && c.Count == 16 {
			found8 = true
		}
		if c.Length == 12 && c.Count == 299 {
			found12 = true
		}
	}
	if !found8 || !found12 {
		t.Errorf("missing expected entries: %+v", r.Counts)
	}
}

func TestParseBGPPerPrefixLength_Empty(t *testing.T) {
	r := parseBGPPerPrefixLength("")
	if len(r.Counts) != 0 {
		t.Errorf("expected 0 counts, got %d", len(r.Counts))
	}
}

// TestParseBGPPerPrefixLength_Malformed exercises the skip branches: non-/
// tokens, /-tokens without colon, and non-numeric length/count.
func TestParseBGPPerPrefixLength_Malformed(t *testing.T) {
	const in = " /8:16   notok   /8   /x:5   /8:y\n"
	r := parseBGPPerPrefixLength(in)
	if len(r.Counts) != 1 {
		t.Fatalf("counts = %d, want 1", len(r.Counts))
	}
	if r.Counts[0].Length != 8 || r.Counts[0].Count != 16 {
		t.Errorf("unexpected count: %+v", r.Counts[0])
	}
}

const sampleBGPUsedAutnums = `     1 LVLT-1 - Level 3 Parent, LLC, US
     2 UDEL-DCN - University of Delaware, US
   13335 CLOUDFLARENET - Cloudflare, Inc., US
`

func TestParseBGPUsedAutnums(t *testing.T) {
	r := parseBGPUsedAutnums(sampleBGPUsedAutnums)
	if len(r.Autnums) != 3 {
		t.Fatalf("autnums = %d, want 3", len(r.Autnums))
	}
	a := r.Autnums[0]
	if a.ASN != "1" || a.Name != "LVLT-1" || a.Country != "US" {
		t.Errorf("autnum[0] = %+v", a)
	}
	if a.FullName != "LVLT-1 - Level 3 Parent, LLC" {
		t.Errorf("FullName = %q", a.FullName)
	}
	cf := r.Autnums[2]
	if cf.ASN != "13335" || cf.Country != "US" {
		t.Errorf("cloudflare autnum = %+v", cf)
	}
}

func TestParseBGPUsedAutnums_Empty(t *testing.T) {
	r := parseBGPUsedAutnums("")
	if len(r.Autnums) != 0 {
		t.Errorf("expected 0 autnums, got %d", len(r.Autnums))
	}
}

// TestParseBGPUsedAutnums_Malformed exercises the skip branches: blank line,
// single-word line, and a line without a comma.
func TestParseBGPUsedAutnums_Malformed(t *testing.T) {
	const in = "\n" + // blank line
		"justoneword\n" + // < 2 fields
		"1 LVLT-1 no comma here\n" + // no comma
		"2 UDEL-DCN - University of Delaware, US\n" // valid
	r := parseBGPUsedAutnums(in)
	if len(r.Autnums) != 1 {
		t.Fatalf("autnums = %d, want 1", len(r.Autnums))
	}
	if r.Autnums[0].ASN != "2" || r.Autnums[0].Country != "US" {
		t.Errorf("unexpected: %+v", r.Autnums[0])
	}
}

// TestParseBGPUsedAutnums_CommaBeforeASN exercises the defensive slice-bound
// guard: when a comma appears at or before the ASN token (e.g. "1, Foo Bar" or
// ", foo"), len(asn) > commaIdx must not panic and the line is skipped.
func TestParseBGPUsedAutnums_CommaBeforeASN(t *testing.T) {
	const in = "1, Foo Bar\n" + // comma right after ASN token boundary
		", foo\n" + // comma before any ASN field
		"3 UDEL-DCN - University of Delaware, US\n" // valid line still parsed
	r := parseBGPUsedAutnums(in)
	if len(r.Autnums) != 1 {
		t.Fatalf("autnums = %d, want 1 (only the valid line)", len(r.Autnums))
	}
	if r.Autnums[0].ASN != "3" || r.Autnums[0].Country != "US" {
		t.Errorf("unexpected: %+v", r.Autnums[0])
	}
}

const sampleBGPSparPrefixes = `Prefixes from the Special Purpose Address Registry (Global)
-----------------------------------------------------------

Prefix                Origin AS  Description
192.88.99.0/24             6939  HURRICANE - Hurricane Electric LLC, US
`

func TestParseBGPSparPrefixes(t *testing.T) {
	r := parseBGPSparPrefixes(sampleBGPSparPrefixes)
	if len(r.Prefixes) != 1 {
		t.Fatalf("prefixes = %d, want 1", len(r.Prefixes))
	}
	p := r.Prefixes[0]
	if p.Prefix != "192.88.99.0/24" || p.OriginAS != "6939" {
		t.Errorf("prefix = %+v", p)
	}
	if p.Description != "HURRICANE - Hurricane Electric LLC, US" {
		t.Errorf("Description = %q", p.Description)
	}
}

func TestParseBGPSparPrefixes_Empty(t *testing.T) {
	r := parseBGPSparPrefixes("")
	if len(r.Prefixes) != 0 {
		t.Errorf("expected 0 prefixes, got %d", len(r.Prefixes))
	}
}

// TestParseBGPSparPrefixes_Malformed exercises the single-field skip branch and
// the column-header skip, plus a 2-field line (empty description via tab-split).
func TestParseBGPSparPrefixes_Malformed(t *testing.T) {
	const in = "onlyoneword\n" + // Split -> 1 field, Fields -> 1 field -> skipped
		"Prefix\tOriginAS\tDescription\n" + // column header -> skipped
		"10.0.0.0/8\t1239\n" + // 2 fields, empty desc
		"192.0.2.0/24\t6939\tHURRICANE\n" // valid, 3 fields
	r := parseBGPSparPrefixes(in)
	if len(r.Prefixes) != 2 {
		t.Fatalf("prefixes = %d, want 2", len(r.Prefixes))
	}
	if r.Prefixes[0].Description != "" {
		t.Errorf("expected empty desc, got %q", r.Prefixes[0].Description)
	}
	if r.Prefixes[1].Description != "HURRICANE" {
		t.Errorf("expected HURRICANE, got %q", r.Prefixes[1].Description)
	}
}

const sampleBGPSinglePfx = `Number of ASNs announcing less than 20 prefixes
-----------------------------------------------

No. of Prefixes  No. of ASNs  RIR
       1              27539   Global
       2              10000    Global
       3                500   APNIC
`

func TestParseBGPSinglePfx(t *testing.T) {
	r := parseBGPSinglePfx(sampleBGPSinglePfx)
	if len(r.Counts) != 3 {
		t.Fatalf("counts = %d, want 3", len(r.Counts))
	}
	c := r.Counts[0]
	if c.PrefixCount != 1 || c.ASNCount != 27539 || c.RIR != "Global" {
		t.Errorf("count[0] = %+v", c)
	}
}

func TestParseBGPSinglePfx_Empty(t *testing.T) {
	r := parseBGPSinglePfx("")
	if len(r.Counts) != 0 {
		t.Errorf("expected 0 counts, got %d", len(r.Counts))
	}
}

// TestParseBGPSinglePfx_Malformed exercises the <3-field skip, the column
// header skip, and the non-numeric prefix/asn count skips.
func TestParseBGPSinglePfx_Malformed(t *testing.T) {
	const in = "1 2\n" + // < 3 fields
		"No. of Prefixes  No. of ASNs  RIR\n" + // column header
		"x 2 Global\n" + // prefixCount non-numeric
		"1 y Global\n" + // asnCount non-numeric
		"1 2 APNIC\n" // valid
	r := parseBGPSinglePfx(in)
	if len(r.Counts) != 1 {
		t.Fatalf("counts = %d, want 1", len(r.Counts))
	}
	if r.Counts[0].PrefixCount != 1 || r.Counts[0].ASNCount != 2 || r.Counts[0].RIR != "APNIC" {
		t.Errorf("unexpected: %+v", r.Counts[0])
	}
}

func TestFetchBGPBadPrefixes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(sampleBGPBadPrefixes))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL), transport.WithJitter(0, 0))
	r, err := FetchBGPBadPrefixes(context.Background(), client, "")
	if err != nil {
		t.Fatalf("FetchBGPBadPrefixes() error: %v", err)
	}
	if len(r.Prefixes) != 2 {
		t.Errorf("prefixes = %d, want 2", len(r.Prefixes))
	}
}

func TestFetchBGPPerPrefixLength(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleBGPPerPrefixLength))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL), transport.WithJitter(0, 0))
	r, err := FetchBGPPerPrefixLength(context.Background(), client, "au")
	if err != nil {
		t.Fatalf("FetchBGPPerPrefixLength() error: %v", err)
	}
	if len(r.Counts) != 9 {
		t.Errorf("counts = %d, want 9", len(r.Counts))
	}
	// Source selection is verified via the URL seen by the mock server below
	// in TestFetchBGP_SourceAU; here we only assert parse correctness.
}

// TestFetchBGP_SourceAU verifies that source="au" routes the request to the
// /au/ path segment on the thyme server.
func TestFetchBGP_SourceAU(t *testing.T) {
	var seenPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Write([]byte(sampleBGPPerPrefixLength))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL), transport.WithJitter(0, 0))
	if _, err := FetchBGPPerPrefixLength(context.Background(), client, "au"); err != nil {
		t.Fatalf("FetchBGPPerPrefixLength() source=au error: %v", err)
	}
	if !strings.Contains(seenPath, "/au/data-pfx-nos") {
		t.Errorf("expected /au/data-pfx-nos in path, got %q", seenPath)
	}
}

func TestFetchBGPUsedAutnums(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleBGPUsedAutnums))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL), transport.WithJitter(0, 0))
	r, err := FetchBGPUsedAutnums(context.Background(), client, "")
	if err != nil {
		t.Fatalf("FetchBGPUsedAutnums() error: %v", err)
	}
	if len(r.Autnums) != 3 {
		t.Errorf("autnums = %d, want 3", len(r.Autnums))
	}
}

func TestFetchBGPSparPrefixes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleBGPSparPrefixes))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL), transport.WithJitter(0, 0))
	r, err := FetchBGPSparPrefixes(context.Background(), client, "hk")
	if err != nil {
		t.Fatalf("FetchBGPSparPrefixes() error: %v", err)
	}
	if len(r.Prefixes) != 1 {
		t.Errorf("prefixes = %d, want 1", len(r.Prefixes))
	}
}

func TestFetchBGPSinglePfx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleBGPSinglePfx))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL), transport.WithJitter(0, 0))
	r, err := FetchBGPSinglePfx(context.Background(), client, "")
	if err != nil {
		t.Fatalf("FetchBGPSinglePfx() error: %v", err)
	}
	if len(r.Counts) != 3 {
		t.Errorf("counts = %d, want 3", len(r.Counts))
	}
}

func TestFetchBGPAdditionalFiles_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL), transport.WithJitter(0, 0))
	ctx := context.Background()
	if _, err := FetchBGPBadPrefixes(ctx, client, ""); err == nil {
		t.Error("expected error on 500 for badpfx")
	}
	if _, err := FetchBGPPerPrefixLength(ctx, client, ""); err == nil {
		t.Error("expected error on 500 for pfx-nos")
	}
	if _, err := FetchBGPUsedAutnums(ctx, client, ""); err == nil {
		t.Error("expected error on 500 for used-autnums")
	}
	if _, err := FetchBGPSparPrefixes(ctx, client, ""); err == nil {
		t.Error("expected error on 500 for spar")
	}
	if _, err := FetchBGPSinglePfx(ctx, client, ""); err == nil {
		t.Error("expected error on 500 for singlepfx")
	}
}

// TestWithThymeSource exercises the transport.WithThymeSource option by observing that a
// per-call source overrides the client default and routes to /au/.
func TestWithThymeSource(t *testing.T) {
	var seenPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Write([]byte(sampleBGPSinglePfx))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithThymeBaseURL(srv.URL), transport.WithThymeSource("au"), transport.WithJitter(0, 0))
	if _, err := FetchBGPSinglePfx(context.Background(), client, ""); err != nil {
		t.Fatalf("FetchBGPSinglePfx() error: %v", err)
	}
	if !strings.Contains(seenPath, "/au/data-singlepfx") {
		t.Errorf("expected /au/data-singlepfx, got %q", seenPath)
	}
}

// TestBuildThymeURL_EmptySource exercises the empty-source default branch in
// buildThymeURL (source defaults to "current").
func TestBuildThymeURL_EmptySource(t *testing.T) {
	got := transport.BuildThymeURL("https://thyme.apnic.net/", "", "data-summary")
	if got != "https://thyme.apnic.net/current/data-summary" {
		t.Errorf("unexpected URL: %q", got)
	}
}
