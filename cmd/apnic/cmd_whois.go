package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/query"
)

func init() {
	whoisCmd.AddCommand(whoisIPCmd)
	whoisCmd.AddCommand(whoisASNCmd)
	whoisCmd.AddCommand(whoisRawCmd)
	whoisCmd.AddCommand(whoisRulesCmd)
	rootCmd.AddCommand(whoisCmd)
	rootCmd.AddCommand(reverseDNSCmd)

	// whois raw: pass arbitrary APNIC whois flags (e.g. "-L", "B", "r").
	whoisRawCmd.Flags().StringVar(&flagWhoisFlags, "flags", "", "whois query flags (e.g. \"-L\" all-less-specific, \"B\" brief, \"r\" no recursion)")
	// whois rules: select inetnum/route specificity scope.
	whoisRulesCmd.Flags().String("scope", "all-less", "rule scope: exact, one-less, all-less, one-more, all-more")
}

var whoisCmd = &cobra.Command{
	Use:   "whois",
	Short: "Whois queries against the APNIC whois server",
}

var whoisIPCmd = &cobra.Command{
	Use:   "ip <ip>",
	Short: "Parsed whois lookup for an IP address",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		info, err := client.QueryWhoisIP(ctx, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(info)
			return nil
		}
		fmt.Printf("Network:      %s\n", info.Network)
		fmt.Printf("NetName:      %s\n", info.NetName)
		fmt.Printf("CIDR:         %v\n", info.CIDR)
		fmt.Printf("Country:      %s\n", info.Country)
		fmt.Printf("Org:          %s\n", info.OrgName)
		fmt.Printf("Status:       %s\n", info.Status)
		fmt.Printf("Origin ASN:   %s\n", info.OriginASN)
		fmt.Printf("Abuse:        %s\n", info.AbuseContact)
		fmt.Printf("Parent:       %s\n", info.Parent)
		fmt.Printf("Created:      %s\n", info.Created)
		fmt.Printf("LastUpdated:  %s\n", info.LastUpdated)
		return nil
	},
}

var whoisASNCmd = &cobra.Command{
	Use:   "asn <asn>",
	Short: "Parsed whois lookup for an ASN (e.g. 13335 or AS13335)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asn, err := strconv.ParseInt(normalizeASN(args[0]), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ASN %q: %w", args[0], err)
		}
		client := newClient()
		ctx := context.Background()
		info, err := client.QueryWhoisASN(ctx, asn)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(info)
			return nil
		}
		fmt.Printf("Network:      %s\n", info.Network)
		fmt.Printf("NetName:      %s\n", info.NetName)
		fmt.Printf("Country:      %s\n", info.Country)
		fmt.Printf("Org:          %s\n", info.OrgName)
		fmt.Printf("Status:       %s\n", info.Status)
		fmt.Printf("Origin ASN:   %s\n", info.OriginASN)
		fmt.Printf("Abuse:        %s\n", info.AbuseContact)
		fmt.Printf("LastUpdated:  %s\n", info.LastUpdated)
		return nil
	},
}

var whoisRawCmd = &cobra.Command{
	Use:   "raw <query>",
	Short: "Raw whois query (returns unparsed text)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		resp, err := client.QueryWhoisWithFlags(ctx, args[0], flagWhoisFlags)
		if err != nil {
			return err
		}
		fmt.Print(resp)
		return nil
	},
}

// whoisRuleScopes maps a human-friendly scope name to the APNIC whois flag that
// selects inetnum/route objects at that specificity level. See "whois.apnic.net"
// help output for the flag semantics.
var whoisRuleScopes = map[string]string{
	"exact":    "-x",
	"one-less": "-l",
	"all-less": "-L",
	"one-more": "-m",
	"all-more": "-M",
}

var whoisRulesCmd = &cobra.Command{
	Use:   "rules <ip>",
	Short: "Query inetnum/route objects around an IP by scope (all-less, one-less, all-more, one-more, exact)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scope, _ := cmd.Flags().GetString("scope")
		flag, ok := whoisRuleScopes[scope]
		if !ok {
			return fmt.Errorf("invalid scope %q: valid scopes are exact, one-less, all-less, one-more, all-more", scope)
		}
		client := newClient()
		ctx := context.Background()
		resp, err := client.QueryWhoisWithFlags(ctx, args[0], flag)
		if err != nil {
			return err
		}
		list := query.ParseWhoisResponseList(resp)
		if flagJSON {
			printJSON(list)
			return nil
		}
		if len(list) == 0 {
			fmt.Println("(no matching inetnum/route objects)")
			return nil
		}
		for i, info := range list {
			fmt.Printf("--- object %d ---\n", i+1)
			fmt.Printf("Network:      %s\n", info.Network)
			fmt.Printf("NetName:      %s\n", info.NetName)
			fmt.Printf("CIDR:         %v\n", info.CIDR)
			fmt.Printf("Country:      %s\n", info.Country)
			fmt.Printf("Status:       %s\n", info.Status)
			fmt.Printf("Origin ASN:   %s\n", info.OriginASN)
			fmt.Printf("Abuse:        %s\n", info.AbuseContact)
			fmt.Printf("AbuseMailbox: %s\n", info.AbuseMailbox)
			fmt.Printf("LastUpdated:  %s\n", info.LastUpdated)
		}
		return nil
	},
}

var reverseDNSCmd = &cobra.Command{
	Use:   "reverse-dns <ip>",
	Short: "Reverse DNS lookup for an IP address",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		names, err := client.ReverseDNS(ctx, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(names)
			return nil
		}
		if len(names) == 0 {
			fmt.Println("(no PTR records)")
			return nil
		}
		for _, n := range names {
			fmt.Println(n)
		}
		return nil
	},
}

// normalizeASN strips an optional "AS" prefix so the user can pass either 13335 or AS13335.
func normalizeASN(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "AS")
	s = strings.TrimPrefix(s, "as")
	return s
}
