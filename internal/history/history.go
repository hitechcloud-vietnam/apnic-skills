package history

import (
	"context"
	"fmt"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/stats"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// FetchHistoricalDelegated fetches delegated stats for a specific date.
// date must be in YYYYMMDD format.
func FetchHistoricalDelegated(ctx context.Context, c *transport.Client, date string) (*models.DelegatedResult, error) {
	if len(date) != 8 {
		return nil, fmt.Errorf("%w: date must be in YYYYMMDD format, got %s", transport.ErrInvalidDate, date)
	}
	return stats.FetchDelegatedResult(ctx, c, date)
}

// FetchHistoricalExtended fetches extended delegated stats for a specific date.
// date must be in YYYYMMDD format.
func FetchHistoricalExtended(ctx context.Context, c *transport.Client, date string) (*models.ExtendedResult, error) {
	if len(date) != 8 {
		return nil, fmt.Errorf("%w: date must be in YYYYMMDD format, got %s", transport.ErrInvalidDate, date)
	}
	return stats.FetchExtendedResult(ctx, c, date)
}

// FetchHistoricalAssigned fetches assigned stats for a specific date.
// date must be in YYYYMMDD format.
func FetchHistoricalAssigned(ctx context.Context, c *transport.Client, date string) (*models.AssignedResult, error) {
	if len(date) != 8 {
		return nil, fmt.Errorf("%w: date must be in YYYYMMDD format, got %s", transport.ErrInvalidDate, date)
	}
	return stats.FetchAssignedResult(ctx, c, date)
}

// FetchHistoricalLegacy fetches legacy stats for a specific date.
// date must be in YYYYMMDD format.
func FetchHistoricalLegacy(ctx context.Context, c *transport.Client, date string) (*models.LegacyResult, error) {
	if len(date) != 8 {
		return nil, fmt.Errorf("%w: date must be in YYYYMMDD format, got %s", transport.ErrInvalidDate, date)
	}
	return stats.FetchLegacyResult(ctx, c, date)
}

// FetchDelegatedByYear fetches the delegated stats for the last day of the given year.
// year must be a valid year (2001 or later, as APNIC stats start from 2001).
// The file is served from the {year}/ archive subdirectory as a gzip-compressed file.
func FetchDelegatedByYear(ctx context.Context, c *transport.Client, year int) (*models.DelegatedResult, error) {
	if year < 2001 {
		return nil, fmt.Errorf("%w: year must be 2001 or later, got %d", transport.ErrInvalidYear, year)
	}
	url := fmt.Sprintf("%s%d/delegated-apnic-%d1231.gz", c.StatsBaseURL(), year, year)
	body, err := c.FetchTextStr(ctx, url)
	if err != nil {
		return nil, err
	}
	return stats.ParseDelegatedFullFromString(body)
}

// FetchExtendedByYear fetches the extended delegated stats for the last day of the given year.
// The file is served from the {year}/ archive subdirectory as a gzip-compressed file.
func FetchExtendedByYear(ctx context.Context, c *transport.Client, year int) (*models.ExtendedResult, error) {
	if year < 2001 {
		return nil, fmt.Errorf("%w: year must be 2001 or later, got %d", transport.ErrInvalidYear, year)
	}
	url := fmt.Sprintf("%s%d/delegated-apnic-extended-%d1231.gz", c.StatsBaseURL(), year, year)
	body, err := c.FetchTextStr(ctx, url)
	if err != nil {
		return nil, err
	}
	return stats.ParseExtendedFullFromString(body)
}

// ListAvailableYears returns the list of years for which historical data is available.
// APNIC provides stats data from 2001 onwards.
func ListAvailableYears() []int {
	currentYear := 2026 // Updated periodically; APNIC stats available from 2001
	years := make([]int, 0, currentYear-2001+1)
	for y := 2001; y <= currentYear; y++ {
		years = append(years, y)
	}
	return years
}
