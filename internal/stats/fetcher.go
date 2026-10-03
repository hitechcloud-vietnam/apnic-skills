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

// FetchDelegatedEntries fetches the latest standard delegated stats from APNIC.
func FetchDelegatedEntries(ctx context.Context, c *transport.Client) ([]models.DelegatedEntry, error) {
	result, err := FetchDelegatedResult(ctx, c, "")
	if err != nil {
		return nil, err
	}
	return result.Entries, nil
}

// FetchDelegatedEntriesByDate fetches delegated stats for a specific date.
// date must be in YYYYMMDD format.
func FetchDelegatedEntriesByDate(ctx context.Context, c *transport.Client, date string) ([]models.DelegatedEntry, error) {
	result, err := FetchDelegatedResult(ctx, c, date)
	if err != nil {
		return nil, err
	}
	return result.Entries, nil
}

// FetchDelegatedResult fetches and parses the full delegated stats result including header and summaries.
// If date is empty, fetches the latest; otherwise fetches the specified date (YYYYMMDD).
func FetchDelegatedResult(ctx context.Context, c *transport.Client, date string) (*models.DelegatedResult, error) {
	url := transport.BuildStatsURL(c.StatsBaseURL(), "delegated", date)
	r, err := c.FetchReader(ctx, url)
	if err != nil {
		return nil, err
	}
	return ParseDelegatedFull(r)
}

// FetchDelegatedResultByYear fetches the delegated stats for the last day of the given year.
// The file is served from the {year}/ archive subdirectory as a gzip-compressed file.
func FetchDelegatedResultByYear(ctx context.Context, c *transport.Client, year int) (*models.DelegatedResult, error) {
	url := fmt.Sprintf("%s%d/delegated-apnic-%d1231.gz", c.StatsBaseURL(), year, year)
	r, err := c.FetchReader(ctx, url)
	if err != nil {
		return nil, err
	}
	return ParseDelegatedFull(r)
}

// ParseDelegatedFull parses the complete delegated stats file including header, summaries, and entries.
func ParseDelegatedFull(r io.Reader) (*models.DelegatedResult, error) {
	scanner := bufio.NewScanner(r)
	result := &models.DelegatedResult{
		Summaries: make([]models.StatsSummary, 0),
		Entries:   make([]models.DelegatedEntry, 0, 5000),
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

		// Parse data line
		parts := strings.Split(line, "|")
		if len(parts) < 7 {
			continue
		}

		entry := models.DelegatedEntry{
			Registry:   parts[0],
			Country:    parts[1],
			Type:       parts[2],
			Start:      parts[3],
			Status:     parts[6],
			Extensions: parts[7:],
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

// ParseDelegatedFullFromString parses the complete delegated stats from a string.
func ParseDelegatedFullFromString(data string) (*models.DelegatedResult, error) {
	return ParseDelegatedFull(strings.NewReader(data))
}

// ParseDelegatedData parses only the data entries from a delegated stats file (legacy compatibility).
func ParseDelegatedData(r io.Reader) ([]models.DelegatedEntry, error) {
	result, err := ParseDelegatedFull(r)
	if err != nil {
		return nil, err
	}
	return result.Entries, nil
}
