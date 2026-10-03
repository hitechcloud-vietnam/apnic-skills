package transport

import "errors"

var (
	ErrInvalidIP         = errors.New("invalid IP address")
	ErrInvalidASN        = errors.New("invalid ASN format")
	ErrUnsupportedType   = errors.New("unsupported resource type")
	ErrWhoisTimeout      = errors.New("whois query timed out")
	ErrCacheExpired      = errors.New("cache expired")
	ErrRDAPQueryFailed   = errors.New("RDAP query failed")
	ErrInvalidDate       = errors.New("invalid date format")
	ErrInvalidCIDR       = errors.New("invalid CIDR notation")
	ErrTransferParseFail = errors.New("failed to parse transfer data")
	ErrChangesParseFail  = errors.New("failed to parse changes data")
	ErrVerifyFailed      = errors.New("data verification failed")
	ErrNotFound          = errors.New("resource not found")
	ErrInvalidYear       = errors.New("invalid year")
	ErrInvalidStatsType  = errors.New("invalid stats file type")
	ErrInvalidIRRType    = errors.New("invalid IRR object type")
	ErrInvalidRExParam   = errors.New("invalid REx query parameter")
)
