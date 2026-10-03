package stats

import (
	"bufio"
	"context"
	"io"
	"strings"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// FetchLegacyEntries fetches the latest legacy stats from APNIC.
// Legacy stats contain historical resource records that were registered before APNIC's establishment.
func FetchLegacyEntries(ctx context.Context, c *transport.Client) (*models.LegacyResult, error) {
	return FetchLegacyResult(ctx, c, "")
}

// FetchLegacyEntriesByDate fetches legacy stats for a specific date.
// date must be in YYYYMMDD format.
func FetchLegacyEntriesByDate(ctx context.Context, c *transport.Client, date string) (*models.LegacyResult, error) {
	return FetchLegacyResult(ctx, c, date)
}

// FetchLegacyResult fetches and parses the full legacy stats result.
// If date is empty, fetches the latest; otherwise fetches the specified date (YYYYMMDD).
func FetchLegacyResult(ctx context.Context, c *transport.Client, date string) (*models.LegacyResult, error) {
	url := transport.BuildStatsURL(c.StatsBaseURL(), "legacy", date)
	r, err := c.FetchReader(ctx, url)
	if err != nil {
		return nil, err
	}
	return ParseLegacyFull(r)
}

// ParseLegacyFull parses the complete legacy stats file.
// Legacy format: version|registry|serial|records|startdate|enddate|UTCoffset
// followed by: registry|cc|type|start|value|date|status
// Note: Legacy entries may have empty country codes.
func ParseLegacyFull(r io.Reader) (*models.LegacyResult, error) {
	scanner := bufio.NewScanner(r)
	result := &models.LegacyResult{
		Summaries: make([]models.StatsSummary, 0),
		Entries:   make([]models.LegacyEntry, 0, 2000),
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

		parts := strings.Split(line, "|")
		if len(parts) < 7 {
			continue
		}

		entry := models.LegacyEntry{
			Registry: parts[0],
			Country:  parts[1], // may be empty for legacy entries
			Type:     parts[2],
			Start:    parts[3],
			Status:   parts[6],
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
