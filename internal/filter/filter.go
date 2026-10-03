package filter

import (
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
)

// EntryFilter provides a chainable filter API for DelegatedEntry slices.
type EntryFilter struct {
	entries []models.DelegatedEntry
}

// NewFilter creates a new EntryFilter with the given entries.
func NewFilter(entries []models.DelegatedEntry) *EntryFilter {
	return &EntryFilter{entries: entries}
}

// ByCountry filters entries by ISO 3166 country code.
func (f *EntryFilter) ByCountry(country string) *EntryFilter {
	result := make([]models.DelegatedEntry, 0, len(f.entries))
	for _, e := range f.entries {
		if e.Country == country {
			result = append(result, e)
		}
	}
	f.entries = result
	return f
}

// ByType filters entries by resource type (ipv4, ipv6, asn).
func (f *EntryFilter) ByType(resType string) *EntryFilter {
	result := make([]models.DelegatedEntry, 0, len(f.entries))
	for _, e := range f.entries {
		if e.Type == resType {
			result = append(result, e)
		}
	}
	f.entries = result
	return f
}

// ByStatus filters entries by status (allocated, assigned, reserved, available).
func (f *EntryFilter) ByStatus(status string) *EntryFilter {
	result := make([]models.DelegatedEntry, 0, len(f.entries))
	for _, e := range f.entries {
		if e.Status == status {
			result = append(result, e)
		}
	}
	f.entries = result
	return f
}

// ByDateRange filters entries by date range (inclusive on both ends).
func (f *EntryFilter) ByDateRange(start, end time.Time) *EntryFilter {
	result := make([]models.DelegatedEntry, 0, len(f.entries))
	for _, e := range f.entries {
		if e.Date.IsZero() {
			continue
		}
		if !start.IsZero() && e.Date.Before(start) {
			continue
		}
		if !end.IsZero() && e.Date.After(end) {
			continue
		}
		result = append(result, e)
	}
	f.entries = result
	return f
}

// ByRegistry filters entries by registry name.
func (f *EntryFilter) ByRegistry(registry string) *EntryFilter {
	result := make([]models.DelegatedEntry, 0, len(f.entries))
	for _, e := range f.entries {
		if e.Registry == registry {
			result = append(result, e)
		}
	}
	f.entries = result
	return f
}

// Result returns the filtered entries.
func (f *EntryFilter) Result() []models.DelegatedEntry {
	return f.entries
}

// Count returns the number of entries after filtering.
func (f *EntryFilter) Count() int {
	return len(f.entries)
}

// ExtendedEntryFilter provides a chainable filter API for DelegatedExtendedEntry slices.
type ExtendedEntryFilter struct {
	entries []models.DelegatedExtendedEntry
}

// NewExtendedFilter creates a new ExtendedEntryFilter with the given entries.
func NewExtendedFilter(entries []models.DelegatedExtendedEntry) *ExtendedEntryFilter {
	return &ExtendedEntryFilter{entries: entries}
}

// ByCountry filters entries by ISO 3166 country code.
func (f *ExtendedEntryFilter) ByCountry(country string) *ExtendedEntryFilter {
	result := make([]models.DelegatedExtendedEntry, 0, len(f.entries))
	for _, e := range f.entries {
		if e.Country == country {
			result = append(result, e)
		}
	}
	f.entries = result
	return f
}

// ByType filters entries by resource type (ipv4, ipv6, asn).
func (f *ExtendedEntryFilter) ByType(resType string) *ExtendedEntryFilter {
	result := make([]models.DelegatedExtendedEntry, 0, len(f.entries))
	for _, e := range f.entries {
		if e.Type == resType {
			result = append(result, e)
		}
	}
	f.entries = result
	return f
}

// ByStatus filters entries by status.
func (f *ExtendedEntryFilter) ByStatus(status string) *ExtendedEntryFilter {
	result := make([]models.DelegatedExtendedEntry, 0, len(f.entries))
	for _, e := range f.entries {
		if e.Status == status {
			result = append(result, e)
		}
	}
	f.entries = result
	return f
}

// ByOpaqueID filters entries by opaque-id (organization identifier).
func (f *ExtendedEntryFilter) ByOpaqueID(opaqueID string) *ExtendedEntryFilter {
	result := make([]models.DelegatedExtendedEntry, 0, len(f.entries))
	for _, e := range f.entries {
		if e.OpaqueID == opaqueID {
			result = append(result, e)
		}
	}
	f.entries = result
	return f
}

// ByDateRange filters entries by date range (inclusive on both ends).
func (f *ExtendedEntryFilter) ByDateRange(start, end time.Time) *ExtendedEntryFilter {
	result := make([]models.DelegatedExtendedEntry, 0, len(f.entries))
	for _, e := range f.entries {
		if e.Date.IsZero() {
			continue
		}
		if !start.IsZero() && e.Date.Before(start) {
			continue
		}
		if !end.IsZero() && e.Date.After(end) {
			continue
		}
		result = append(result, e)
	}
	f.entries = result
	return f
}

// Result returns the filtered entries.
func (f *ExtendedEntryFilter) Result() []models.DelegatedExtendedEntry {
	return f.entries
}

// Count returns the number of entries after filtering.
func (f *ExtendedEntryFilter) Count() int {
	return len(f.entries)
}
