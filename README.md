# AFVI Color Master Teach - 原生应用版本

Go + Dear ImGui 原生实现的 AFVI 彩色检查机教学桌面软件。

## 项目概述

本项目是 Color AFVI Master Teach 的原生 Windows 应用实现，使用 Go 语言和 Dear ImGui 构建，取代了基于 WebView2 的方案。包含两个版本：

- **teach-imgui**：完整功能版本，包含光照、对齐、缺陷三大功能页面
- **teach-imgui-spike**：可行性验证原型，功能简化，适合快速验证

## 快速开始

### 前置要求

**开发环境（WSL2/Linux）：**
- Go 1.27+
- MinGW-w64 交叉编译工具链

**运行环境（Windows）：**
- Windows 10+
- reference 目录（包含机台测试数据）

### 安装依赖

```bash
# 安装 MinGW-w64
sudo apt-get update
sudo apt-get install -y mingw-w64

# 验证安装
x86_64-w64-mingw32-gcc --version
```

### 构建

```bash
# 构建 teach-imgui（推荐）
cd teach-imgui
./build-wsl.sh

# 或构建 teach-imgui-spike（原型）
cd teach-imgui-spike
./build-wsl.sh
```

详细构建说明见 [BUILD_NATIVE.md](BUILD_NATIVE.md)。

### 运行（Windows 环境）

```powershell
# 基本运行
.\teach-imgui.exe

# 指定参数
.\teach-imgui.exe -ref C:\path\to\reference -theme dark -lang zh

# 指定机台配置文件
.\teach-imgui.exe -config afvi.toml -machine AFVI-01

# 启动时打开特定型号
.\teach-imgui.exe -model FM1/25fbo015-15

# 自测模式
.\teach-imgui.exe -selftest -out selftest-output
```

界面采用经典 Win32（"Windows Classic"）外观：3D 立体边框、灰色面板、深蓝渐变对话框标题栏、属性表风格页签，以及菜单栏、工具栏和多窗格状态栏。默认主题为经典浅灰（`light`），深色主题（`dark`）沿用同一套立体边框方案。机台信息（名称、地址、FM1/FM2/BM 数据根目录）保存在 `afvi.toml` 中，可通过「文件 ▸ 机台配置…」或工具栏的 `…` 按钮编辑。

## 项目结构

```
AFVI_TEACH/
├── teach-imgui/              # 完整功能版本
│   ├── core/                 # 核心算法（Go 移植）
│   ├── config/               # 机台配置包（afvi.toml，纯 Go）
│   ├── main.go               # 入口
│   ├── app.go                # 应用框架（菜单栏 / 工具栏 / 页签 / 状态栏）
│   ├── win32.go              # Win32 立体边框原语
│   ├── dlg_config.go         # 机台配置对话框
│   ├── afvi.example.toml     # 机台配置示例
│   ├── panel_*.go            # 功能页面
│   ├── build-wsl.sh          # WSL2 构建脚本
│   ├── build.ps1             # Windows 构建脚本
│   └── README.md             # 详细说明
│
├── teach-imgui-spike/        # 可行性验证原型
│   ├── main.go               # 主程序
│   ├── build-wsl.sh          # WSL2 构建脚本
│   └── README.md             # 说明
│
├── reference/                # 机台测试数据（需单独配置）
│   ├── FM1/
│   ├── FM2/
│   └── BM/
│
├── docs/                     # 文档
│   └── REFERENCE_SETUP.md    # reference 目录设置说明
│
├── BUILD_NATIVE.md           # 完整构建指南
└── README.md                 # 本文件
```

## 功能特性

### teach-imgui（完整版）

- **光照页面**：亮度调节、GV 测量
- **对齐页面**：ROI 管理、图像对齐、遮挡处理
- **缺陷页面**：缺陷标注（点/线/块/异物/污染）、缺陷检测
- **多语言支持**：中文、英文、韩文
- **主题切换**：经典浅灰 / 深色主题运行时切换（视图菜单）
- **经典 Win32 外观**：3D 立体边框、菜单栏、工具栏、多窗格状态栏
- **机台配置**：`afvi.toml` 记录每台机台的名称、地址与 FM1/FM2/BM 数据根目录，可在「文件 ▸ 机台配置…」对话框中增删改并检查路径
- **性能优化**：
  - 加载型号：~300 ms
  - 光照渲染：~29 ms
  - 对齐计算：~150 ms
  - UI 帧率：60 fps

### teach-imgui-spike（原型）

- 基础图像显示
- 亮度调节
- 缩放、平移操作
- 性能测试功能

## 操作说明

### 通用操作

- **滚轮**：缩放（以光标为中心）
- **中键/右键拖动**：平移画布
- **双击**：适应窗口
- **菜单栏**：文件（机台配置…、重新扫描型号、退出）、视图（主题、语言、适应窗口）、帮助（关于…）
- **工具栏**：机台下拉框 + `…` 按钮、型号下拉框、加载状态
- **页签**：切换光照 / 对齐 / 缺陷页

### teach-imgui 页面操作

**光照页**：
- 左侧滑块：调节亮度
- 左键拖框：测量 GV

**对齐页**：
- 左键：新建 ROI/遮挡
- 选择移动：拖动 ROI
- Delete：删除选中的 ROI

**缺陷页**：
- 画缺陷：点/线/块/异物/污染
- 探查：查看缺陷详情

## Reference 目录配置

reference 目录包含机台测试数据，是应用运行所必需的。详细设置说明见 [docs/REFERENCE_SETUP.md](docs/REFERENCE_SETUP.md)。

机台信息（名称、地址、FM1/FM2/BM 数据根目录）保存在 `afvi.toml` 中：查找顺序为 `-config <文件>` → 环境变量 `AFVI_CONFIG` → exe 同级的 `afvi.toml` → 工作目录的 `afvi.toml`；相对路径相对 `afvi.toml` 所在目录解析。示例见 `teach-imgui/afvi.example.toml`，真正的 `afvi.toml` 已被 git 忽略。没有配置文件时，应用按下面的 reference 查找顺序模拟一台 `REFERENCE` 机台。

### 快速设置

**方法 1：环境变量**
```bash
export AFVI_REFERENCE_DIR=/path/to/your/reference
```

**方法 2：项目根目录**
```bash
cp -r /path/to/existing/reference ./reference/
# 或创建符号链接
ln -s /path/to/existing/reference ./reference
```

## 技术栈

- **语言**：Go 1.27
- **UI 框架**：Dear ImGui（cimgui-go v1.6.1）
- **后端**：GLFW + OpenGL3
- **图像处理**：golang.org/x/image
- **交叉编译**：MinGW-w64

## 测试与验证

### 单元测试

```bash
cd teach-imgui
go test ./config/... ./core/...
```

配置包测试 + 58 个核心单元测试，覆盖核心算法，与 TypeScript 版本跨语言验证一致。

### 自测

```bash
# WSL2 构建后，传输到 Windows 运行
.\teach-imgui.exe -selftest -out selftest-output
```

自测会：
- 遍历所有页签、主题、语言
- 注入合成鼠标输入
- 保存截图
- 输出性能指标（native-metrics.json）

## 性能指标（reference 25fbo015-15）

| 操作 | 耗时 |
|------|------|
| 加载型号 | ~300 ms |
| 光照渲染 | ~29 ms |
| GV 测量 | 15-25 ms |
| 对齐计算 | ~150 ms（1/2 工作分辨率）|
| 缺陷检出 | ~200 ms |
| UI 帧率 | 60 fps |
| 空闲 CPU | ~3.4% 单核 |

## 已知限制

- 纹理固定线性过滤，高倍放大时像素会被插值
- 60 fps 持续重绘
- 字体取自 `C:\Windows\Fonts`，仅支持 Windows
- 未适配高 DPI 缩放
- 部分机台参数（Margine、Offset1/2 等）为教学近似

## 与 Tauri 版本的关系

本项目是独立的原生实现。原 Tauri + Svelte 5 版本保留在另一个仓库（DMT_AFVI_Color_Parameter_AutoParse），作为界面还原和规格说明的参考实现。

两个版本的核心算法保持一致，但实现技术栈不同：
- **原生版（本项目）**：Go + Dear ImGui，更快的启动速度和渲染性能
- **Tauri 版**：Rust + TypeScript + Svelte 5，更现代的 Web 技术栈

## 开发指南

### 添加新页面

1. 实现 `Panel` 接口（在 `app.go` 中定义）
2. 在 `init()` 中调用 `registerPanel`
3. 实现页面的 `Render` 方法

### 修改核心算法

1. 编辑 `teach-imgui/core/` 中的相关文件
2. 运行单元测试：`go test ./core/...`
3. 重新构建应用

### 构建和测试流程

1. 修改代码
2. 在 WSL2 中运行 `./build-wsl.sh`
3. 将生成的 `.exe` 传输到 Windows
4. 在 Windows 上运行和测试

## 故障排除

见 [BUILD_NATIVE.md](BUILD_NATIVE.md) 中的故障排除章节。

## 贡献

本项目由 Claude Code 辅助开发。如有问题或建议，请参考项目文档或联系开发团队。

## 许可

[项目许可信息]
