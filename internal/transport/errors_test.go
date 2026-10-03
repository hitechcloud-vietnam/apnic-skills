package transport

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrors(t *testing.T) {
	errList := []error{
		ErrInvalidIP,
		ErrInvalidASN,
		ErrUnsupportedType,
		ErrWhoisTimeout,
		ErrCacheExpired,
		ErrRDAPQueryFailed,
		ErrInvalidDate,
		ErrInvalidCIDR,
		ErrTransferParseFail,
		ErrChangesParseFail,
		ErrVerifyFailed,
		ErrNotFound,
		ErrInvalidYear,
		ErrInvalidStatsType,
	}

	for _, err := range errList {
		if err == nil {
			t.Error("error should not be nil")
		}
		if err.Error() == "" {
			t.Error("error message should not be empty")
		}
	}
}

func TestErrorWrapping(t *testing.T) {
	wrapped := fmt.Errorf("%w: test", ErrInvalidIP)
	if !errors.Is(wrapped, ErrInvalidIP) {
		t.Error("expected error to wrap ErrInvalidIP")
	}
}
