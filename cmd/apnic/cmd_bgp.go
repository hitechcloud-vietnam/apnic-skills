package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	bgpCmd.AddCommand(bgpSummaryCmd)
	bgpCmd.AddCommand(bgpRawTableCmd)
	bgpCmd.AddCommand(bgpASNMapCmd)
	bgpCmd.AddCommand(bgpBadPrefixesCmd)
	bgpCmd.AddCommand(bgpPerPrefixLengthCmd)
	bgpCmd.AddCommand(bgpUsedAutnumsCmd)
	bgpCmd.AddCommand(bgpSparPrefixesCmd)
	bgpCmd.AddCommand(bgpSinglePfxCmd)
	rootCmd.AddCommand(bgpCmd)
}

var bgpCmd = &cobra.Command{
	Use:   "bgp",
	Short: "Fetch APNIC thyme BGP routing table analysis",
	Long: `Fetch APNIC thyme BGP routing table analysis.

APNIC thyme (https://thyme.apnic.net) publishes a periodic snapshot of the
Internet BGP routing table. Two raw files are available under /current/:
  - data-summary : colon-separated key/value metrics (entries examined, AS
    counts, ROA coverage, address-space % announced, ...).
  - data-raw-table: every announced route as "prefix\tASN" lines.

Use 'bgp summary' or 'bgp raw-table' to fetch them; 'bgp asn-map' aggregates
the raw table by origin ASN locally (no extra request).`,
}

var bgpSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Fetch the thyme data-summary metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		s, err := client.FetchBGPSummary(context.Background())
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(s)
			return nil
		}
		fmt.Printf("# bgp summary: %d metrics\n", len(s.Entries))
		for _, e := range s.Entries {
			fmt.Printf("%s\t%s\n", e.Key, e.Value)
		}
		return nil
	},
}

var bgpRawTableCmd = &cobra.Command{
	Use:   "raw-table",
	Short: "Fetch the thyme data-raw-table (every announced route)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		rt, err := client.FetchBGPRawTable(context.Background())
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(rt)
			return nil
		}
		fmt.Printf("# bgp raw-table: %d routes\n", len(rt.Routes))
		limit := len(rt.Routes)
		if limit > 50 {
			limit = 50
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("%s\t%s\n", rt.Routes[i].Prefix, rt.Routes[i].ASN)
		}
		if len(rt.Routes) > limit {
			fmt.Printf("... (%d more)\n", len(rt.Routes)-limit)
		}
		return nil
	},
}

var bgpASNMapCmd = &cobra.Command{
	Use:   "asn-map",
	Short: "Aggregate the raw BGP table by origin ASN (derived locally)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		m, err := client.FetchBGPASNMap(context.Background())
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(m)
			return nil
		}
		fmt.Printf("# bgp asn-map: %d unique origin ASNs\n", len(m.ASNs))
		return nil
	},
}

var bgpBadPrefixesCmd = &cobra.Command{
	Use:   "bad-prefixes",
	Short: "Fetch prefixes longer than /24 and their origin AS (route-leak candidates)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		r, err := client.FetchBGPBadPrefixes(context.Background(), flagBGPSource)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(r)
			return nil
		}
		fmt.Printf("# bgp bad-prefixes: %d entries (source=%s)\n", len(r.Prefixes), sourceLabel(flagBGPSource))
		limit := len(r.Prefixes)
		if limit > 50 {
			limit = 50
		}
		for i := 0; i < limit; i++ {
			p := r.Prefixes[i]
			fmt.Printf("%s\t%s\n", p.OriginAS, p.Address)
		}
		if len(r.Prefixes) > limit {
			fmt.Printf("... (%d more)\n", len(r.Prefixes)-limit)
		}
		return nil
	},
}

var bgpPerPrefixLengthCmd = &cobra.Command{
	Use:   "per-prefix-length",
	Short: "Count announced prefixes per prefix length",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		r, err := client.FetchBGPPerPrefixLength(context.Background(), flagBGPSource)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(r)
			return nil
		}
		fmt.Printf("# bgp per-prefix-length: %d entries (source=%s)\n", len(r.Counts), sourceLabel(flagBGPSource))
		for _, c := range r.Counts {
			fmt.Printf("/%d\t%d\n", c.Length, c.Count)
		}
		return nil
	},
}

var bgpUsedAutnumsCmd = &cobra.Command{
	Use:   "used-autnums",
	Short: "List every in-use ASN with registered name and country",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		r, err := client.FetchBGPUsedAutnums(context.Background(), flagBGPSource)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(r)
			return nil
		}
		fmt.Printf("# bgp used-autnums: %d ASNs (source=%s)\n", len(r.Autnums), sourceLabel(flagBGPSource))
		limit := len(r.Autnums)
		if limit > 50 {
			limit = 50
		}
		for i := 0; i < limit; i++ {
			a := r.Autnums[i]
			fmt.Printf("%s\t%s\t%s\n", a.ASN, a.Country, a.FullName)
		}
		if len(r.Autnums) > limit {
			fmt.Printf("... (%d more)\n", len(r.Autnums)-limit)
		}
		return nil
	},
}

var bgpSparPrefixesCmd = &cobra.Command{
	Use:   "spar-prefixes",
	Short: "Prefixes from the Special Purpose Address Registry (RFC 6890)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		r, err := client.FetchBGPSparPrefixes(context.Background(), flagBGPSource)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(r)
			return nil
		}
		fmt.Printf("# bgp spar-prefixes: %d entries (source=%s)\n", len(r.Prefixes), sourceLabel(flagBGPSource))
		for _, p := range r.Prefixes {
			fmt.Printf("%s\t%s\t%s\n", p.Prefix, p.OriginAS, p.Description)
		}
		return nil
	},
}

var bgpSinglePfxCmd = &cobra.Command{
	Use:   "single-pfx",
	Short: "Tally ASNs announcing fewer than 20 prefixes, by RIR",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		r, err := client.FetchBGPSinglePfx(context.Background(), flagBGPSource)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(r)
			return nil
		}
		fmt.Printf("# bgp single-pfx: %d rows (source=%s)\n", len(r.Counts), sourceLabel(flagBGPSource))
		for _, c := range r.Counts {
			fmt.Printf("%d\t%d\t%s\n", c.PrefixCount, c.ASNCount, c.RIR)
		}
		return nil
	},
}

// sourceLabel returns the thyme source for display, defaulting to "current".
func sourceLabel(s string) string {
	if s == "" {
		return "current"
	}
	return s
}
