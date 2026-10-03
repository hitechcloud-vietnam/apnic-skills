# cmd/apnic BGP 子命令 JSON 分支 100% 覆盖率补齐计划

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`
> Steps use checkbox (`- [ ]`) syntax.

**Goal:** 为 `cmd/apnic` 中 4 个 thyme BGP 子命令（`per-prefix-length`、`used-autnums`、`spar-prefixes`、`single-pfx`）补齐 `--json` 输出分支的单元测试，使 `cmd/apnic` 包的语句覆盖率从 99.1% 提升到 100.0%，与根包一致。

**Architecture:** 现有 4 个命令的非 JSON（表格）输出分支已有测试覆盖（`TestCLI_BGPPerPrefixLength` 等），但 `flagJSON == true` 分支（`if flagJSON { printJSON(r); return nil }`）未被任何测试触发，导致 `cmd_bgp.go:142-145/163-166/192-195/213-216` 共 4 个代码块计数为 0。补测方式与已通过的 `TestCLI_BGPSummaryJSON`/`TestCLI_BGPRawTableJSON`/`TestCLI_BGPASNMapJSON`/`TestCLI_BGPBadPrefixesJSON` 完全同构：`resetFlags()` → `flagJSON = true` → `runWithStatsServer(t, []string{"bgp", "<sub>"})` → 断言输出含对应结构体的 JSON 字段名。无需新增源码，仅追加测试函数。

**Tech Stack:** Go 1.25，标准库 `testing`/`strings`，cobra CLI，`httptest` mock 服务器。复用 `cmd/apnic` 现有测试辅助函数 `resetFlags`/`runWithStatsServer`，无新依赖。

**Risks:**
- JSON 字段名断言需匹配 `models.go` 中结构体字段名 → 缓解：已从 `models.go:469-513` 确认 4 个结构体的顶层 slice 字段名（`Counts`/`Autnums`/`Prefixes`/`Counts`），与 `printJSON` 走 `encoding/json` 默认字段名输出一致
- `runWithStatsServer` 内部已用 `defer` 恢复全局 flag，但 `flagJSON = true` 在调用前设置、调用中被 `defer` 恢复 → 缓解：与现有 4 个 JSON 测试完全相同的写法，已被验证可行
- `single-pfx` 与 `per-prefix-length` 的结构体顶层字段都叫 `Counts` → 断言 `"Counts"` 都能命中，但为区分语义各自断言一个独有子字段（`single-pfx` 断 `"RIR"`，`per-prefix-length` 断 `"/8"` 风格的 Length 数字不便在 JSON 中匹配，改用 `"PrefixCount"`）

---

### Task 1: 为 4 个 BGP 子命令补齐 JSON 输出分支测试

**Depends on:** None
**Files:**
- Modify: `cmd/apnic/cli_test.go:2163`（在 `TestCLI_BGPASNMapJSON` 函数闭合 `}` 之后、`TestCLI_RPKINotificationJSON` 之前插入 4 个测试函数）
- Test: `cmd/apnic/cli_test.go`（同文件，追加的 4 个函数即为测试本体）

- [ ] **Step 1: 在 cli_test.go 追加 4 个 BGP JSON 分支测试函数 — 覆盖 per-prefix-length/used-autnums/spar-prefixes/single-pfx 的 --json 分支**

文件: `cmd/apnic/cli_test.go:2163`（在 `TestCLI_BGPASNMapJSON` 的闭合 `}` 之后、`TestCLI_RPKINotificationJSON` 注释之前插入）

```go
func TestCLI_BGPPerPrefixLengthJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"bgp", "per-prefix-length"})
	if err != nil {
		t.Fatalf("bgp per-prefix-length --json: %v", err)
	}
	if !strings.Contains(out, `"Counts"`) {
		t.Errorf("expected JSON output with Counts field, got: %s", out)
	}
}

func TestCLI_BGPUsedAutnumsJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"bgp", "used-autnums"})
	if err != nil {
		t.Fatalf("bgp used-autnums --json: %v", err)
	}
	if !strings.Contains(out, `"Autnums"`) {
		t.Errorf("expected JSON output with Autnums field, got: %s", out)
	}
}

func TestCLI_BGPSparPrefixesJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"bgp", "spar-prefixes"})
	if err != nil {
		t.Fatalf("bgp spar-prefixes --json: %v", err)
	}
	if !strings.Contains(out, `"Prefixes"`) {
		t.Errorf("expected JSON output with Prefixes field, got: %s", out)
	}
}

func TestCLI_BGPSinglePfxJSON(t *testing.T) {
	resetFlags()
	flagJSON = true
	out, err := runWithStatsServer(t, []string{"bgp", "single-pfx"})
	if err != nil {
		t.Fatalf("bgp single-pfx --json: %v", err)
	}
	if !strings.Contains(out, `"Counts"`) {
		t.Errorf("expected JSON output with Counts field, got: %s", out)
	}
	if !strings.Contains(out, `"RIR"`) {
		t.Errorf("expected JSON output with RIR field, got: %s", out)
	}
}
```

- [ ] **Step 2: 验证 4 个新测试全部通过**
Run: `go test ./cmd/apnic/ -run 'TestCLI_BGP(PerPrefixLength|UsedAutnums|SparPrefixes|SinglePfx)JSON' -v`
Expected:
  - Exit code: 0
  - Output contains: "PASS"
  - Output contains: "4 passed" 或 4 个独立的 "--- PASS: TestCLI_BGP" 行
  - Output does NOT contain: "FAIL" or "Error"

- [ ] **Step 3: 验证 cmd/apnic 包整体覆盖率达到 100%**
Run: `go test ./cmd/apnic/ -coverprofile=/tmp/cmd_cov_final.out && go tool cover -func=/tmp/cmd_cov_final.out | tail -1`
Expected:
  - Exit code: 0
  - Output contains: "100.0%" （`total:` 行显示 100.0%）
  - Output does NOT contain: 任何 "0$" 未覆盖块

- [ ] **Step 4: 验证全仓库（根包 + cmd/apnic）测试全绿且覆盖率 100%**
Run: `go test ./... -coverprofile=/tmp/full_cov.out 2>&1 | tail -5 && echo "---" && go tool cover -func=/tmp/full_cov.out | grep -v "100.0%" || echo "ALL 100%"`
Expected:
  - Exit code: 0
  - Output contains: "ok" for both packages
  - 末尾输出: "ALL 100%"（无任何低于 100% 的函数）

- [ ] **Step 5: 提交**
Run: `git add cmd/apnic/cli_test.go && git commit -m "test(cli): cover --json branch of 4 thyme BGP subcommands for 100% cmd/apnic coverage"`
