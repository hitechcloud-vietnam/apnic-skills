package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rexCmd.AddCommand(rexNetworkCmd)
	rexCmd.AddCommand(rexResourcesCmd)
	rexCmd.AddCommand(rexHolderCmd)
	rexCmd.AddCommand(rexCountCmd)
	rootCmd.AddCommand(rexCmd)
}

// rexCmd exposes the APNIC REx cross-RIR resource registry REST API
// (api.rex.apnic.net/v1/*). REx provides a unified, holder-aggregated view of
// delegated resources across all five RIRs — capabilities that go beyond the
// per-RIR stats and RDAP endpoints elsewhere in this CLI.
var rexCmd = &cobra.Command{
	Use:   "rex",
	Short: "Query the APNIC REx cross-RIR resource registry",
	Long: `Query the APNIC REx cross-RIR resource registry (api.rex.apnic.net).

REx (Resource EXplorer) is a public REST API that aggregates delegated
resources across all five RIRs (APNIC, ARIN, RIPE, LACNIC, AFRINIC) and
attributes them to resource-holder organisations via opaque identifiers. It
offers capabilities the per-RIR stats/RDAP services cannot:

  - user-network : locate the caller's own network (covering prefix, origin
    ASN, economy) from the source IP — no parameters required.
  - resources    : a recent window of cross-RIR delegated prefixes/ASNs with
    holder attribution, optionally filtered by type (ipv4|ipv6|asn). Returns
    the most recently delegated resources, not the full history.
  - holder       : every ASN and prefix held by one organisation, given its
    opaqueId and responsible RIR (one of afrinic, apnic, arin, lacnic,
    ripencc — note the RIPE NCC code is "ripencc", not "ripe").
  - count        : the total number of distinct resource holders across all
    RIRs.

All endpoints return JSON and require no authentication.`,
}

var rexNetworkCmd = &cobra.Command{
	Use:   "network",
	Short: "Locate the caller's own network (covering prefix, origin ASN, economy)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		res, err := client.FetchRExUserNetwork(context.Background())
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(res)
			return nil
		}
		fmt.Printf("# rex user-network\n")
		fmt.Printf("ip\t%s\n", res.IP)
		fmt.Printf("prefix\t%s\n", res.Prefix)
		fmt.Printf("asn\t%d\n", res.ASN)
		fmt.Printf("economy\t%s\n", res.Economy)
		return nil
	},
}

var rexResourcesCmd = &cobra.Command{
	Use:   "resources [type]",
	Short: "List recently delegated cross-RIR resources with holder attribution",
	Long: `List recently delegated cross-RIR resources with holder attribution.

The optional type argument filters by resource kind: ipv4, ipv6, or asn.
Omit it to list all kinds. REx returns a bounded recent window of the most
recently delegated resources (newest-first), not the full historical list.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		resType := ""
		if len(args) == 1 {
			resType = args[0]
		}
		res, err := client.FetchRExResources(context.Background(), resType)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(res)
			return nil
		}
		fmt.Printf("# rex resources: %d items\n", len(res.Items))
		for _, it := range res.Items {
			fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n", it.Type, it.Resource, it.HolderName, it.RIR, it.CC, it.OpaqueID)
		}
		return nil
	},
}

var rexHolderCmd = &cobra.Command{
	Use:   "holder <opaqueId> <rir>",
	Short: "Aggregate every ASN and prefix held by one organisation",
	Long: `Aggregate every ASN and prefix held by one organisation.

Given a holder's opaque identifier and the responsible RIR, returns the
holder's full ASN and prefix inventory with derived size metrics. The rir
argument must be one of: afrinic, apnic, arin, lacnic, ripencc (note the RIPE
NCC code is "ripencc", not "ripe"). The opaqueId is obtainable from 'rex
resources' or from the extended delegated stats.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		res, err := client.FetchRExHolder(context.Background(), args[0], args[1])
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(res)
			return nil
		}
		fmt.Printf("# rex holder: %s (%s)\n", res.HolderName, res.Registry)
		fmt.Printf("asns\t%d\n", res.ASNsCount)
		for _, a := range res.ASNs {
			fmt.Printf("asn\t%s\n", a)
		}
		fmt.Printf("ipv4\t%d (/24 units: %g)\n", len(res.IPv4), res.IPv4_24Count)
		for _, p := range res.IPv4 {
			fmt.Printf("ipv4\t%s\n", p)
		}
		fmt.Printf("ipv6\t%d (/48 units: %g)\n", len(res.IPv6), res.IPv6_48Count)
		for _, p := range res.IPv6 {
			fmt.Printf("ipv6\t%s\n", p)
		}
		return nil
	},
}

var rexCountCmd = &cobra.Command{
	Use:   "count",
	Short: "Total distinct resource-holder organisations across all RIRs",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		res, err := client.FetchRExHoldersUniqueCount(context.Background())
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(res)
			return nil
		}
		fmt.Printf("# rex holders unique-count\n")
		fmt.Printf("count\t%d\n", res.Count)
		return nil
	},
}
