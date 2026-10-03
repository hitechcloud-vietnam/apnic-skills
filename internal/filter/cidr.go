package filter

import (
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
)

// FilterEntries filters delegated entries by country and resource type.
func FilterEntries(entries []models.DelegatedEntry, country, resType string) []models.DelegatedEntry {
	result := make([]models.DelegatedEntry, 0, len(entries))
	for _, e := range entries {
		if (country == "" || e.Country == country) &&
			(resType == "" || e.Type == resType) {
			result = append(result, e)
		}
	}
	return result
}

// FilterByStatus filters delegated entries by status (allocated, assigned, reserved, available).
func FilterByStatus(entries []models.DelegatedEntry, status string) []models.DelegatedEntry {
	result := make([]models.DelegatedEntry, 0, len(entries))
	for _, e := range entries {
		if status == "" || e.Status == status {
			result = append(result, e)
		}
	}
	return result
}

// FilterByDateRange filters delegated entries by date range (inclusive).
func FilterByDateRange(entries []models.DelegatedEntry, start, end time.Time) []models.DelegatedEntry {
	result := make([]models.DelegatedEntry, 0, len(entries))
	for _, e := range entries {
		if !e.Date.IsZero() {
			if !start.IsZero() && e.Date.Before(start) {
				continue
			}
			if !end.IsZero() && e.Date.After(end) {
				continue
			}
			result = append(result, e)
		}
	}
	return result
}

// FilterExtendedByOpaqueID filters extended entries by opaque-id (organization identifier).
func FilterExtendedByOpaqueID(entries []models.DelegatedExtendedEntry, opaqueID string) []models.DelegatedExtendedEntry {
	result := make([]models.DelegatedExtendedEntry, 0, len(entries))
	for _, e := range entries {
		if opaqueID == "" || e.OpaqueID == opaqueID {
			result = append(result, e)
		}
	}
	return result
}

// FilterExtendedByCountry filters extended entries by country code.
func FilterExtendedByCountry(entries []models.DelegatedExtendedEntry, country string) []models.DelegatedExtendedEntry {
	result := make([]models.DelegatedExtendedEntry, 0, len(entries))
	for _, e := range entries {
		if country == "" || e.Country == country {
			result = append(result, e)
		}
	}
	return result
}

// FilterExtendedByType filters extended entries by resource type.
func FilterExtendedByType(entries []models.DelegatedExtendedEntry, resType string) []models.DelegatedExtendedEntry {
	result := make([]models.DelegatedExtendedEntry, 0, len(entries))
	for _, e := range entries {
		if resType == "" || e.Type == resType {
			result = append(result, e)
		}
	}
	return result
}

// FilterExtendedByStatus filters extended entries by status.
func FilterExtendedByStatus(entries []models.DelegatedExtendedEntry, status string) []models.DelegatedExtendedEntry {
	result := make([]models.DelegatedExtendedEntry, 0, len(entries))
	for _, e := range entries {
		if status == "" || e.Status == status {
			result = append(result, e)
		}
	}
	return result
}

// GroupByCountry groups delegated entries by country code.
func GroupByCountry(entries []models.DelegatedEntry) map[string][]models.DelegatedEntry {
	result := make(map[string][]models.DelegatedEntry)
	for _, e := range entries {
		result[e.Country] = append(result[e.Country], e)
	}
	return result
}

// GroupExtendedByOpaqueID groups extended entries by opaque-id (organization).
func GroupExtendedByOpaqueID(entries []models.DelegatedExtendedEntry) map[string][]models.DelegatedExtendedEntry {
	result := make(map[string][]models.DelegatedExtendedEntry)
	for _, e := range entries {
		result[e.OpaqueID] = append(result[e.OpaqueID], e)
	}
	return result
}

// GroupExtendedByCountry groups extended entries by country code.
func GroupExtendedByCountry(entries []models.DelegatedExtendedEntry) map[string][]models.DelegatedExtendedEntry {
	result := make(map[string][]models.DelegatedExtendedEntry)
	for _, e := range entries {
		result[e.Country] = append(result[e.Country], e)
	}
	return result
}
