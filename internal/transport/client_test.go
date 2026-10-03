package transport

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("client should not be nil")
	}
	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
	if client.whoisServer != "whois.apnic.net:43" {
		t.Errorf("unexpected whoisServer: %s", client.whoisServer)
	}
	if client.whoisTimeout != 10*time.Second {
		t.Errorf("unexpected whoisTimeout: %v", client.whoisTimeout)
	}
	if client.rdapBaseURL != "https://rdap.apnic.net" {
		t.Errorf("unexpected rdapBaseURL: %s", client.rdapBaseURL)
	}
	if client.statsBaseURL != "https://ftp.apnic.net/apnic/stats/apnic/" {
		t.Errorf("unexpected statsBaseURL: %s", client.statsBaseURL)
	}
	if client.cache == nil {
		t.Error("cache should not be nil")
	}
	if client.userAgent != "APNIC-Go-SDK/1.0 (security)" {
		t.Errorf("unexpected userAgent: %s", client.userAgent)
	}
}

func TestWithHTTPClient(t *testing.T) {
	hc := &http.Client{Timeout: 5 * time.Second}
	client := NewClient(WithHTTPClient(hc))
	if client.httpClient != hc {
		t.Error("httpClient should be the custom one")
	}
}

func TestWithCacheTTL(t *testing.T) {
	ttl := 10 * time.Minute
	client := NewClient(WithCacheTTL(ttl))
	if client.cache.ttl != ttl {
		t.Errorf("expected TTL %v, got %v", ttl, client.cache.ttl)
	}
}

func TestWithUserAgent(t *testing.T) {
	ua := "test-agent/1.0"
	client := NewClient(WithUserAgent(ua))
	if client.userAgent != ua {
		t.Errorf("expected userAgent %s, got %s", ua, client.userAgent)
	}
}

func TestWithRDAPBaseURL(t *testing.T) {
	url := "https://custom-rdap.example.com"
	client := NewClient(WithRDAPBaseURL(url))
	if client.rdapBaseURL != url {
		t.Errorf("expected rdapBaseURL %s, got %s", url, client.rdapBaseURL)
	}
}

func TestWithWhoisServer(t *testing.T) {
	server := "whois.example.com:43"
	client := NewClient(WithWhoisServer(server))
	if client.whoisServer != server {
		t.Errorf("expected whoisServer %s, got %s", server, client.whoisServer)
	}
}

func TestWithWhoisTimeout(t *testing.T) {
	timeout := 20 * time.Second
	client := NewClient(WithWhoisTimeout(timeout))
	if client.whoisTimeout != timeout {
		t.Errorf("expected whoisTimeout %v, got %v", timeout, client.whoisTimeout)
	}
}

func TestWithStatsBaseURL(t *testing.T) {
	url := "https://custom-stats.example.com/"
	client := NewClient(WithStatsBaseURL(url))
	if client.statsBaseURL != url {
		t.Errorf("expected statsBaseURL %s, got %s", url, client.statsBaseURL)
	}
}

func TestMultipleOptions(t *testing.T) {
	client := NewClient(
		WithUserAgent("multi-test"),
		WithCacheTTL(5*time.Minute),
		WithRDAPBaseURL("https://rdap.test.com"),
		WithStatsBaseURL("https://stats.test.com/"),
	)
	if client.userAgent != "multi-test" {
		t.Error("userAgent not applied")
	}
	if client.cache.ttl != 5*time.Minute {
		t.Error("cacheTTL not applied")
	}
	if client.rdapBaseURL != "https://rdap.test.com" {
		t.Error("rdapBaseURL not applied")
	}
	if client.statsBaseURL != "https://stats.test.com/" {
		t.Error("statsBaseURL not applied")
	}
}
