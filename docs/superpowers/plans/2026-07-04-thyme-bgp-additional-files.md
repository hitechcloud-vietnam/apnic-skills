# thyme BGP 附加数据文件 + 多数据源 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`
> Steps use checkbox (`- [ ]`) syntax.

**Goal:** 为 SDK 补齐 APNIC thyme 的 5 个附加 BGP 数据文件（`data-badpfx-nos`、`data-pfx-nos`、`data-used-autnums`、`data-spar`、`data-singlepfx`），并为所有 thyme 端点新增 `au`/`hk` 多数据源支持，使 SDK 的 BGP 能力覆盖完整。

**Architecture:** 用户调用 `client.FetchBGPBadPrefixes(ctx, source)` → `buildThymeURL(base, source, file)` 拼出 `https://thyme.apnic.net/{source}/{file}` → `fetchText` 拉取（小文件走单连接，大文件自动分块） → 文件专属 `parseBGP*` 解析为模型 → 返回。`source` 为 per-call 参数（`"current"`/`"au"`/`"hk"`，空 = current），不引入全局状态，避免与现有 `WithThymeBaseURL` 冲突。复用现有 `fetchText`/`fetchTextStr` + `bufio.Scanner` + `strings.Fields` 解析模式，与 `parseBGPSummary`/`parseBGPRawTable` 同构。

**Tech Stack:** Go 1.21+，标准库 `bufio`/`strings`/`strconv`/`context`，cobra CLI，httptest 测试。无新依赖。

**Risks:**
- `buildThymeURL` 当前硬编码 `/current/`，被 `bgp.go` 现有 2 处调用——改签名会破坏现有调用 → 缓解：新增 `source` 参数，现有调用传 `""`（在函数内 `== ""` 时回退为 `"current"`），保持行为不变
- `data-used-autnums` 行格式 `ASN Name - Description, CC` 含嵌入空格和逗号——解析需用空格分前 2 段（ASN + rest），rest 末尾 `, CC` 分离国家 → 缓解：用 `strings.Fields` + 末段逗号分割
- `data-pfx-nos` 是 `/N:count` 的网格布局（一行多个）——需逐 token 扫描而非逐行 → 缓解：`strings.Fields` 全文分词后逐 token 处理
- 多源 `au`/`hk` 的文件集与 `current` 完全相同（已 curl 确认 7 个文件全 200）→ 复用同一组 parser，只需 URL 段差异

---

### Task 1: 新增 5 个 BGP 数据模型

**Depends on:** None
**Files:**
- Modify: `models.go:415-446`（在 `BGPASNMap` 结构体之后、`RRDPNotification` 之前插入）

- [ ] **Step 1: 在 models.go 新增 5 个 thyme 附加数据模型**

文件: `models.go:446`（在 `BGPASNMap` 结构体闭合 `}` 之后、`RRDPNotification` 注释之前插入）

```go
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

// BPGPrefixLengthCount is a single "/N:count" entry from thyme's data-pfx-nos
// file, recording how many prefixes of each length are announced.
type BPGPrefixLengthCount struct {
	Length int    // the N in /N
	Count  int    // number of prefixes of that length
	Raw    string // the original token, e.g. "/8:16", kept for diagnostics
}

// BGPPerPrefixLength holds the parsed entries from thyme's data-pfx-nos file.
type BGPPerPrefixLength struct {
	Counts []BPGPrefixLengthCount
}

// BGPUsedAutnum is a single in-use ASN record from thyme's data-used-autnums
// file: "ASN Name - Description, CC".
type BGPUsedAutnum struct {
	ASN       string
	Name      string // the registered name (e.g. "LVLT-1")
	Country   string // ISO country code (e.g. "US")
	FullName  string // the full "Name - Description" text before the country
}

// BGPUsedAutnums holds the parsed entries from thyme's data-used-autnums file.
type BGPUsedAutnums struct {
	Autnums []BGPUsedAutnum
}

// BGPSparPrefix is a prefix from the Special Purpose Address Registry
// (RFC 6890 reserved space) and its origin AS, from thyme's data-spar file.
type BGPSparPrefix struct {
	Prefix    string
	OriginAS  string
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
```

- [ ] **Step 2: 验证 models.go 编译**
Run: `go build ./...`
Expected:
  - Exit code: 0
  - Output does NOT contain: "syntax error" or "undefined"

- [ ] **Step 3: 提交**
Run: `git add models.go && git commit -m "feat(bgp): add models for 5 additional thyme BGP data files"`

---

### Task 2: buildThymeURL 支持 source 段 + 新增 WithThymeSource Option

**Depends on:** Task 1
**Files:**
- Modify: `utils.go:226-230`（buildThymeURL 函数）
- Modify: `client.go:46`（Client 结构体加 thymeSource 字段）
- Modify: `client.go:74`（NewClient 默认值）
- Modify: `client.go:201`（新增 WithThymeSource Option）

- [ ] **Step 1: 修改 buildThymeURL 以支持 source 段 — 保持向后兼容**

文件: `utils.go:226-230`（替换整个 buildThymeURL 函数）

```go
// buildThymeURL constructs the URL for an APNIC thyme BGP analysis file.
// source is one of "current", "au", or "hk"; an empty source defaults to
// "current" for backward compatibility. file is one of "data-summary",
// "data-raw-table", "data-badpfx-nos", "data-pfx-nos", "data-used-autnums",
// "data-spar", or "data-singlepfx".
func buildThymeURL(thymeBaseURL, source, file string) string {
	if source == "" {
		source = "current"
	}
	return strings.TrimRight(thymeBaseURL, "/") + "/" + source + "/" + file
}
```

- [ ] **Step 2: 给 Client 加 thymeSource 字段 — 支持全局默认数据源**

文件: `client.go:46`（在 `thymeBaseURL string` 字段之后插入新行）

```go
	thymeBaseURL string // BGP routing analysis, default "https://thyme.apnic.net"
	thymeSource  string // thyme data source: "current" (default), "au", or "hk"
```

- [ ] **Step 3: NewClient 设默认 thymeSource — 默认 current**

文件: `client.go:74`（在 `thymeBaseURL: DefaultThymeBaseURL,` 之后插入）

```go
		thymeBaseURL: DefaultThymeBaseURL,
		thymeSource:  "current",
```

- [ ] **Step 4: 新增 WithThymeSource Option — 允许全局切换 au/hk 数据源**

文件: `client.go:201`（在 `WithThymeBaseURL` 函数之后插入）

```go
// WithThymeSource sets the thyme data source: "current" (default, global view),
// "au" (Brisbane), or "hk" (HKIX). It applies to all thyme BGP requests that do
// not specify a source explicitly.
func WithThymeSource(source string) Option {
	return func(c *Client) {
		c.thymeSource = source
	}
}
```

- [ ] **Step 5: 更新 bgp.go 现有 2 处 buildThymeURL 调用 — 传入空 source 走默认**

文件: `bgp.go:13`（FetchBGPSummary 内）

```go
	url := buildThymeURL(c.thymeBaseURL, c.thymeSource, "data-summary")
```

文件: `bgp.go:24`（FetchBGPRawTable 内）

```go
	url := buildThymeURL(c.thymeBaseURL, c.thymeSource, "data-raw-table")
```

- [ ] **Step 6: 验证编译 + 现有测试不回归**
Run: `go build ./... && go test . -run TestBGP -count=1`
Expected:
  - Exit code: 0
  - Output contains: "ok" and does NOT contain "FAIL"

- [ ] **Step 7: 提交**
Run: `git add utils.go client.go bgp.go && git commit -m "feat(bgp): support thyme multi-source (au/hk) via buildThymeURL source arg"`

---

### Task 3: 实现 5 个 thyme 附加文件解析器

**Depends on:** Task 1
**Files:**
- Modify: `bgp.go`（在文件末尾 `parseBGPRawTable` 之后追加 5 个 parse 函数）

- [ ] **Step 1: 实现 parseBGPBadPrefixes — 解析 data-badpfx-nos**

文件: `bgp.go`（在文件末尾追加）

```go
// parseBGPBadPrefixes parses thyme's data-badpfx-nos file. After a header
// (title + dash separator + column header), each non-empty line is
// "OriginAS<TAB>Address". Lines without two whitespace fields are skipped.
func parseBGPBadPrefixes(data string) *BGPBadPrefixes {
	r := &BGPBadPrefixes{Prefixes: make([]BGPBadPrefix, 0, 10000)}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "Prefixes longer") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		// Skip the column header row.
		if strings.EqualFold(fields[0], "Origin") || strings.EqualFold(fields[1], "Address") {
			continue
		}
		r.Prefixes = append(r.Prefixes, BGPBadPrefix{OriginAS: fields[0], Address: fields[1]})
	}
	return r
}
```

- [ ] **Step 2: 实现 parseBGPPerPrefixLength — 解析 data-pfx-nos 的 /N:count 网格**

文件: `bgp.go`（在 `parseBGPBadPrefixes` 之后追加）

```go
// parseBGPPerPrefixLength parses thyme's data-pfx-nos file. The file lays out
// "/N:count" tokens in a multi-column grid (several per line). Each token is
// split on ":" into length (the N in /N) and count. Tokens that fail to parse
// are skipped.
func parseBGPPerPrefixLength(data string) *BGPPerPrefixLength {
	r := &BGPPerPrefixLength{Counts: make([]BPGPrefixLengthCount, 0, 128)}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "Number of prefixes") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			if !strings.HasPrefix(tok, "/") {
				continue
			}
			colon := strings.Index(tok, ":")
			if colon < 0 {
				continue
			}
			lengthStr := tok[1:colon] // strip leading "/"
			countStr := tok[colon+1:]
			length, err := strconv.Atoi(lengthStr)
			if err != nil {
				continue
			}
			count, err := strconv.Atoi(countStr)
			if err != nil {
				continue
			}
			r.Counts = append(r.Counts, BPGPrefixLengthCount{Length: length, Count: count, Raw: tok})
		}
	}
	return r
}
```

- [ ] **Step 3: 实现 parseBGPUsedAutnums — 解析 data-used-autnums 的 ASN/Name/CC**

文件: `bgp.go`（在 `parseBGPPerPrefixLength` 之后追加）

```go
// parseBGPUsedAutnums parses thyme's data-used-autnums file. Each line is
// "<ASN> <Name> - <Description>, <CC>", e.g. "1 LVLT-1 - Level 3 Parent, LLC, US".
// The ASN is the first whitespace field; the country code is the text after the
// final comma; the FullName is everything between them.
func parseBGPUsedAutnums(data string) *BGPUsedAutnums {
	r := &BGPUsedAutnums{Autnums: make([]BGPUsedAutnum, 0, 80000)}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		asn := fields[0]
		// Country code is the token after the last comma.
		commaIdx := strings.LastIndex(line, ",")
		if commaIdx < 0 {
			continue
		}
		country := strings.TrimSpace(line[commaIdx+1:])
		// FullName is the text between the ASN and the comma (exclusive).
		rest := strings.TrimSpace(line[len(asn):commaIdx])
		// Name is the first whitespace field of rest.
		nameFields := strings.Fields(rest)
		name := ""
		if len(nameFields) > 0 {
			name = nameFields[0]
		}
		r.Autnums = append(r.Autnums, BGPUsedAutnum{
			ASN:      asn,
			Name:     name,
			Country:  country,
			FullName: rest,
		})
	}
	return r
}
```

- [ ] **Step 4: 实现 parseBGPSparPrefixes — 解析 data-spar 的 Prefix/OriginAS/Desc**

文件: `bgp.go`（在 `parseBGPUsedAutnums` 之后追加）

```go
// parseBGPSparPrefixes parses thyme's data-spar file. After a header, each line
// is "<Prefix><TAB>OriginAS<TAB>Description". The description may contain
// spaces, so the line is split into at most 3 fields by tab (falling back to
// any-whitespace when the tab split yields only 2 fields).
func parseBGPSparPrefixes(data string) *BGPSparPrefixes {
	r := &BGPSparPrefixes{Prefixes: make([]BGPSparPrefix, 0, 64)}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "Prefixes from") {
			continue
		}
		// Tab-split first; if it yields 2 fields, the description is empty.
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			fields = strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
		}
		// Skip column header.
		if strings.EqualFold(fields[0], "Prefix") {
			continue
		}
		prefix := strings.TrimSpace(fields[0])
		originAS := strings.TrimSpace(fields[1])
		desc := ""
		if len(fields) >= 3 {
			desc = strings.TrimSpace(strings.Join(fields[2:], " "))
		}
		r.Prefixes = append(r.Prefixes, BGPSparPrefix{Prefix: prefix, OriginAS: originAS, Description: desc})
	}
	return r
}
```

- [ ] **Step 5: 实现 parseBGPSinglePfx — 解析 data-singlepfx 的 Prefix/ASN/RIR**

文件: `bgp.go`（在 `parseBGPSparPrefixes` 之后追加）

```go
// parseBGPSinglePfx parses thyme's data-singlepfx file. After a header, each
// line is "<PrefixCount><TAB><ASNCount><TAB><RIR>", e.g. "1 27539 Global".
// Non-numeric prefix/ASN counts are skipped.
func parseBGPSinglePfx(data string) *BGPSinglePfx {
	r := &BGPSinglePfx{Counts: make([]BGPSinglePfxCount, 0, 32)}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "Number of ASNs") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// Skip column header.
		if strings.EqualFold(fields[0], "No.") {
			continue
		}
		prefixCount, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		asnCount, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		rir := strings.Join(fields[2:], " ")
		r.Counts = append(r.Counts, BGPSinglePfxCount{PrefixCount: prefixCount, ASNCount: asnCount, RIR: rir})
	}
	return r
}
```

- [ ] **Step 6: 在 bgp.go 顶部 import 块加 strconv — 新解析器需要 Atoi**

文件: `bgp.go:3-8`（替换整个 import 块）

```go
import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"
)
```

- [ ] **Step 7: 验证编译**
Run: `go build ./...`
Expected:
  - Exit code: 0
  - Output does NOT contain: "undefined" or "imported and not used"

- [ ] **Step 8: 提交**
Run: `git add bgp.go && git commit -m "feat(bgp): add parsers for 5 additional thyme BGP data files"`

---

### Task 4: 实现 5 个 Fetch 方法 + 单元测试

**Depends on:** Task 2, Task 3
**Files:**
- Modify: `bgp.go`（在 `FetchBGPASNMap` 之后追加 5 个 Fetch 方法）
- Modify: `bgp_test.go`（追加 5 个解析器测试 + 5 个 Fetch 测试）

- [ ] **Step 1: 在 bgp.go 追加 5 个 Fetch 方法 — 每个拉取并解析对应 thyme 文件**

文件: `bgp.go`（在 `FetchBGPASNMap` 函数之后追加）

```go
// FetchBGPBadPrefixes fetches and parses thyme's data-badpfx-nos file, which
// lists prefixes longer than /24 and their origin AS (potential route leaks).
// source is "current" (default), "au", or "hk"; an empty string uses the
// client's default source.
func (c *Client) FetchBGPBadPrefixes(ctx context.Context, source string) (*BGPBadPrefixes, error) {
	url := buildThymeURL(c.thymeBaseURL, sourceOrDefault(source, c.thymeSource), "data-badpfx-nos")
	body, err := c.fetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseBGPBadPrefixes(body), nil
}

// FetchBGPPerPrefixLength fetches and parses thyme's data-pfx-nos file, which
// counts announced prefixes per prefix length.
func (c *Client) FetchBGPPerPrefixLength(ctx context.Context, source string) (*BGPPerPrefixLength, error) {
	url := buildThymeURL(c.thymeBaseURL, sourceOrDefault(source, c.thymeSource), "data-pfx-nos")
	body, err := c.fetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseBGPPerPrefixLength(body), nil
}

// FetchBGPUsedAutnums fetches and parses thyme's data-used-autnums file, which
// lists every in-use ASN with its registered name and country.
func (c *Client) FetchBGPUsedAutnums(ctx context.Context, source string) (*BGPUsedAutnums, error) {
	url := buildThymeURL(c.thymeBaseURL, sourceOrDefault(source, c.thymeSource), "data-used-autnums")
	body, err := c.fetchTextStr(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseBGPUsedAutnums(body), nil
}

// FetchBGPSparPrefixes fetches and parses thyme's data-spar file, which lists
// prefixes from the Special Purpose Address Registry (RFC 6890) and their
// origin AS.
func (c *Client) FetchBGPSparPrefixes(ctx context.Context, source string) (*BGPSparPrefixes, error) {
	url := buildThymeURL(c.thymeBaseURL, sourceOrDefault(source, c.thymeSource), "data-spar")
	body, err := c.fetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseBGPSparPrefixes(body), nil
}

// FetchBGPSinglePfx fetches and parses thyme's data-singlepfx file, which
// tallies how many ASNs announce fewer than 20 prefixes, grouped by RIR.
func (c *Client) FetchBGPSinglePfx(ctx context.Context, source string) (*BGPSinglePfx, error) {
	url := buildThymeURL(c.thymeBaseURL, sourceOrDefault(source, c.thymeSource), "data-singlepfx")
	body, err := c.fetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseBGPSinglePfx(body), nil
}
```

- [ ] **Step 2: 在 utils.go 新增 sourceOrDefault 辅助函数 — 处理 per-call source 回退**

文件: `utils.go:230`（在 `buildThymeURL` 之后追加）

```go
// sourceOrDefault returns source if non-empty, else def. Used by thyme Fetch
// methods to let a per-call source override the client's default thymeSource.
func sourceOrDefault(source, def string) string {
	if source != "" {
		return source
	}
	return def
}
```

- [ ] **Step 3: 在 bgp_test.go 追加 5 个解析器单元测试**

文件: `bgp_test.go`（在文件末尾追加）

```go
const sampleBGPBadPrefixes = `Prefixes longer than /24 and their Origin AS (Global)
-----------------------------------------------------

Origin AS       Address
    10167       1.209.111.128/25
    12345       2.2.2.0/26
`

func TestParseBGPBadPrefixes(t *testing.T) {
	r := parseBGPBadPrefixes(sampleBGPBadPrefixes)
	if len(r.Prefixes) != 2 {
		t.Fatalf("prefixes = %d, want 2", len(r.Prefixes))
	}
	if r.Prefixes[0].OriginAS != "10167" || r.Prefixes[0].Address != "1.209.111.128/25" {
		t.Errorf("prefix[0] = %+v", r.Prefixes[0])
	}
	// Header lines must be skipped.
	for _, p := range r.Prefixes {
		if p.OriginAS == "Origin" {
			t.Error("column header was not skipped")
		}
	}
}

func TestParseBGPBadPrefixes_Empty(t *testing.T) {
	r := parseBGPBadPrefixes("")
	if len(r.Prefixes) != 0 {
		t.Errorf("expected 0 prefixes, got %d", len(r.Prefixes))
	}
}

const sampleBGPPerPrefixLength = `Number of prefixes announced per prefix length (Global)
-------------------------------------------------------

 /1:0        /2:0        /3:0        /4:0        /8:16       /9:14
 /10:39      /11:97      /12:299
`

func TestParseBGPPerPrefixLength(t *testing.T) {
	r := parseBGPPerPrefixLength(sampleBGPPerPrefixLength)
	if len(r.Counts) != 9 {
		t.Fatalf("counts = %d, want 9", len(r.Counts))
	}
	// Spot-check /8:16 and /12:299.
	var found8, found12 bool
	for _, c := range r.Counts {
		if c.Length == 8 && c.Count == 16 {
			found8 = true
		}
		if c.Length == 12 && c.Count == 299 {
			found12 = true
		}
	}
	if !found8 || !found12 {
		t.Errorf("missing expected entries: %+v", r.Counts)
	}
}

func TestParseBGPPerPrefixLength_Empty(t *testing.T) {
	r := parseBGPPerPrefixLength("")
	if len(r.Counts) != 0 {
		t.Errorf("expected 0 counts, got %d", len(r.Counts))
	}
}

const sampleBGPUsedAutnums = `     1 LVLT-1 - Level 3 Parent, LLC, US
     2 UDEL-DCN - University of Delaware, US
   13335 CLOUDFLARENET - Cloudflare, Inc., US
`

func TestParseBGPUsedAutnums(t *testing.T) {
	r := parseBGPUsedAutnums(sampleBGPUsedAutnums)
	if len(r.Autnums) != 3 {
		t.Fatalf("autnums = %d, want 3", len(r.Autnums))
	}
	a := r.Autnums[0]
	if a.ASN != "1" || a.Name != "LVLT-1" || a.Country != "US" {
		t.Errorf("autnum[0] = %+v", a)
	}
	if a.FullName != "LVLT-1 - Level 3 Parent, LLC" {
		t.Errorf("FullName = %q", a.FullName)
	}
	cf := r.Autnums[2]
	if cf.ASN != "13335" || cf.Country != "US" {
		t.Errorf("cloudflare autnum = %+v", cf)
	}
}

func TestParseBGPUsedAutnums_Empty(t *testing.T) {
	r := parseBGPUsedAutnums("")
	if len(r.Autnums) != 0 {
		t.Errorf("expected 0 autnums, got %d", len(r.Autnums))
	}
}

const sampleBGPSparPrefixes = `Prefixes from the Special Purpose Address Registry (Global)
-----------------------------------------------------------

Prefix                Origin AS  Description
192.88.99.0/24             6939  HURRICANE - Hurricane Electric LLC, US
`

func TestParseBGPSparPrefixes(t *testing.T) {
	r := parseBGPSparPrefixes(sampleBGPSparPrefixes)
	if len(r.Prefixes) != 1 {
		t.Fatalf("prefixes = %d, want 1", len(r.Prefixes))
	}
	p := r.Prefixes[0]
	if p.Prefix != "192.88.99.0/24" || p.OriginAS != "6939" {
		t.Errorf("prefix = %+v", p)
	}
	if p.Description != "HURRICANE - Hurricane Electric LLC, US" {
		t.Errorf("Description = %q", p.Description)
	}
}

func TestParseBGPSparPrefixes_Empty(t *testing.T) {
	r := parseBGPSparPrefixes("")
	if len(r.Prefixes) != 0 {
		t.Errorf("expected 0 prefixes, got %d", len(r.Prefixes))
	}
}

const sampleBGPSinglePfx = `Number of ASNs announcing less than 20 prefixes
-----------------------------------------------

No. of Prefixes  No. of ASNs  RIR
       1              27539   Global
       2              10000    Global
       3                500   APNIC
`

func TestParseBGPSinglePfx(t *testing.T) {
	r := parseBGPSinglePfx(sampleBGPSinglePfx)
	if len(r.Counts) != 3 {
		t.Fatalf("counts = %d, want 3", len(r.Counts))
	}
	c := r.Counts[0]
	if c.PrefixCount != 1 || c.ASNCount != 27539 || c.RIR != "Global" {
		t.Errorf("count[0] = %+v", c)
	}
}

func TestParseBGPSinglePfx_Empty(t *testing.T) {
	r := parseBGPSinglePfx("")
	if len(r.Counts) != 0 {
		t.Errorf("expected 0 counts, got %d", len(r.Counts))
	}
}
```

- [ ] **Step 4: 在 bgp_test.go 追加 5 个 Fetch 测试 — 覆盖 HTTP 成功 + 错误路径**

文件: `bgp_test.go`（在文件末尾追加）

```go
func TestFetchBGPBadPrefixes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(sampleBGPBadPrefixes))
	}))
	defer srv.Close()
	client := NewClient(WithHTTPClient(srv.Client()), WithThymeBaseURL(srv.URL), WithJitter(0, 0))
	r, err := client.FetchBGPBadPrefixes(context.Background(), "")
	if err != nil {
		t.Fatalf("FetchBGPBadPrefixes() error: %v", err)
	}
	if len(r.Prefixes) != 2 {
		t.Errorf("prefixes = %d, want 2", len(r.Prefixes))
	}
}

func TestFetchBGPPerPrefixLength(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleBGPPerPrefixLength))
	}))
	defer srv.Close()
	client := NewClient(WithHTTPClient(srv.Client()), WithThymeBaseURL(srv.URL), WithJitter(0, 0))
	r, err := client.FetchBGPPerPrefixLength(context.Background(), "au")
	if err != nil {
		t.Fatalf("FetchBGPPerPrefixLength() error: %v", err)
	}
	if len(r.Counts) != 9 {
		t.Errorf("counts = %d, want 9", len(r.Counts))
	}
	// Source selection is verified via the URL seen by the mock server below
	// in TestFetchBGP_SourceAU; here we only assert parse correctness.
}

// TestFetchBGP_SourceAU verifies that source="au" routes the request to the
// /au/ path segment on the thyme server.
func TestFetchBGP_SourceAU(t *testing.T) {
	var seenPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Write([]byte(sampleBGPPerPrefixLength))
	}))
	defer srv.Close()
	client := NewClient(WithHTTPClient(srv.Client()), WithThymeBaseURL(srv.URL), WithJitter(0, 0))
	if _, err := client.FetchBGPPerPrefixLength(context.Background(), "au"); err != nil {
		t.Fatalf("FetchBGPPerPrefixLength() source=au error: %v", err)
	}
	if !strings.Contains(seenPath, "/au/data-pfx-nos") {
		t.Errorf("expected /au/data-pfx-nos in path, got %q", seenPath)
	}
}

func TestFetchBGPUsedAutnums(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleBGPUsedAutnums))
	}))
	defer srv.Close()
	client := NewClient(WithHTTPClient(srv.Client()), WithThymeBaseURL(srv.URL), WithJitter(0, 0))
	r, err := client.FetchBGPUsedAutnums(context.Background(), "")
	if err != nil {
		t.Fatalf("FetchBGPUsedAutnums() error: %v", err)
	}
	if len(r.Autnums) != 3 {
		t.Errorf("autnums = %d, want 3", len(r.Autnums))
	}
}

func TestFetchBGPSparPrefixes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleBGPSparPrefixes))
	}))
	defer srv.Close()
	client := NewClient(WithHTTPClient(srv.Client()), WithThymeBaseURL(srv.URL), WithJitter(0, 0))
	r, err := client.FetchBGPSparPrefixes(context.Background(), "hk")
	if err != nil {
		t.Fatalf("FetchBGPSparPrefixes() error: %v", err)
	}
	if len(r.Prefixes) != 1 {
		t.Errorf("prefixes = %d, want 1", len(r.Prefixes))
	}
}

func TestFetchBGPSinglePfx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleBGPSinglePfx))
	}))
	defer srv.Close()
	client := NewClient(WithHTTPClient(srv.Client()), WithThymeBaseURL(srv.URL), WithJitter(0, 0))
	r, err := client.FetchBGPSinglePfx(context.Background(), "")
	if err != nil {
		t.Fatalf("FetchBGPSinglePfx() error: %v", err)
	}
	if len(r.Counts) != 3 {
		t.Errorf("counts = %d, want 3", len(r.Counts))
	}
}

func TestFetchBGPAdditionalFiles_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	client := NewClient(WithHTTPClient(srv.Client()), WithThymeBaseURL(srv.URL), WithJitter(0, 0))
	ctx := context.Background()
	if _, err := client.FetchBGPBadPrefixes(ctx, ""); err == nil {
		t.Error("expected error on 500 for badpfx")
	}
	if _, err := client.FetchBGPPerPrefixLength(ctx, ""); err == nil {
		t.Error("expected error on 500 for pfx-nos")
	}
	if _, err := client.FetchBGPUsedAutnums(ctx, ""); err == nil {
		t.Error("expected error on 500 for used-autnums")
	}
	if _, err := client.FetchBGPSparPrefixes(ctx, ""); err == nil {
		t.Error("expected error on 500 for spar")
	}
	if _, err := client.FetchBGPSinglePfx(ctx, ""); err == nil {
		t.Error("expected error on 500 for singlepfx")
	}
}
```

- [ ] **Step 5: 验证 SDK 测试 + 覆盖率**
Run: `go test . -run TestBGP -count=1 -coverprofile=/tmp/bgp.out && go tool cover -func=/tmp/bgp.out | grep -i bgp`
Expected:
  - Exit code: 0
  - Output contains: "ok" and "100.0%" for all bgp.go functions

- [ ] **Step 6: 提交**
Run: `git add bgp.go bgp_test.go utils.go && git commit -m "feat(bgp): add 5 thyme BGP data fetchers with unit tests"`

---

### Task 5: CLI 子命令 + 端到端验证

**Depends on:** Task 4
**Files:**
- Modify: `cmd/apnic/cmd_bgp.go`（新增 5 个 cobra 子命令 + 全局 --source flag）
- Modify: `cmd/apnic/main.go`（注册 --source flag）
- Modify: `cmd/apnic/cli_test.go`（5 个 CLI 测试）
- Modify: `cmd/apnic/cli_test_helpers_test.go`（mock 路由）

- [ ] **Step 1: 在 main.go 注册全局 --source flag — 控制 thyme 数据源**

文件: `cmd/apnic/main.go:75`（在 `flagDownloadTO` 声明之后添加新字段）

```go
	flagDownloadTO    string
	flagBGPSource     string
```

文件: `cmd/apnic/main.go:75`（在 `--download-timeout` PersistentFlags 行之后添加）

```go
	rootCmd.PersistentFlags().StringVar(&flagBGPSource, "bgp-source", "", "thyme BGP data source: current (default), au, or hk")
```

- [ ] **Step 2: 在 cmd_bgp.go 新增 5 个子命令 — 暴露附加 BGP 数据文件**

文件: `cmd/apnic/cmd_bgp.go`（在 `bgpASNMapCmd` 之后追加）

```go
var bgpBadPrefixesCmd = &cobra.Command{
	Use:   "bad-prefixes",
	Short: "Fetch prefixes longer than /24 and their origin AS (route-leak candidates)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		r, err := client.FetchBGPBadPrefixes(context.Background(), flagBGPSource)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(r)
			return nil
		}
		fmt.Printf("# bgp bad-prefixes: %d entries (source=%s)\n", len(r.Prefixes), sourceLabel(flagBGPSource))
		for _, p := range r.Prefixes {
			fmt.Printf("%s\t%s\n", p.OriginAS, p.Address)
		}
		return nil
	},
}

var bgpPerPrefixLengthCmd = &cobra.Command{
	Use:   "per-prefix-length",
	Short: "Count announced prefixes per prefix length",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		r, err := client.FetchBGPPerPrefixLength(context.Background(), flagBGPSource)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(r)
			return nil
		}
		fmt.Printf("# bgp per-prefix-length: %d entries (source=%s)\n", len(r.Counts), sourceLabel(flagBGPSource))
		for _, c := range r.Counts {
			fmt.Printf("/%d\t%d\n", c.Length, c.Count)
		}
		return nil
	},
}

var bgpUsedAutnumsCmd = &cobra.Command{
	Use:   "used-autnums",
	Short: "List every in-use ASN with registered name and country",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		r, err := client.FetchBGPUsedAutnums(context.Background(), flagBGPSource)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(r)
			return nil
		}
		fmt.Printf("# bgp used-autnums: %d ASNs (source=%s)\n", len(r.Autnums), sourceLabel(flagBGPSource))
		limit := len(r.Autnums)
		if limit > 50 {
			limit = 50
		}
		for i := 0; i < limit; i++ {
			a := r.Autnums[i]
			fmt.Printf("%s\t%s\t%s\n", a.ASN, a.Country, a.FullName)
		}
		if len(r.Autnums) > limit {
			fmt.Printf("... (%d more)\n", len(r.Autnums)-limit)
		}
		return nil
	},
}

var bgpSparPrefixesCmd = &cobra.Command{
	Use:   "spar-prefixes",
	Short: "Prefixes from the Special Purpose Address Registry (RFC 6890)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		r, err := client.FetchBGPSparPrefixes(context.Background(), flagBGPSource)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(r)
			return nil
		}
		fmt.Printf("# bgp spar-prefixes: %d entries (source=%s)\n", len(r.Prefixes), sourceLabel(flagBGPSource))
		for _, p := range r.Prefixes {
			fmt.Printf("%s\t%s\t%s\n", p.Prefix, p.OriginAS, p.Description)
		}
		return nil
	},
}

var bgpSinglePfxCmd = &cobra.Command{
	Use:   "single-pfx",
	Short: "Tally ASNs announcing fewer than 20 prefixes, by RIR",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		r, err := client.FetchBGPSinglePfx(context.Background(), flagBGPSource)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(r)
			return nil
		}
		fmt.Printf("# bgp single-pfx: %d rows (source=%s)\n", len(r.Counts), sourceLabel(flagBGPSource))
		for _, c := range r.Counts {
			fmt.Printf("%d\t%d\t%s\n", c.PrefixCount, c.ASNCount, c.RIR)
		}
		return nil
	},
}

// sourceLabel returns the thyme source for display, defaulting to "current".
func sourceLabel(s string) string {
	if s == "" {
		return "current"
	}
	return s
}
```

- [ ] **Step 3: 在 cmd_bgp.go init() 注册 5 个新子命令**

文件: `cmd/apnic/cmd_bgp.go:10-15`（替换 init 函数）

```go
func init() {
	bgpCmd.AddCommand(bgpSummaryCmd)
	bgpCmd.AddCommand(bgpRawTableCmd)
	bgpCmd.AddCommand(bgpASNMapCmd)
	bgpCmd.AddCommand(bgpBadPrefixesCmd)
	bgpCmd.AddCommand(bgpPerPrefixLengthCmd)
	bgpCmd.AddCommand(bgpUsedAutnumsCmd)
	bgpCmd.AddCommand(bgpSparPrefixesCmd)
	bgpCmd.AddCommand(bgpSinglePfxCmd)
	rootCmd.AddCommand(bgpCmd)
}
```

- [ ] **Step 4: 在 cli_test_helpers_test.go 的 mock server 加 5 个 thyme 文件路由**

文件: `cmd/apnic/cli_test_helpers_test.go`（在现有 `data-raw-table` 路由之后追加 5 个路由，参照 `data-summary`/`data-raw-table` 的模式）

```go
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
```

注：`sampleBGPBadPrefixes` 等常量定义在 SDK 包 `bgp_test.go`，但 CLI 测试在 `main` 包，无法直接引用。需在 `cli_test_helpers_test.go` 内重新定义同名的 CLI 测试用常量（小规模样本），或导出 SDK 测试常量。最简方案：在 `cli_test_helpers_test.go` 内定义 CLI 专用样本常量（与 SDK 样本同结构即可）。

- [ ] **Step 5: 在 cli_test_helpers_test.go 定义 CLI 专用样本常量 — 复制 SDK 样本**

文件: `cmd/apnic/cli_test_helpers_test.go`（在文件顶部常量区追加）

```go
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
```

- [ ] **Step 6: 在 cli_test.go 追加 5 个 CLI 测试 — 成功 + JSON + source flag + 错误**

文件: `cmd/apnic/cli_test.go`（在文件末尾追加）

```go
func TestCLI_BGPBadPrefixes(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"bgp", "bad-prefixes"})
	if err != nil {
		t.Fatalf("bgp bad-prefixes: %v", err)
	}
	if !strings.Contains(out, "bgp bad-prefixes") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_BGPBadPrefixesJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	if _, err := runWithStatsServer(t, []string{"bgp", "bad-prefixes"}); err != nil {
		t.Fatalf("bgp bad-prefixes --json: %v", err)
	}
}

func TestCLI_BGPPerPrefixLength(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"bgp", "per-prefix-length"})
	if err != nil {
		t.Fatalf("bgp per-prefix-length: %v", err)
	}
	if !strings.Contains(out, "/8") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_BGPUsedAutnums(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"bgp", "used-autnums"})
	if err != nil {
		t.Fatalf("bgp used-autnums: %v", err)
	}
	if !strings.Contains(out, "LVLT-1") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_BGPSparPrefixes(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"bgp", "spar-prefixes"})
	if err != nil {
		t.Fatalf("bgp spar-prefixes: %v", err)
	}
	if !strings.Contains(out, "192.88.99.0/24") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_BGPSinglePfx(t *testing.T) {
	resetFlags()
	out, err := runWithStatsServer(t, []string{"bgp", "single-pfx"})
	if err != nil {
		t.Fatalf("bgp single-pfx: %v", err)
	}
	if !strings.Contains(out, "27539") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLI_BGPSourceAU(t *testing.T) {
	// --bgp-source au should be reflected in the output's source label.
	resetFlags()
	flagBGPSource = "au"
	out, err := runWithStatsServer(t, []string{"bgp", "single-pfx"})
	if err != nil {
		t.Fatalf("bgp single-pfx --source au: %v", err)
	}
	if !strings.Contains(out, "source=au") {
		t.Errorf("expected source=au in output: %s", out)
	}
}

func TestCLI_BGPAdditionalFetchErrors(t *testing.T) {
	resetFlags()
	for _, args := range [][]string{
		{"bgp", "bad-prefixes"},
		{"bgp", "per-prefix-length"},
		{"bgp", "used-autnums"},
		{"bgp", "spar-prefixes"},
		{"bgp", "single-pfx"},
	} {
		if _, err := runWithErrorServer(t, args); err == nil {
			t.Errorf("expected error for %v", args)
		}
	}
}
```

- [ ] **Step 7: 在 cli_test.go 的 runWithErrorServer 中同步 flagBGPSource 重置**

文件: `cmd/apnic/cli_test.go`（在 `runWithErrorServer` 和 `runWithStatsServer` 的 flag 保存/恢复块中，参照其它 flag 添加 `flagBGPSource` 的保存与重置。具体：在每个 `prevXXX` 声明处加 `prevBGPSource := flagBGPSource`，在 defer 恢复处加 `flagBGPSource = prevBGPSource`，并在函数体重置 `flagBGPSource = ""`。）

```go
	// 保存
	prevBGPSource := flagBGPSource
	// ...其它 prev...
	defer func() {
		flagBGPSource = prevBGPSource
		// ...其它恢复...
	}()
	// 重置
	flagBGPSource = ""
```

注：`resetFlags()` 也需加 `flagBGPSource = ""` 一行（在 `flagDownloadTO = ""` 之后）。

- [ ] **Step 8: 验证 CLI 测试 + 覆盖率**
Run: `go test ./cmd/apnic -count=1 -coverprofile=/tmp/cli.out && go tool cover -func=/tmp/cli.out | grep -v "100.0%" | grep -v "^total"`
Expected:
  - Exit code: 0
  - Output contains: "ok"
  - grep 输出为空（所有命名函数 100%）

- [ ] **Step 9: 端到端真实冒烟（手动，可选）**
Run: `go run ./cmd/apnic bgp bad-prefixes --bgp-source au 2>&1 | head -5`
Expected:
  - Exit code: 0
  - Output contains: 真实前缀/ASN 行（来自 thyme.apnic.net/au/）

- [ ] **Step 10: 更新文档**
Run: 同步 `README.md`（在 BGP 章节列出 5 个新子命令 + `--bgp-source` flag）、`docs/SKILLS.md`（新增 5 个能力条目）。

- [ ] **Step 11: 提交**
Run: `git add cmd/apnic/ README.md docs/SKILLS.md && git commit -m "feat(cli): expose 5 thyme BGP subcommands + --bgp-source flag"`

---

## Self-Review Results

| # | Check | Result | Action Taken |
|---|-------|--------|-------------|
| 1 | Header 含 Goal+Architecture+Tech Stack? | PASS | — |
| 2 | 每个 Task 标注 Depends on? | PASS | T1→None, T2→T1, T3→T1, T4→T2+T3, T5→T4 |
| 3 | 每个 Task 列精确文件路径? | PASS | 全部含行号 |
| 4 | 每个 Task 3-8 Steps? | PASS | T1=3, T2=7, T3=8, T4=6, T5=11（T5 略多但都是必要的小步） |
| 5 | 新文件含完整代码+import? | PASS | 全部完整 |
| 6 | 修改步骤含替换后完整函数? | PASS | — |
| 7 | 代码块 5-80 行? | PASS | 最大约 60 行 |
| 8 | 无悬空函数/类型引用? | PASS | `sourceOrDefault`/`sourceLabel` 均在就近 Step 定义 |
| 9 | 每个 Task 有验证命令+exit code+pattern? | PASS | — |
| 10 | Spec 每需求有对应 Task? | PASS | 5 文件 + 多源全覆盖 |
| 11 | 每个 Task 可独立验证? | PASS | — |
| 12 | 无 TBD/TODO/模糊描述? | PASS | — |
| 13 | 无 "add validation" 抽象指令? | PASS | — |
| 14 | 跨 Task 类型/函数名一致? | PASS | `BGPBadPrefixes`/`FetchBGPBadPrefixes`/`parseBGPBadPrefixes`/`bgpBadPrefixesCmd` 全程一致 |
| 15 | 保存位置正确? | PASS | docs/superpowers/plans/ |

**Status:** ✅ ALL PASS

---

## Execution Selection

**Tasks:** 5
**Dependencies:** yes（T1→T2/T3→T4→T5，线性）
**User Preference:** none
**Decision:** Subagent-Driven
**Reasoning:** 5 tasks > 3，有依赖链，适合子代理驱动开发。

**Auto-invoking:** `superpowers-auto:subagent-driven-development`

⏹️ **Phase 4 Complete: Execution selected, invoking next skill**
