package query

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// fetchJSON performs an HTTP GET and decodes a JSON response into out. It is the
// shared transport for REx endpoints, which return application/json (or, on a
// parameter error, a short plain-text message that must surface as an error).
// It goes through DoHTTPRequest so stealth/rate-limit/jitter all apply. When
// stealth advertises Accept-Encoding: gzip and the server honours it, the body
// is decompressed here (Go's Transport does not auto-decompress when the header
// is set explicitly — the same pitfall fetchText handles for .gz archives).
func fetchJSON(ctx context.Context, c *transport.Client, requestURL, accept string, out interface{}) error {
	resp, err := c.DoHTTPRequest(ctx, "GET", requestURL, accept)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// REx returns short plain-text messages for client errors (e.g. missing
		// query params); include the body so the caller sees the reason.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("unexpected status code: %d for URL: %s: %s", resp.StatusCode, requestURL, strings.TrimSpace(string(body)))
	}

	body := resp.Body
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("gzip init failed: %w", err)
		}
		defer gz.Close()
		body = gz
	}

	dec := json.NewDecoder(body)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("JSON decode failed: %w", err)
	}
	return nil
}

// FetchRExUserNetwork calls the REx /v1/user-network endpoint. APNIC's REx
// service determines the caller's source IP (no parameters required) and
// returns the covering prefix, its origin ASN, and the registered economy code.
// This is the cross-RIR "which network am I in?" lookup.
func FetchRExUserNetwork(ctx context.Context, c *transport.Client) (*models.RExUserNetwork, error) {
	u := buildRExURL(c.RExBaseURL(), "user-network", nil)
	var res models.RExUserNetwork
	if err := fetchJSON(ctx, c, u, "application/json", &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// FetchRExResources calls the REx /v1/resources endpoint, returning a recent
// view of delegated resources across all RIRs (prefixes and ASNs) with their
// holder attribution. The server returns the most recently delegated resources
// (a bounded window, not the full 100k+ history — use FetchRExHoldersUniqueCount
// for aggregate scale). type may be empty for all kinds, or one of "ipv4",
// "ipv6", "asn" to filter server-side.
func FetchRExResources(ctx context.Context, c *transport.Client, resourceType string) (*models.RExResourcesResult, error) {
	q := url.Values{}
	if resourceType != "" {
		q.Set("type", resourceType)
	}
	u := buildRExURL(c.RExBaseURL(), "resources", q)
	var res models.RExResourcesResult
	if err := fetchJSON(ctx, c, u, "application/json", &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// FetchRExHolder calls the REx /v1/holder endpoint, aggregating every ASN and
// prefix held by one organisation. opaqueID is the holder's opaque identifier
// (available from RExResource.OpaqueID or DelegatedExtendedEntry.OpaqueID);
// rir is the responsible RIR — one of "afrinic", "apnic", "arin", "lacnic",
// "ripencc" (note: the RIPE NCC code is "ripencc", not "ripe"). Both are
// required by the server.
func FetchRExHolder(ctx context.Context, c *transport.Client, opaqueID, rir string) (*models.RExHolder, error) {
	if opaqueID == "" || rir == "" {
		return nil, transport.ErrInvalidRExParam
	}
	q := url.Values{}
	q.Set("opaqueId", opaqueID)
	q.Set("rir", rir)
	u := buildRExURL(c.RExBaseURL(), "holder", q)
	var res models.RExHolder
	if err := fetchJSON(ctx, c, u, "application/json", &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// FetchRExHoldersUniqueCount calls the REx /v1/holders/unique-count endpoint,
// returning the total number of distinct resource-holder organisations across
// all RIRs. No parameters are required.
func FetchRExHoldersUniqueCount(ctx context.Context, c *transport.Client) (*models.RExHoldersCount, error) {
	u := buildRExURL(c.RExBaseURL(), "holders/unique-count", nil)
	var res models.RExHoldersCount
	if err := fetchJSON(ctx, c, u, "application/json", &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// buildRExURL constructs a REx API URL from the configured base URL, an endpoint
// path, and optional query parameters. Leading/trailing slashes on the path are
// normalised so callers may pass "user-network" or "/user-network".
func buildRExURL(baseURL, path string, q url.Values) string {
	path = strings.Trim(path, "/")
	u := strings.TrimRight(baseURL, "/") + "/v1/" + path
	if encoded := q.Encode(); encoded != "" {
		u += "?" + encoded
	}
	return u
}

// isRExAPIError reports whether an error returned by a REx method originated
// from the REx server's plain-text client-error body (e.g. a missing required
// query parameter). Callers can use this to distinguish bad input from
// transport failures.
func isRExAPIError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, transport.ErrInvalidRExParam) ||
		strings.Contains(err.Error(), "unexpected status code")
}
