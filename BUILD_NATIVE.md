# 原生应用构建指南

本文档说明如何在 WSL2/Linux 环境中交叉编译 Go + Dear ImGui 原生应用到 Windows。

## 项目说明

- **teach-imgui-spike**：可行性验证原型，功能简化，适合快速验证
- **teach-imgui**：完整的原生实现，包含光照、对齐、缺陷三大功能页

两者都是 Go + Dear ImGui（cimgui-go）实现，使用 GLFW + OpenGL3 后端。

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

### 选项 1：构建 teach-imgui-spike（快速验证）

```bash
cd teach-imgui-spike
./build-wsl.sh
```

输出：`teach-imgui-spike.exe` (~13 MB)

### 选项 2：构建 teach-imgui（完整功能）

```bash
cd teach-imgui
./build-wsl.sh                    # 基本构建
./build-wsl.sh --offline          # 离线构建（需要模块缓存）
```

输出：`teach-imgui.exe` (~7 MB)

可选：运行配置包与核心算法测试（纯 Go，无需交叉编译）
```bash
cd teach-imgui
go test ./config/... ./core/...
```

## 运行（在 Windows 环境中）

将生成的 `.exe` 文件传输到 Windows 环境，然后：

### teach-imgui-spike

```powershell
# 默认使用 ..\reference\FM1\PxRepository\25fbo015-15
.\teach-imgui-spike.exe

# 指定 reference 目录
.\teach-imgui-spike.exe -ref C:\path\to\reference

# 自测模式
.\teach-imgui-spike.exe -selftest -out selftest-output

# 定时退出（测试用）
.\teach-imgui-spike.exe -exitafter 12s
```

### teach-imgui

```powershell
# 基本运行
.\teach-imgui.exe

# 完整参数
.\teach-imgui.exe -ref <reference目录> -theme dark -lang zh -model FM1/25fbo015-15

# 指定机台配置文件与机台
.\teach-imgui.exe -config afvi.toml -machine AFVI-01

# 自测模式
.\teach-imgui.exe -selftest -out selftest-output
```

机台配置保存在 `afvi.toml`（示例见 `teach-imgui/afvi.example.toml`）：查找顺序为 `-config <文件>` → 环境变量 `AFVI_CONFIG` → exe 同级的 `afvi.toml` → 工作目录的 `afvi.toml`。没有配置文件时，应用按 `-ref` / `AFVI_REFERENCE_DIR` / `reference/` 查找顺序模拟一台 `REFERENCE` 机台。

## 操作说明

### 通用操作
- **滚轮**：以光标为中心缩放
- **中键/右键拖动**：平移画布
- **双击**：适应窗口
- **菜单栏**：文件（机台配置…、重新扫描型号、退出）、视图（主题、语言、适应窗口）、帮助（关于…）
- **工具栏**：机台下拉框 + `…` 按钮、型号下拉框、加载状态
- **页签**：切换光照 / 对齐 / 缺陷页

### teach-imgui 页面功能
- **光照页**：左侧滑块调节亮度，左键拖框测量 GV
- **对齐页**：左键新建 ROI/遮挡，选择移动，Delete 删除
- **缺陷页**：画点/线/块/异物/污染，探查缺陷

## 技术特性

### teach-imgui-spike 性能（参考数据）
- 加载 4 张 TIFF：171-186 ms
- 调光全分辨率重算：6.4-8.6 ms
- 空闲渲染：~1380 fps
- 1 万个 ROI 框：~610 fps

### teach-imgui 功能
- 三种语言：中文（微软雅黑）、韩文（Malgun Gothic）、英文（Tahoma）
- 经典浅灰 / 深色主题运行时切换（视图菜单）
- 经典 Win32 外观：3D 立体边框、菜单栏、工具栏、多窗格状态栏
- 机台配置（`afvi.toml`）与机台配置对话框
- 完整的 teach-core 算法移植（gray、rng、light、warp、align、defect）
- 配置包测试 + 58 个单元测试，与 TypeScript 版本跨语言验证一致

## 目录结构

```
teach-imgui/
├── core/           # teach-core 的 Go 移植
├── config/         # 机台配置包（afvi.toml，纯 Go）
├── main.go         # 入口、字体、窗口
├── app.go          # Panel 接口、共享状态、菜单栏 / 工具栏 / 页签 / 状态栏
├── win32.go        # Win32 立体边框原语
├── dlg_config.go   # 机台配置对话框
├── afvi.example.toml # 机台配置示例
├── jobs.go         # 后台任务管理
├── viewer.go       # 图像视图与叠加
├── models.go       # 型号扫描与加载
├── panel_*.go      # 各功能页面
├── theme.go        # 主题
├── i18n.go         # 国际化
└── build-wsl.sh    # WSL2 构建脚本
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
./build-wsl.sh --offline
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

1. **纯算法修改**：修改 `teach-imgui/core/`，运行 `go test ./config/... ./core/...`
2. **UI 修改**：修改对应的 `panel_*.go` 或 `viewer.go`
3. **构建验证**：`./build-wsl.sh`，传输到 Windows 运行
4. **自测验证**：在 Windows 运行 `.\teach-imgui.exe -selftest -out <dir>`

## 相关文件

- `teach-imgui/README.md`：完整功能说明
- `teach-imgui-spike/README.md`：原型验证说明
- `teach-imgui/build.ps1`：Windows 原生构建脚本
- 本文件的 WSL2 构建脚本：
  - `teach-imgui/build-wsl.sh`
  - `teach-imgui-spike/build-wsl.sh`
