# 机台配置与 Reference 目录设置说明

## 概述

应用通过机台配置文件 `afvi.toml` 描述每台机台的名称、网络地址以及三个数据根目录（FM1 / FM2 / BM）。每个数据根目录下包含 `PxRepository`（型号数据）和 `PxInventory/INSPECT_SPEC`（对齐规格）。由于数据量较大且可能包含敏感信息，reference 目录不包含在 Git 仓库中。

## 配置文件 afvi.toml

### 查找顺序

应用按以下顺序查找配置文件，使用第一个存在的文件：

1. `-config <文件>` 命令行参数指定的路径
2. `AFVI_CONFIG` 环境变量指定的路径
3. 可执行文件旁的 `afvi.toml`
4. 当前工作目录下的 `afvi.toml`

如果都没有找到，应用会以“模拟机台 REFERENCE”启动（路径来自 `FindReferenceRoot`，即 `AFVI_REFERENCE_DIR`、可执行文件旁的 `reference/` 或工作目录），只有在机台配置对话框中点击“确定/应用”时才会写出配置文件。

### 键说明

```toml
# 启动时选中的机台名称（必须与某个 [[machine]] 的 name 一致）
active = "REFERENCE"

# 界面主题："light"（经典，默认）或 "dark"
theme = "light"

# 界面语言："zh"、"en" 或 "ko"
lang = "zh"

[[machine]]
name = "REFERENCE"          # 机台名称，唯一（不区分大小写）
address = "localhost"       # 网络地址，可为空
fm1 = '../reference/FM1'    # FM1 数据根目录
fm2 = '../reference/FM2'    # FM2 数据根目录
bm  = '../reference/BM'     # BM 数据根目录
```

- 路径可以是绝对路径（`D:\FM1`）、UNC 路径（`'\\server\share\FM1'`）或相对于 `afvi.toml` 所在目录的相对路径。
- Windows 路径建议使用单引号 `'...'`，反斜杠不会被转义。
- 字符串中 `#` 不会被当作注释；行尾 `#` 注释在引号外有效。
- 未知键会被忽略，便于向前兼容。

### 相对路径规则

相对路径以 `afvi.toml` 所在目录为基准解析。例如 `afvi.toml` 位于 `teach-imgui/`，则 `'../reference/FM1'` 指向项目根目录下的 `reference/FM1`。

### 模拟机台 REFERENCE

`REFERENCE` 是内置的模拟机台，指向本机 reference 目录。使用 `-ref <目录>` 参数会新增或替换 `REFERENCE` 机台并使其成为当前机台，便于在没有真实机台时演示。

### 命令行参数

- `-config <文件>`：指定配置文件。
- `-machine <名称>`：启动时激活指定机台（不区分大小写）。
- `-ref <目录>`：设置模拟机台 REFERENCE 的路径并激活它。
- `-theme`、`-lang`：覆盖配置文件中的主题与语言。

### 兼容性说明

`AFVI_REFERENCE_DIR` 环境变量仍然有效：它通过 `FindReferenceRoot` 为模拟机台 REFERENCE 提供路径。已有脚本无需修改。

## 目录结构

```
reference/
├── FM1/
│   └── PxRepository/
│       ├── 25fbo015-15/
│       └── [其他型号]/
├── FM2/
│   └── PxRepository/
└── BM/
    └── PxRepository/
```

## 设置方法

### 方法 1：机台配置文件（推荐）

复制示例文件并修改：

```bash
cp teach-imgui/afvi.example.toml teach-imgui/afvi.toml
# 编辑 afvi.toml，填写机台名称、地址与 FM1/FM2/BM 路径
```

### 方法 2：环境变量

设置环境变量指向你的 reference 目录（供模拟机台 REFERENCE 使用）：

**Linux/WSL:**
```bash
export AFVI_REFERENCE_DIR=/path/to/your/reference
```

建议添加到 `~/.bashrc` 或 `~/.zshrc`：
```bash
echo 'export AFVI_REFERENCE_DIR=/path/to/your/reference' >> ~/.bashrc
```

**Windows PowerShell:**
```powershell
$env:AFVI_REFERENCE_DIR = "C:\path\to\your\reference"
```

或设置系统环境变量（永久）：
```powershell
[System.Environment]::SetEnvironmentVariable('AFVI_REFERENCE_DIR', 'C:\path\to\your\reference', 'User')
```

### 方法 3：放置在项目根目录

将 reference 目录直接放在项目根目录：

**Linux/WSL:**
```bash
# 如果你有现有的 reference 目录
cp -r /path/to/existing/reference ~/dev/AFVI_TEACH/reference/

# 或创建符号链接
ln -s /path/to/existing/reference ~/dev/AFVI_TEACH/reference
```

**Windows:**
```powershell
# 复制目录
Copy-Item -Recurse C:\path\to\existing\reference C:\dev\AFVI_TEACH\reference

# 或创建目录链接（需要管理员权限）
New-Item -ItemType SymbolicLink -Path C:\dev\AFVI_TEACH\reference -Target C:\path\to\existing\reference
```

### 方法 4：放置在可执行文件旁

将 reference 目录放在编译后的 .exe 文件旁边：

**Windows 运行时：**
```
teach-imgui/
├── teach-imgui.exe
└── reference/
    ├── FM1/
    ├── FM2/
    └── BM/
```

## reference 目录查找顺序

当没有配置文件时，应用按以下顺序查找 reference 目录（用于模拟机台 REFERENCE）：

1. `AFVI_REFERENCE_DIR` 环境变量指定的路径
2. 可执行文件旁的 `reference/` 目录
3. 编译时项目根目录的 `reference/` 目录
4. 向上 5 级查找 `reference/` 目录

## 测试数据示例

如果你没有真实的机台数据，可以创建一个最小的测试结构：

```bash
mkdir -p reference/FM1/PxRepository/test-model
# 在 test-model 目录中放置必要的配置文件
```

具体需要的文件结构请参考原始机台数据或联系开发团队。

## 常见问题

### Q: reference 目录有多大？
A: 取决于包含的型号数量，通常每个型号包含多张 TIFF 图像，单个型号可能有几百 MB。

### Q: 可以使用网络路径吗？
A: 可以，但建议使用本地路径以获得最佳性能。在 Windows 上使用 UNC 路径时，应用会自动处理凭据。

### Q: 如何验证设置是否正确？
A: 运行应用后，如果能正常加载型号列表，说明机台路径设置正确。也可以在机台配置对话框中点击“检查”，每个路径会显示 `OK — N models` 或具体错误。

## 安全注意事项

- reference 目录与 `afvi.toml` 已被添加到 .gitignore，不会被 Git 跟踪
- 不要将 reference 目录或包含机台地址的 `afvi.toml` 提交到公开的代码仓库
- 如果包含敏感的生产数据，请妥善保管访问权限
