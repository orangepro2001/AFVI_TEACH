# 原生应用构建指南

本文档说明如何在 WSL2/Linux 环境中交叉编译 Go + Dear ImGui 原生应用到 Windows。

## 项目说明

同一个模块构建两个 GUI 程序与一个命令行相机程序，均使用 Go + Dear ImGui（cimgui-go），GLFW + OpenGL3 后端：

- **teach-imgui.exe**（教学程序）：对齐页与缺陷检出页。
- **camera-gui.exe**（相机合成工作室）：光照与 GV 页、缺陷合成页，camera.exe 的 GUI 前端。
- **camera.exe**（命令行）：相机模拟程序，两个 GUI 都以子进程方式调用它。

## 前置要求

### WSL2/Linux 环境

1. **Go 1.27+**
   ```bash
   go version  # 确认版本
   ```

2. **MinGW-w64 交叉编译工具链**
   ```bash
   sudo apt-get update
   sudo apt-get install -y mingw-w64
   ```

   验证安装：
   ```bash
   x86_64-w64-mingw32-gcc --version
   ```

### Windows 环境（可选，用于运行和测试）

- reference/ 目录包含测试数据（FM1/FM2/BM 机台数据）
- Windows 10+ （应用运行环境）

## 构建步骤

### 构建

```bash
./scripts/build-wsl.sh                    # 基本构建
./scripts/build-wsl.sh --offline          # 离线构建（需要模块缓存）
```

输出：`bin/teach-imgui.exe` (~7 MB)、`bin/camera-gui.exe` 与 `bin/camera.exe`（纯 Go，无 cgo）

构建脚本会用 windres 把图标资源脚本编译成 `cmd/teach-imgui/rsrc_windows_amd64.syso` 与 `cmd/camera-gui/rsrc_windows_amd64.syso`（已被 git 忽略；没有 windres 时 exe 仍能构建，只是使用默认图标）。`go run ./cmd/mkicon` 重新生成 `assets/icons/*.ico`，详见 [ICON.md](ICON.md)。

Windows 原生构建：

```powershell
.\scripts\build.ps1
```

可选：运行配置包与核心算法测试（纯 Go，无需交叉编译）
```bash
go test ./internal/config/... ./internal/core/... ./internal/inspect/... ./internal/model/... ./internal/snap/... ./internal/camera/... ./internal/link/...
```

相机模拟程序是纯 Go，可直接在 Linux 上原生构建与运行：

```bash
CGO_ENABLED=0 go build -o bin/camera ./cmd/camera
bin/camera -ref reference -list
```

## 运行（在 Windows 环境中）

将生成的 `.exe` 文件传输到 Windows 环境，然后：

### 命令行

```powershell
# 基本运行（教学程序）
.\bin\teach-imgui.exe

# 相机合成工作室
.\bin\camera-gui.exe

# 完整参数（两个 GUI 通用，-fontsize 为界面字号 10–28，默认 16）
.\bin\teach-imgui.exe -ref <reference目录> -theme dark -lang zh -model FM1/25fbo015-15 -fontsize 16

# 指定机台配置文件与机台
.\bin\teach-imgui.exe -config afvi.toml -machine AFVI-01

# 实时联机（默认 127.0.0.1:47631，两个程序用同一个地址；off 关闭，详见 LINK.md）
.\bin\camera-gui.exe -link 127.0.0.1:47700
.\bin\teach-imgui.exe -link 127.0.0.1:47700

# 自测模式（两个程序各自运行；加 -link 时教学程序多一个 live 步骤，需要另一个进程里有空闲的工作室）
.\bin\teach-imgui.exe -selftest -out selftest-output
.\bin\camera-gui.exe -selftest -out selftest-output
```

机台配置保存在 `afvi.toml`（示例见 `configs/afvi.example.toml`）：查找顺序为 `-config <文件>` → 环境变量 `AFVI_CONFIG` → exe 同级的 `afvi.toml` → 工作目录的 `afvi.toml`。没有配置文件时，应用按 `-ref` / `AFVI_REFERENCE_DIR` / `reference/` 查找顺序模拟一台 `REFERENCE` 机台。

## 操作说明

### 通用操作
- **滚轮**：以光标为中心缩放
- **Shift + 鼠标左键点击**：放大（×1.5，以光标为中心）；**Shift + 鼠标右键点击**：缩小（÷1.5，以光标为中心）
- **中键/右键拖动**：平移画布
- **双击**：适应窗口（画框工具绘制过程中不会触发）
- **菜单栏**：文件（机台配置…、重新扫描型号、退出）、视图（主题、语言、适应窗口）、帮助（关于…）
- **工具栏**：机台下拉框 + `…` 按钮、型号下拉框、加载状态
- **页签**：切换光照 / 对齐 / 缺陷页

### 页面功能
- **光照页**：左侧滑块调节亮度，左键拖框测量 GV
- **对齐页**：左键新建 ROI/遮挡，选择移动，Delete 删除
- **缺陷页**：画点/线/块/异物/污染，探查缺陷；画框工具（矩形 / 椭圆 / 钢笔，右键闭合多边形，Esc 取消，Backspace 撤销顶点）；不良颜色（按颜色模拟，R/G/B 滑块与七个预设）；虚拟良品（纹理强度 / 种子 / 相关长度倍率，换一块板、恢复默认）与默认不良（约 10 个 SR 内的小亮点，可重新生成 / 移除 / 隐藏轮廓）；选择工具（悬浮面板的点选 / 矩形框 / 椭圆框 / 多边形框）与浮动"选中的不良"窗口批量调参；检出参数按 F1 唤出/关闭浮动窗口，窗口内按相机通道各有一套参数，逐材质（metal / sr / open / base）设置可接受亮度区间 [Low, High]、连续像素数 Size 与最小面积 MinArea，区间由良品样本学习得到，并有全局的边缘忽略（0–64 像素，默认 6）

## 技术特性

### 功能
- 三种语言：中文（微软雅黑）、韩文（Malgun Gothic）、英文（Tahoma）
- 经典浅灰 / 深色主题运行时切换（视图菜单）
- 经典 Win32 外观：3D 立体边框、菜单栏、工具栏、多窗格状态栏
- 机台配置（`afvi.toml`）与机台配置对话框
- 完整的 teach-core 仿真移植（gray、rng、light、warp、align、defect 涂抹与渲染）
- 虚拟良品：加载型号时把确定性的表面不均匀烘焙进光照场景，所有页面都渲染"真实"良品；缺陷页自动加入约 10 个肉眼几乎不可见的默认亮点
- 悬浮工具面板（点选 / 矩形框 / 椭圆框 / 多边形框 / 专注）与专注模式（F11 隐藏两侧面板），缺陷合成页可用它选择并批量调整已画的不良
- 两个 GUI 程序：教学程序（对齐 + 缺陷检出）与相机合成工作室（光照与 GV + 缺陷合成，含「输出」栏，把场景保存成抓拍交给教学程序载入）
- 界面字号 `-fontsize <10-28>`（默认 16），布局常量按字号缩放
- 独立的检出算法包 `internal/inspect`（只看测试图与设计材质图，含全局边缘忽略）
- 配置包 16 个测试 + 核心 73 个测试 + 检出 41 个测试 + 模型 9 个测试 + 抓拍格式 11 个测试 + 相机 19 个测试；渲染、涂抹、RNG 与颗粒仍与 TypeScript 版本跨语言验证一致，检出不再与 TS 版本对照

## 目录结构

```
AFVI_TEACH/
├── go.mod, go.sum    # 模块 afvi/teach-imgui（Go 1.27）
├── cmd/teach-imgui/  # 教学程序入口（main.go）
├── cmd/camera-gui/   # 相机合成工作室入口（main.go）
├── internal/
│   ├── app/          # UI 应用（run.go、app.go、win32.go、dlg_config.go、jobs.go、
│   │                 # viewer.go、shapetool.go、models.go、pack.go、widgets.go、theme.go、i18n.go、
│   │                 # panel_light.go、panel_align.go、panel_defect.go、toolpalette.go、
│   │                 # defect_select.go、defect_product.go、align_view.go、warp_ui.go、inspect_bridge.go、
│   │                 # profile.go、scale.go、studio_capture.go、
│   │                 # live.go、live_studio.go、live_teach.go、teach_save.go、live_selftest.go、
│   │                 # selftest.go、capture_windows.go）
│   ├── core/         # 仿真（teach-core 的 Go 移植，含虚拟良品与笔画几何）与测试
│   ├── inspect/      # 检出算法：只看测试图与设计材质图，含单元测试与隔离测试
│   ├── config/       # 机台配置包（afvi.toml，纯 Go）与测试
│   ├── model/        # 模型扫描与加载（纯 Go，含 modeltest 合成参考树）
│   ├── snap/         # 抓拍文件格式（snap.json + 灰度 PNG，仅标准库与 core）
│   ├── link/         # 实时联机契约（协议、Server、Client；仅标准库与 snap）
│   └── camera/       # 相机渲染与命令行（纯 Go，无 cgo）
├── configs/          # 机台配置示例（afvi.example.toml）
├── scripts/          # build-wsl.sh（WSL2）、build.ps1（Windows）
├── docs/             # BUILD.md、USAGE.md、LINK.md、REFERENCE_SETUP.md
├── reference/        # 机台测试数据（需单独配置）
└── bin/              # 构建输出（bin/teach-imgui.exe）
```

## 故障排除

### CGO 编译失败
确保 MinGW-w64 正确安装：
```bash
which x86_64-w64-mingw32-gcc
x86_64-w64-mingw32-gcc --version
```

### 首次编译很慢
首次编译需要构建 cimgui-go 的 C++ 包装层，约 2-3 分钟。后续增量编译约 2 秒。

### 模块下载失败
```bash
# 确保网络连接，或使用代理
export GOPROXY=https://goproxy.cn,direct

# 缓存就绪后可离线构建
./scripts/build-wsl.sh --offline
```

### Windows 运行时缺少 DLL
应用已静态链接 libstdc++、libgcc、winpthread，只依赖系统 DLL（Win10+ 自带 UCRT）。如果仍有问题，检查 Windows 更新。

## 已知限制

- 纹理固定线性过滤，高倍放大时像素会被插值
- 60 fps 持续重绘（空闲约占单核 3.4% CPU）
- 字体取自 `C:\Windows\Fonts`，仅支持 Windows
- 未适配高 DPI 缩放
- 检出是教学用的简化亮度区间算法（Metal / SR 各一个区间；真实产品的 Space、Open 等分层与 Margine、Offset 等机台参数未模拟），成像噪声也是近似
- 检出的边缘忽略（Margine）是单一全局像素数；虚拟良品是统计意义上的非均匀（多尺度噪声 + 颗粒），不模拟真实的划痕、氧化、批次差异；默认不良只放在 SR 内部亮点；选择工具只选 GV 笔画（异物/污染笔画不可选）

## 开发建议

如果要修改代码：

1. **纯算法修改**：仿真改 `internal/core/`（含虚拟良品 `product.go` 与笔画几何 `strokegeom.go`），检出改 `internal/inspect/`（检出算法只依赖标准库，只看测试图与设计材质图），相机改 `internal/camera/` 与 `internal/snap/`，联机协议改 `internal/link/`（同时更新 `docs/LINK.md`），运行 `go test ./internal/config/... ./internal/core/... ./internal/inspect/... ./internal/model/... ./internal/snap/... ./internal/camera/... ./internal/link/...`（配置包 16 个测试 + 核心 73 个测试 + 检出 41 个测试 + 模型 9 个测试 + 抓拍格式 11 个测试 + 相机 20 个测试 + 联机 15 个测试）
2. **UI 修改**：修改对应的 `internal/app/panel_*.go` 或 `internal/app/viewer.go`
3. **构建验证**：`./scripts/build-wsl.sh`，传输到 Windows 运行
4. **自测验证**：在 Windows 运行 `.\bin\teach-imgui.exe -selftest -out <dir>` 与 `.\bin\camera-gui.exe -selftest -out <dir>`（步骤按程序过滤，两者都需要 `camera.exe` 放在一起）

## 相关文件

- `docs/USAGE.md`：完整功能说明
- `docs/REFERENCE_SETUP.md`：机台配置与 reference 目录设置说明
- `scripts/build.ps1`：Windows 原生构建脚本
- `scripts/build-wsl.sh`：WSL2 构建脚本
- `configs/afvi.example.toml`：机台配置示例
