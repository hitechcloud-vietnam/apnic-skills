package stats

import (
	"bufio"
	"context"
	"io"
	"strings"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// FetchIPv6AssignedEntries fetches the latest delegated-apnic-ipv6-assigned stats from APNIC.
// This file lists individual IPv6 assignments (a per-prefix view, distinct from the
// aggregated "assigned" stats which count assignments by prefix size).
func FetchIPv6AssignedEntries(ctx context.Context, c *transport.Client) ([]models.IPv6AssignedEntry, error) {
	result, err := FetchIPv6AssignedResult(ctx, c, "")
	if err != nil {
		return nil, err
	}
	return result.Entries, nil
}

// FetchIPv6AssignedEntriesByDate fetches delegated-apnic-ipv6-assigned stats for a specific date.
// date must be in YYYYMMDD format.
func FetchIPv6AssignedEntriesByDate(ctx context.Context, c *transport.Client, date string) ([]models.IPv6AssignedEntry, error) {
	result, err := FetchIPv6AssignedResult(ctx, c, date)
	if err != nil {
		return nil, err
	}
	return result.Entries, nil
}

// FetchIPv6AssignedResult fetches and parses the full delegated-apnic-ipv6-assigned stats
// result including header and summaries.
// If date is empty, fetches the latest; otherwise fetches the specified date (YYYYMMDD).
func FetchIPv6AssignedResult(ctx context.Context, c *transport.Client, date string) (*models.IPv6AssignedResult, error) {
	url := transport.BuildStatsURL(c.StatsBaseURL(), "delegated-ipv6-assigned", date)
	r, err := c.FetchReader(ctx, url)
	if err != nil {
		return nil, err
	}
	return ParseIPv6AssignedFull(r)
}

// ParseIPv6AssignedFull parses the complete delegated-apnic-ipv6-assigned stats file.
// Row format: registry|cc|ipv6|start|prefix|date  (6 fields, no status/extensions)
func ParseIPv6AssignedFull(r io.Reader) (*models.IPv6AssignedResult, error) {
	scanner := bufio.NewScanner(r)
	result := &models.IPv6AssignedResult{
		Summaries: make([]models.StatsSummary, 0),
		Entries:   make([]models.IPv6AssignedEntry, 0, 1000),
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

		// Parse data line: registry|cc|ipv6|start|prefix|date
		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}

		// Only ipv6 rows are present in this file.
		if parts[2] != "ipv6" {
			continue
		}

		prefix, err := transport.ParseIPv6Prefix(parts[4])
		if err != nil {
			continue
		}

		entry := models.IPv6AssignedEntry{
			Registry: parts[0],
			Country:  parts[1],
			Start:    parts[3],
			Value:    prefix,
		}

		if date, err := transport.ParseStatsDate(parts[5]); err == nil {
			entry.Date = date
		}

		result.Entries = append(result.Entries, entry)
	}

	return result, scanner.Err()
}
