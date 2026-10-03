package query

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// FetchTelemetry fetches the APNIC whois/RDAP service query telemetry
// (whois-rdap-stats.json), published hourly with per-query-type distribution
// and top-queried ASNs. date == "" fetches the latest; a YYYYMMDD date fetches
// the archived snapshot for that day.
func FetchTelemetry(ctx context.Context, c *transport.Client, date string) (*models.WhoisRDAPTelemetry, error) {
	url := transport.BuildTelemetryURL(c.FTPBaseURL(), date)
	body, err := c.FetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	var t models.WhoisRDAPTelemetry
	if err := json.Unmarshal([]byte(body), &t); err != nil {
		return nil, fmt.Errorf("telemetry JSON decode failed: %w", err)
	}
	return &t, nil
}

// FetchTelemetryMD5 fetches the MD5 checksum for the telemetry JSON.
func FetchTelemetryMD5(ctx context.Context, c *transport.Client, date string) (string, error) {
	url := transport.BuildTelemetrySidecarURL(c.FTPBaseURL(), date)
	content, err := c.FetchText(ctx, url)
	if err != nil {
		return "", err
	}
	return transport.ParseMD5Checksum(content)
}
