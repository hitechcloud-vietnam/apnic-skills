package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	irrCmd.AddCommand(irrSerialCmd)
	rootCmd.AddCommand(irrCmd)
}

var irrCmd = &cobra.Command{
	Use:   "irr <type>",
	Short: "Fetch APNIC IRR (RPSL) database dumps and serial",
	Long: `Fetch APNIC Internet Routing Registry (IRR) database dumps.

APNIC publishes gzipped RPSL dumps for each object type under
ftp.apnic.net/apnic/whois/apnic.db.<type>.gz (e.g. inetnum, aut-num, route,
domain). The 'domain' dumps in particular carry reverse-DNS delegation
records (x.in-addr.arpa with nserver/zone-c attributes).

Run 'apnic irr <type>' to fetch and parse a dump (e.g. 'apnic irr inetnum'),
or 'apnic irr serial' for the APNIC.CURRENTSERIAL value (the current IRR
database serial number). Valid types: as-block, as-set, aut-num, domain,
filter-set, inet6num, inetnum, inet-rtr, irt, key-cert, limerick, mntner,
organisation, peering-set, role, route, route6, route-set, rtr-set.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		objType := args[0]
		client := newClient()
		ctx := context.Background()

		// GetIRRDatabase caches within the configured TTL; repeated invocations
		// within the TTL are cheap.
		db, err := client.GetIRRDatabase(ctx, objType)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(db)
			return nil
		}
		fmt.Printf("# irr %s: %d objects\n", db.Type, len(db.Objects))
		limit := len(db.Objects)
		if limit > 50 {
			limit = 50
		}
		for i := 0; i < limit; i++ {
			o := db.Objects[i]
			fmt.Printf("%s\t%s\n", o.Type, o.PrimaryKey)
		}
		if len(db.Objects) > limit {
			fmt.Printf("... (%d more)\n", len(db.Objects)-limit)
		}
		return nil
	},
}

var irrSerialCmd = &cobra.Command{
	Use:   "serial",
	Short: "Fetch the APNIC.CURRENTSERIAL value",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		serial, err := client.FetchIRRCurrentSerial(context.Background())
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(map[string]int64{"serial": serial})
			return nil
		}
		fmt.Println(serial)
		return nil
	},
}
