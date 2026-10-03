# apnic CLI 使用技能（SKILLS）

> 渐进式披露文档：先看摘要与典型场景，需要细节时展开下方折叠块。
> 本文档解释 `apnic` 命令行（`cmd/apnic`）**每一个子命令与参数**如何映射到 APNIC 数据服务。

---

## 一句话摘要

`apnic` 是基于 cobra 的命令行，封装了 APNIC 全部公开数据服务（统计文件、转移、变更、RDAP、whois、反查 DNS、数据校验、历史快照），所有命令共享一组全局标志，并支持 `--json` 机器可读输出与链式过滤。

## 适用场景

- **网络资源盘点**：查询某国/某组织在 APNIC 拥有的 IP 段与 ASN。
- **历史溯源**：取任意日期的统计快照，对比资源随时间的变迁。
- **真实性校验**：下载统计文件并用 APNIC 官方 MD5/PGP 签名验证完整性。
- **转移与变更追踪**：审计 IP/ASN 在 RIR 间或组织间的转移、状态变更。
- **自动化流水线**：用 `--json` 输出接入脚本/CI，配合 `jq`/`grep` 二次处理。

---

## 全局标志（所有子命令通用）

| 标志 | 默认 | 说明 |
|------|------|------|
| `--stats-base-url <url>` | `https://ftp.apnic.net/apnic/stats/apnic/` | 统计/FTP 数据根地址 |
| `--rdap-base-url <url>` | `https://rdap.apnic.net` | RDAP 服务根地址 |
| `--whois-server <addr>` | `whois.apnic.net:43` | whois TCP 服务器（host:port） |
| `--user-agent <str>` | `apnic-skills/1.0` | 自定义 User-Agent（stealth 关闭时使用） |
| `--cache-ttl <dur>` | `30m` | 缓存有效期（如 `30m`/`2h`，`0` 禁用） |
| `--timeout <dur>` | 无（HTTP 客户端默认） | HTTP 请求超时（如 `30s`/`2m`） |
| `--json` | `false` | 以缩进 JSON 输出（适用支持 JSON 的命令） |
| `--stealth` | `true` | 浏览器伪装头 + 请求抖动，避免被识别为爬虫 |
| `--browser-ua <str>` | Chrome UA | stealth 开启时使用的浏览器 UA |
| `--jitter <range>` | `200ms-800ms` | 每请求随机延迟区间（stealth 开启时生效） |
| `--rate-limit <n>` | `0`（不限） | 全局每秒最大请求数（令牌桶） |
| `--ftp-base-url <url>` | `https://ftp.apnic.net/` | IRR/transfers-all/telemetry 的 FTP 根 |
| `--rrdp-base-url <url>` | `https://rrdp.apnic.net` | RPKI RRDP 根地址 |
| `--thyme-base-url <url>` | `https://thyme.apnic.net` | thyme BGP 分析根地址 |
| `--bgp-source <src>` | `current` | thyme BGP 数据源：`current`（全球）/`au`（Brisbane）/`hk`（HKIX） |
| `--rex-base-url <url>` | `https://api.rex.apnic.net` | REx 跨 RIR 资源注册库根地址 |
| `--max-concurrent-downloads <n>` | `4` | 大文件并行 Range 请求数（0/1 禁用分块，回退单连接） |
| `--chunk-size <sz>` | `2MiB` | 每块字节数（如 `1MB`/`512KB`；默认按 2MiB 自适应切块） |
| `--download-timeout <dur>` | 继承 `--timeout` | 单块下载超时（如 `300s`，建议 ≥ 单块大小/8KBps） |

> 全局标志可置于子命令前或后：`apnic --json delegated` 与 `apnic delegated --json` 等价。

> 大文件分块：APNIC FTP 对单连接限速 ~8-22 KB/s，50MB 的 IRR dump 单连接需 ~40 分钟。`--max-concurrent-downloads 4`（默认）将文件切成 ~2MB 小块以 4 并发轮转下载，总吞吐提升 3-4 倍；每个分块请求仍继承 stealth 伪装头与限速。遇单块超时可调小 `--chunk-size`（如 `1MB`）或调大 `--download-timeout`。

---

## 命令总览

```
apnic
├── delegated           # 标准版分配记录
├── extended            # 扩展版（含组织 opaque-id）
├── assigned            # 按前缀大小聚合的分配统计
├── ipv6-assigned       # 逐条 IPv6 分配记录
├── legacy              # 历史遗留资源
├── transfers           # IP/ASN 转移记录（每日 JSON 快照）
├── transfers-all       # 累积转移全集（管道分隔，自 2010）
├── changes             # 资源变更记录（JSON Lines）
├── stats-telemetry     # whois/RDAP 服务查询遥测
├── irr                 # IRR 数据库转储（RPSL，19 类）/ serial
├── bgp                 # thyme BGP 路由表（summary/raw-table/asn-map + 5 附加文件，支持 --bgp-source au/hk）
├── rpki                # RPKI/RRDP（notification/snapshot）
├── rex                 # REx 跨 RIR 资源注册库（network/resources/holder/count）
├── years               # 列出可用历史年份
├── history             # 按日期/年份取历史快照
├── filter              # 链式过滤 delegated/extended
├── rdap                # RDAP 查询（ip/cidr/asn/domain/entity/search/help/domains）
├── whois               # whois 查询（ip/asn/raw）
├── reverse-dns         # IP 反向 DNS
└── verify              # 数据完整性校验（md5/asc/pubkey/integrity）
```

<details>
<summary><b>📊 统计文件类命令详解</b>（delegated / extended / assigned / ipv6-assigned / legacy）</summary>

这五个命令结构一致，均支持 `--date YYYYMMDD`（默认取 latest）与 `--json`。

### `apnic delegated`
获取标准版分配记录（RIR 统计交换格式：`registry|cc|type|start|value|date|status`）。
```bash
apnic delegated                      # 最新
apnic delegated --date 20240101      # 2024-01-01 快照（自动从 {year}/ 子目录取 .gz）
apnic delegated --json               # JSON 输出（含 header/summary）
```
默认输出列：`Country  Type  Start  Value  Status  Date`。

### `apnic extended`
扩展版，每条记录额外携带 `opaque-id`（资源持有组织标识）。
```bash
apnic extended --json
apnic extended --date 20240101
```
默认输出列：`Country  Type  Start  Value  Status  OpaqueID`。

### `apnic assigned`
按前缀大小聚合的分配统计（每个国家每种前缀长度有多少条分配）。
```bash
apnic assigned
```
默认输出列：`Country  Type  Prefix  Count`。

### `apnic ipv6-assigned`
逐条 IPv6 分配记录（`registry|cc|ipv6|start|prefix|date`，无 status 列）。
```bash
apnic ipv6-assigned
```
默认输出列：`Country  Start  PrefixLength`。

### `apnic legacy`
历史遗留资源（在现行 RIR 统计框架建立前转入 APNIC 的地址空间）。
```bash
apnic legacy
```
默认输出列：`Country  Type  Start  Value  Status`。

> **URL 命名约定**：latest 文件在根目录未压缩（如 `delegated-apnic-latest`）；日期文件在 `{year}/` 子目录且为 `.gz`（如 `2024/delegated-apnic-20240101.gz`）。SDK 自动透明 gzip 解压。

</details>

<details>
<summary><b>🔁 转移、变更与遥测</b>（transfers / transfers-all / changes / stats-telemetry / years）</summary>

### `apnic transfers`
获取 IP/ASN 转移记录。默认取 `transfers_latest.json`（JSON 格式）。
```bash
apnic transfers                 # 最新（JSON）
apnic transfers --year 2023     # 2023 年 JCR 格式转移日志
apnic transfers --json
```
`--year YYYY` 切换到该年度的 JCR 日志文件。

### `apnic changes`
资源变更记录（JSON Lines，每行一条 `delegated`/`cc-changed`/`status-changed` 事件）。
```bash
apnic changes                   # 最新
apnic changes --date 20240101   # 指定日期快照
apnic changes --json
```

### `apnic transfers-all`
累积转移全集（管道分隔格式，覆盖自 2010 年起的全部 IP/ASN 转移）。与 `transfers`（每日 JSON 快照）互补。
```bash
apnic transfers-all                     # 最新累积全集
apnic transfers-all --date 20220904     # 当日归档快照
apnic transfers-all --json
```
输出列为 `resource_type|resource|from_organisation|from_economy|from_rir|previous_delegation_date|to_organisation|to_economy|to_rir|transfer_date|transfer_type`。

### `apnic stats-telemetry`
whois/RDAP 服务查询遥测（每小时发布的 `whois-rdap-stats.json`）：查询总量、按类型分布（ip/autnum/entity/domain/*_history）、Top 查询 ASN。
```bash
apnic stats-telemetry                  # 最新
apnic stats-telemetry --date 20260701  # 当日归档
apnic stats-telemetry --json
```

### `apnic years`
列出可获取历史统计的年份（≥2001）。
```bash
apnic years
apnic years --json
```

</details>

<details>
<summary><b>📚 历史快照</b>（history）</summary>

### `apnic history`
按日期或年份取历史统计快照。`--date` 与 `--year` 互斥，必须二选一。

| 参数 | 说明 |
|------|------|
| `--type <t>` | `delegated`（默认）/ `extended` / `assigned` / `legacy` |
| `--date YYYYMMDD` | 取该日期快照（四类均支持） |
| `--year YYYY` | 取该年度最新文件（仅 `delegated`/`extended` 支持） |

```bash
apnic history --type delegated --date 20240101
apnic history --type extended --year 2023
apnic history --type legacy --date 20240101 --json
```
- `--year` 用于 `assigned`/`legacy` 会报错。
- 日期快照走 `{year}/{name}.gz`，年度文件走 `{year}/delegated-apnic-{year}1231.gz`。

</details>

<details>
<summary><b>🔍 链式过滤</b>（filter）</summary>

### `apnic filter`
拉取最新 delegated/extended 后以 AND 语义链式过滤。

| 参数 | 说明 |
|------|------|
| `--source <s>` | `delegated`（默认）/ `extended` |
| `--country <cc>` | ISO 3166 国家码（如 `CN`） |
| `--type <t>` | `ipv4` / `ipv6` / `asn` |
| `--status <s>` | `allocated` / `assigned` / `reserved` / `available` |
| `--opaque-id <id>` | 组织标识（仅 extended） |

```bash
apnic filter --source delegated --country CN --type ipv4 --status allocated
apnic filter --source extended --country JP --opaque-id A92E1062 --json
```

</details>

<details>
<summary><b>🌐 RDAP 查询</b>（rdap ip/cidr/asn/domain/entity/search/help/domains）</summary>

RDAP 子命令返回结构化注册数据。除 `search` 外，所有 lookup 子命令支持 `--date`（RFC3339，如 `2020-06-01T00:00:00Z`）做**点对点历史查询**，返回该 UTC 时刻的资源状态（APNIC `history_version_0` 扩展）。

### `apnic rdap ip <ip>`
```bash
apnic rdap ip 1.1.1.1
apnic rdap ip 1.1.1.1 --date 2020-06-01T00:00:00Z --json
```

### `apnic rdap cidr <cidr>`
```bash
apnic rdap cidr 1.1.1.0/24
apnic rdap cidr 2001:db8::/32       # IPv6
```

### `apnic rdap asn <asn>`
ASN 传纯数字（如 `13335`）。
```bash
apnic rdap asn 13335
```

### `apnic rdap domain <domain>`
通常用于反向 DNS 域名。
```bash
apnic rdap domain 1.0.0.1.in-addr.arpa
```

### `apnic rdap entity <handle>`
实体/联系人查询（如 `ORG-ARAD1-AP`、`AIC3-AP`）。
```bash
apnic rdap entity AIC3-AP
```

### `apnic rdap search <query>`
按名称（`fn`）或 handle 搜索实体。`fn` 需通配符做子串匹配（`*CLOUD*`），精确名只匹配同名实体。

| 参数 | 说明 |
|------|------|
| `--field <f>` | `fn`（默认，名称搜索，支持 `*` 通配）/ `handle`（精确 handle） |

```bash
apnic rdap search "*CLOUD*"                  # 名称模糊搜索
apnic rdap search AIC3-AP --field handle     # 精确 handle
```

> **端点**：`/entities?fn=<q>`（名称）/ `/entities?handle=<q>`（精确），遵循 RFC 7482。

### `apnic rdap help`
RDAP `/help` 端点：服务能力声明（`rdapConformance` 扩展，如 `history_version_0`/`cidr0`/`nro_rdap_profile_0`）与通知（服务条款、误差报告入口）。
```bash
apnic rdap help
apnic rdap help --json
```

### `apnic rdap domains <name>`
按名称搜索 in-addr.arpa 反向 DNS 域（`/domains?name=<q>`，RFC 7482）。
```bash
apnic rdap domains 1       # 匹配 1.in-addr.arpa 等
apnic rdap domains 1 --json
```

</details>

<details>
<summary><b>📟 whois 与反向 DNS</b>（whois / reverse-dns）</summary>

### `apnic whois ip <ip>`
解析后的 whois 查询（Network/CIDR/Country/Org/Parent/Created/Updated）。
```bash
apnic whois ip 1.1.1.1 --json
```

### `apnic whois asn <asn>`
ASN 可传 `13335` 或 `AS13335`（自动去 `AS` 前缀）。
```bash
apnic whois asn AS13335
```

### `apnic whois raw <query>`
原始 whois 文本（不解析）。
```bash
apnic whois raw 1.1.1.1
```

### `apnic reverse-dns <ip>`
IP 反向 DNS（PTR 记录）。
```bash
apnic reverse-dns 1.1.1.1
```
无 PTR 时输出 `(no PTR records)`。

</details>

<details>
<summary><b>🗄️ IRR 数据库转储</b>（irr &lt;type&gt; / irr serial）</summary>

### `apnic irr <type>`
获取并解析 APNIC IRR（RPSL）数据库转储（`apnic.db.<type>.gz`）。`<type>` 取自 19 类对象：`as-block`、`as-set`、`aut-num`、`domain`、`filter-set`、`inet6num`、`inetnum`、`inet-rtr`、`irt`、`key-cert`、`limerick`、`mntner`、`organisation`、`peering-set`、`role`、`route`、`route6`、`route-set`、`rtr-set`。
```bash
apnic irr inetnum          # IPv4 inetnum 对象（含组织/国家/联系人）
apnic irr aut-num          # ASN 自治系统对象
apnic irr route            # 路由对象（前缀 + origin ASN）
apnic irr domain           # 反向 DNS 委派（x.in-addr.arpa + nserver + zone-c）
apnic irr inetnum --json
```
默认走缓存（`--cache-ttl`），重复调用在 TTL 内不重复请求。每个对象解析为 `{Type, PrimaryKey, Attributes}`，多值属性（如 `descr`）保留全部出现顺序；续行（行首空白，可选 `+` 抑制额外空格）折叠到上一属性。

### `apnic irr serial`
获取 `APNIC.CURRENTSERIAL`——IRR 数据库当前序号，可用于判断自上次同步以来的变更量。
```bash
apnic irr serial
apnic irr serial --json    # {"serial": <n>}
```

</details>

<details>
<summary><b>🌍 thyme BGP 路由表</b>（bgp summary / raw-table / asn-map）</summary>

### `apnic bgp summary`
thyme `data-summary`：BGP 路由表分析指标（冒号键值），包括路由条目数、最大聚合前缀数、ROA 覆盖（valid/invalid/none）、AS 总数、平均 AS 路径长度、已宣告地址空间占比等。
```bash
apnic bgp summary
apnic bgp summary --json
```

### `apnic bgp raw-table`
thyme `data-raw-table`：每一条已宣告路由，格式 `prefix\tASN`。
```bash
apnic bgp raw-table        # 默认预览前 50 条
apnic bgp raw-table --json
```

### `apnic bgp asn-map`
按 origin ASN 聚合 raw table（客户端本地派生，无额外网络请求），返回每个 origin ASN 宣告的前缀列表。
```bash
apnic bgp asn-map          # 输出唯一 origin ASN 数量
apnic bgp asn-map --json
```

### `apnic bgp bad-prefixes`
thyme `data-badpfx-nos`：长度超过 /24 的前缀及其 origin AS（疑似路由泄漏或误宣告）。
```bash
apnic bgp bad-prefixes
apnic bgp bad-prefixes --bgp-source au    # Brisbane 视图
```

### `apnic bgp per-prefix-length`
thyme `data-pfx-nos`：按前缀长度统计已宣告前缀数（/N:count 网格布局，逐 token 解析）。
```bash
apnic bgp per-prefix-length
apnic bgp per-prefix-length --bgp-source hk    # HKIX 视图
```

### `apnic bgp used-autnums`
thyme `data-used-autnums`：所有在用 ASN 及注册名、国家码（`ASN Name - Description, CC`，预览前 50 条）。
```bash
apnic bgp used-autnums
apnic bgp used-autnums --json
```

### `apnic bgp spar-prefixes`
thyme `data-spar`：特殊用途地址注册表（RFC 6890 保留空间）前缀及其 origin AS 与描述。
```bash
apnic bgp spar-prefixes
apnic bgp spar-prefixes --bgp-source au
```

### `apnic bgp single-pfx`
thyme `data-singlepfx`：宣告少于 20 个前缀的 ASN 计数，按 RIR 分组（`PrefixCount / ASNCount / RIR`）。
```bash
apnic bgp single-pfx
apnic bgp single-pfx --bgp-source hk
```

> 以上 5 个附加文件均支持 `--bgp-source current|au|hk` 选择 thyme 数据源（默认 `current` 全球视图）。

</details>

<details>
<summary><b>🔑 RPKI / RRDP</b>（rpki notification / snapshot）</summary>

### `apnic rpki notification`
获取 RRDP `notification.xml`：当前 session_id、serial、snapshot 引用（URI + SHA-256）与最近 deltas 列表（默认预览前 20 条）。
```bash
apnic rpki notification
apnic rpki notification --json
```

### `apnic rpki snapshot [uri]`
流式解析 RRDP snapshot.xml，统计 `<publish>`/`<withdraw>` 对象数量。URI 可省略（自动从 notification 解析）、传相对路径（拼 `--rrdp-base-url`）或绝对 URI（取自 notification 输出）。base64 CMS body 在流式解析时丢弃，内存占用有界。
```bash
apnic rpki snapshot                          # 自动从 notification 解析 snapshot URI
apnic rpki snapshot snapshot.xml             # 相对路径
apnic rpki snapshot <绝对URI> --json
```

> RRDP 响应为 `Content-Encoding: gzip`，SDK 透明解压（notification 经 fetchText，snapshot 经流式 gzip.Reader），无需双重解压。

</details>

<details>
<summary><b>🌐 REx 跨 RIR 资源注册库</b>（rex network / resources / holder / count）</summary>

REx（Resource EXplorer，`api.rex.apnic.net/v1/*`）将五大 RIR 的委派资源聚合为统一视图，并按持有组织（opaqueId）归集——能力超出各 RIR 独立 stats/RDAP。公开 JSON，免认证，复用统一 HTTP 出口（自动获益于反爬伪装）。

### `apnic rex network`
自定位网络：按调用方源 IP 返回覆盖前缀、起源 ASN、经济体代码（ISO 国家码）。无需参数。
```bash
apnic rex network
apnic rex network --json
```

### `apnic rex resources [type]`
跨 RIR 最近委派资源视图（带持有者归因：opaqueId、holderName、rir、cc、delegationDate）。`type` 可为 `ipv4`/`ipv6`/`asn` 或省略。返回最近委派的有限窗口（非全量历史）。
```bash
apnic rex resources            # 全部类型
apnic rex resources ipv4
apnic rex resources asn --json
```

### `apnic rex holder <opaqueId> <rir>`
按 opaqueId 聚合某组织持有的全部 ASN 与前缀，含规模指标（`ipv4_24Count` 以 /24 为单位、`ipv6_48Count` 以 /48 为单位）。`rir` 取值：`afrinic`/`apnic`/`arin`/`lacnic`/`ripencc`（RIPE NCC 代码是 `ripencc`，不是 `ripe`）。opaqueId 可从 `rex resources` 或扩展 delegated stats 获取。
```bash
apnic rex holder 522be47e60b5c2ef81bbbab8deaa6b85 arin
apnic rex holder <opaqueId> ripencc --json
```

### `apnic rex count`
全 RIR 去重持有者总数（跨五大 RIR 的独立资源持有组织数）。
```bash
apnic rex count
apnic rex count --json
```

> REx 走 HTTPS 公开 API；缺失必填参数时服务端返回纯文本错误（如 holder 缺 opaqueId/rir），SDK 将其作为错误透传而非 JSON 解码失败。

</details>

<details>
<summary><b>🔐 数据完整性校验</b>（verify md5/asc/pubkey/integrity）</summary>

APNIC 为每个统计文件发布 `.md5` 与 `.asc`（PGP 签名）旁挂文件，并用 `CURRENT_PUBLIC_KEY` 签名。

### `apnic verify md5`
获取某统计文件的 MD5 校验值（BSD 风格 `MD5 (file) = <hash>` 与 GNU 风格皆可解析）。
```bash
apnic verify md5 --type delegated
apnic verify md5 --type delegated-extended --date 20240101
```
`--type` 取值：`delegated` / `delegated-extended` / `assigned` / `delegated-ipv6-assigned` / `legacy`。

### `apnic verify asc`
获取 PGP 签名（`.asc`）。
```bash
apnic verify asc --type delegated
```

### `apnic verify pubkey`
获取 APNIC 签名公钥（`CURRENT_PUBLIC_KEY`）。
```bash
apnic verify pubkey
```

### `apnic verify integrity`
端到端校验：下载数据文件 + 其 MD5，本地计算并比对。匹配输出 `OK: ... MD5 verified`，不匹配或失败则非零退出。
```bash
apnic verify integrity --type delegated
apnic verify integrity --type delegated --date 20240101
```

</details>

---

## 输出格式说明

- **默认**：人类可读的 TSV（制表符分隔），首行以 `#` 注释给出条目数与日期/年份。
- **`--json`**：缩进 JSON。统计类返回 `*Result`（含 `Entries` + header/summary 元数据）；RDAP 返回原始 RDAP 对象；whois 返回 `WhoisInfo`。
- **退出码**：成功 `0`，任何 `RunE` 返回错误（网络失败、参数校验失败、MD5 不匹配）则 `1`。

## 构建与运行

```bash
# 构建 CLI 到 ./bin/apnic
go build -o bin/apnic ./cmd/apnic

# 直接运行
go run ./cmd/apnic delegated --json | jq '.Entries | length'

# 运行测试（SDK 100%，CLI 98.2%，仅 main() 不可单元测试）
go test ./...
```

## 相关

- SDK API 总览：见仓库根 `README.md`。
- 工作流封装（多命令组合编排）：见 [workflows.md](./workflows.md)。
