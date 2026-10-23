# ImGui 界面打磨（对齐 Wails 版观感与易用性）

保持架构（Go + Dear ImGui），在严肃工控风格内提升易用性：

- **Trackbar 滑块**（`widgets.go: trackbar/sliderCore`）：凹陷轨道 + 凸起滑块，数值显示在右侧；点击/拖动直接定位；Ctrl+点击改为键入数值；带输入框的 `SliderInput*` 保留。光照页的 R/G/B 光源行已改用同一控件。
- **表格**（`DataTableColored`）：经典凸起斜面表头；选中行深蓝底白字；内容放得下时列按比例铺满，放不下时才出现横向滚动条（并按 500 ms 重新测量内容宽度）。
- **自绘窗口外壳**（`chrome*.go`）：经典渐变标题栏（活动深蓝/非活动灰）、最小化/最大化/关闭按钮、双击标题栏最大化、5 px 边缘缩放（光标随边缘变化）。拖动与缩放只用 GLFW 的位置/大小调用加绝对光标位置；最小化与"工作区最大化"在 Windows 上用 Win32（`ShowWindow`、`SystemParametersInfo`）。Windows 默认启用，`-nativeframe` 回到系统边框；其它平台 `-chrome` 开启（Xvfb 测试用）。**Windows 上的拖动/缩放/最小化未实机验证。**
- **键盘导航**：开启 ImGui 键盘导航（Tab/方向键/空格/回车），窗口最小尺寸 960×600。
- **间距**：区块标题之间加 2 个设计像素的间隔。

验证：`SELFTEST_ARGS=-chrome ./scripts/selftest-xvfb.sh out`（Linux + Xvfb）与不带参数各跑一遍：teach 107 步、studio 33 步，全部通过，`errors: null`。
