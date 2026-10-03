// Package main implements the apnic CLI, a cobra-based command-line interface that
// exposes every capability of the apnic-skills SDK.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the apnic CLI entry point.
var rootCmd = &cobra.Command{
	Use:   "apnic",
	Short: "APNIC data & query toolkit",
	Long: `apnic is a command-line interface to the APNIC (Asia-Pacific Network Information Centre)
public data services, built on the github.com/hitechcloud-vietnam/apnic-skills SDK.

It provides access to:
  - Delegated / Extended / Assigned / IPv6-assigned / Legacy stats
  - IP/ASN transfers and resource changes
  - RDAP lookups (IP, CIDR, ASN, domain, entity) incl. point-in-time history
  - RDAP entity search
  - Whois queries and reverse DNS
  - MD5 / PGP signature verification of published data
  - Historical stats by date and by year

Run 'apnic <command> --help' for details on any subcommand.`,
}

// global flags shared across subcommands
var (
	flagStatsBaseURL  string
	flagRDAPBaseURL   string
	flagWhoisServer   string
	flagWhoisFlags    string
	flagUserAgent     string
	flagCacheTTL      string
	flagTimeout       string
	flagJSON          bool
	flagStealth       bool
	flagJitter        string
	flagRateLimit     float64
	flagRRDPBaseURL   string
	flagThymeBaseURL  string
	flagFTPBaseURL    string
	flagBrowserUA     string
	flagRExBaseURL    string
	flagMaxConcurrent int
	flagChunkSize     string
	flagDownloadTO    string
	flagBGPSource     string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&flagStatsBaseURL, "stats-base-url", "", "base URL for APNIC stats/FTP data (default: https://ftp.apnic.net/apnic/stats/apnic/)")
	rootCmd.PersistentFlags().StringVar(&flagRDAPBaseURL, "rdap-base-url", "", "base URL for RDAP queries (default: https://rdap.apnic.net)")
	rootCmd.PersistentFlags().StringVar(&flagWhoisServer, "whois-server", "", "whois server address (default: whois.apnic.net:43)")
	rootCmd.PersistentFlags().StringVar(&flagUserAgent, "user-agent", "", "custom User-Agent header (used when --stealth=false)")
	rootCmd.PersistentFlags().StringVar(&flagCacheTTL, "cache-ttl", "", "cache time-to-live, e.g. 30m, 2h (default 30m; 0 disables)")
	rootCmd.PersistentFlags().StringVar(&flagTimeout, "timeout", "", "HTTP request timeout, e.g. 30s, 2m")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "output raw JSON where supported")
	// Anti-scraping / stealth flags.
	rootCmd.PersistentFlags().BoolVar(&flagStealth, "stealth", true, "send browser-mimicry headers + request jitter to avoid bot detection (default true)")
	rootCmd.PersistentFlags().StringVar(&flagJitter, "jitter", "", "random per-request delay range, e.g. 200ms-800ms (default 200ms-800ms; stealth only)")
	rootCmd.PersistentFlags().Float64Var(&flagRateLimit, "rate-limit", 0, "global max requests per second (0 = unlimited)")
	rootCmd.PersistentFlags().StringVar(&flagBrowserUA, "browser-ua", "", "User-Agent used when stealth is enabled (default: a Chrome UA)")
	// Additional service base URLs.
	rootCmd.PersistentFlags().StringVar(&flagRRDPBaseURL, "rrdp-base-url", "", "base URL for RPKI RRDP (default: https://rrdp.apnic.net)")
	rootCmd.PersistentFlags().StringVar(&flagThymeBaseURL, "thyme-base-url", "", "base URL for thyme BGP analysis (default: https://thyme.apnic.net)")
	rootCmd.PersistentFlags().StringVar(&flagFTPBaseURL, "ftp-base-url", "", "APNIC FTP root for IRR/transfers-all/telemetry (default: https://ftp.apnic.net/)")
	rootCmd.PersistentFlags().StringVar(&flagRExBaseURL, "rex-base-url", "", "base URL for the REx cross-RIR resource registry (default: https://api.rex.apnic.net)")
	// Chunked download flags — APNIC FTP throttles large files per-connection,
	// so parallel Range requests multiply throughput.
	rootCmd.PersistentFlags().IntVar(&flagMaxConcurrent, "max-concurrent-downloads", 4, "parallel Range requests for large-file download (0 or 1 disables chunking)")
	rootCmd.PersistentFlags().StringVar(&flagChunkSize, "chunk-size", "", "target chunk size for download, e.g. 1MB, 512KB (default: split evenly)")
	rootCmd.PersistentFlags().StringVar(&flagDownloadTO, "download-timeout", "", "per-chunk download timeout, e.g. 120s (default: inherits --timeout)")
	rootCmd.PersistentFlags().StringVar(&flagBGPSource, "bgp-source", "", "thyme BGP data source: current (default), au, or hk")
}

// runRoot executes the root command and returns its error, so main and tests
// share a single code path.
func runRoot() error {
	return rootCmd.Execute()
}

// osExit is indirected so tests can exercise main's error path without killing
// the test binary. It mirrors os.Exit's signature.
var osExit = os.Exit

// runMain is the testable body of main: it runs the root command and, on error,
// prints the message to stderr and exits non-zero.
func runMain() int {
	if err := runRoot(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func main() {
	osExit(runMain())
}
