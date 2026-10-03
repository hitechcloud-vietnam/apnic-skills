package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var telemetryDate string

func init() {
	telemetryCmd.Flags().StringVar(&telemetryDate, "date", "", "fetch telemetry for a specific date (YYYYMMDD; default: latest)")
	rootCmd.AddCommand(telemetryCmd)
}

var telemetryCmd = &cobra.Command{
	Use:   "stats-telemetry",
	Short: "Fetch APNIC whois/RDAP service query telemetry",
	Long: `Fetch the APNIC whois-rdap-stats telemetry (published hourly).

Reports total query volume, per-query-type distribution (ip/autnum/entity/
domain/*_history), and the top-queried ASNs with their query counts. Useful for
understanding APNIC service load and popular resources. Use --date YYYYMMDD for
an archived snapshot.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		t, err := client.FetchTelemetry(ctx, telemetryDate)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(t)
			return nil
		}
		fmt.Printf("# whois-rdap-telemetry: range=%s..%s total=%d asns=%d (date=%s)\n",
			t.RDAP.DateRange.Start, t.RDAP.DateRange.End, t.RDAP.TotalQueries, t.RDAP.TotalASNs, dateOrDefault(telemetryDate))
		fmt.Println("query_type_distribution:")
		for k, v := range t.RDAP.QueryTypeDistribution {
			fmt.Printf("  %s\t%d\n", k, v)
		}
		fmt.Printf("top_asns: %d\n", len(t.RDAP.ASNs))
		return nil
	},
}
