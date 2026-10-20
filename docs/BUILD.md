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
./scripts/build-wsl.sh                    # 一键构建：检查 + 测试 + 三个程序 + 安装包
./scripts/build-wsl.sh --offline          # 离线构建（需要模块缓存）
./scripts/build-wsl.sh --no-installer     # 只构建三个程序，不生成安装包
./scripts/build-wsl.sh --version 1.0 --reference reference   # 安装包的版本号 / 连同机台数据
```

输出：`bin/teach-imgui.exe` (~7 MB)、`bin/camera-gui.exe` 与 `bin/camera.exe`（纯 Go，无 cgo），以及最后一步生成的安装包 `dist/AFVI-Teach-Setup-<版本>.exe`（见下一节）。一键构建的最后一步是 `package-wsl.sh --no-build`，直接打包刚构建的 `bin/` 里的三个 exe；`--offline`、`--version`、`--reference` 会原样传给它。

构建脚本会用 windres 把图标资源脚本编译成 `cmd/teach-imgui/rsrc_windows_amd64.syso` 与 `cmd/camera-gui/rsrc_windows_amd64.syso`（已被 git 忽略；没有 windres 时 exe 仍能构建，只是使用默认图标）。`go run ./cmd/mkicon` 重新生成 `assets/icons/*.ico`，详见 [ICON.md](ICON.md)。

两个 GUI 都用 `-extldflags=-static` 链接（libgcc、libstdc++、winpthread 编进 exe），只依赖 Windows 自带的 DLL；否则它们会引用 `libgcc_s_seh-1.dll`、`libstdc++-6.dll`，在没有 MinGW 的 Windows 上起不来（开发机能跑只是因为 PATH 里有 MinGW 运行库）。

Windows 原生构建：

```powershell
.\scripts\build.ps1                       # 检查 + 测试 + 三个程序 + 安装包
.\scripts\build.ps1 -NoInstaller          # 只构建三个程序
.\scripts\build.ps1 -Version 1.0 -Reference reference   # 安装包的版本号 / 连同机台数据
```

`build.ps1` 为了让 `-selftest` 的 JSON 输出能回到控制台，GUI 不加 `-H windowsgui`；所以它最后调用的 `package.ps1` 会自己再构建一份 `-H windowsgui` 版本（放在 `dist\bin`），安装包里装的是这一份，不是 `bin\` 里的控制台版本。PowerShell 这条路径目前只做过语法检查，没有在带原生 Go / MinGW 的 Windows 上实际跑过。

### 安装包（setup.exe）

整个项目（教学程序、相机工作室、camera.exe、七份文档、机台配置示例）打成一个自解压安装程序 `dist/AFVI-Teach-Setup-<版本>.exe`。上一节的一键构建（`build-wsl.sh` / `build.ps1`）最后一步就会生成它；下面的脚本用于只想重新打包的时候：

```bash
./scripts/package-wsl.sh                          # 先跑 build-wsl.sh --no-installer，再打包
./scripts/package-wsl.sh --offline --no-build     # 离线；直接用 bin/ 里已有的 exe
./scripts/package-wsl.sh --version 1.0 --out dist/setup.exe
./scripts/package-wsl.sh --reference reference    # 连同机台数据一起打包
```

```powershell
.\scripts\package.ps1 [-Offline] [-Version 1.0] [-Out dist\setup.exe] [-Reference reference]   # Windows 原生
```

- 版本号默认取 `git describe --tags --always --dirty`。
- 载入的文档写在 `cmd/mkinstaller/main.go` 的固定清单里（README、USAGE、MANUAL、REFERENCE_SETUP、LINK、INSPECT_RULES、SCENARIO）；USAGE.md 新链接到别的文档时，要把它加进这份清单，否则安装后的链接是断的。
- 机台数据（`reference/`）带有机台相关、可能保密的内容，**只有显式加 `--reference` / `-Reference` 才会打进安装包**。不打包时，安装完成页会提示首次使用要放置机台数据（见 [REFERENCE_SETUP.md](REFERENCE_SETUP.md)）。
- 安装程序本身是纯 Go（`cmd/setup`，`CGO_ENABLED=0`，无需 NSIS / Inno Setup）：`cmd/mkinstaller` 把 zip 载荷（`manifest.json` 加全部文件）接在 `setup` 外壳的后面，`uninstall.exe` 就是去掉载荷的外壳。载荷里的 exe 若引用 `lib*.dll`（MinGW 运行库），`mkinstaller` 会拒绝打包。
- 安装程序带 `asInvoker` 清单（文件名叫 setup 时 Windows 否则会弹出提升权限的提示），并开启通用控件 v6 与 DPI 感知；图标与清单由 `windres` 编译 `cmd/setup/setup.rc` 得到 `cmd/setup/rsrc_windows_amd64.syso`（已被 git 忽略）。**没有 windres 脚本会直接报错。**
- 没有代码签名，Windows SmartScreen 可能提示“未知发布者”。

用户使用：

```text
AFVI-Teach-Setup-<版本>.exe                 # 图形向导：安装位置、开始菜单/桌面快捷方式、加入 PATH
AFVI-Teach-Setup-<版本>.exe -silent         # 静默安装；日志在 %TEMP%\afvi-setup.log
    -dir <文件夹>  -menu=false  -desktop=false  -path  -launch  -lang zh|en|ko
uninstall.exe                               # 卸载（开始菜单、“应用和功能”里也有入口）
    -silent  -purge                         # 静默卸载；-purge 同时删除 afvi.toml 与 captures
```

默认装到 `%LOCALAPPDATA%\Programs\AFVI Teach`（每用户安装，不需要管理员权限）。程序会把 `afvi.toml` 和相机输出写在自己旁边，所以不要装进 Program Files 之类不可写的位置。覆盖安装会升级文件、删掉新版本不再有的旧文件，但保留 `afvi.toml` 与 `captures`；正在运行的 exe 会先改名为 `.old` 再写入新文件。卸载只删除 `uninstall.json` 里记录的文件、安装程序创建的快捷方式、PATH 条目和注册表项。退出码：0 成功，1 失败，2 用法错误，3 取消。

图形向导自测：`setup.exe -dir <文件夹> -lang en -selftest <输出目录>` 会自己点完整个向导并保存 `setup-options.png`、`setup-done.png`（会真的安装到 `-dir`，并创建快捷方式，之后用 `uninstall.exe -uninstall -silent -purge` 清理）。

### 自动构建与发布（构建服务器）

推送之后的构建和发布由一台 Linux 服务器（ssh 别名 `AI`）完成，本机只做开发和测试。服务器上的 systemd 用户定时器每分钟 fetch 一次 GitHub，发现 `main` 有新提交或出现新的 `v*` tag，就在临时检出里运行 `./scripts/build-wsl.sh --version <版本>`（gofmt、vet、纯 Go 测试、三个 exe、安装包），再通过 GitHub API 发布 `AFVI-Teach-Setup-<版本>.exe` 和 `SHA256SUMS.txt`：

- **推送到 `main`**：发布到固定的滚动预发布 `nightly`。先删掉上一次的 `nightly`（连同它的 tag），再在本次提交上重新创建，所以 Release 页上只有最新一次 `main` 的安装包。版本号取 `git describe --tags --always`（如 `v0.1.0-rc2-3-gabc1234`，即最近的 tag + 之后的提交数 + 短哈希）。构建失败（gofmt、vet、测试、打包任一步）时不会动现有的 `nightly`。
- **推送 `v*` 的 tag**：发布同名 Release（自动生成发布说明；tag 里含 `-`，如 `v1.0.0-rc1`，标为预发布），版本号就是 tag 名，这个 Release 不会被覆盖。

```bash
git tag v1.0.0
git push origin v1.0.0
```

- 服务器的安装、GitHub token、目录布局、日常命令（`ci status`、手动构建、重新发布）、失败重试规则和安全注意事项见 [SERVER_CI.md](SERVER_CI.md)。每次构建一个文件夹（安装包、`SHA256SUMS.txt`、`build.log`、`meta.json`），在服务器的 `~/ci/afvi-teach/out/`。
- 每次推送到 `main` 都会公开发布一个未签名的安装包，包括只改文档的推送。
- `.github/workflows/release.yml` 只剩手动触发：在 GitHub 的 Actions 页选 Release → Run workflow，可以在干净的云端环境试跑一次构建，安装包作为 artifact 保留 14 天，不创建 Release。它没有 push / tag 触发器，免得和服务器同时改 `nightly`。
- 不用 GoReleaser：程序是 cgo + MinGW 交叉编译，还要 windres 与自解压安装包，现有脚本已经完成这些，服务器和工作流都只是准备环境并调用它。
- 安装包不含 `reference/` 机台数据（被 git 忽略，也可能保密）；自动构建不运行 `-selftest`（需要真实的 Windows 桌面窗口），自测仍在本机 Windows 上做；安装包未签名。
- 构建用的是仓库里的 `scripts/build-wsl.sh`，所以脚本依赖的文件（`scripts/package-wsl.sh`、`cmd/setup`、`cmd/mkinstaller`、`internal/installer`、`assets/icons` 等）必须已提交并推送，否则服务器上的构建会失败。
- 服务器保留 Go 编译缓存，第一次编译 cgo 包装层较慢，之后的构建快得多。

可选：运行配置包与核心算法测试（纯 Go，无需交叉编译）
```bash
go test ./internal/config/... ./internal/core/... ./internal/inspect/... ./internal/model/... ./internal/snap/... ./internal/camera/... ./internal/link/... ./internal/appicon/... ./internal/installer/...
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
├── cmd/setup/        # 安装程序（纯 Go，Windows：向导、静默安装、卸载；setup.rc 与清单）
├── cmd/mkinstaller/  # 把载荷接到 setup 外壳后面，生成 AFVI-Teach-Setup-<版本>.exe
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
│   ├── camera/       # 相机渲染与命令行（纯 Go，无 cgo）
│   └── installer/    # 安装包载荷、解压、安装日志、卸载、PATH 编辑与文字（纯 Go，仅标准库）
├── configs/          # 机台配置示例（afvi.example.toml）
├── scripts/          # build-wsl.sh / package-wsl.sh（WSL2）、build.ps1 / package.ps1（Windows）
├── deploy/server/    # 构建服务器：ci 驱动脚本、install.sh、systemd 单元（见 SERVER_CI.md）
├── docs/             # BUILD.md、SERVER_CI.md、USAGE.md、LINK.md、REFERENCE_SETUP.md
├── reference/        # 机台测试数据（需单独配置）
├── bin/              # 构建输出（bin/teach-imgui.exe 等）
└── dist/             # 安装包输出（dist/AFVI-Teach-Setup-<版本>.exe，已被 git 忽略）
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
应用已静态链接 libstdc++、libgcc、winpthread，只依赖系统 DLL（Win10+ 自带 UCRT）。如果提示缺 `libgcc_s_seh-1.dll` 或 `libstdc++-6.dll`，说明这个 exe 不是用当前的 `build-wsl.sh` 构建的（旧版没有 `-extldflags=-static`），重新构建即可；可用 `x86_64-w64-mingw32-objdump -p bin/teach-imgui.exe | grep "DLL Name"` 检查，只应看到 GDI32、KERNEL32、msvcrt、OPENGL32、SHELL32、USER32。

### 测试安装程序时的注意事项
不要在 `\\wsl.localhost\...` 上运行安装程序后又重新构建覆盖它：Windows 按需从该文件分页加载映像，旧进程还活着时替换文件，新旧进程的运行时符号表会错乱，表现为 `invalid runtime symbol table` 之类的崩溃。把安装包复制到 Windows 磁盘再运行，并且用 `taskkill` 结束测试留下的进程（WSL 的 `timeout` 只会杀掉互操作外壳，Windows 进程会继续存活）。

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
- `docs/SERVER_CI.md`、`deploy/server/`：构建服务器的自动构建与发布
- `configs/afvi.example.toml`：机台配置示例
