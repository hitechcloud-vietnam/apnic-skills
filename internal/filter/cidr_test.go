package filter

import (
	"testing"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
)

func TestFilterEntries(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Country: "CN", Type: "ipv4", Start: "1.0.1.0", Value: 256, Status: "allocated"},
		{Country: "AU", Type: "ipv4", Start: "1.0.0.0", Value: 256, Status: "assigned"},
		{Country: "CN", Type: "ipv6", Start: "2001:240::", Value: 32, Status: "allocated"},
		{Country: "JP", Type: "asn", Start: "173", Value: 1, Status: "allocated"},
	}

	cn := FilterEntries(entries, "CN", "")
	if len(cn) != 2 {
		t.Errorf("CN entries = %d, want 2", len(cn))
	}

	ipv4 := FilterEntries(entries, "", "ipv4")
	if len(ipv4) != 2 {
		t.Errorf("ipv4 entries = %d, want 2", len(ipv4))
	}

	cnIPv4 := FilterEntries(entries, "CN", "ipv4")
	if len(cnIPv4) != 1 {
		t.Errorf("CN ipv4 entries = %d, want 1", len(cnIPv4))
	}

	all := FilterEntries(entries, "", "")
	if len(all) != 4 {
		t.Errorf("all entries = %d, want 4", len(all))
	}
}

func TestFilterByStatus(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Country: "CN", Type: "ipv4", Status: "allocated"},
		{Country: "AU", Type: "ipv4", Status: "assigned"},
		{Country: "JP", Type: "ipv4", Status: "allocated"},
	}

	allocated := FilterByStatus(entries, "allocated")
	if len(allocated) != 2 {
		t.Errorf("allocated entries = %d, want 2", len(allocated))
	}

	assigned := FilterByStatus(entries, "assigned")
	if len(assigned) != 1 {
		t.Errorf("assigned entries = %d, want 1", len(assigned))
	}

	all := FilterByStatus(entries, "")
	if len(all) != 3 {
		t.Errorf("all entries = %d, want 3", len(all))
	}
}

func TestFilterByDateRange(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Country: "AU", Date: time.Date(2011, 8, 11, 0, 0, 0, 0, time.UTC)},
		{Country: "CN", Date: time.Date(2011, 4, 14, 0, 0, 0, 0, time.UTC)},
		{Country: "JP", Date: time.Date(2002, 8, 1, 0, 0, 0, 0, time.UTC)},
		{Country: "US", Date: time.Time{}},                                 // zero date
		{Country: "UK", Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}, // after end
	}

	start := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC)

	filtered := FilterByDateRange(entries, start, end)
	if len(filtered) != 2 {
		t.Errorf("filtered entries = %d, want 2", len(filtered))
	}

	// Only start
	filteredStart := FilterByDateRange(entries, start, time.Time{})
	if len(filteredStart) != 3 {
		t.Errorf("filtered by start = %d, want 3", len(filteredStart))
	}

	// Only end
	filteredEnd := FilterByDateRange(entries, time.Time{}, end)
	if len(filteredEnd) != 3 {
		t.Errorf("filtered by end = %d, want 3", len(filteredEnd))
	}

	// Both zero - should include all non-zero dates
	filteredBoth := FilterByDateRange(entries, time.Time{}, time.Time{})
	if len(filteredBoth) != 4 {
		t.Errorf("filtered by both zero = %d, want 4", len(filteredBoth))
	}
}

func TestFilterExtendedByOpaqueID(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{OpaqueID: "A91872ED", Country: "AU"},
		{OpaqueID: "A92E1062", Country: "CN"},
		{OpaqueID: "A92E1062", Country: "CN"},
	}

	filtered := FilterExtendedByOpaqueID(entries, "A92E1062")
	if len(filtered) != 2 {
		t.Errorf("filtered = %d, want 2", len(filtered))
	}

	all := FilterExtendedByOpaqueID(entries, "")
	if len(all) != 3 {
		t.Errorf("all = %d, want 3", len(all))
	}
}

func TestFilterExtendedByCountry(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Country: "AU"},
		{Country: "CN"},
		{Country: "CN"},
	}

	filtered := FilterExtendedByCountry(entries, "CN")
	if len(filtered) != 2 {
		t.Errorf("filtered = %d, want 2", len(filtered))
	}
}

func TestFilterExtendedByType(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Type: "ipv4"},
		{Type: "ipv6"},
		{Type: "ipv4"},
	}

	filtered := FilterExtendedByType(entries, "ipv4")
	if len(filtered) != 2 {
		t.Errorf("filtered = %d, want 2", len(filtered))
	}
}

func TestFilterExtendedByStatus(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Status: "allocated"},
		{Status: "assigned"},
		{Status: "allocated"},
	}

	filtered := FilterExtendedByStatus(entries, "allocated")
	if len(filtered) != 2 {
		t.Errorf("filtered = %d, want 2", len(filtered))
	}
}

func TestGroupByCountry(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Country: "CN"},
		{Country: "AU"},
		{Country: "CN"},
	}

	grouped := GroupByCountry(entries)
	if len(grouped["CN"]) != 2 {
		t.Errorf("CN count = %d, want 2", len(grouped["CN"]))
	}
	if len(grouped["AU"]) != 1 {
		t.Errorf("AU count = %d, want 1", len(grouped["AU"]))
	}
}

func TestGroupExtendedByOpaqueID(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{OpaqueID: "A1"},
		{OpaqueID: "A2"},
		{OpaqueID: "A1"},
	}

	grouped := GroupExtendedByOpaqueID(entries)
	if len(grouped["A1"]) != 2 {
		t.Errorf("A1 count = %d, want 2", len(grouped["A1"]))
	}
}

func TestGroupExtendedByCountry(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Country: "CN"},
		{Country: "JP"},
		{Country: "CN"},
	}

	grouped := GroupExtendedByCountry(entries)
	if len(grouped["CN"]) != 2 {
		t.Errorf("CN count = %d, want 2", len(grouped["CN"]))
	}
}
