package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	apnic "github.com/hitechcloud-vietnam/apnic-skills"
	"github.com/spf13/cobra"
)

var rdapDateFlag string

func init() {
	rdapCmd.PersistentFlags().StringVar(&rdapDateFlag, "date", "", "point-in-time RDAP query (RFC3339, e.g. 2020-06-01T00:00:00Z); empty = live")

	rdapIPCmd.Flags().StringVar(&rdapDateFlag, "date", "", "point-in-time query (RFC3339)")
	rdapCIDRCmd.Flags().StringVar(&rdapDateFlag, "date", "", "point-in-time query (RFC3339)")
	rdapASNCmd.Flags().StringVar(&rdapDateFlag, "date", "", "point-in-time query (RFC3339)")
	rdapDomainCmd.Flags().StringVar(&rdapDateFlag, "date", "", "point-in-time query (RFC3339)")
	rdapEntityCmd.Flags().StringVar(&rdapDateFlag, "date", "", "point-in-time query (RFC3339)")

	rdapCmd.AddCommand(rdapIPCmd)
	rdapCmd.AddCommand(rdapCIDRCmd)
	rdapCmd.AddCommand(rdapASNCmd)
	rdapCmd.AddCommand(rdapDomainCmd)
	rdapCmd.AddCommand(rdapEntityCmd)
	rdapCmd.AddCommand(rdapSearchCmd)
	rdapCmd.AddCommand(rdapHelpCmd)
	rdapCmd.AddCommand(rdapDomainsCmd)
	rootCmd.AddCommand(rdapCmd)
}

var rdapCmd = &cobra.Command{
	Use:   "rdap",
	Short: "RDAP lookups (IP, CIDR, ASN, domain, entity, search)",
	Long: `Query the APNIC RDAP service for structured registration data.

All lookup subcommands support an optional --date flag for point-in-time
(historical) queries, returning the resource state as it was at that UTC instant.
This uses APNIC's history_version_0 extension.`,
}

func rdapDateOption() apnic.Option {
	// noOp is returned when no date is set, so newClient never receives a nil Option.
	noOp := func(c *apnic.Client) {}
	if rdapDateFlag == "" {
		return noOp
	}
	if t, err := time.Parse(time.RFC3339, rdapDateFlag); err == nil {
		return apnic.WithRDAPDate(t)
	}
	return noOp
}

var rdapIPCmd = &cobra.Command{
	Use:   "ip <ip>",
	Short: "RDAP lookup for an IP address",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient(rdapDateOption())
		ctx := context.Background()
		n, err := client.RDAPLookupIP(ctx, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(n)
			return nil
		}
		printRDAPNetwork(n)
		return nil
	},
}

var rdapCIDRCmd = &cobra.Command{
	Use:   "cidr <cidr>",
	Short: "RDAP lookup for a CIDR block (e.g. 1.1.1.0/24)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient(rdapDateOption())
		ctx := context.Background()
		n, err := client.RDAPLookupCIDR(ctx, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(n)
			return nil
		}
		printRDAPNetwork(n)
		return nil
	},
}

var rdapASNCmd = &cobra.Command{
	Use:   "asn <asn>",
	Short: "RDAP lookup for an Autonomous System Number (plain number, e.g. 13335)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asn, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ASN %q: %w", args[0], err)
		}
		client := newClient(rdapDateOption())
		ctx := context.Background()
		a, err := client.RDAPLookupASN(ctx, asn)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(a)
			return nil
		}
		fmt.Printf("Handle:  %s\n", a.Handle)
		fmt.Printf("ASN:     %d - %d\n", a.StartAutnum, a.EndAutnum)
		fmt.Printf("Name:    %s\n", a.Name)
		fmt.Printf("Type:    %s\n", a.Type)
		fmt.Printf("Country: %s\n", a.Country)
		return nil
	},
}

var rdapDomainCmd = &cobra.Command{
	Use:   "domain <domain>",
	Short: "RDAP lookup for a domain (typically reverse-DNS, e.g. 1.0.0.1.in-addr.arpa)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient(rdapDateOption())
		ctx := context.Background()
		d, err := client.RDAPLookupDomain(ctx, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(d)
			return nil
		}
		fmt.Printf("Handle: %s\n", d.Handle)
		fmt.Printf("Name:   %s\n", d.LDHName)
		return nil
	},
}

var rdapEntityCmd = &cobra.Command{
	Use:   "entity <handle>",
	Short: "RDAP lookup for an entity/contact (e.g. ORG-ARAD1-AP, AIC3-AP)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient(rdapDateOption())
		ctx := context.Background()
		e, err := client.RDAPLookupEntity(ctx, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(e)
			return nil
		}
		fmt.Printf("Handle: %s\n", e.Handle)
		fmt.Printf("Roles:  %v\n", e.Roles)
		return nil
	},
}

var rdapSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search RDAP entities by name (fn) or handle",
	Long: `Search the APNIC RDAP entity database.

By default searches by friendly name (fn). APNIC requires wildcards for
substring matches (e.g. "*CLOUD*"); an exact name only matches an entity
whose name equals that string. Use --field handle for exact handle matching.

Use --field fn|handle to select the search criterion.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		field, _ := cmd.Flags().GetString("field")
		if field == "" {
			field = "fn"
		}
		client := newClient()
		ctx := context.Background()
		r, err := client.RDAPSearchEntities(ctx, field, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(r)
			return nil
		}
		fmt.Printf("# search (%s=%s): %d results\n", field, args[0], len(r.EntitySearchResults))
		for _, e := range r.EntitySearchResults {
			fmt.Printf("%s\t%v\n", e.Handle, e.Roles)
		}
		return nil
	},
}

var rdapHelpCmd = &cobra.Command{
	Use:   "help",
	Short: "Fetch RDAP server capability description (/help)",
	Long: `Query the RDAP /help endpoint (RFC 7483).

Returns the server's rdapConformance extensions (e.g. history_version_0,
cidr0, nro_rdap_profile_0), notices (terms of service, inaccuracy reporting),
and port43. Useful to discover which RDAP extensions APNIC supports.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		h, err := client.RDAPHelp(ctx)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(h)
			return nil
		}
		fmt.Printf("Port43:       %s\n", h.Port43)
		fmt.Printf("Conformance:  %v\n", h.Conformance)
		for _, n := range h.Notices {
			fmt.Printf("Notice:       %s\n", n.Title)
			for _, d := range n.Description {
				fmt.Printf("  %s\n", d)
			}
		}
		return nil
	},
}

var rdapDomainsCmd = &cobra.Command{
	Use:   "domains <name>",
	Short: "Search RDAP reverse-DNS domain objects by name",
	Long: `Search the APNIC RDAP database for reverse-DNS domain objects (RFC 7482
/domains?name=). Returns matching domains (e.g. in-addr.arpa zones) in
domainSearchResults.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		r, err := client.RDAPSearchDomains(ctx, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(r)
			return nil
		}
		fmt.Printf("# domains search (name=%s): %d results\n", args[0], len(r.DomainSearchResults))
		for _, d := range r.DomainSearchResults {
			fmt.Printf("%s\t%s\n", d.Handle, d.LDHName)
		}
		return nil
	},
}

func init() {
	rdapSearchCmd.Flags().String("field", "fn", "search field: fn (name, supports wildcards) or handle (exact)")
}

func printRDAPNetwork(n *apnic.RDAPNetwork) {
	fmt.Printf("Handle:   %s\n", n.Handle)
	fmt.Printf("Start:    %s\n", n.StartAddress)
	fmt.Printf("End:      %s\n", n.EndAddress)
	fmt.Printf("Version:  %s\n", n.IPVersion)
	fmt.Printf("Name:     %s\n", n.Name)
	fmt.Printf("Country:  %s\n", n.Country)
	fmt.Printf("Type:     %s\n", n.Type)
	fmt.Printf("Status:   %v\n", n.Status)
	for _, c := range n.CIDR0CIDRs {
		if c.V4Prefix != "" {
			fmt.Printf("CIDR:     %s/%d\n", c.V4Prefix, c.Length)
		}
		if c.V6Prefix != "" {
			fmt.Printf("CIDR:     %s/%d\n", c.V6Prefix, c.Length)
		}
	}
	for _, e := range n.Entities {
		fmt.Printf("Entity:  %s\t%v\n", e.Handle, e.Roles)
	}
	for _, ev := range n.Events {
		fmt.Printf("Event:   %s @ %s\n", ev.EventAction, ev.EventDate)
	}
}
