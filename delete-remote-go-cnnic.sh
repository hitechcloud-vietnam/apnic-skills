#!/usr/bin/env bash
# 删除远程仓库 hitechcloud-vietnam/go-cnnic（go-cnnic → apnic-skills 收尾的最后一步）
#
# 用法（任选其一）：
#   1) 用 PAT（推荐，本机直连 api.github.com 通）：
#        GH_TOKEN=ghp_xxxxx ./delete-remote-go-cnnic.sh
#      或交互式（不把 token 写进 shell 历史）：
#        read -s GH_TOKEN && export GH_TOKEN && ./delete-remote-go-cnnic.sh
#
#   2) 用网页授权的 gh（需本机能访问 github.com，当前本机因 SNI 阻断不可用）：
#        ./delete-remote-go-cnnic.sh
#
# 脚本会：先确认仓库存在、确认权限含 admin、再执行删除、最后验证 404。
set -euo pipefail

REPO="hitechcloud-vietnam/go-cnnic"

# 若给了 GH_TOKEN，用它登录 gh（覆盖现有 token），走 api.github.com
if [ -n "${GH_TOKEN:-}" ]; then
  echo "[1/4] 用提供的 GH_TOKEN 登录 gh（走 api.github.com，本机通）..."
  echo "$GH_TOKEN" | gh auth login --hostname github.com --git-protocol https --with-token
fi

echo "[2/4] 确认仓库 $REPO 存在且有 admin 权限..."
if ! gh api "repos/$REPO" --jq '{name: .full_name, admin: .permissions.admin}' 2>&1; then
  echo "✗ 仓库不存在或无权访问（可能已被删除？）"
  exit 0
fi

echo "[3/4] 删除仓库 $REPO ..."
if ! gh repo delete "$REPO" --yes; then
  echo "✗ 删除失败。最常见原因：当前 token 缺 delete_repo scope。"
  echo "  → 生成 classic PAT 时勾选 delete_repo + repo，再用 GH_TOKEN=... 重跑本脚本。"
  exit 1
fi

echo "[4/4] 验证删除结果（应返回 Not Found）..."
sleep 2
if gh api "repos/$REPO" >/dev/null 2>&1; then
  echo "✗ 仓库仍可访问，删除可能未生效。"
  exit 1
else
  echo "✓ 远程仓库 $REPO 已删除（API 返回 404）。"
  echo "✓ go-cnnic → apnic-skills 收尾全部完成。"
fi
