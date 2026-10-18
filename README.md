# AFVI Color Master Teach - 原生应用版本

Go + Dear ImGui 原生实现的 AFVI 彩色检查机教学桌面软件。

## 项目概述

本项目是 Color AFVI Master Teach 的原生 Windows 应用实现，使用 Go 语言和 Dear ImGui 构建，取代了基于 WebView2 的方案。应用为 **teach-imgui**，包含光照、对齐、缺陷三大功能页面。

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
# 构建 teach-imgui
./scripts/build-wsl.sh
```

输出为 `bin/teach-imgui.exe` 与 `bin/camera.exe`。

详细构建说明见 [docs/BUILD.md](docs/BUILD.md)。

### 运行（Windows 环境）

```powershell
# 基本运行
.\bin\teach-imgui.exe

# 指定参数
.\bin\teach-imgui.exe -ref C:\path\to\reference -theme dark -lang zh

# 指定机台配置文件
.\bin\teach-imgui.exe -config afvi.toml -machine AFVI-01

# 启动时打开特定型号
.\bin\teach-imgui.exe -model FM1/25fbo015-15

# 自测模式
.\bin\teach-imgui.exe -selftest -out selftest-output
```

界面采用经典 Win32（"Windows Classic"）外观：3D 立体边框、灰色面板、深蓝渐变对话框标题栏、属性表风格页签，以及菜单栏、工具栏和多窗格状态栏。默认主题为经典浅灰（`light`），深色主题（`dark`）沿用同一套立体边框方案。机台信息（名称、地址、FM1/FM2/BM 数据根目录）保存在 `afvi.toml` 中，可通过「文件 ▸ 机台配置…」或工具栏的 `…` 按钮编辑。

## 项目结构

```
AFVI_TEACH/
├── go.mod, go.sum            # 模块 afvi/teach-imgui（Go 1.27）
├── cmd/teach-imgui/main.go   # 程序入口（锁定 OS 线程，调用 app.Run）
├── cmd/camera/main.go        # 相机模拟程序入口（纯 Go，无 cgo）
├── internal/
│   ├── app/                  # UI 应用（package app）
│   │   ├── run.go            # Run()：命令行参数、配置加载、字体、窗口、主循环
│   │   ├── app.go            # Panel 接口、共享状态、布局（菜单栏 / 工具栏 / 页签 / 状态栏）
│   │   ├── win32.go          # Win32 立体边框原语
│   │   ├── dlg_config.go     # 机台配置对话框
│   │   ├── jobs.go  viewer.go  shapetool.go  models.go  pack.go  widgets.go  theme.go  i18n.go
│   │   ├── panel_light.go  panel_align.go  panel_defect.go
│   │   ├── toolpalette.go    # 悬浮工具面板（点选 / 矩形框 / 椭圆框 / 多边形框 / 专注）
│   │   ├── defect_select.go  # 缺陷选择工具与浮动选择编辑器
│   │   ├── defect_product.go # 虚拟良品与默认不良
│   │   ├── inspect_bridge.go # 把测试图与设计材质图交给检出的唯一桥接
│   │   ├── snap_source.go    # 相机抓拍来源（调用 camera.exe、读取抓拍文件）
│   │   └── selftest.go  capture_windows.go
│   ├── core/                 # 仿真（光照模型、渲染、不良涂抹、虚拟良品、笔画几何）与测试
│   ├── inspect/              # 检出算法：只看测试图与设计材质图，含单元测试与隔离测试
│   ├── config/               # 机台配置包（afvi.toml，纯 Go）与测试
│   ├── model/                # 模型扫描与加载（纯 Go，含 modeltest 合成参考树）
│   ├── snap/                 # 抓拍文件格式（snap.json + 灰度 PNG，仅标准库与 core）
│   └── camera/               # 相机渲染与命令行（纯 Go，无 cgo）
├── configs/afvi.example.toml # 机台配置示例
├── scripts/build-wsl.sh      # WSL2 构建脚本
├── scripts/build.ps1         # Windows 构建脚本
├── docs/BUILD.md             # 完整构建指南
├── docs/USAGE.md             # 使用说明
├── docs/REFERENCE_SETUP.md   # reference 目录设置说明
├── reference/                # 机台测试数据（需单独配置）
│   ├── FM1/
│   ├── FM2/
│   └── BM/
├── bin/                      # 构建输出（bin/teach-imgui.exe、bin/camera.exe）
├── .gitignore
└── README.md                 # 本文件
```

## 功能特性

- **光照页面**：亮度调节、GV 测量
- **对齐页面**：ROI 管理、图像对齐、遮挡处理
- **缺陷页面**：缺陷标注（点/线/块/异物/污染）、缺陷检测
- **虚拟良品与默认不良**：加载型号时把确定性的表面不均匀烘焙进光照场景，所有页面都渲染"真实"良品；缺陷页自动加入约 10 个肉眼几乎不可见的默认亮点
- **悬浮工具面板与专注模式**：视图右侧的 Photoshop 风格工具面板（点选 / 矩形框 / 椭圆框 / 多边形框 / 专注），F11 或视图菜单可隐藏两侧面板放大中央图像
- **选择与调参**：用悬浮面板选中已画的不良，在浮动窗口中批量调整幅度、半径、柔和度、颗粒、边缘轮廓与启用状态，并显示哪些不良已被 Inspect 检出
- **相机模拟与抓拍（Snap）**：独立的 `camera.exe`（纯 Go，无 cgo）把完美的 MEDIAN 渲染成有缺陷的"实拍"图，缺陷页可勾选"使用相机抓拍图像"后 Snap（把当前场景写成描述文件、调用 `camera.exe`、读回 PNG 并检出）或 Load（读入已有抓拍并导入其笔画与噪声）；抓拍与进程内渲染逐字节一致，检出依旧只看测试图与设计材质图
- **多语言支持**：中文、英文、韩文
- **主题切换**：经典浅灰 / 深色主题运行时切换（视图菜单）
- **经典 Win32 外观**：3D 立体边框、菜单栏、工具栏、多窗格状态栏
- **机台配置**：`afvi.toml` 记录每台机台的名称、地址与 FM1/FM2/BM 数据根目录，可在「文件 ▸ 机台配置…」对话框中增删改并检查路径
- **性能优化**：
  - 加载型号：~300 ms
  - 光照渲染：~29 ms
  - 对齐计算：~150 ms
  - UI 帧率：60 fps

## 操作说明

### 通用操作

- **滚轮**：缩放（以光标为中心）
- **Shift + 鼠标左键点击**：放大（×1.5，以光标为中心）；**Shift + 鼠标右键点击**：缩小（÷1.5，以光标为中心）
- **中键/右键拖动**：平移画布
- **双击**：适应窗口（画框工具绘制过程中不会触发）
- **悬浮工具面板**：视图右侧的 Photoshop 风格工具面板，含点选、矩形框、椭圆框、多边形框与专注按钮；当前页面不支持的工具会变灰并忽略点击，再次点击已激活的工具即回到平移
- **专注模式**：点击面板上的专注按钮、按 F11 或选择「视图 ▸ 专注模式」，隐藏左右两侧面板让中央图像铺满窗口；F1 参数窗口与选中的不良窗口仍可正常使用，双击适应窗口的行为不变
- **菜单栏**：文件（机台配置…、重新扫描型号、退出）、视图（主题、语言、适应窗口、专注模式）、帮助（关于…）
- **工具栏**：机台下拉框 + `…` 按钮、型号下拉框、加载状态
- **页签**：切换光照 / 对齐 / 缺陷页

### 页面操作

**光照页**：
- 左侧滑块：调节亮度
- 左键拖框：测量 GV
- 悬浮工具面板：只支持矩形框（GV 测量框，始终激活）

**对齐页**：
- 左键：新建 ROI/遮挡
- 选择移动：拖动 ROI
- Delete：删除选中的 ROI
- 悬浮工具面板：点选（选择/移动 ROI）与矩形框（新建 ROI / 遮挡）

**缺陷页**：
- 画缺陷：点/线/块/异物/污染
- 画框工具：矩形 / 椭圆 / 钢笔（类 Photoshop 钢笔：左键依次添加顶点，右键闭合多边形，Esc 取消，Backspace 撤销上一个顶点），可选择"亮/暗"极性；幅度、边缘轮廓、柔和度沿用左侧对应设置；画出的框成为不良（矩形/椭圆/多边形）并出现在笔画列表中
- 不良颜色：勾选"按颜色模拟"后，新画的不良带有 RGB 颜色（R、G、B 滑块 0–1 与七个预设：红、绿、蓝、白、黄、青、品红，列表中显示 RGB 后缀，视图中轮廓用该颜色显示）；渲染通道（Red/Green/Blue）下画面仍是黑白灰度，不良在某通道中的亮度变化 = 幅度 × 该通道的颜色分量（例如纯红不良只在 Red 通道可见，白色不良在三个通道都可见）；颜色设置只作用于之后新画的不良，随机批量生成也使用当前颜色
- 虚拟产品：左侧第二个分组（紧跟渲染通道之后）。真实板面不可能完全均匀——约 2 µm/像素下再好的表面工艺也有起伏，而加载的 MEDIAN 是多块良品的中值、过于平滑；加载型号时会把一份确定性的、带种子的空间相关变化（多尺度值噪声 + 逐像素颗粒，按材质各自的相对 σ）烘焙进光照场景，因此光照 / 对齐 / 缺陷三页渲染的都是略微不均匀的"真实"良品。控件：启用虚拟良品、纹理强度 0–3（变化幅度）、纹理种子 1–9999、相关长度倍率 0.25–4、换一块板（换随机种子）、恢复默认；任何改动都会即时重新烘焙场景。注意光照页自己的"纹理强度"滑块（0–1）会整体缩放纹理（含这份变化），把它设为 0 即得到理想均匀样品。默认种子 1、强度 1；参考型号上 SR 纹理的 RMS 变化约 0.118（相对值），5860×4020 生成一次约 0.25 s。`Scene.Stats`（逐材质 MEDIAN 统计）仍是原图的统计；Inspect 依旧只看测试图与设计材质图
- 默认不良：每次加载型号时缺陷页会自动加入约 10 个很小的亮点（数量 10、种子 7）：半径 1.5–3 像素、ΔGV 45–90（绝对）、高斯轮廓、柔和度 0.3，只放在 SR 内，且要求中心周围 `边缘忽略 + ceil(半径) + 3` 像素的整个窗口都是 SR（因此绝不会落在 Inspect 忽略的边缘带里）。它们是普通笔画（标记为 auto），出现在笔画列表中，可被选中、编辑、删除。参考型号实测（自测 `defaults-detected`）：10 条笔画，5860×4020 图中只有 175 个像素与干净渲染不同（肉眼不可见），自动学习后 10 个全部被 Inspect 检出，另有 1 个自然过检（良品并非绝对干净），学到的 SR 区间为 [2,30]。控件：默认不良数量 0–50、默认不良种子 1–9999、重新生成默认不良、移除默认不良，以及复选框"显示模拟笔画轮廓"（用至少 4 像素的标记画出笔画轮廓；关掉即可对学生隐藏答案，让他们必须通过 Inspect 找不良；选中高亮仍然可见）
- 选择与调参：悬浮工具面板的点选 / 矩形框 / 椭圆框 / 多边形框用于选择 GV 笔画。点选 = 单击选中最近的笔画（容差 6 屏幕像素，距离相同时取更小的那个；Ctrl+单击加入/移出选择；单击后继续拖动仍是平移）；矩形 / 椭圆 / 多边形框（拖动；多边形左键添加顶点、右键闭合；Esc 取消、Backspace 撤销上一个顶点）选中包围盒中心落在框内的所有笔画（Ctrl+拖动为加入而非替换；没有真正拖动的单击等同点选）。Delete 删除选中的笔画，Esc 清空选择。左侧面板武装的笔刷 / 画框放置工具优先，用完后选择工具恢复。选中的笔画以黄色高亮并保证最小屏幕尺寸（7 像素圆环 / 14 像素方框），适应窗口下 2 像素的不良也看得见；笔画列表给选中行加"● "前缀。浮动"选中的不良"窗口（选择非空时出现，`x` 清空选择）对全部选中笔画生效：更隐蔽（幅度 ×0.8、半径 ×0.9）与更清晰（幅度 ×1.25、半径 ×1.1）快捷按钮（目标模式的笔画保持目标值不变）、删除、取消选择、类型（亮/暗）、幅度模式（绝对/相对/目标）与跟随第一个选中笔画的数值滑块、半径、柔和度、颗粒、边缘轮廓、启用。窗口还显示"检出：N / M 个已被 Inspect 检出"——这只是 UI 层把报告框与笔画包围盒做比较的显示（`strokeDetected`），`internal/inspect` 依旧对笔画一无所知。自测 `select-edit`：ΔGV 80 的亮点被检出，执行"更隐蔽"（ΔGV 4、半径 1）后不再被检出，未选中的邻居仍被检出
- 检出参数：按 F1 唤出/关闭的浮动窗口（可拖动、可滚动，调整时检出结果实时刷新；Esc 或标题栏 × 关闭），左侧只保留"检出参数…（F1）"按钮与提示。窗口内按相机通道（Red/Green/Blue）各有一套参数：学习系数 k（0.5–12，默认 5）、"重新学习区间"与"恢复默认并重学"按钮、边缘忽略（像素）0–64（默认 6，三个通道共用，修改后会重新学习区间，下方给出说明与被忽略的像素数），以及 metal / sr / open / base 每个材质一节的启用、下限 Low、上限 High、分组尺寸 Size、最小面积 MinArea 与当前图统计
- 检出原理：只看测试图的亮度，按设计材质图逐材质判断——某材质的像素落在可接受区间 [Low, High] 之外就被标记（高于 High 为过亮，低于 Low 为过暗）；良品亮度越界也会被标记（过检），区间过宽则会漏检。Size 把每个被标记像素膨胀 Size/2 后把相邻分组并成一个缺陷（结果里的"碎块数"即合并的碎块数），MinArea 丢弃标记像素数不足的分组；过亮/过暗与不同材质之间不会合并。区间由良品样本学习得到：首次显示某型号/通道时（或按"重新学习区间"）渲染一张不含不良的良品样本（噪声种子不同），每个启用材质取 Low/High = 均值 ∓ k·σ；改变光照不会自动重学，因此过检/漏检会显现出来
- 边缘忽略（Margine）：与不同材质的像素的切比雪夫距离不超过"边缘忽略"（0–64 像素，默认 6）的像素会被忽略——不计入逐材质统计、不参与学习、永不被标记、探查时也不统计。报告里的"被忽略的边缘像素"与探查里的"框内被边缘忽略的像素"给出被忽略的数量。原因：参考型号的 SR 层约有 8000 个亮度 28–61 的像素（SR 均值 15），它们是 SR/Metal 交界处被模糊的过渡像素；没有边缘忽略时良品会报出 139 个过检，设为 6 后只剩 11 个（10 个真实 + 1 个）。自测：`edge-margin-off` 139 个缺陷，`edge-margin-on` 11 个
- 局部背景（BackgroundRadius，默认 2，0 为关闭；每个材质有"使用局部背景"开关，默认只对金属开启）：金属的亮度是双峰的（暗约 36、亮约 165–225），单个全局区间下，只要上限低于亮条的亮度，整条亮条就被报成一个巨大缺陷（红光 375、金属上限 148：146 个缺陷、最大 16128 像素、45 万标记像素）。打开后，该材质的每个有效像素先减去同材质 (2r+1)² 邻域的中位数（局部背景，r 为半径），区间 [Low, High] 作用于扣除背景后的值 f = (v − 背景) + 背景均值（限制在 0–255，均值不变，统计表的标准差为残差的标准差，缺陷的峰值/均值 GV 仍是原始灰度）；同样的上限下只剩 36 个很小的缺陷（最大 19 像素、296 个标记像素），也就是"过检表现为密密麻麻的小点"，并且学到的金属区间不再是盲的（同一光照下 [0,255] → [50,202]）。代价：中位数会跟随有清晰边缘的平台，只检出比窗口小的局部异常，大块平整的缺陷会被当作背景（例如 SR 上画的 7×7、+80 的方块完全检不出），所以 SR 等亮度单峰的材质默认保持全局区间；半径越大越能保留大缺陷，但结构边缘处的残差也越大（半径 3 起金属已出现上千像素的分组），所以默认取 2。
- 结果：检出数、因 MinArea 滤除的分组数、越界像素总数；逐材质统计表（材质、像素、均值、标准差、最小、最大、区间、过亮、过暗）；缺陷表（材质、极性、面积、宽×高、峰值 GV、超出量、均值 GV、碎块数，最多显示面积最大的前 500 个，点击某行视图居中）。视图上的缺陷框：橙色为过亮、蓝色为过暗，并带"材质 面积"标签
- 探查：拖框后按材质显示像素数、均值、最小/最大、区间与过亮/过暗计数，并给出通俗解释（全部落在区间内 → 没有标记，区间过宽会漏检；有标记但分组小于 MinArea → 被丢弃；否则给出框内缺陷数）
- 标准图 / 差异图只是仿真的显示辅助，检出本身从不接收标准图

## 相机模拟（camera.exe）与抓拍（Snap）

`camera.exe` 是独立的相机模拟程序（源码 `cmd/camera/main.go`，逻辑在 `internal/camera`，纯 Go、`CGO_ENABLED=0`，Linux 与 Windows 都能构建运行）：它把完美的 MEDIAN 渲染成有缺陷的"实拍"图，教学应用通过"抓拍（Snap）"读取这份实拍。两侧靠文件契约与进程边界解耦，互不依赖。

抓拍是一个文件夹：描述文件 `snap.json`（也可单独作为 `*.json` 或 gzip 压缩的 `*.json.gz` 读取；未知字段忽略；`version` 为 1）加上 8 位灰度 PNG `red.png`、`green.png`、`blue.png`（kind 为 `test`）与 `red-good.png` 等（kind 为 `good`，不含不良的良品样本，用于学习区间）。`snap.json` 携带全部元信息：`format`（`afvi-snap`）、`version`、`producer`、`created`（UTC）、`model`（id、host、model、side 以及 Metal/SR/Open/MEDIAN 各层的绝对路径）、`width`、`height`、`recipe`（`product` 虚拟良品选项、`light` 光照设置、`noise` 成像噪声、`defects` GV 不良笔画、`sceneStrokes`）与 `images`（通道、kind、文件、宽高、原始像素的 sha256，读取时校验）。用 `-mode recipe` 只写描述文件：只有几 KB（gzip 压缩后更小），任何消费者都能据此确定性地重新合成完全相同的像素。

命令行示例：

```bash
# 列出参考目录里的型号
bin/camera -ref reference -list

# 渲染一个型号的三个通道（含良品样本）
bin/camera -ref reference -model FM1/25fbo015-15 -channels red,green,blue -good -out capture/

# 只写描述文件（几 KB，可 gzip）
bin/camera -ref reference -model FM1/25fbo015-15 -mode recipe -out desc.json.gz

# 由描述文件重新渲染某个通道
bin/camera -in desc.json.gz -out capture/ -channels blue

# 查看抓拍信息
bin/camera -info capture/snap.json
```

不带 `-model` 与 `-in` 直接运行 `bin/camera` 时，使用当前机台的第一个型号（与教学应用一致），默认只渲染红色通道并写入 `snap-out/`（已被 Git 忽略）。`-channels` 接受 `red,green,blue` 或 `all`；`-count N -seed N -strength X -no-product -noise-read S -noise-shot G -noise-seed N` 调整默认场景（默认与教学应用一致：标称虚拟良品、红/绿/蓝三路光照 512、无成像噪声、10 个很小的亮点，种子 7，由应用与相机共用的 `core.DefaultTinyDefects` 生成）。`-in <snap.json|desc.json.gz|文件夹>` 重新渲染描述文件；不带 `-model` 时使用描述文件里记录的层路径；输出文件夹若已存有同一份 recipe 则保留原有图像。`-config/-machine/-ref` 与教学应用相同。退出码：0 成功、1 运行失败、2 用法错误。参考型号 25fbo015-15（5860×4020）实测：Windows 上渲染一个通道加良品样本约 4 s。

教学侧（缺陷页左侧面板"相机抓拍"分组，紧跟渲染通道分组之后）：复选框"使用相机抓拍图像"；按钮 Snap（把当前场景——启用的笔画、虚拟良品、光照、噪声——写成描述文件，运行 `camera.exe`，读回 PNG 并检出）；文本框 + Load（读取已有抓拍文件夹或描述文件，并导入其中的笔画与噪声）；抓拍元信息表格；抓拍过期时的警告（抓拍后又编辑过：需重新 Snap，在此之前仍显示已抓拍的图像）；切换通道时按需向 `camera.exe` 索取缺失的通道；取消勾选即回到进程内仿真。标准图 / 差异图仍是本地干净渲染。`camera.exe` 的查找顺序为环境变量 `AFVI_CAMERA_EXE` → 教学程序 exe 同级 → PATH。临时抓拍文件夹在被替换、重置或退出时删除。检出依旧只接收测试图与设计材质图。

解耦规则由测试守护：`internal/app` 从不导入 `internal/camera`；`internal/camera` 从不导入 `internal/inspect` 或 `internal/app`；`internal/core` 不导入 snap/model/camera/inspect 中的任何一个；`internal/snap` 只导入标准库与 `internal/core`；`internal/inspect` 保持只用标准库。`app` 包内把格式包导入为 `snapfmt`，因为该包已有一个名为 `snap` 的函数。

限制：只有缺陷页能使用抓拍（光照页与对齐页仍在进程内渲染）；每次抓拍进程都会重新加载 TIFF 各层；自测截图需要未锁屏的 Windows 会话。

## Reference 目录配置

reference 目录包含机台测试数据，是应用运行所必需的。详细设置说明见 [docs/REFERENCE_SETUP.md](docs/REFERENCE_SETUP.md)。

机台信息（名称、地址、FM1/FM2/BM 数据根目录）保存在 `afvi.toml` 中：查找顺序为 `-config <文件>` → 环境变量 `AFVI_CONFIG` → exe 同级的 `afvi.toml` → 工作目录的 `afvi.toml`；相对路径相对 `afvi.toml` 所在目录解析。示例见 `configs/afvi.example.toml`，真正的 `afvi.toml` 已被 git 忽略。没有配置文件时，应用按下面的 reference 查找顺序模拟一台 `REFERENCE` 机台。

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
go test ./internal/config/... ./internal/core/... ./internal/inspect/... ./internal/model/... ./internal/snap/... ./internal/camera/...
```

配置包 16 个测试 + 核心 73 个测试 + 检出 31 个测试 + 模型 9 个测试 + 抓拍格式 11 个测试 + 相机 19 个测试。核心的 73 个测试 = 原有 59 个 + 6 个虚拟良品测试（确定性、关闭/零强度、统计、相关性、对渲染的影响、全尺寸运行时间）+ 5 个笔画几何测试 + 3 个默认小缺陷生成器测试；检出的 31 个测试 = 原有 21 个 + 10 个局部背景测试（逐像素与朴素中位数对照、半径 0 与朴素检出一致、不同材质不混合、平滑斜坡、双峰条纹下的散点、学习区间收紧、探查、参数校验、确定性、逐材质开关）。渲染、涂抹、RNG 与颗粒仍与 TypeScript 版本跨语言验证一致；检出算法（`internal/inspect`）是新的纯 Go 实现，不再与 TS 版本对照（旧的 golden-diff 检出代码已随该实现一起移除）。

### 自测

```bash
# WSL2 构建后，传输到 Windows 运行
.\bin\teach-imgui.exe -selftest -out selftest-output
```

自测会：
- 遍历所有页签、主题、语言
- 注入合成鼠标输入
- 断言 Shift+左键放大 ×1.5 / Shift+右键缩小 ÷1.5、缺陷页椭圆与钢笔多边形创建、彩色不良在三个渲染通道下的亮度差异（红色不良只在 Red 通道变化等）
- 断言五个检出步骤：良品学习（标记像素很少）、区间收紧后良品被标记（过检）、Size 变大后缺陷数减少且碎块合并而越界像素数不变、区间过宽时画出的不良被漏检、重新学习后该不良又被检出
- 断言悬浮工具面板的逐页能力与对齐页的工具映射、专注模式把画布加宽（测试窗口下 856 → 1584 像素）并能切回、点选 / Ctrl 加选 / 矩形框选、编辑选择会改变可检出性
- 断言虚拟良品：`product-variation`（RMS ≥ 0.05）、`product-off`（恢复原始纹理，RMS 0）、`product-on`；`defaults-detected`（10 个默认笔画很小、≥ 9 个被检出、额外缺陷 ≤ 6）；`edge-margin-off`（> 20 个缺陷）/ `edge-margin-on`（≤ 20 个）
- 断言相机抓拍：`snap-camera`（抓拍图与进程内渲染逐字节相同，10 个默认笔画被检出）、`snap-stale`（编辑后报告过期）、`snap-resnap`（重新抓拍后不再过期且图像改变）、`snap-channel`（按需渲染绿色通道）、`snap-load`（读入抓拍并导入笔画）、`snap-off`（临时文件夹被删除）；自测需要 `camera.exe` 与 `teach-imgui.exe` 放在一起，时限 300 s
- 断言局部背景：bg-global-learn、bg-global-blobs（半径 0、金属上限 = 均值 + 22 时出现面积 ≥ 5000 的巨大缺陷）、bg-local-learn（学到的金属下限 ≥ 30，不再是盲的）、bg-local-dots（半径 2 时缺陷数 ≥ 20、最大面积 ≤ 400、标记像素不到全局区间的 1/10）
- 截取检出参数窗口截图
- 保存 19 张截图（原 13 张加 palette-light、palette-focus、defect-virtual、defect-edge-off、defect-snap、defect-snap-stale），共 51 个步骤
- 输出性能指标（native-metrics.json）

## 性能指标（reference 25fbo015-15）

| 操作 | 耗时 |
|------|------|
| 加载型号 | ~300 ms |
| 光照渲染 | ~29 ms |
| GV 测量 | 15-25 ms |
| 对齐计算 | ~150 ms（1/2 工作分辨率）|
| 缺陷渲染 + 检出 | ~0.8 s（最大 ~1.0 s；其中检出约 0.35 s）|
| UI 帧率 | 60 fps |
| 空闲 CPU | ~3.4% 单核 |

## 已知限制

- 纹理固定线性过滤，高倍放大时像素会被插值
- 60 fps 持续重绘
- 字体取自 `C:\Windows\Fonts`，仅支持 Windows
- 未适配高 DPI 缩放
- 检出是教学用的简化亮度区间算法（Metal / SR 各一个区间；真实产品的 Space、Open 等分层与 Margine、Offset 等机台参数未模拟），成像噪声也是近似
- 检出的边缘忽略（Margine）是单一全局像素数；虚拟良品是统计意义上的非均匀（多尺度噪声 + 颗粒），不模拟真实的划痕、氧化、批次差异；默认不良只放在 SR 内部亮点；选择工具只选 GV 笔画（异物/污染笔画不可选）

## 与 Tauri 版本的关系

本项目是独立的原生实现。原 Tauri + Svelte 5 版本保留在另一个仓库（DMT_AFVI_Color_Parameter_AutoParse），作为界面还原和规格说明的参考实现。

两个版本的核心算法保持一致，但实现技术栈不同：
- **原生版（本项目）**：Go + Dear ImGui，更快的启动速度和渲染性能
- **Tauri 版**：Rust + TypeScript + Svelte 5，更现代的 Web 技术栈

## 开发指南

### 添加新页面

1. 实现 `Panel` 接口（在 `internal/app/app.go` 中定义）
2. 在 `init()` 中调用 `registerPanel`
3. 实现页面的 `Render` 方法

### 修改核心算法

1. 编辑 `internal/core/` 中的相关文件
2. 运行单元测试：`go test ./internal/core/...`
3. 重新构建应用

### 构建和测试流程

1. 修改代码
2. 在 WSL2 中运行 `./scripts/build-wsl.sh`
3. 将生成的 `.exe` 传输到 Windows
4. 在 Windows 上运行和测试

## 故障排除

见 [docs/BUILD.md](docs/BUILD.md) 中的故障排除章节。

## 贡献

本项目由 Claude Code 辅助开发。如有问题或建议，请参考项目文档或联系开发团队。

## 许可

[项目许可信息]
