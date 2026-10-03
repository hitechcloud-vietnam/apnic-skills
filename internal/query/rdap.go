package query

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// RDAPLookupIP queries the RDAP service for an IP address.
// Returns structured information about the IP network including entities, status, and CIDR.
// If a point-in-time date was set via WithRDAPDate, the historical state is returned.
func RDAPLookupIP(ctx context.Context, c *transport.Client, ip string) (*models.RDAPNetwork, error) {
	return rdapLookupIPAt(ctx, c, ip, time.Time{})
}

// RDAPLookupIPAt queries the RDAP service for an IP address as it was at the given
// UTC instant (point-in-time / history_version_0). A zero time returns the live state.
func RDAPLookupIPAt(ctx context.Context, c *transport.Client, ip string, date time.Time) (*models.RDAPNetwork, error) {
	return rdapLookupIPAt(ctx, c, ip, date)
}

func rdapLookupIPAt(ctx context.Context, c *transport.Client, ip string, date time.Time) (*models.RDAPNetwork, error) {
	var result models.RDAPNetwork
	err := doRDAPRequestAt(ctx, c, "/ip/"+ip, &result, date)
	if err != nil {
		return nil, fmt.Errorf("%w: IP lookup for %s: %v", transport.ErrRDAPQueryFailed, ip, err)
	}
	return &result, nil
}

// RDAPLookupCIDR queries the RDAP service for a CIDR block.
// cidr should be in standard notation (e.g. "1.1.1.0/24").
func RDAPLookupCIDR(ctx context.Context, c *transport.Client, cidr string) (*models.RDAPNetwork, error) {
	return RDAPLookupCIDRAt(ctx, c, cidr, time.Time{})
}

// RDAPLookupCIDRAt queries the RDAP service for a CIDR block as it was at the given
// UTC instant (point-in-time). A zero time returns the live state.
func RDAPLookupCIDRAt(ctx context.Context, c *transport.Client, cidr string, date time.Time) (*models.RDAPNetwork, error) {
	var result models.RDAPNetwork
	err := doRDAPRequestAt(ctx, c, "/ip/"+cidr, &result, date)
	if err != nil {
		return nil, fmt.Errorf("%w: CIDR lookup for %s: %v", transport.ErrRDAPQueryFailed, cidr, err)
	}
	return &result, nil
}

// RDAPLookupASN queries the RDAP service for an Autonomous System Number.
// asn should be a plain number (e.g. 13335), not "AS13335".
func RDAPLookupASN(ctx context.Context, c *transport.Client, asn int64) (*models.RDAPAutnum, error) {
	return RDAPLookupASNAt(ctx, c, asn, time.Time{})
}

// RDAPLookupASNAt queries the RDAP service for an ASN as it was at the given
// UTC instant (point-in-time). A zero time returns the live state.
func RDAPLookupASNAt(ctx context.Context, c *transport.Client, asn int64, date time.Time) (*models.RDAPAutnum, error) {
	var result models.RDAPAutnum
	path := fmt.Sprintf("/autnum/%d", asn)
	err := doRDAPRequestAt(ctx, c, path, &result, date)
	if err != nil {
		return nil, fmt.Errorf("%w: ASN lookup for %d: %v", transport.ErrRDAPQueryFailed, asn, err)
	}
	return &result, nil
}

// RDAPLookupDomain queries the RDAP service for a domain object.
// Typically used for reverse DNS domains (e.g. "1.0.0.1.in-addr.arpa").
func RDAPLookupDomain(ctx context.Context, c *transport.Client, domain string) (*models.RDAPDomain, error) {
	return RDAPLookupDomainAt(ctx, c, domain, time.Time{})
}

// RDAPLookupDomainAt queries the RDAP service for a domain object as it was at the
// given UTC instant (point-in-time). A zero time returns the live state.
func RDAPLookupDomainAt(ctx context.Context, c *transport.Client, domain string, date time.Time) (*models.RDAPDomain, error) {
	var result models.RDAPDomain
	err := doRDAPRequestAt(ctx, c, "/domain/"+domain, &result, date)
	if err != nil {
		return nil, fmt.Errorf("%w: domain lookup for %s: %v", transport.ErrRDAPQueryFailed, domain, err)
	}
	return &result, nil
}

// RDAPLookupEntity queries the RDAP service for an entity (contact/organization).
// handle is the entity handle (e.g. "ORG-ARAD1-AP", "AIC3-AP").
func RDAPLookupEntity(ctx context.Context, c *transport.Client, handle string) (*models.RDAPEntity, error) {
	return RDAPLookupEntityAt(ctx, c, handle, time.Time{})
}

// RDAPLookupEntityAt queries the RDAP service for an entity as it was at the given
// UTC instant (point-in-time). A zero time returns the live state.
func RDAPLookupEntityAt(ctx context.Context, c *transport.Client, handle string, date time.Time) (*models.RDAPEntity, error) {
	var result models.RDAPEntity
	err := doRDAPRequestAt(ctx, c, "/entity/"+handle, &result, date)
	if err != nil {
		return nil, fmt.Errorf("%w: entity lookup for %s: %v", transport.ErrRDAPQueryFailed, handle, err)
	}
	return &result, nil
}

// RDAPSearch performs an entity name search against the APNIC RDAP database.
// APNIC does not provide a generic /search endpoint; search is performed via the
// RFC 7482 /entities endpoint using the "fn" (friendly name) query parameter.
// The query is matched against entity names; APNIC requires wildcard characters
// (e.g. "*CLOUD*") for substring matches — an exact name with no wildcard only
// matches an entity whose name is exactly that string.
// This is a convenience wrapper around RDAPSearchEntities with the "fn" field.
// For handle-based exact lookups, use RDAPLookupEntity instead.
func RDAPSearch(ctx context.Context, c *transport.Client, query string) (*models.RDAPSearchResult, error) {
	return RDAPSearchEntities(ctx, c, "fn", query)
}

// RDAPSearchEntities searches the APNIC RDAP entity database.
// field selects the search criterion: "fn" (fuzzy name match, supports "*"
// wildcards) or "handle" (exact entity handle match).
// Returns matching entities in EntitySearchResults.
func RDAPSearchEntities(ctx context.Context, c *transport.Client, field, query string) (*models.RDAPSearchResult, error) {
	if field != "fn" && field != "handle" {
		return nil, fmt.Errorf("%w: unsupported search field %q (use fn or handle)", transport.ErrRDAPQueryFailed, field)
	}
	if query == "" {
		return nil, fmt.Errorf("%w: empty search query", transport.ErrRDAPQueryFailed)
	}

	var result models.RDAPSearchResult
	path := fmt.Sprintf("/entities?%s=%s", field, url.QueryEscape(query))
	err := doRDAPRequestAt(ctx, c, path, &result, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("%w: entity search (%s=%s): %v", transport.ErrRDAPQueryFailed, field, query, err)
	}
	return &result, nil
}

// RDAPHelp queries the RDAP /help endpoint (RFC 7483), returning the server's
// capability description: rdapConformance extensions (e.g. history_version_0,
// cidr0, nro_rdap_profile_0), notices (terms of service, inaccuracy reporting),
// and port43. Useful for discovering which extensions the server supports.
func RDAPHelp(ctx context.Context, c *transport.Client) (*models.RDAPHelpInfo, error) {
	var result models.RDAPHelpInfo
	err := doRDAPRequestAt(ctx, c, "/help", &result, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("%w: help: %v", transport.ErrRDAPQueryFailed, err)
	}
	return &result, nil
}

// RDAPSearchDomains searches the APNIC RDAP database for reverse-DNS domain
// objects by name (RFC 7482 /domains?name=). Returns matching domains in
// DomainSearchResults. This is the domain analogue of RDAPSearchEntities.
func RDAPSearchDomains(ctx context.Context, c *transport.Client, name string) (*models.RDAPDomainSearchResult, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: empty domain search query", transport.ErrRDAPQueryFailed)
	}
	var result models.RDAPDomainSearchResult
	path := "/domains?name=" + url.QueryEscape(name)
	err := doRDAPRequestAt(ctx, c, path, &result, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("%w: domain search (name=%s): %v", transport.ErrRDAPQueryFailed, name, err)
	}
	return &result, nil
}

// doRDAPRequestAt performs a generic RDAP HTTP GET and decodes the JSON response.
// A non-zero date overrides the client default and adds a "date" query parameter
// for a point-in-time (history_version_0) lookup. A zero date falls back to the
// client's rdapDate, or a live query if that is also zero.
func doRDAPRequestAt(ctx context.Context, c *transport.Client, path string, result interface{}, date time.Time) error {
	effectiveDate := date
	if effectiveDate.IsZero() {
		effectiveDate = c.RDAPDate()
	}
	if !effectiveDate.IsZero() {
		sep := "&"
		if !strings.Contains(path, "?") {
			sep = "?"
		}
		path = path + sep + "date=" + url.QueryEscape(effectiveDate.UTC().Format(time.RFC3339))
	}

	requestURL := c.RDAPBaseURL() + path

	resp, err := c.DoHTTPRequest(ctx, "GET", requestURL, "application/rdap+json, application/json")
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	// Check for RDAP error responses
	if resp.StatusCode == http.StatusNotFound {
		var rdapErr models.RDAPError
		if json.Unmarshal(body, &rdapErr) == nil {
			return fmt.Errorf("%w: %s", transport.ErrNotFound, rdapErr.Title)
		}
		return transport.ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		var rdapErr models.RDAPError
		if json.Unmarshal(body, &rdapErr) == nil && rdapErr.Title != "" {
			return fmt.Errorf("RDAP error %d: %s", rdapErr.ErrorCode, rdapErr.Title)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("JSON decode failed: %w", err)
	}

	return nil
}
