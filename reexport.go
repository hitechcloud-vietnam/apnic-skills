package apnic

import (
	"context"
	"net/http"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/filter"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/history"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// Client is the APNIC SDK client. It embeds *transport.Client, inheriting all
// transport-level methods (ReverseDNS, VerifyMD5, FetchText, DoHTTPRequest,
// etc.), and adds the higher-level query/stats/history methods that the
// subpackages expose as free functions.
type Client struct {
	*transport.Client
}

// Option is a functional option for configuring the Client. It receives the
// root *Client (the wrapper), so options that need to inspect the wrapper can;
// most options simply forward to the embedded *transport.Client.
type Option func(*Client)

// === const re-export (Go has no const alias; redeclare each) ===

const (
	DefaultStatsBaseURL = transport.DefaultStatsBaseURL
	DefaultRDAPBaseURL  = transport.DefaultRDAPBaseURL
	DefaultRRDPBaseURL  = transport.DefaultRRDPBaseURL
	DefaultThymeBaseURL = transport.DefaultThymeBaseURL
	DefaultFTPBaseURL   = transport.DefaultFTPBaseURL
	DefaultRExBaseURL   = transport.DefaultRExBaseURL
)

// === Type re-export (type alias, zero runtime cost) ===

type (
	// transport / models
	RDAPNetwork            = models.RDAPNetwork
	DelegatedEntry         = models.DelegatedEntry
	DelegatedExtendedEntry = models.DelegatedExtendedEntry

	// filter
	EntryFilter         = filter.EntryFilter
	ExtendedEntryFilter = filter.ExtendedEntryFilter
)

// NewClient creates a new APNIC client with the given options.
func NewClient(opts ...Option) *Client {
	c := &Client{transport.NewClient()}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewFilter creates a chainable filter for DelegatedEntry slices.
func NewFilter(entries []DelegatedEntry) *EntryFilter {
	return filter.NewFilter(entries)
}

// NewExtendedFilter creates a chainable filter for DelegatedExtendedEntry slices.
func NewExtendedFilter(entries []DelegatedExtendedEntry) *ExtendedEntryFilter {
	return filter.NewExtendedFilter(entries)
}

// ListAvailableYears returns the list of years for which historical stats exist.
func ListAvailableYears() []int { return history.ListAvailableYears() }

// SetLookupAddr overrides the reverse-DNS resolver (test/diagnostic hook).
func SetLookupAddr(fn func(ctx context.Context, ip string) ([]string, error)) {
	transport.SetLookupAddr(fn)
}

// === Option factory re-exports ===
//
// Each factory returns an Option (*Client -> void) that forwards to the
// matching transport.Option, applied to the embedded *transport.Client.

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { transport.WithHTTPClient(hc)(c.Client) }
}

func WithCacheTTL(ttl time.Duration) Option {
	return func(c *Client) { transport.WithCacheTTL(ttl)(c.Client) }
}

func WithUserAgent(ua string) Option {
	return func(c *Client) { transport.WithUserAgent(ua)(c.Client) }
}

func WithRDAPBaseURL(url string) Option {
	return func(c *Client) { transport.WithRDAPBaseURL(url)(c.Client) }
}

func WithWhoisServer(server string) Option {
	return func(c *Client) { transport.WithWhoisServer(server)(c.Client) }
}

func WithStatsBaseURL(url string) Option {
	return func(c *Client) { transport.WithStatsBaseURL(url)(c.Client) }
}

func WithRDAPDate(t time.Time) Option {
	return func(c *Client) { transport.WithRDAPDate(t)(c.Client) }
}

func WithStealth(enable bool) Option {
	return func(c *Client) { transport.WithStealth(enable)(c.Client) }
}

func WithBrowserUserAgent(ua string) Option {
	return func(c *Client) { transport.WithBrowserUserAgent(ua)(c.Client) }
}

func WithJitter(min, max time.Duration) Option {
	return func(c *Client) { transport.WithJitter(min, max)(c.Client) }
}

func WithRateLimit(perSecond float64) Option {
	return func(c *Client) { transport.WithRateLimit(perSecond)(c.Client) }
}

func WithRRDPBaseURL(url string) Option {
	return func(c *Client) { transport.WithRRDPBaseURL(url)(c.Client) }
}

func WithThymeBaseURL(url string) Option {
	return func(c *Client) { transport.WithThymeBaseURL(url)(c.Client) }
}

func WithFTPBaseURL(url string) Option {
	return func(c *Client) { transport.WithFTPBaseURL(url)(c.Client) }
}

func WithRExBaseURL(url string) Option {
	return func(c *Client) { transport.WithRExBaseURL(url)(c.Client) }
}

func WithMaxConcurrentDownloads(n int) Option {
	return func(c *Client) { transport.WithMaxConcurrentDownloads(n)(c.Client) }
}

func WithChunkSize(bytes int64) Option {
	return func(c *Client) { transport.WithChunkSize(bytes)(c.Client) }
}

func WithDownloadTimeout(d time.Duration) Option {
	return func(c *Client) { transport.WithDownloadTimeout(d)(c.Client) }
}

func WithWhoisTimeout(timeout time.Duration) Option {
	return func(c *Client) { transport.WithWhoisTimeout(timeout)(c.Client) }
}
