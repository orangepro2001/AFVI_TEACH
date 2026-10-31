# deepseek-mcp（LLM-MCP）

把任意 **OpenAI 兼容端点**（DeepSeek 官方、OmniRouter、OpenRouter、vLLM、Ollama…）包装成一个
**MCP 服务**。客户端**只知道一个地址**就能用：不用 API key、不用模型名、不用 token。

```
Agent ──MCP(https://egg.tail9f7f1f.ts.net/mcp)──▶ llm-mcp ──HTTPS──▶ 你的 LLM 端点
                                                   ▲
                              密钥（600，服务端）+ 默认模型 + reasoning effort 全在这里
```

- 纯 Go 标准库，**零依赖**，单二进制。
- **两种传输**：`stdio`（同机 Agent 当子进程拉起）与 **Streamable HTTP**（tailnet HTTPS，客户端只要 URL）。
- 多 provider、多别名、按 `cost`/`note` 自动生成「什么任务用哪个模型」的说明给客户端看。
- **上游流式 + MCP progress 通知**：慢模型（十几秒首字节、几分钟长回答）不会再撞客户端工具超时。

---

## 0. 现在就能调

```
https://egg.tail9f7f1f.ts.net/mcp                 ← 给所有 Agent（tailnet 内任何机器，无需任何凭据）
http://127.0.0.1:8790/mcp                         ← 本机回环
/home/ubuntu/dev/AFVI_TEACH/LLM-MCP/bin/llm-mcp   ← stdio（同机 Agent）
```

```bash
cd /home/ubuntu/dev/AFVI_TEACH/LLM-MCP
./scripts/ask.sh --url                    # 自动解析地址（域名忘了就跑这个）
./scripts/ask.sh --health                 # 服务活着吗
./scripts/ask.sh --models                 # 目标 / 别名 / 成本档
./scripts/ask.sh "用一句话解释 QUIC"       # 默认 deepseek（中等任务）
./scripts/ask.sh "设计一个限流方案" gpt    # 高价高推理档（复杂任务）
```

裸 curl（不需要任何认证头）：

```bash
curl -s -X POST https://egg.tail9f7f1f.ts.net/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"ask","arguments":{"prompt":"你好"}}}'
```

忘了域名怎么办：

```bash
./scripts/ask.sh --url                                                              # 最省事
tailscale serve status | grep /mcp                                                  # /mcp 那行上方就是域名
tailscale status --json | python3 -c 'import sys,json;print(json.load(sys.stdin)["Self"]["DNSName"])'
```

---

## 1. 当前部署状态

| 项 | 值 |
|---|---|
| 二进制 | `bin/llm-mcp` |
| 服务端配置 | `~/.config/llm-mcp/config.json`（600） |
| 上游密钥 | `~/.config/llm-mcp/api-key`（deepseek）、`~/.config/llm-mcp/gpt-key`（gpt），均 600 |
| 客户端凭据 | **无**（不传 key、不传模型、不传 token） |
| 上游端点 | `https://api.omnirouter.dpdns.org/v1`（两个 provider 共用） |
| 默认目标（中等及以下） | `deepseek` → `deepseek-v4.1-flash-1017`（别名 `flash`），cost `low` |
| 高价目标（复杂任务） | `gpt` → `gpt-6-astra`（别名 `astra`/`hard`），cost `high`，`reasoning_effort: max` |
| 本机监听 | `127.0.0.1:8790`（systemd 用户服务 `llm-mcp.service`，已 enable） |
| **Tailnet 地址** | **`https://egg.tail9f7f1f.ts.net/mcp`**（Tailscale Serve，仅 tailnet 内可达） |
| 客户端 JSON | `~/.config/llm-mcp/mcp-client.json` / 仓库 `mcp-client.example.json`（只有 url 两行） |

---

## 2. 客户端零配置：为什么可以

三样东西都**只存在于服务端**，客户端只有「地址」这一个概念：

| 客户端需要提供 | 实际情况 |
|---|---|
| LLM 的 API key | 不需要。服务端从 `api_key_file` 读，每个 provider 各一把 |
| 模型名 | 不需要。省略 `model` 就用默认 provider（deepseek）；想用贵的才传 `model: "gpt"` |
| MCP 访问 token | **不需要**（默认关闭）。既然服务只绑 `127.0.0.1` + tailnet，凭据是多余的 |

所以一个客户端的完整配置就是 **`url` 一行**：

```json
{ "mcpServers": { "deepseek": { "url": "https://egg.tail9f7f1f.ts.net/mcp" } } }
```

模型分流也由服务端告诉客户端（写进 `initialize.instructions`、`ask`/`chat` 工具描述、`list_models`）：

```
Model routing: omit `model` for ordinary and medium tasks (default deepseek -> deepseek-v4.1-flash-1017).
Pass model="gpt" for genuinely hard or high-stakes work: the most reliable but slowest and most expensive target.
deepseek: 中等及以下任务的默认选择：快、便宜，日常问答/总结/翻译/改代码都够用。
gpt: 可靠但高价：已配置最高档 reasoning effort（reasoning_effort=max）。真正棘手、高风险或需要最强推理的任务才用它，比默认模型慢且贵很多。
```

> 「ultra effort」：这个端点只接受 `none / minimal / low / medium / high / xhigh / max`，`ultra` 会被 400 拒绝，
> 所以 GPT 配的是最高档 **`reasoning_effort: "max"`**（实测推理 token 是 `high` 的 6 倍：77 vs 12）。

---

## 3. 把客户端配置贴进去就行

**任何支持 Streamable HTTP 的客户端**（pi、Claude Code、Cursor…）：

```json
{
  "mcpServers": {
    "deepseek": {
      "type": "http",
      "url": "https://egg.tail9f7f1f.ts.net/mcp",
      "description": "本机 LLM 网关：默认 deepseek（中等任务），model=\"gpt\" 用于复杂任务",
      "timeout": 300
    }
  }
}
```

- 现成文件：`~/.config/llm-mcp/mcp-client.json`（`cat` 出来整段拷走即可）。
- `timeout: 300`（秒）建议给：上游偶发 10–30 s 才回，默认 60 s 会超时。
- `exposure: "direct"` 是 pi 的字段（让模型直接看到工具而不是走 codemode 脚本），别的客户端忽略即可。

**同机 stdio（同样零配置）**：

```json
{ "mcpServers": { "deepseek": { "command": "/home/ubuntu/dev/AFVI_TEACH/LLM-MCP/bin/llm-mcp" } } }
```

命令行：`pi mcp add deepseek --url https://egg.tail9f7f1f.ts.net/mcp`（已实测 `connected, 4 tools`）。

---

## 4. 构建与运行

```bash
cd LLM-MCP
go build -o bin/llm-mcp .                                               # 本机
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/llm-mcp.exe .   # Windows

./bin/llm-mcp                 # stdio（客户端拉起子进程）
./bin/llm-mcp -http 8790      # HTTP，监听 127.0.0.1:8790
./bin/llm-mcp -list           # provider / 模型 / 密钥来源 / 令牌状态
./bin/llm-mcp -print-config   # 生效配置（密钥打码）
```

| 参数 | 说明 |
|---|---|
| `-config PATH` | 配置文件；留空按搜索顺序找 |
| `-endpoint URL` / `-model NAME` / `-name NAME` | 覆盖或新增 provider |
| `-api-key-file FILE` / `-api-key KEY` | 密钥（推荐 `api_key_file`） |
| `-http ADDR` | `8790` = `127.0.0.1:8790`；留空 = stdio |
| `-token TOKEN` | **可选**，只有你想收紧 HTTP 访问时才设（也可放 `token_file`） |
| `-log FILE` | 日志文件（600）；stdio 模式日志永不写 stdout |
| `-timeout DUR` / `-retries N` | 单次超时（默认 180s）/ 尝试次数（默认 3，仅 429·5xx·网络错误重试） |
| `-list` / `-print-config` / `-version` | 打印后退出 |

---

## 5. 配置

搜索顺序：`-config` → `$LLM_MCP_CONFIG` → `./llm-mcp.json` → 可执行文件旁 → `~/.config/llm-mcp/config.json`。
`${VAR}` 与 `~/` 展开可用（`base_url`、`api_key_file`、`token_file`）。

```json
{
  "default": "deepseek",
  "providers": [
    {
      "name": "deepseek",
      "base_url": "https://api.omnirouter.dpdns.org/v1",
      "api_key_file": "~/.config/llm-mcp/api-key",
      "model": "deepseek-v4.1-flash-1017",
      "models": { "flash": "deepseek-v4.1-flash-1017" },
      "max_tokens": 8192,
      "timeout": "300s",
      "cost": "low",
      "note": "中等及以下任务的默认选择：快、便宜"
    },
    {
      "name": "gpt",
      "base_url": "https://api.omnirouter.dpdns.org/v1",
      "api_key_file": "~/.config/llm-mcp/gpt-key",
      "model": "gpt-6-astra",
      "models": { "astra": "gpt-6-astra", "hard": "gpt-6-astra" },
      "params": { "reasoning_effort": "max" },
      "max_tokens": 32768,
      "timeout": "600s",
      "cost": "high",
      "note": "可靠但高价，复杂任务用它"
    }
  ]
}
```

- `base_url`：API 根（`…/v1`）或完整 `…/v1/chat/completions`，缺的自动补。
- `models`：别名表；`model: "reasoner"` / `"gpt/hard"` 都能用。
- `params`：原样并入请求体，**每个 provider 独立**（gpt 的 `reasoning_effort` 不会污染 deepseek）。
- `cost` / `note`：给调用方看的元信息；`cost: high` 的非默认 provider 会被当成「贵但最强」目标写进说明。
- 密钥来源：`api_key` → `api_key_file`（推荐）→ `api_key_env` → `LLM_MCP_API_KEY`。
- HTTP 令牌（**可选**）：`token` 或 `token_file`；不设就是不校验，只靠 tailnet 边界。

模型解析（`model` 参数）：省略 = `default` 的 provider 及其默认模型；`deepseek` = 该 provider 默认模型；
`gpt/hard` = provider + 别名；`hard` = 第一个定义该别名的 provider；任意新名字 = 默认 provider + 字面模型名。

环境变量：`LLM_MCP_CONFIG` `LLM_MCP_BASE_URL` `LLM_MCP_MODEL` `LLM_MCP_NAME` `LLM_MCP_API_KEY_FILE`
`LLM_MCP_API_KEY` `LLM_MCP_HTTP` `LLM_MCP_TOKEN` `LLM_MCP_LOG` `LLM_MCP_TIMEOUT`。

---

## 6. 工具

| 工具 | 作用 | 参数 |
|---|---|---|
| `ask` | 一问一答（主入口，**流式 + 进度心跳**）。远端模型**没有工具、读不到你的文件**，信息都写进 prompt | `prompt`(必填) `system` `model` `temperature` `max_tokens` `session` `verbose` |
| `chat` | 自己控制完整 transcript（few-shot / 续写） | `messages`(必填) `system` `model` … |
| `list_models` | provider / 别名 / 默认模型 / 成本档 / `params` / 密钥来源（不含密钥本身） | `remote` `provider` |
| `reset_session` | 丢掉某会话记忆 | `session`(必填) |

`session`：同 id 的多次 `ask` 会带上前文（服务端内存保留最近 ~60 条 / ~120k 字符，进程重启即清空）。

协议要点：`initialize`（回显客户端版本，未知版本回落 `2025-06-18`）、`tools/list`、`tools/call`、`ping`、
`prompts/list` / `resources/list` 空列表、`notifications/cancelled`（真的中断上游请求）、JSON-RPC 批量。
HTTP：`POST /mcp`（`Accept: text/event-stream` 时回 SSE 单事件；无状态，不要求 `Mcp-Session-Id`）、
`GET /healthz`、`GET /` 信息页；挂在反代子路径下时根路径 POST 也当 MCP（前缀被剥掉也能用）。

---

## 7. 部署：systemd + tailnet HTTPS

`~/.config/systemd/user/llm-mcp.service`（已 enable，linger 已开）：

```ini
[Unit]
Description=llm-mcp - MCP bridge to DeepSeek / OpenAI-compatible endpoints
After=network-online.target

[Service]
Type=simple
ExecStart=/home/ubuntu/dev/AFVI_TEACH/LLM-MCP/bin/llm-mcp -http 127.0.0.1:8790 -log %h/.local/state/llm-mcp.log
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=default.target
```

只监听回环，出口交给 Tailscale Serve：

```bash
tailscale serve --bg --set-path /mcp http://127.0.0.1:8790/mcp
tailscale serve status | grep /mcp
```

---

## 8. 安全边界（为什么不要令牌是可以接受的）

- 服务只绑 `127.0.0.1`，外网**根本进不来**；唯一入口是 Tailscale Serve，**只有你自己 tailnet 里的设备**能到达，
  而且走 HTTPS。上游密钥始终留在服务端，客户端拿不到、也传不进来。
- 代价很明确：**tailnet 内任何设备/任何人**都能用掉你的额度（目前只有 `egg`、`afvi*` 这几台自家机器）。
- 想收紧随时可以（一行，客户端仍只需 URL 的版本不受影响 —— 只是需要带头的客户端）：
  ```bash
  # 1) 配置里加回 "token_file": "~/.config/llm-mcp/token"（文件已存在，600）
  # 2) systemctl --user restart llm-mcp
  # 3) 客户端加 "headers": { "Authorization": "Bearer <token 内容>" }
  ```
- 密钥只存在 `~/.config/llm-mcp/`（`api-key`、`gpt-key`、`config.json` 全 600），不在仓库、不在环境变量、
  不在命令行（`ps` 看不到）、日志不记 prompt/回答正文。仓库 `.gitignore` 挡住 `KEY.txt`/`api-key`/`*.key`/`.env*`/`bin/`。
- 轮换上游 key：只改 `~/.config/llm-mcp/api-key` 或 `gpt-key`，**无需重启**。

---

## 9. 长调用不再超时（为什么以前会失败）

症状：客户端报 `ask` 超时、重试、大单反复失败，而服务端日志里其实还在正常生成。

原因有两个，都在我们这一侧：

1. **请求是非流式的**：上游这个代理要 10–30 s 才吐首字节，长回答更久，客户端只看得到「还没返回」。
2. **没有发 progress 通知**：MCP 客户端（pi 实测默认 60 s）的**每次 `notifications/progress` 都会重置单请求超时**
   （源码里就是这样：`handleProgress` → `armTimeout`）。我们从没发过，所以必然在 60 s 处被掐断。

现在的实现：

- 请求上游时带 `"stream": true` + `"stream_options": {"include_usage": true}`，边收边累积；对方忽略 stream
  直接返回整段 JSON 时自动退回（`streamed=false`）。
- 客户端带了 `_meta.progressToken`（pi 对每个工具调用都会带）时，服务端在**同一请求**里推
  `notifications/progress`：
  - `0` 秒：「已提交给模型…」
  - 每 5 秒心跳：`等待 deepseek 响应（15 秒）` 或 `生成中：407 字（20 秒）`
  - 结束：`完成：407 字，用时 22 秒`
  传输方式按客户端能力选：stdio 直接写 stdout 行；HTTP 用 SSE（`Accept: text/event-stream`）。
- `notifications/cancelled` 现在会真的取消在跑的上游请求（含流式读取）。

实测（真实上游，走 tailnet）：

```
[  0.0s] progress p1/0 已提交给模型…
[  5.0s] progress p1/5.000 生成中：45 字（5 秒）
[  6.5s] progress p1/6.426 完成：266 字，用时 6 秒
[  6.5s] RESULT 1. MCP（Model Context Protocol…）
```

**升级后要重启客户端**（pi 的 stdio 子进程是旧二进制）：`/mcp reconnect deepseek`，或退出重开 pi。
另外仍建议在客户端配置里给 `timeout: 300`：它只影响「没有 progressToken 的客户端」和首字节前的等待。

还有一点是调用习惯：单次让模型产出 1000 行代码本来就慢且容易失败。`ask` 的描述里已经写明
「一次一件事、超出几百行就拆分」，工程上也建议拆成小单（截图里拆到 R01/R02/R03 后就好了）。

## 10. 开发

```bash
cd LLM-MCP
command gofmt -l .     # 应为空（zsh 的 gofmt 别名会改写文件，必须加 command）
go vet ./...
go test ./...          # 纯标准库，Linux 直接跑
```

覆盖：配置/环境变量展开/密钥与令牌文件与权限告警/模型解析/请求体构造/重试与放弃/401 不重试/超时/取消/
SSE 流式（累积、usage、[DONE]、keep-alive、坏帧、error 帧、忽略 stream 的端点回退）/progress 通知（token 透传、
单调递增、无 token 不发、HTTP 走 SSE、进度先于结果）/
工具四件套/会话记忆/上游错误→isError/未知工具/通知无响应/批量/HTTP(401·200·SSE·202·405·healthz·子路径 POST)/
分流说明（instructions + 工具描述 + list_models + 高价目标独立 key/params 不串档）/stdio 端到端（stdout 只有 JSON-RPC）。

```
main.go        参数与环境变量、配置装配、模式选择、-list/-print-config
config.go      Config/Provider、搜索顺序、密钥来源、模型解析、cost/note、打码输出
provider.go    OpenAI 兼容客户端（重试、超时、错误→可操作提示）
mcp.go         JSON-RPC 分发、initialize、路由策略生成、取消注册表、会话存储
mcp_tools.go   ask / chat / list_models / reset_session
stdio.go       stdio 传输（每行一条消息，stdout 只写协议，退出前等在途请求）
http.go        Streamable HTTP + 可选 Bearer + CORS + healthz + 子路径容错
scripts/ask.sh 命令行调用入口（--url / --health / --models / "问题" [model]）
```

---

## 11. 已知限制

- 客户端看不到逐字输出（MCP 工具结果是一次性返回的），但每 5 秒有进度心跳，UI 上能看到在动。
- 无并发限流 / 无缓存；多 Agent 同时用会在上游排队（实测偶发 10–30 s 才返回、偶发 503，会自动重试）。
- 会话在内存里，进程重启即丢。
- 不支持老式 SSE 传输（pi 也不支持）。
