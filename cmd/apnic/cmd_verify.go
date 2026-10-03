package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	verifyDataType string
	verifyDate     string
)

func init() {
	verifyCmd.AddCommand(verifyMD5Cmd)
	verifyCmd.AddCommand(verifyASCCmd)
	verifyCmd.AddCommand(verifyPubKeyCmd)
	verifyCmd.AddCommand(verifyIntegrityCmd)
	rootCmd.AddCommand(verifyCmd)
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify integrity of published APNIC data (MD5, PGP signatures, public key)",
	Long: `Verify the integrity and authenticity of APNIC published data files.

APNIC publishes MD5 checksums and PGP signatures (.asc) for each stats file,
signed with a public key (CURRENT_PUBLIC_KEY). Use these subcommands to fetch
checksums, fetch PGP signatures, or fetch the signing public key.`,
}

// dataTypeValues lists the stats file types that carry .md5/.asc sidecar files.
const verifyDataTypeUsage = "data type: delegated, delegated-extended, assigned, delegated-ipv6-assigned, legacy"

var verifyMD5Cmd = &cobra.Command{
	Use:   "md5",
	Short: "Fetch the MD5 checksum for a stats file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireDataType(); err != nil {
			return err
		}
		client := newClient()
		ctx := context.Background()
		sum, err := client.FetchMD5Checksum(ctx, verifyDataType, verifyDate)
		if err != nil {
			return err
		}
		fmt.Println(sum)
		return nil
	},
}

var verifyASCCmd = &cobra.Command{
	Use:   "asc",
	Short: "Fetch the PGP signature (.asc) for a stats file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireDataType(); err != nil {
			return err
		}
		client := newClient()
		ctx := context.Background()
		sig, err := client.FetchASCSignature(ctx, verifyDataType, verifyDate)
		if err != nil {
			return err
		}
		fmt.Print(sig)
		return nil
	},
}

var verifyPubKeyCmd = &cobra.Command{
	Use:   "pubkey",
	Short: "Fetch the APNIC signing public key (CURRENT_PUBLIC_KEY)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		key, err := client.FetchPublicKey(ctx)
		if err != nil {
			return err
		}
		fmt.Print(key)
		return nil
	},
}

var verifyIntegrityCmd = &cobra.Command{
	Use:   "integrity",
	Short: "Download a stats file and verify its MD5 against the published checksum",
	Long: `Verify the integrity of a published stats file end-to-end.

Fetches both the data file and its MD5 checksum, computes the MD5 of the
downloaded data, and compares. Exits non-zero on mismatch or fetch failure.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireDataType(); err != nil {
			return err
		}
		client := newClient()
		ctx := context.Background()
		if err := client.VerifyMD5(ctx, verifyDataType, verifyDate); err != nil {
			return err
		}
		fmt.Printf("OK: %s (date=%s) MD5 verified\n", verifyDataType, dateOrDefault(verifyDate))
		return nil
	},
}

func init() {
	verifyMD5Cmd.Flags().StringVar(&verifyDataType, "type", "", verifyDataTypeUsage)
	verifyMD5Cmd.Flags().StringVar(&verifyDate, "date", "", "data date in YYYYMMDD (default: latest)")
	verifyASCCmd.Flags().StringVar(&verifyDataType, "type", "", verifyDataTypeUsage)
	verifyASCCmd.Flags().StringVar(&verifyDate, "date", "", "data date in YYYYMMDD (default: latest)")
	verifyIntegrityCmd.Flags().StringVar(&verifyDataType, "type", "", verifyDataTypeUsage)
	verifyIntegrityCmd.Flags().StringVar(&verifyDate, "date", "", "data date in YYYYMMDD (default: latest)")
}

func requireDataType() error {
	if verifyDataType == "" {
		return fmt.Errorf("--type is required (%s)", verifyDataTypeUsage)
	}
	return nil
}
