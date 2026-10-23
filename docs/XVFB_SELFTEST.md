# Linux 原生 + Xvfb 自测（ImGui 版，分支 `imgui-linux-xvfb`）

结论：旧 ImGui 版只需要三处小改动就能在 Linux 上原生构建并在 Xvfb 里跑完全部 `-selftest`。

实测（2026-10-23，Mesa llvmpipe，`reference/` 真实数据）：
- teach：107 个步骤，`errors: null`，51 张截图，180 s；studio：31 个步骤，`errors: null`，12 张截图，29 s。
- 首次 cgo 编译约 3.5 分钟，之后增量很快。

改动：
1. `capture_other.go`：Linux 截图用 `xdotool search --name` + ImageMagick `import -window`（Windows 仍用 PrintWindow）。
2. `fonts_windows.go` / `fonts_other.go`：字体路径按平台选择（Linux 用 DejaVu / Noto CJK / WenQuanYi，缺失则跳过）。
3. 运行时必须强制 X11：GLFW 在存在 `/run/user/<uid>/wayland-0` 时优先 Wayland，窗口会跑到真实桌面而不是 Xvfb；
   把 `XDG_RUNTIME_DIR` 指到空目录即可（脚本已处理）。

用法：`./scripts/selftest-xvfb.sh [输出目录]`（需要 xvfb、xdotool、imagemagick、libgl/x11 开发头文件、gcc）。

局限：仍然依赖 cgo；Windows 交叉编译仍需 MinGW 并静态链接；软件渲染的像素与 Windows 不完全一致；
仍然没有真正的跨平台 UI 自动化（断言靠 selftest 内置步骤，截图只用于人工检查）。
