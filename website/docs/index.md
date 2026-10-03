# apnic-skills

> A comprehensive Go SDK for APNIC (Asia-Pacific Network Information Centre) public data services, providing full coverage of all APNIC data endpoints and query capabilities.

[![Go Reference](https://img.shields.io/badge/go-reference-00ADD8?logo=go)](https://pkg.go.dev/github.com/hitechcloud-vietnam/apnic-skills)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/hitechcloud-vietnam/apnic-skills/blob/main/LICENSE)
[![Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen)](https://github.com/hitechcloud-vietnam/apnic-skills)
[![Go Report Card](https://goreportcard.com/badge/github.com/hitechcloud-vietnam/apnic-skills)](https://goreportcard.com/report/github.com/hitechcloud-vietnam/apnic-skills)

## Overview

```mermaid
graph TB
    subgraph SDK["apnic-skills SDK"]
        direction TB
        Client["Client<br/>HTTP + Anti-scraping<br/>+ Chunked Download<br/>+ Cache"]
        
        subgraph DataSources["Data Sources"]
            Stats["Stats<br/>delegated/extended<br/>assigned/ipv6-assigned/legacy"]
            BGP["Thyme BGP<br/>8 files × 3 sources<br/>(current/au/hk)"]
            RDAP["RDAP<br/>IP/CIDR/ASN<br/>domain/entity/search"]
            Whois["Whois<br/>+ Reverse DNS"]
            IRR["IRR Database<br/>19 RPSL object types"]
            RPKI["RPKI/RRDP<br/>notification/snapshot/delta"]
            REx["REx Cross-RIR<br/>resources/holder/network/count"]
            Transfers["Transfers + Changes<br/>+ Telemetry"]
        end
        
        subgraph Features["Built-in Features"]
            Stealth["Browser Mimicry<br/>Headers + Jitter"]
            Chunked["Chunked Download<br/>Range × 4 workers"]
            Cache["Token Bucket Cache"]
            Filter["Chain Filtering"]
            Verify["Data Integrity<br/>MD5 + PGP"]
        end
        
        Client --> DataSources
        Client --> Features
    end
    
    subgraph CLI["CLI (cobra, 24 commands)"]
        Commands["Full SDK coverage"]
    end
    
    CLI --> SDK
    
    subgraph APNIC["APNIC Services"]
        FTP["ftp.apnic.net"]
        RDAP_Svc["rdap.apnic.net"]
        Thyme["thyme.apnic.net"]
        RExAPI["api.rex.apnic.net"]
        WhoisSvc["whois.apnic.net:43"]
    end
    
    SDK -->|HTTPS| APNIC
```

## Key Features

:material-speedometer: **High Performance** — Multi-connection chunked download for large files (delegated 4.3MB, IRR 50MB+), bypassing APNIC FTP single-connection throttling.

:material-shield-check: **Anti-Scraping** — Browser mimicry headers + token bucket rate limiting + random jitter, default on, configurable.

:material-database-search: **Comprehensive Coverage** — All APNIC public data services: stats, RDAP, whois, IRR, RPKI/RRDP, thyme BGP, REx cross-RIR registry, transfers, changes, telemetry.

:material-filter-variant: **Chain Filtering** — Fluent API for filtering delegated/extended stats by country, type, status, date range, opaque-id.

:material-check-circle: **Data Integrity** — MD5 and PGP signature verification of all published data.

:material-console: **Full CLI** — 24 cobra subcommands covering every SDK capability, with JSON output and global flags.

:material-test-tube: **100% Test Coverage** — SDK statement coverage 100%, CLI named functions 100%.

## Data Source Map

```mermaid
graph LR
    subgraph APNIC_Services["APNIC Services"]
        FTP["ftp.apnic.net<br/>━━━━━━━━━━━━<br/>Stats (delegated/extended/<br/>assigned/ipv6-assigned/legacy)<br/>IRR (apnic.db.*.gz)<br/>Transfers-all<br/>Telemetry<br/>MD5/.asc/.pgp"]
        RDAP["rdap.apnic.net<br/>━━━━━━━━━━━━<br/>IP/CIDR/ASN/domain/entity<br/>search/domains/help<br/>+ point-in-time history"]
        Thyme["thyme.apnic.net<br/>━━━━━━━━━━━━<br/>BGP: current/au/hk<br/>summary/raw-table<br/>bad-prefixes/used-autnums<br/>spar/single-pfx"]
        REx["api.rex.apnic.net<br/>━━━━━━━━━━━━<br/>Cross-RIR registry:<br/>resources/holder/network/count"]
        Whois["whois.apnic.net:43<br/>━━━━━━━━━━━━<br/>IP/ASN/raw queries"]
    end
    
    subgraph SDK_Methods["SDK Fetch Methods"]
        M1["FetchDelegated*"]
        M2["FetchExtended*"]
        M3["FetchAssigned*"]
        M4["FetchLegacy*"]
        M5["FetchIRRDatabase"]
        M6["FetchTransfers*"]
        M7["FetchTelemetry"]
        M8["FetchMD5/ASC/PublicKey"]
        M9["RDAPLookup*"]
        M10["RDAPSearch*"]
        M11["RDAPHelp"]
        M12["FetchBGP*"]
        M13["FetchREx*"]
        M14["QueryWhois*"]
        M15["ReverseDNS"]
    end
    
    FTP --> M1 & M2 & M3 & M4 & M5 & M6 & M7 & M8
    RDAP --> M9 & M10 & M11
    Thyme --> M12
    REx --> M13
    Whois --> M14 & M15
```

## Quick Links

| Resource | Description |
|----------|-------------|
| [Getting Started](getting-started/index.md) | Installation, quick start, configuration |
| [SDK Reference](sdk/index.md) | Complete API documentation by data source |
| [CLI Reference](cli/index.md) | All 24 subcommands documented |
| [Workflows](workflows/index.md) | Real-world usage workflows with examples |
| [Architecture](architecture/index.md) | HTTP client, anti-scraping, chunked download design |
| [API Types](types/index.md) | Struct/type reference |

## Quick Start

```bash
# Install
go get github.com/hitechcloud-vietnam/apnic-skills
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    apnic "github.com/hitechcloud-vietnam/apnic-skills"
)

func main() {
    client := apnic.NewClient()
    ctx := context.Background()

    // RDAP IP lookup
    network, err := client.RDAPLookupIP(ctx, "1.1.1.1")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Network: %s, Country: %s, Type: %s\n",
        network.Handle, network.Country, network.Type)

    // Delegated stats with chain filtering
    entries, err := client.GetDelegatedEntries(ctx)
    if err != nil {
        log.Fatal(err)
    }
    cn := apnic.NewFilter(entries).
        ByCountry("CN").
        ByType("ipv4").
        ByStatus("allocated").
        Result()
    fmt.Printf("CN allocated IPv4 entries: %d\n", len(cn))
}
```

## CLI Quick Reference

```bash
# Build
go build -o bin/apnic ./cmd/apnic

# Common commands
apnic delegated --json | jq '.Entries | length'
apnic rdap ip 1.1.1.1
apnic whois ip 1.1.1.1
apnic reverse-dns 1.1.1.1
apnic bgp summary --bgp-source au
apnic irr inetnum
apnic rex holder <opaqueId> apnic
apnic verify integrity --type delegated
```

## End-to-End Data Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant SDK as apnic-skills SDK
    participant HTTP as HTTP Client
    participant APNIC as APNIC Service
    
    User->>CLI: apnic <command> [flags]
    CLI->>SDK: client.FetchX(ctx, ...)
    SDK->>HTTP: doHTTPRequest()
    HTTP->>HTTP: Apply browser headers
    HTTP->>HTTP: waitRateLimit() (token bucket)
    HTTP->>HTTP: jitter() (random delay)
    
    alt Large file + Range support
        HTTP->>APNIC: Probe Range:0-0
        APNIC-->>HTTP: 206 + Content-Range
        HTTP->>APNIC: N concurrent Range requests (2MiB each)
        APNIC-->>HTTP: Chunk bytes
        HTTP->>HTTP: Merge via io.Pipe + gzip decompress
    else Small file / no Range
        HTTP->>APNIC: Single GET
        APNIC-->>HTTP: Full response
    end
    
    HTTP-->>SDK: io.Reader
    SDK->>SDK: Parse (parseXFull)
    SDK-->>CLI: Struct result
    CLI-->>User: JSON or human-readable output
```

## License

MIT — see [LICENSE](https://github.com/hitechcloud-vietnam/apnic-skills/blob/main/LICENSE).