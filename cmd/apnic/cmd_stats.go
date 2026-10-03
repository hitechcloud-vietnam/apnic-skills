package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// printJSON marshals v as indented JSON to stdout.
func printJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "json encode error:", err)
	}
}

var statsDateFlag string

func init() {
	for _, cmd := range []*cobra.Command{
		delegatedCmd, extendedCmd, assignedCmd, ipv6AssignedCmd, legacyCmd,
	} {
		cmd.Flags().StringVar(&statsDateFlag, "date", "", "data date in YYYYMMDD format (default: latest)")
	}
	rootCmd.AddCommand(delegatedCmd)
	rootCmd.AddCommand(extendedCmd)
	rootCmd.AddCommand(assignedCmd)
	rootCmd.AddCommand(ipv6AssignedCmd)
	rootCmd.AddCommand(legacyCmd)
}

var delegatedCmd = &cobra.Command{
	Use:   "delegated",
	Short: "Fetch standard delegated stats (IP/ASN allocations)",
	Long: `Fetch the APNIC standard delegated stats file.

This file lists every IP/ASN allocation and assignment recorded by APNIC, in the
RIR statistics exchange format (registry|cc|type|start|value|date|status).

Use --date YYYYMMDD to fetch a historical snapshot; omit for the latest file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		if flagJSON {
			result, err := client.FetchDelegatedResult(ctx, statsDateFlag)
			if err != nil {
				return err
			}
			printJSON(result)
			return nil
		}
		entries, err := client.FetchDelegatedEntriesByDate(ctx, statsDateFlag)
		if err != nil {
			return err
		}
		fmt.Printf("# delegated stats: %d entries (date=%s)\n", len(entries), dateOrDefault(statsDateFlag))
		for _, e := range entries {
			fmt.Printf("%s\t%s\t%s\t%d\t%s\t%s\n", e.Country, e.Type, e.Start, e.Value, e.Status, e.Date.Format("20060102"))
		}
		return nil
	},
}

var extendedCmd = &cobra.Command{
	Use:   "extended",
	Short: "Fetch extended delegated stats (includes organization opaque-IDs)",
	Long: `Fetch the APNIC extended delegated stats file.

Like the standard delegated file, but each record carries an opaque-id that
identifies the resource holder organization. Useful for per-organization
aggregation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		result, err := client.FetchExtendedResult(ctx, statsDateFlag)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(result)
			return nil
		}
		fmt.Printf("# extended stats: %d entries (date=%s)\n", len(result.Entries), dateOrDefault(statsDateFlag))
		for _, e := range result.Entries {
			fmt.Printf("%s\t%s\t%s\t%d\t%s\t%s\n", e.Country, e.Type, e.Start, e.Value, e.Status, e.OpaqueID)
		}
		return nil
	},
}

var assignedCmd = &cobra.Command{
	Use:   "assigned",
	Short: "Fetch aggregated assignment stats by prefix size",
	Long: `Fetch the APNIC assigned stats file.

This file aggregates assignment counts by prefix size per country, showing how
many assignments of each prefix length exist (e.g. how many /24s assigned).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		result, err := client.FetchAssignedResult(ctx, statsDateFlag)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(result)
			return nil
		}
		fmt.Printf("# assigned stats: %d entries (date=%s)\n", len(result.Entries), dateOrDefault(statsDateFlag))
		for _, e := range result.Entries {
			fmt.Printf("%s\t%s\t%s\t%d\n", e.Country, e.Type, e.Prefix, e.Count)
		}
		return nil
	},
}

var ipv6AssignedCmd = &cobra.Command{
	Use:   "ipv6-assigned",
	Short: "Fetch per-prefix IPv6 assignment records",
	Long: `Fetch the APNIC delegated-apnic-ipv6-assigned stats file.

Unlike the aggregated "assigned" file, this lists each individual IPv6
assignment as a separate record (registry|cc|ipv6|start|prefix|date), with no
status or extension columns.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		result, err := client.FetchIPv6AssignedResult(ctx, statsDateFlag)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(result)
			return nil
		}
		fmt.Printf("# ipv6-assigned stats: %d entries (date=%s)\n", len(result.Entries), dateOrDefault(statsDateFlag))
		for _, e := range result.Entries {
			fmt.Printf("%s\t%s\t%d\n", e.Country, e.Start, e.Value)
		}
		return nil
	},
}

var legacyCmd = &cobra.Command{
	Use:   "legacy",
	Short: "Fetch historical legacy resource records",
	Long: `Fetch the APNIC legacy stats file.

Legacy resources are address space transferred to APNIC from other registries
before the current RIR statistics framework was established.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		result, err := client.FetchLegacyResult(ctx, statsDateFlag)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(result)
			return nil
		}
		fmt.Printf("# legacy stats: %d entries (date=%s)\n", len(result.Entries), dateOrDefault(statsDateFlag))
		for _, e := range result.Entries {
			fmt.Printf("%s\t%s\t%s\t%d\t%s\n", e.Country, e.Type, e.Start, e.Value, e.Status)
		}
		return nil
	},
}

func dateOrDefault(d string) string {
	if d == "" {
		return "latest"
	}
	return d
}
