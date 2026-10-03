package stats

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// FetchExtendedEntries fetches the latest extended delegated stats from APNIC.
// Extended stats include the OpaqueID field that uniquely identifies the resource holder.
func FetchExtendedEntries(ctx context.Context, c *transport.Client) (*models.ExtendedResult, error) {
	return FetchExtendedResult(ctx, c, "")
}

// FetchExtendedEntriesByDate fetches extended delegated stats for a specific date.
// date must be in YYYYMMDD format.
func FetchExtendedEntriesByDate(ctx context.Context, c *transport.Client, date string) (*models.ExtendedResult, error) {
	return FetchExtendedResult(ctx, c, date)
}

// FetchExtendedResult fetches and parses the full extended delegated stats result.
// If date is empty, fetches the latest; otherwise fetches the specified date (YYYYMMDD).
func FetchExtendedResult(ctx context.Context, c *transport.Client, date string) (*models.ExtendedResult, error) {
	url := transport.BuildStatsURL(c.StatsBaseURL(), "delegated-extended", date)
	r, err := c.FetchReader(ctx, url)
	if err != nil {
		return nil, err
	}
	return ParseExtendedFull(r)
}

// FetchExtendedResultByYear fetches extended delegated stats for a specific year.
func FetchExtendedResultByYear(ctx context.Context, c *transport.Client, year int) (*models.ExtendedResult, error) {
	url := fmt.Sprintf("%s%d/delegated-apnic-extended-%d1231.gz", c.StatsBaseURL(), year, year)
	r, err := c.FetchReader(ctx, url)
	if err != nil {
		return nil, err
	}
	return ParseExtendedFull(r)
}

// ParseExtendedFull parses the complete extended delegated stats file.
// Extended format: registry|cc|type|start|value|date|status|opaque-id[|extensions...]
func ParseExtendedFull(r io.Reader) (*models.ExtendedResult, error) {
	scanner := bufio.NewScanner(r)
	result := &models.ExtendedResult{
		Summaries: make([]models.StatsSummary, 0),
		Entries:   make([]models.DelegatedExtendedEntry, 0, 10000),
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse header line
		if transport.IsHeaderLine(line) {
			header, err := transport.ParseStatsHeader(line)
			if err == nil {
				result.Header = *header
			}
			continue
		}

		// Parse summary line
		if transport.IsSummaryLine(line) {
			summary, err := transport.ParseSummaryLine(line)
			if err == nil {
				result.Summaries = append(result.Summaries, *summary)
			}
			continue
		}

		// Parse data line — extended format has at least 8 fields
		parts := strings.Split(line, "|")
		if len(parts) < 8 {
			continue
		}

		entry := models.DelegatedExtendedEntry{
			Registry:   parts[0],
			Country:    parts[1],
			Type:       parts[2],
			Start:      parts[3],
			Status:     parts[6],
			OpaqueID:   transport.ParseOpaqueID(parts[7]),
			Extensions: parts[8:],
		}

		var parseErr error
		switch entry.Type {
		case "ipv4":
			entry.Value, parseErr = transport.ParseIPv4Count(parts[4])
		case "ipv6":
			entry.Value, parseErr = transport.ParseIPv6Prefix(parts[4])
		case "asn":
			entry.Value, parseErr = transport.ParseASNCount(parts[4])
		default:
			continue
		}

		if parseErr != nil {
			continue
		}

		if date, err := transport.ParseStatsDate(parts[5]); err == nil {
			entry.Date = date
		}

		result.Entries = append(result.Entries, entry)
	}

	return result, scanner.Err()
}

// ParseExtendedFullFromString parses the complete extended delegated stats from a string.
func ParseExtendedFullFromString(data string) (*models.ExtendedResult, error) {
	return ParseExtendedFull(strings.NewReader(data))
}
