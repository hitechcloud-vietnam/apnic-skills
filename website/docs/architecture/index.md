# Architecture

The apnic-skills SDK is built around a single `Client` that funnels every outbound request — HTTP, chunked Range download, and whois — through one anti-scraping-aware transport. This section documents the internal design of that transport and the parser pipeline behind it.

## Architecture at a Glance

The diagram below shows the full request path from a caller's `Fetch*` / `Get*` method down to the wire. Every layer is documented in its own page under this section.

```mermaid
graph TB
    subgraph Caller["Caller"]
        Get["Get* methods<br/>cache-first"]
        Fetch["Fetch* methods<br/>bypass cache"]
    end

    subgraph Cache["Cache Layer"]
        CacheBox["cache<br/>sync.RWMutex + TTL<br/>default 30min"]
    end

    subgraph Parser["Parser Layer"]
        StreamParse["parseXFull(io.Reader)<br/>streaming, e.g. delegated"]
        StrParse["parseX(string)<br/>full-buffer, e.g. IRR / BGP"]
        XMLParse["xml.Decoder stream<br/>RRDP snapshot/delta"]
        JSONParse["json.Decode<br/>RDAP / REx"]
    end

    subgraph Fetch["Fetch Helpers"]
        FetchReader["fetchReader<br/>chunked or single-stream"]
        FetchText["fetchText<br/>string + gzip"]
        FetchTextStr["fetchTextStr<br/>string buffer"]
        FetchJSON["fetchJSON<br/>REx JSON + gzip"]
    end

    subgraph Download["Chunked Download"]
        Probe["probeRange<br/>Range: bytes=0-0"]
        Plan["planChunks<br/>2MiB blocks, max 64"]
        Workers["fetchChunkWithRetry<br/>× maxConcurrent (≤16)"]
        Merge["io.Pipe<br/>serial merge"]
        Split["slow-chunk split<br/>degraded retry"]
    end

    subgraph HTTP["HTTP Layer"]
        DoReq["doHTTPRequest<br/>unified outlet"]
    end

    subgraph AntiScraping["Anti-Scraping Middleware"]
        Headers["applyBrowserHeaders<br/>UA + Sec-Fetch-* + Sec-Ch-Ua-*"]
        Rate["waitRateLimit<br/>token bucket"]
        Jitter["jitter<br/>200–800ms"]
    end

    subgraph Wire["Network"]
        APNICFTP["ftp.apnic.net<br/>stats / IRR / transfers"]
        APNICRDAP["rdap.apnic.net"]
        APNICThyme["thyme.apnic.net<br/>BGP analysis"]
        APNICREx["api.rex.apnic.net"]
        APNICRRDP["rrdp.apnic.net"]
    end

    Get -->|"miss"| CacheBox
    CacheBox -->|"miss"| Fetch
    Get -->|"hit"| Caller
    Fetch --> FetchReader
    Fetch --> FetchText
    Fetch --> FetchTextStr
    Fetch --> FetchJSON

    FetchReader --> Probe
    Probe -->|"206 + Accept-Ranges"| Plan
    Plan --> Workers
    Workers --> Merge
    Workers -.->|"deadline"| Split
    Probe -->|"no Range / 200"| Single["singleStream"]
    Merge --> StreamParse
    Single --> StreamParse
    FetchReader --> StreamParse
    FetchText --> StrParse
    FetchTextStr --> StrParse
    FetchJSON --> JSONParse

    DoReqReaders["RRDP stream"] --> XMLParse

    FetchReader --> DoReq
    FetchText --> DoReq
    FetchTextStr --> DoReq
    FetchJSON --> DoReq
    Probe --> DoReq
    Workers --> DoReq
    Single --> DoReq
    DoReqReaders --> DoReq

    DoReq --> Headers
    Headers --> Rate
    Rate --> Jitter
    Jitter --> HTTPClient["httpClient.Do"]

    HTTPClient --> APNICFTP
    HTTPClient --> APNICRDAP
    HTTPClient --> APNICThyme
    HTTPClient --> APNICREx
    HTTPClient --> APNICRRDP
```

## Layer Responsibilities

| Layer | Source file | Responsibility |
|-------|-------------|----------------|
| **HTTP Client** | `internal/transport/client.go` | Holds all configuration, base URLs, and the `Option` functional-options pattern. `doHTTPRequest` is the single execution outlet. |
| **Anti-Scraping** | `internal/transport/stealth.go` | Browser-mimicry headers, token-bucket rate limiter, request jitter, and explicit gzip handling. Applied inside `doHTTPRequest`. |
| **Chunked Download** | `internal/transport/downloader.go` | Range-probe, chunk planning, concurrent worker pool, retry with slow-chunk splitting, `io.Pipe` merge. Used by `fetchReader`. |
| **Caching** | `internal/transport/cache.go` | `sync.RWMutex`-guarded map with per-key TTL. Backs every `Get*` method. |
| **Parser Design** | `internal/stats/fetcher.go`, `internal/query/bgp.go`, `internal/query/irr.go`, `internal/query/rrdp.go`, `internal/query/rdap.go`, `internal/query/rex.go` | Streaming vs. full-buffer parsers, boundary defense, error handling. |

## Design Principles

1. **One outlet.** All HTTP traffic — including the Range probe and every chunk — goes through `doHTTPRequest`, so stealth, rate limiting, and jitter apply uniformly and cannot be bypassed by a sub-path.
2. **Configurable, not optional.** Anti-scraping is on by default (`stealth: true`), but every knob is a functional option (`WithStealth`, `WithJitter`, `WithRateLimit`, `WithMaxConcurrentDownloads`, ...).
3. **Degrade gracefully.** Chunked download falls back to a single connection when the server does not honor `Range`; a stalled chunk is split in half and re-fetched on fresh connections rather than failing the whole download.
4. **Stream where it matters.** Multi-megabyte files (delegated stats, RRDP snapshots, IRR dumps) are parsed from an `io.Reader` so peak memory stays bounded; only small thyme BGP files are buffered into a string.
5. **Cache is opt-in per call.** `Get*` methods wrap `Fetch*` methods with a TTL cache; callers that need freshness call `Fetch*` directly.

## Pages in this section

- [HTTP Client](http-client.md) — `Client` struct, functional options, `doHTTPRequest` request lifecycle.
- [Anti-Scraping](anti-scraping.md) — browser headers, token-bucket limiter, jitter, gzip handling.
- [Chunked Download](chunked-download.md) — Range probe, chunk planning, worker pool, slow-chunk splitting.
- [Caching](caching.md) — `cache` struct, `Get*` vs `Fetch*`, TTL control.
- [Parser Design](parser-design.md) — streaming vs. full-buffer parsers, boundary defense, error handling.
