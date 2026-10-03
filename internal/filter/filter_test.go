package filter

import (
	"testing"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
)

func TestNewFilter(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Country: "CN", Type: "ipv4", Status: "allocated"},
		{Country: "AU", Type: "ipv4", Status: "assigned"},
		{Country: "JP", Type: "ipv6", Status: "allocated"},
	}

	f := NewFilter(entries)
	if f == nil {
		t.Error("expected non-nil filter")
	}
}

func TestEntryFilterByCountry(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Country: "CN"},
		{Country: "AU"},
		{Country: "CN"},
	}

	result := NewFilter(entries).ByCountry("CN").Result()
	if len(result) != 2 {
		t.Errorf("result count = %d, want 2", len(result))
	}
}

func TestEntryFilterByType(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Type: "ipv4"},
		{Type: "ipv6"},
		{Type: "ipv4"},
	}

	result := NewFilter(entries).ByType("ipv4").Result()
	if len(result) != 2 {
		t.Errorf("result count = %d, want 2", len(result))
	}
}

func TestEntryFilterByStatus(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Status: "allocated"},
		{Status: "assigned"},
		{Status: "allocated"},
	}

	result := NewFilter(entries).ByStatus("allocated").Result()
	if len(result) != 2 {
		t.Errorf("result count = %d, want 2", len(result))
	}
}

func TestEntryFilterByDateRange(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Time{}}, // zero date should be excluded
	}

	start := time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)

	result := NewFilter(entries).ByDateRange(start, end).Result()
	if len(result) != 1 {
		t.Errorf("result count = %d, want 1", len(result))
	}
}

func TestEntryFilterByDateRangeOnlyStart(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Time{}},
	}

	start := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	result := NewFilter(entries).ByDateRange(start, time.Time{}).Result()
	if len(result) != 2 {
		t.Errorf("result count = %d, want 2", len(result))
	}
}

func TestEntryFilterByDateRangeOnlyEnd(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Time{}},
	}

	end := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	result := NewFilter(entries).ByDateRange(time.Time{}, end).Result()
	if len(result) != 2 {
		t.Errorf("result count = %d, want 2", len(result))
	}
}

func TestEntryFilterByDateRangeBothZero(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Time{}},
	}

	result := NewFilter(entries).ByDateRange(time.Time{}, time.Time{}).Result()
	if len(result) != 2 {
		t.Errorf("result count = %d, want 2 (zero dates excluded)", len(result))
	}
}

func TestEntryFilterByRegistry(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Registry: "apnic"},
		{Registry: "arin"},
		{Registry: "apnic"},
	}

	result := NewFilter(entries).ByRegistry("apnic").Result()
	if len(result) != 2 {
		t.Errorf("result count = %d, want 2", len(result))
	}
}

func TestEntryFilterChained(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Country: "CN", Type: "ipv4", Status: "allocated"},
		{Country: "CN", Type: "ipv6", Status: "allocated"},
		{Country: "AU", Type: "ipv4", Status: "assigned"},
		{Country: "CN", Type: "ipv4", Status: "assigned"},
	}

	result := NewFilter(entries).
		ByCountry("CN").
		ByType("ipv4").
		ByStatus("allocated").
		Result()

	if len(result) != 1 {
		t.Errorf("result count = %d, want 1", len(result))
	}
}

func TestEntryFilterCount(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Country: "CN"},
		{Country: "AU"},
		{Country: "CN"},
	}

	count := NewFilter(entries).ByCountry("CN").Count()
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestEntryFilterEmptyResult(t *testing.T) {
	entries := []models.DelegatedEntry{
		{Country: "CN"},
	}

	result := NewFilter(entries).ByCountry("US").Result()
	if len(result) != 0 {
		t.Errorf("result count = %d, want 0", len(result))
	}
}

// ExtendedEntryFilter tests

func TestNewExtendedFilter(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Country: "CN", OpaqueID: "A1"},
	}
	f := NewExtendedFilter(entries)
	if f == nil {
		t.Error("expected non-nil filter")
	}
}

func TestExtendedFilterByCountry(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Country: "CN"},
		{Country: "AU"},
		{Country: "CN"},
	}

	result := NewExtendedFilter(entries).ByCountry("CN").Result()
	if len(result) != 2 {
		t.Errorf("result count = %d, want 2", len(result))
	}
}

func TestExtendedFilterByType(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Type: "ipv4"},
		{Type: "ipv6"},
		{Type: "ipv4"},
	}

	result := NewExtendedFilter(entries).ByType("ipv4").Result()
	if len(result) != 2 {
		t.Errorf("result count = %d, want 2", len(result))
	}
}

func TestExtendedFilterByStatus(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Status: "allocated"},
		{Status: "assigned"},
		{Status: "allocated"},
	}

	result := NewExtendedFilter(entries).ByStatus("allocated").Result()
	if len(result) != 2 {
		t.Errorf("result count = %d, want 2", len(result))
	}
}

func TestExtendedFilterByOpaqueID(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{OpaqueID: "A1"},
		{OpaqueID: "A2"},
		{OpaqueID: "A1"},
	}

	result := NewExtendedFilter(entries).ByOpaqueID("A1").Result()
	if len(result) != 2 {
		t.Errorf("result count = %d, want 2", len(result))
	}
}

func TestExtendedFilterByDateRange(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Time{}},
	}

	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	result := NewExtendedFilter(entries).ByDateRange(start, end).Result()
	if len(result) != 1 {
		t.Errorf("result count = %d, want 1", len(result))
	}
}

func TestExtendedFilterByDateRangeOnlyStart(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Time{}},
	}

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	result := NewExtendedFilter(entries).ByDateRange(start, time.Time{}).Result()
	if len(result) != 2 {
		t.Errorf("result count = %d, want 2", len(result))
	}
}

func TestExtendedFilterByDateRangeOnlyEnd(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Time{}},
	}

	end := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	result := NewExtendedFilter(entries).ByDateRange(time.Time{}, end).Result()
	if len(result) != 1 {
		t.Errorf("result count = %d, want 1", len(result))
	}
}

func TestExtendedFilterByDateRangeBothZero(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Time{}},
	}

	result := NewExtendedFilter(entries).ByDateRange(time.Time{}, time.Time{}).Result()
	if len(result) != 1 {
		t.Errorf("result count = %d, want 1 (zero dates excluded)", len(result))
	}
}

func TestEntryFilterByDateRangeBeforeStart(t *testing.T) {
	// Test the e.Date.Before(start) branch
	entries := []models.DelegatedEntry{
		{Date: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)}, // before start
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}, // in range
		{Date: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}, // after end
		{Date: time.Time{}}, // zero date
	}

	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	result := NewFilter(entries).ByDateRange(start, end).Result()
	if len(result) != 1 {
		t.Errorf("result count = %d, want 1", len(result))
	}
}

func TestExtendedFilterByDateRangeBeforeStart(t *testing.T) {
	// Test the e.Date.Before(start) branch for ExtendedEntryFilter
	entries := []models.DelegatedExtendedEntry{
		{Date: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)}, // before start
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}, // in range
		{Date: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}, // after end
		{Date: time.Time{}}, // zero date
	}

	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	result := NewExtendedFilter(entries).ByDateRange(start, end).Result()
	if len(result) != 1 {
		t.Errorf("result count = %d, want 1", len(result))
	}
}

func TestExtendedFilterChained(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Country: "CN", Type: "ipv4", OpaqueID: "A1"},
		{Country: "CN", Type: "ipv6", OpaqueID: "A2"},
		{Country: "AU", Type: "ipv4", OpaqueID: "A1"},
	}

	result := NewExtendedFilter(entries).
		ByCountry("CN").
		ByType("ipv4").
		Result()

	if len(result) != 1 {
		t.Errorf("result count = %d, want 1", len(result))
	}
}

func TestExtendedFilterCount(t *testing.T) {
	entries := []models.DelegatedExtendedEntry{
		{Country: "CN"},
		{Country: "AU"},
	}

	count := NewExtendedFilter(entries).ByCountry("CN").Count()
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}
