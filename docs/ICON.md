# 应用图标

## 设计

两个 GUI 程序各有一个 Win32 经典风格的图标：银灰底色的方块，黑色外框，左上白色高光、右下 #808080 阴影，与界面里的立体边框一致。

- **教学程序（teach.ico）**：金属焊盘色带上放一枚放大镜，镜片里有一个红色缺陷斑点，下方是红-绿-蓝三色条（对应三个相机通道）。
- **工作室（studio.ico）**：黑色相机镜头，里面是红、绿、蓝三个相加的光斑（对应三路光照）。

图标由 `internal/appicon` 用纯 Go（只用标准库）以画家算法 + 4x4 超采样绘制，尺寸为 16/24/32/48/64/128/256，以 PNG-in-ICO 形式打包。

## 重新生成

```bash
go run ./cmd/mkicon
```

写出 `assets/icons/teach.ico` 与 `assets/icons/studio.ico`（已提交到仓库）。参数：`-out <目录>` 指定输出目录（默认 `assets/icons`），`-png <目录>` 额外写出 `<kind>-256.png` 便于查看。

## 嵌入方式

`cmd/teach-imgui/teach.rc` 与 `cmd/camera-gui/studio.rc` 把图标命名为 `GLFW_ICON`（GLFW 用它作为窗口与任务栏图标）和组 1（资源管理器显示的文件图标）。两个构建脚本在仓库根目录运行 windres，生成 `cmd/teach-imgui/rsrc_windows_amd64.syso` 与 `cmd/camera-gui/rsrc_windows_amd64.syso`（已被 git 忽略），Go 链接器会自动链接包目录下的 `*_windows_amd64.syso`。没有 windres 时 exe 仍能构建，只是使用默认图标。`camera.exe` 是命令行程序，没有图标。

## 验证

在 Windows 上用 `ExtractAssociatedIcon` 查看构建出的 exe 即可看到图标；`internal/appicon` 的单元测试固定了关键像素与 ICO 结构（目录项、偏移、PNG 解码尺寸）。
