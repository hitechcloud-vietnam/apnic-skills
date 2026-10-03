package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	apnic "github.com/hitechcloud-vietnam/apnic-skills"
)

// newClient builds an apnic.Client from the global CLI flags.
func newClient(extra ...apnic.Option) *apnic.Client {
	opts := []apnic.Option{}

	if flagStatsBaseURL != "" {
		opts = append(opts, apnic.WithStatsBaseURL(flagStatsBaseURL))
	}
	if flagRDAPBaseURL != "" {
		opts = append(opts, apnic.WithRDAPBaseURL(flagRDAPBaseURL))
	}
	if flagWhoisServer != "" {
		opts = append(opts, apnic.WithWhoisServer(flagWhoisServer))
	}
	if flagUserAgent != "" {
		opts = append(opts, apnic.WithUserAgent(flagUserAgent))
	}
	if flagCacheTTL != "" {
		if d, err := time.ParseDuration(flagCacheTTL); err == nil {
			opts = append(opts, apnic.WithCacheTTL(d))
		}
	}
	if flagTimeout != "" {
		if d, err := time.ParseDuration(flagTimeout); err == nil {
			opts = append(opts, apnic.WithHTTPClient(&http.Client{Timeout: d}))
		}
	}

	// Anti-scraping / stealth options.
	opts = append(opts, apnic.WithStealth(flagStealth))
	if flagBrowserUA != "" {
		opts = append(opts, apnic.WithBrowserUserAgent(flagBrowserUA))
	}
	if flagJitter != "" {
		if mn, mx, ok := parseJitterRange(flagJitter); ok {
			opts = append(opts, apnic.WithJitter(mn, mx))
		}
	}
	if flagRateLimit > 0 {
		opts = append(opts, apnic.WithRateLimit(flagRateLimit))
	}

	// Additional service base URLs.
	if flagRRDPBaseURL != "" {
		opts = append(opts, apnic.WithRRDPBaseURL(flagRRDPBaseURL))
	}
	if flagThymeBaseURL != "" {
		opts = append(opts, apnic.WithThymeBaseURL(flagThymeBaseURL))
	}
	if flagFTPBaseURL != "" {
		opts = append(opts, apnic.WithFTPBaseURL(flagFTPBaseURL))
	}
	if flagRExBaseURL != "" {
		opts = append(opts, apnic.WithRExBaseURL(flagRExBaseURL))
	}

	// Chunked download options.
	opts = append(opts, apnic.WithMaxConcurrentDownloads(flagMaxConcurrent))
	if flagChunkSize != "" {
		if n, ok := parseByteSize(flagChunkSize); ok {
			opts = append(opts, apnic.WithChunkSize(n))
		}
	}
	if flagDownloadTO != "" {
		if d, err := time.ParseDuration(flagDownloadTO); err == nil {
			opts = append(opts, apnic.WithDownloadTimeout(d))
		}
	}

	opts = append(opts, extra...)
	return apnic.NewClient(opts...)
}

// parseByteSize parses a human-readable byte size such as "1MB", "512KB", "2M",
// or "1048576" into bytes. Returns false if the string cannot be parsed.
func parseByteSize(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	mult := int64(1)
	num := s
	switch {
	case strings.HasSuffix(s, "KB"), strings.HasSuffix(s, "kb"):
		mult, num = 1024, s[:len(s)-2]
	case strings.HasSuffix(s, "MB"), strings.HasSuffix(s, "mb"):
		mult, num = 1024*1024, s[:len(s)-2]
	case strings.HasSuffix(s, "GB"), strings.HasSuffix(s, "gb"):
		mult, num = 1024*1024*1024, s[:len(s)-2]
	case strings.HasSuffix(s, "K"), strings.HasSuffix(s, "k"):
		mult, num = 1024, s[:len(s)-1]
	case strings.HasSuffix(s, "M"), strings.HasSuffix(s, "m"):
		mult, num = 1024*1024, s[:len(s)-1]
	case strings.HasSuffix(s, "G"), strings.HasSuffix(s, "g"):
		mult, num = 1024*1024*1024, s[:len(s)-1]
	}
	n, err := strconv.ParseInt(num, 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n * mult, true
}

// parseJitterRange parses a "min-max" duration range (e.g. "200ms-800ms").
// Returns false if the string cannot be parsed.
func parseJitterRange(s string) (time.Duration, time.Duration, bool) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	mn, err := time.ParseDuration(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, false
	}
	mx, err := time.ParseDuration(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, false
	}
	return mn, mx, true
}
