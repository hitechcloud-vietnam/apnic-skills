package main

import (
	"context"
	"fmt"

	apnic "github.com/hitechcloud-vietnam/apnic-skills"
	"github.com/spf13/cobra"
)

var (
	histType string
	histDate string
	histYear int
)

func init() {
	historyCmd.Flags().StringVar(&histType, "type", "delegated", "data type: delegated, extended, assigned, legacy")
	historyCmd.Flags().StringVar(&histDate, "date", "", "fetch snapshot for a date (YYYYMMDD)")
	historyCmd.Flags().IntVar(&histYear, "year", 0, "fetch latest file for a given year (>=2001)")
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(yearsCmd)
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Fetch historical stats snapshots by date or by year",
	Long: `Fetch historical APNIC stats snapshots.

Use --date YYYYMMDD for a specific day's snapshot, or --year YYYY for the latest
file published in that year. --type selects the data file (delegated, extended,
assigned, legacy). Exactly one of --date or --year must be given.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if histDate == "" && histYear == 0 {
			return fmt.Errorf("either --date or --year must be specified")
		}
		if histDate != "" && histYear != 0 {
			return fmt.Errorf("--date and --year are mutually exclusive")
		}
		client := newClient()
		ctx := context.Background()

		if histYear != 0 {
			return runHistoryByYear(ctx, client)
		}
		return runHistoryByDate(ctx, client)
	},
}

var yearsCmd = &cobra.Command{
	Use:   "years",
	Short: "List years for which historical APNIC stats are available",
	RunE: func(cmd *cobra.Command, args []string) error {
		years := apnic.ListAvailableYears()
		if flagJSON {
			printJSON(years)
			return nil
		}
		for _, y := range years {
			fmt.Println(y)
		}
		return nil
	},
}

func runHistoryByDate(ctx context.Context, client *apnic.Client) error {
	switch histType {
	case "delegated":
		r, err := client.FetchHistoricalDelegated(ctx, histDate)
		if err != nil {
			return err
		}
		fmt.Printf("# delegated history: %d entries (date=%s)\n", len(r.Entries), histDate)
		if flagJSON {
			printJSON(r)
		}
	case "extended":
		r, err := client.FetchHistoricalExtended(ctx, histDate)
		if err != nil {
			return err
		}
		fmt.Printf("# extended history: %d entries (date=%s)\n", len(r.Entries), histDate)
		if flagJSON {
			printJSON(r)
		}
	case "assigned":
		r, err := client.FetchHistoricalAssigned(ctx, histDate)
		if err != nil {
			return err
		}
		fmt.Printf("# assigned history: %d entries (date=%s)\n", len(r.Entries), histDate)
		if flagJSON {
			printJSON(r)
		}
	case "legacy":
		r, err := client.FetchHistoricalLegacy(ctx, histDate)
		if err != nil {
			return err
		}
		fmt.Printf("# legacy history: %d entries (date=%s)\n", len(r.Entries), histDate)
		if flagJSON {
			printJSON(r)
		}
	default:
		return fmt.Errorf("unknown --type %q (use delegated, extended, assigned, legacy)", histType)
	}
	return nil
}

func runHistoryByYear(ctx context.Context, client *apnic.Client) error {
	switch histType {
	case "delegated":
		r, err := client.FetchDelegatedByYear(ctx, histYear)
		if err != nil {
			return err
		}
		fmt.Printf("# delegated by-year: %d entries (year=%d)\n", len(r.Entries), histYear)
		if flagJSON {
			printJSON(r)
		}
	case "extended":
		r, err := client.FetchExtendedByYear(ctx, histYear)
		if err != nil {
			return err
		}
		fmt.Printf("# extended by-year: %d entries (year=%d)\n", len(r.Entries), histYear)
		if flagJSON {
			printJSON(r)
		}
	default:
		return fmt.Errorf("--year only supports --type delegated or extended (got %q)", histType)
	}
	return nil
}
