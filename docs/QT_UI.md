# Qt 前端（分支 `qt-ui`，基于 `wails-ui`）

第三种前端：用 **原生 Qt 6 Widgets**（Go 绑定 `github.com/mappu/miqt`）渲染同一棵控件树。面板逻辑、`internal/ui`、
`internal/host` 完全复用；`internal/qtui` 在进程内直接读 `host.Latest()` / `host.Image()`，用 `host.Do()` 回传命令，不经过 HTTP。

## 结构
- `internal/qtui/build.go`：节点种类 → 原生控件（QSlider/QDoubleSpinBox/QComboBox/QTabBar/QTableWidget/QGroupBox…）。
  结构签名不变时只就地刷新数值，变化时才重建该区域（左/右栏、工具栏、标签、对话框各自独立）。
- `viewer.go`：QWidget 画布，沿用 Vue 版的交互契约（滚轮 1.2×、Shift+点击缩放、中/右键平移、`claim`、右键点击上报）与图像金字塔分级取图，叠加层用 QPainter 绘制。
- `window.go`：QMainWindow、菜单、浮动工具面板、状态栏、模态对话框、Fusion 样式 + 经典灰 Win32 调色板（含暗色）。
- `script.go`：脚本驱动测试（`AFVI_QT_SCRIPT=*.json`）：`cmd`（走 host 命令）、`click`（按下真实 Qt 按钮）、`text`（断言屏幕上有该文字）、`shot`（保存窗口 PNG）。
- `cmd/teach-qt`、`cmd/studio-qt`：入口（`app.WindowRunner` 钩子，`internal/app` 仍然无 cgo）。

## 验证（Linux，`QT_QPA_PLATFORM=offscreen`，无需显示器）
`./scripts/qt-test.sh [输出目录]` 构建并运行 `qt-tests/studio.json`、`qt-tests/teach.json`：
- studio：加载 5860×4020 真实模型、改增益/材料图、切到缺陷合成、点击真实"亮点"按钮、在画布上点一下，断言缺陷列表新增一条；
- teach：操作员对位页、点击"确定对位"、金板页、检测页、菜单登录管理员，断言出现"对位试验台"页；每步截图。
结果：全部断言通过（退出码 0），截图在输出目录。

## 工作量与体积
- miqt 首次编译约 4 分 14 秒（12 核，之后缓存），之后改动后增量构建约 3 秒；
- Linux 动态链接、`-ldflags "-s -w"` 后 teach-qt 约 37 MB，依赖 4 个 Qt6 共享库（Core/Gui/Widgets/DBus 等）。

## 与 Windows 交叉编译
- 本机没有 Qt for MinGW。miqt 提供 `docker/win64-cross-*-qt6-*` 镜像（`miqt-docker win64-qt6-static -windows-build`），
  理论上可在 Linux 上交叉出 Windows exe，但需要拉取/构建很大的镜像，**本次未尝试**；
- 动态发布要带 Qt DLL 与平台插件（几十 MB，目录式安装）；静态链接需遵守 Qt 的 LGPL（用户可替换库）或购买商业许可；
- 不再需要 WebView2，但换来的是 Qt 运行时 / 许可证的负担。

## 已知差距（相对 Vue/Wails 版）
- 外观是 Fusion + 灰色调色板，不是逐像素的 Win32 经典斜面；
- 一行内控件不自动换行（行内用 QHBoxLayout）；
- 浮动工具面板用字符图标；表格高度按行数估算；
- 图像级别在 UI 线程里复制（缩放到 100% 以上的整图级别在拖滑块时可能略卡）；
- Windows 上未验证。
