// Sample data fixtures shared across subpackage test suites.
package testutil

const SampleDelegatedData = `2|apnic|20260627|5|19850701|20260626|+1000
apnic|*|asn|*|100|summary
apnic|*|ipv4|*|200|summary
apnic|*|ipv6|*|50|summary
apnic|JP|asn|173|1|20020801|allocated
apnic|AU|asn|1221|1|20000131|allocated
apnic|AU|ipv4|1.0.0.0|256|20110811|assigned
apnic|CN|ipv4|1.0.1.0|256|20110414|allocated
apnic|CN|ipv4|1.0.2.0|512|20110414|allocated
apnic|AU|ipv4|1.0.4.0|1024|20110412|allocated
apnic|JP|ipv6|2001:240::|32|20020801|allocated
apnic|AU|ipv6|2001:200::|32|20000131|allocated
`

const SampleExtendedData = `2.3|apnic|20260627|5|19850701|20260626|+1000
apnic|*|asn|*|100|summary
apnic|*|ipv4|*|200|summary
apnic|*|ipv6|*|50|summary
apnic|AU|ipv4|1.0.0.0|256|20110811|assigned|A91872ED
apnic|CN|ipv4|1.0.1.0|256|20110414|allocated|A92E1062
apnic|CN|ipv4|1.0.2.0|512|20110414|allocated|A92E1062
apnic|JP|ipv6|2001:240::|32|20020801|allocated|A92D9378
apnic|AU|asn|1221|1|20000131|allocated|A91872ED
`

const SampleAssignedData = `1|apnic|20260627|5|19850701|20260626|+1000
apnic|*|ipv4|*|100|summary
apnic|*|ipv6|*|50|summary
apnic|ae|ipv4||4||assigned||1
apnic|ae|ipv4||16||assigned||3
apnic|ae|ipv4||256||assigned||12
apnic|jp|ipv6||48||assigned||5
`

const SampleIPv6AssignedData = `2|apnic|20260629|7621|20020116|20260626|+1000
apnic|*|ipv6|*|7621|summary
apnic|HK|ipv6|2001:7fa:0:1::|64|20020116
apnic|KR|ipv6|2001:7fa:0:2::|64|20020117
apnic|JP|ipv6|2001:7fa:0:3::|64|20020226
apnic|TW|ipv6|2001:7fa:1::|48|20021023
`

const SampleLegacyData = `1|apnic|20260627|3|19850701|20260626|+1000
apnic|*|ipv4|*|100|summary
apnic||ipv4|128.134.0.0|65536|20040401|allocated
apnic||ipv4|128.184.0.0|65536|20040401|allocated
apnic||ipv4|128.250.0.0|65536|20040401|allocated
apnic||ipv6|2001:db8::|32|20040401|allocated
apnic||asn|237|1|20020801|allocated
`

const SampleTransfersJSON = `{
  "version": {
    "producer": "APNIC",
    "production_date": "2026-06-25T18:00:04Z",
    "remarks": [],
    "UTC_offset": 10,
    "stats_version": "4.0",
    "records_interval": {
      "start_date": "2010-04-07T00:29:32Z",
      "end_date": "2026-06-25T18:00:04Z"
    }
  },
  "transfers": [
    {
      "transfer_date": "2020-01-15T00:00:00Z",
      "type": "RESOURCE_TRANSFER",
      "source_rir": "APNIC",
      "recipient_rir": "APNIC",
      "source_organization": {"name": "Org A", "country_code": "AU"},
      "recipient_organization": {"name": "Org B", "country_code": "CN"},
      "ip4nets": {"transfer_set": [{"start_address": "1.2.3.0", "end_address": "1.2.3.255"}]},
      "ip6nets": null,
      "asns": null
    },
    {
      "transfer_date": "2021-03-20T00:00:00Z",
      "type": "INTER_RIR_TRANSFER",
      "source_rir": "ARIN",
      "recipient_rir": "APNIC",
      "source_organization": {"name": "Org C", "country_code": "US"},
      "recipient_organization": {"name": "Org D", "country_code": "JP"},
      "ip4nets": null,
      "ip6nets": {"transfer_set": [{"start_address": "2001:db8::", "end_address": "2001:db8:0:ffff:ffff:ffff:ffff:ffff"}]},
      "asns": {"transfer_set": [{"start_as_number": 64512, "end_as_number": 64520}]}
    }
  ]
}`

const SampleTransfersAll = `######################################################################
#
# CONDITIONS OF USE
#
######################################################################
resource_type|resource|from_organisation|from_economy|from_rir|previous_delegation_date|to_organisation|to_economy|to_rir|transfer_date|transfer_type
asn|45745|Gambit Group Pty Ltd|AU|APNIC|20090417|Bathurst One Pty Limited|AU|APNIC|20120620|M&A
ipv4|1.2.3.0|Org A|AU|APNIC|20100101|Org B|CN|APNIC|20200115|RESOURCE_TRANSFER
ipv6|2001:db8::|Org C|US|ARIN|20100202|Org D|JP|APNIC|20210320|INTER_RIR_TRANSFER
`

const SampleTransfersAllMD5 = `MD5 (file) = 0123456789abcdef0123456789abcdef`

const SampleIRRDump = `# APNIC IRR dump (c) APNIC
# Conditions of use...
#

inetnum:        1.1.1.0 - 1.1.1.255
netname:        APNIC-LABS
descr:          APNIC and Cloudflare DNS Resolver project
descr:          second descr line
country:        AU
admin-c:        AIC3-AP
tech-c:         AIC3-AP
remarks:        leading remarks line
 + continuation folded without extra space
mnt-by:         APNIC-HM
last-modified:  2023-04-26T22:57:58Z
source:         APNIC

inetnum:        1.0.1.0 - 1.0.1.255
netname:        CN-NET
country:        CN
mnt-by:         MAINT-CNNIC
source:         APNIC
`

const SampleTelemetryJSON = `{
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

const SampleTelemetryMD5 = `0123456789abcdef0123456789abcdef  file`

const SampleChangesData = `{"count":3,"stats-begin":"https://ftp.apnic.net/stats/apnic/2026/delegated-apnic-extended-20260626.gz","stats-end":"https://ftp.apnic.net/stats/apnic/2026/delegated-apnic-extended-20260627.gz","timestamp":"2026-06-26 15:23:38","version":"0.1"}
{"cc":"IN","custodian":"A91ED89F","resources":["160.236.32.0/23"],"status":"allocated","timestamp":"2026-06-25T22:16:15","type":"delegated"}
{"cc":"BD","resources":["202.136.88.0/22"],"timestamp":"2026-06-25T22:27:30","type":"cc-changed"}
{"cc":"IN","custodian":"A9139ECA","resources":["152478"],"status":"allocated","timestamp":"2026-06-25T23:36:41","type":"delegated"}
`

const SampleRDAPNetworkJSON = `{
  "rdapConformance": ["rdap_level_0"],
  "objectClassName": "ip network",
  "handle": "1.1.1.0 - 1.1.1.255",
  "startAddress": "1.1.1.0",
  "endAddress": "1.1.1.255",
  "ipVersion": "v4",
  "name": "APNIC-LABS",
  "country": "AU",
  "type": "ASSIGNED PORTABLE",
  "status": ["active"],
  "cidr0_cidrs": [{"v4prefix": "1.1.1.0", "length": 24}],
  "entities": [
    {
      "objectClassName": "entity",
      "handle": "AIC3-AP",
      "roles": ["administrative", "technical"],
      "events": [{"eventAction": "registration", "eventDate": "2023-04-26T00:42:16Z"}]
    }
  ],
  "events": [
    {"eventAction": "registration", "eventDate": "2011-08-10T23:12:35Z"},
    {"eventAction": "last changed", "eventDate": "2023-04-26T22:57:58Z"}
  ],
  "remarks": [{"title": "description", "description": ["APNIC and Cloudflare DNS Resolver project"]}],
  "port43": "whois.apnic.net"
}`

const SampleRDAPAutnumJSON = `{
  "rdapConformance": ["rdap_level_0"],
  "objectClassName": "autnum",
  "handle": "AS13335",
  "startAutnum": 13335,
  "endAutnum": 13335,
  "name": "CLOUDFLARE",
  "type": "ASSIGNED PORTABLE",
  "status": ["active"],
  "country": "AU",
  "entities": [],
  "events": [{"eventAction": "registration", "eventDate": "2010-07-14T00:00:00Z"}],
  "remarks": [],
  "port43": "whois.apnic.net"
}`

const SampleRDAPDomainJSON = `{
  "rdapConformance": ["rdap_level_0"],
  "objectClassName": "domain",
  "handle": "1.0.0.1.in-addr.arpa",
  "ldhName": "1.0.0.1.in-addr.arpa",
  "nameservers": [{"ldhName": "ns1.example.com"}],
  "entities": [],
  "events": [{"eventAction": "registration", "eventDate": "2018-03-27T00:30:48Z"}],
  "port43": "whois.apnic.net"
}`

const SampleRDAPEntityJSON = `{
  "rdapConformance": ["rdap_level_0"],
  "objectClassName": "entity",
  "handle": "AIC3-AP",
  "roles": ["administrative", "technical"],
  "events": [{"eventAction": "registration", "eventDate": "2023-04-26T00:42:16Z"}],
  "links": [{"rel": "self", "href": "https://rdap.apnic.net/entity/AIC3-AP", "type": "application/rdap+json"}],
  "vcardArray": ["vcard", [["fn", {}, "text", "APNIC Contact"]]],
  "port43": "whois.apnic.net"
}`

const SampleRDAPSearchJSON = `{
  "rdapConformance": ["rdap_level_0"],
  "entitySearchResults": [
    {"objectClassName": "entity", "handle": "AIC3-AP", "roles": ["administrative"]},
    {"objectClassName": "entity", "handle": "IRA1-AP", "roles": ["technical"]}
  ],
  "port43": "whois.apnic.net"
}`

const SampleRDAPDomainsSearchJSON = `{
  "rdapConformance": ["rdap_level_0", "nro_rdap_profile_0"],
  "domainSearchResults": [
    {"objectClassName": "domain", "handle": "1.in-addr.arpa", "ldhName": "1.in-addr.arpa"},
    {"objectClassName": "domain", "handle": "2.in-addr.arpa", "ldhName": "2.in-addr.arpa"}
  ],
  "port43": "whois.apnic.net"
}`

const SampleRDAPHelpJSON = `{
  "rdapConformance": ["rdap_level_0", "history_version_0", "cidr0", "nro_rdap_profile_0", "redirect_with_content"],
  "notices": [
    {"title": "Terms of Service", "description": ["By using the APNIC RDAP service you agree to the APNIC terms of service."]},
    {"title": "Inaccuracy Reports", "description": ["Use the APNIC inaccuracy report form."]}
  ],
  "port43": "whois.apnic.net"
}`

const SampleRDAPNotFoundJSON = `{
  "errorCode": 404,
  "title": "Not Found",
  "description": ["The server has not found anything matching the Request-URI."]
}`

// SampleWhoisResponse is a realistic APNIC whois response for 1.1.1.1: a primary
// inetnum object followed by a route object, separated by a blank line. It uses
// only fields APNIC actually emits (no fabricated CIDR/parent/created keys), and
// includes an abuse-mailbox so AbuseContact/AbuseMailbox parsing can be asserted.
const SampleWhoisResponse = `% Whois information

inetnum:        1.1.1.0 - 1.1.1.255
netname:        APNIC-LABS
descr:          APNIC and Cloudflare DNS Resolver project
country:        AU
org:            ORG-ARAD1-AP
abuse-c:        AA1412-AP
abuse-mailbox:  helpdesk@apnic.net
status:         ASSIGNED PORTABLE
last-modified:  2023-04-26T22:57:58Z
source:         APNIC

route:          1.1.1.0/24
origin:         AS13335
descr:          APNIC Research and Development
last-modified:  2023-04-26T02:42:44Z
source:         APNIC
`

// SampleWhoisListResponse simulates the response to a "-L" (all-less-specific)
// query for 1.1.1.1: multiple inetnum objects of decreasing specificity plus a
// route object, each terminated by "source: APNIC" and separated by blank
// lines. Used to exercise ParseWhoisResponseList.
const SampleWhoisListResponse = `inetnum:        1.1.1.0 - 1.1.1.255
netname:        APNIC-LABS
descr:          APNIC and Cloudflare DNS Resolver project
country:        AU
status:         ASSIGNED PORTABLE
last-modified:  2023-04-26T22:57:58Z
source:         APNIC

route:          1.1.1.0/24
origin:         AS13335
source:         APNIC

inetnum:        1.0.0.0 - 1.255.255.255
netname:        APNIC-AP
country:        AU
status:         ALLOCATED PORTABLE
last-modified:  2011-08-10T23:12:35Z
source:         APNIC
`
