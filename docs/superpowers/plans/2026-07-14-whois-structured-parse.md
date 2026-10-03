# Whois 结构化解析修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`
> Steps use checkbox (`- [ ]`) syntax.

**Goal:** 修复 whois 解析器使其对真实 APNIC whois 响应正确产出结构化结果——当前 `WhoisInfo` 的 CIDR/Created/Parent 字段对真实响应恒为空/零值（因解析器等的是 APNIC 从不发送的 `CIDR:`/`created:`/`parent:` key），且多对象串联响应会把 org/role 等次要对象的字段污染进主对象。

**Architecture:** 真实 APNIC whois 对一个 IP 的响应是**多个对象用空行分隔串联**（inetnum 主对象 + irt + organisation + role + route 对象）。当前 `ParseWhoisResponse` 逐行累加所有对象的 key:value，导致：(1) 等待的 `CIDR:`/`parent:`/`created:` key 在真实响应中不存在，故 CIDR/Parent/Created 恒空；(2) 次要对象的 `country`/`org-name` 覆盖主对象值。修复数据流：原始 whois 文本 → 按空行切对象块 → 主对象（第一个含 inetnum/inet6num/aut-num/route 的块）提 Network/Country/NetName/Status/LastUpdated → 跨对象补充 route→CIDR+OriginASN、org-name→OrgName、abuse-c→AbuseContact → 返回结构化 WhoisInfo。CIDR 优先从 `route:` 字段取（真实来源），不实现 range→CIDR 计算（避免引入计算错误，无 route 对象时 CIDR 留空符合真实情况）。同步修正 testutil 样本（当前是含虚假字段的编造样本）和 cmd_whois.go 输出。

**Tech Stack:** Go 1.25.0, 标准库 net(仅 type，不强制)/strings/bufio/time, 模块 `github.com/hitechcloud-vietnam/apnic-skills`, 子包 `internal/models` `internal/query` `internal/testutil` `cmd/apnic`

**Risks:**
- Task 1 多对象切分：APNIC 响应用空行分隔对象，但注释行 `%` 也穿插在对象间。须先按空行切块，每块内部跳过 `%`/`#` 注释行。主对象识别用"块内是否含 inetnum/inet6num/aut-num/route key" → 缓解：遍历块，第一个匹配的块为主对象，后续块仅提取补充字段（route→CIDR+Origin, org-name→OrgName）
- Task 2 修改 `SampleWhoisResponse`（testutil/fixtures.go）和 `sampleWhois`（cli_test.go 本地常量）会影响 11+5 处引用测试 → 缓解：先 grep 全部引用点，样本改为真实格式后同步更新断言；保留样本的核心字段（inetnum/country/descr/last-modified）使多数"response 非空"类断言不受影响
- Task 1 的 `route:` 提取 CIDR：route 对象在响应末尾，且一个 inetnum 可能对应多个 route（如 1.1.1.0/24 + 1.1.1.0/25）→ 缓解：CIDR 用 `[]string` 收集所有 route 值，去重
- Task 4 实测依赖网络（whois.apnic.net:43）→ 缓解：实测用 `go run` 手动验证，不加入测试套件；测试套件仍用本地 mockWhoisServer 避免网络依赖

---

### Task 1: 扩展 WhoisInfo 模型并重写 ParseWhoisResponse 按对象切分

**Depends on:** None
**Files:**
- Modify: `internal/models/models.go:123-132`（WhoisInfo 加字段）
- Modify: `internal/query/whois.go:97-142`（重写 ParseWhoisResponse）

- [ ] **Step 1: 扩展 WhoisInfo 模型 — 新增 NetName/Status/OriginASN/AbuseContact 字段**

文件: `internal/models/models.go:123-132`（替换整个 WhoisInfo 结构体）

```go
// WhoisInfo represents parsed Whois response information.
// Fields are extracted from the primary object (inetnum/inet6num/aut-num/route)
// of an APNIC whois response, with CIDR and OriginASN supplemented from any
// route object. Empty fields mean the corresponding key was absent from the
// response (e.g. no route object → CIDR is nil).
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
	Created      time.Time
	LastUpdated  time.Time
}
```

- [ ] **Step 2: 重写 ParseWhoisResponse — 按空行切对象块，主对象提字段，跨对象补充 route/org-name**

文件: `internal/query/whois.go:97-142`（替换整个 ParseWhoisResponse 函数）

```go
// ParseWhoisResponse parses a raw Whois response into a structured WhoisInfo.
//
// A real APNIC whois response for an IP is a concatenation of several RPSL
// objects separated by blank lines: the primary inetnum/inet6num object, plus
// secondary irt/organisation/role/route objects. We extract the primary object
// (first block containing an inetnum/inet6num/aut-num/route key) for network,
// country, status, and dates, then supplement CIDR/OriginASN from any route
// object and OrgName from any organisation object. This avoids secondary
// objects (e.g. a role object with country: ZZ) polluting the primary fields.
func ParseWhoisResponse(response string) models.WhoisInfo {
	info := models.WhoisInfo{CIDR: []string{}}
	blocks := splitWhoisBlocks(response)

	primaryFound := false
	for _, block := range blocks {
		kv := parseWhoisBlock(block)
		if len(kv) == 0 {
			continue
		}

		// Identify the primary object (inetnum/inet6num/aut-num/route).
		isPrimary := false
		for _, key := range []string{"inetnum", "inet6num", "aut-num", "route", "as-block"} {
			if _, ok := kv[key]; ok {
				isPrimary = true
				break
			}
		}

		if isPrimary && !primaryFound {
			primaryFound = true
			if v, ok := kv["inetnum"]; ok {
				info.Network = v
			} else if v, ok := kv["inet6num"]; ok {
				info.Network = v
			} else if v, ok := kv["aut-num"]; ok {
				info.Network = v
			} else if v, ok := kv["route"]; ok {
				info.Network = v
			} else if v, ok := kv["as-block"]; ok {
				info.Network = v
			}
			if v, ok := kv["netname"]; ok {
				info.NetName = v
			}
			if v, ok := kv["country"]; ok {
				info.Country = v
			}
			if v, ok := kv["status"]; ok {
				info.Status = v
			}
			if v, ok := kv["descr"]; ok && info.OrgName == "" {
				info.OrgName = v
			}
			if v, ok := kv["abuse-c"]; ok {
				info.AbuseContact = v
			}
			if v, ok := kv["parent"]; ok {
				info.Parent = v
			}
			if v, ok := kv["created"]; ok {
				if t, err := parseWhoisDate(v); err == nil {
					info.Created = t
				}
			}
			if v, ok := kv["last-modified"]; ok {
				if t, err := parseWhoisDate(v); err == nil {
					info.LastUpdated = t
				}
			}
		}

		// Supplement CIDR + OriginASN from any route object (may be in its own
		// block or the primary block itself).
		if v, ok := kv["route"]; ok {
			info.CIDR = appendCIDR(info.CIDR, v)
		}
		if v, ok := kv["origin"]; ok && info.OriginASN == "" {
			info.OriginASN = v
		}
		// Supplement OrgName from organisation object if descr did not set it.
		if v, ok := kv["org-name"]; ok && info.OrgName == "" {
			info.OrgName = v
		}
		if v, ok := kv["organisation"]; ok && info.OrgName == "" {
			info.OrgName = v
		}
	}

	return info
}

// splitWhoisBlocks splits a raw whois response into RPSL object blocks on blank
// lines, stripping comment lines (% or #) within each block.
func splitWhoisBlocks(response string) []string {
	var blocks []string
	var current []string
	for _, raw := range strings.Split(response, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			if len(current) > 0 {
				blocks = append(blocks, strings.Join(current, "\n"))
				current = nil
			}
			continue
		}
		if strings.HasPrefix(line, "%") || strings.HasPrefix(line, "#") {
			continue
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		blocks = append(blocks, strings.Join(current, "\n"))
	}
	return blocks
}

// parseWhoisBlock parses a single RPSL object block into a key→value map. Only
// the first value of a repeated key is kept (e.g. the first descr line), since
// the structured model holds single strings. Multi-valued keys like route are
// handled by the caller scanning for that key across blocks.
func parseWhoisBlock(block string) map[string]string {
	kv := make(map[string]string)
	for _, line := range strings.Split(block, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if _, exists := kv[key]; !exists {
			kv[key] = value
		}
	}
	return kv
}

// appendCIDR appends a CIDR string to the list if not already present (a single
// inetnum may map to multiple route objects; dedupe keeps the list clean).
func appendCIDR(list []string, cidr string) []string {
	for _, c := range list {
		if c == cidr {
			return list
		}
	}
	return append(list, cidr)
}
```

- [ ] **Step 3: 验证 query 包编译通过**
Run: `go build ./internal/query/ ./internal/models/ 2>&1 | tail -5`
Expected:
  - Exit code: 0
  - Output 无内容（编译通过）

- [ ] **Step 4: 提交**
Run: `git add internal/models/models.go internal/query/whois.go && git commit -m "refactor(whois): parse multi-object APNIC response, extract CIDR from route object"`

---

### Task 2: 更新 testutil 样本为真实 APNIC 格式并同步解析测试断言

**Depends on:** Task 1
**Files:**
- Modify: `internal/testutil/fixtures.go:249-259`（SampleWhoisResponse 改为真实格式）
- Modify: `internal/query/whois_test.go:40-48,110-127`（更新 CIDR/Parent 断言）
- Modify: `cmd/apnic/cli_test.go:906-916`（sampleWhois 改为真实格式）

- [ ] **Step 1: 更新 SampleWhoisResponse 为真实 APNIC 多对象格式**

文件: `internal/testutil/fixtures.go:249-259`（替换整个 SampleWhoisResponse 常量）

```go
// SampleWhoisResponse is a realistic APNIC whois response for 1.1.1.1: a primary
// inetnum object followed by a route object, separated by a blank line. It uses
// only fields APNIC actually emits (no fabricated CIDR/parent/created keys).
const SampleWhoisResponse = `% Whois information

inetnum:        1.1.1.0 - 1.1.1.255
netname:        APNIC-LABS
descr:          APNIC and Cloudflare DNS Resolver project
country:        AU
org:            ORG-ARAD1-AP
abuse-c:        AA1412-AP
status:         ASSIGNED PORTABLE
last-modified:  2023-04-26T22:57:58Z
source:         APNIC

route:          1.1.1.0/24
origin:         AS13335
descr:          APNIC Research and Development
last-modified:  2023-04-26T02:42:44Z
source:         APNIC
`
```

- [ ] **Step 2: 更新 cmd/apnic cli_test.go 的 sampleWhois 本地样本为真实格式**

文件: `cmd/apnic/cli_test.go:906-916`（替换 sampleWhois 常量）

```go
const sampleWhois = `inetnum:  1.1.1.0 - 1.1.1.255
netname:  APNIC-LABS
descr:    APNIC and Cloudflare DNS Resolver project
country:  AU
status:   ASSIGNED PORTABLE
last-modified: 2023-04-26T22:57:58Z

route:    1.1.1.0/24
origin:   AS13335
last-modified: 2023-04-26T02:42:44Z
`
```

- [ ] **Step 3: 更新 whois_test.go 的 ParseWhoisResponse 断言 — 匹配真实字段**

文件: `internal/query/whois_test.go:110-127`（替换整个 TestParseWhoisResponse 函数）

```go
func TestParseWhoisResponse(t *testing.T) {
	info := ParseWhoisResponse(testutil.SampleWhoisResponse)
	if info.Network != "1.1.1.0 - 1.1.1.255" {
		t.Errorf("network = %q", info.Network)
	}
	if info.NetName != "APNIC-LABS" {
		t.Errorf("netName = %q, want APNIC-LABS", info.NetName)
	}
	if info.Country != "AU" {
		t.Errorf("country = %q", info.Country)
	}
	if len(info.CIDR) != 1 || info.CIDR[0] != "1.1.1.0/24" {
		t.Errorf("cidr = %v, want [1.1.1.0/24]", info.CIDR)
	}
	if info.OriginASN != "AS13335" {
		t.Errorf("originASN = %q, want AS13335", info.OriginASN)
	}
	if info.Status != "ASSIGNED PORTABLE" {
		t.Errorf("status = %q, want ASSIGNED PORTABLE", info.Status)
	}
	if info.OrgName != "APNIC and Cloudflare DNS Resolver project" {
		t.Errorf("orgName = %q", info.OrgName)
	}
	if info.LastUpdated.IsZero() {
		t.Error("expected non-zero lastUpdated date")
	}
}
```

- [ ] **Step 4: 更新 TestQueryWhoisIP 的断言 — Parent 不再来自虚假字段，改为断言新字段**

文件: `internal/query/whois_test.go:40-48`（替换 TestQueryWhoisIP 中的 CIDR/Parent 断言块）

```go
	if len(info.CIDR) != 1 || info.CIDR[0] != "1.1.1.0/24" {
		t.Errorf("cidr = %v, want [1.1.1.0/24]", info.CIDR)
	}
	if info.OrgName != "APNIC and Cloudflare DNS Resolver project" {
		t.Errorf("orgName = %q, want APNIC and Cloudflare DNS Resolver project", info.OrgName)
	}
	if info.OriginASN != "AS13335" {
		t.Errorf("originASN = %q, want AS13335", info.OriginASN)
	}
	if info.NetName != "APNIC-LABS" {
		t.Errorf("netName = %q, want APNIC-LABS", info.NetName)
	}
```

- [ ] **Step 5: 验证 query + cli 测试通过**
Run: `go test ./internal/query/ ./cmd/apnic/ -run 'Whois|ParseWhois' -v 2>&1 | tail -25`
Expected:
  - Exit code: 0
  - Output contains: "PASS" for each whois test
  - Output does NOT contain: "FAIL"

- [ ] **Step 6: 提交**
Run: `git add internal/testutil/fixtures.go internal/query/whois_test.go cmd/apnic/cli_test.go && git commit -m "test(whois): align samples and assertions with real APNIC response format"`

---

### Task 3: 更新 cmd_whois.go CLI 输出补全新字段并对齐 ASN 子命令

**Depends on:** Task 1
**Files:**
- Modify: `cmd/apnic/cmd_whois.go:25-49`（whoisIPCmd 输出补 NetName/Status/OriginASN/AbuseContact）
- Modify: `cmd/apnic/cmd_whois.go:51-75`（whoisASNCmd 输出对齐 IP 子命令）

- [ ] **Step 1: 更新 whoisIPCmd 非 JSON 输出 — 补全 NetName/Status/OriginASN/AbuseContact**

文件: `cmd/apnic/cmd_whois.go:25-49`（替换 whoisIPCmd 整个 RunE 函数体中的非 JSON 输出部分）

```go
var whoisIPCmd = &cobra.Command{
	Use:   "ip <ip>",
	Short: "Parsed whois lookup for an IP address",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()
		ctx := context.Background()
		info, err := client.QueryWhoisIP(ctx, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(info)
			return nil
		}
		fmt.Printf("Network:      %s\n", info.Network)
		fmt.Printf("NetName:      %s\n", info.NetName)
		fmt.Printf("CIDR:         %v\n", info.CIDR)
		fmt.Printf("Country:      %s\n", info.Country)
		fmt.Printf("Org:          %s\n", info.OrgName)
		fmt.Printf("Status:       %s\n", info.Status)
		fmt.Printf("Origin ASN:   %s\n", info.OriginASN)
		fmt.Printf("Abuse:        %s\n", info.AbuseContact)
		fmt.Printf("Parent:       %s\n", info.Parent)
		fmt.Printf("Created:      %s\n", info.Created)
		fmt.Printf("LastUpdated:  %s\n", info.LastUpdated)
		return nil
	},
}
```

- [ ] **Step 2: 更新 whoisASNCmd 非 JSON 输出 — 对齐 IP 子命令字段集**

文件: `cmd/apnic/cmd_whois.go:51-75`（替换 whoisASNCmd 整个 RunE 函数体中的非 JSON 输出部分）

```go
var whoisASNCmd = &cobra.Command{
	Use:   "asn <asn>",
	Short: "Parsed whois lookup for an ASN (e.g. 13335 or AS13335)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asn, err := strconv.ParseInt(normalizeASN(args[0]), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ASN %q: %w", args[0], err)
		}
		client := newClient()
		ctx := context.Background()
		info, err := client.QueryWhoisASN(ctx, asn)
		if err != nil {
			return err
		}
		if flagJSON {
			printJSON(info)
			return nil
		}
		fmt.Printf("Network:      %s\n", info.Network)
		fmt.Printf("NetName:      %s\n", info.NetName)
		fmt.Printf("Country:      %s\n", info.Country)
		fmt.Printf("Org:          %s\n", info.OrgName)
		fmt.Printf("Status:       %s\n", info.Status)
		fmt.Printf("Origin ASN:   %s\n", info.OriginASN)
		fmt.Printf("Abuse:        %s\n", info.AbuseContact)
		fmt.Printf("LastUpdated:  %s\n", info.LastUpdated)
		return nil
	},
}
```

- [ ] **Step 3: 验证 CLI 测试通过（输出格式变更不破坏现有断言）**
Run: `go test ./cmd/apnic/ -run 'Whois' -v 2>&1 | tail -15`
Expected:
  - Exit code: 0
  - Output contains: "PASS"（TestCLI_WhoisIP / TestCLI_WhoisASN / TestCLI_WhoisRaw 等断言只检查 "Network:" 和 "inetnum" 子串，新格式仍含这些）
  - Output does NOT contain: "FAIL"

- [ ] **Step 4: 提交**
Run: `git add cmd/apnic/cmd_whois.go && git commit -m "feat(whois): surface netname/status/origin/abuse in CLI output, align asn with ip"`

---

### Task 4: 实测真实 APNIC whois 查询并验证全量测试

**Depends on:** Task 1, Task 2, Task 3
**Files:**
- None（仅验证，依赖网络访问 whois.apnic.net:43）

- [ ] **Step 1: 实测 whois ip 1.1.1.1 结构化输出 — 确认 CIDR/NetName/Status/Origin 非空**
Run: `go run ./cmd/apnic whois ip 1.1.1.1`
Expected:
  - Exit code: 0
  - Output contains: "Network:      1.1.1.0 - 1.1.1.255"
  - Output contains: "NetName:      APNIC-LABS"
  - Output contains: "CIDR:         [1.1.1.0/24]"
  - Output contains: "Country:      AU"
  - Output contains: "Status:       ASSIGNED PORTABLE"
  - Output contains: "Origin ASN:   AS13335"
  - Output contains: "LastUpdated:  2023"（非零值）

- [ ] **Step 2: 实测 whois ip 1.1.1.1 --json — 确认 JSON 字段非空**
Run: `go run ./cmd/apnic whois ip 1.1.1.1 --json`
Expected:
  - Exit code: 0
  - Output contains: `"CIDR": ["1.1.1.0/24"]`
  - Output contains: `"NetName": "APNIC-LABS"`
  - Output contains: `"OriginASN": "AS13335"`
  - Output contains: `"Status": "ASSIGNED PORTABLE"`

- [ ] **Step 3: 实测 whois ip 203.0.113.0 — 另一 APNIC 块验证 CIDR 提取**
Run: `go run ./cmd/apnic whois ip 203.0.113.0`
Expected:
  - Exit code: 0
  - Output contains: "Network:"（非空）
  - Output contains: "Country:"（非空）

- [ ] **Step 4: 运行全量测试确认无回归**
Run: `go test ./... 2>&1 | tail -12`
Expected:
  - Exit code: 0
  - 所有包输出 `ok`
  - 无 FAIL

- [ ] **Step 5: 提交（如有遗留改动）**
Run: `git status --short || echo "clean"`
Expected:
  - Exit code: 0
  - 工作树干净（所有改动已在 Task 1-3 提交）
