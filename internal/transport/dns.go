package transport

import (
	"context"
	"net"
)

// defaultLookupAddr is the production reverse-DNS resolver: the standard
// library's net.Resolver. It is a named function (not an anonymous closure) so
// it has no unreachable closure body to cover.
func defaultLookupAddr(ctx context.Context, ip string) ([]string, error) {
	return (&net.Resolver{}).LookupAddr(ctx, ip)
}

// lookupAddr is the reverse-DNS resolution function used by ReverseDNS. It is
// indirected via a package-level variable so tests can inject a stub that
// returns controlled results (e.g. an empty slice with nil error, which the
// real net.Resolver does not produce for missing PTR records but which the
// reverse-dns CLI command must still handle defensively).
var lookupAddr = defaultLookupAddr

// SetLookupAddr overrides the reverse-DNS resolver used by ReverseDNS. It is
// intended for tests that need deterministic PTR results (including the
// empty-slice-with-nil-error case the real resolver does not produce). Passing
// nil restores the default net.Resolver-based lookup.
func SetLookupAddr(fn func(ctx context.Context, ip string) ([]string, error)) {
	if fn == nil {
		fn = defaultLookupAddr
	}
	lookupAddr = fn
}

// ReverseDNS performs a reverse DNS (PTR) lookup for the given IP address.
func (c *Client) ReverseDNS(ctx context.Context, ip string) ([]string, error) {
	return lookupAddr(ctx, ip)
}
