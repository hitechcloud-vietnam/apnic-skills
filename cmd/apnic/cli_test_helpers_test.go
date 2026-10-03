package main

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

// sampleDelegated mirrors the SDK test sample for end-to-end CLI testing.
const sampleDelegated = `2|apnic|20260627|5|19850701|20260626|+1000
apnic|*|asn|*|100|summary
apnic|*|ipv4|*|200|summary
apnic|*|ipv6|*|50|summary
apnic|JP|asn|173|1|20020801|allocated
apnic|AU|ipv4|1.0.0.0|256|20110811|assigned
apnic|CN|ipv4|1.0.1.0|256|20110414|allocated
apnic|JP|ipv6|2001:240::|32|20020801|allocated
`

const sampleExtended = `2.3|apnic|20260627|5|19850701|20260626|+1000
apnic|*|asn|*|100|summary
apnic|*|ipv4|*|200|summary
apnic|AU|ipv4|1.0.0.0|256|20110811|assigned|A91872ED
apnic|CN|ipv4|1.0.1.0|256|20110414|allocated|A92E1062
apnic|AU|asn|1221|1|20000131|allocated|A91872ED
`

const sampleAssigned = `1|apnic|20260627|5|19850701|20260626|+1000
apnic|*|ipv4|*|100|summary
apnic|ae|ipv4||4||assigned||1
apnic|ae|ipv4||256||assigned||12
`

const sampleIPv6Assigned = `2|apnic|20260629|4|20020116|20260626|+1000
apnic|*|ipv6|*|4|summary
apnic|HK|ipv6|2001:7fa:0:1::|64|20020116
apnic|KR|ipv6|2001:7fa:0:2::|64|20020117
`

const sampleLegacy = `1|apnic|20260627|3|19850701|20260626|+1000
apnic|*|ipv4|*|100|summary
apnic||ipv4|128.134.0.0|65536|20040401|allocated
`

const sampleTransfers = `{
  "version": {"producer":"APNIC","production_date":"2026-06-25T18:00:04Z","UTC_offset":10,"stats_version":"4.0","records_interval":{"start_date":"2010-04-07T00:29:32Z","end_date":"2026-06-25T18:00:04Z"}},
  "transfers": [
    {"transfer_date":"2020-01-15T00:00:00Z","type":"RESOURCE_TRANSFER","source_rir":"APNIC","recipient_rir":"APNIC","source_organization":{"name":"Org A","country_code":"AU"},"recipient_organization":{"name":"Org B","country_code":"CN"},"ip4nets":{"transfer_set":[{"start_address":"1.2.3.0","end_address":"1.2.3.255"}]},"ip6nets":null,"asns":null}
  ]
}`

const sampleChanges = `{"count":1,"stats-begin":"x","stats-end":"y","timestamp":"2026-06-26 15:23:38","version":"0.1"}
{"cc":"IN","custodian":"A91ED89F","resources":["160.236.32.0/23"],"status":"allocated","timestamp":"2026-06-25T22:16:15","type":"delegated"}
`

const sampleRDAPNetwork = `{
  "rdapConformance":["rdap_level_0"],
  "objectClassName":"ip network",
  "handle":"1.1.1.0 - 1.1.1.255",
  "startAddress":"1.1.1.0","endAddress":"1.1.1.255",
  "ipVersion":"v4","name":"APNIC-LABS","country":"AU","type":"ASSIGNED PORTABLE",
  "status":["active"],
  "cidr0_cidrs":[{"v4prefix":"1.1.1.0","length":24}],
  "entities":[{"objectClassName":"entity","handle":"AIC3-AP","roles":["technical"]}],
  "events":[{"eventAction":"registration","eventDate":"2011-08-10T23:12:35Z"}],
  "port43":"whois.apnic.net"
}`

const sampleRDAPNetworkV6 = `{
  "rdapConformance":["rdap_level_0"],
  "objectClassName":"ip network",
  "handle":"2001:db8:: - 2001:db8:ffff:ffff:ffff:ffff:ffff:ffff",
  "startAddress":"2001:db8::","endAddress":"2001:db8:ffff:ffff:ffff:ffff:ffff:ffff",
  "ipVersion":"v6","name":"V6-NET","country":"AU","type":"ASSIGNED PORTABLE",
  "status":["active"],
  "cidr0_cidrs":[{"v6prefix":"2001:db8::","length":32}],
  "entities":[],"events":[],"port43":"whois.apnic.net"
}`

const sampleRDAPAutnum = `{
  "rdapConformance":["rdap_level_0"],
  "objectClassName":"autnum","handle":"AS13335","startAutnum":13335,"endAutnum":13335,
  "name":"CLOUDFLARE","type":"ASSIGNED PORTABLE","country":"AU","entities":[],"events":[],"remarks":[],"port43":"whois.apnic.net"
}`

const sampleRDAPDomain = `{
  "rdapConformance":["rdap_level_0"],
  "objectClassName":"domain","handle":"1.0.0.1.in-addr.arpa","ldhName":"1.0.0.1.in-addr.arpa",
  "nameservers":[],"entities":[],"events":[],"port43":"whois.apnic.net"
}`

const sampleRDAPEntity = `{
  "rdapConformance":["rdap_level_0"],
  "objectClassName":"entity","handle":"AIC3-AP","roles":["administrative"],
  "events":[{"eventAction":"registration","eventDate":"2023-04-26T00:42:16Z"}],"port43":"whois.apnic.net"
}`

const sampleRDAPSearch = `{
  "rdapConformance":["rdap_level_0"],
  "entitySearchResults":[{"objectClassName":"entity","handle":"AIC3-AP","roles":["technical"]}],
  "port43":"whois.apnic.net"
}`

const sampleRDAPDomainsSearch = `{
  "rdapConformance":["rdap_level_0","nro_rdap_profile_0"],
  "domainSearchResults":[
    {"objectClassName":"domain","handle":"1.in-addr.arpa","ldhName":"1.in-addr.arpa"},
    {"objectClassName":"domain","handle":"2.in-addr.arpa","ldhName":"2.in-addr.arpa"}
  ],
  "port43":"whois.apnic.net"
}`

const sampleRDAPHelp = `{
  "rdapConformance":["rdap_level_0","history_version_0","cidr0"],
  "notices":[{"title":"Terms of Service","description":["APNIC RDAP terms."]}],
  "port43":"whois.apnic.net"
}`

const samplePubKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----
Comment: APNIC public key
mockkeydata
-----END PGP PUBLIC KEY BLOCK-----`

const sampleTransfersAll = `######################################################################
#
# CONDITIONS OF USE
#
######################################################################
resource_type|resource|from_organisation|from_economy|from_rir|previous_delegation_date|to_organisation|to_economy|to_rir|transfer_date|transfer_type
asn|45745|Gambit Group Pty Ltd|AU|APNIC|20090417|Bathurst One Pty Limited|AU|APNIC|20120620|M&A
ipv4|1.2.3.0|Org A|AU|APNIC|20100101|Org B|CN|APNIC|20200115|RESOURCE_TRANSFER
ipv6|2001:db8::|Org C|US|ARIN|20100202|Org D|JP|APNIC|20210320|INTER_RIR_TRANSFER
`

const sampleTelemetry = `{
  "RDAP": {
    "date_range": {"start": "2026-07-01T06:00:00Z", "end": "2026-07-01T07:00:00Z"},
    "total_queries": 3070925,
    "total_asns": 1737,
    "query_type_distribution": {"ip": 3030224, "autnum": 28141, "entity": 10441, "domain": 2044},
    "asns": [
      {"asn": "45102", "query_count": 2274463, "query_count_by_type": {"ip": 2274457, "entity": 6}}
    ]
  }
}`

const sampleIRRDump = `# APNIC IRR dump
inetnum:        1.1.1.0 - 1.1.1.255
netname:        APNIC-LABS
country:        AU
source:         APNIC

inetnum:        1.0.1.0 - 1.0.1.255
netname:        CN-NET
country:        CN
source:         APNIC
`

const sampleCurrentSerial = "16159398"

const sampleBGPSummary = `Analysis Summary
----------------

BGP routing table entries examined:                             1059904
    Prefixes after maximum aggregation (per Origin AS):          407882
Total ASes present in the Internet Routing Table:                 78800
Average AS path length visible in the Internet Routing Table:       4.7
`

const sampleBGPRawTable = `1.0.0.0/24	13335
1.0.4.0/24	38803
1.1.1.0/24	13335
`

// CLI-test samples for the 5 additional thyme BGP files (mirror SDK samples in
// bgp_test.go; kept here because CLI tests are in package main and cannot
// reference the SDK test package's unexported constants).
const sampleBGPBadPrefixes = `Origin AS       Address
    10167       1.209.111.128/25
    12345       2.2.2.0/26
`
const sampleBGPPerPrefixLength = ` /8:16       /9:14      /12:299
`
const sampleBGPUsedAutnums = `     1 LVLT-1 - Level 3 Parent, LLC, US
   13335 CLOUDFLARENET - Cloudflare, Inc., US
`
const sampleBGPSparPrefixes = `192.88.99.0/24             6939  HURRICANE - Hurricane Electric LLC, US
`
const sampleBGPSinglePfx = `       1              27539   Global
       3                500   APNIC
`

const sampleRRDPNotification = `<notification xmlns="http://www.ripe.net/rpki/rrdp" version="1" session_id="8dad0cc8" serial="65148">
  <snapshot uri="https://rrdp.apnic.net/8dad0cc8/snapshot.xml" hash="479c1351cc5372febc3487abe80bad01ea04118a78f59100004c213f944022d9"/>
  <delta serial="65148" uri="https://rrdp.apnic.net/8dad0cc8/delta-65148.xml" hash="45ff4de1ac87c9b41009b5e71d7ff175adb01ce69af26bfbe5b7093a027cc0c5"/>
</notification>`

const sampleRRDPSnapshot = `<snapshot version="1" session_id="8dad0cc8" serial="65148" xmlns="http://www.ripe.net/rpki/rrdp">
<publish uri="rsync://rpki.apnic.net/rep/roa1.roa">AAAABASE64BODY1</publish>
<withdraw uri="rsync://rpki.apnic.net/rep/old.roa"/>
</snapshot>`

const sampleRExUserNetwork = `{"ip":"219.142.144.241","prefix":"219.142.128.0/18","asn":4847,"economy":"CN"}`

const sampleRExResources = `{"items":[
{"resource":"23.160.212.0/24","type":"ipv4","opaqueId":"522be47e60b5c2ef81bbbab8deaa6b85","holderName":"ERIN AVENUE LLC","rir":"arin","nir":null,"delegationDate":"2026-06-30","transferDate":null,"cc":"US"},
{"resource":"AS402676","type":"asn","opaqueId":"522be47e60b5c2ef81bbbab8deaa6b85","holderName":"ERIN AVENUE LLC","rir":"arin","nir":null,"delegationDate":"2026-06-30","transferDate":null,"cc":"US"}
]}`

const sampleRExHolder = `{"opaqueId":"522be47e60b5c2ef81bbbab8deaa6b85","registry":"arin","nir":null,"holderName":"ERIN AVENUE LLC","asns":["AS402676"],"asnsCount":1,"ipv4":["23.160.212.0/24"],"ipv4_24Count":1.0,"ipv6":["2602:f373::/40"],"ipv6_48Count":256.0}`

const sampleRExHoldersCount = `{"count":129665}`

// pickSample selects the sample payload matching a stats URL path.
func pickSample(path string) (string, string) {
	switch {
	case strings.Contains(path, "apnic-extended"):
		return sampleExtended, "text/plain"
	case strings.Contains(path, "apnic-ipv6-assigned"):
		return sampleIPv6Assigned, "text/plain"
	case strings.Contains(path, "assigned"):
		return sampleAssigned, "text/plain"
	case strings.Contains(path, "legacy"):
		return sampleLegacy, "text/plain"
	case strings.Contains(path, "delegated"):
		return sampleDelegated, "text/plain"
	case strings.Contains(path, "transfer-all"):
		return sampleTransfersAll, "text/plain"
	case strings.Contains(path, "transfers"):
		return sampleTransfers, "application/json"
	case strings.Contains(path, "whois-rdap-stats"):
		return sampleTelemetry, "application/json"
	case strings.Contains(path, "changes"):
		return sampleChanges, "application/json"
	case strings.Contains(path, "apnic.db."):
		return sampleIRRDump, "text/plain"
	default:
		return "", "text/plain"
	}
}

// cliHandler serves both stats (with .gz + .md5/.asc sidecars) and RDAP routes.
func cliHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// RDAP routes
		if strings.HasPrefix(path, "/ip/") || strings.HasPrefix(path, "/autnum/") ||
			strings.HasPrefix(path, "/domain/") || strings.HasPrefix(path, "/domains") ||
			strings.HasPrefix(path, "/entity/") || strings.HasPrefix(path, "/entities") ||
			strings.HasSuffix(path, "/help") {
			w.Header().Set("Content-Type", "application/rdap+json")
			switch {
			case strings.Contains(path, "/autnum/"):
				w.Write([]byte(sampleRDAPAutnum))
			case strings.Contains(path, "/domain/"):
				w.Write([]byte(sampleRDAPDomain))
			case strings.Contains(path, "/domains"):
				w.Write([]byte(sampleRDAPDomainsSearch))
			case strings.Contains(path, "/entities"):
				w.Write([]byte(sampleRDAPSearch))
			case strings.HasSuffix(path, "/help"):
				w.Write([]byte(sampleRDAPHelp))
			case strings.HasPrefix(path, "/entity/"):
				w.Write([]byte(sampleRDAPEntity))
			default:
				// Serve an IPv6 network object for IPv6 lookups, IPv4 otherwise.
				if strings.Contains(path, ":") {
					w.Write([]byte(sampleRDAPNetworkV6))
				} else {
					w.Write([]byte(sampleRDAPNetwork))
				}
			}
			return
		}

		// Public key
		if strings.HasSuffix(path, "CURRENT_PUBLIC_KEY") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(samplePubKey))
			return
		}

		// IRR current serial
		if strings.HasSuffix(path, "CURRENTSERIAL") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(sampleCurrentSerial))
			return
		}

		// thyme BGP files
		if strings.HasSuffix(path, "data-summary") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(sampleBGPSummary))
			return
		}
		if strings.HasSuffix(path, "data-raw-table") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(sampleBGPRawTable))
			return
		}

		// thyme BGP additional files
		if strings.HasSuffix(path, "data-badpfx-nos") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(sampleBGPBadPrefixes))
			return
		}
		if strings.HasSuffix(path, "data-pfx-nos") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(sampleBGPPerPrefixLength))
			return
		}
		if strings.HasSuffix(path, "data-used-autnums") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(sampleBGPUsedAutnums))
			return
		}
		if strings.HasSuffix(path, "data-spar") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(sampleBGPSparPrefixes))
			return
		}
		if strings.HasSuffix(path, "data-singlepfx") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(sampleBGPSinglePfx))
			return
		}

		// RRDP / RPKI files
		if strings.HasSuffix(path, "notification.xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(sampleRRDPNotification))
			return
		}
		if strings.HasSuffix(path, ".xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(sampleRRDPSnapshot))
			return
		}

		// REx cross-RIR resource registry API routes (api.rex.apnic.net/v1/*)
		if strings.HasPrefix(path, "/v1/") {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case strings.HasSuffix(path, "/user-network"):
				w.Write([]byte(sampleRExUserNetwork))
			case strings.HasSuffix(path, "/resources"):
				w.Write([]byte(sampleRExResources))
			case strings.HasSuffix(path, "/holder"):
				// REx returns a plain-text 400 when required params are missing.
				if r.URL.Query().Get("opaqueId") == "" || r.URL.Query().Get("rir") == "" {
					w.Header().Set("Content-Type", "text/plain")
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("Either resource or opaque ID and RIR are required as query parameters."))
					return
				}
				w.Write([]byte(sampleRExHolder))
			case strings.HasSuffix(path, "/unique-count"):
				w.Write([]byte(sampleRExHoldersCount))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
			return
		}

		// Stats + sidecars
		sample, _ := pickSample(path)
		if sample == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// MD5 sidecar: return the real MD5 of the (decompressed) sample so VerifyMD5 passes.
		if strings.Contains(path, ".md5") {
			payload := "MD5 (file) = " + fmt.Sprintf("%x", md5.Sum([]byte(sample)))
			serveGzIfDated(w, r, payload)
			return
		}
		// ASC sidecar
		if strings.Contains(path, ".asc") {
			serveGzIfDated(w, r, "-----BEGIN PGP SIGNATURE-----\nmock\n-----END PGP SIGNATURE-----")
			return
		}

		serveGzIfDated(w, r, sample)
	}
}

// serveGzIfDated writes payload gzip-compressed when the URL ends in .gz, else plain.
func serveGzIfDated(w http.ResponseWriter, r *http.Request, payload string) {
	if strings.HasSuffix(r.URL.Path, ".gz") {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(gzipCompress([]byte(payload)))
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(payload))
}

// newCLITestServer starts a mock server covering all CLI routes.
func newCLITestServer() *httptest.Server {
	return httptest.NewServer(cliHandler())
}

// bigBGPRawTable builds a thyme data-raw-table payload with n route lines so
// the >50 truncation path in 'bgp raw-table' is exercised.
func bigBGPRawTable(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "10.%d.%d.0/24\t%d\n", (i/256)%256, i%256, 65000+i%100)
	}
	return b.String()
}

// bigRRDPNotification builds a notification.xml with n delta entries so the
// >20 truncation path in 'rpki notification' is exercised.
func bigRRDPNotification(n int) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<notification xmlns="http://www.ripe.net/rpki/rrdp" version="1" session_id="sess" serial="%d">`, n)
	fmt.Fprintf(&b, `<snapshot uri="https://rrdp.apnic.net/sess/snapshot.xml" hash="abc"/>`)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, `<delta serial="%d" uri="https://rrdp.apnic.net/sess/delta-%d.xml" hash="h%d"/>`, i, i, i)
	}
	b.WriteString(`</notification>`)
	return b.String()
}

// bigIRRDump builds an RPSL inetnum dump with n objects so the >50 truncation
// path in 'irr inetnum' is exercised.
func bigIRRDump(n int) string {
	var b strings.Builder
	b.WriteString("# APNIC IRR dump\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "inetnum:        10.%d.%d.0 - 10.%d.%d.255\n", (i/256)%256, i%256, (i/256)%256, i%256)
		b.WriteString("netname:        NET\n")
		b.WriteString("country:        AU\n")
		b.WriteString("source:         APNIC\n\n")
	}
	return b.String()
}

// bigBGPUsedAutnums builds a thyme data-used-autnums payload with n ASN lines
// so the >50 truncation path in 'bgp used-autnums' is exercised.
func bigBGPUsedAutnums(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "%d AS%d - Autonom%d, LLC, US\n", i+1, i+1, i+1)
	}
	return b.String()
}

// bigBGPBadPrefixes builds a thyme data-badpfx-nos payload with n route lines
// so the >50 truncation path in 'bgp bad-prefixes' is exercised.
func bigBGPBadPrefixes(n int) string {
	var b strings.Builder
	b.WriteString("Prefixes longer than /24\n")
	b.WriteString("-----------------------\n")
	b.WriteString("Origin AS       Address\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "    %d       10.%d.%d.128/25\n", 65000+i%100, (i/256)%256, i%256)
	}
	return b.String()
}

// newLargeDatasetServer starts a mock server whose BGP raw-table, RRDP
// notification and IRR inetnum routes return datasets large enough to trigger
// the truncation (limit > N) code paths. Other routes delegate to cliHandler().
func newLargeDatasetServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "data-raw-table"):
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(bigBGPRawTable(60)))
			return
		case strings.HasSuffix(path, "data-used-autnums"):
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(bigBGPUsedAutnums(60)))
			return
		case strings.HasSuffix(path, "data-badpfx-nos"):
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(bigBGPBadPrefixes(60)))
			return
		case strings.HasSuffix(path, "notification.xml"):
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(bigRRDPNotification(25)))
			return
		case strings.HasSuffix(path, "apnic.db.inetnum.gz"):
			serveGzIfDated(w, r, bigIRRDump(60))
			return
		default:
			cliHandler()(w, r)
			return
		}
	}))
}
