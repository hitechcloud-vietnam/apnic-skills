package models

import (
	"errors"
	"fmt"
	"math"
)

// Errors raised by the CIDR conversion helpers. They are local to the models
// package so models does not need to import transport (which would create an
// import cycle, since transport imports models).
var (
	errInvalidIPForCIDR    = errors.New("invalid IP address")
	errUnsupportedCIDRType = errors.New("unsupported resource type")
)

// CIDR converts a DelegatedEntry to CIDR notation.
func (e DelegatedEntry) CIDR() (string, error) {
	switch e.Type {
	case "ipv4":
		if e.Value <= 0 || e.Value > 1<<32 {
			return "", fmt.Errorf("%w: invalid IPv4 count %d", errInvalidIPForCIDR, e.Value)
		}
		prefix := 32 - int(math.Log2(float64(e.Value)))
		return fmt.Sprintf("%s/%d", e.Start, prefix), nil
	case "ipv6":
		if e.Value < 0 || e.Value > 128 {
			return "", fmt.Errorf("%w: invalid IPv6 prefix %d", errInvalidIPForCIDR, e.Value)
		}
		return fmt.Sprintf("%s/%d", e.Start, e.Value), nil
	default:
		return "", errUnsupportedCIDRType
	}
}

// CIDR converts a DelegatedExtendedEntry to CIDR notation.
func (e DelegatedExtendedEntry) CIDR() (string, error) {
	switch e.Type {
	case "ipv4":
		if e.Value <= 0 || e.Value > 1<<32 {
			return "", fmt.Errorf("%w: invalid IPv4 count %d", errInvalidIPForCIDR, e.Value)
		}
		prefix := 32 - int(math.Log2(float64(e.Value)))
		return fmt.Sprintf("%s/%d", e.Start, prefix), nil
	case "ipv6":
		if e.Value < 0 || e.Value > 128 {
			return "", fmt.Errorf("%w: invalid IPv6 prefix %d", errInvalidIPForCIDR, e.Value)
		}
		return fmt.Sprintf("%s/%d", e.Start, e.Value), nil
	default:
		return "", errUnsupportedCIDRType
	}
}

// CIDR converts a LegacyEntry to CIDR notation.
func (e LegacyEntry) CIDR() (string, error) {
	switch e.Type {
	case "ipv4":
		if e.Value <= 0 || e.Value > 1<<32 {
			return "", fmt.Errorf("%w: invalid IPv4 count %d", errInvalidIPForCIDR, e.Value)
		}
		prefix := 32 - int(math.Log2(float64(e.Value)))
		return fmt.Sprintf("%s/%d", e.Start, prefix), nil
	case "ipv6":
		if e.Value < 0 || e.Value > 128 {
			return "", fmt.Errorf("%w: invalid IPv6 prefix %d", errInvalidIPForCIDR, e.Value)
		}
		return fmt.Sprintf("%s/%d", e.Start, e.Value), nil
	default:
		return "", errUnsupportedCIDRType
	}
}
