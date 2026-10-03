# IRR Types

The IRR (Internet Routing Registry) family models APNIC's RPSL (Routing Policy Specification Language) database dumps, published as gzipped text files at `https://ftp.apnic.net/apnic/whois/apnic.db.<type>.gz`. Each dump contains objects of a single RPSL type; the SDK parses them into a generic `IRRObject` representation rather than one struct per type.

All types live in [`internal/models/models.go`](https://github.com/hitechcloud-vietnam/apnic-skills/blob/main/internal/models/models.go); the canonical type list is `IRRObjectTypes` in [`internal/query/irr.go`](https://github.com/hitechcloud-vietnam/apnic-skills/blob/main/internal/query/irr.go).

## Class Diagram

```mermaid
classDiagram
    class IRRDatabase {
        +string Type
        +[]IRRObject Objects
    }

    class IRRObject {
        +string Type
        +string PrimaryKey
        +map~string,[]string~ Attributes
    }

    class IRRObjectTypes {
        <<enum of 19 string values>>
        as-block
        as-set
        aut-num
        domain
        filter-set
        inet6num
        inetnum
        inet-rtr
        irt
        key-cert
        limerick
        mntner
        organisation
        peering-set
        role
        route
        route6
        route-set
        rtr-set
    }

    IRRDatabase --> IRRObject
    IRRObject ..> IRRObjectTypes : Type is one of
```

## `IRRObject`

A single RPSL object. In the source file the first attribute of an object is its type (e.g. `inetnum:`) and its value is the object's primary key; the SDK preserves this convention.

| Field | Type | Description |
|-------|------|-------------|
| `Type` | `string` | RPSL object type, e.g. `"inetnum"`, `"aut-num"`. Always one of `IRRObjectTypes`. |
| `PrimaryKey` | `string` | Value of the type attribute — the object's unique key within the dump. |
| `Attributes` | `map[string][]string` | Attribute name → values, in file order. The key attribute is included here as well, so the type name can be looked up under `Attributes[<type>]`. Multi-valued attributes (e.g. `descr`, `mnt-by`) collect every occurrence in slice order. |

Continuation lines (lines beginning with whitespace, optionally `+`) are folded into the preceding attribute's last value, exactly as RPSL specifies.

## `IRRDatabase`

Holds the parsed objects from one dump.

| Field | Type | Description |
|-------|------|-------------|
| `Type` | `string` | The object type this database holds (echoes `IRRObject.Type` for every object). |
| `Objects` | `[]IRRObject` | Parsed RPSL objects, in file order. |

## The 19 RPSL object types

`IRRObjectTypes` lists every APNIC IRR object type published as a gzipped dump. Pass any of these to `FetchIRRDatabase` / `GetIRRDatabase`.

| Type | Purpose |
|------|---------|
| `as-block` | A block of AS numbers delegated as a unit. |
| `as-set` | A named set of AS numbers (used in routing policy). |
| `aut-num` | An Autonomous System object — routing policy of one AS. |
| `domain` | A reverse DNS domain (in-addr.arpa / ip6.arpa). |
| `filter-set` | A named set of route filters. |
| `inet6num` | An IPv6 address range. |
| `inetnum` | An IPv4 address range. |
| `inet-rtr` | An Internet router (router identity + peers). |
| `irt` | An IR trust object (CERT contact). |
| `key-cert` | A PGP key certificate. |
| `limerick` | Legacy object type (mostly historical). |
| `mntner` | A maintainer object — controls who can modify child objects. |
| `organisation` | An organization record. |
| `peering-set` | A named set of peerings. |
| `role` | A role contact (group, not a person). |
| `route` | An IPv4 route object (prefix + origin AS). |
| `route6` | An IPv6 route object. |
| `route-set` | A named set of routes. |
| `rtr-set` | A named set of routers. |

## Parse flow

```mermaid
sequenceDiagram
    participant SDK as SDK
    participant FTP as ftp.apnic.net
    participant Parser as parseIRRDatabase

    SDK->>FTP: GET apnic.db.<type>.gz
    FTP-->>SDK: gzip stream
    SDK->>SDK: gzip decompress
    SDK->>Parser: io.Reader + type
    loop each object
        Parser->>Parser: read type attribute (first line)
        Parser->>Parser: capture PrimaryKey = value
        loop each subsequent attribute
            alt continuation line (whitespace/+ prefix)
                Parser->>Parser: fold into previous value
            else new "key: value"
                Parser->>Parser: append to Attributes[key]
            end
        end
        Parser->>Parser: emit IRRObject
    end
    Parser-->>SDK: IRRDatabase{Type, Objects}
```
