package models

import "time"

// DelegatedEntry represents a single allocation/assignment record from the standard delegated stats file.
type DelegatedEntry struct {
	Registry   string
	Country    string
	Type       string // ipv4, ipv6, asn
	Start      string
	Value      int64
	Date       time.Time
	Status     string
	Extensions []string
}

// DelegatedExtendedEntry represents a single allocation/assignment record from the extended delegated stats file.
// It includes the OpaqueID field that uniquely identifies the resource holder organization.
type DelegatedExtendedEntry struct {
	Registry   string
	Country    string
	Type       string // ipv4, ipv6, asn
	Start      string
	Value      int64
	Date       time.Time
	Status     string // available, allocated, assigned, reserved
	OpaqueID   string // unique identifier for the organization
	Extensions []string
}

// AssignedEntry represents an aggregated assignment count by prefix size from the assigned stats file.
type AssignedEntry struct {
	Registry string
	Country  string
	Type     string // ipv4, ipv6
	Prefix   string // prefix size (e.g. "256", "512")
	Count    int64  // number of assignments of this prefix size
	Status   string
}

// IPv6AssignedEntry represents a single IPv6 assignment record from the
// delegated-apnic-ipv6-assigned stats file.
// Unlike the standard delegated file, this file has no status or extension
// columns: registry|cc|ipv6|start|prefix|date
type IPv6AssignedEntry struct {
	Registry string
	Country  string
	Start    string
	Value    int64 // IPv6 prefix length
	Date     time.Time
}

// IPv6AssignedResult represents the full result of parsing a
// delegated-apnic-ipv6-assigned stats file.
type IPv6AssignedResult struct {
	Header    StatsFileHeader
	Summaries []StatsSummary
	Entries   []IPv6AssignedEntry
}

// LegacyEntry represents a historical (legacy) resource record from the legacy stats file.
type LegacyEntry struct {
	Registry string
	Country  string
	Type     string // ipv4, ipv6, asn
	Start    string
	Value    int64
	Date     time.Time
	Status   string
}

// TransferRecord represents a single IP/ASN transfer record.
type TransferRecord struct {
	TransferDate          time.Time
	Type                  string // RESOURCE_TRANSFER, INTER_RIR_TRANSFER
	SourceRIR             string
	RecipientRIR          string
	SourceOrganization    Organization
	RecipientOrganization Organization
	IPv4Nets              *TransferNetSet
	IPv6Nets              *TransferNetSet
	ASNs                  *TransferASNSet
}

// Organization represents an organization involved in a transfer.
type Organization struct {
	Name        string
	CountryCode string
}

// TransferNetSet represents a set of IP network ranges transferred.
type TransferNetSet struct {
	TransferSet []NetRange
}

// NetRange represents a range of IP addresses.
type NetRange struct {
	StartAddress string
	EndAddress   string
}

// TransferASNSet represents a set of ASN ranges transferred.
type TransferASNSet struct {
	TransferSet []ASNRange
}

// ASNRange represents a range of AS numbers.
type ASNRange struct {
	StartASN int64
	EndASN   int64
}

// ChangeRecord represents a single resource change record.
type ChangeRecord struct {
	Country   string
	Custodian string   // opaque-id
	Resources []string // CIDR or ASN values
	Status    string   // allocated, assigned
	Timestamp time.Time
	Type      string // delegated, cc-changed
}

// WhoisInfo represents parsed Whois response information.
// Fields are extracted from the primary object (inetnum/inet6num/aut-num/route)
// of an APNIC whois response, with CIDR and OriginASN supplemented from any
// route object. Empty fields mean the corresponding key was absent from the
// response (e.g. no route object → CIDR is nil). AbuseMailbox holds the email
// address from an "abuse-mailbox:" attribute (populated by the "-b" flag query),
// distinct from AbuseContact which holds the "abuse-c:" handle.
type WhoisInfo struct {
	Network      string
	NetName      string
	CIDR         []string
	Country      string
	OrgName      string
	Parent       string
	Status       string
	OriginASN    string
	AbuseContact string
	AbuseMailbox string
	Created      time.Time
	LastUpdated  time.Time
}

// StatsFileHeader represents the header line of a delegated stats file.
type StatsFileHeader struct {
	Version   string
	Registry  string
	Serial    int64
	Records   int64
	StartDate time.Time
	EndDate   time.Time
	UTCOffset int
}

// StatsSummary represents a summary line from a delegated stats file.
type StatsSummary struct {
	Registry string
	Type     string // asn, ipv4, ipv6
	Count    int64
}

// TransfersMetadata represents the metadata section of the transfers data.
type TransfersMetadata struct {
	Producer       string
	ProductionDate time.Time
	StatsVersion   string
	StartDate      time.Time
	EndDate        time.Time
}

// ChangesMetadata represents the metadata section of the changes data.
type ChangesMetadata struct {
	Count      int64
	StatsBegin string
	StatsEnd   string
	Timestamp  time.Time
	Version    string
}

// DelegatedResult represents the full result of parsing a delegated stats file,
// including header, summaries, and entries.
type DelegatedResult struct {
	Header    StatsFileHeader
	Summaries []StatsSummary
	Entries   []DelegatedEntry
}

// ExtendedResult represents the full result of parsing an extended delegated stats file.
type ExtendedResult struct {
	Header    StatsFileHeader
	Summaries []StatsSummary
	Entries   []DelegatedExtendedEntry
}

// AssignedResult represents the full result of parsing an assigned stats file.
type AssignedResult struct {
	Header    StatsFileHeader
	Summaries []StatsSummary
	Entries   []AssignedEntry
}

// LegacyResult represents the full result of parsing a legacy stats file.
type LegacyResult struct {
	Header    StatsFileHeader
	Summaries []StatsSummary
	Entries   []LegacyEntry
}

// TransfersResult represents the full result of parsing transfers data.
type TransfersResult struct {
	Metadata  TransfersMetadata
	Transfers []TransferRecord
}

// TransferAllRecord represents a single record in the cumulative transfers-all
// log (transfer-all-apnic-latest and per-year archives). Unlike TransferRecord
// (JSON), this is the historical pipe-delimited format covering all transfers
// since 2010.
type TransferAllRecord struct {
	ResourceType           string // asn | ipv4 | ipv6
	Resource               string // the ASN or prefix
	FromOrganisation       string
	FromEconomy            string
	FromRIR                string
	PreviousDelegationDate time.Time
	ToOrganisation         string
	ToEconomy              string
	ToRIR                  string
	TransferDate           time.Time
	TransferType           string // e.g. M&A, RESOURCE_TRANSFER
}

// TransfersAllResult represents the parsed cumulative transfers-all log.
type TransfersAllResult struct {
	Records []TransferAllRecord
}

// WhoisRDAPTelemetry represents the APNIC whois/RDAP service query telemetry
// (whois-rdap-stats.json), published hourly. It captures total query volume,
// per-type distribution, and top-queried ASNs.
type WhoisRDAPTelemetry struct {
	RDAP struct {
		DateRange struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"date_range"`
		TotalQueries          int64            `json:"total_queries"`
		TotalASNs             int64            `json:"total_asns"`
		QueryTypeDistribution map[string]int64 `json:"query_type_distribution"`
		ASNs                  []TelemetryASN   `json:"asns"`
	} `json:"RDAP"`
}

// TelemetryASN represents a top-queried ASN entry in the telemetry.
type TelemetryASN struct {
	ASN              string           `json:"asn"`
	QueryCount       int64            `json:"query_count"`
	QueryCountByType map[string]int64 `json:"query_count_by_type"`
}

// ChangesResult represents the full result of parsing changes data.
type ChangesResult struct {
	Metadata ChangesMetadata
	Changes  []ChangeRecord
}

// --- RDAP Models ---

// RDAPResponse represents the common fields in all RDAP responses.
type RDAPResponse struct {
	Conformance []string     `json:"rdapConformance"`
	Notices     []RDAPNotice `json:"notices,omitempty"`
	Links       []RDAPLink   `json:"links,omitempty"`
	Port43      string       `json:"port43,omitempty"`
}

// RDAPNetwork represents an RDAP IP network object.
type RDAPNetwork struct {
	RDAPResponse
	ObjectClassName string       `json:"objectClassName"`
	Handle          string       `json:"handle,omitempty"`
	StartAddress    string       `json:"startAddress,omitempty"`
	EndAddress      string       `json:"endAddress,omitempty"`
	IPVersion       string       `json:"ipVersion,omitempty"`
	Name            string       `json:"name,omitempty"`
	Country         string       `json:"country,omitempty"`
	Type            string       `json:"type,omitempty"`
	Status          []string     `json:"status,omitempty"`
	CIDR0CIDRs      []CIDR0      `json:"cidr0_cidrs,omitempty"`
	Entities        []RDAPEntity `json:"entities,omitempty"`
	Events          []RDAPEvent  `json:"events,omitempty"`
	Remarks         []RDAPRemark `json:"remarks,omitempty"`
	ParentHandle    string       `json:"parentHandle,omitempty"`
}

// RDAPAutnum represents an RDAP Autonomous System Number object.
type RDAPAutnum struct {
	RDAPResponse
	ObjectClassName string       `json:"objectClassName"`
	Handle          string       `json:"handle,omitempty"`
	StartAutnum     int64        `json:"startAutnum,omitempty"`
	EndAutnum       int64        `json:"endAutnum,omitempty"`
	Name            string       `json:"name,omitempty"`
	Type            string       `json:"type,omitempty"`
	Status          []string     `json:"status,omitempty"`
	Country         string       `json:"country,omitempty"`
	Entities        []RDAPEntity `json:"entities,omitempty"`
	Events          []RDAPEvent  `json:"events,omitempty"`
	Remarks         []RDAPRemark `json:"remarks,omitempty"`
}

// RDAPDomain represents an RDAP domain object (reverse DNS).
type RDAPDomain struct {
	RDAPResponse
	ObjectClassName string           `json:"objectClassName"`
	Handle          string           `json:"handle,omitempty"`
	LDHName         string           `json:"ldhName,omitempty"`
	Nameservers     []RDAPNameserver `json:"nameservers,omitempty"`
	Entities        []RDAPEntity     `json:"entities,omitempty"`
	Events          []RDAPEvent      `json:"events,omitempty"`
	Remarks         []RDAPRemark     `json:"remarks,omitempty"`
}

// RDAPEntity represents an RDAP entity (contact/organization) object.
type RDAPEntity struct {
	ObjectClassName string        `json:"objectClassName"`
	Handle          string        `json:"handle,omitempty"`
	Roles           []string      `json:"roles,omitempty"`
	Events          []RDAPEvent   `json:"events,omitempty"`
	Links           []RDAPLink    `json:"links,omitempty"`
	VcardArray      []interface{} `json:"vcardArray,omitempty"`
	Entities        []RDAPEntity  `json:"entities,omitempty"`
	Remarks         []RDAPRemark  `json:"remarks,omitempty"`
	Status          []string      `json:"status,omitempty"`
}

// RDAPSearchResult represents the result of an RDAP search query.
// APNIC's RDAP search endpoint is /entities (RFC 7482 entitySearch), which returns
// matches in the entitySearchResults field. The generic Results field is retained for
// forward compatibility with future search endpoints.
type RDAPSearchResult struct {
	RDAPResponse
	Results             []interface{} `json:"results,omitempty"`
	EntitySearchResults []RDAPEntity  `json:"entitySearchResults,omitempty"`
}

// RDAPDomainSearchResult represents the result of an RDAP /domains?name= search
// (RFC 7482 domainSearch). Matching domains are returned in DomainSearchResults.
type RDAPDomainSearchResult struct {
	RDAPResponse
	DomainSearchResults []RDAPDomain `json:"domainSearchResults,omitempty"`
}

// RDAPHelpInfo represents the response from the RDAP /help endpoint (RFC 7483),
// describing the server's capabilities, conformance, and notices.
type RDAPHelpInfo struct {
	RDAPResponse
}

// RDAPNotice represents a notice in an RDAP response.
type RDAPNotice struct {
	Title       string     `json:"title,omitempty"`
	Description []string   `json:"description,omitempty"`
	Links       []RDAPLink `json:"links,omitempty"`
}

// RDAPLink represents a link in an RDAP response.
type RDAPLink struct {
	Value string `json:"value,omitempty"`
	Rel   string `json:"rel,omitempty"`
	Href  string `json:"href,omitempty"`
	Type  string `json:"type,omitempty"`
}

// RDAPEvent represents an event in an RDAP response.
type RDAPEvent struct {
	EventAction string `json:"eventAction,omitempty"`
	EventDate   string `json:"eventDate,omitempty"`
}

// RDAPRemark represents a remark in an RDAP response.
type RDAPRemark struct {
	Title       string   `json:"title,omitempty"`
	Description []string `json:"description,omitempty"`
}

// CIDR0 represents a CIDR0 notation entry in an RDAP response.
type CIDR0 struct {
	V4Prefix string `json:"v4prefix,omitempty"`
	V6Prefix string `json:"v6prefix,omitempty"`
	Length   int    `json:"length"`
}

// RDAPNameserver represents a nameserver in an RDAP domain response.
type RDAPNameserver struct {
	LDHName string   `json:"ldhName,omitempty"`
	IPs     []string `json:"ipAddresses,omitempty"`
}

// RDAPError represents an error response from the RDAP server.
type RDAPError struct {
	ErrorCode   int      `json:"errorCode"`
	Title       string   `json:"title,omitempty"`
	Description []string `json:"description,omitempty"`
}

// IRRObject represents a single RPSL object parsed from an APNIC IRR database
// dump. The first attribute of an RPSL object is its type (e.g. "inetnum",
// "aut-num"); its value becomes the object's primary key. All attributes
// (including the key) are preserved in Attributes, keyed by attribute name
// (without the trailing colon). Multi-valued attributes (e.g. "descr") collect
// all occurrences in slice order.
type IRRObject struct {
	Type       string              // RPSL object type, e.g. "inetnum", "aut-num".
	PrimaryKey string              // value of the type attribute.
	Attributes map[string][]string // attribute name -> values, in file order.
}

// IRRDatabase holds the parsed objects from one APNIC IRR database dump.
type IRRDatabase struct {
	Type    string      // the object type this database holds.
	Objects []IRRObject // parsed RPSL objects, in file order.
}

// BGPSummary represents the parsed contents of APNIC thyme's data-summary file.
// Each line in the source is a "key: value" pair (with a leading dash separator
// line); values are kept verbatim as strings because thyme mixes counts,
// percentages and prose in the same column.
type BGPSummary struct {
	Entries []BGPKeyValue
}

// BGPKeyValue is a single key/value entry from the thyme data-summary file.
type BGPKeyValue struct {
	Key   string
	Value string
}

// BGPRoute is a single prefix-to-origin-ASN mapping from thyme's
// data-raw-table file (one "prefix\tASN" line per route).
type BGPRoute struct {
	Prefix string
	ASN    string
}

// BGPRawTable holds the parsed routes from thyme's data-raw-table file.
type BGPRawTable struct {
	Routes []BGPRoute
}

// BGPASNMap aggregates routes by origin ASN. ASNs are mapped to the list of
// prefixes they originate; it is derived locally from BGPRawTable (no extra
// network request).
type BGPASNMap struct {
	ASNs map[string][]string // origin ASN -> prefixes (in first-seen order).
}

// BGPBadPrefix represents a prefix longer than /24 and its origin AS, from
// thyme's data-badpfx-nos file. Such prefixes often indicate route leaks or
// mis-announcements.
type BGPBadPrefix struct {
	OriginAS string
	Address  string
}

// BGPBadPrefixes holds the parsed entries from thyme's data-badpfx-nos file.
type BGPBadPrefixes struct {
	Prefixes []BGPBadPrefix
}

// BGPPrefixLengthCount is a single "/N:count" entry from thyme's data-pfx-nos
// file, recording how many prefixes of each length are announced.
type BGPPrefixLengthCount struct {
	Length int    // the N in /N
	Count  int    // number of prefixes of that length
	Raw    string // the original token, e.g. "/8:16", kept for diagnostics
}

// BGPPerPrefixLength holds the parsed entries from thyme's data-pfx-nos file.
type BGPPerPrefixLength struct {
	Counts []BGPPrefixLengthCount
}

// BGPUsedAutnum is a single in-use ASN record from thyme's data-used-autnums
// file: "ASN Name - Description, CC".
type BGPUsedAutnum struct {
	ASN      string
	Name     string // the registered name (e.g. "LVLT-1")
	Country  string // ISO country code (e.g. "US")
	FullName string // the full "Name - Description" text before the country
}

// BGPUsedAutnums holds the parsed entries from thyme's data-used-autnums file.
type BGPUsedAutnums struct {
	Autnums []BGPUsedAutnum
}

// BGPSparPrefix is a prefix from the Special Purpose Address Registry
// (RFC 6890 reserved space) and its origin AS, from thyme's data-spar file.
type BGPSparPrefix struct {
	Prefix      string
	OriginAS    string
	Description string
}

// BGPSparPrefixes holds the parsed entries from thyme's data-spar file.
type BGPSparPrefixes struct {
	Prefixes []BGPSparPrefix
}

// BGPSinglePfxCount is a single row from thyme's data-singlepfx file:
// "No. of Prefixes / No. of ASNs / RIR", recording how many ASNs announce
// fewer than 20 prefixes.
type BGPSinglePfxCount struct {
	PrefixCount int
	ASNCount    int
	RIR         string
}

// BGPSinglePfx holds the parsed entries from thyme's data-singlepfx file.
type BGPSinglePfx struct {
	Counts []BGPSinglePfxCount
}

// RRDPNotification represents the parsed RRDP notification.xml file. It points
// to the current snapshot and a list of deltas (incremental updates) that
// together allow an RRDP client to synchronise RPKI repository state.
type RRDPNotification struct {
	Version   string
	SessionID string
	Serial    int64
	Snapshot  RRDPRef
	Deltas    []RRDPRef
}

// RRDPRef is a reference to an RRDP snapshot or delta: a URI and its expected
// SHA-256 hash (deltas additionally carry a serial number).
type RRDPRef struct {
	Serial int64
	URI    string
	Hash   string
}

// RPKISnapshot is the parsed metadata of an RRDP snapshot.xml file. The
// snapshot contains <publish> and <withdraw> elements; each publish carries a
// rsync URI and a base64-encoded CMS object. To keep memory bounded for the
// multi-megabyte snapshots, only the URIs are retained (the base64 bodies are
// skipped during streaming decode).
type RPKISnapshot struct {
	Version   string
	SessionID string
	Serial    int64
	Published []string // rsync URIs of <publish> elements, in file order.
	Withdrawn []string // rsync URIs of <withdraw> elements, in file order.
}

// RExUserNetwork is the result of the REx /v1/user-network endpoint: the APNIC
// REx service geo-locates the caller's source IP and reports the covering
// prefix, its origin ASN, and the economy (ISO country code) the network is
// registered in. It is the cross-RIR analogue of "which network am I in?".
type RExUserNetwork struct {
	IP      string `json:"ip"`
	Prefix  string `json:"prefix"`
	ASN     int64  `json:"asn"`
	Economy string `json:"economy"`
}

// RExResource is a single resource record returned by REx /v1/resources. Each
// item is one delegated prefix or ASN, attributed to its holder via opaqueId and
// tagged with the responsible RIR (and NIR where one applies). type is one of
// "ipv4", "ipv6", or "asn".
type RExResource struct {
	Resource       string `json:"resource"`
	Type           string `json:"type"`
	OpaqueID       string `json:"opaqueId"`
	HolderName     string `json:"holderName"`
	RIR            string `json:"rir"`
	NIR            string `json:"nir"`
	DelegationDate string `json:"delegationDate"`
	TransferDate   string `json:"transferDate"`
	CC             string `json:"cc"`
}

// RExResourcesResult wraps the /v1/resources response. REx returns a bounded
// recent window of delegated resources (newest-first), not the full history;
// for aggregate scale use FetchRExHoldersUniqueCount.
type RExResourcesResult struct {
	Items []RExResource `json:"items"`
}

// RExHolder is the aggregated per-holder view from /v1/holder. Given a holder's
// opaqueId and the responsible RIR, REx returns every ASN and prefix attributed
// to that holder along with derived size metrics (ipv4_24Count is the holder's
// IPv4 space expressed in /24 units; ipv6_48Count in /48 units).
type RExHolder struct {
	OpaqueID     string   `json:"opaqueId"`
	Registry     string   `json:"registry"`
	NIR          string   `json:"nir"`
	HolderName   string   `json:"holderName"`
	ASNs         []string `json:"asns"`
	ASNsCount    int      `json:"asnsCount"`
	IPv4         []string `json:"ipv4"`
	IPv4_24Count float64  `json:"ipv4_24Count"`
	IPv6         []string `json:"ipv6"`
	IPv6_48Count float64  `json:"ipv6_48Count"`
}

// RExHoldersCount is the unique-holder count from /v1/holders/unique-count —
// the total number of distinct resource-holder organisations across all RIRs.
type RExHoldersCount struct {
	Count int64 `json:"count"`
}
