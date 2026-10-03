package query

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/testutil"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

const sampleRExUserNetwork = `{"ip":"219.142.144.241","prefix":"219.142.128.0/18","asn":4847,"economy":"CN"}`

const sampleRExResources = `{"items":[
{"resource":"23.160.212.0/24","type":"ipv4","opaqueId":"522be47e60b5c2ef81bbbab8deaa6b85","holderName":"ERIN AVENUE LLC","rir":"arin","nir":null,"delegationDate":"2026-06-30","transferDate":null,"cc":"US"},
{"resource":"AS402676","type":"asn","opaqueId":"522be47e60b5c2ef81bbbab8deaa6b85","holderName":"ERIN AVENUE LLC","rir":"arin","nir":null,"delegationDate":"2026-06-30","transferDate":null,"cc":"US"}
]}`

const sampleRExHolder = `{"opaqueId":"522be47e60b5c2ef81bbbab8deaa6b85","registry":"arin","nir":null,"holderName":"ERIN AVENUE LLC","asns":["AS402676"],"asnsCount":1,"ipv4":["23.160.212.0/24"],"ipv4_24Count":1.0,"ipv6":["2602:f373::/40"],"ipv6_48Count":256.0}`

const sampleRExHoldersCount = `{"count":129665}`

func TestBuildRExURL(t *testing.T) {
	cases := []struct {
		base, path string
		q          string
		want       string
	}{
		{"https://api.rex.apnic.net", "user-network", "", "https://api.rex.apnic.net/v1/user-network"},
		{"https://api.rex.apnic.net/", "/user-network/", "", "https://api.rex.apnic.net/v1/user-network"},
		{"https://api.rex.apnic.net", "holders/unique-count", "", "https://api.rex.apnic.net/v1/holders/unique-count"},
	}
	for _, tc := range cases {
		got := buildRExURL(tc.base, tc.path, nil)
		if got != tc.want {
			t.Errorf("buildRExURL(%q,%q) = %q, want %q", tc.base, tc.path, got, tc.want)
		}
	}
	// Query encoding for the holder endpoint.
	q := url.Values{}
	q.Set("opaqueId", "abc")
	q.Set("rir", "arin")
	got := buildRExURL("https://api.rex.apnic.net", "holder", q)
	if !strings.HasPrefix(got, "https://api.rex.apnic.net/v1/holder?") {
		t.Errorf("holder url missing query: %q", got)
	}
	if !strings.Contains(got, "opaqueId=abc") || !strings.Contains(got, "rir=arin") {
		t.Errorf("holder url missing params: %q", got)
	}
}

func TestFetchRExUserNetwork(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/user-network" {
			t.Errorf("path = %q, want /v1/user-network", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleRExUserNetwork))
	}))
	defer srv.Close()
	c := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRExBaseURL(srv.URL), transport.WithJitter(0, 0))

	res, err := FetchRExUserNetwork(context.Background(), c)
	if err != nil {
		t.Fatalf("FetchRExUserNetwork() error: %v", err)
	}
	if res.IP != "219.142.144.241" {
		t.Errorf("ip = %q", res.IP)
	}
	if res.Prefix != "219.142.128.0/18" {
		t.Errorf("prefix = %q", res.Prefix)
	}
	if res.ASN != 4847 {
		t.Errorf("asn = %d, want 4847", res.ASN)
	}
	if res.Economy != "CN" {
		t.Errorf("economy = %q", res.Economy)
	}
}

func TestFetchRExResources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/resources" {
			t.Errorf("path = %q, want /v1/resources", r.URL.Path)
		}
		if r.URL.Query().Get("type") != "asn" {
			t.Errorf("type param = %q, want asn", r.URL.Query().Get("type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleRExResources))
	}))
	defer srv.Close()
	c := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRExBaseURL(srv.URL), transport.WithJitter(0, 0))

	res, err := FetchRExResources(context.Background(), c, "asn")
	if err != nil {
		t.Fatalf("FetchRExResources() error: %v", err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(res.Items))
	}
	first := res.Items[0]
	if first.Resource != "23.160.212.0/24" || first.Type != "ipv4" {
		t.Errorf("first item = %+v", first)
	}
	if first.OpaqueID != "522be47e60b5c2ef81bbbab8deaa6b85" {
		t.Errorf("opaqueId = %q", first.OpaqueID)
	}
	if first.HolderName != "ERIN AVENUE LLC" || first.RIR != "arin" || first.CC != "US" {
		t.Errorf("holder fields = %+v", first)
	}
	if first.DelegationDate != "2026-06-30" {
		t.Errorf("delegationDate = %q", first.DelegationDate)
	}
	// transferDate is null in JSON; decoded into a string it stays empty.
	if first.TransferDate != "" {
		t.Errorf("transferDate = %q, want empty for null", first.TransferDate)
	}
}

func TestFetchRExResources_NoFilter(t *testing.T) {
	// Empty type must not append a query string.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query, got %q", r.URL.RawQuery)
		}
		w.Write([]byte(sampleRExResources))
	}))
	defer srv.Close()
	c := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRExBaseURL(srv.URL), transport.WithJitter(0, 0))

	if _, err := FetchRExResources(context.Background(), c, ""); err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestFetchRExHolder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/holder" {
			t.Errorf("path = %q, want /v1/holder", r.URL.Path)
		}
		if r.URL.Query().Get("opaqueId") != "522be47e60b5c2ef81bbbab8deaa6b85" {
			t.Errorf("opaqueId param = %q", r.URL.Query().Get("opaqueId"))
		}
		if r.URL.Query().Get("rir") != "arin" {
			t.Errorf("rir param = %q", r.URL.Query().Get("rir"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleRExHolder))
	}))
	defer srv.Close()
	c := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRExBaseURL(srv.URL), transport.WithJitter(0, 0))

	res, err := FetchRExHolder(context.Background(), c, "522be47e60b5c2ef81bbbab8deaa6b85", "arin")
	if err != nil {
		t.Fatalf("FetchRExHolder() error: %v", err)
	}
	if res.HolderName != "ERIN AVENUE LLC" {
		t.Errorf("holderName = %q", res.HolderName)
	}
	if res.Registry != "arin" {
		t.Errorf("registry = %q", res.Registry)
	}
	if len(res.ASNs) != 1 || res.ASNs[0] != "AS402676" {
		t.Errorf("asns = %+v", res.ASNs)
	}
	if res.ASNsCount != 1 {
		t.Errorf("asnsCount = %d", res.ASNsCount)
	}
	if len(res.IPv4) != 1 || res.IPv4[0] != "23.160.212.0/24" {
		t.Errorf("ipv4 = %+v", res.IPv4)
	}
	if res.IPv4_24Count != 1.0 {
		t.Errorf("ipv4_24Count = %v", res.IPv4_24Count)
	}
	if len(res.IPv6) != 1 || res.IPv6[0] != "2602:f373::/40" {
		t.Errorf("ipv6 = %+v", res.IPv6)
	}
	if res.IPv6_48Count != 256.0 {
		t.Errorf("ipv6_48Count = %v", res.IPv6_48Count)
	}
}

func TestFetchRExHolder_MissingParams(t *testing.T) {
	c := transport.NewClient(transport.WithJitter(0, 0))
	_, err := FetchRExHolder(context.Background(), c, "", "arin")
	if !errors.Is(err, transport.ErrInvalidRExParam) {
		t.Errorf("err = %v, want transport.ErrInvalidRExParam", err)
	}
	_, err = FetchRExHolder(context.Background(), c, "abc", "")
	if !errors.Is(err, transport.ErrInvalidRExParam) {
		t.Errorf("err = %v, want transport.ErrInvalidRExParam", err)
	}
}

// TestFetchRExHolder_ServerParamError reproduces REx's real behaviour: when a
// required query parameter is missing the server answers 400/404 with a short
// plain-text message rather than JSON. fetchJSON must surface that as an error
// carrying the message, not a JSON decode failure.
func TestFetchRExHolder_ServerParamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Either resource or opaque ID and RIR are required as query parameters."))
	}))
	defer srv.Close()
	c := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRExBaseURL(srv.URL), transport.WithJitter(0, 0))

	_, err := FetchRExHolder(context.Background(), c, "abc", "arin")
	if err == nil {
		t.Fatal("expected error for server param error")
	}
	if !strings.Contains(err.Error(), "unexpected status code: 400") {
		t.Errorf("err = %v, want status 400 in message", err)
	}
	if !strings.Contains(err.Error(), "opaque ID and RIR") {
		t.Errorf("err = %v, want server message in body", err)
	}
	if !isRExAPIError(err) {
		t.Errorf("isRExAPIError should be true for server error")
	}
}

func TestFetchRExHoldersUniqueCount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/holders/unique-count" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleRExHoldersCount))
	}))
	defer srv.Close()
	c := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRExBaseURL(srv.URL), transport.WithJitter(0, 0))

	res, err := FetchRExHoldersUniqueCount(context.Background(), c)
	if err != nil {
		t.Fatalf("FetchRExHoldersUniqueCount() error: %v", err)
	}
	if res.Count != 129665 {
		t.Errorf("count = %d, want 129665", res.Count)
	}
}

func TestFetchRExHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRExBaseURL(srv.URL), transport.WithJitter(0, 0))

	if _, err := FetchRExUserNetwork(context.Background(), c); err == nil {
		t.Error("expected error on HTTP 500 for user-network")
	}
	if _, err := FetchRExResources(context.Background(), c, ""); err == nil {
		t.Error("expected error on HTTP 500 for resources")
	}
	if _, err := FetchRExHolder(context.Background(), c, "abc", "arin"); err == nil {
		t.Error("expected error on HTTP 500 for holder")
	}
	if _, err := FetchRExHoldersUniqueCount(context.Background(), c); err == nil {
		t.Error("expected error on HTTP 500 for unique-count")
	}
}

func TestFetchRExInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()
	c := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRExBaseURL(srv.URL), transport.WithJitter(0, 0))

	if _, err := FetchRExUserNetwork(context.Background(), c); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// TestFetchREx_StealthHeadersInjected verifies REx requests go through the
// unified doHTTPRequest exit and therefore carry browser-mimicry headers when
// stealth is enabled (the anti-scraping guarantee for the new endpoint).
func TestFetchREx_StealthHeadersInjected(t *testing.T) {
	srv, hdr := testutil.CapturingHandler(t, http.StatusOK, sampleRExUserNetwork)
	defer srv.Close()
	c := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRExBaseURL(srv.URL), transport.WithJitter(0, 0))

	if _, err := FetchRExUserNetwork(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if got := hdr.Get("User-Agent"); !strings.Contains(got, "Mozilla") {
		t.Errorf("stealth UA should look like a browser, got %q", got)
	}
	for _, want := range []string{"Accept-Language", "Accept-Encoding", "Sec-Fetch-Site", "Sec-Ch-Ua"} {
		if hdr.Get(want) == "" {
			t.Errorf("stealth should set %s header for REx", want)
		}
	}
}

// TestFetchREx_GzipDecompressedOnce ensures REx JSON served with
// Content-Encoding: gzip (common for large /v1/resources responses) is
// decompressed exactly once by fetchJSON before decoding — mirroring the RRDP
// regression. Without stealth's Accept-Encoding:gzip the server would not
// compress, so this also confirms the gzip path is exercised.
func TestFetchREx_GzipDecompressedOnce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(testutil.GzipBytes(sampleRExHoldersCount))
	}))
	defer srv.Close()
	c := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRExBaseURL(srv.URL), transport.WithJitter(0, 0), transport.WithStealth(true))

	res, err := FetchRExHoldersUniqueCount(context.Background(), c)
	if err != nil {
		t.Fatalf("gzip decode error: %v", err)
	}
	if res.Count != 129665 {
		t.Errorf("count = %d, want 129665", res.Count)
	}
}

func TestIsRExAPIError(t *testing.T) {
	if isRExAPIError(nil) {
		t.Error("nil should not be a REx API error")
	}
	if isRExAPIError(errors.New("transport: connection reset")) {
		t.Error("plain transport error should not be a REx API error")
	}
	if !isRExAPIError(transport.ErrInvalidRExParam) {
		t.Error("transport.ErrInvalidRExParam should be a REx API error")
	}
}

// TestFetchREx_RequestError covers fetchJSON's doHTTPRequest-error branch via
// a transport that always fails.
func TestFetchREx_RequestError(t *testing.T) {
	c := transport.NewClient(
		transport.WithHTTPClient(&http.Client{Transport: testutil.DialErrRoundTripper{}}),
		transport.WithRExBaseURL("http://x"), transport.WithJitter(0, 0), transport.WithCacheTTL(0))
	if _, err := FetchRExUserNetwork(context.Background(), c); err == nil {
		t.Error("expected request error for user-network")
	}
}

// TestFetchREx_GzipInitError covers fetchJSON's gzip.NewReader-error branch:
// a 200 with Content-Encoding: gzip but a body that is not valid gzip.
func TestFetchREx_GzipInitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-gzip"))
	}))
	defer srv.Close()
	c := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRExBaseURL(srv.URL),
		transport.WithJitter(0, 0), transport.WithCacheTTL(0))
	if _, err := FetchRExUserNetwork(context.Background(), c); err == nil {
		t.Error("expected gzip init error")
	}
}
