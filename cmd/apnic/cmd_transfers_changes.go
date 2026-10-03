package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	transfersYear    int
	transfersAllDate string
	changesDate      string
)

func init() {
	transfersCmd.Flags().IntVar(&transfersYear, "year", 0, "fetch transfers for a specific year (JCR log)")
	transfersAllCmd.Flags().StringVar(&transfersAllDate, "date", "", "fetch cumulative transfers-all for a specific date (YYYYMMDD)")
	changesCmd.Flags().StringVar(&changesDate, "date", "", "fetch changes for a specific date (YYYYMMDD)")
	rootCmd.AddCommand(transfersCmd)
	rootCmd.AddCommand(transfersAllCmd)
	rootCmd.AddCommand(changesCmd)
}

var transfersCmd = &cobra.Command{
	Use:   "transfers",
	Short: "Fetch IP/ASN transfer records",
	Long: `Fetch APNIC inter- and intra-RIR IP/ASN transfer records.

By default fetches the latest transfers (transfers_latest.json). Use --year YYYY
to fetch the JCR-format transfer log for that year.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()

		var result interface{}
		var err error
		var count int
		if transfersYear != 0 {
			r, e := client.FetchTransfersByYear(ctx, transfersYear)
			if e != nil {
				err = e
			} else {
				result, count = r, len(r.Transfers)
			}
		} else {
			r, e := client.FetchTransfers(ctx)
			if e != nil {
				err = e
			} else {
				result, count = r, len(r.Transfers)
			}
		}
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(result)
			return nil
		}
		fmt.Printf("# transfers: %d records (year=%s)\n", count, yearOrDefault(transfersYear))
		return nil
	},
}

var changesCmd = &cobra.Command{
	Use:   "changes",
	Short: "Fetch resource change records",
	Long: `Fetch APNIC resource change records (JSON Lines).

Each record describes a delegated/cc-changed/status-changed event for a resource.
By default fetches the latest; use --date YYYYMMDD for a specific snapshot.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()

		var result interface{}
		var err error
		var count int
		if changesDate != "" {
			r, e := client.FetchChangesByDate(ctx, changesDate)
			if e != nil {
				err = e
			} else {
				result, count = r, len(r.Changes)
			}
		} else {
			r, e := client.FetchChanges(ctx)
			if e != nil {
				err = e
			} else {
				result, count = r, len(r.Changes)
			}
		}
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(result)
			return nil
		}
		fmt.Printf("# changes: %d records (date=%s)\n", count, dateOrDefault(changesDate))
		return nil
	},
}

func yearOrDefault(y int) string {
	if y == 0 {
		return "latest"
	}
	return strconv.Itoa(y)
}

var transfersAllCmd = &cobra.Command{
	Use:   "transfers-all",
	Short: "Fetch the cumulative transfers-all log (all transfers since 2010)",
	Long: `Fetch the APNIC cumulative transfers-all log.

Unlike 'transfers' (the daily JSON snapshot), this is the historical
pipe-delimited format covering every IP/ASN transfer recorded since 2010.
Use --date YYYYMMDD to fetch a specific daily archive; omit for the latest
cumulative file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		r, err := client.FetchTransfersAll(ctx, transfersAllDate)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(r)
			return nil
		}
		dateStr := transfersAllDate
		if dateStr == "" {
			dateStr = "latest"
		}
		fmt.Printf("# transfers-all: %d records (date=%s)\n", len(r.Records), dateStr)
		for _, rec := range r.Records {
			fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n", rec.ResourceType, rec.Resource, rec.FromOrganisation, rec.ToOrganisation, rec.TransferType, rec.TransferDate.Format("20060102"))
		}
		return nil
	},
}
