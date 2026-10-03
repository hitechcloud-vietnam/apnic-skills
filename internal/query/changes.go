package query

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// FetchChanges fetches the latest resource change records from APNIC.
func FetchChanges(ctx context.Context, c *transport.Client) (*models.ChangesResult, error) {
	url := c.StatsBaseURL() + "changes/changes_latest.json"
	body, err := c.FetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseChangesData(body)
}

// FetchChangesByDate fetches change records for a specific date.
// date must be in YYYYMMDD format.
func FetchChangesByDate(ctx context.Context, c *transport.Client, date string) (*models.ChangesResult, error) {
	url := fmt.Sprintf("%schanges/%s/changes_%s.json", c.StatsBaseURL(), date[:4], date)
	body, err := c.FetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseChangesData(body)
}

// changesMetadataJSON represents the first line (metadata) of the changes JSON Lines file.
type changesMetadataJSON struct {
	Count      int64  `json:"count"`
	StatsBegin string `json:"stats-begin"`
	StatsEnd   string `json:"stats-end"`
	Timestamp  string `json:"timestamp"`
	Version    string `json:"version"`
}

// changeRecordJSON represents a single change record in JSON format.
type changeRecordJSON struct {
	Country   string   `json:"cc"`
	Custodian string   `json:"custodian"`
	Resources []string `json:"resources"`
	Status    string   `json:"status"`
	Timestamp string   `json:"timestamp"`
	Type      string   `json:"type"`
}

// parseChangesData parses the JSON Lines changes data.
// The first line is metadata, subsequent lines are change records.
func parseChangesData(data string) (*models.ChangesResult, error) {
	result := &models.ChangesResult{
		Changes: make([]models.ChangeRecord, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(data))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// First non-empty line is metadata
		if lineNum == 1 {
			var meta changesMetadataJSON
			if err := json.Unmarshal([]byte(line), &meta); err != nil {
				return nil, fmt.Errorf("%w: metadata parse error: %v", transport.ErrChangesParseFail, err)
			}
			result.Metadata = models.ChangesMetadata{
				Count:      meta.Count,
				StatsBegin: meta.StatsBegin,
				StatsEnd:   meta.StatsEnd,
				Version:    meta.Version,
			}
			if t, err := time.Parse("2006-01-02 15:04:05", meta.Timestamp); err == nil {
				result.Metadata.Timestamp = t
			}
			continue
		}

		// Subsequent lines are change records
		var rec changeRecordJSON
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			// Skip malformed lines
			continue
		}

		record := models.ChangeRecord{
			Country:   rec.Country,
			Custodian: rec.Custodian,
			Resources: rec.Resources,
			Status:    rec.Status,
			Type:      rec.Type,
		}

		if t, err := time.Parse("2006-01-02T15:04:05", rec.Timestamp); err == nil {
			record.Timestamp = t
		} else if t, err := time.Parse(time.RFC3339, rec.Timestamp); err == nil {
			record.Timestamp = t
		}

		result.Changes = append(result.Changes, record)
	}

	return result, scanner.Err()
}
