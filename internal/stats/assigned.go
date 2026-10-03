package stats

import (
	"bufio"
	"context"
	"io"
	"strings"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// FetchAssignedEntries fetches the latest assigned stats from APNIC.
// Assigned stats show aggregated assignment counts by prefix size per country.
func FetchAssignedEntries(ctx context.Context, c *transport.Client) (*models.AssignedResult, error) {
	return FetchAssignedResult(ctx, c, "")
}

// FetchAssignedEntriesByDate fetches assigned stats for a specific date.
// date must be in YYYYMMDD format.
func FetchAssignedEntriesByDate(ctx context.Context, c *transport.Client, date string) (*models.AssignedResult, error) {
	return FetchAssignedResult(ctx, c, date)
}

// FetchAssignedResult fetches and parses the full assigned stats result.
// If date is empty, fetches the latest; otherwise fetches the specified date (YYYYMMDD).
func FetchAssignedResult(ctx context.Context, c *transport.Client, date string) (*models.AssignedResult, error) {
	url := transport.BuildStatsURL(c.StatsBaseURL(), "assigned", date)
	r, err := c.FetchReader(ctx, url)
	if err != nil {
		return nil, err
	}
	return ParseAssignedFull(r)
}

// ParseAssignedFull parses the complete assigned stats file.
// Assigned format: registry|cc|type||prefix_size||status|||count
// Example: apnic|ae|ipv4||4||assigned|||1
func ParseAssignedFull(r io.Reader) (*models.AssignedResult, error) {
	scanner := bufio.NewScanner(r)
	result := &models.AssignedResult{
		Summaries: make([]models.StatsSummary, 0),
		Entries:   make([]models.AssignedEntry, 0, 1000),
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
		if len(parts) < 9 {
			continue
		}

		// Skip non-ipv4/ipv6 entries (assigned stats only have ipv4/ipv6)
		entryType := parts[2]
		if entryType != "ipv4" && entryType != "ipv6" {
			continue
		}

		var count int64
		if parsed, err := transport.ParseIPv4Count(parts[8]); err == nil {
			count = parsed
		}

		entry := models.AssignedEntry{
			Registry: parts[0],
			Country:  parts[1],
			Type:     entryType,
			Prefix:   parts[4], // prefix size (e.g. "4", "256")
			Count:    count,
			Status:   parts[6],
		}

		result.Entries = append(result.Entries, entry)
	}

	return result, scanner.Err()
}
