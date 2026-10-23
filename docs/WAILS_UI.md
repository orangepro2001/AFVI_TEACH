# Wails + Vue UI（分支 `wails-ui`）技术文档

本文档描述从 Dear ImGui 迁移到 **Wails (WebView2) + Vue 3 + win32-vue** 之后的 UI 架构、协议、开发/测试/构建流程、
性能设计与已知限制。迁移过程与阶段见 [WAILS_MIGRATION.md](WAILS_MIGRATION.md)；仓库级说明见 `CLAUDE.md` 的 "UI stack"。

## 1. 目标与边界

- 只换 UI：`internal/{core,inspect,golden,snap,link,model,camera,access,config,installer}` 与 `cmd/camera`、安装器逻辑未改。
- 视觉保持经典 Win32（win32-vue 令牌与旧 ImGui 调色板一致，含暗色主题）。
- 两个程序都迁移：`cmd/teach-imgui`（教学程序）与 `cmd/camera-gui`（相机工作室），产物名不变。
- `internal/app` 不再含 cgo：全部 Go 测试可在 Linux 原生运行；Windows exe 由纯 Go 交叉编译。

## 2. 架构

```
Vue 3 前端 (frontend/)  ──HTTP──  internal/host  ──  internal/app (面板/状态/任务)  ──  core/inspect/golden/...
  win32-vue 组件                  帧循环+协议          Panel 记录控件树 (internal/ui)
  canvas 查看器
```

### 2.1 控件树（internal/ui）
面板保持立即模式写法，但不绘制，而是向 `ui.Builder` 记录节点：

- `ui.Button / SliderF64 / SliderInt / Check / Combo / InputText / InputNum / TabStrip / SubTabStrip / DataTable / KVTable /
  Section / GroupBox / Dialog / Menu / Histogram / Badge / Swatch ...`，签名与旧 `widgets.go` 兼容（`internal/app/widgets.go` 是薄封装）。
- `SameLine()` 把相邻节点并成一行（`rowify`）；`Section` 之后的节点自动成为其内容。
- 用户操作以 `Cmd{id, op, v}` 回传：`set`（值）、`click`、`toggle`（折叠）、`key`、`ptr`（指针）、`view`、`hover`。控件在下一帧构建时消费对应命令，
  返回 "changed"，与 ImGui 的语义一致。
- `ui.Viewer`：持有图像与叠加形状（`Overlay.Rect/FillRect/DashedRect/Line/Circle/Poly/Text`，图像坐标，颜色仍为 0xAABBGGRR）。
  缩放/平移由前端持有（避免往返），前端上报 `view` 供 Go 做裁剪判断；`Viewer.Claim` 表示左键拖动归面板（否则平移）。
- 扩展：新增节点种类 = Go 侧 `ui.Add(&ui.Node{K:"x"})` + `frontend/src/render/kinds/X.vue`（文件名小写即种类）。

### 2.2 Host（internal/host）
单 goroutine 帧循环：每 33 ms 或收到命令立即执行 `Program.Build(b)`，序列化成 JSON，**内容变化才发布**。HTTP 接口：

| 路径 | 说明 |
|---|---|
| `POST /cmd` | 命令数组（JSON） |
| `GET /poll?seq=N` | 长轮询，返回比 N 新的整帧 JSON（20 s 超时 204） |
| `GET /img?lvl=k` | 当前查看器图像第 k 级（见 §4），头 `X-W/X-H/X-Fmt(gray\|rgba)/X-Rev` |
| `GET /stats` | 最近一帧构建/序列化耗时、帧字节数、请求计数（排查性能） |

同一个 `Handler` 既作为 Wails `AssetServer.Handler`（窗口）又用于 `-web ADDR`（浏览器/测试模式，任意 OS）。

### 2.3 App（internal/app）
- `App.Build` 产出槽位：`menu / toolbar / tabs / sub / left / right / palette / message / status / dialogs`，外加 `meta`
  （profile、theme、lang、fontPx、focusView、quit、role）。
- 面板接口保持：`DrawLeft/DrawRight/Overlay/Pointer/Hover/...`；新增 `ClaimsDrag() bool`（每帧读取，镜像旧 "Pointer 在 PointerDown 返回 true"）。
  未被认领的左键按下仍会作为 `PointerDown` 发给面板（点选），但不会收到 Move/Up。
- 模态窗口 = 面板状态标志 + `ui.Dialog`；快捷键 = `ui.KeyPressed("F5")`（名称：F1、F5、F11、Escape、Backspace、Delete、Enter、Ctrl+Z）。
- 角色/登录、实时联机（link）、场景文件、金板、对位流程等业务逻辑原样保留。

### 2.4 前端（frontend/）
- `App.vue` 外壳：`WTitleBar`（无边框窗口）、`WMenuBar`、页签、左/中/右栏、`WStatusBar`、对话框层；`meta` 控制主题、字号（`zoom = fontPx/12`）、专注模式、退出。
- `render/NodeView.vue`：节点种类到 win32-vue 组件的映射；`render/Viewer.vue`：canvas 查看器（滚轮 1.2×、Shift+左/右键 1.5× 缩放、中/右键平移、双击适应、
  右键点击上报 `rclick`）；`render/Palette.vue`：浮动工具面板。
- `bridge.ts`：命令批量发送（见 §4）、长轮询、`local` 乐观值（滑块拖动不被旧帧回弹）。
- win32-vue 以 `frontend/vendor/win32-vue` 形式 vendored（来源 `../win32-theme-for-wails`，提交号见 `SOURCE_COMMIT`）；`frontend/dist` 已提交并由
  `frontend/embed.go` 嵌入，所以没有 node 的构建服务器也能构建。

## 3. 开发、测试、构建

```bash
# 前端
cd frontend && npx vue-tsc --noEmit && npx vitest run && npx vite build      # node 在 ~/.local/node-v24.22.1-linux-x64/bin
# Go（纯 Go，Linux 原生）
CGO_ENABLED=0 go vet ./... && CGO_ENABLED=0 go test ./...
# 在浏览器里跑真实程序（任意 OS，无窗口）
go run ./cmd/camera-gui -ref reference -web 127.0.0.1:47701 -link off -login admin:1
go run ./cmd/teach-imgui -ref reference -web 127.0.0.1:47702 -link off
# UI 端到端（Playwright + Chrome，对上面两个真实程序 + 录制帧）
./scripts/e2e-wsl.sh
# 一键构建（前端 → gofmt → vet → 全部测试 → 交叉编译 → 安装包）
./scripts/build-wsl.sh --version X
```

- Linux 交叉编译 Windows：`CGO_ENABLED=0 GOOS=windows go build -tags production -ldflags "-s -w -H windowsgui"`。不需要 MinGW 运行时静态链接；
  `windres` 仅用于图标（`3 ICON` 是 Wails 窗口图标）。
- 目标 Windows 需要 WebView2 运行时（Win11 自带，Win10 需安装）。
- 测试分层：`internal/ui`（控件语义）、`internal/host`（金字塔）、`internal/app`（面板逻辑 + `TestBuildFrames` 整帧）、
  `frontend/e2e/mock.spec.ts`（录制帧：外观、无圆角、命令协议）、`frontend/e2e/app.spec.ts`（真实后端：加载大图、滑块、画缺陷、登录/注销、对位）。
- 性能脚本：`node frontend/e2e/perf.mjs [url]`（滑块往返延迟、图像传输、平移/缩放帧间隔）。

## 4. 性能设计（实测驱动）

最初版本每次图像变化都传 94 MB RGBA 整图，滑块往返 180–500 ms。现在：

- **图像金字塔**（`host/pyramid.go`）：按需生成 1/2^k（k=0..5）各级，全灰图转单字节；前端选"刚好够用的最粗一级"并只绘制可见区域；
  新图先取小预览级，稳定 250 ms 后补高清级；新版本到达时中止旧请求。
- **命令合并**：`set/view/hover/ptr move` 每帧最多一批，点击/按键在当前任务末尾发出；`view` 上报 60 ms 节流。
- **帧去重**：JSON 不变不发布；空闲时 Build ≈ 0.3 ms。
- 实测（Linux 无头 Chrome，5860×4020）：拖滑块往返中位 16.7 ms（原 182 ms），平移 p95 20 ms（原 98 ms），单次图像 0.4 MB/≈100 ms（原 94 MB/2–3 s）。

## 5. 已知限制与后续

- **Windows 窗口（WebView2）未在 Windows 上实机验证**：HTTP 经 Wails AssetServer 的 POST/长轮询依据源码判断可行；若发现延迟，备选是改走 Wails 事件通道。
- `x/image` 因离线缓存限制降到 v0.41.0（v0.47 需要 x/text v0.43）。
- 原 ImGui `-selftest`（截图/鼠标注入）已删除，由 Playwright 取代；Windows 端像素级截图回归暂无。
- 选择编辑器由浮动窗口改为右栏内嵌（模态对话框会挡画布点击）。
- `scripts/build.ps1`/`package.ps1` 仅做了去 cgo 的盲改，未在 Windows 上运行。
- 级别 0（≥50% 缩放）整图仍一次取回（灰度 23 MB、彩色 94 MB）；如需更省可改为按可见区域裁剪的瓦片。

## 6. 与旧 ImGui 版的关系
旧实现保留在 `main` 分支（提交 `be8666d` 及之前）。本分支 `wails-ui` 删除了 `internal/app` 中全部 ImGui 代码与 cimgui-go 依赖。
