package transport

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"
)

// VerifyMD5 verifies the MD5 checksum of a stats data file.
// dataType: "delegated", "delegated-extended", "assigned", "delegated-ipv6-assigned", "legacy"
// date: optional date in YYYYMMDD format; if empty, uses "latest"
func (c *Client) VerifyMD5(ctx context.Context, dataType, date string) error {
	// Fetch the data
	dataURL := BuildStatsURL(c.statsBaseURL, dataType, date)
	data, err := c.FetchText(ctx, dataURL)
	if err != nil {
		return err
	}

	// Fetch the MD5 checksum
	md5URL := BuildStatsMD5URL(c.statsBaseURL, dataType, date)
	expectedMD5, err := c.FetchText(ctx, md5URL)
	if err != nil {
		return err
	}

	expectedHash, err := ParseMD5Checksum(expectedMD5)
	if err != nil {
		return err
	}

	// Calculate the MD5 of the data
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	if hash != expectedHash {
		return fmt.Errorf("%w: MD5 mismatch (expected %s, got %s)", ErrVerifyFailed, expectedHash, hash)
	}

	return nil
}

// FetchMD5Checksum fetches the MD5 checksum for a stats data file.
// dataType: "delegated", "delegated-extended", "assigned", "delegated-ipv6-assigned", "legacy"
// date: optional date in YYYYMMDD format; if empty, uses "latest
func (c *Client) FetchMD5Checksum(ctx context.Context, dataType, date string) (string, error) {
	md5URL := BuildStatsMD5URL(c.statsBaseURL, dataType, date)
	content, err := c.FetchText(ctx, md5URL)
	if err != nil {
		return "", err
	}
	return ParseMD5Checksum(content)
}

// parseMD5Checksum extracts the hex MD5 hash from a checksum file.
// Supports both BSD-style "MD5 (file) = <hash>" and GNU-style "<hash>  file".
func ParseMD5Checksum(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("%w: empty MD5 checksum file", ErrVerifyFailed)
	}

	// BSD style: "MD5 (filename) = <hash>"
	if strings.Contains(content, "=") {
		parts := strings.SplitN(content, "=", 2)
		hash := strings.TrimSpace(parts[1])
		if isMD5Hex(hash) {
			return hash, nil
		}
	}

	// GNU style: "<hash>  filename" — take the first whitespace-delimited field.
	fields := strings.Fields(content)
	for _, f := range fields {
		if isMD5Hex(f) {
			return f, nil
		}
	}

	return "", fmt.Errorf("%w: could not parse MD5 checksum from: %s", ErrVerifyFailed, content)
}

// isMD5Hex reports whether s is a 32-character lowercase hex string.
func isMD5Hex(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// FetchASCSignature fetches the ASC (PGP) signature for a stats data file.
// dataType: "delegated", "delegated-extended", "assigned", "legacy"
// date: optional date in YYYYMMDD format; if empty, uses "latest"
func (c *Client) FetchASCSignature(ctx context.Context, dataType, date string) (string, error) {
	ascURL := BuildStatsASCURL(c.statsBaseURL, dataType, date)
	return c.FetchText(ctx, ascURL)
}

// FetchPublicKey fetches the current APNIC public key used for signing stats files.
func (c *Client) FetchPublicKey(ctx context.Context) (string, error) {
	url := c.statsBaseURL + "CURRENT_PUBLIC_KEY"
	return c.FetchText(ctx, url)
}
