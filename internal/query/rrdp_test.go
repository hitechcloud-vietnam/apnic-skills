package query

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/testutil"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

const sampleRRDPNotification = `<notification xmlns="http://www.ripe.net/rpki/rrdp" version="1" session_id="8dad0cc8-0bc8-4021-88ed-e75e295df946" serial="65148">
  <snapshot uri="https://rrdp.apnic.net/8dad0cc8/snapshot.xml" hash="479c1351cc5372febc3487abe80bad01ea04118a78f59100004c213f944022d9"/>
  <delta serial="65148" uri="https://rrdp.apnic.net/8dad0cc8/delta-65148.xml" hash="45ff4de1ac87c9b41009b5e71d7ff175adb01ce69af26bfbe5b7093a027cc0c5"/>
  <delta serial="65147" uri="https://rrdp.apnic.net/8dad0cc8/delta-65147.xml" hash="7b2822ef60ec93726e0d3af6f05341b517e8fd9f4544dce94b31196cf5ca7142"/>
</notification>`

const sampleRRDPSnapshot = `<snapshot version="1" session_id="8dad0cc8" serial="65148" xmlns="http://www.ripe.net/rpki/rrdp">
<publish uri="rsync://rpki.apnic.net/rep/A9110009/roa1.roa">AAAABASE64BODY1</publish>
<publish uri="rsync://rpki.apnic.net/rep/A9110009/roa2.roa">AAAABASE64BODY2</publish>
<withdraw uri="rsync://rpki.apnic.net/rep/A9110009/old.roa"/>
</snapshot>`

func TestParseRRDPNotification(t *testing.T) {
	n, err := parseRRDPNotification(strings.NewReader(sampleRRDPNotification))
	if err != nil {
		t.Fatalf("parseRRDPNotification() error: %v", err)
	}
	if n.Version != "1" {
		t.Errorf("version = %q", n.Version)
	}
	if n.SessionID != "8dad0cc8-0bc8-4021-88ed-e75e295df946" {
		t.Errorf("session_id = %q", n.SessionID)
	}
	if n.Serial != 65148 {
		t.Errorf("serial = %d, want 65148", n.Serial)
	}
	if n.Snapshot.URI != "https://rrdp.apnic.net/8dad0cc8/snapshot.xml" {
		t.Errorf("snapshot uri = %q", n.Snapshot.URI)
	}
	if n.Snapshot.Hash == "" {
		t.Error("expected non-empty snapshot hash")
	}
	if len(n.Deltas) != 2 {
		t.Fatalf("deltas = %d, want 2", len(n.Deltas))
	}
	if n.Deltas[0].Serial != 65148 || n.Deltas[1].Serial != 65147 {
		t.Errorf("delta serials = %d, %d", n.Deltas[0].Serial, n.Deltas[1].Serial)
	}
}

func TestParseRRDPNotification_BadXML(t *testing.T) {
	_, err := parseRRDPNotification(strings.NewReader("not xml"))
	if err == nil {
		t.Error("expected error for bad XML")
	}
}

func TestParseRRDPSnapshot(t *testing.T) {
	s, err := parseRPKISnapshot(strings.NewReader(sampleRRDPSnapshot))
	if err != nil {
		t.Fatalf("parseRPKISnapshot() error: %v", err)
	}
	if s.SessionID != "8dad0cc8" {
		t.Errorf("session_id = %q", s.SessionID)
	}
	if s.Serial != 65148 {
		t.Errorf("serial = %d, want 65148", s.Serial)
	}
	if len(s.Published) != 2 {
		t.Fatalf("published = %d, want 2", len(s.Published))
	}
	if s.Published[0] != "rsync://rpki.apnic.net/rep/A9110009/roa1.roa" {
		t.Errorf("published[0] = %q", s.Published[0])
	}
	if len(s.Withdrawn) != 1 {
		t.Fatalf("withdrawn = %d, want 1", len(s.Withdrawn))
	}
	if s.Withdrawn[0] != "rsync://rpki.apnic.net/rep/A9110009/old.roa" {
		t.Errorf("withdrawn[0] = %q", s.Withdrawn[0])
	}
}

func TestParseRRDPSnapshot_Empty(t *testing.T) {
	s, err := parseRPKISnapshot(strings.NewReader(`<snapshot xmlns="http://www.ripe.net/rpki/rrdp" version="1" session_id="x" serial="1"></snapshot>`))
	if err != nil {
		t.Fatalf("parseRPKISnapshot() error: %v", err)
	}
	if len(s.Published) != 0 || len(s.Withdrawn) != 0 {
		t.Errorf("expected empty snapshot, got published=%d withdrawn=%d", len(s.Published), len(s.Withdrawn))
	}
}

func TestParseRRDPNotification_NonIntSerial(t *testing.T) {
	// Serials that fail to parse are left at zero rather than erroring.
	data := `<notification xmlns="http://www.ripe.net/rpki/rrdp" version="1" session_id="s" serial="not-a-number">
  <snapshot uri="https://x/s.xml" hash="h"/>
  <delta serial="also-bad" uri="https://x/d.xml" hash="h"/>
</notification>`
	n, err := parseRRDPNotification(strings.NewReader(data))
	if err != nil {
		t.Fatalf("parseRRDPNotification() error: %v", err)
	}
	if n.Serial != 0 {
		t.Errorf("serial = %d, want 0 for non-int", n.Serial)
	}
	if len(n.Deltas) != 1 || n.Deltas[0].Serial != 0 {
		t.Errorf("delta = %+v", n.Deltas)
	}
}

func TestParseRPKISnapshot_Malformed(t *testing.T) {
	// Truncated/malformed XML: an unknown element is opened but never closed,
	// leaving non-zero depth at EOF with no publish/withdraw collected.
	_, err := parseRPKISnapshot(strings.NewReader(`<snapshot xmlns="http://www.ripe.net/rpki/rrdp" version="1"><weird>`))
	if err == nil {
		t.Error("expected error for malformed/truncated snapshot")
	}
}

func TestFetchRRDPNotification(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(sampleRRDPNotification))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRRDPBaseURL(srv.URL), transport.WithJitter(0, 0))

	n, err := FetchRRDPNotification(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchRRDPNotification() error: %v", err)
	}
	if n.Serial != 65148 {
		t.Errorf("serial = %d", n.Serial)
	}
}

func TestFetchRRDPSnapshot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(sampleRRDPSnapshot))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRRDPBaseURL(srv.URL), transport.WithJitter(0, 0))

	s, err := FetchRRDPSnapshot(context.Background(), client, srv.URL+"/snapshot.xml")
	if err != nil {
		t.Fatalf("FetchRRDPSnapshot() error: %v", err)
	}
	if len(s.Published) != 2 {
		t.Errorf("published = %d, want 2", len(s.Published))
	}
	if len(s.Withdrawn) != 1 {
		t.Errorf("withdrawn = %d, want 1", len(s.Withdrawn))
	}
}

func TestFetchRRDPDelta(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleRRDPSnapshot))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRRDPBaseURL(srv.URL), transport.WithJitter(0, 0))

	d, err := FetchRRDPDelta(context.Background(), client, srv.URL+"/delta.xml")
	if err != nil {
		t.Fatalf("FetchRRDPDelta() error: %v", err)
	}
	if d.Serial != 65148 {
		t.Errorf("serial = %d", d.Serial)
	}
}

func TestFetchRRDPHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRRDPBaseURL(srv.URL), transport.WithJitter(0, 0))
	if _, err := FetchRRDPNotification(context.Background(), client); err == nil {
		t.Error("expected error on HTTP 500 for notification")
	}
	if _, err := FetchRRDPSnapshot(context.Background(), client, srv.URL+"/snapshot.xml"); err == nil {
		t.Error("expected error on HTTP 500 for snapshot")
	}
}

// TestFetchRRDPNotification_GzipDecompressedOnce is a regression test: when
// stealth is enabled the client advertises Accept-Encoding: gzip, and the RRDP
// server responds with Content-Encoding: gzip. The notification must be
// decompressed exactly once (by fetchText) before XML decoding, otherwise the
// decoder sees raw gzip bytes and fails on illegal control characters.
func TestFetchRRDPNotification_GzipDecompressedOnce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(testutil.GzipBytes(sampleRRDPNotification))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRRDPBaseURL(srv.URL), transport.WithJitter(0, 0), transport.WithStealth(true))

	n, err := FetchRRDPNotification(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchRRDPNotification() with gzip error: %v", err)
	}
	if n.Serial != 65148 {
		t.Errorf("serial = %d, want 65148", n.Serial)
	}
	if !strings.HasPrefix(n.SessionID, "8dad0cc8") {
		t.Errorf("session = %q", n.SessionID)
	}
}

// TestFetchRRDPSnapshot_GzipDecompressedOnce mirrors the notification test for
// the streaming snapshot path: a gzip Content-Encoding must be decompressed by
// the snapshot fetcher before XML token streaming.
func TestFetchRRDPSnapshot_GzipDecompressedOnce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(testutil.GzipBytes(sampleRRDPSnapshot))
	}))
	defer srv.Close()
	client := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRRDPBaseURL(srv.URL), transport.WithJitter(0, 0), transport.WithStealth(true))

	s, err := FetchRRDPSnapshot(context.Background(), client, srv.URL+"/snapshot.xml")
	if err != nil {
		t.Fatalf("FetchRRDPSnapshot() with gzip error: %v", err)
	}
	if len(s.Published) != 2 {
		t.Errorf("published = %d, want 2", len(s.Published))
	}
	if len(s.Withdrawn) != 1 {
		t.Errorf("withdrawn = %d, want 1", len(s.Withdrawn))
	}
}

// TestParseRRDPNotification_SnapshotSerial covers the snapshot.Serial parse
// branch (line 90-92) when the snapshot element carries a numeric serial.
func TestParseRRDPNotification_SnapshotSerial(t *testing.T) {
	data := `<notification xmlns="http://www.ripe.net/rpki/rrdp" version="1" session_id="s" serial="10">
  <snapshot uri="https://x/s.xml" hash="h" serial="42"/>
</notification>`
	n, err := parseRRDPNotification(strings.NewReader(data))
	if err != nil {
		t.Fatalf("parseRRDPNotification() error: %v", err)
	}
	if n.Snapshot.Serial != 42 {
		t.Errorf("snapshot serial = %d, want 42", n.Snapshot.Serial)
	}
}

// TestFetchRRDPSnapshot_RequestError covers the doHTTPRequest-error branch.
func TestFetchRRDPSnapshot_RequestError(t *testing.T) {
	c := transport.NewClient(
		transport.WithHTTPClient(&http.Client{Transport: testutil.DialErrRoundTripper{}}),
		transport.WithRRDPBaseURL("http://x"), transport.WithJitter(0, 0), transport.WithCacheTTL(0))
	if _, err := FetchRRDPSnapshot(context.Background(), c, "http://x/s.xml"); err == nil {
		t.Error("expected request error for RRDP snapshot")
	}
}

// TestFetchRRDPSnapshot_GzipInitError covers the gzip.NewReader-error branch:
// 200 + Content-Encoding: gzip but non-gzip body.
func TestFetchRRDPSnapshot_GzipInitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-gzip"))
	}))
	defer srv.Close()
	c := transport.NewClient(transport.WithHTTPClient(srv.Client()), transport.WithRRDPBaseURL(srv.URL),
		transport.WithJitter(0, 0), transport.WithCacheTTL(0))
	if _, err := FetchRRDPSnapshot(context.Background(), c, srv.URL+"/s.xml"); err == nil {
		t.Error("expected gzip init error for RRDP snapshot")
	}
}

// TestLocalName_AttrLocal_Boundaries covers the empty-Local and no-match
// branches of the rrdp XML helpers.
func TestLocalName_AttrLocal_Boundaries(t *testing.T) {
	// localName with empty Local returns Space.
	if got := localName(xmlName("", "ns-only")); got != "ns-only" {
		t.Errorf("localName empty-Local = %q, want %q", got, "ns-only")
	}
	// localName with non-empty Local returns Local.
	if got := localName(xmlName("local", "ns")); got != "local" {
		t.Errorf("localName = %q, want %q", got, "local")
	}
	// attrLocal with no matching attribute returns "".
	if got := attrLocal([]xml.Attr{{Name: xmlName("other", ""), Value: "v"}}, "missing"); got != "" {
		t.Errorf("attrLocal no-match = %q, want %q", got, "")
	}
}

// xmlName is a tiny helper to build xml.Name in tests.
func xmlName(local, space string) xml.Name { return xml.Name{Local: local, Space: space} }
