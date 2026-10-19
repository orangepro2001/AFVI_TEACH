# 场景文件（afvi-scenario，XML）

相机合成工作室（`camera-gui.exe`）把调好的**所有设置**高密度地存成一个 XML 文件，下次直接读取；教学程序（`teach-imgui.exe`）读取同一个文件时**不需要启动 `camera.exe`**——它在进程内按文件里的配方渲染图像（与 `camera.exe` 逐字节一致）。

## 保存什么

| 内容 | XML 元素 | 说明 |
| --- | --- | --- |
| 型号 | `<model>` | id、机台、型号、面别、版本、四层图的路径、图像宽高。读取时要求型号 id 和尺寸与已加载的型号一致，否则整份拒绝（不会应用一半） |
| 光照混合 | `<light>` + `<channel>` + `<reflectance>` / `<crosstalk>` | 各光源的颜色、亮度、开关；增益、偏置、纹理强度；四种材质对 R/G/B 的反射率；可选的串扰矩阵 |
| 虚拟良品 | `<product>` + `<material>` | 开关、种子、强度、四种材质的 sigma / length / grain |
| 传感器噪声 | `<noise>` | 读出噪声、散粒噪声增益、种子 |
| 错位与翘起 | `<warp>` | 平移、旋转、缩放、剪切、翘起、弯曲（identity 时全为 0 / 1） |
| 全部笔画 | `<strokes>` → `<defect>` / `<scene>` | **所有**笔画，保持面板顺序，含被禁用的（`enabled="false"`）和程序自动生成的默认小缺陷（`auto="true"`）；多边形顶点放在 `<p x= y=/>` 子元素里 |
| 界面状态 | `<studio>` → `<brush>` `<random>` `<auto>` `<output>` | 视图、通道、工具、笔刷、随机缺陷生成器、默认缺陷与笔画种子、输出栏设置。教学程序读取时忽略这一块 |

枚举一律用名字而不是数字：`kind` = `bright` / `dark`，`shape` = `spot` / `line` / `rect` / `ellipse` / `polygon`，`amplitude` = `absolute` / `relative` / `target`，`profile` = `linear` / `gaussian`，场景笔画 `mode` = `material` / `texture`，`material` = `base` / `sr` / `open` / `metal`，光源 `color` = `red` / `green` / `blue` / `white`。浮点数用最短的精确十进制写出，读回后与原值逐位相同（所以同一份文件渲染出的像素永远一样）。

`enabled` 缺省为 `true`。**只有启用的笔画**才进入渲染配方（`Scenario.Meta().Recipe`）；禁用的笔画只是被保存下来，打开后仍在面板里，可以再启用。

## 文件名与位置

* 默认路径：`captures/<型号 id>/scenario.xml`（`/` 换成 `-`），`captures` 位于程序旁边；环境变量 `AFVI_CAPTURES` 可以改变 `captures` 的位置。
* 扩展名 `.xml.gz` 是 gzip 压缩的同一份内容（几 KB）。
* 相机工作室的"输出"栏保存抓拍时，会在抓拍文件夹里同时写一份 `scenario.xml`。

## 谁来写、谁来读

**相机工作室**（右侧输出栏的"场景文件（XML）"一节）：

* **保存场景 / 载入场景**按钮，路径可编辑。
* **自动保存，并在载入型号时自动恢复**（默认开）：设置改动约 0.5 秒后自动写入；下次载入同一型号时先恢复上次的场景，再开始自动保存（恢复尚未完成时不会覆盖旧文件）。
* 型号 id 或尺寸不一致的场景被拒绝，面板保持不变。自检（`-selftest`）从默认值开始，不读写记住的文件。

**教学程序**（"相机抓拍"一节 → 路径 → 载入）：路径可以是 `scenario.xml`、`.xml.gz`、含 `scenario.xml` 的文件夹，也可以是原来的 `snap.json` / `snap.json.gz` / 抓拍文件夹。文件夹里没有图像的通道，教学程序按文件里的配方在进程内渲染——找不到 `camera.exe` 时也一样（"抓拍"按钮和"保存会话"都不再需要 `camera.exe`）。进程内渲染与 `camera.exe` 用同一套函数；如果描述里的虚拟良品参数与当前已加载型号不同，会先按描述重新生成。

**命令行**：`camera.exe -in scenario.xml -out <目录> -channels red` 同样接受它（场景文件就是一种抓拍描述）。

## 兼容性

* `internal/snap` 的 `ReadMeta` / `Unmarshal` 按内容识别 XML（跳过 BOM 和空白），JSON 抓拍描述的读取方式完全不变。
* 未知元素与属性被忽略；缺少的元素用默认值（缺 `<light>` 等于默认光照）；`version` 目前为 1。
* 文件不含任何检出结果；检出参数仍然存在 `inspect.json`（见 `docs/INSPECT_RULES.md`）。

## 例子（节选）

```xml
<afvi-scenario format="afvi-scenario" version="1" producer="afvi-camera-studio 1" created="2026-10-20T07:41:00Z" note="">
  <model id="FM1/25fbo015-15" host="FM1" model="25fbo015-15" side="TOP" version="2.0" metal="..." sr="..." open="..." median="..." width="5860" height="4020"></model>
  <product enabled="true" seed="1" strength="1">
    <material name="sr" sigma="0.1" length="24" grain="0.06"></material>
    ...
  </product>
  <light gain="1" offset="0" textureStrength="1">
    <channel color="red" value="512" enable="true"></channel>
    ...
  </light>
  <noise readSigma="3" shotGain="0.02" seed="9"></noise>
  <warp tx="2.5" ty="0" thetaDeg="0.4" scale="1" shearX="0" shearY="0" tiltX="0" tiltY="0" bendAmplitude="0" bendWavelength="0"></warp>
  <strokes>
    <defect enabled="true" auto="true" kind="bright" shape="spot" x="79.9" y="37.5" radius="1.86" deltaGv="78.8" ...></defect>
    <defect enabled="false" auto="false" kind="dark" shape="polygon" ...><p x="100" y="120"/><p x="140" y="120"/><p x="120" y="160"/></defect>
    <scene enabled="true" auto="false" mode="material" shape="spot" x="60" y="40" radius="4" material="metal" factor="0"></scene>
  </strokes>
  <studio view="test" channel="red" showMarks="true" tool="0">
    <brush radius="17" deltaGv="60" .../>
    ...
  </studio>
</afvi-scenario>
```
