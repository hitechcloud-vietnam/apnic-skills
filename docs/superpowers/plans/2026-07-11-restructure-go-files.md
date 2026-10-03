# 根目录 Go 文件按职责重组进子目录 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`
> Steps use checkbox (`- [ ]`) syntax.

**Goal:** 将根目录下 27 个 `package apnic` 的 `.go` 文件（+ 28 个测试）从平铺结构重组进按职责划分的子目录，同时保证 `cmd/apnic` 的 import 路径 `github.com/hitechcloud-vietnam/apnic-skills` 与所有 `apnic.Xxx` 引用完全不变、所有现有测试持续通过、website/docs 文件名引用同步修正。

**Architecture:** Go 的目录 = package 边界（与 TS/JS 不同，同一 package 不能跨目录）。因此本重组采用「分层子包 + 根 re-export」方案：

- **数据流向**：`cmd/apnic` → `import apnic "github.com/hitechcloud-vietnam/apnic-skills"`（根包，路径不变）→ 根包通过 type alias / 函数 re-export 暴露子包符号 → 子包按职责分层。
- **关键组件**：根目录保留 `client.go`（`Client` 定义，含所有子包字段）+ `doc.go`（包文档）+ `reexport.go`（把子包的导出类型/函数/错误以 alias 形式重新导出到根 `package apnic`，使 CLI 的 `apnic.Client`、`apnic.NewClient`、`apnic.WithChunkSize`、`apnic.NewFilter` 等全部沿用旧路径）。子目录划分：`internal/stats/`（delegated/extended/assigned/ipv6/legacy 解析器）、`internal/query/`（rdap/rex/whois/irr/rrdp/bgp/telemetry/transfers/changes 查询）、`internal/filter/`（cidr/filter 链式过滤）、`internal/history/`（历史快照）、`internal/transport/`（client 核心 + downloader + stealth + cache + dns + errors + utils + verify）、`internal/models/`（所有 struct 类型定义）。
- **为什么这样做**：用 `internal/` 确保子包不被外部项目导入（API 收敛）；根包 re-export 保证 `cmd/apnic` 零改动、用户 SDK 调用零改动；未导出符号（`parseDelegatedFull`、`doHTTPRequest`、`fetchTextStr`、`cacheKeyIRR`、`defaultLookupAddr` 等）随其归属文件移入子包后，因同子包内仍可互访，仅需把跨子包调用的小写符号提升为导出（加 `Internal` 后缀或移入 transport 子包共享），改动可控。

**Tech Stack:** Go 1.25.0, module `github.com/hitechcloud-vietnam/apnic-skills`, cobra v1.10.2 CLI, 标准库 `net/http`/`compress/gzip`/`encoding/xml`/`encoding/json`, go test + `go:build e2e` 标签

**Risks:**
- **跨子包未导出符号耦合**：`doHTTPRequest`(stealth)、`fetchTextStr`(downloader)、`parseDelegatedFullFromString`(fetcher) 等被多个文件调用 → 缓解：Task 2 在移动前先识别所有跨文件未导出符号，凡被「将落入不同子包」的文件调用者，提升为导出（`DoHTTPRequest`、`FetchTextStr`、`ParseDelegatedFullFromString`），并在根 re-export 中暴露；同子包内调用的保持小写。
- **CLI import 路径漂移**：`cmd/apnic` 用 `apnic "github.com/hitechcloud-vietnam/apnic-skills"` 引用了约 20 个导出符号 → 缓解：Task 3 根 `reexport.go` 必须覆盖 CLI 实际用到的每一个符号，Task 5 全量 `go build ./cmd/apnic` 验证。
- **测试文件移动后编译断裂**：`_test.go` 访问未导出符号 → 缓解：每个 `_test.go` 必须与其被测文件移入同一子包；`test_helpers_test.go` 被多文件复用 → 移入 `internal/transport/` 并按需拆分。
- **website/docs 文件名引用失效**：约 70 处 `xxx.go` 引用 → 缓解：Task 4 用 `grep -rl` 逐一替换为新路径，文件名本身不变所以多数引用只需补全目录前缀。
- **`apnic` 二进制 / coverage 产物**：已 gitignore，无需处理。

---

### Task 1: 移动前基线与跨文件未导出符号清单

**Depends on:** None
**Files:**
- Create: `docs/superpowers/plans/.baseline-unexported-symbols.txt`
- Run: `go test ./...` 生成基线

- [x] **Step 1: 记录当前测试基线 — 确认重组前全绿，作为回归对照**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
go vet ./... && go test ./... -count=1 2>&1 | tee /tmp/apnic-baseline.txt
```

Expected:
  - Exit code: 0
  - `/tmp/apnic-baseline.txt` 末尾包含 `ok` 行（每包一行）
  - Output does NOT contain: `FAIL` 或 `build failed`

- [x] **Step 2: 生成跨文件未导出符号清单 — 识别哪些小写符号被不同文件调用，决定移动后是否需提升为导出**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
# 对每个未导出顶层符号，找出定义文件与所有调用文件
for sym in doHTTPRequest fetchTextStr fetchReader fetchChunkRaw downloadChunked singleStream parseDelegatedFull parseDelegatedFullFromString parseExtendedFull parseExtendedFullFromString parseAssignedFull parseLegacyFull parseIPv6AssignedFull parseDelegatedData parseTransfersAll parseTransfersData parseChangesData parseRRDPNotification parseRPKISnapshot parseBGPSummary parseBGPRawTable parseBGPBadPrefixes parseBGPPerPrefixLength parseBGPUsedAutnums parseBGPSparPrefixes parseBGPSinglePfx parseIRRDatabase parseMD5Checksum parseWhoisDate parseWhoisResponse parseIPv4Count parseIPv6Prefix parseASNValue parseASNCount parseStatsHeader parseSummaryLine parseOpaqueID isSummaryLine isHeaderLine isMD5Hex isIRRObjectType isRExAPIError isDeadlineError statsFileName buildStatsURL buildStatsMD5URL buildStatsASCURL buildTransfersAllURL buildTransfersAllSidecarURL buildTelemetryURL buildTelemetrySidecarURL buildIRRDBURL buildIRRCurrentSerialURL buildThymeURL buildRExURL cacheKeyIRR newCache newRateLimiter defaultLookupAddr bytesJoin localName attrLocal effectiveConcurrency planChunks probeRange; do
  def=$(grep -lE "func ${sym}\b|func.*\b${sym}\b|type ${sym}\b" *.go 2>/dev/null | grep -v _test.go | head -1)
  callers=$(grep -lE "\b${sym}\b" *.go 2>/dev/null | grep -v _test.go | grep -v "^${def}$" | tr '\n' ' ')
  [ -n "$callers" ] && printf "%-32s def=%-18s callers=%s\n" "$sym" "$def" "$callers"
done | tee docs/superpowers/plans/.baseline-unexported-symbols.txt
```

Expected:
  - Exit code: 0
  - 生成清单文件，列出每个被跨文件调用的未导出符号及其定义文件/调用文件
  - 该清单是 Task 2 提升-导出决策的依据

- [x] **Step 3: 记录 cmd/apnic 实际引用的根包导出符号全集 — reexport.go 必须覆盖这些符号**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
grep -rhoE "apnic\.[A-Z][A-Za-z0-9]+" cmd/apnic/*.go | sort -u | tee docs/superpowers/plans/.baseline-cli-symbols.txt
```

Expected:
  - Exit code: 0
  - 输出 CLI 实际用到的所有 `apnic.Xxx` 符号（约 20 个），Task 3 的 reexport.go 逐一对照

- [x] **Step 4: 提交**

```bash
git add docs/superpowers/plans/.baseline-unexported-symbols.txt docs/superpowers/plans/.baseline-cli-symbols.txt && \
git commit -m "chore(refactor): record pre-restructure baseline (tests + unexported symbol map + cli symbol set)"
```

---

### Task 2: 创建子目录骨架并移动文件（按职责分包，成对移动 _test.go）

**Depends on:** Task 1
**Files:**
- Create dirs: `internal/models/`, `internal/transport/`, `internal/stats/`, `internal/query/`, `internal/filter/`, `internal/history/`
- Move (git mv) 27 个非测试 .go + 28 个 _test.go 到对应子目录

**子目录分组方案：**

| 子目录 | package | 文件（非测试） | 理由 |
|--------|---------|---------------|------|
| `internal/models/` | `models` | `models.go` | 所有 struct 类型集中，被全包共享 |
| `internal/transport/` | `transport` | `client.go`, `downloader.go`, `stealth.go`, `cache.go`, `dns.go`, `errors.go`, `utils.go`, `verify.go` | HTTP 传输层：Client 定义、分块下载、限速/隐身、缓存、DNS、错误、URL 构造、校验 |
| `internal/stats/` | `stats` | `fetcher.go`, `extended.go`, `assigned.go`, `ipv6_assigned.go`, `legacy.go` | delegated stats 五种变体的抓取与解析 |
| `internal/query/` | `query` | `rdap.go`, `rex.go`, `whois.go`, `irr.go`, `rrdp.go`, `bgp.go`, `telemetry.go`, `transfers.go`, `changes.go` | 各类查询服务（RDAP/REx/Whois/IRR/RRDP/BGP/telemetry/transfers/changes） |
| `internal/filter/` | `filter` | `cidr.go`, `filter.go` | 链式过滤与 CIDR 计算 |
| `internal/history/` | `history` | `history.go` | 历史快照按日期/年份查询 |

- [x] **Step 1: 创建六个子目录骨架 — 建立分包物理结构**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
mkdir -p internal/models internal/transport internal/stats internal/query internal/filter internal/history
```

Expected:
  - Exit code: 0
  - `ls internal/` 输出六个子目录

- [x] **Step 2: git mv models 层文件 — 类型定义独立成包，成对移动测试**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
git mv models.go internal/models/models.go
git mv models_test.go internal/models/models_test.go
# 子目录文件改 package 名
sed -i 's/^package apnic$/package models/' internal/models/models.go internal/models/models_test.go
```

Expected:
  - Exit code: 0
  - `head -1 internal/models/models.go` 输出 `package models`

- [x] **Step 3: git mv transport 层文件 — Client 核心与传输基础设施成包**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
for f in client.go downloader.go stealth.go cache.go dns.go errors.go utils.go verify.go \
         client_test.go downloader_test.go stealth_test.go cache_test.go dns_test.go \
         errors_test.go utils_test.go verify_test.go fetchText_test.go downloader_e2e_test.go \
         test_helpers_test.go; do
  git mv "$f" "internal/transport/$f"
done
sed -i 's/^package apnic$/package transport/' internal/transport/*.go
```

Expected:
  - Exit code: 0
  - `ls internal/transport/` 列出 18 个文件，`head -1 internal/transport/client.go` 输出 `package transport`

- [x] **Step 4: git mv stats 层文件 — delegated stats 五变体成包**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
for f in fetcher.go extended.go assigned.go ipv6_assigned.go legacy.go \
         fetcher_test.go extended_test.go assigned_test.go ipv6_assigned_test.go legacy_test.go; do
  git mv "$f" "internal/stats/$f"
done
sed -i 's/^package apnic$/package stats/' internal/stats/*.go
```

Expected:
  - Exit code: 0
  - `head -1 internal/stats/fetcher.go` 输出 `package stats`

- [x] **Step 5: git mv query 层文件 — 九类查询服务成包**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
for f in rdap.go rex.go whois.go irr.go rrdp.go bgp.go telemetry.go transfers.go changes.go \
         rdap_test.go rex_test.go whois_test.go irr_test.go rrdp_test.go bgp_test.go \
         telemetry_test.go transfers_test.go changes_test.go; do
  git mv "$f" "internal/query/$f"
done
sed -i 's/^package apnic$/package query/' internal/query/*.go
```

Expected:
  - Exit code: 0
  - `head -1 internal/query/rdap.go` 输出 `package query`

- [x] **Step 6: git mv filter 与 history 层文件**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
for f in cidr.go filter.go cidr_test.go filter_test.go; do
  git mv "$f" "internal/filter/$f"
done
sed -i 's/^package apnic$/package filter/' internal/filter/*.go

git mv history.go internal/history/history.go
git mv history_test.go internal/history/history_test.go
sed -i 's/^package apnic$/package history/' internal/history/*.go
```

Expected:
  - Exit code: 0
  - 四个目录各含其文件，package 名分别为 `filter`、`history`

- [x] **Step 7: 验证文件移动完整性 — 确认根目录已无 .go 文件，子目录 package 正确**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
echo "根目录残留 .go:"; ls *.go 2>/dev/null || echo "(无，正确)"
echo "各子包 package 声明:"
for d in models transport stats query filter history; do
  printf "  internal/%s: " "$d"; head -1 "internal/$d/$(ls internal/$d/*.go | grep -v _test | head -1 | xargs basename)"
done
```

Expected:
  - Exit code: 0
  - 根目录 `ls *.go` 输出 `(无，正确)`
  - 六个子包 package 声明分别匹配其目录名

- [x] **Step 8: 提交（此时编译会失败，属预期——下一 Task 修复 import）**

```bash
git add -A && git commit -m "refactor: move root .go files into internal/{models,transport,stats,query,filter,history} subpackages"
```

Expected:
  - Exit code: 0
  - commit 成功（代码处于编译失败状态，Task 3 修复）

---

### Task 3: 修复子包 import 与跨包符号引用，创建根 reexport.go 保持 CLI API 不变

**Depends on:** Task 2
**Files:**
- Modify: 各子包 `.go` 文件的 import 块（添加对 `models`/`transport` 等的 import）
- Modify: `internal/transport/client.go`（跨子包调用的未导出符号提升为导出）
- Create: `internal/models/doc.go`、`internal/transport/doc.go` 等（可选包文档）
- Create: 根 `doc.go`（`package apnic` 包文档）
- Create: 根 `reexport.go`（type alias / 函数 re-export，覆盖 CLI 用到的全部符号）

- [x] **Step 1: 在每个子包 .go 文件顶部补充子包 import — 替换跨文件引用为带包前缀引用**

依据 Task 1 生成的 `.baseline-unexported-symbols.txt`，对所有「定义文件与调用文件现已落入不同子包」的未导出符号，执行：

1. 在定义文件中把符号首字母改大写（导出），如：
   - `transport/doHTTPRequest` → `DoHTTPRequest`
   - `transport/fetchTextStr` → `FetchTextStr`（但 `Client` 方法 `fetchTextStr` 改名 `FetchTextStr`）
   - `transport/fetchReader` → `FetchReader`
   - `stats/parseDelegatedFullFromString` → `ParseDelegatedFullFromString`
   - `stats/parseDelegatedFull` → `ParseDelegatedFull`
   - `stats/parseExtendedFullFromString` → `ParseExtendedFullFromString`
   - `stats/parseAssignedFull` → `ParseAssignedFull`
   - `stats/parseLegacyFull` → `ParseLegacyFull`
   - `transport/cacheKeyIRR` → `CacheKeyIRR`
   - `transport/defaultLookupAddr` → 保持未导出（仅 dns.go 内部与 client.go 同包，移入 transport 后仍同包）
   - `utils.go` 中的 `buildStatsURL`/`buildTransfersAllURL`/`buildTelemetryURL`/`buildIRRDBURL`/`buildThymeURL`/`buildRExURL` 等 URL 构造函数 → 首字母大写导出（`BuildStatsURL` 等），供 stats/query 子包调用

2. 在调用文件中：
   - 添加 import，如 `import "github.com/hitechcloud-vietnam/apnic-skills/internal/transport"`
   - 把 `doHTTPRequest(` 改为 `transport.DoHTTPRequest(`（若调用方是 Client 方法内，注意 receiver）
   - 把 `parseDelegatedFullFromString(` 改为 `stats.ParseDelegatedFullFromString(`

```go
// 示例：internal/query/rdap.go 顶部 import 块改造
// 文件: internal/query/rdap.go（替换现有 import 块）
package query

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// 调用处改造（rdap.go:186 原为 c.doHTTPRequest(...)）：
//   resp, err := c.DoHTTPRequest(ctx, "GET", url, "application/rdap+json, application/json")
// 注意：DoHTTPRequest 是 Client 方法，receiver c 仍是 *transport.Client？——
// 见 Step 2 决策：Client 定义留在 transport 子包，各 query/stats 方法改为
// 接收 *transport.Client 的独立函数，或 Client 方法定义回根包。
```

**关键架构决策（Step 1 内阐明）：** `Client` 类型定义在 `internal/transport/client.go`。所有 `func (c *Client) FetchXxx` 方法原本分散在 stats/query/filter/history 子包的文件里——Go 不允许跨包给类型加方法。因此采用「**方法体改为独立函数，接收 `*transport.Client`**」策略：

```go
// 改造前（根包内方法）：
// func (c *Client) FetchDelegatedEntries(ctx context.Context) ([]DelegatedEntry, error) { ... }

// 改造后（stats 子包独立函数，接收 transport.Client）：
// 文件: internal/stats/fetcher.go
package stats

import (
	"context"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// FetchDelegatedEntries fetches the latest standard delegated stats from APNIC.
func FetchDelegatedEntries(ctx context.Context, c *transport.Client) ([]models.DelegatedEntry, error) {
	// 方法体不变，原 c.fetchText(...) 改为 c.FetchText(...)
	// 原 parseDelegatedFull(...) 改为 ParseDelegatedFull(...)（同子包，仍小写也可，但若被 history 调用则需导出）
}
```

- [x] **Step 2: 改造 transport.Client 与所有 Option — 保留 Client 类型与 NewClient 在 transport 包，方法转为独立函数或保留必要方法**

```go
// 文件: internal/transport/client.go（替换 Client struct 与 Option 定义区块）
package transport

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
)

// dialFunc is the signature for dialing a network connection.
type dialFunc func(ctx context.Context, network, address string) (net.Conn, error)

// Client is the APNIC SDK client that provides access to all APNIC services.
type Client struct {
	httpClient    *http.Client
	cache         *cache
	rateLimiter   *rateLimiter
	downloadCfg   downloadConfig
	// ... 其余字段保持不变（stealth/rdap/whois/stats base URL 等）
}

// Option configures a Client.
type Option func(*Client)

// NewClient creates a new Client with the given options.
func NewClient(opts ...Option) *Client { /* 方法体不变 */ }

// 所有 WithXxx 函数保持不变（WithHTTPClient/WithCacheTTL/WithChunkSize 等），
// 它们都在 transport 包内，操作 *Client 字段。
```

**说明：** `cache`/`rateLimiter`/`downloadConfig` 类型同在 transport 包，未导出符号 `newCache`/`newRateLimiter` 仍同包可访问，无需改动。`Client` 的方法（`fetchText`/`fetchTextStr`/`fetchReader`/`downloadChunked`/`doHTTPRequest`/`jitter`/`waitRateLimit`/`applyBrowserHeaders`/`ReverseDNS`/`VerifyMD5` 等）保持在 transport 包作为 `*Client` 方法（因为它们操作 transport 内部字段），仅把被外部子包调用的方法首字母大写：`FetchText`/`FetchTextStr`/`FetchReader`/`DoHTTPRequest`。

- [x] **Step 3: 改造 stats 子包 — 五个 Fetcher 文件的方法转函数**

```go
// 文件: internal/stats/fetcher.go（替换整个文件）
package stats

import (
	"context"
	"io"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// FetchDelegatedEntries fetches the latest standard delegated stats from APNIC.
func FetchDelegatedEntries(ctx context.Context, c *transport.Client) ([]models.DelegatedEntry, error) {
	result, err := FetchDelegatedResult(ctx, c, "")
	if err != nil {
		return nil, err
	}
	return result.Entries, nil
}

// FetchDelegatedResult fetches delegated stats for a specific date.
// date must be in YYYYMMDD format; empty string fetches "latest".
func FetchDelegatedResult(ctx context.Context, c *transport.Client, date string) (*models.DelegatedResult, error) {
	url := transport.BuildStatsURL(c.StatsBaseURL(), "delegated", date)
	body, err := c.FetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	return ParseDelegatedFullFromString(body)
}

// ParseDelegatedFullFromString parses a delegated stats string.
// 实现方式：把原根包 parseDelegatedFullFromString 的函数体逐字搬入此处
// （仅函数名首字母改大写，逻辑零改动），内部对 DelegatedEntry 的构造
// 改为引用 models.DelegatedEntry。
func ParseDelegatedFullFromString(data string) (*models.DelegatedResult, error) {
	// ↓↓↓ 以下为从原 fetcher.go:parseDelegatedFullFromString 逐字搬入的函数体 ↓↓↓
	// 例如：
	//   lines := strings.Split(data, "\n")
	//   result := &models.DelegatedResult{Header: &models.StatsFileHeader{}}
	//   for _, line := range lines { ... 原解析逻辑保持不变 ... }
	// ↑↑↑ 搬运完毕，勿改动逻辑，仅类型前缀补 models. ↑↑↑
}

// ParseDelegatedFull parses a delegated stats reader.
// 实现方式：把原根包 parseDelegatedFull 的函数体逐字搬入此处。
func ParseDelegatedFull(r io.Reader) (*models.DelegatedResult, error) {
	// ↓↓↓ 从原 fetcher.go:parseDelegatedFull 逐字搬入的函数体 ↓↓↓
	// 例如：data, err := io.ReadAll(r); if err != nil { return nil, err }
	//       return ParseDelegatedFullFromString(string(data))
	// ↑↑↑ 搬运完毕 ↑↑↑
}
```

（`extended.go`/`assigned.go`/`ipv6_assigned.go`/`legacy.go` 同模式改造，函数签名统一为 `func FetchXxx(ctx, c *transport.Client, ...) (*models.XxxResult, error)`。）

- [x] **Step 4: 改造 query 子包 — 九个查询文件的方法转函数**

```go
// 文件: internal/query/rdap.go（替换 RDAPLookupIP 及 doRDAPRequestAt 等方法）
package query

import (
	"context"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// RDAPLookupIP queries the RDAP service for an IP address.
func RDAPLookupIP(ctx context.Context, c *transport.Client, ip string) (*models.RDAPNetwork, error) {
	return RDAPLookupIPAt(ctx, c, ip, time.Time{})
}

// RDAPLookupIPAt queries RDAP for an IP at a point in time.
func RDAPLookupIPAt(ctx context.Context, c *transport.Client, ip string, date time.Time) (*models.RDAPNetwork, error) {
	var result models.RDAPNetwork
	if err := doRDAPRequestAt(ctx, c, "/ip/"+ip, &result, date); err != nil {
		return nil, err
	}
	return &result, nil
}

// doRDAPRequestAt 保持未导出（仅 query 子包内调用）。
func doRDAPRequestAt(ctx context.Context, c *transport.Client, path string, result interface{}, date time.Time) error {
	url := /* 用 transport.BuildXxxURL 构造 */
	resp, err := c.DoHTTPRequest(ctx, "GET", url, "application/rdap+json, application/json")
	/* 方法体不变 */
}
```

（`rex.go`/`whois.go`/`irr.go`/`rrdp.go`/`bgp.go`/`telemetry.go`/`transfers.go`/`changes.go` 同模式：`func XxxLookup(ctx, c *transport.Client, ...) (*models.Xxx, error)`，`whois.go` 的 `ParseWhoisResponse` 因不依赖 Client，保持包级函数。）

- [x] **Step 5: 改造 filter 与 history 子包**

```go
// 文件: internal/filter/filter.go（替换 EntryFilter/ExtendedEntryFilter 及方法）
package filter

import (
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
)

// EntryFilter provides a chainable filter API for DelegatedEntry slices.
type EntryFilter struct {
	entries []models.DelegatedEntry
}

// NewFilter creates a new EntryFilter with the given entries.
func NewFilter(entries []models.DelegatedEntry) *EntryFilter {
	return &EntryFilter{entries: entries}
}

// ByCountry filters entries by ISO 3166 country code.
func (f *EntryFilter) ByCountry(country string) *EntryFilter { /* 方法体不变 */ }

// 其余 ByType/ByStatus/ByDateRange/ByRegistry/Result/Count 方法签名不变（仅 import models）
```

```go
// 文件: internal/history/history.go（替换历史查询方法为函数）
package history

import (
	"context"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/stats"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// FetchHistoricalDelegated fetches delegated stats for a specific date.
func FetchHistoricalDelegated(ctx context.Context, c *transport.Client, date string) (*models.DelegatedResult, error) {
	return stats.FetchDelegatedResult(ctx, c, date)
}

// ListAvailableYears returns the list of years for which historical stats exist.
func ListAvailableYears() []int { /* 方法体不变 */ }
```

- [x] **Step 6: 改造 transport 子包内的 Client 字段访问器 — 为子包需要的字段提供导出方法**

```go
// 文件: internal/transport/client.go（追加字段访问方法）
// StatsBaseURL returns the configured stats base URL (for stats/query subpackages).
func (c *Client) StatsBaseURL() string { return c.statsBaseURL }

// FTPBaseURL returns the configured FTP base URL.
func (c *Client) FTPBaseURL() string { return c.ftpBaseURL }

// ThymeBaseURL returns the configured thyme base URL.
func (c *Client) ThymeBaseURL() string { return c.thymeBaseURL }

// RRDPBaseURL returns the configured RRDP base URL.
func (c *Client) RRDPBaseURL() string { return c.rrdpBaseURL }

// RDAPBaseURL returns the configured RDAP base URL.
func (c *Client) RDAPBaseURL() string { return c.rdapBaseURL }

// WhoisServer returns the configured whois server.
func (c *Client) WhoisServer() string { return c.whoisServer }

// RExBaseURL returns the configured REx cross-RIR base URL.
func (c *Client) RExBaseURL() string { return c.rexBaseURL }
```

（字段名以 client.go 实际为准；Step 执行时按 Task 1 调研记录的实际字段名补全。）

- [x] **Step 7: 创建根 doc.go 与 reexport.go — 保持 import 路径与 apnic.Xxx API 完全不变**

```go
// 文件: doc.go
// Package apnic provides a Go SDK for APNIC public data services.
//
// This is the root package of github.com/hitechcloud-vietnam/apnic-skills. The actual
// implementation lives in subpackages under internal/; this package re-exports
// the public API so that importers can keep using "github.com/hitechcloud-vietnam/apnic-skills"
// as the import path and apnic.Client / apnic.NewClient / etc. as before.
package apnic
```

```go
// 文件: reexport.go
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

// === 类型 re-export（type alias，零运行时开销） ===
// 覆盖 cmd/apnic 实际引用的全部类型（见 Task 1 Step 3 清单）。

type (
	// transport
	Client = transport.Client
	Option = transport.Option

	// models（CLI 用到的 + 公开 API 主类型）
	RDAPNetwork           = models.RDAPNetwork
	DelegatedEntry        = models.DelegatedEntry
	DelegatedExtendedEntry = models.DelegatedExtendedEntry

	// filter
	EntryFilter         = filter.EntryFilter
	ExtendedEntryFilter = filter.ExtendedEntryFilter
)

// === const re-export（Go 不支持 const alias，逐个重新声明） ===

const (
	DefaultStatsBaseURL  = transport.DefaultStatsBaseURL
	DefaultRDAPBaseURL   = transport.DefaultRDAPBaseURL
	DefaultRRDPBaseURL   = transport.DefaultRRDPBaseURL
	DefaultThymeBaseURL  = transport.DefaultThymeBaseURL
	DefaultFTPBaseURL    = transport.DefaultFTPBaseURL
	DefaultRExBaseURL    = transport.DefaultRExBaseURL
)

// === 函数 re-export（薄包装；因 Option = transport.Option 可直接透传） ===

// NewClient creates a new APNIC client.
func NewClient(opts ...Option) *Client { return transport.NewClient(opts...) }

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

// Option factories — 全部透传到 transport 子包。
func WithHTTPClient(hc *http.Client) Option          { return transport.WithHTTPClient(hc) }
func WithCacheTTL(ttl time.Duration) Option           { return transport.WithCacheTTL(ttl) }
func WithUserAgent(ua string) Option                  { return transport.WithUserAgent(ua) }
func WithRDAPBaseURL(url string) Option               { return transport.WithRDAPBaseURL(url) }
func WithWhoisServer(server string) Option            { return transport.WithWhoisServer(server) }
func WithStatsBaseURL(url string) Option              { return transport.WithStatsBaseURL(url) }
func WithRDAPDate(t time.Time) Option                 { return transport.WithRDAPDate(t) }
func WithStealth(enable bool) Option                 { return transport.WithStealth(enable) }
func WithBrowserUserAgent(ua string) Option          { return transport.WithBrowserUserAgent(ua) }
func WithJitter(min, max time.Duration) Option       { return transport.WithJitter(min, max) }
func WithRateLimit(perSecond float64) Option          { return transport.WithRateLimit(perSecond) }
func WithRRDPBaseURL(url string) Option               { return transport.WithRRDPBaseURL(url) }
func WithThymeBaseURL(url string) Option              { return transport.WithThymeBaseURL(url) }
func WithFTPBaseURL(url string) Option                { return transport.WithFTPBaseURL(url) }
func WithRExBaseURL(url string) Option                { return transport.WithRExBaseURL(url) }
func WithMaxConcurrentDownloads(n int) Option        { return transport.WithMaxConcurrentDownloads(n) }
func WithChunkSize(bytes int64) Option               { return transport.WithChunkSize(bytes) }
func WithDownloadTimeout(d time.Duration) Option     { return transport.WithDownloadTimeout(d) }
func WithWhoisTimeout(timeout time.Duration) Option   { return transport.WithWhoisTimeout(timeout) }
```

**说明：** 因 `Option = transport.Option`（type alias），所有 `WithXxx` 直接 `return transport.WithXxx(opts...)` 即可。CLI 的 `apnic.WithChunkSize(1024)` 调用链不变。上述清单逐项对应 Task 1 Step 3 生成的 `.baseline-cli-symbols.txt`——执行时若该清单含本文件未列出的符号，按同模式补齐薄包装。

- [x] **Step 8: 验证全量编译 — go build 所有包 + go vet**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
go build ./... && go vet ./...
```

Expected:
  - Exit code: 0
  - Output does NOT contain: `undefined:` 或 `cannot use` 或 `not enough arguments`

- [x] **Step 9: 验证 cmd/apnic 编译 — CLI import 路径不变**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
go build ./cmd/apnic
```

Expected:
  - Exit code: 0
  - 无输出（编译成功）
  - `cmd/apnic/*.go` 文件未做任何 import 修改（仍为 `apnic "github.com/hitechcloud-vietnam/apnic-skills"`）

- [x] **Step 10: 提交**

```bash
git add -A && git commit -m "refactor: split into internal subpackages + root reexport preserving apnic.* API"
```

---

### Task 4: 修复测试文件的子包归属与未导出符号访问

**Depends on:** Task 3
**Files:**
- Modify: 各子包 `_test.go`（package 声明随被测文件已改，但测试体内对未导出符号/跨包符号的引用需修正）
- Modify: `internal/transport/test_helpers_test.go`（原 `test_helpers_test.go`，被多子包测试依赖 → 决策：留在 transport，被依赖处改为调用导出的测试 helper 或各自复制所需 fixture）

- [x] **Step 1: 修复 transport 子包测试 — 多数测试访问 Client 未导出字段/方法，已随文件移入 transport，同包可见**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
# test_helpers_test.go 已在 transport 子包，其内 helper 若被 stats/query 测试引用，
# 需识别并迁移。先 grep 找出跨子包引用：
grep -rn "test_helpers\|newTestClient\|setupTestServer" internal/stats/*_test.go internal/query/*_test.go internal/filter/*_test.go internal/history/*_test.go 2>/dev/null
```

依据输出，对每个被跨子包引用的测试 helper：
- 若 helper 仅依赖导出 API（如 `NewClient`、`httptest.NewServer`）→ 复制到调用方子包的 `helpers_test.go`，或抽到 `internal/testutil/`（非 internal 的 testutil 也可）。
- 若依赖未导出符号 → 该 helper 必须留在 transport 子包，调用方改用导出 API 重写。

```go
// 示例：若需创建 internal/testutil/testutil.go（package testutil）
package testutil

import (
	"net/http/httptest"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// NewTestClient builds a Client pointing at a test server.
func NewTestClient(ts *httptest.Server) *transport.Client {
	return transport.NewClient(
		transport.WithHTTPClient(ts.Client()),
		transport.WithStatsBaseURL(ts.URL),
		transport.WithFTPBaseURL(ts.URL),
		transport.WithThymeBaseURL(ts.URL),
		transport.WithRDAPBaseURL(ts.URL),
	)
}
```

- [x] **Step 2: 修复 stats/query/filter/history 子包测试 — 把对原 Client 方法的调用改为函数调用**

```go
// 改造前（stats_test.go 内）：
// entries, err := c.FetchDelegatedEntries(ctx)

// 改造后：
// entries, err := stats.FetchDelegatedEntries(ctx, c)
```

对每个子包 `_test.go`，逐文件执行：把 `c.FetchXxx(ctx, ...)` 改为 `subpackage.FetchXxx(ctx, c, ...)`，把 `apnic.NewClient(...)` 改为 `transport.NewClient(...)`（测试在子包内，直接用 transport）或通过 testutil。

- [x] **Step 3: 修复 cli 子包测试 — cmd/apnic 内测试已用 apnic. 前缀，无需改（验证）**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
go test ./cmd/apnic/... -count=1 2>&1 | tail -5
```

Expected:
  - Exit code: 0
  - Output contains: `ok` 行

- [x] **Step 4: 验证全量测试 — 与 Task 1 基线对比**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
go test ./... -count=1 2>&1 | tee /tmp/apnic-post.txt
diff <(grep -E "^(ok|FAIL)" /tmp/apnic-baseline.txt | sort) <(grep -E "^(ok|FAIL)" /tmp/apnic-post.txt | sort) && echo "BASELINE MATCH"
```

Expected:
  - Exit code: 0
  - Output contains: `BASELINE MATCH`（测试结果与重组前一致）
  - 若子包拆分导致 ok 行数量增加（包变多），逐包确认全 ok 即可，diff 仅有包路径变化无 FAIL

- [x] **Step 5: 提交**

```bash
git add -A && git commit -m "test: fix subpackage test wiring after restructure, baseline green"
```

---

### Task 5: 同步 website/docs 文件名引用与 README

**Depends on:** Task 4
**Files:**
- Modify: `website/docs/architecture/*.md`、`website/docs/cli/*.md`、`website/docs/types/*.md`（约 70 处 `xxx.go` 引用补全为 `internal/<sub>/xxx.go`）
- Modify: `README.md`、`README.zh-CN.md`（若有根 .go 路径引用）
- Modify: `docs/superpowers/plans/*.md`（历史 plan，可选——仅修正指向当前已移动文件的引用）

- [x] **Step 1: 定位所有指向已移动 .go 文件的文档引用**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
# 文件名 → 新路径映射表
cat > /tmp/filemap.txt <<'EOF'
models.go:internal/models/models.go
client.go:internal/transport/client.go
downloader.go:internal/transport/downloader.go
stealth.go:internal/transport/stealth.go
cache.go:internal/transport/cache.go
dns.go:internal/transport/dns.go
errors.go:internal/transport/errors.go
utils.go:internal/transport/utils.go
verify.go:internal/transport/verify.go
fetcher.go:internal/stats/fetcher.go
extended.go:internal/stats/extended.go
assigned.go:internal/stats/assigned.go
ipv6_assigned.go:internal/stats/ipv6_assigned.go
legacy.go:internal/stats/legacy.go
rdap.go:internal/query/rdap.go
rex.go:internal/query/rex.go
whois.go:internal/query/whois.go
irr.go:internal/query/irr.go
rrdp.go:internal/query/rrdp.go
bgp.go:internal/query/bgp.go
telemetry.go:internal/query/telemetry.go
transfers.go:internal/query/transfers.go
changes.go:internal/query/changes.go
cidr.go:internal/filter/cidr.go
filter.go:internal/filter/filter.go
history.go:internal/history/history.go
EOF
# 列出待修改文档
grep -rlE "\b(models|client|downloader|stealth|cache|dns|errors|utils|verify|fetcher|extended|assigned|ipv6_assigned|legacy|rdap|rex|whois|irr|rrdp|bgp|telemetry|transfers|changes|cidr|filter|history)\.go\b" website/docs README.md README.zh-CN.md 2>/dev/null
```

- [x] **Step 2: 用 sed 批量替换文档中的文件名引用为完整路径 — 逐文件执行**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
while IFS=: read -r name path; do
  # 仅替换裸文件名引用（前非 internal/ 的），避免重复替换
  grep -rlE "(^|[^/])\b${name}\b" website/docs README.md README.zh-CN.md 2>/dev/null | \
    xargs -r sed -i -E "s@(^|[^/])\b${name}\b@\1${path}@g"
done < /tmp/filemap.txt
```

Expected:
  - Exit code: 0
  - `grep -rn "stealth.go" website/docs/architecture/anti-scraping.md` 输出含 `internal/transport/stealth.go`

- [x] **Step 3: 验证文档无残留裸文件名引用**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
grep -rnE "\b(models|client|downloader|stealth|cache|dns|errors|utils|verify|fetcher|extended|assigned|ipv6_assigned|legacy|rdap|rex|whois|irr|rrdp|bgp|telemetry|transfers|changes|cidr|filter|history)\.go\b" website/docs README.md README.zh-CN.md 2>/dev/null | grep -v "internal/" || echo "ALL REPLACED"
```

Expected:
  - Exit code: 0
  - Output contains: `ALL REPLACED`

- [x] **Step 4: 提交**

```bash
git add -A && git commit -m "docs: update website/readme file references to new internal/ subpackage paths"
```

---

### Task 6: 全量验证、覆盖基线对比与收尾

**Depends on:** Task 5
**Files:**
- Run: 全量验证命令
- Modify: `docs/superpowers/plans/2026-07-11-restructure-go-files.md`（本 plan，勾选完成）

- [x] **Step 1: 全量编译 + vet + 测试**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
go build ./... && go vet ./... && go test ./... -count=1
```

Expected:
  - Exit code: 0
  - 每个包输出 `ok`
  - Output does NOT contain: `FAIL` 或 `panic`

- [x] **Step 2: 验证 CLI 二进制可构建且运行**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
go build -o /tmp/apnic-restructured ./cmd/apnic && /tmp/apnic-restructured --help | head -5
```

Expected:
  - Exit code: 0
  - Output contains: `APNIC data & query toolkit`

- [x] **Step 3: 覆盖率基线对比 — 确认重组未改变覆盖率**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
go test ./... -coverprofile=/tmp/cover-new.out -count=1 2>/dev/null
go tool cover -func=/tmp/cover-new.out | tail -1
# 与原 coverage.out（gitignore 内的旧产物，若存在）对比总覆盖率百分比
```

Expected:
  - Exit code: 0
  - 总覆盖率百分比与重组前持平或更高（不应下降）

- [x] **Step 4: 验证根目录整洁 — 仅剩 doc.go / reexport.go + 非代码资产**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
ls *.go
```

Expected:
  - Exit code: 0
  - 输出仅 `doc.go` 与 `reexport.go`（可选 `reexport_stats.go` 等按拆分）

- [x] **Step 5: 清理临时基线文件 + 提交收尾**

```bash
cd /home/cc11001100/github/hitechcloud-vietnam/apnic-skills
git rm -f docs/superpowers/plans/.baseline-unexported-symbols.txt docs/superpowers/plans/.baseline-cli-symbols.txt 2>/dev/null || true
git add -A && git commit -m "chore: finalize root .go restructure, remove temp baselines"
```

- [x] **Step 6: 提交**

（已包含在 Step 5）
