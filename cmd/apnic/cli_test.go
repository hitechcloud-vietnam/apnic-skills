package main

import (
	"bytes"
	"context"
	"errors"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	apnic "github.com/hitechcloud-vietnam/apnic-skills"
)

// init disables request jitter for the whole CLI test binary so it runs fast.
func init() {
	_ = os.Setenv("APNIC_NO_JITTER", "1")
}

// runWithStatsServer executes a cobra args vector against a mock stats+RDAP server,
// returning combined stdout output. It resets global flag state between runs.
// stdout is captured via an os.Pipe because commands write directly with fmt.Print.
func runWithStatsServer(t *testing.T, args []string) (string, error) {
	t.Helper()
	server := newCLITestServer()
	defer server.Close()

	// Inject server URLs via the global flags consumed by newClient().
	prevStats, prevRDAP := flagStatsBaseURL, flagRDAPBaseURL
	prevJSON, prevTimeout := flagJSON, flagTimeout
	prevFTP, prevRRDP, prevThyme := flagFTPBaseURL, flagRRDPBaseURL, flagThymeBaseURL
	prevREx := flagRExBaseURL
	prevMC, prevCS, prevDTO := flagMaxConcurrent, flagChunkSize, flagDownloadTO
	prevBGPSource := flagBGPSource
	flagStatsBaseURL = server.URL + "/"
	flagRDAPBaseURL = server.URL
	flagFTPBaseURL = server.URL + "/"
	flagRRDPBaseURL = server.URL
	flagThymeBaseURL = server.URL
	flagRExBaseURL = server.URL
	flagTimeout = "30s"
	// Disable chunked download for CLI tests: the mock handlers don't honour
	// Range, so chunking would only add a probe round-trip. singleStream
	// preserves the pre-chunking behavior the existing assertions expect.
	flagMaxConcurrent = 0
	flagChunkSize = ""
	flagDownloadTO = ""
	defer func() {
		flagStatsBaseURL, flagRDAPBaseURL = prevStats, prevRDAP
		flagJSON, flagTimeout = prevJSON, prevTimeout
		flagFTPBaseURL, flagRRDPBaseURL, flagThymeBaseURL = prevFTP, prevRRDP, prevThyme
		flagRExBaseURL = prevREx
		flagMaxConcurrent, flagChunkSize, flagDownloadTO = prevMC, prevCS, prevDTO
		flagBGPSource = prevBGPSource
	}()

	// Capture stdout.
	oldStdout := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe
	rootCmd.SetArgs(args)
	execErr := make(chan error, 1)
	go func() {
		execErr <- rootCmd.Execute()
		_ = wPipe.Close()
	}()
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = buf.ReadFrom(rPipe)
		close(done)
	}()
	err := <-execErr
	_ = wPipe.Sync()
	os.Stdout = oldStdout
	<-done
	rootCmd.SetArgs([]string{}) // reset
	return buf.String(), err
}

func resetFlags() {
	flagStatsBaseURL = ""
	flagRDAPBaseURL = ""
	flagWhoisServer = ""
	flagUserAgent = ""
	flagCacheTTL = ""
	flagTimeout = ""
	flagJSON = false
	flagStealth = true
	flagJitter = ""
	flagRateLimit = 0
	flagRRDPBaseURL = ""
	flagThymeBaseURL = ""
	flagFTPBaseURL = ""
	flagBrowserUA = ""
	flagRExBaseURL = ""
	flagMaxConcurrent = 0
	flagChunkSize = ""
	flagDownloadTO = ""
	flagBGPSource = ""
	statsDateFlag = ""
	transfersYear = 0
	transfersAllDate = ""
	telemetryDate = ""
	changesDate = ""
	rdapDateFlag = ""
	histType = "delegated"
	histDate = ""
	histYear = 0
	filterSource = "delegated"
	filterCountry = ""
	filterType = ""
	filterStatus = ""
	filterOpaqueID = ""
	verifyDataType = ""
	verifyDate = ""
}

func TestCLI_Delegated(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"delegated"})
	if err != nil {
		t.Fatalf("delegated: %v", err)
	}
	if !strings.Contains(out, "delegated stats") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_DelegatedJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"delegated"})
	if err != nil {
		t.Fatalf("delegated --json: %v", err)
	}
	if !strings.Contains(out, `"Entries"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_Extended(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"extended"})
	if err != nil {
		t.Fatalf("extended: %v", err)
	}
	if !strings.Contains(out, "extended stats") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_Assigned(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"assigned"})
	if err != nil {
		t.Fatalf("assigned: %v", err)
	}
	if !strings.Contains(out, "assigned stats") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_IPv6Assigned(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"ipv6-assigned"})
	if err != nil {
		t.Fatalf("ipv6-assigned: %v", err)
	}
	if !strings.Contains(out, "ipv6-assigned stats") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_Legacy(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"legacy"})
	if err != nil {
		t.Fatalf("legacy: %v", err)
	}
	if !strings.Contains(out, "legacy stats") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_Transfers(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"transfers"})
	if err != nil {
		t.Fatalf("transfers: %v", err)
	}
	if !strings.Contains(out, "transfers:") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_TransfersByYear(t *testing.T) {
	resetFlags()
	transfersYear = 2026
	out, err := runWithStatsServer(t, []string{"transfers"})
	if err != nil {
		t.Fatalf("transfers --year: %v", err)
	}
	if !strings.Contains(out, "year=2026") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_Changes(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"changes"})
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if !strings.Contains(out, "changes:") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_TransfersAll(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"transfers-all"})
	if err != nil {
		t.Fatalf("transfers-all: %v", err)
	}
	if !strings.Contains(out, "transfers-all:") {
		t.Errorf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "45745") {
		t.Errorf("expected ASN record in output: %s", out)
	}
}

func TestCLI_TransfersAllByDate(t *testing.T) {
	resetFlags()
	transfersAllDate = "20200615"
	out, err := runWithStatsServer(t, []string{"transfers-all"})
	if err != nil {
		t.Fatalf("transfers-all --date: %v", err)
	}
	if !strings.Contains(out, "date=20200615") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_TransfersAllJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"transfers-all"})
	if err != nil {
		t.Fatalf("transfers-all --json: %v", err)
	}
	if !strings.Contains(out, `"Records"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_Telemetry(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"stats-telemetry"})
	if err != nil {
		t.Fatalf("stats-telemetry: %v", err)
	}
	if !strings.Contains(out, "whois-rdap-telemetry:") {
		t.Errorf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "total=3070925") {
		t.Errorf("expected total queries in output: %s", out)
	}
}

func TestCLI_TelemetryByDate(t *testing.T) {
	resetFlags()
	telemetryDate = "20260701"
	out, err := runWithStatsServer(t, []string{"stats-telemetry"})
	if err != nil {
		t.Fatalf("stats-telemetry --date: %v", err)
	}
	if !strings.Contains(out, "date=20260701") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_TelemetryJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"stats-telemetry"})
	if err != nil {
		t.Fatalf("stats-telemetry --json: %v", err)
	}
	if !strings.Contains(out, `"RDAP"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_IRR(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"irr", "inetnum"})
	if err != nil {
		t.Fatalf("irr inetnum: %v", err)
	}
	if !strings.Contains(out, "irr inetnum:") {
		t.Errorf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "1.1.1.0 - 1.1.1.255") {
		t.Errorf("expected primary key in output: %s", out)
	}
}

func TestCLI_IRRSerial(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"irr", "serial"})
	if err != nil {
		t.Fatalf("irr serial: %v", err)
	}
	if !strings.Contains(out, "16159398") {
		t.Errorf("expected serial in output: %s", out)
	}
}

func TestCLI_IRRSerialJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"irr", "serial"})
	if err != nil {
		t.Fatalf("irr serial --json: %v", err)
	}
	if !strings.Contains(out, `"serial"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_IRRInvalidType(t *testing.T) {
	resetFlags()
	if _, err := runWithStatsServer(t, []string{"irr", "bogus"}); err == nil {
		t.Error("expected error for invalid IRR type")
	}
}

func TestCLI_BGPSummary(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"bgp", "summary"})
	if err != nil {
		t.Fatalf("bgp summary: %v", err)
	}
	if !strings.Contains(out, "bgp summary:") {
		t.Errorf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "BGP routing table entries examined") {
		t.Errorf("expected metric in output: %s", out)
	}
}

func TestCLI_BGPRawTable(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"bgp", "raw-table"})
	if err != nil {
		t.Fatalf("bgp raw-table: %v", err)
	}
	if !strings.Contains(out, "bgp raw-table:") {
		t.Errorf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "1.0.0.0/24") {
		t.Errorf("expected route in output: %s", out)
	}
}

func TestCLI_BGPASNMap(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"bgp", "asn-map"})
	if err != nil {
		t.Fatalf("bgp asn-map: %v", err)
	}
	if !strings.Contains(out, "bgp asn-map:") {
		t.Errorf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "unique origin ASNs") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_RPKINotification(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rpki", "notification"})
	if err != nil {
		t.Fatalf("rpki notification: %v", err)
	}
	if !strings.Contains(out, "rpki notification:") {
		t.Errorf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "session=8dad0cc8") {
		t.Errorf("expected session in output: %s", out)
	}
	if !strings.Contains(out, "serial=65148") {
		t.Errorf("expected serial in output: %s", out)
	}
}

func TestCLI_RPKISnapshot(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rpki", "snapshot", "snapshot.xml"})
	if err != nil {
		t.Fatalf("rpki snapshot: %v", err)
	}
	if !strings.Contains(out, "rpki snapshot:") {
		t.Errorf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "published=1") {
		t.Errorf("expected published count in output: %s", out)
	}
	if !strings.Contains(out, "withdrawn=1") {
		t.Errorf("expected withdrawn count in output: %s", out)
	}
}

// TestCLI_RPKISnapshot_AbsoluteURI covers the absolute-URI branch of
// resolveSnapshotURI (http:// prefix is used as-is).
func TestCLI_RPKISnapshot_AbsoluteURI(t *testing.T) {
	resetFlags()
	srv := newCLITestServer()
	defer srv.Close()
	prev := flagRRDPBaseURL
	flagRRDPBaseURL = srv.URL
	defer func() { flagRRDPBaseURL = prev }()
	// Use the server's own snapshot.xml absolute URL.
	out, err := runWithStatsServer(t, []string{"rpki", "snapshot", srv.URL + "/snapshot.xml"})
	if err != nil {
		t.Fatalf("rpki snapshot absolute: %v", err)
	}
	if !strings.Contains(out, "rpki snapshot:") {
		t.Errorf("unexpected output: %s", out)
	}
}

// TestCLI_RPKISnapshot_FromNotification covers the no-arg branch of
// resolveSnapshotURI (fetch notification, use its snapshot URI). The notification
// fetch itself is exercised against the mock server; the returned snapshot URI
// is the notification's own (absolute) reference, so we only assert it is
// non-empty here.
func TestCLI_RPKISnapshot_FromNotification(t *testing.T) {
	resetFlags()
	srv := newCLITestServer()
	defer srv.Close()
	prev := flagRRDPBaseURL
	flagRRDPBaseURL = srv.URL
	defer func() { flagRRDPBaseURL = prev }()
	client := newClient()
	uri, err := resolveSnapshotURI(client, nil)
	if err != nil {
		t.Fatalf("resolveSnapshotURI no-arg: %v", err)
	}
	if uri == "" {
		t.Error("expected non-empty snapshot URI from notification")
	}
}

// TestCLI_RPKISnapshot_FromNotificationFetchError covers the no-arg error
// branch of resolveSnapshotURI: when the notification fetch fails, the error
// must propagate (cmd_rpki.go lines 114-115).
func TestCLI_RPKISnapshot_FromNotificationFetchError(t *testing.T) {
	resetFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	prev := flagRRDPBaseURL
	flagRRDPBaseURL = srv.URL
	defer func() { flagRRDPBaseURL = prev }()
	client := newClient()
	if _, err := resolveSnapshotURI(client, nil); err == nil {
		t.Error("expected error when notification fetch fails")
	}
}

// TestRRDPBaseURL_Default covers the default-branch (flag unset returns SDK default).
func TestRRDPBaseURL_Default(t *testing.T) {
	resetFlags()
	if got := rrdpBaseURL(); got != apnic.DefaultRRDPBaseURL {
		t.Errorf("rrdpBaseURL() default = %q, want %q", got, apnic.DefaultRRDPBaseURL)
	}
	flagRRDPBaseURL = "https://custom.example.com/"
	if got := rrdpBaseURL(); got != "https://custom.example.com/" {
		t.Errorf("rrdpBaseURL() flag = %q, want custom", got)
	}
}

// TestRunMain_SuccessAndError exercises runMain (the testable body of main) on
// both the success path (no args → Execute returns nil) and the error path
// (unknown subcommand → Execute returns an error, runMain returns 1). It also
// drives runRoot directly to cover that function. The osExit indirection is
// replaced so main() can be called without terminating the test binary, and
// os.Args is saved/restored around the call.
func TestRunMain_SuccessAndError(t *testing.T) {
	resetFlags()
	savedArgs := os.Args
	savedExit := osExit
	defer func() {
		os.Args = savedArgs
		osExit = savedExit
	}()

	// Success path: `apnic` with no subcommand. cobra prints help to stdout
	// and returns nil for the bare root command.
	var exitCalls int
	osExit = func(int) { exitCalls++ }
	os.Args = []string{"apnic"}
	rootCmd.SetArgs([]string{})
	if rc := runMain(); rc != 0 {
		t.Errorf("runMain() success = %d, want 0", rc)
	}
	if err := runRoot(); err != nil {
		t.Errorf("runRoot() success error: %v", err)
	}

	// Error path: an unknown subcommand makes Execute return an error.
	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"definitely-not-a-subcommand"})
	// cobra writes usage/error to stderr; capture via SetErr + SetOut.
	rootCmd.SetErr(&buf)
	prevOut := rootCmd.OutOrStdout()
	rootCmd.SetOut(&buf)
	defer rootCmd.SetOut(prevOut)
	if rc := runMain(); rc != 1 {
		t.Errorf("runMain() error = %d, want 1", rc)
	}
	if buf.Len() == 0 {
		t.Error("expected error output on stderr/stdout for unknown subcommand")
	}
	if exitCalls != 0 {
		t.Errorf("osExit called %d times during runMain; main() should not be invoked", exitCalls)
	}
}

// TestMain_Entry covers the program entry point main() itself by invoking it
// directly with a neutered osExit (so the test binary survives). This covers
// the osExit(runMain()) statement in main.go.
func TestMain_Entry(t *testing.T) {
	resetFlags()
	savedExit := osExit
	savedArgs := os.Args
	defer func() {
		osExit = savedExit
		os.Args = savedArgs
	}()
	osExit = func(int) {} // no-op so main() returns
	os.Args = []string{"apnic"}
	rootCmd.SetArgs([]string{})
	main()
}

func TestCLI_Years(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"years"})
	if err != nil {
		t.Fatalf("years: %v", err)
	}
	if !strings.Contains(out, "2001") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_HistoryByDate(t *testing.T) {
	resetFlags()
	histType = "delegated"
	histDate = "20260627"
	out, err := runWithStatsServer(t, []string{"history"})
	if err != nil {
		t.Fatalf("history --date: %v", err)
	}
	if !strings.Contains(out, "delegated history") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_HistoryByDateExtended(t *testing.T) {
	resetFlags()
	histType = "extended"
	histDate = "20260627"
	out, err := runWithStatsServer(t, []string{"history"})
	if err != nil {
		t.Fatalf("history extended --date: %v", err)
	}
	if !strings.Contains(out, "extended history") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_HistoryByDateAssigned(t *testing.T) {
	resetFlags()
	histType = "assigned"
	histDate = "20260627"
	out, err := runWithStatsServer(t, []string{"history"})
	if err != nil {
		t.Fatalf("history assigned --date: %v", err)
	}
	if !strings.Contains(out, "assigned history") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_HistoryByDateLegacy(t *testing.T) {
	resetFlags()
	histType = "legacy"
	histDate = "20260627"
	out, err := runWithStatsServer(t, []string{"history"})
	if err != nil {
		t.Fatalf("history legacy --date: %v", err)
	}
	if !strings.Contains(out, "legacy history") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_HistoryByYear(t *testing.T) {
	resetFlags()
	histType = "delegated"
	histYear = 2026
	out, err := runWithStatsServer(t, []string{"history"})
	if err != nil {
		t.Fatalf("history --year: %v", err)
	}
	if !strings.Contains(out, "by-year") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_HistoryByYearExtended(t *testing.T) {
	resetFlags()
	histType = "extended"
	histYear = 2026
	out, err := runWithStatsServer(t, []string{"history"})
	if err != nil {
		t.Fatalf("history extended --year: %v", err)
	}
	if !strings.Contains(out, "by-year") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_HistoryErrors(t *testing.T) {
	resetFlags()
	histDate = ""
	histYear = 0
	if _, err := runWithStatsServer(t, []string{"history"}); err == nil {
		t.Error("expected error when neither --date nor --year given")
	}

	resetFlags()
	histDate = "20260627"
	histYear = 2026
	if _, err := runWithStatsServer(t, []string{"history"}); err == nil {
		t.Error("expected error when both --date and --year given")
	}

	resetFlags()
	histType = "bogus"
	histDate = "20260627"
	if _, err := runWithStatsServer(t, []string{"history"}); err == nil {
		t.Error("expected error for unknown --type")
	}

	resetFlags()
	histType = "legacy"
	histYear = 2026
	if _, err := runWithStatsServer(t, []string{"history"}); err == nil {
		t.Error("expected error for --year with unsupported --type legacy")
	}
}

func TestCLI_FilterDelegated(t *testing.T) {
	resetFlags()
	filterSource = "delegated"
	filterCountry = "CN"
	out, err := runWithStatsServer(t, []string{"filter"})
	if err != nil {
		t.Fatalf("filter delegated: %v", err)
	}
	if !strings.Contains(out, "after filter") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_FilterExtended(t *testing.T) {
	resetFlags()
	filterSource = "extended"
	filterOpaqueID = "A92E1062"
	out, err := runWithStatsServer(t, []string{"filter"})
	if err != nil {
		t.Fatalf("filter extended: %v", err)
	}
	if !strings.Contains(out, "after filter") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_FilterBadSource(t *testing.T) {
	resetFlags()
	filterSource = "bogus"
	if _, err := runWithStatsServer(t, []string{"filter"}); err == nil {
		t.Error("expected error for bad --source")
	}
}

func TestCLI_RDAPIP(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rdap", "ip", "1.1.1.1"})
	if err != nil {
		t.Fatalf("rdap ip: %v", err)
	}
	if !strings.Contains(out, "Handle:") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_RDAPCIDR(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rdap", "cidr", "1.1.1.0/24"})
	if err != nil {
		t.Fatalf("rdap cidr: %v", err)
	}
	if !strings.Contains(out, "Handle:") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_RDAPASN(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rdap", "asn", "13335"})
	if err != nil {
		t.Fatalf("rdap asn: %v", err)
	}
	if !strings.Contains(out, "ASN:") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_RDAPASNInvalid(t *testing.T) {
	resetFlags()
	if _, err := runWithStatsServer(t, []string{"rdap", "asn", "notanumber"}); err == nil {
		t.Error("expected error for non-numeric ASN")
	}
}

func TestCLI_RDAPDomain(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rdap", "domain", "1.0.0.1.in-addr.arpa"})
	if err != nil {
		t.Fatalf("rdap domain: %v", err)
	}
	if !strings.Contains(out, "Handle:") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_RDAPEntity(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rdap", "entity", "AIC3-AP"})
	if err != nil {
		t.Fatalf("rdap entity: %v", err)
	}
	if !strings.Contains(out, "Handle:") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_RDAPSearch(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rdap", "search", "*CLOUD*"})
	if err != nil {
		t.Fatalf("rdap search: %v", err)
	}
	if !strings.Contains(out, "results") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_RDAPSearchHandle(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rdap", "search", "AIC3-AP", "--field", "handle"})
	if err != nil {
		t.Fatalf("rdap search handle: %v", err)
	}
	if !strings.Contains(out, "handle=") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_RDAPDateFlag(t *testing.T) {
	resetFlags()
	rdapDateFlag = "2020-06-01T00:00:00Z"
	out, err := runWithStatsServer(t, []string{"rdap", "ip", "1.1.1.1"})
	if err != nil {
		t.Fatalf("rdap ip --date: %v", err)
	}
	if !strings.Contains(out, "Handle:") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_VerifyMD5(t *testing.T) {
	resetFlags()
	verifyDataType = "delegated"
	out, err := runWithStatsServer(t, []string{"verify", "md5"})
	if err != nil {
		t.Fatalf("verify md5: %v", err)
	}
	if out == "" {
		t.Error("expected md5 output")
	}
}

func TestCLI_VerifyMD5MissingType(t *testing.T) {
	resetFlags()
	verifyDataType = ""
	if _, err := runWithStatsServer(t, []string{"verify", "md5"}); err == nil {
		t.Error("expected error when --type missing")
	}
}

func TestCLI_VerifyASC(t *testing.T) {
	resetFlags()
	verifyDataType = "delegated"
	out, err := runWithStatsServer(t, []string{"verify", "asc"})
	if err != nil {
		t.Fatalf("verify asc: %v", err)
	}
	if out == "" {
		t.Error("expected asc output")
	}
}

func TestCLI_VerifyASCMissingType(t *testing.T) {
	resetFlags()
	verifyDataType = ""
	if _, err := runWithStatsServer(t, []string{"verify", "asc"}); err == nil {
		t.Error("expected error when --type missing")
	}
}

func TestCLI_VerifyPubKey(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"verify", "pubkey"})
	if err != nil {
		t.Fatalf("verify pubkey: %v", err)
	}
	if out == "" {
		t.Error("expected pubkey output")
	}
}

func TestCLI_VerifyIntegrity(t *testing.T) {
	resetFlags()
	verifyDataType = "delegated"
	out, err := runWithStatsServer(t, []string{"verify", "integrity"})
	if err != nil {
		t.Fatalf("verify integrity: %v", err)
	}
	if !strings.Contains(out, "verified") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_VerifyIntegrityMissingType(t *testing.T) {
	resetFlags()
	verifyDataType = ""
	if _, err := runWithStatsServer(t, []string{"verify", "integrity"}); err == nil {
		t.Error("expected error when --type missing")
	}
}

func TestCLI_HelpNoError(t *testing.T) {
	resetFlags()
	_, err := runWithStatsServer(t, []string{"--help"})
	if err != nil {
		t.Errorf("--help should not error: %v", err)
	}
}

// mockWhoisTCPServer starts a minimal TCP whois server returning the given payload.
func mockWhoisTCPServer(t *testing.T, response string) (addr string, cleanup func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				c.Read(buf)
				c.Write([]byte(response))
			}(conn)
		}
	}()
	return ln.Addr().String(), func() { ln.Close() }
}

const sampleWhois = `inetnum:  1.1.1.0 - 1.1.1.255
netname:  APNIC-LABS
descr:    APNIC and Cloudflare DNS Resolver project
country:  AU
abuse-mailbox: helpdesk@apnic.net
status:   ASSIGNED PORTABLE
last-modified: 2023-04-26T22:57:58Z

route:    1.1.1.0/24
origin:   AS13335
last-modified: 2023-04-26T02:42:44Z
`

func TestCLI_WhoisIP(t *testing.T) {
	resetFlags()
	addr, cleanup := mockWhoisTCPServer(t, sampleWhois)
	defer cleanup()
	flagWhoisServer = addr
	out, err := runWithStatsServer(t, []string{"whois", "ip", "1.1.1.1"})
	if err != nil {
		t.Fatalf("whois ip: %v", err)
	}
	if !strings.Contains(out, "Network:") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_WhoisASN(t *testing.T) {
	resetFlags()
	addr, cleanup := mockWhoisTCPServer(t, sampleWhois)
	defer cleanup()
	flagWhoisServer = addr
	out, err := runWithStatsServer(t, []string{"whois", "asn", "AS13335"})
	if err != nil {
		t.Fatalf("whois asn: %v", err)
	}
	if !strings.Contains(out, "Network:") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_WhoisASNInvalid(t *testing.T) {
	resetFlags()
	if _, err := runWithStatsServer(t, []string{"whois", "asn", "notanumber"}); err == nil {
		t.Error("expected error for non-numeric ASN")
	}
}

func TestCLI_WhoisRaw(t *testing.T) {
	resetFlags()
	addr, cleanup := mockWhoisTCPServer(t, sampleWhois)
	defer cleanup()
	flagWhoisServer = addr
	out, err := runWithStatsServer(t, []string{"whois", "raw", "1.1.1.1"})
	if err != nil {
		t.Fatalf("whois raw: %v", err)
	}
	if !strings.Contains(out, "inetnum") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_WhoisRules(t *testing.T) {
	resetFlags()
	addr, cleanup := mockWhoisTCPServer(t, sampleWhois)
	defer cleanup()
	flagWhoisServer = addr
	out, err := runWithStatsServer(t, []string{"whois", "rules", "1.1.1.1", "--scope", "all-less"})
	if err != nil {
		t.Fatalf("whois rules error: %v", err)
	}
	if !strings.Contains(out, "Network:") {
		t.Errorf("expected 'Network:' in output, got %q", out)
	}
	if !strings.Contains(out, "1.1.1.0 - 1.1.1.255") {
		t.Errorf("expected network range in output, got %q", out)
	}
}

func TestCLI_WhoisRulesInvalidScope(t *testing.T) {
	resetFlags()
	addr, cleanup := mockWhoisTCPServer(t, sampleWhois)
	defer cleanup()
	flagWhoisServer = addr
	_, err := runWithStatsServer(t, []string{"whois", "rules", "1.1.1.1", "--scope", "bogus"})
	if err == nil {
		t.Error("expected error for invalid scope")
	}
}

func TestCLI_WhoisRulesJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	addr, cleanup := mockWhoisTCPServer(t, sampleWhois)
	defer cleanup()
	flagWhoisServer = addr
	out, err := runWithStatsServer(t, []string{"whois", "rules", "1.1.1.1", "--scope", "all-less"})
	if err != nil {
		t.Fatalf("whois rules --json error: %v", err)
	}
	if !strings.Contains(out, "[") {
		t.Errorf("expected JSON array '[', got: %s", out)
	}
	if !strings.Contains(out, `"Network"`) {
		t.Errorf("expected 'Network' field in JSON, got: %s", out)
	}
}

func TestCLI_ReverseDNS(t *testing.T) {
	resetFlags()
	apnic.SetLookupAddr(func(ctx context.Context, ip string) ([]string, error) {
		return []string{"one.one.one.one."}, nil
	})
	defer apnic.SetLookupAddr(nil)
	out, err := runWithStatsServer(t, []string{"reverse-dns", "1.1.1.1"})
	if err != nil {
		t.Fatalf("reverse-dns: %v", err)
	}
	if !strings.Contains(out, "one.one.one.one") {
		t.Errorf("expected PTR in output, got: %s", out)
	}
}

func TestCLI_ReverseDNSEmpty(t *testing.T) {
	// Covers the `len(names) == 0` defensive branch (cmd_whois.go:108-111):
	// a successful lookup with no PTR records.
	resetFlags()
	apnic.SetLookupAddr(func(ctx context.Context, ip string) ([]string, error) {
		return []string{}, nil
	})
	defer apnic.SetLookupAddr(nil)
	out, err := runWithStatsServer(t, []string{"reverse-dns", "192.0.2.1"})
	if err != nil {
		t.Fatalf("reverse-dns empty: %v", err)
	}
	if !strings.Contains(out, "(no PTR records)") {
		t.Errorf("expected no-PTR marker, got: %s", out)
	}
}

func TestCLI_ReverseDNSInvalidIP(t *testing.T) {
	resetFlags()
	apnic.SetLookupAddr(func(ctx context.Context, ip string) ([]string, error) {
		return nil, errors.New("invalid IP")
	})
	defer apnic.SetLookupAddr(nil)
	if _, err := runWithStatsServer(t, []string{"reverse-dns", "not-an-ip"}); err == nil {
		t.Error("expected error for invalid IP")
	}
}

func TestCLI_RDAPCIDRv6(t *testing.T) {
	// Exercise the v6prefix branch of printRDAPNetwork.
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rdap", "cidr", "2001:db8::/32"})
	if err != nil {
		t.Fatalf("rdap cidr v6: %v", err)
	}
	if !strings.Contains(out, "Handle:") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_NewClientAllFlags(t *testing.T) {
	// Exercise every global-flag branch in newClient.
	resetFlags()
	flagStatsBaseURL = "https://example.com/stats/"
	flagRDAPBaseURL = "https://example.com/rdap"
	flagWhoisServer = "whois.example.com:43"
	flagUserAgent = "test-agent/1.0"
	flagCacheTTL = "5m"
	flagTimeout = "10s"
	c := newClient()
	if c == nil {
		t.Fatal("newClient returned nil")
	}
}

func TestCLI_NewClientBadDuration(t *testing.T) {
	// Invalid durations should be silently ignored, not panic.
	resetFlags()
	flagCacheTTL = "not-a-duration"
	flagTimeout = "also-bad"
	c := newClient()
	if c == nil {
		t.Fatal("newClient returned nil")
	}
}

// TestCLI_NewClientChunkFlags covers the chunkSize + downloadTimeout branches
// in newClient (valid values are applied as Options).
func TestCLI_NewClientChunkFlags(t *testing.T) {
	resetFlags()
	flagChunkSize = "2MB"
	flagDownloadTO = "300s"
	flagMaxConcurrent = 8
	c := newClient()
	if c == nil {
		t.Fatal("newClient returned nil")
	}
}

// TestCLI_NewClientBadChunkFlags covers the invalid-chunkSize / bad-downloadTO
// branches (silently ignored, no panic).
func TestCLI_NewClientBadChunkFlags(t *testing.T) {
	resetFlags()
	flagChunkSize = "not-a-size"
	flagDownloadTO = "not-a-duration"
	c := newClient()
	if c == nil {
		t.Fatal("newClient returned nil")
	}
}

func TestCLI_RdapDateOptionInvalid(t *testing.T) {
	// An invalid --date is ignored (no-op), query still succeeds.
	resetFlags()
	rdapDateFlag = "not-a-date"
	out, err := runWithStatsServer(t, []string{"rdap", "ip", "1.1.1.1"})
	if err != nil {
		t.Fatalf("rdap ip with bad --date: %v", err)
	}
	if !strings.Contains(out, "Handle:") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_FilterDelegatedAllFilters(t *testing.T) {
	// Exercise ByType + ByStatus branches in the delegated path.
	resetFlags()
	filterSource = "delegated"
	filterCountry = "CN"
	filterType = "ipv4"
	filterStatus = "allocated"
	out, err := runWithStatsServer(t, []string{"filter"})
	if err != nil {
		t.Fatalf("filter delegated all: %v", err)
	}
	if !strings.Contains(out, "after filter") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_FilterDelegatedJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	filterSource = "delegated"
	filterCountry = "CN"
	out, err := runWithStatsServer(t, []string{"filter"})
	if err != nil {
		t.Fatalf("filter delegated --json: %v", err)
	}
	if !strings.Contains(out, `"Country"`) {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestCLI_FilterExtendedCountry(t *testing.T) {
	// Use a country filter that matches the sample so the print loop body runs.
	resetFlags()
	filterSource = "extended"
	filterCountry = "AU"
	out, err := runWithStatsServer(t, []string{"filter"})
	if err != nil {
		t.Fatalf("filter extended country: %v", err)
	}
	if !strings.Contains(out, "A91872ED") {
		t.Errorf("expected matched opaque id in output: %s", out)
	}
}

func TestCLI_FilterExtendedJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	filterSource = "extended"
	filterCountry = "AU"
	out, err := runWithStatsServer(t, []string{"filter"})
	if err != nil {
		t.Fatalf("filter extended --json: %v", err)
	}
	if !strings.Contains(out, `"OpaqueID"`) {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestCLI_HistoryByDateJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	histType = "delegated"
	histDate = "20260627"
	out, err := runWithStatsServer(t, []string{"history"})
	if err != nil {
		t.Fatalf("history delegated --json: %v", err)
	}
	if !strings.Contains(out, `"Entries"`) {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestCLI_HistoryByDateExtendedJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	histType = "extended"
	histDate = "20260627"
	out, err := runWithStatsServer(t, []string{"history"})
	if err != nil {
		t.Fatalf("history extended --json: %v", err)
	}
	if !strings.Contains(out, `"Entries"`) {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestCLI_HistoryByDateAssignedJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	histType = "assigned"
	histDate = "20260627"
	out, err := runWithStatsServer(t, []string{"history"})
	if err != nil {
		t.Fatalf("history assigned --json: %v", err)
	}
	if !strings.Contains(out, `"Entries"`) {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestCLI_HistoryByDateLegacyJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	histType = "legacy"
	histDate = "20260627"
	out, err := runWithStatsServer(t, []string{"history"})
	if err != nil {
		t.Fatalf("history legacy --json: %v", err)
	}
	if !strings.Contains(out, `"Entries"`) {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestCLI_HistoryByYearJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	histType = "delegated"
	histYear = 2026
	out, err := runWithStatsServer(t, []string{"history"})
	if err != nil {
		t.Fatalf("history by-year --json: %v", err)
	}
	if !strings.Contains(out, `"Entries"`) {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestCLI_HistoryByYearExtendedJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	histType = "extended"
	histYear = 2026
	out, err := runWithStatsServer(t, []string{"history"})
	if err != nil {
		t.Fatalf("history extended by-year --json: %v", err)
	}
	if !strings.Contains(out, `"Entries"`) {
		t.Errorf("expected JSON, got: %s", out)
	}
}

// runWith404StatsServer runs a server whose stats routes all 404 (only RDAP works),
// to exercise fetch-error branches in history/stats commands.
func runWith404StatsServer(t *testing.T, args []string) (string, error) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only RDAP entity routes respond; everything else 404s.
		if strings.HasPrefix(r.URL.Path, "/ip/") || strings.HasPrefix(r.URL.Path, "/entity/") ||
			strings.HasPrefix(r.URL.Path, "/entities") || strings.HasPrefix(r.URL.Path, "/autnum/") ||
			strings.HasPrefix(r.URL.Path, "/domain/") {
			cliHandler()(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	prevStats, prevRDAP := flagStatsBaseURL, flagRDAPBaseURL
	flagStatsBaseURL = server.URL + "/"
	flagRDAPBaseURL = server.URL
	defer func() { flagStatsBaseURL, flagRDAPBaseURL = prevStats, prevRDAP }()
	oldStdout := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe
	rootCmd.SetArgs(args)
	execErr := make(chan error, 1)
	go func() {
		execErr <- rootCmd.Execute()
		_ = wPipe.Close()
	}()
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() { _, _ = buf.ReadFrom(rPipe); close(done) }()
	err := <-execErr
	os.Stdout = oldStdout
	<-done
	rootCmd.SetArgs([]string{})
	return buf.String(), err
}

// runWithErrorServer runs a cobra args vector against a server that returns
// HTTP 500 for every request, with all base-URL flags (stats/RDAP/FTP/RRDP/
// Thyme/REx) pointed at it. It returns cobra's execution error so callers can
// assert that fetch-error branches return non-nil. flagWhoisServer is left
// untouched (whois error tests set it directly to a dial-failing address).
func runWithErrorServer(t *testing.T, args []string) error {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	prevStats, prevRDAP := flagStatsBaseURL, flagRDAPBaseURL
	prevFTP, prevRRDP, prevThyme := flagFTPBaseURL, flagRRDPBaseURL, flagThymeBaseURL
	prevREx := flagRExBaseURL
	prevMC, prevCS, prevDTO := flagMaxConcurrent, flagChunkSize, flagDownloadTO
	prevBGPSource := flagBGPSource
	flagStatsBaseURL = server.URL + "/"
	flagRDAPBaseURL = server.URL
	flagFTPBaseURL = server.URL + "/"
	flagRRDPBaseURL = server.URL
	flagThymeBaseURL = server.URL
	flagRExBaseURL = server.URL
	flagMaxConcurrent = 0
	flagChunkSize = ""
	flagDownloadTO = ""
	defer func() {
		flagStatsBaseURL, flagRDAPBaseURL = prevStats, prevRDAP
		flagFTPBaseURL, flagRRDPBaseURL, flagThymeBaseURL = prevFTP, prevRRDP, prevThyme
		flagRExBaseURL = prevREx
		flagMaxConcurrent, flagChunkSize, flagDownloadTO = prevMC, prevCS, prevDTO
		flagBGPSource = prevBGPSource
	}()

	oldStdout := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe
	rootCmd.SetArgs(args)
	execErr := make(chan error, 1)
	go func() {
		execErr <- rootCmd.Execute()
		_ = wPipe.Close()
	}()
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() { _, _ = buf.ReadFrom(rPipe); close(done) }()
	err := <-execErr
	os.Stdout = oldStdout
	<-done
	rootCmd.SetArgs([]string{})
	return err
}

// runWithServer runs a cobra args vector against a caller-provided server,
// capturing stdout. All base-URL flags are pointed at srv. Used by tests that
// need a custom handler (e.g. large datasets for the >50/>20 truncation paths).
func runWithServer(t *testing.T, srv *httptest.Server, args []string) (string, error) {
	t.Helper()
	prevStats, prevRDAP := flagStatsBaseURL, flagRDAPBaseURL
	prevJSON, prevTimeout := flagJSON, flagTimeout
	prevFTP, prevRRDP, prevThyme := flagFTPBaseURL, flagRRDPBaseURL, flagThymeBaseURL
	prevREx := flagRExBaseURL
	prevMC, prevCS, prevDTO := flagMaxConcurrent, flagChunkSize, flagDownloadTO
	prevBGPSource := flagBGPSource
	flagStatsBaseURL = srv.URL + "/"
	flagRDAPBaseURL = srv.URL
	flagFTPBaseURL = srv.URL + "/"
	flagRRDPBaseURL = srv.URL
	flagThymeBaseURL = srv.URL
	flagRExBaseURL = srv.URL
	flagTimeout = "30s"
	flagMaxConcurrent = 0
	flagChunkSize = ""
	flagDownloadTO = ""
	defer func() {
		flagStatsBaseURL, flagRDAPBaseURL = prevStats, prevRDAP
		flagJSON, flagTimeout = prevJSON, prevTimeout
		flagFTPBaseURL, flagRRDPBaseURL, flagThymeBaseURL = prevFTP, prevRRDP, prevThyme
		flagRExBaseURL = prevREx
		flagMaxConcurrent, flagChunkSize, flagDownloadTO = prevMC, prevCS, prevDTO
		flagBGPSource = prevBGPSource
	}()

	oldStdout := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe
	rootCmd.SetArgs(args)
	execErr := make(chan error, 1)
	go func() {
		execErr <- rootCmd.Execute()
		_ = wPipe.Close()
	}()
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() { _, _ = buf.ReadFrom(rPipe); close(done) }()
	err := <-execErr
	_ = wPipe.Sync()
	os.Stdout = oldStdout
	<-done
	rootCmd.SetArgs([]string{})
	return buf.String(), err
}

func TestCLI_HistoryByDateFetchError(t *testing.T) {
	// Each --type exercises a fetch-error return-err branch.
	for _, typ := range []string{"delegated", "extended", "assigned", "legacy"} {
		resetFlags()
		histType = typ
		histDate = "20260627"
		if _, err := runWith404StatsServer(t, []string{"history"}); err == nil {
			t.Errorf("expected fetch error for type %s", typ)
		}
	}
}

func TestCLI_HistoryByYearFetchError(t *testing.T) {
	for _, typ := range []string{"delegated", "extended"} {
		resetFlags()
		histType = typ
		histYear = 2026
		if _, err := runWith404StatsServer(t, []string{"history"}); err == nil {
			t.Errorf("expected fetch error for type %s", typ)
		}
	}
}

func TestCLI_VerifyIntegrityWithDate(t *testing.T) {
	// Cover the dateOrDefault non-empty branch via --date.
	resetFlags()
	verifyDataType = "delegated"
	verifyDate = "20260627"
	out, err := runWithStatsServer(t, []string{"verify", "integrity"})
	if err != nil {
		t.Fatalf("verify integrity --date: %v", err)
	}
	if !strings.Contains(out, "date=20260627") {
		t.Errorf("expected date in output: %s", out)
	}
}

func TestCLI_PrintJSONError(t *testing.T) {
	// math.NaN() cannot be encoded as JSON, exercising the error branch.
	resetFlags()
	flagJSON = true
	filterSource = "delegated"
	// Inject a NaN by filtering for a country that yields entries, then rely on
	// the encoder failing only when given NaN — here we instead exercise the
	// path indirectly via years --json with a normal value (already covered).
	// Direct unit test of printJSON with NaN:
	old := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe
	printJSON(math.NaN())
	_ = wPipe.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(rPipe)
	if buf.Len() != 0 {
		t.Errorf("expected no stdout on encode error, got: %s", buf.String())
	}
}

func TestCLI_StatsWithDate(t *testing.T) {
	// Cover dateOrDefault non-empty branch in stats commands.
	resetFlags()
	statsDateFlag = "20260627"
	out, err := runWithStatsServer(t, []string{"delegated"})
	if err != nil {
		t.Fatalf("delegated --date: %v", err)
	}
	if !strings.Contains(out, "delegated stats") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_TransfersJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"transfers"})
	if err != nil {
		t.Fatalf("transfers --json: %v", err)
	}
	if !strings.Contains(out, `"Transfers"`) {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestCLI_ChangesJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"changes"})
	if err != nil {
		t.Fatalf("changes --json: %v", err)
	}
	if !strings.Contains(out, `"Country"`) {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestCLI_YearsJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"years"})
	if err != nil {
		t.Fatalf("years --json: %v", err)
	}
	if !strings.Contains(out, "2001") {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestCLI_VerifyMD5WithDate(t *testing.T) {
	resetFlags()
	verifyDataType = "delegated"
	verifyDate = "20260627"
	out, err := runWithStatsServer(t, []string{"verify", "md5"})
	if err != nil {
		t.Fatalf("verify md5 --date: %v", err)
	}
	if out == "" {
		t.Error("expected md5 output")
	}
}

func TestCLI_WhoisIPJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	addr, cleanup := mockWhoisTCPServer(t, sampleWhois)
	defer cleanup()
	flagWhoisServer = addr
	out, err := runWithStatsServer(t, []string{"whois", "ip", "1.1.1.1"})
	if err != nil {
		t.Fatalf("whois ip --json: %v", err)
	}
	if !strings.Contains(out, `"Network"`) {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestCLI_ReverseDNSJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	apnic.SetLookupAddr(func(ctx context.Context, ip string) ([]string, error) {
		return []string{"one.one.one.one."}, nil
	})
	defer apnic.SetLookupAddr(nil)
	if _, err := runWithStatsServer(t, []string{"reverse-dns", "1.1.1.1"}); err != nil {
		t.Fatalf("reverse-dns --json: %v", err)
	}
}

func TestCLI_RDAPJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rdap", "ip", "1.1.1.1"})
	if err != nil {
		t.Fatalf("rdap ip --json: %v", err)
	}
	if !strings.Contains(out, "1.1.1.0") {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestCLI_RDAPASNJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rdap", "asn", "13335"})
	if err != nil {
		t.Fatalf("rdap asn --json: %v", err)
	}
	if !strings.Contains(out, "AS13335") {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestCLI_RDAPSearchJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rdap", "search", "*CLOUD*"})
	if err != nil {
		t.Fatalf("rdap search --json: %v", err)
	}
	if !strings.Contains(out, `entitySearchResults`) {
		t.Errorf("expected JSON, got: %s", out)
	}
}

func TestParseJitterRange(t *testing.T) {
	cases := []struct {
		in  string
		ok  bool
		mnS string
		mxS string
	}{
		{"200ms-800ms", true, "200ms", "800ms"},
		{"1s-2s", true, "1s", "2s"},
		{"bad", false, "", ""},
		{"200ms", false, "", ""},
		{"200ms-notdur", false, "", ""},
	}
	for _, c := range cases {
		mn, mx, ok := parseJitterRange(c.in)
		if ok != c.ok {
			t.Errorf("parseJitterRange(%q) ok=%v want %v", c.in, ok, c.ok)
			continue
		}
		if ok && (mn.String() != c.mnS || mx.String() != c.mxS) {
			t.Errorf("parseJitterRange(%q) = %s,%s want %s,%s", c.in, mn, mx, c.mnS, c.mxS)
		}
	}
}

func TestCLI_NewClientStealthFlags(t *testing.T) {
	// Exercise the stealth/jitter/rate-limit/base-url flag branches in newClient.
	resetFlags()
	flagStealth = false
	flagJitter = "50ms-100ms"
	flagRateLimit = 5.0
	flagBrowserUA = "TestBrowser/1.0"
	flagRRDPBaseURL = "https://x/rrdp"
	flagThymeBaseURL = "https://x/thyme"
	flagFTPBaseURL = "https://x/ftp"
	c := newClient()
	if c == nil {
		t.Fatal("newClient returned nil")
	}
}

func TestCLI_NewClientBadJitter(t *testing.T) {
	// A bad --jitter string is silently ignored (no panic).
	resetFlags()
	flagJitter = "not-a-range"
	c := newClient()
	if c == nil {
		t.Fatal("newClient returned nil")
	}
}

func TestCLI_RDAPHelp(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rdap", "help"})
	if err != nil {
		t.Fatalf("rdap help: %v", err)
	}
	if !strings.Contains(out, "whois.apnic.net") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_RDAPHelpJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rdap", "help"})
	if err != nil {
		t.Fatalf("rdap help --json: %v", err)
	}
	if !strings.Contains(out, "history_version_0") {
		t.Errorf("expected conformance in JSON: %s", out)
	}
}

func TestCLI_RDAPDomains(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rdap", "domains", "1.in-addr.arpa"})
	if err != nil {
		t.Fatalf("rdap domains: %v", err)
	}
	if !strings.Contains(out, "1.in-addr.arpa") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_RDAPDomainsJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rdap", "domains", "1.in-addr.arpa"})
	if err != nil {
		t.Fatalf("rdap domains --json: %v", err)
	}
	if !strings.Contains(out, "domainSearchResults") {
		t.Errorf("expected JSON: %s", out)
	}
}

func TestCLI_RExNetwork(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rex", "network"})
	if err != nil {
		t.Fatalf("rex network: %v", err)
	}
	if !strings.Contains(out, "219.142.144.241") || !strings.Contains(out, "219.142.128.0/18") {
		t.Errorf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "4847") || !strings.Contains(out, "CN") {
		t.Errorf("missing asn/economy: %s", out)
	}
}

func TestCLI_RExNetworkJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rex", "network"})
	if err != nil {
		t.Fatalf("rex network --json: %v", err)
	}
	if !strings.Contains(out, `"economy": "CN"`) {
		t.Errorf("expected JSON: %s", out)
	}
}

func TestCLI_RExResources(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rex", "resources", "ipv4"})
	if err != nil {
		t.Fatalf("rex resources: %v", err)
	}
	if !strings.Contains(out, "ERIN AVENUE LLC") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_RExResourcesNoFilter(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rex", "resources"})
	if err != nil {
		t.Fatalf("rex resources (no filter): %v", err)
	}
	if !strings.Contains(out, "23.160.212.0/24") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_RExResourcesJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rex", "resources"})
	if err != nil {
		t.Fatalf("rex resources --json: %v", err)
	}
	if !strings.Contains(out, `"items"`) {
		t.Errorf("expected JSON: %s", out)
	}
}

func TestCLI_RExHolder(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rex", "holder", "522be47e60b5c2ef81bbbab8deaa6b85", "arin"})
	if err != nil {
		t.Fatalf("rex holder: %v", err)
	}
	if !strings.Contains(out, "ERIN AVENUE LLC") {
		t.Errorf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "AS402676") {
		t.Errorf("missing asn: %s", out)
	}
	if !strings.Contains(out, "2602:f373::/40") {
		t.Errorf("missing ipv6: %s", out)
	}
}

func TestCLI_RExHolderJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rex", "holder", "522be47e60b5c2ef81bbbab8deaa6b85", "arin"})
	if err != nil {
		t.Fatalf("rex holder --json: %v", err)
	}
	if !strings.Contains(out, `"holderName": "ERIN AVENUE LLC"`) {
		t.Errorf("expected JSON: %s", out)
	}
}

func TestCLI_RExHolderMissingArgs(t *testing.T) {
	// Only one arg violates ExactArgs(2).
	resetFlags()
	_, err := runWithStatsServer(t, []string{"rex", "holder", "only-one-arg"})
	if err == nil {
		t.Error("expected error for missing holder arg")
	}
}

func TestCLI_RExCount(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rex", "count"})
	if err != nil {
		t.Fatalf("rex count: %v", err)
	}
	if !strings.Contains(out, "129665") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_RExCountJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rex", "count"})
	if err != nil {
		t.Fatalf("rex count --json: %v", err)
	}
	if !strings.Contains(out, `"count": 129665`) {
		t.Errorf("expected JSON: %s", out)
	}
}

func TestCLI_ParseByteSize(t *testing.T) {
	cases := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"1MB", 1024 * 1024, true},
		{"512KB", 512 * 1024, true},
		{"2M", 2 * 1024 * 1024, true},
		{"1024", 1024, true},
		{"1GB", 1024 * 1024 * 1024, true},
		{"4K", 4 * 1024, true},               // single-letter K suffix
		{"3G", 3 * 1024 * 1024 * 1024, true}, // single-letter G suffix
		{"8k", 8 * 1024, true},               // lowercase k
		{"2g", 2 * 1024 * 1024 * 1024, true}, // lowercase g
		{"2m", 2 * 1024 * 1024, true},        // lowercase m
		{"512kb", 512 * 1024, true},          // lowercase kb
		{"1mb", 1024 * 1024, true},           // lowercase mb
		{"1gb", 1024 * 1024 * 1024, true},    // lowercase gb
		{"", 0, false},
		{"abc", 0, false},
		{"-1MB", 0, false},
		{"12x", 0, false}, // unknown suffix + non-numeric
	}
	for _, tc := range cases {
		got, ok := parseByteSize(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Errorf("parseByteSize(%q) = (%d, %v), want (%d, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// ---------------------------------------------------------------------------
// Fetch-error branch coverage (RunE "if err != nil { return err }")
// ---------------------------------------------------------------------------

// TestCLI_StatsFetchErrors drives every stats command against a 500 server to
// cover each RunE fetch-error return branch (both the JSON-path and the
// plain-path fetch errors).
func TestCLI_StatsFetchErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		json bool
	}{
		{"delegated", []string{"delegated"}, false},
		{"delegated-json", []string{"delegated"}, true},
		{"extended", []string{"extended"}, false},
		{"extended-json", []string{"extended"}, true},
		{"assigned", []string{"assigned"}, false},
		{"assigned-json", []string{"assigned"}, true},
		{"ipv6-assigned", []string{"ipv6-assigned"}, false},
		{"ipv6-assigned-json", []string{"ipv6-assigned"}, true},
		{"legacy", []string{"legacy"}, false},
		{"legacy-json", []string{"legacy"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resetFlags()
			flagJSON = c.json
			if err := runWithErrorServer(t, c.args); err == nil {
				t.Errorf("expected fetch error for %s", c.name)
			}
		})
	}
}

// TestCLI_RDAPFetchErrors covers the fetch-error branch of each RDAP subcommand.
func TestCLI_RDAPFetchErrors(t *testing.T) {
	cases := [][]string{
		{"rdap", "ip", "1.1.1.1"},
		{"rdap", "cidr", "1.1.1.0/24"},
		{"rdap", "asn", "13335"},
		{"rdap", "domain", "1.0.0.1.in-addr.arpa"},
		{"rdap", "entity", "AIC3-AP"},
		{"rdap", "search", "*CLOUD*"},
		{"rdap", "help"},
		{"rdap", "domains", "1.in-addr.arpa"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			resetFlags()
			if err := runWithErrorServer(t, args); err == nil {
				t.Errorf("expected fetch error for %v", args)
			}
		})
	}
}

// TestCLI_BGPFetchErrors covers the fetch-error branch of each BGP subcommand.
func TestCLI_BGPFetchErrors(t *testing.T) {
	for _, args := range [][]string{
		{"bgp", "summary"},
		{"bgp", "raw-table"},
		{"bgp", "asn-map"},
	} {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			resetFlags()
			if err := runWithErrorServer(t, args); err == nil {
				t.Errorf("expected fetch error for %v", args)
			}
		})
	}
}

// TestCLI_RPKIFetchErrors covers the fetch-error branches of the RPKI commands.
// The no-arg 'rpki snapshot' path triggers resolveSnapshotURI's notification
// fetch error; the explicit-uri path triggers FetchRRDPSnapshot's fetch error.
func TestCLI_RPKIFetchErrors(t *testing.T) {
	t.Run("notification", func(t *testing.T) {
		resetFlags()
		if err := runWithErrorServer(t, []string{"rpki", "notification"}); err == nil {
			t.Error("expected fetch error for rpki notification")
		}
	})
	t.Run("snapshot-noarg", func(t *testing.T) {
		resetFlags()
		if err := runWithErrorServer(t, []string{"rpki", "snapshot"}); err == nil {
			t.Error("expected fetch error for rpki snapshot (no arg)")
		}
	})
	t.Run("snapshot-uri", func(t *testing.T) {
		resetFlags()
		// A relative path resolves against the 500 RRDP base URL.
		if err := runWithErrorServer(t, []string{"rpki", "snapshot", "snapshot.xml"}); err == nil {
			t.Error("expected fetch error for rpki snapshot (uri)")
		}
	})
}

// TestCLI_VerifyFetchErrors covers the fetch-error branch of each verify
// subcommand. md5/asc/integrity require --type to pass requireDataType first.
func TestCLI_VerifyFetchErrors(t *testing.T) {
	t.Run("md5", func(t *testing.T) {
		resetFlags()
		verifyDataType = "delegated"
		if err := runWithErrorServer(t, []string{"verify", "md5"}); err == nil {
			t.Error("expected fetch error for verify md5")
		}
	})
	t.Run("asc", func(t *testing.T) {
		resetFlags()
		verifyDataType = "delegated"
		if err := runWithErrorServer(t, []string{"verify", "asc"}); err == nil {
			t.Error("expected fetch error for verify asc")
		}
	})
	t.Run("pubkey", func(t *testing.T) {
		resetFlags()
		if err := runWithErrorServer(t, []string{"verify", "pubkey"}); err == nil {
			t.Error("expected fetch error for verify pubkey")
		}
	})
	t.Run("integrity", func(t *testing.T) {
		resetFlags()
		verifyDataType = "delegated"
		if err := runWithErrorServer(t, []string{"verify", "integrity"}); err == nil {
			t.Error("expected fetch error for verify integrity")
		}
	})
}

// TestCLI_TransfersChangesFetchErrors covers the fetch-error branch of the
// transfers, changes and transfers-all commands. For 'changes' the --date
// branch is also exercised against the 500 server.
func TestCLI_TransfersChangesFetchErrors(t *testing.T) {
	t.Run("transfers", func(t *testing.T) {
		resetFlags()
		if err := runWithErrorServer(t, []string{"transfers"}); err == nil {
			t.Error("expected fetch error for transfers")
		}
	})
	t.Run("changes", func(t *testing.T) {
		resetFlags()
		if err := runWithErrorServer(t, []string{"changes"}); err == nil {
			t.Error("expected fetch error for changes")
		}
	})
	t.Run("changes-bydate", func(t *testing.T) {
		resetFlags()
		changesDate = "20200615"
		if err := runWithErrorServer(t, []string{"changes"}); err == nil {
			t.Error("expected fetch error for changes --date")
		}
	})
	t.Run("transfers-all", func(t *testing.T) {
		resetFlags()
		if err := runWithErrorServer(t, []string{"transfers-all"}); err == nil {
			t.Error("expected fetch error for transfers-all")
		}
	})
	t.Run("transfers-byyear", func(t *testing.T) {
		resetFlags()
		transfersYear = 2026
		if err := runWithErrorServer(t, []string{"transfers"}); err == nil {
			t.Error("expected fetch error for transfers --year")
		}
	})
}

// TestCLI_RexFetchErrors covers the fetch-error branch of each REx subcommand.
func TestCLI_RexFetchErrors(t *testing.T) {
	for _, args := range [][]string{
		{"rex", "network"},
		{"rex", "resources"},
		{"rex", "holder", "522be47e60b5c2ef81bbbab8deaa6b85", "arin"},
		{"rex", "count"},
	} {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			resetFlags()
			if err := runWithErrorServer(t, args); err == nil {
				t.Errorf("expected fetch error for %v", args)
			}
		})
	}
}

// TestCLI_IRRSerialFetchError covers the FetchIRRCurrentSerial fetch-error branch.
func TestCLI_IRRSerialFetchError(t *testing.T) {
	resetFlags()
	if err := runWithErrorServer(t, []string{"irr", "serial"}); err == nil {
		t.Error("expected fetch error for irr serial")
	}
}

// TestCLI_FilterFetchErrors covers the fetch-error branch of the filter
// command for both delegated and extended sources.
func TestCLI_FilterFetchErrors(t *testing.T) {
	t.Run("delegated", func(t *testing.T) {
		resetFlags()
		filterSource = "delegated"
		if err := runWithErrorServer(t, []string{"filter"}); err == nil {
			t.Error("expected fetch error for filter delegated")
		}
	})
	t.Run("extended", func(t *testing.T) {
		resetFlags()
		filterSource = "extended"
		if err := runWithErrorServer(t, []string{"filter"}); err == nil {
			t.Error("expected fetch error for filter extended")
		}
	})
}

// TestCLI_TelemetryFetchError covers the telemetry fetch-error branch.
func TestCLI_TelemetryFetchError(t *testing.T) {
	resetFlags()
	if err := runWithErrorServer(t, []string{"stats-telemetry"}); err == nil {
		t.Error("expected fetch error for stats-telemetry")
	}
}

// TestCLI_WhoisFetchErrors covers the fetch-error branch of the whois
// subcommands by pointing flagWhoisServer at a dial-failing address (TCP port
// 1 on loopback refuses connections). A valid ASN is used so the ParseInt
// validation passes and the dial failure triggers the error branch.
func TestCLI_WhoisFetchErrors(t *testing.T) {
	for _, args := range [][]string{
		{"whois", "ip", "1.1.1.1"},
		{"whois", "asn", "13335"},
		{"whois", "raw", "1.1.1.1"},
	} {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			resetFlags()
			flagWhoisServer = "127.0.0.1:1"
			if _, err := runWithStatsServer(t, args); err == nil {
				t.Errorf("expected dial error for %v", args)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// JSON output branch coverage (RunE "if flagJSON { printJSON(...); return nil }")
// ---------------------------------------------------------------------------

func TestCLI_ExtendedJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"extended"})
	if err != nil {
		t.Fatalf("extended --json: %v", err)
	}
	if !strings.Contains(out, `"Entries"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_AssignedJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"assigned"})
	if err != nil {
		t.Fatalf("assigned --json: %v", err)
	}
	if !strings.Contains(out, `"Entries"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_IPv6AssignedJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"ipv6-assigned"})
	if err != nil {
		t.Fatalf("ipv6-assigned --json: %v", err)
	}
	if !strings.Contains(out, `"Entries"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_LegacyJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"legacy"})
	if err != nil {
		t.Fatalf("legacy --json: %v", err)
	}
	if !strings.Contains(out, `"Entries"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_RDAPCIDRJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rdap", "cidr", "1.1.1.0/24"})
	if err != nil {
		t.Fatalf("rdap cidr --json: %v", err)
	}
	if !strings.Contains(out, "1.1.1.0") {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_RDAPDomainJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rdap", "domain", "1.0.0.1.in-addr.arpa"})
	if err != nil {
		t.Fatalf("rdap domain --json: %v", err)
	}
	if !strings.Contains(out, "1.0.0.1.in-addr.arpa") {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_RDAPEntityJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rdap", "entity", "AIC3-AP"})
	if err != nil {
		t.Fatalf("rdap entity --json: %v", err)
	}
	if !strings.Contains(out, "AIC3-AP") {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_BGPSummaryJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"bgp", "summary"})
	if err != nil {
		t.Fatalf("bgp summary --json: %v", err)
	}
	if !strings.Contains(out, `"Entries"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_BGPRawTableJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"bgp", "raw-table"})
	if err != nil {
		t.Fatalf("bgp raw-table --json: %v", err)
	}
	if !strings.Contains(out, `"Routes"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_BGPASNMapJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"bgp", "asn-map"})
	if err != nil {
		t.Fatalf("bgp asn-map --json: %v", err)
	}
	if !strings.Contains(out, `"ASNs"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_BGPPerPrefixLengthJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"bgp", "per-prefix-length"})
	if err != nil {
		t.Fatalf("bgp per-prefix-length --json: %v", err)
	}
	if !strings.Contains(out, `"Counts"`) {
		t.Errorf("expected JSON output with Counts field, got: %s", out)
	}
}

func TestCLI_BGPUsedAutnumsJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"bgp", "used-autnums"})
	if err != nil {
		t.Fatalf("bgp used-autnums --json: %v", err)
	}
	if !strings.Contains(out, `"Autnums"`) {
		t.Errorf("expected JSON output with Autnums field, got: %s", out)
	}
}

func TestCLI_BGPSparPrefixesJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"bgp", "spar-prefixes"})
	if err != nil {
		t.Fatalf("bgp spar-prefixes --json: %v", err)
	}
	if !strings.Contains(out, `"Prefixes"`) {
		t.Errorf("expected JSON output with Prefixes field, got: %s", out)
	}
}

func TestCLI_BGPSinglePfxJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"bgp", "single-pfx"})
	if err != nil {
		t.Fatalf("bgp single-pfx --json: %v", err)
	}
	if !strings.Contains(out, `"Counts"`) {
		t.Errorf("expected JSON output with Counts field, got: %s", out)
	}
	if !strings.Contains(out, `"RIR"`) {
		t.Errorf("expected JSON output with RIR field, got: %s", out)
	}
}

func TestCLI_RPKINotificationJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rpki", "notification"})
	if err != nil {
		t.Fatalf("rpki notification --json: %v", err)
	}
	if !strings.Contains(out, `"SessionID"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_RPKISnapshotJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"rpki", "snapshot", "snapshot.xml"})
	if err != nil {
		t.Fatalf("rpki snapshot --json: %v", err)
	}
	if !strings.Contains(out, `"SessionID"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_IRRJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"irr", "inetnum"})
	if err != nil {
		t.Fatalf("irr inetnum --json: %v", err)
	}
	if !strings.Contains(out, `"Objects"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// Truncation (>N) branch coverage
// ---------------------------------------------------------------------------

// TestCLI_BGPRawTableTruncation exercises the >50 routes truncation path
// (limit cap + "... (N more)" line) in 'bgp raw-table'.
func TestCLI_BGPRawTableTruncation(t *testing.T) {
	resetFlags()
	srv := newLargeDatasetServer()
	defer srv.Close()
	out, err := runWithServer(t, srv, []string{"bgp", "raw-table"})
	if err != nil {
		t.Fatalf("bgp raw-table truncation: %v", err)
	}
	if !strings.Contains(out, "more)") {
		t.Errorf("expected truncation marker in output: %s", out)
	}
}

// TestCLI_BGPUsedAutnumsTruncation exercises the >50 ASNs truncation path
// (limit cap + "... (N more)" line) in 'bgp used-autnums'.
func TestCLI_BGPUsedAutnumsTruncation(t *testing.T) {
	resetFlags()
	srv := newLargeDatasetServer()
	defer srv.Close()
	out, err := runWithServer(t, srv, []string{"bgp", "used-autnums"})
	if err != nil {
		t.Fatalf("bgp used-autnums truncation: %v", err)
	}
	if !strings.Contains(out, "more)") {
		t.Errorf("expected truncation marker in output: %s", out)
	}
}

// TestCLI_BGPBadPrefixesTruncation exercises the >50 entries truncation path
// (limit cap + "... (N more)" line) in 'bgp bad-prefixes'.
func TestCLI_BGPBadPrefixesTruncation(t *testing.T) {
	resetFlags()
	srv := newLargeDatasetServer()
	defer srv.Close()
	out, err := runWithServer(t, srv, []string{"bgp", "bad-prefixes"})
	if err != nil {
		t.Fatalf("bgp bad-prefixes truncation: %v", err)
	}
	if !strings.Contains(out, "more)") {
		t.Errorf("expected truncation marker in output: %s", out)
	}
}

// TestCLI_RPKINotificationTruncation exercises the >20 deltas truncation path
// (limit cap + "... (N more deltas)" line) in 'rpki notification'.
func TestCLI_RPKINotificationTruncation(t *testing.T) {
	resetFlags()
	srv := newLargeDatasetServer()
	defer srv.Close()
	out, err := runWithServer(t, srv, []string{"rpki", "notification"})
	if err != nil {
		t.Fatalf("rpki notification truncation: %v", err)
	}
	if !strings.Contains(out, "more deltas)") {
		t.Errorf("expected truncation marker in output: %s", out)
	}
}

// TestCLI_IRRTruncation exercises the >50 objects truncation path (limit cap +
// "... (N more)" line) in 'irr inetnum'.
func TestCLI_IRRTruncation(t *testing.T) {
	resetFlags()
	srv := newLargeDatasetServer()
	defer srv.Close()
	out, err := runWithServer(t, srv, []string{"irr", "inetnum"})
	if err != nil {
		t.Fatalf("irr inetnum truncation: %v", err)
	}
	if !strings.Contains(out, "more)") {
		t.Errorf("expected truncation marker in output: %s", out)
	}
}

// ---------------------------------------------------------------------------
// Misc branch coverage
// ---------------------------------------------------------------------------

// TestCLI_RDAPSearchEmptyField covers the `if field == ""` default branch of
// 'rdap search' by passing an explicit empty --field value.
func TestCLI_RDAPSearchEmptyField(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"rdap", "search", "*CLOUD*", "--field", ""})
	if err != nil {
		t.Fatalf("rdap search --field empty: %v", err)
	}
	if !strings.Contains(out, "results") {
		t.Errorf("unexpected output: %s", out)
	}
}

// TestCLI_ChangesByDateSuccess covers the FetchChangesByDate branch (changesDate
// != "") of the 'changes' command using the success mock.
func TestCLI_ChangesByDateSuccess(t *testing.T) {
	resetFlags()
	changesDate = "20200615"
	out, err := runWithStatsServer(t, []string{"changes"})
	if err != nil {
		t.Fatalf("changes --date: %v", err)
	}
	if !strings.Contains(out, "changes:") {
		t.Errorf("unexpected output: %s", out)
	}
}

// TestCLI_FilterExtendedTypeStatus covers the extended-source ByType and
// ByStatus filter branches (filterType and filterStatus both set).
func TestCLI_FilterExtendedTypeStatus(t *testing.T) {
	resetFlags()
	filterSource = "extended"
	filterType = "ipv4"
	filterStatus = "assigned"
	out, err := runWithStatsServer(t, []string{"filter"})
	if err != nil {
		t.Fatalf("filter extended type+status: %v", err)
	}
	if !strings.Contains(out, "after filter") {
		t.Errorf("unexpected output: %s", out)
	}
}

// TestCLI_WhoisASNJSON covers the JSON output branch of 'whois asn'.
func TestCLI_WhoisASNJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	addr, cleanup := mockWhoisTCPServer(t, sampleWhois)
	defer cleanup()
	flagWhoisServer = addr
	out, err := runWithStatsServer(t, []string{"whois", "asn", "AS13335"})
	if err != nil {
		t.Fatalf("whois asn --json: %v", err)
	}
	if !strings.Contains(out, `"Network"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCLI_BGPBadPrefixes(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"bgp", "bad-prefixes"})
	if err != nil {
		t.Fatalf("bgp bad-prefixes: %v", err)
	}
	if !strings.Contains(out, "bgp bad-prefixes") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_BGPBadPrefixesJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	if _, err := runWithStatsServer(t, []string{"bgp", "bad-prefixes"}); err != nil {
		t.Fatalf("bgp bad-prefixes --json: %v", err)
	}
}

func TestCLI_BGPPerPrefixLength(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"bgp", "per-prefix-length"})
	if err != nil {
		t.Fatalf("bgp per-prefix-length: %v", err)
	}
	if !strings.Contains(out, "/8") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_BGPUsedAutnums(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"bgp", "used-autnums"})
	if err != nil {
		t.Fatalf("bgp used-autnums: %v", err)
	}
	if !strings.Contains(out, "LVLT-1") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_BGPSparPrefixes(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"bgp", "spar-prefixes"})
	if err != nil {
		t.Fatalf("bgp spar-prefixes: %v", err)
	}
	if !strings.Contains(out, "192.88.99.0/24") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_BGPSinglePfx(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"bgp", "single-pfx"})
	if err != nil {
		t.Fatalf("bgp single-pfx: %v", err)
	}
	if !strings.Contains(out, "27539") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_BGPSourceAU(t *testing.T) {
	// --bgp-source au should be reflected in the output's source label.
	resetFlags()
	flagBGPSource = "au"
	out, err := runWithStatsServer(t, []string{"bgp", "single-pfx"})
	if err != nil {
		t.Fatalf("bgp single-pfx --source au: %v", err)
	}
	if !strings.Contains(out, "source=au") {
		t.Errorf("expected source=au in output: %s", out)
	}
}

func TestCLI_BGPAdditionalFetchErrors(t *testing.T) {
	resetFlags()
	for _, args := range [][]string{
		{"bgp", "bad-prefixes"},
		{"bgp", "per-prefix-length"},
		{"bgp", "used-autnums"},
		{"bgp", "spar-prefixes"},
		{"bgp", "single-pfx"},
	} {
		if err := runWithErrorServer(t, args); err == nil {
			t.Errorf("expected error for %v", args)
		}
	}
}
