package models

import "testing"

func TestDelegatedEntryCIDR(t *testing.T) {
	tests := []struct {
		entry    DelegatedEntry
		expected string
		hasErr   bool
	}{
		{DelegatedEntry{Type: "ipv4", Start: "1.1.1.0", Value: 256}, "1.1.1.0/24", false},
		{DelegatedEntry{Type: "ipv4", Start: "1.0.0.0", Value: 1024}, "1.0.0.0/22", false},
		{DelegatedEntry{Type: "ipv6", Start: "2001:240::", Value: 32}, "2001:240::/32", false},
		{DelegatedEntry{Type: "ipv4", Start: "1.0.0.0", Value: 0}, "", true},
		{DelegatedEntry{Type: "ipv4", Start: "1.0.0.0", Value: int64(1) << 33}, "", true},
		{DelegatedEntry{Type: "ipv6", Start: "2001::", Value: -1}, "", true},
		{DelegatedEntry{Type: "ipv6", Start: "2001::", Value: 129}, "", true},
		{DelegatedEntry{Type: "asn", Start: "13335"}, "", true},
		{DelegatedEntry{Type: "ipv4", Start: "10.0.0.0", Value: 1}, "10.0.0.0/32", false},
	}

	for _, tt := range tests {
		result, err := tt.entry.CIDR()
		if tt.hasErr {
			if err == nil {
				t.Errorf("CIDR() expected error for %+v", tt.entry)
			}
		} else {
			if err != nil {
				t.Errorf("CIDR() unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("CIDR() = %q, want %q", result, tt.expected)
			}
		}
	}
}

func TestDelegatedExtendedEntryCIDR(t *testing.T) {
	entry := DelegatedExtendedEntry{Type: "ipv4", Start: "1.1.1.0", Value: 256}
	cidr, err := entry.CIDR()
	if err != nil {
		t.Fatalf("CIDR() error: %v", err)
	}
	if cidr != "1.1.1.0/24" {
		t.Errorf("CIDR() = %q, want 1.1.1.0/24", cidr)
	}

	// Test IPv6 success path
	entry6 := DelegatedExtendedEntry{Type: "ipv6", Start: "2001:240::", Value: 32}
	cidr6, err := entry6.CIDR()
	if err != nil {
		t.Fatalf("CIDR() IPv6 error: %v", err)
	}
	if cidr6 != "2001:240::/32" {
		t.Errorf("CIDR() IPv6 = %q, want 2001:240::/32", cidr6)
	}
}

func TestLegacyEntryCIDR(t *testing.T) {
	entry := LegacyEntry{Type: "ipv4", Start: "128.134.0.0", Value: 65536}
	cidr, err := entry.CIDR()
	if err != nil {
		t.Fatalf("CIDR() error: %v", err)
	}
	if cidr != "128.134.0.0/16" {
		t.Errorf("CIDR() = %q, want 128.134.0.0/16", cidr)
	}

	// Test IPv6 success path
	entry6 := LegacyEntry{Type: "ipv6", Start: "2001:db8::", Value: 48}
	cidr6, err := entry6.CIDR()
	if err != nil {
		t.Fatalf("CIDR() IPv6 error: %v", err)
	}
	if cidr6 != "2001:db8::/48" {
		t.Errorf("CIDR() IPv6 = %q, want 2001:db8::/48", cidr6)
	}
}

func TestExtendedEntryCIDRErrors(t *testing.T) {
	tests := []struct {
		entry  DelegatedExtendedEntry
		hasErr bool
	}{
		{DelegatedExtendedEntry{Type: "ipv4", Value: 0}, true},
		{DelegatedExtendedEntry{Type: "ipv4", Value: int64(1) << 33}, true},
		{DelegatedExtendedEntry{Type: "ipv6", Value: -1}, true},
		{DelegatedExtendedEntry{Type: "ipv6", Value: 129}, true},
		{DelegatedExtendedEntry{Type: "asn"}, true},
	}

	for i, tt := range tests {
		_, err := tt.entry.CIDR()
		if tt.hasErr && err == nil {
			t.Errorf("test %d: expected error", i)
		}
	}
}

func TestLegacyEntryCIDRErrors(t *testing.T) {
	tests := []struct {
		entry  LegacyEntry
		hasErr bool
	}{
		{LegacyEntry{Type: "ipv4", Value: 0}, true},
		{LegacyEntry{Type: "ipv4", Value: int64(1) << 33}, true},
		{LegacyEntry{Type: "ipv6", Value: -1}, true},
		{LegacyEntry{Type: "ipv6", Value: 129}, true},
		{LegacyEntry{Type: "asn"}, true},
	}

	for i, tt := range tests {
		_, err := tt.entry.CIDR()
		if tt.hasErr && err == nil {
			t.Errorf("test %d: expected error", i)
		}
	}
}
