package main

import (
	"context"
	"fmt"
	"strings"

	apnic "github.com/hitechcloud-vietnam/apnic-skills"

	"github.com/spf13/cobra"
)

func init() {
	rpkiCmd.AddCommand(rpkiNotificationCmd)
	rpkiCmd.AddCommand(rpkiSnapshotCmd)
	rootCmd.AddCommand(rpkiCmd)
}

var rpkiCmd = &cobra.Command{
	Use:   "rpki",
	Short: "Fetch APNIC RPKI repository data via RRDP",
	Long: `Fetch APNIC RPKI repository data via the RRDP protocol.

APNIC publishes its RPKI repository through RRDP (https://rrdp.apnic.net).
The notification.xml file points to the current snapshot (a full dump of all
RPKI objects) and a list of deltas (incremental updates). Use 'rpki
notification' for the notification metadata, or 'rpki snapshot <uri>' to stream
and summarise a snapshot by its URI (taken from the notification's snapshot ref).
Snapshot URIs are large; only the rsync object URIs are listed, the base64 CMS
bodies are discarded during streaming.`,
}

var rpkiNotificationCmd = &cobra.Command{
	Use:   "notification",
	Short: "Fetch the RRDP notification.xml (session, serial, snapshot & deltas)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		n, err := client.FetchRRDPNotification(context.Background())
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(n)
			return nil
		}
		fmt.Printf("# rpki notification: session=%s serial=%d\n", n.SessionID, n.Serial)
		fmt.Printf("snapshot\t%s\t%s\n", n.Snapshot.URI, n.Snapshot.Hash)
		fmt.Printf("deltas: %d\n", len(n.Deltas))
		limit := len(n.Deltas)
		if limit > 20 {
			limit = 20
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("delta\t%d\t%s\n", n.Deltas[i].Serial, n.Deltas[i].URI)
		}
		if len(n.Deltas) > limit {
			fmt.Printf("... (%d more deltas)\n", len(n.Deltas)-limit)
		}
		return nil
	},
}

var rpkiSnapshotCmd = &cobra.Command{
	Use:   "snapshot [uri]",
	Short: "Stream an RRDP snapshot.xml by URI and summarise published/withdrawn objects",
	Long: `Stream an RRDP snapshot.xml by URI and summarise published/withdrawn objects.

If no URI is given, the snapshot URI is resolved from the notification.xml.
If a relative path (e.g. "snapshot.xml") is given, it is resolved against the
--rrdp-base-url. An absolute URI (as returned by 'rpki notification') is
fetched as-is.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		uri, err := resolveSnapshotURI(client, args)
		if err != nil {
			return err
		}
		s, err := client.FetchRRDPSnapshot(context.Background(), uri)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(s)
			return nil
		}
		fmt.Printf("# rpki snapshot: session=%s serial=%d published=%d withdrawn=%d\n",
			s.SessionID, s.Serial, len(s.Published), len(s.Withdrawn))
		return nil
	},
}

// rrdpBaseURL returns the effective RRDP base URL (the flag value, or the
// SDK default if the flag is unset).
func rrdpBaseURL() string {
	if flagRRDPBaseURL != "" {
		return flagRRDPBaseURL
	}
	return apnic.DefaultRRDPBaseURL
}

// resolveSnapshotURI resolves the snapshot URI for the 'rpki snapshot' command:
// if an explicit absolute URI is given, use it; if a relative path is given,
// join it to the RRDP base URL; otherwise fetch the notification and use its
// snapshot reference.
func resolveSnapshotURI(client *apnic.Client, args []string) (string, error) {
	if len(args) == 1 {
		if strings.HasPrefix(args[0], "http://") || strings.HasPrefix(args[0], "https://") {
			return args[0], nil
		}
		return strings.TrimRight(rrdpBaseURL(), "/") + "/" + strings.TrimLeft(args[0], "/"), nil
	}
	n, err := client.FetchRRDPNotification(context.Background())
	if err != nil {
		return "", err
	}
	return n.Snapshot.URI, nil
}
