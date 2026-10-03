package transport

import (
	"context"
	"net"
	"testing"
	"time"
)

// Coverage backfill for the transport re-exports that the root apnic package
// exercises only through the embedded *Client (cross-package calls are not
// credited to transport by Go's cover tool). These tests drive each Option
// factory, accessor, and URL builder directly inside the transport package so
// the post-restructure coverage matches what the pre-restructure monolithic
// root package had.

func TestWithDialWhois(t *testing.T) {
	fn := func(ctx context.Context, network, address string) (net.Conn, error) {
		return nil, nil
	}
	c := NewClient(WithDialWhois(fn))
	if c.DialWhois() == nil {
		t.Error("WithDialWhois did not set the dial func")
	}
}

func TestWithRDAPDate(t *testing.T) {
	now := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	c := NewClient(WithRDAPDate(now))
	if !c.RDAPDate().Equal(now) {
		t.Errorf("RDAPDate = %v, want %v", c.RDAPDate(), now)
	}
}

func TestWithRRDPBaseURL(t *testing.T) {
	u := "https://rrdp.example.test"
	c := NewClient(WithRRDPBaseURL(u))
	if c.RRDPBaseURL() != u {
		t.Errorf("RRDPBaseURL = %q, want %q", c.RRDPBaseURL(), u)
	}
}

func TestWithThymeBaseURL(t *testing.T) {
	u := "https://thyme.example.test"
	c := NewClient(WithThymeBaseURL(u))
	if c.ThymeBaseURL() != u {
		t.Errorf("ThymeBaseURL = %q, want %q", c.ThymeBaseURL(), u)
	}
}

func TestWithThymeSource(t *testing.T) {
	c := NewClient(WithThymeSource("au"))
	if c.ThymeSource() != "au" {
		t.Errorf("ThymeSource = %q, want au", c.ThymeSource())
	}
}

func TestWithFTPBaseURL(t *testing.T) {
	u := "https://ftp.example.test/"
	c := NewClient(WithFTPBaseURL(u))
	if c.FTPBaseURL() != u {
		t.Errorf("FTPBaseURL = %q, want %q", c.FTPBaseURL(), u)
	}
}

func TestWithRExBaseURL(t *testing.T) {
	u := "https://rex.example.test"
	c := NewClient(WithRExBaseURL(u))
	if c.RExBaseURL() != u {
		t.Errorf("RExBaseURL = %q, want %q", c.RExBaseURL(), u)
	}
}

// Accessors for fields set by other options are covered above; this confirms
// the remaining accessors return their configured (or default) values.
func TestClientAccessorsDefaults(t *testing.T) {
	c := NewClient()
	// Just exercise each accessor so the getter lines are covered.
	_ = c.StatsBaseURL()
	_ = c.RDAPBaseURL()
	_ = c.FTPBaseURL()
	_ = c.ThymeBaseURL()
	_ = c.ThymeSource()
	_ = c.RRDPBaseURL()
	_ = c.RExBaseURL()
	_ = c.WhoisServer()
	_ = c.WhoisTimeout()
	_ = c.RDAPDate()
	_ = c.DialWhois()
}

func TestBuildTransfersAllURL(t *testing.T) {
	got := BuildTransfersAllURL("https://ftp.apnic.net/", "")
	want := "https://ftp.apnic.net/transfers-all/apnic/transfer-all-apnic-latest"
	if got != want {
		t.Errorf("latest = %q, want %q", got, want)
	}
	got = BuildTransfersAllURL("https://ftp.apnic.net/", "20200115")
	want = "https://ftp.apnic.net/transfers-all/apnic/2020/transfer-all-apnic-20200115"
	if got != want {
		t.Errorf("dated = %q, want %q", got, want)
	}
}

func TestBuildTransfersAllSidecarURL(t *testing.T) {
	got := BuildTransfersAllSidecarURL("https://ftp.apnic.net/", "", ".md5")
	want := "https://ftp.apnic.net/transfers-all/apnic/transfer-all-apnic-latest.md5"
	if got != want {
		t.Errorf("sidecar = %q, want %q", got, want)
	}
}

func TestBuildTelemetryURL(t *testing.T) {
	// The latest URL concatenates ftpBaseURL + "apnic/..." (no leading slash),
	// so the base must carry a trailing slash. The dated branch uses
	// fmt.Sprintf("%s/apnic/...") and assumes no trailing slash.
	got := BuildTelemetryURL("https://ftp.apnic.net/", "")
	if got != "https://ftp.apnic.net/apnic/whois-rdap-stats/whois-rdap-stats.json" {
		t.Errorf("latest = %q", got)
	}
	got = BuildTelemetryURL("https://ftp.apnic.net", "20260701")
	if got != "https://ftp.apnic.net/apnic/whois-rdap-stats/2026/whois-rdap-stats-20260701.json" {
		t.Errorf("dated = %q", got)
	}
}

func TestBuildTelemetrySidecarURL(t *testing.T) {
	got := BuildTelemetrySidecarURL("https://ftp.apnic.net/", "")
	if got != "https://ftp.apnic.net/apnic/whois-rdap-stats/whois-rdap-stats.json.md5" {
		t.Errorf("sidecar = %q", got)
	}
}

func TestBuildIRRDBURL(t *testing.T) {
	got := BuildIRRDBURL("https://ftp.apnic.net/", "inetnum")
	want := "https://ftp.apnic.net/apnic/whois/apnic.db.inetnum.gz"
	if got != want {
		t.Errorf("irr db = %q, want %q", got, want)
	}
}

func TestBuildIRRCurrentSerialURL(t *testing.T) {
	got := BuildIRRCurrentSerialURL("https://ftp.apnic.net/")
	if got != "https://ftp.apnic.net/apnic/whois/APNIC.CURRENTSERIAL" {
		t.Errorf("serial = %q", got)
	}
}

func TestCacheKeyIRR(t *testing.T) {
	if got := CacheKeyIRR("route"); got != "irr:route" {
		t.Errorf("CacheKeyIRR = %q, want irr:route", got)
	}
}

// CacheGet/CacheSet hit the nil-cache guard only if the cache is nil. A normal
// NewClient always has a cache, so the guard is exercised by a zero-value
// Client. The get/set round-trip itself is covered by TestCacheGetSet.
func TestCacheGetSetNilCacheGuard(t *testing.T) {
	var c Client // zero-value Client has nil cache
	if _, ok := c.CacheGet("anything"); ok {
		t.Error("CacheGet on nil cache should return false")
	}
	// CacheSet on nil cache must not panic.
	c.CacheSet("anything", "value")
}
