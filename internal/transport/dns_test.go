package transport

import (
	"context"
	"errors"
	"testing"
)

// withLookupAddr replaces the package-level lookupAddr for the duration of a
// test and restores it on return.
func withLookupAddr(t *testing.T, fn func(ctx context.Context, ip string) ([]string, error)) {
	t.Helper()
	SetLookupAddr(fn)
	t.Cleanup(func() { SetLookupAddr(nil) })
}

func TestReverseDNS(t *testing.T) {
	withLookupAddr(t, func(ctx context.Context, ip string) ([]string, error) {
		if ip != "1.1.1.1" {
			t.Errorf("LookupAddr received ip=%q, want 1.1.1.1", ip)
		}
		return []string{"one.one.one.one."}, nil
	})
	client := NewClient()
	names, err := client.ReverseDNS(context.Background(), "1.1.1.1")
	if err != nil {
		t.Fatalf("ReverseDNS() error: %v", err)
	}
	if len(names) != 1 || names[0] != "one.one.one.one." {
		t.Errorf("names = %v", names)
	}
}

func TestReverseDNSEmpty(t *testing.T) {
	// A successful lookup that returns no PTR records (empty slice, nil error).
	withLookupAddr(t, func(ctx context.Context, ip string) ([]string, error) {
		return []string{}, nil
	})
	client := NewClient()
	names, err := client.ReverseDNS(context.Background(), "192.0.2.1")
	if err != nil {
		t.Fatalf("ReverseDNS() error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty names, got %v", names)
	}
}

func TestReverseDNSError(t *testing.T) {
	wantErr := errors.New("lookup failed")
	withLookupAddr(t, func(ctx context.Context, ip string) ([]string, error) {
		return nil, wantErr
	})
	client := NewClient()
	if _, err := client.ReverseDNS(context.Background(), "1.1.1.1"); err != wantErr {
		t.Errorf("ReverseDNS() error = %v, want %v", err, wantErr)
	}
}

// TestSetLookupAddr_RestoreDefault covers the nil-restore branch of
// SetLookupAddr: passing nil must install a working default resolver so that a
// subsequent ReverseDNS call still resolves without panicking.
func TestSetLookupAddr_RestoreDefault(t *testing.T) {
	// Inject a stub, then restore the default.
	SetLookupAddr(func(ctx context.Context, ip string) ([]string, error) {
		return []string{"stub"}, nil
	})
	SetLookupAddr(nil) // restores the net.Resolver-based default
	client := NewClient()
	// Call through the restored default resolver. We do not assert on the
	// result (DNS may be unavailable in CI); we only exercise the default
	// closure's body so it is covered.
	_, _ = client.ReverseDNS(context.Background(), "127.0.0.1")
}
