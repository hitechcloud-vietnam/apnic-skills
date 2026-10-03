package apnic

import (
	"context"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/query"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/stats"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// GetDelegatedEntries returns cached delegated entries, fetching fresh data if expired.
func (c *Client) GetDelegatedEntries(ctx context.Context) ([]models.DelegatedEntry, error) {
	if data, ok := c.CacheGet(transport.CacheKeyDelegated); ok {
		return data.([]models.DelegatedEntry), nil
	}
	entries, err := stats.FetchDelegatedEntries(ctx, c.Client)
	if err != nil {
		return nil, err
	}
	c.CacheSet(transport.CacheKeyDelegated, entries)
	return entries, nil
}

// GetExtendedEntries returns cached extended delegated entries, fetching fresh data if expired.
func (c *Client) GetExtendedEntries(ctx context.Context) ([]models.DelegatedExtendedEntry, error) {
	if data, ok := c.CacheGet(transport.CacheKeyExtended); ok {
		return data.([]models.DelegatedExtendedEntry), nil
	}
	result, err := stats.FetchExtendedEntries(ctx, c.Client)
	if err != nil {
		return nil, err
	}
	c.CacheSet(transport.CacheKeyExtended, result.Entries)
	return result.Entries, nil
}

// GetAssignedEntries returns cached assigned entries, fetching fresh data if expired.
func (c *Client) GetAssignedEntries(ctx context.Context) ([]models.AssignedEntry, error) {
	if data, ok := c.CacheGet(transport.CacheKeyAssigned); ok {
		return data.([]models.AssignedEntry), nil
	}
	result, err := stats.FetchAssignedEntries(ctx, c.Client)
	if err != nil {
		return nil, err
	}
	c.CacheSet(transport.CacheKeyAssigned, result.Entries)
	return result.Entries, nil
}

// GetIPv6AssignedEntries returns cached IPv6 assigned entries, fetching fresh data if expired.
func (c *Client) GetIPv6AssignedEntries(ctx context.Context) ([]models.IPv6AssignedEntry, error) {
	if data, ok := c.CacheGet(transport.CacheKeyIPv6Assigned); ok {
		return data.([]models.IPv6AssignedEntry), nil
	}
	entries, err := stats.FetchIPv6AssignedEntries(ctx, c.Client)
	if err != nil {
		return nil, err
	}
	c.CacheSet(transport.CacheKeyIPv6Assigned, entries)
	return entries, nil
}

// GetLegacyEntries returns cached legacy entries, fetching fresh data if expired.
func (c *Client) GetLegacyEntries(ctx context.Context) ([]models.LegacyEntry, error) {
	if data, ok := c.CacheGet(transport.CacheKeyLegacy); ok {
		return data.([]models.LegacyEntry), nil
	}
	result, err := stats.FetchLegacyEntries(ctx, c.Client)
	if err != nil {
		return nil, err
	}
	c.CacheSet(transport.CacheKeyLegacy, result.Entries)
	return result.Entries, nil
}

// GetTransfers returns cached transfer records, fetching fresh data if expired.
func (c *Client) GetTransfers(ctx context.Context) (*models.TransfersResult, error) {
	if data, ok := c.CacheGet(transport.CacheKeyTransfers); ok {
		return data.(*models.TransfersResult), nil
	}
	result, err := query.FetchTransfers(ctx, c.Client)
	if err != nil {
		return nil, err
	}
	c.CacheSet(transport.CacheKeyTransfers, result)
	return result, nil
}

// GetChanges returns cached change records, fetching fresh data if expired.
func (c *Client) GetChanges(ctx context.Context) (*models.ChangesResult, error) {
	if data, ok := c.CacheGet(transport.CacheKeyChanges); ok {
		return data.(*models.ChangesResult), nil
	}
	result, err := query.FetchChanges(ctx, c.Client)
	if err != nil {
		return nil, err
	}
	c.CacheSet(transport.CacheKeyChanges, result)
	return result, nil
}

// GetIRRDatabase returns a cached IRR database for the given object type,
// fetching fresh data if expired or absent. objType must be a known IRR object
// type (see query.IRRObjectTypes).
func (c *Client) GetIRRDatabase(ctx context.Context, objType string) (*models.IRRDatabase, error) {
	key := transport.CacheKeyIRR(objType)
	if data, ok := c.CacheGet(key); ok {
		return data.(*models.IRRDatabase), nil
	}
	result, err := query.FetchIRRDatabase(ctx, c.Client, objType)
	if err != nil {
		return nil, err
	}
	c.CacheSet(key, result)
	return result, nil
}
