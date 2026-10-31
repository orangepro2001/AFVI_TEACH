#!/usr/bin/env bash
# 一句话调用本机的 llm-mcp（自动解析 tailnet 地址与令牌）。
#
#   ./scripts/ask.sh "用一句话解释 QUIC"        # 默认目标（deepseek，中等任务）
#   ./scripts/ask.sh "设计一个限流方案" gpt     # 高价高推理档（复杂任务）
#   ./scripts/ask.sh --models                  # 列出可用目标/成本档/别名
#   ./scripts/ask.sh --url                     # 打印当前使用的 MCP 地址
#   ./scripts/ask.sh --health                  # 健康检查
#   LLM_MCP_URL=http://127.0.0.1:8790/mcp ./scripts/ask.sh --url
set -euo pipefail

CONF_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/llm-mcp"
TOKEN_FILE="$CONF_DIR/token"
PORT="${LLM_MCP_PORT:-8790}"

# 地址解析顺序：LLM_MCP_URL → tailscale 自报的 MagicDNS 名 → tailscale serve 状态 → 本机回环
resolve_url() {
	if [ -n "${LLM_MCP_URL:-}" ]; then
		printf '%s\n' "$LLM_MCP_URL"
		return
	fi
	if command -v tailscale >/dev/null 2>&1; then
		local dns
		dns=$(tailscale status --json 2>/dev/null |
			python3 -c 'import sys,json;print(json.load(sys.stdin)["Self"]["DNSName"].rstrip("."))' 2>/dev/null || true)
		if [ -n "$dns" ]; then
			printf 'https://%s/mcp\n' "$dns"
			return
		fi
		dns=$(tailscale serve status 2>/dev/null | grep -o 'https://[^ ]*' | head -1 | sed 's#/$##')
		if [ -n "$dns" ]; then
			printf '%s/mcp\n' "$dns"
			return
		fi
	fi
	printf 'http://127.0.0.1:%s/mcp\n' "$PORT"
}

URL=$(resolve_url)
# 令牌是可选的：配置了 token_file 才需要（默认为空 = 纯 URL 即可调用）
TOKEN=$(cat "$TOKEN_FILE" 2>/dev/null || true)
AUTH=()
if [ -n "$TOKEN" ]; then
	AUTH=(-H "Authorization: Bearer $TOKEN")
fi

rpc() { # rpc <json-body>
	curl -sS -m 900 -X POST "$URL" \
		-H 'Content-Type: application/json' "${AUTH[@]}" -d "$1"
}

case "${1:-}" in
--url | -u)
	printf '%s\n' "$URL"
	exit 0
	;;
--health)
	curl -s "${AUTH[@]}" "$URL/healthz"
	echo
	exit 0
	;;
--models | --list | -l)
	rpc '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_models","arguments":{}}}' |
		python3 -c '
import sys, json
m = json.load(sys.stdin)
if "error" in m:
    print("协议错误:", m["error"].get("message")); raise SystemExit(1)
d = json.loads(m["result"]["content"][0]["text"])
for p in d["providers"]:
    mark = "*" if p.get("default") else " "
    aliases = ",".join(k + "=" + v for k, v in (p.get("models") or {}).items()) or "-"
    effort = (p.get("params") or {}).get("reasoning_effort", "-")
    print(mark, "%-9s %-26s cost=%-5s effort=%-5s %s" % (
        p["name"], p["default_model"], str(p.get("cost") or "-"), effort, aliases))
    if p.get("note"):
        print("   ", p["note"])
'
	exit 0
	;;
--help | -h | "")
	sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'
	exit 0
	;;
esac

PROMPT="$1"
MODEL="${2:-}"
BODY=$(PROMPT="$PROMPT" MODEL="$MODEL" python3 -c '
import json, os
args = {"prompt": os.environ["PROMPT"], "verbose": False}
if os.environ.get("MODEL"):
    args["model"] = os.environ["MODEL"]
print(json.dumps({"jsonrpc": "2.0", "id": 1, "method": "tools/call",
                  "params": {"name": "ask", "arguments": args}}, ensure_ascii=False))
')

rpc "$BODY" | python3 -c '
import sys, json
raw = sys.stdin.read()
try:
    m = json.loads(raw)
except Exception:
    print("服务没有返回 JSON（检查服务是否在跑、地址是否正确；body 开头：%r）" % raw[:200])
    raise SystemExit(1)
if "error" in m:
    print("协议错误:", m["error"].get("message"))
    raise SystemExit(1)
r = m["result"]
print(r["content"][0]["text"])
raise SystemExit(1 if r.get("isError") else 0)
'
