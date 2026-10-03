package apnic

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/filter"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/history"
)

// This file exercises the With* Option factories and the remaining free
// re-export helpers in reexport.go that are not otherwise covered by the
// root wrapper-method tests or cache_test.go. Each Option is invoked through
// NewClient and, where a public accessor exists, the configured value is read
// back to confirm the option actually wired through to the embedded
// *transport.Client. Options backed by private fields (stealth, browser-UA,
// rate-limit, chunk-size, download-timeout, user-agent) have no accessor, so
// the smoke test asserts only that constructing a client with them does not
// panic.

func TestRootWithHTTPClient(t *testing.T) {
	hc := &http.Client{Timeout: 7 * time.Second}
	c := NewClient(WithHTTPClient(hc))
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestRootWithCacheTTL(t *testing.T) {
	c := NewClient(WithCacheTTL(5 * time.Minute))
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestRootWithUserAgent(t *testing.T) {
	// No public accessor for userAgent; just assert no panic and client usable.
	c := NewClient(WithUserAgent("test-ua/1.0"))
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestRootWithRDAPBaseURL(t *testing.T) {
	c := NewClient(WithRDAPBaseURL("https://rdap.example.test/"))
	if got := c.RDAPBaseURL(); got != "https://rdap.example.test/" {
		t.Errorf("RDAPBaseURL = %q", got)
	}
}

func TestRootWithWhoisServer(t *testing.T) {
	c := NewClient(WithWhoisServer("whois.example.test:4343"))
	if got := c.WhoisServer(); got != "whois.example.test:4343" {
		t.Errorf("WhoisServer = %q", got)
	}
}

func TestRootWithStatsBaseURL(t *testing.T) {
	c := NewClient(WithStatsBaseURL("https://stats.example.test/"))
	if got := c.StatsBaseURL(); got != "https://stats.example.test/" {
		t.Errorf("StatsBaseURL = %q", got)
	}
}

func TestRootWithRDAPDate(t *testing.T) {
	now := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	c := NewClient(WithRDAPDate(now))
	if got := c.RDAPDate(); !got.Equal(now) {
		t.Errorf("RDAPDate = %v, want %v", got, now)
	}
}

func TestRootWithStealth(t *testing.T) {
	// No public accessor; both true and false must construct without panic.
	if c := NewClient(WithStealth(true)); c == nil {
		t.Fatal("NewClient(WithStealth(true)) returned nil")
	}
	if c := NewClient(WithStealth(false)); c == nil {
		t.Fatal("NewClient(WithStealth(false)) returned nil")
	}
}

func TestRootWithBrowserUserAgent(t *testing.T) {
	c := NewClient(WithBrowserUserAgent("Mozilla/5.0 (test)"))
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestRootWithJitter(t *testing.T) {
	c := NewClient(WithJitter(0, 0))
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestRootWithRateLimit(t *testing.T) {
	c := NewClient(WithRateLimit(10.0))
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestRootWithRRDPBaseURL(t *testing.T) {
	c := NewClient(WithRRDPBaseURL("https://rrdp.example.test"))
	if got := c.RRDPBaseURL(); got != "https://rrdp.example.test" {
		t.Errorf("RRDPBaseURL = %q", got)
	}
}

func TestRootWithThymeBaseURL(t *testing.T) {
	c := NewClient(WithThymeBaseURL("https://thyme.example.test"))
	if got := c.ThymeBaseURL(); got != "https://thyme.example.test" {
		t.Errorf("ThymeBaseURL = %q", got)
	}
}

func TestRootWithFTPBaseURL(t *testing.T) {
	c := NewClient(WithFTPBaseURL("https://ftp.example.test/"))
	if got := c.FTPBaseURL(); got != "https://ftp.example.test/" {
		t.Errorf("FTPBaseURL = %q", got)
	}
}

func TestRootWithRExBaseURL(t *testing.T) {
	c := NewClient(WithRExBaseURL("https://rex.example.test"))
	if got := c.RExBaseURL(); got != "https://rex.example.test" {
		t.Errorf("RExBaseURL = %q", got)
	}
}

func TestRootWithMaxConcurrentDownloads(t *testing.T) {
	c := NewClient(WithMaxConcurrentDownloads(8))
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestRootWithChunkSize(t *testing.T) {
	c := NewClient(WithChunkSize(1 << 20))
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestRootWithDownloadTimeout(t *testing.T) {
	c := NewClient(WithDownloadTimeout(30 * time.Second))
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestRootWithWhoisTimeout(t *testing.T) {
	c := NewClient(WithWhoisTimeout(3 * time.Second))
	if got := c.WhoisTimeout(); got != 3*time.Second {
		t.Errorf("WhoisTimeout = %v, want 3s", got)
	}
}

// --- free re-export helpers (non-Option) --------------------------------

func TestRootNewFilter(t *testing.T) {
	entries := []DelegatedEntry{
		{Country: "AU", Type: "ipv4", Start: "1.0.0.0", Value: 256},
	}
	f := NewFilter(entries)
	if _, ok := interface{}(f).(*filter.EntryFilter); !ok {
		t.Errorf("NewFilter returned %T, want *filter.EntryFilter", f)
	}
}

func TestRootNewExtendedFilter(t *testing.T) {
	entries := []DelegatedExtendedEntry{
		{OpaqueID: "A1", Country: "AU"},
	}
	f := NewExtendedFilter(entries)
	if _, ok := interface{}(f).(*filter.ExtendedEntryFilter); !ok {
		t.Errorf("NewExtendedFilter returned %T, want *filter.ExtendedEntryFilter", f)
	}
}

func TestRootListAvailableYears(t *testing.T) {
	years := ListAvailableYears()
	if len(years) == 0 {
		t.Fatal("expected non-empty year list")
	}
	if years[0] != 2001 {
		t.Errorf("first year = %d, want 2001", years[0])
	}
	if years[len(years)-1] < 2026 {
		t.Errorf("last year = %d, want >= 2026", years[len(years)-1])
	}
	// Sanity: matches the underlying history helper.
	if got, want := len(years), len(history.ListAvailableYears()); got != want {
		t.Errorf("len = %d, want %d", got, want)
	}
}

func TestRootSetLookupAddr(t *testing.T) {
	// SetLookupAddr overrides the reverse-DNS resolver used by ReverseDNS.
	// Install a deterministic stub and confirm it is invoked.
	called := false
	SetLookupAddr(func(ctx context.Context, ip string) ([]string, error) {
		called = true
		return []string{"test.example."}, nil
	})
	defer SetLookupAddr(nil) // reset to default resolver

	c := NewClient(WithJitter(0, 0))
	got, err := c.ReverseDNS(context.Background(), "1.1.1.1")
	if err != nil {
		t.Fatalf("ReverseDNS: %v", err)
	}
	if !called {
		t.Error("custom lookup func was not invoked")
	}
	if len(got) == 0 || got[0] != "test.example." {
		t.Errorf("ReverseDNS = %v", got)
	}
}
