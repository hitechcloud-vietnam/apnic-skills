package transport

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"time"
)

// dialFunc is the signature for dialing a network connection.
type dialFunc func(ctx context.Context, network, address string) (net.Conn, error)

// Default base URLs for APNIC data services.
const (
	DefaultStatsBaseURL = "https://ftp.apnic.net/apnic/stats/apnic/"
	DefaultRDAPBaseURL  = "https://rdap.apnic.net"
	DefaultRRDPBaseURL  = "https://rrdp.apnic.net"
	DefaultThymeBaseURL = "https://thyme.apnic.net"
	DefaultFTPBaseURL   = "https://ftp.apnic.net/"
	DefaultRExBaseURL   = "https://api.rex.apnic.net"
)

// Client is the APNIC SDK client that provides access to all APNIC services.
type Client struct {
	httpClient   *http.Client
	whoisServer  string
	whoisTimeout time.Duration
	rdapBaseURL  string
	statsBaseURL string // base URL for stats/FTP data, defaults to "https://ftp.apnic.net/apnic/stats/apnic/"
	cache        *cache
	userAgent    string    // used when stealth is disabled
	dialWhois    dialFunc  // optional override for whois dial, used in testing
	rdapDate     time.Time // optional point-in-time for RDAP historical queries (history_version_0); zero means live data

	// Stealth / anti-scraping fields.
	stealth     bool          // when true, send browser-mimicry headers + jitter (default true)
	browserUA   string        // User-Agent used when stealth is enabled
	jitterMin   time.Duration // min request jitter
	jitterMax   time.Duration // max request jitter
	rateLimiter *rateLimiter  // optional global token-bucket rate limiter
	rand        *randSource   // deterministic jitter source

	// Additional service base URLs for capabilities beyond core stats/RDAP.
	rrdpBaseURL  string // RPKI RRDP, default "https://rrdp.apnic.net"
	thymeBaseURL string // BGP routing analysis, default "https://thyme.apnic.net"
	thymeSource  string // thyme data source: "current" (default), "au", or "hk"
	ftpBaseURL   string // APNIC FTP root, default "https://ftp.apnic.net/"
	rexBaseURL   string // REx cross-RIR resource registry, default "https://api.rex.apnic.net"

	// Chunked-download configuration for large files (delegated/extended/IRR
	// dumps), which APNIC FTP throttles per-connection to ~8-18 KB/s. Multiple
	// parallel Range requests multiply throughput.
	downloadCfg downloadConfig
}

// Option is a functional option for configuring the Client.
type Option func(*Client)

// NewClient creates a new APNIC client with the given options.
func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		whoisServer:  "whois.apnic.net:43",
		whoisTimeout: 10 * time.Second,
		rdapBaseURL:  DefaultRDAPBaseURL,
		statsBaseURL: DefaultStatsBaseURL,
		cache:        newCache(30 * time.Minute),
		userAgent:    "APNIC-Go-SDK/1.0 (security)",
		stealth:      true,
		browserUA:    defaultBrowserUA,
		jitterMin:    200 * time.Millisecond,
		jitterMax:    800 * time.Millisecond,
		rrdpBaseURL:  DefaultRRDPBaseURL,
		thymeBaseURL: DefaultThymeBaseURL,
		thymeSource:  "current",
		ftpBaseURL:   DefaultFTPBaseURL,
		rexBaseURL:   DefaultRExBaseURL,
		downloadCfg: downloadConfig{
			maxConcurrent: 4,
			minSize:       512 * 1024, // 512KB — smaller files skip chunking
			targetChunk:   defaultTargetChunkSize,
		},
		rand: &randSource{r: rand.New(rand.NewSource(1))},
	}

	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithHTTPClient sets a custom HTTP client for API requests.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithCacheTTL sets the cache time-to-live duration.
func WithCacheTTL(ttl time.Duration) Option {
	return func(c *Client) {
		c.cache.ttl = ttl
	}
}

// WithUserAgent sets a custom User-Agent header for requests.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// WithRDAPBaseURL sets a custom base URL for RDAP queries.
func WithRDAPBaseURL(url string) Option {
	return func(c *Client) {
		c.rdapBaseURL = url
	}
}

// WithWhoisServer sets a custom Whois server address.
func WithWhoisServer(server string) Option {
	return func(c *Client) {
		c.whoisServer = server
	}
}

// WithWhoisTimeout sets the Whois connection timeout.
func WithWhoisTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.whoisTimeout = timeout
	}
}

// DialFunc is the exported signature for dialing a network connection. It is
// an alias of the unexported dialFunc so callers (mainly tests in other
// subpackages) can inject a custom whois dialer via WithDialWhois without
// duplicating the type.
type DialFunc = func(ctx context.Context, network, address string) (net.Conn, error)

// WithDialWhois sets a custom dial function for whois connections. It is the
// exported equivalent of the test-only withDialWhois helper, exposed so that
// subpackage tests can exercise whois dial-error branches.
func WithDialWhois(fn DialFunc) Option {
	return func(c *Client) {
		c.dialWhois = fn
	}
}

// WithStatsBaseURL sets a custom base URL for stats/FTP data requests.
func WithStatsBaseURL(url string) Option {
	return func(c *Client) {
		c.statsBaseURL = url
	}
}

// WithRDAPDate sets a default point-in-time timestamp for all RDAP queries.
// APNIC's RDAP service supports historical lookups (history_version_0) via a
// "date" query parameter; when set, queries return the resource state as it
// was at that UTC instant. Use the zero value (or do not call this option) for
// live, current-state queries. Individual *At methods override this per-call.
func WithRDAPDate(t time.Time) Option {
	return func(c *Client) {
		c.rdapDate = t
	}
}

// WithStealth enables or disables browser-mimicry request headers and request
// jitter (anti-scraping). Enabled by default. When disabled, only User-Agent
// and Accept headers are sent (pre-stealth behavior).
func WithStealth(enable bool) Option {
	return func(c *Client) {
		c.stealth = enable
	}
}

// WithBrowserUserAgent sets the User-Agent used when stealth is enabled. By
// default a mainstream Chrome UA is used; this option overrides it.
func WithBrowserUserAgent(ua string) Option {
	return func(c *Client) {
		if ua != "" {
			c.browserUA = ua
		}
	}
}

// WithJitter configures the random per-request delay range applied when stealth
// is enabled. If max < min, the values are silently swapped. A zero or negative
// min disables jitter.
func WithJitter(min, max time.Duration) Option {
	return func(c *Client) {
		if max < min {
			min, max = max, min
		}
		c.jitterMin = min
		c.jitterMax = max
	}
}

// WithRateLimit enables a global token-bucket rate limiter allowing perSecond
// requests per second (burst 1). A zero or negative value disables limiting.
func WithRateLimit(perSecond float64) Option {
	return func(c *Client) {
		c.rateLimiter = newRateLimiter(perSecond)
	}
}

// WithRRDPBaseURL sets the base URL for RPKI RRDP requests.
func WithRRDPBaseURL(url string) Option {
	return func(c *Client) {
		if url != "" {
			c.rrdpBaseURL = url
		}
	}
}

// WithThymeBaseURL sets the base URL for thyme BGP routing analysis requests.
func WithThymeBaseURL(url string) Option {
	return func(c *Client) {
		if url != "" {
			c.thymeBaseURL = url
		}
	}
}

// WithThymeSource sets the thyme data source: "current" (default, global view),
// "au" (Brisbane), or "hk" (HKIX). It applies to all thyme BGP requests that do
// not specify a source explicitly.
func WithThymeSource(source string) Option {
	return func(c *Client) {
		c.thymeSource = source
	}
}

// WithFTPBaseURL sets the APNIC FTP root URL used by capabilities whose data
// lives outside the stats subdirectory (IRR dumps, transfers-all, zones, lame,
// whois-rdap-stats telemetry).
func WithFTPBaseURL(url string) Option {
	return func(c *Client) {
		if url != "" {
			c.ftpBaseURL = url
		}
	}
}

// WithRExBaseURL sets the base URL for the APNIC REx cross-RIR resource
// registry REST API (api.rex.apnic.net).
func WithRExBaseURL(url string) Option {
	return func(c *Client) {
		if url != "" {
			c.rexBaseURL = url
		}
	}
}

// WithMaxConcurrentDownloads sets the number of parallel Range requests used to
// download large files that APNIC FTP throttles per-connection. Default 4. A
// value <= 1 disables chunked download (single-connection, legacy behavior).
func WithMaxConcurrentDownloads(n int) Option {
	return func(c *Client) {
		c.downloadCfg.maxConcurrent = n
	}
}

// WithChunkSize sets a target chunk size in bytes for chunked download. 0 (the
// default) means split the file evenly across MaxConcurrentDownloads connections.
func WithChunkSize(bytes int64) Option {
	return func(c *Client) {
		c.downloadCfg.chunkSize = bytes
	}
}

// WithDownloadTimeout sets the per-chunk request timeout for chunked downloads,
// independent of the global HTTP client timeout. 0 means inherit the client's
// httpClient.Timeout.
func WithDownloadTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.downloadCfg.timeout = d
	}
}

// --- Field accessors for subpackages ---
// The Client fields are unexported; subpackages (stats/query/history) reach
// them through these read-only accessors so the transport internals stay
// encapsulated while still allowing the restructured subpackages to build URLs
// and dial whois.

// StatsBaseURL returns the configured stats/FTP base URL.
func (c *Client) StatsBaseURL() string { return c.statsBaseURL }

// FTPBaseURL returns the configured APNIC FTP root URL.
func (c *Client) FTPBaseURL() string { return c.ftpBaseURL }

// ThymeBaseURL returns the configured thyme base URL.
func (c *Client) ThymeBaseURL() string { return c.thymeBaseURL }

// ThymeSource returns the configured default thyme data source.
func (c *Client) ThymeSource() string { return c.thymeSource }

// RRDPBaseURL returns the configured RRDP base URL.
func (c *Client) RRDPBaseURL() string { return c.rrdpBaseURL }

// RDAPBaseURL returns the configured RDAP base URL.
func (c *Client) RDAPBaseURL() string { return c.rdapBaseURL }

// RDAPDate returns the configured point-in-time RDAP date (zero = live).
func (c *Client) RDAPDate() time.Time { return c.rdapDate }

// RExBaseURL returns the configured REx cross-RIR base URL.
func (c *Client) RExBaseURL() string { return c.rexBaseURL }

// WhoisServer returns the configured whois server address.
func (c *Client) WhoisServer() string { return c.whoisServer }

// WhoisTimeout returns the configured whois connection timeout.
func (c *Client) WhoisTimeout() time.Duration { return c.whoisTimeout }

// DialWhois returns the optional whois dial override (nil in production).
func (c *Client) DialWhois() dialFunc { return c.dialWhois }
