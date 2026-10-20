# 检出规则：小格平均的绝对值（Offset 1）与百分比（Offset 2）规则

包 `afvi/teach-imgui/internal/inspect`，只依赖标准库。类型已在 `internal/inspect/rules.go` 与 `inspect.go` 里（只读参考，别改）：
`TileRule{Enabled, Bright, Dark, BrightSize, DarkSize}`、`MaterialParams.{IntervalOff, Abs, Pct}`、`Params.{TileSize, TileSmooth}`、`Defect.Rules`、`MaterialStats.{Tiles, TileMeanMin, TileMeanMax}`、常量 `RuleInterval/RuleAbsolute/RulePercent`、`DefaultTileSize=400`、`MinTileSize=16`、`MaxTileSize=4096`、`MaxAbsOffset=255`、`MaxPctOffset=1000`。

## 语义

图像被切成 `TileSize x TileSize` 的小格（TileSize 为 0 时用 400）。小格网格：从 (0,0) 起；每个方向上
`n = W / T`（整除）, `rem = W - n*T`；若 `n == 0` 或 `rem >= T/2`（整数除法）则 `n++`；像素 x 属于小格 `min(x / T, n-1)`。
（因此不足半格的尾巴并入最后一格，最后一格可达 1.5T 宽；不小于半格的尾巴自成一格。）

某材质 m 的"小格平均"= 该小格内、材质为 m 且**未被边缘排除**（EdgeMargin）的像素的 GV 算术平均（包含缺陷像素本身，不做剔除）。
小格内这类像素少于 64 个时该格没有自己的可靠平均，参考值改用：以该格为中心的 3x3 小格邻域内这类像素合并后的平均（合并后仍少于 64 个则）→ 整幅图该材质有效像素的平均。若整幅图该材质没有有效像素，则该材质不产生任何小格规则标记。

像素的参考值 ref：
- `TileSmooth=false`：本格的参考值。
- `TileSmooth=true`：在相邻小格中心之间做双线性插值（小格 [x0,x1) 的中心为 (x0+x1-1)/2，y 同理；在第一个中心之前/最后一个中心之后取端点值，不外推）。

规则（只对启用的材质、材质为 m、未被边缘排除的像素判断；偏移量 <= 0 表示该极性不检查；比较都是严格不等号，用 float64）：
- Offset 1 `Abs`：亮：`v > ref + Bright`；暗：`v < ref - Dark`。
- Offset 2 `Pct`：亮：`v > ref * (1 + Bright/100)`；暗：`v < ref * (1 - Dark/100)`。
- 区间规则（原有）：亮：`f > High`；暗：`f < Low`（f 为原始 GV，局部背景模式下为归一化值）。`IntervalOff=true` 时区间规则不起作用。

每条规则、每个极性各自标记像素，并各自做 8 连通分组；一个连通块的被标记像素数 < `max(该规则该极性的 Size, MinArea)` 就被丢弃
（区间规则的 Size 是 `MaterialParams.Size`；Abs/Pct 的 Size 是 `BrightSize`/`DarkSize`）。被接受的像素对三条规则取并集，
再对并集（同一材质、同一极性）做 8 连通分组，每组就是一个报告的缺陷：`Area` = 并集像素数，`Rules` = 对该缺陷有贡献（其被接受像素被该规则标记）的规则位的或。
`Excess` = 缺陷内各像素超出各自判定线的距离（GV）的最大值，一个像素被多条规则标记时取最大者；`PeakGV` 是该像素的原始 GV。
判定线：区间 High/Low；Abs 为 `ref+Bright` / `ref-Dark`；Pct 为 `ref*(1+Bright/100)` / `ref*(1-Dark/100)`。
只开区间规则时结果必须与改动前**逐字段相同**（`Rules == RuleInterval`）。
`Report.Filtered` = 被丢弃的"原始标记并集"连通块数（一个并集连通块若没有任何被接受像素，计 1）。
`Report.PerMaterial[m].Bright/Dark` = 所有规则的标记像素并集的数目（过滤前）；`FlaggedPixels` 为其总和。
`Report.PerMaterial[m].Tiles` = 含有该材质有效像素的小格数；`TileMeanMin/Max` = 这些小格自身平均的最小/最大值；该材质没有任何起作用的小格规则（Abs/Pct 对某极性 active）时三者为 0。
`ProbeRect` 的 `Bright/Dark` 计入所有规则的标记像素（不做 Size 过滤）。
`Learn` 只学习区间 Low/High，必须原样保留 IntervalOff/Abs/Pct/TileSize/TileSmooth。

## 校验（`Params.Validate`，已实现）

`TileSize` 为 0 或 16..4096；启用的材质里，启用的 Abs 规则 Bright/Dark 在 [0,255]，Pct 规则在 [0,1000]，均为有限数，BrightSize/DarkSize >= 1；
关闭的规则、关闭的材质不校验这些字段。

## 参数文件（JSON，`ParamsFile`，版本仍为 1，新增字段是向后兼容的）

材质对象新增 `"intervalOff": false`、`"abs": {"enabled","bright","dark","brightSize","darkSize"}`、`"pct": {同左}`；
Params 对象新增 `"tileSize": 400`、`"tileSmooth": false`。读取时缺少的字段保持 `DefaultParams()` 的值（包括材质对象里缺的字段，
例如旧文件里的材质没有 abs/pct，读出后 Abs/Pct 是 `defaultAbsRule()`/`defaultPctRule()` 的值，TileSize 是 400）。

## 无数据像素（Regions.Skip）

对位后的图在边缘会有“拍摄时没有数据”的像素（位移打开的边框，用最近的有效像素补齐，再加上图像最外一圈 `core.AlignBorderPx` = 8 像素，因为残余对位误差和插值无法保证最外圈）。`Regions.Skip[i]` 为 true 的像素被当作另一种材质：它本身和它周围 EdgeMargin 以内的像素一起被排除，不参与统计、`Learn`、标记和探查（`Report.EdgeIgnored` 计入）。`Skip` 是对位的几何结果，不含任何模拟信息；长度必须等于宽×高，否则报错；为 nil 时与以前完全相同。
