# UI 迁移：ImGui → Wails + Vue 3 (win32-theme-for-wails)

> 状态：阶段 0–5 已完成（见文末"完成情况"）。

范围：只换 UI（`internal/app` 与 `cmd/teach-imgui`、`cmd/camera-gui`），算法包
（`core`/`inspect`/`golden`/`snap`/`link`/`model`/`camera`/`access`/`config`）一行不动，
视觉保持现有 Win32 Classic 皮肤。组件库：本机 `/home/tony/dev/win32-theme-for-wails`
（`@win32-vue/win32`，Vue 3 + TS，Wails v2）。

## 现状（已量过）

- `internal/app` 约 3 万行，其中 6.7k 行是测试；96 个文件。
- 34 个文件完全不含 imgui（golden_board、live_*、studio_scenario、teach_save、align_flow、
  snap_*、jobs、i18n 等），**可原样复用**。
- 32 个文件含 imgui，但面板只使用约 25 个控件封装（`widgets.go`：Section/Slider*/Check/Combo/
  Button/GroupBox/TabStrip/DataTable/KVTable/Histogram/Heatmap…）加约 500 处裸 imgui 调用。
- `Overlay`（查看器叠加层）已经是"图像坐标 → 绘制指令"的数据形态，可以改成输出结构化列表。
- 难点：状态与绘制交织在 `DrawLeft/DrawRight` 里；selftest（2.8k 行）依赖 ImGui 窗口与截图。

## 架构

```
cmd/teach-wails  cmd/studio-wails          (Go, 无 cgo，Wails 窗口入口)
        │                │
        └── internal/shell ─ 控件树记录器 + 命令分发 + 图像服务 + 事件推送
                 │
        internal/app (去 imgui 后无 cgo)   ← 面板逻辑原样保留
                 │
   core / inspect / golden / snap / link / model / camera
frontend/ (Vue 3 + @win32-vue/win32)       ← 唯一渲染端，teach 与 studio 共用，按 profile 切换
```

1. **控件树记录器（server-driven UI）**。`widgets.go` 的 25 个控件封装改为向"帧树"追加节点
   （`SameLine` → 行内标记，`GroupBox/Section/TabStrip` → 容器节点，`Slider/Check/Combo/Input`
   → 带 id 的值节点）。Vue 端用 `WGroupBox/WTrackbar/WCheckBox/WComboBox/WEdit/WButton/
   WListView…` 渲染该树，用户操作以 `{id, value}` 命令回传，Go 在下一帧把值写回面板状态。
   这样 3 万行面板逻辑不重写，只替换控件层；状态仍只在 Go 里，保持 `Jobs` 线程模型。
2. **查看器**。Go 侧只管图像与叠加数据；缩放/平移/拾取在前端 canvas 完成（滚轮 1.2/档、
   Shift 点击缩放、中右键平移、右键点击上报，沿用 viewer.go 的输入契约），图像经 Wails
   `AssetServer.Handler` 以 `/img/<texture-id>.png` 提供（大图 5860×4020 灰度，PNG 编码在
   `Jobs` 里完成），叠加层以 JSON 绘制指令下发。
3. **窗口**。Wails `Frameless` + `WTitleBar/WMenuBar/WToolBar/WTabBar/WStatusBar`；对话框
   （登录、F1 检测参数、配置）用 `WDialog`。
4. **不用 Wails CLI 绑定生成**：只暴露一个 `Dispatch(cmd, json)` 与事件 `frame`/`image`，免去
   wailsjs 生成；前端用 vite 构建，`go:embed` 包 `frontend/dist`，Windows 交叉编译无需 cgo、
   无需静态链接 MinGW（WebView2 运行时是前置条件，安装器需检测）。
5. **测试**。后端：沿用 headless 测试（去 cgo 后可在 Linux 原生跑，`go test ./internal/app`）。
   前端：vitest + Playwright 以录制的帧树 JSON 做快照，替代 ImGui selftest 的截图；
   Windows 上 `-selftest` 改为用 WebView2 打开真实窗口截图。

## 阶段

| # | 内容 | 验收 |
|---|------|------|
| 0 | 前端骨架：共享 shell（标题栏/菜单/标签/状态栏）、主题 token 对齐现有调色板、帧树渲染器、查看器 canvas；Go `internal/shell` 帧树与命令回路 | Chrome 里用假帧树渲染出与现有截图一致的页面 |
| 1 | 控件层替换：`widgets.go`/`win32.go`/`theme.go`/`viewer.go`/`toolpalette.go` 改为记录器；`app.go` 去 GLFW 主循环，改 Wails 驱动的帧循环 | `internal/app` 无 `cimgui-go` import，Linux 可编译 |
| 2 | camera studio：光照/GV、缺陷合成（笔刷、画框、选择面板）、Output/Scenario | studio 全部标签可操作，`camera.exe` 保存字节一致测试通过 |
| 3 | teach：操作员三页（对位/金板/检测）→ 管理员页（ROI、对位试验台、缺陷检出含 UserMask、Metal\|SR 子标签、F1 参数窗口） | 现有 headless 测试 + 新前端快照全绿 |
| 4 | 登录/角色、实时链路、Load Image 文件对话框、配置对话框、i18n 三语 | 双程序联调（studio 发布 → teach 跟随） |
| 5 | 打包：`build-wsl.sh`/`package-wsl.sh` 改为 vite build + 无 cgo 构建；安装器 payload 与 WebView2 检测；删除 cimgui-go 依赖与 selftest 旧实现 | 一键构建出 installer，Windows 实机截图对比 |

## 风险与取舍

- 帧树方案对前端是"薄渲染端"，不是惯用的 Vue 状态管理；换来的是业务行为零回归和约 5 倍
  少的工作量。若以后要更原生的 Vue 体验，可逐页把记录器节点替换为视图模型。
- 帧率：面板控件树小（<500 节点），按变更推送（哈希去重），查看器图像与叠加分通道，滑块拖动
  只发最新值（沿用 `Jobs` 的合并语义）。
- WebView2 默认缩放/字体与 ImGui 度量不同：用 `-fontsize` 对应根字号，所有 px 长度走 CSS 变量，
  替代 `sc()`。
- 本机目前无 `node` 在 PATH（`~/.local/node-v24.22.1-linux-x64/bin`）与前端依赖；安装依赖前需
  用户同意。

## 完成情况

- `internal/ui`、`internal/host`、`frontend/` 与无 cgo 的 `internal/app` 已落地；`internal/app` 全部 Go 测试在 Linux 原生通过。
- 前端：vue-tsc、vitest、Playwright（录制帧 + 真实 Go 后端 `-web` 模式，studio 与 teach 各自的主流程）通过。
- 构建：`scripts/build-wsl.sh` 不再需要 MinGW 运行时静态链接；安装器沿用。
- 未验证：Windows 上的 WebView2 窗口（本机为 Linux，无 Windows 互操作），首次安装后需手动冒烟。
- 取舍：x/image 随 `go.mod` 降到 v0.41.0（离线缓存无 v0.47 所需的 x/text v0.43）。
