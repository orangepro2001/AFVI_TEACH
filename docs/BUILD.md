# 原生应用构建指南

本文档说明如何在 WSL2/Linux 环境中交叉编译 Go + Dear ImGui 原生应用到 Windows。

## 项目说明

**teach-imgui** 是完整的原生实现，使用 Go + Dear ImGui（cimgui-go），GLFW + OpenGL3 后端，包含光照、对齐、缺陷三大功能页。

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

输出：`bin/teach-imgui.exe` (~7 MB)

Windows 原生构建：

```powershell
.\scripts\build.ps1
```

可选：运行配置包与核心算法测试（纯 Go，无需交叉编译）
```bash
go test ./internal/config/... ./internal/core/...
```

## 运行（在 Windows 环境中）

将生成的 `.exe` 文件传输到 Windows 环境，然后：

### 命令行

```powershell
# 基本运行
.\bin\teach-imgui.exe

# 完整参数
.\bin\teach-imgui.exe -ref <reference目录> -theme dark -lang zh -model FM1/25fbo015-15

# 指定机台配置文件与机台
.\bin\teach-imgui.exe -config afvi.toml -machine AFVI-01

# 自测模式
.\bin\teach-imgui.exe -selftest -out selftest-output
```

机台配置保存在 `afvi.toml`（示例见 `configs/afvi.example.toml`）：查找顺序为 `-config <文件>` → 环境变量 `AFVI_CONFIG` → exe 同级的 `afvi.toml` → 工作目录的 `afvi.toml`。没有配置文件时，应用按 `-ref` / `AFVI_REFERENCE_DIR` / `reference/` 查找顺序模拟一台 `REFERENCE` 机台。

## 操作说明

### 通用操作
- **滚轮**：以光标为中心缩放
- **中键/右键拖动**：平移画布
- **双击**：适应窗口
- **菜单栏**：文件（机台配置…、重新扫描型号、退出）、视图（主题、语言、适应窗口）、帮助（关于…）
- **工具栏**：机台下拉框 + `…` 按钮、型号下拉框、加载状态
- **页签**：切换光照 / 对齐 / 缺陷页

### 页面功能
- **光照页**：左侧滑块调节亮度，左键拖框测量 GV
- **对齐页**：左键新建 ROI/遮挡，选择移动，Delete 删除
- **缺陷页**：画点/线/块/异物/污染，探查缺陷

## 技术特性

### 功能
- 三种语言：中文（微软雅黑）、韩文（Malgun Gothic）、英文（Tahoma）
- 经典浅灰 / 深色主题运行时切换（视图菜单）
- 经典 Win32 外观：3D 立体边框、菜单栏、工具栏、多窗格状态栏
- 机台配置（`afvi.toml`）与机台配置对话框
- 完整的 teach-core 算法移植（gray、rng、light、warp、align、defect）
- 配置包测试 + 58 个单元测试，与 TypeScript 版本跨语言验证一致

## 目录结构

```
AFVI_TEACH/
├── go.mod, go.sum    # 模块 afvi/teach-imgui（Go 1.27）
├── cmd/teach-imgui/  # 程序入口（main.go）
├── internal/
│   ├── app/          # UI 应用（run.go、app.go、win32.go、dlg_config.go、jobs.go、
│   │                 # viewer.go、models.go、pack.go、widgets.go、theme.go、i18n.go、
│   │                 # panel_light.go、panel_align.go、panel_defect.go、
│   │                 # selftest.go、capture_windows.go）
│   ├── core/         # teach-core 的 Go 移植与测试
│   └── config/       # 机台配置包（afvi.toml，纯 Go）与测试
├── configs/          # 机台配置示例（afvi.example.toml）
├── scripts/          # build-wsl.sh（WSL2）、build.ps1（Windows）
├── docs/             # BUILD.md、USAGE.md、REFERENCE_SETUP.md
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
- 教学近似的阈值模型，部分机台参数未模拟

## 开发建议

如果要修改代码：

1. **纯算法修改**：修改 `internal/core/`，运行 `go test ./internal/config/... ./internal/core/...`
2. **UI 修改**：修改对应的 `internal/app/panel_*.go` 或 `internal/app/viewer.go`
3. **构建验证**：`./scripts/build-wsl.sh`，传输到 Windows 运行
4. **自测验证**：在 Windows 运行 `.\bin\teach-imgui.exe -selftest -out <dir>`

## 相关文件

- `docs/USAGE.md`：完整功能说明
- `docs/REFERENCE_SETUP.md`：机台配置与 reference 目录设置说明
- `scripts/build.ps1`：Windows 原生构建脚本
- `scripts/build-wsl.sh`：WSL2 构建脚本
- `configs/afvi.example.toml`：机台配置示例
