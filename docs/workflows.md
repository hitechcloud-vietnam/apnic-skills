# APNIC 工作流封装（Workflows）

> 基于 `apnic` CLI 的多命令组合编排。每个工作流解决一个真实运维/审计问题，可逐条复制运行，或用 `--json` + `jq` 串成脚本。
> 前置：已构建 CLI（`go build -o bin/apnic ./cmd/apnic`）或用 `go run ./cmd/apnic` 替代下文 `apnic`。

---

## 工作流 1：国家资源审计

**目的**：盘点某国（如 CN）在 APNIC 拥有的全部 IPv4/IPv6/ASN 资源，并定位其持有组织。

```bash
# 1) CN 的 IPv4 分配（allocated + assigned）
apnic filter --source delegated --country CN --type ipv4

# 2) CN 的 IPv6 分配
apnic filter --source delegated --country CN --type ipv6

# 3) CN 的 ASN
apnic filter --source delegated --country CN --type asn

# 4) 定位持有组织（扩展版按 opaque-id 聚合）
apnic filter --source extended --country CN --json | jq -r '.Entries[] | "\(.OpaqueID)\t\(.Start)"' | sort -u
```

**预期输出**：步骤 1–3 输出 TSV 行（`CN  ipv4  <start>  <value>  <status>  <date>`）；步骤 4 列出每个 opaque-id 对应的资源起点。

**变体**：
- 只看 allocated（已分配未公告）：`--status allocated`。
- 审计某组织：`--source extended --opaque-id <ID>`。
- 历史对比：`apnic history --type delegated --date 20200101` 与今日 `filter` 结果 diff。

<details>
<summary><b>一键脚本（输出 CN 全资源计数）</b></summary>

```bash
#!/usr/bin/env bash
set -e
for t in ipv4 ipv6 asn; do
  n=$(apnic filter --source delegated --country CN --type $t --json | jq '.Entries | length')
  echo "CN $t: $n 条"
done
```

</details>

---

## 工作流 2：IP 全景调查

**目的**：给定一个 IP（如 `1.1.1.1`），汇聚其 RDAP 注册信息、whois 详情、反向 DNS，形成完整画像。

```bash
IP=1.1.1.1

# 1) RDAP：归属网络、CIDR、国家、持有实体、注册时间
apnic rdap ip $IP --json

# 2) whois：解析后的 Network/CIDR/Country/Org/Parent/Created/Updated
apnic whois ip $IP --json

# 3) 反向 DNS：PTR 记录
apnic reverse-dns $IP

# 4) 点对点历史：看该 IP 在某历史时刻的归属（RFC3339）
apnic rdap ip $IP --date 2020-06-01T00:00:00Z --json
```

**预期输出**：
- RDAP JSON 含 `handle`、`cidr0_cidrs`、`entities`（持有者 handle，如 `AIC3-AP`）、`events`（注册/最后修改时间）。
- whois JSON 含 `Network`、`CIDR`、`Country`、`OrgName`、`Parent`、`Created`、`LastUpdated`。
- 反查 DNS 给出 PTR 域名（若无则 `(no PTR records)`）。

**延伸**：用 RDAP 拿到的实体 handle 深挖持有人：
```bash
apnic rdap entity AIC3-AP --json
```

**跨 RIR 归集**：REx 可把同一持有组织在所有 RIR 的资源一次性聚齐（APNIC RDAP 只覆盖本区域）。先用 `rex resources` 找到该 IP 所属持有者的 opaqueId 与 rir，再聚合：
```bash
# 自定位当前出口网络（覆盖前缀/ASN/经济体）
apnic rex network

# 取最近委派资源，定位目标持有者的 opaqueId 与 rir
apnic rex resources ipv4

# 聚合该持有组织在对应 RIR 的全部 ASN/前缀与规模
apnic rex holder <opaqueId> <rir> --json
```

<details>
<summary><b>一键脚本（合并为单 JSON 报告）</b></summary>

```bash
#!/usr/bin/env bash
IP="${1:?usage: $0 <ip>}"
jq -n --arg ip "$IP" \
  --argjson rdap "$(apnic rdap ip $IP --json)" \
  --argjson whois "$(apnic whois ip $IP --json)" \
  --argjson ptr "$(apnic reverse-dns $IP --json)" \
  '{ip:$ip, rdap:$rdap, whois:$whois, ptr:$ptr}'
```

</details>

---

## 工作流 3：转移与变更追踪

**目的**：审计 IP/ASN 在 RIR 间或组织间的转移，以及资源状态变更，定位异常迁移。

```bash
# 1) 最新转移记录（含跨 RIR 与 APNIC 内部转移）
apnic transfers --json | jq '.Transfers[] | {date:.transfer_date, type, from:.source_organization.name, to:.recipient_organization.name, ip4:.ip4nets, ip6:.ip6nets, asn:.asns}'

# 2) 按年度取 JCR 转移日志
apnic transfers --year 2023 --json

# 3) 最新资源变更（JSON Lines）
apnic changes --json | jq -c '.Changes[] | {cc, custodian, resources, status, type, timestamp}'

# 4) 指定日期变更快照
apnic changes --date 20240101 --json
```

**预期输出**：
- 转移：每条含转移日期、类型（`RESOURCE_TRANSFER`）、源/目的组织名与国家、转移的 IPv4/IPv6/ASN 集合。
- 变更：每条含国家、custodian（opaque-id）、资源 CIDR/ASN 列表、状态、类型（`delegated`/`cc-changed`/`status-changed`）、时间戳。

**变体**：
- 仅看跨 RIR 转移：`jq 'select(.source_rir != .recipient_rir)'`。
- 仅看某国家：`jq 'select(.recipient_organization.country_code == "CN")'`。
- 监控某组织变更：`jq -c 'select(.custodian == "A92E1062")'`。

<details>
<summary><b>一键脚本（导出近期待转移 IPv4 段）</b></summary>

```bash
#!/usr/bin/env bash
apnic transfers --json | jq -r '
  .Transfers[]
  | select(.ip4nets != null)
  | .ip4nets.transfer_set[]
  | "\(.start_address) - \(.end_address)"
' | sort -u
```

</details>

---

## 工作流 4：数据完整性校验

**目的**：下载 APNIC 统计文件并验证其未被篡改——MD5 与官方校验值比对，PGP 签名与公钥核验。

```bash
# 1) 端到端 MD5 完整性校验（下载数据 + 旁挂 .md5，本地计算比对）
apnic verify integrity --type delegated
apnic verify integrity --type delegated-extended
apnic verify integrity --type assigned
apnic verify integrity --type delegated-ipv6-assigned
apnic verify integrity --type legacy

# 2) 取原始 MD5 校验值（BSD 风格 MD5 (file) = <hash>）
apnic verify md5 --type delegated

# 3) 取 PGP 签名
apnic verify asc --type delegated

# 4) 取 APNIC 签名公钥（用于离线验签 .asc）
apnic verify pubkey
```

**预期输出**：
- `integrity` 成功：`OK: delegated (date=latest) MD5 verified`，退出码 `0`。
- 失败（内容被改/网络错误）：非零退出码 + 错误信息。

**变体**：
- 校验历史快照：`apnic verify integrity --type delegated --date 20240101`。
- 批量校验所有类型：
```bash
for t in delegated delegated-extended assigned delegated-ipv6-assigned legacy; do
  apnic verify integrity --type $t || echo "FAIL: $t"
done
```

<details>
<summary><b>一键脚本（全部类型完整性巡检，任一失败即报警）</b></summary>

```bash
#!/usr/bin/env bash
set -e
types=(delegated delegated-extended assigned delegated-ipv6-assigned legacy)
fail=0
for t in "${types[@]}"; do
  if apnic verify integrity --type "$t"; then
    echo "✓ $t"
  else
    echo "✗ $t"; fail=1
  fi
done
exit $fail
```

</details>

---

## 工作流 5：RPKI/BGP 路由可信性核查

> 场景：审计某前缀的路由宣告是否与 RPKI 授权一致，并对照 thyme 实际 BGP 表。

<details>
<summary><b>步骤</b></summary>

```bash
# 1) RRDP notification：当前 RPKI 仓库序号与 snapshot 引用
apnic rpki notification --json | jq '{session: .SessionID, serial: .Serial, snapshot: .Snapshot.URI}'

# 2) 流式解析当前 snapshot：发布/撤回的对象计数
apnic rpki snapshot

# 3) thyme BGP 概览：路由条目数、ROA 覆盖（valid/invalid/none）
apnic bgp summary --json | jq '.Entries[] | select(.Key|test("ROA|routing table entries"))'

# 4) 某前缀的实际 origin ASN（raw table 过滤）
apnic bgp raw-table --json | jq -r '.Routes[] | select(.Prefix=="1.1.1.0/24") | .ASN'

# 5) 对照 IRR：该 ASN 的 aut-num 对象与宣告的路由对象
apnic irr aut-num --json | jq '.Objects[] | select(.PrimaryKey|test("13335"))'
apnic irr route --json | jq '.Objects[] | select(.Attributes.route[0]=="1.1.1.0/24")'
```

```bash
#!/usr/bin/env bash
set -euo pipefail
PREFIX="${1:-1.1.1.0/24}"
echo "== RRDP serial =="; apnic rpki notification --json | jq '.Serial'
echo "== BGP summary =="; apnic bgp summary --json | jq '.Entries[] | select(.Key|test("ROA"))'
echo "== origin ASN for $PREFIX =="; apnic bgp raw-table --json | jq -r --arg p "$PREFIX" '.Routes[]|select(.Prefix==$p)|.ASN'
```

</details>

---

## 通用编排技巧

- **管道化**：所有命令支持 `--json`，可用 `jq`/`grep`/`awk` 二次处理；默认 TSV 适合直接 `sort | uniq -c`。
- **缓存**：SDK 默认 30 分钟缓存，批量工作流内重复请求同源数据只命中一次网络。调 `--cache-ttl 0` 可禁用。`irr` 等大转储走 `Get*` 缓存路径，TTL 内复用。
- **离线/自托管**：`--stats-base-url`/`--rdap-base-url`/`--whois-server`/`--ftp-base-url`/`--rrdp-base-url`/`--thyme-base-url`/`--rex-base-url` 可指向镜像或本地 mock，便于 CI 与隔离测试。
- **大文件下载**：APNIC FTP 对单连接限速 ~8-22 KB/s（delegated 4.3MB、extended、IRR `apnic.db.inetnum.gz` 50MB+ 等）。SDK 默认 `--max-concurrent-downloads 4` 自动分块：探测支持 Range 后切成 ~2MB 小块以 4 并发轮转下载，总吞吐提升 3-4 倍，50MB IRR dump 约 5-8 分钟完成。遇单块超时调小 `--chunk-size 1MB` 或调大 `--download-timeout 300s`。每个分块请求仍继承 stealth 伪装头与限速。
- **反爬**：默认 `--stealth`（浏览器伪装头 + 抖动 + 限速）。高频自动化设 `--rate-limit 2 --jitter 500ms-1500ms` 进一步降频；确需高速可 `--stealth=false --jitter 0-0`（仅发 UA+Accept，向后兼容旧 SDK 行为）。
- **退出码**：脚本中用 `set -e` 或显式 `||` 捕获失败，便于流水线门禁。
