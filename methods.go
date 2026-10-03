package apnic

import (
	"context"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/history"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/query"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/stats"
)

// This file adds the higher-level query/stats/history methods to the root
// apnic.Client (which embeds *transport.Client). The transport package cannot
// define these methods itself because it would then import the stats/query
// subpackages, which in turn import transport — an import cycle. Keeping the
// wrappers in the root package breaks the cycle: apnic imports both transport
// and stats/query, but neither stats nor query imports apnic.

// --- stats: delegated ---

func (c *Client) FetchDelegatedEntries(ctx context.Context) ([]models.DelegatedEntry, error) {
	return stats.FetchDelegatedEntries(ctx, c.Client)
}

func (c *Client) FetchDelegatedEntriesByDate(ctx context.Context, date string) ([]models.DelegatedEntry, error) {
	return stats.FetchDelegatedEntriesByDate(ctx, c.Client, date)
}

func (c *Client) FetchDelegatedResult(ctx context.Context, date string) (*models.DelegatedResult, error) {
	return stats.FetchDelegatedResult(ctx, c.Client, date)
}

func (c *Client) FetchDelegatedResultByYear(ctx context.Context, year int) (*models.DelegatedResult, error) {
	return stats.FetchDelegatedResultByYear(ctx, c.Client, year)
}

// --- stats: extended ---

func (c *Client) FetchExtendedResult(ctx context.Context, date string) (*models.ExtendedResult, error) {
	return stats.FetchExtendedResult(ctx, c.Client, date)
}

// --- stats: assigned / ipv6-assigned / legacy ---

func (c *Client) FetchAssignedResult(ctx context.Context, date string) (*models.AssignedResult, error) {
	return stats.FetchAssignedResult(ctx, c.Client, date)
}

func (c *Client) FetchIPv6AssignedResult(ctx context.Context, date string) (*models.IPv6AssignedResult, error) {
	return stats.FetchIPv6AssignedResult(ctx, c.Client, date)
}

func (c *Client) FetchLegacyResult(ctx context.Context, date string) (*models.LegacyResult, error) {
	return stats.FetchLegacyResult(ctx, c.Client, date)
}

// --- history ---

func (c *Client) FetchHistoricalDelegated(ctx context.Context, date string) (*models.DelegatedResult, error) {
	return history.FetchHistoricalDelegated(ctx, c.Client, date)
}

func (c *Client) FetchHistoricalExtended(ctx context.Context, date string) (*models.ExtendedResult, error) {
	return history.FetchHistoricalExtended(ctx, c.Client, date)
}

func (c *Client) FetchHistoricalAssigned(ctx context.Context, date string) (*models.AssignedResult, error) {
	return history.FetchHistoricalAssigned(ctx, c.Client, date)
}

func (c *Client) FetchHistoricalLegacy(ctx context.Context, date string) (*models.LegacyResult, error) {
	return history.FetchHistoricalLegacy(ctx, c.Client, date)
}

func (c *Client) FetchDelegatedByYear(ctx context.Context, year int) (*models.DelegatedResult, error) {
	return history.FetchDelegatedByYear(ctx, c.Client, year)
}

func (c *Client) FetchExtendedByYear(ctx context.Context, year int) (*models.ExtendedResult, error) {
	return history.FetchExtendedByYear(ctx, c.Client, year)
}

// --- query: RDAP ---

func (c *Client) RDAPLookupIP(ctx context.Context, ip string) (*models.RDAPNetwork, error) {
	return query.RDAPLookupIP(ctx, c.Client, ip)
}

func (c *Client) RDAPLookupCIDR(ctx context.Context, cidr string) (*models.RDAPNetwork, error) {
	return query.RDAPLookupCIDR(ctx, c.Client, cidr)
}

func (c *Client) RDAPLookupASN(ctx context.Context, asn int64) (*models.RDAPAutnum, error) {
	return query.RDAPLookupASN(ctx, c.Client, asn)
}

func (c *Client) RDAPLookupDomain(ctx context.Context, domain string) (*models.RDAPDomain, error) {
	return query.RDAPLookupDomain(ctx, c.Client, domain)
}

func (c *Client) RDAPLookupEntity(ctx context.Context, handle string) (*models.RDAPEntity, error) {
	return query.RDAPLookupEntity(ctx, c.Client, handle)
}

func (c *Client) RDAPSearchEntities(ctx context.Context, field, q string) (*models.RDAPSearchResult, error) {
	return query.RDAPSearchEntities(ctx, c.Client, field, q)
}

func (c *Client) RDAPSearchDomains(ctx context.Context, name string) (*models.RDAPDomainSearchResult, error) {
	return query.RDAPSearchDomains(ctx, c.Client, name)
}

func (c *Client) RDAPHelp(ctx context.Context) (*models.RDAPHelpInfo, error) {
	return query.RDAPHelp(ctx, c.Client)
}

// --- query: REx ---

func (c *Client) FetchRExUserNetwork(ctx context.Context) (*models.RExUserNetwork, error) {
	return query.FetchRExUserNetwork(ctx, c.Client)
}

func (c *Client) FetchRExResources(ctx context.Context, resourceType string) (*models.RExResourcesResult, error) {
	return query.FetchRExResources(ctx, c.Client, resourceType)
}

func (c *Client) FetchRExHolder(ctx context.Context, opaqueID, rir string) (*models.RExHolder, error) {
	return query.FetchRExHolder(ctx, c.Client, opaqueID, rir)
}

func (c *Client) FetchRExHoldersUniqueCount(ctx context.Context) (*models.RExHoldersCount, error) {
	return query.FetchRExHoldersUniqueCount(ctx, c.Client)
}

// --- query: whois ---

func (c *Client) QueryWhois(ctx context.Context, q string) (string, error) {
	return query.QueryWhois(ctx, c.Client, q)
}

// QueryWhoisWithFlags performs a raw Whois query with additional flags prepended
// to the query (e.g. "-L" for all-less-specific). An empty flags string behaves
// identically to QueryWhois.
func (c *Client) QueryWhoisWithFlags(ctx context.Context, q string, flags string) (string, error) {
	return query.QueryWhoisWithFlags(ctx, c.Client, q, flags)
}

func (c *Client) QueryWhoisIP(ctx context.Context, ip string) (*models.WhoisInfo, error) {
	return query.QueryWhoisIP(ctx, c.Client, ip)
}

func (c *Client) QueryWhoisASN(ctx context.Context, asn int64) (*models.WhoisInfo, error) {
	return query.QueryWhoisASN(ctx, c.Client, asn)
}

// --- query: IRR ---

func (c *Client) FetchIRRCurrentSerial(ctx context.Context) (int64, error) {
	return query.FetchIRRCurrentSerial(ctx, c.Client)
}

// --- query: RRDP ---

func (c *Client) FetchRRDPNotification(ctx context.Context) (*models.RRDPNotification, error) {
	return query.FetchRRDPNotification(ctx, c.Client)
}

func (c *Client) FetchRRDPSnapshot(ctx context.Context, uri string) (*models.RPKISnapshot, error) {
	return query.FetchRRDPSnapshot(ctx, c.Client, uri)
}

// --- query: BGP ---

func (c *Client) FetchBGPSummary(ctx context.Context) (*models.BGPSummary, error) {
	return query.FetchBGPSummary(ctx, c.Client)
}

func (c *Client) FetchBGPRawTable(ctx context.Context) (*models.BGPRawTable, error) {
	return query.FetchBGPRawTable(ctx, c.Client)
}

func (c *Client) FetchBGPASNMap(ctx context.Context) (*models.BGPASNMap, error) {
	return query.FetchBGPASNMap(ctx, c.Client)
}

func (c *Client) FetchBGPBadPrefixes(ctx context.Context, source string) (*models.BGPBadPrefixes, error) {
	return query.FetchBGPBadPrefixes(ctx, c.Client, source)
}

func (c *Client) FetchBGPPerPrefixLength(ctx context.Context, source string) (*models.BGPPerPrefixLength, error) {
	return query.FetchBGPPerPrefixLength(ctx, c.Client, source)
}

func (c *Client) FetchBGPUsedAutnums(ctx context.Context, source string) (*models.BGPUsedAutnums, error) {
	return query.FetchBGPUsedAutnums(ctx, c.Client, source)
}

func (c *Client) FetchBGPSparPrefixes(ctx context.Context, source string) (*models.BGPSparPrefixes, error) {
	return query.FetchBGPSparPrefixes(ctx, c.Client, source)
}

func (c *Client) FetchBGPSinglePfx(ctx context.Context, source string) (*models.BGPSinglePfx, error) {
	return query.FetchBGPSinglePfx(ctx, c.Client, source)
}

// --- query: telemetry ---

func (c *Client) FetchTelemetry(ctx context.Context, date string) (*models.WhoisRDAPTelemetry, error) {
	return query.FetchTelemetry(ctx, c.Client, date)
}

// --- query: transfers ---

func (c *Client) FetchTransfers(ctx context.Context) (*models.TransfersResult, error) {
	return query.FetchTransfers(ctx, c.Client)
}

func (c *Client) FetchTransfersByYear(ctx context.Context, year int) (*models.TransfersResult, error) {
	return query.FetchTransfersByYear(ctx, c.Client, year)
}

func (c *Client) FetchTransfersAll(ctx context.Context, date string) (*models.TransfersAllResult, error) {
	return query.FetchTransfersAll(ctx, c.Client, date)
}

// --- query: changes ---

func (c *Client) FetchChanges(ctx context.Context) (*models.ChangesResult, error) {
	return query.FetchChanges(ctx, c.Client)
}

func (c *Client) FetchChangesByDate(ctx context.Context, date string) (*models.ChangesResult, error) {
	return query.FetchChangesByDate(ctx, c.Client, date)
}

// --- transport: verify (delegated to embedded *transport.Client via accessors) ---
// VerifyMD5 / FetchMD5Checksum / FetchASCSignature / FetchPublicKey / ReverseDNS
// are inherited from the embedded *transport.Client and need no wrapper here.
