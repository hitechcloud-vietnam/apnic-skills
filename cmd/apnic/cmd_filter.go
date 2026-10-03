package main

import (
	"context"
	"fmt"

	apnic "github.com/hitechcloud-vietnam/apnic-skills"
	"github.com/spf13/cobra"
)

var (
	filterSource   string // delegated | extended
	filterCountry  string
	filterType     string
	filterStatus   string
	filterOpaqueID string
)

func init() {
	filterCmd.Flags().StringVar(&filterSource, "source", "delegated", "data source: delegated or extended")
	filterCmd.Flags().StringVar(&filterCountry, "country", "", "ISO 3166 country code (e.g. CN)")
	filterCmd.Flags().StringVar(&filterType, "type", "", "resource type: ipv4, ipv6, asn")
	filterCmd.Flags().StringVar(&filterStatus, "status", "", "status: allocated, assigned, reserved, available")
	filterCmd.Flags().StringVar(&filterOpaqueID, "opaque-id", "", "opaque-id / org identifier (extended only)")
	rootCmd.AddCommand(filterCmd)
}

var filterCmd = &cobra.Command{
	Use:   "filter",
	Short: "Fetch and chain-filter delegated/extended stats",
	Long: `Fetch the latest delegated (or extended) stats and apply chain filters.

Filters are combined with AND semantics. Example:

  apnic filter --source delegated --country CN --type ipv4 --status allocated
  apnic filter --source extended --country JP --opaque-id A92E1062`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()

		switch filterSource {
		case "delegated":
			entries, err := client.GetDelegatedEntries(ctx)
			if err != nil {
				return err
			}
			f := apnic.NewFilter(entries)
			if filterCountry != "" {
				f = f.ByCountry(filterCountry)
			}
			if filterType != "" {
				f = f.ByType(filterType)
			}
			if filterStatus != "" {
				f = f.ByStatus(filterStatus)
			}
			result := f.Result()
			if flagJSON {
				printJSON(result)
				return nil
			}
			printDelegatedFiltered(result)
			return nil

		case "extended":
			entries, err := client.GetExtendedEntries(ctx)
			if err != nil {
				return err
			}
			f := apnic.NewExtendedFilter(entries)
			if filterCountry != "" {
				f = f.ByCountry(filterCountry)
			}
			if filterType != "" {
				f = f.ByType(filterType)
			}
			if filterStatus != "" {
				f = f.ByStatus(filterStatus)
			}
			if filterOpaqueID != "" {
				f = f.ByOpaqueID(filterOpaqueID)
			}
			result := f.Result()
			if flagJSON {
				printJSON(result)
				return nil
			}
			printExtendedFiltered(result)
			return nil

		default:
			return fmt.Errorf("unknown --source %q (use delegated or extended)", filterSource)
		}
	},
}

func printDelegatedFiltered(entries []apnic.DelegatedEntry) {
	fmt.Printf("# %d entries after filter\n", len(entries))
	for _, e := range entries {
		fmt.Printf("%s\t%s\t%s\t%d\t%s\n", e.Country, e.Type, e.Start, e.Value, e.Status)
	}
}

func printExtendedFiltered(entries []apnic.DelegatedExtendedEntry) {
	fmt.Printf("# %d entries after filter\n", len(entries))
	for _, e := range entries {
		fmt.Printf("%s\t%s\t%s\t%d\t%s\t%s\n", e.Country, e.Type, e.Start, e.Value, e.Status, e.OpaqueID)
	}
}
