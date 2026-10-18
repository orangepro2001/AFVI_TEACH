package app

import (
	"fmt"
	"math"
	"path/filepath"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/inspect"
)

func init() {
	registerPanel(2, func(a *App) Panel { return newDefectPanel(a, modeDetect) }, ProfileTeach)
	registerPanel(1, func(a *App) Panel { return newDefectPanel(a, modeAuthor) }, ProfileStudio)
	Messages(map[string]string{
		"defect.section.channel":           "渲染通道",
		"defect.section.brush":             "笔刷",
		"defect.section.frame":             "画框工具",
		"defect.section.color":             "不良颜色",
		"defect.color.enable":              "按颜色模拟（各通道亮度不同）",
		"defect.color.r":                   "红 R",
		"defect.color.g":                   "绿 G",
		"defect.color.b":                   "蓝 B",
		"defect.color.preset.red":          "红",
		"defect.color.preset.green":        "绿",
		"defect.color.preset.blue":         "蓝",
		"defect.color.preset.white":        "白",
		"defect.color.preset.yellow":       "黄",
		"defect.color.preset.cyan":         "青",
		"defect.color.preset.magenta":      "品红",
		"defect.color.weights":             "通道权重 R {r} · G {g} · B {b}",
		"defect.color.note":                "渲染通道视图仍为灰度图；缺陷在某通道的亮度变化 = 幅度 × 该通道颜色分量。设置只对之后绘制的缺陷生效。",
		"defect.stroke.colorSuffix":        " RGB {r}/{g}/{b}",
		"defect.frame.rect":                "矩形",
		"defect.frame.ellipse":             "椭圆",
		"defect.frame.pen":                 "钢笔",
		"defect.frame.kind":                "类型",
		"defect.frame.hint.rect":           "拖动画矩形框",
		"defect.frame.hint.ellipse":        "拖动画椭圆框",
		"defect.frame.hint.pen":            "左键添加顶点，右键闭合多边形，Esc 取消，Backspace 撤销上一点",
		"defect.frame.note":                "幅度、边缘轮廓与柔和度沿用上方设置；钢笔用右键闭合，Esc 取消，Backspace 撤销上一点。",
		"defect.stroke.ellipse":            "{kind}椭圆 {amp} @{x},{y}–{x2},{y2}",
		"defect.stroke.polygon":            "{kind}多边形 {amp} {n}点 @{x},{y}",
		"defect.tool.brightSpot":           "亮点",
		"defect.tool.darkSpot":             "暗点",
		"defect.tool.brightLine":           "亮线",
		"defect.tool.darkLine":             "暗线",
		"defect.tool.foreign":              "异物",
		"defect.tool.contamination":        "污染",
		"defect.tool.hint":                 "工具：{name}",
		"defect.brush.radius":              "半径",
		"defect.brush.deltaGv":             "ΔGV",
		"defect.brush.softness":            "柔和度",
		"defect.brush.foreignMetal":        "异物材料 metal",
		"defect.brush.foreignSr":           "异物材料 sr",
		"defect.brush.foreignOpen":         "异物材料 open",
		"defect.brush.contaminationFactor": "污染系数",
		"defect.brush.undo":                "撤销",
		"defect.brush.clear":               "清空",
		"defect.brush.empty":               "尚无笔画。",
		"defect.brush.delete":              "删除",
		"defect.section.params":            "检出参数",
		"defect.params.title":              "检出参数",
		"defect.params.open":               "检出参数…（F1）",
		"defect.params.f1hint":             "按 F1 随时打开或关闭检出参数窗口，Esc 关闭。",
		"defect.section.tools":             "工具",
		"defect.tool.probe":                "探查（拖框）",
		"defect.tool.probeHint":            "在视图上拖框进行探查",
		"defect.view.test":                 "测试图",
		"defect.view.golden":               "标准图",
		"defect.view.diff":                 "差异图",
		"defect.section.result":            "检出结果",
		"defect.section.probe":             "探查",
		"defect.result.empty":              "尚未执行检出。",
		"defect.result.defectCount":        "检出数",
		"defect.result.filteredCount":      "因 MinArea 滤除的分组",
		"defect.result.flaggedPixels":      "越界像素总数",
		"defect.mcol.material":             "材质",
		"defect.mcol.pixels":               "像素",
		"defect.mcol.mean":                 "均值",
		"defect.mcol.std":                  "标准差",
		"defect.mcol.min":                  "最小",
		"defect.mcol.max":                  "最大",
		"defect.mcol.range":                "区间",
		"defect.mcol.bright":               "过亮",
		"defect.mcol.dark":                 "过暗",
		"defect.col.material":              "材质",
		"defect.col.polarity":              "极性",
		"defect.col.area":                  "面积",
		"defect.col.size":                  "宽×高",
		"defect.col.peakGv":                "峰值 GV",
		"defect.col.excess":                "超出量",
		"defect.col.meanGv":                "均值 GV",
		"defect.col.fragments":             "碎块数",
		"defect.polarity.bright":           "过亮",
		"defect.polarity.dark":             "过暗",
		"defect.kind.bright":               "亮",
		"defect.kind.dark":                 "暗",
		"defect.probe.empty":               "尚未探查。",
		"defect.probe.summary":             "框 {w}×{h}，相交缺陷 {defects} 个",
		"defect.probe.row":                 "{m}：{pixels} 像素，均值 {mean} [{min}–{max}]，区间 [{low}, {high}]，过亮 {bright} / 过暗 {dark}",
		"defect.probe.disabled":            "{m}：该材质已关闭检出，框内像素不会被标记。",
		"defect.probe.allInside":           "框内所有像素都在可接受区间内，没有像素被标记：区间过宽就会漏检。",
		"defect.probe.filtered":            "有 {pixels} 个越界像素被标记，但所在分组小于 MinArea 被滤除，所以没有缺陷。",
		"defect.probe.detected":            "框内有 {count} 个缺陷。",
		"defect.insp.channel":              "检出参数（通道 {name}；每个通道各自一套）",
		"defect.insp.explain":              "检出只看测试图的亮度：某材质的像素落在 [Low, High] 之外就被标记（良品也会，即过检）；区间放太宽会漏检；Size 小则得到很多小缺陷，Size 大则相邻的合并成一个大缺陷；分组面积小于 MinArea 的被丢弃。",
		"defect.insp.enabled":              "启用该材质",
		"defect.insp.low":                  "下限 Low",
		"defect.insp.high":                 "上限 High",
		"defect.insp.size":                 "分组尺寸 Size",
		"defect.insp.minArea":              "最小面积 MinArea",
		"defect.insp.stats":                "当前图统计：{pixels} 像素，均值 {mean}，σ {std}，范围 [{min}–{max}]",
		"defect.insp.learnK":               "学习系数 k",
		"defect.insp.relearn":              "重新学习区间",
		"defect.insp.reset":                "恢复默认并重学",
		"defect.insp.learnNote":            "学习使用不含不良的良品样本（噪声种子与测试图不同），区间 = 均值 ± k·σ。k 越大区间越宽、越容易漏检；k 越小越容易过检（良品被检出）。",
		"defect.stroke.foreign":            "异物 {material} @{x},{y}",
		"defect.stroke.contamination":      "污染 ×{factor} @{x},{y}",
		"defect.stroke.spot":               "{kind}点 r{r} {amp} @{x},{y}",
		"defect.stroke.line":               "{kind}线 r{r} {amp} @{x},{y}→{x2},{y2}",
		"defect.stroke.rect":               "{kind}块 {amp} @{x},{y}–{x2},{y2}",
		"defect.currentGray":               "缺陷 {kind}",
		"defect.tool.brightRect":           "亮块",
		"defect.tool.darkRect":             "暗块",
		"defect.section.amplitude":         "幅度模式",
		"defect.amplitude.absolute":        "绝对 ΔGV",
		"defect.amplitude.relative":        "相对 %（随光照）",
		"defect.amplitude.target":          "目标 GV",
		"defect.brush.relativePct":         "相对幅度 %",
		"defect.brush.targetBright":        "亮目标 GV",
		"defect.brush.targetDark":          "暗目标 GV",
		"defect.section.profile":           "边缘轮廓",
		"defect.profile.linear":            "线性",
		"defect.profile.gaussian":          "高斯",
		"defect.brush.grain":               "颗粒",
		"defect.brush.enabled":             "启用",
		"defect.section.random":            "随机批量",
		"defect.random.count":              "数量",
		"defect.random.seed":               "种子",
		"defect.random.kind":               "类型",
		"defect.random.kind.bright":        "亮",
		"defect.random.kind.dark":          "暗",
		"defect.random.kind.mixed":         "混合",
		"defect.random.shape":              "形状",
		"defect.random.shape.spot":         "点",
		"defect.random.shape.line":         "线",
		"defect.random.shape.mixed":        "混合",
		"defect.random.radiusMin":          "半径最小",
		"defect.random.radiusMax":          "半径最大",
		"defect.random.valueMin":           "幅度最小",
		"defect.random.valueMax":           "幅度最大",
		"defect.random.material":           "放置材料",
		"defect.random.material.all":       "全部",
		"defect.random.generate":           "生成",
		"defect.section.noise":             "成像噪声",
		"defect.noise.readSigma":           "读出噪声 σ",
		"defect.noise.shotGain":            "散粒噪声系数",
		"defect.noise.seed":                "种子",
		"defect.noise.note":                "噪声只加到测试图；标准图视为多板中值无噪声；σ² = 读出² + 散粒系数×GV，GV 越高噪声越大。",
		"defect.result.truncated":          "仅显示面积最大的前 {shown} 个，共 {total} 个。",
	}, map[string]string{
		"defect.section.channel":           "Render Channel",
		"defect.section.brush":             "Brush",
		"defect.section.frame":             "Frame Tool",
		"defect.section.color":             "Defect Color",
		"defect.color.enable":              "Simulate by color (per-channel brightness)",
		"defect.color.r":                   "Red R",
		"defect.color.g":                   "Green G",
		"defect.color.b":                   "Blue B",
		"defect.color.preset.red":          "Red",
		"defect.color.preset.green":        "Green",
		"defect.color.preset.blue":         "Blue",
		"defect.color.preset.white":        "White",
		"defect.color.preset.yellow":       "Yellow",
		"defect.color.preset.cyan":         "Cyan",
		"defect.color.preset.magenta":      "Magenta",
		"defect.color.weights":             "Channel weights R {r} · G {g} · B {b}",
		"defect.color.note":                "The render channel view stays grayscale; a defect's brightness change in a channel = amplitude × that channel's color component. Settings apply to newly drawn defects.",
		"defect.stroke.colorSuffix":        " RGB {r}/{g}/{b}",
		"defect.frame.rect":                "Rectangle",
		"defect.frame.ellipse":             "Ellipse",
		"defect.frame.pen":                 "Pen",
		"defect.frame.kind":                "Polarity",
		"defect.frame.hint.rect":           "Drag to draw a rectangle",
		"defect.frame.hint.ellipse":        "Drag to draw an ellipse",
		"defect.frame.hint.pen":            "Left click adds a vertex, right click closes the polygon, Esc cancels, Backspace removes the last vertex",
		"defect.frame.note":                "Amplitude, edge profile and softness come from the sections above; the pen closes with a right click, Esc cancels, Backspace removes the last vertex.",
		"defect.stroke.ellipse":            "{kind} ellipse {amp} @{x},{y}–{x2},{y2}",
		"defect.stroke.polygon":            "{kind} polygon {amp} {n} pts @{x},{y}",
		"defect.tool.brightSpot":           "Bright spot",
		"defect.tool.darkSpot":             "Dark spot",
		"defect.tool.brightLine":           "Bright line",
		"defect.tool.darkLine":             "Dark line",
		"defect.tool.foreign":              "Foreign object",
		"defect.tool.contamination":        "Contamination",
		"defect.tool.hint":                 "Tool: {name}",
		"defect.brush.radius":              "Radius",
		"defect.brush.deltaGv":             "ΔGV",
		"defect.brush.softness":            "Softness",
		"defect.brush.foreignMetal":        "Foreign material metal",
		"defect.brush.foreignSr":           "Foreign material sr",
		"defect.brush.foreignOpen":         "Foreign material open",
		"defect.brush.contaminationFactor": "Contamination factor",
		"defect.brush.undo":                "Undo",
		"defect.brush.clear":               "Clear",
		"defect.brush.empty":               "No strokes yet.",
		"defect.brush.delete":              "Delete",
		"defect.section.params":            "Detection Parameters",
		"defect.params.title":              "Detection Parameters",
		"defect.params.open":               "Detection Parameters… (F1)",
		"defect.params.f1hint":             "Press F1 to open or close the detection-parameter window at any time; Esc closes it.",
		"defect.section.tools":             "Tools",
		"defect.tool.probe":                "Probe (drag box)",
		"defect.tool.probeHint":            "Drag a box on the view to probe",
		"defect.view.test":                 "Test image",
		"defect.view.golden":               "Golden image",
		"defect.view.diff":                 "Difference image",
		"defect.section.result":            "Detection Result",
		"defect.section.probe":             "Probe",
		"defect.result.empty":              "Detection has not been run yet.",
		"defect.result.defectCount":        "Defects",
		"defect.result.filteredCount":      "Groups below MinArea",
		"defect.result.flaggedPixels":      "Flagged pixels",
		"defect.mcol.material":             "Material",
		"defect.mcol.pixels":               "Pixels",
		"defect.mcol.mean":                 "Mean",
		"defect.mcol.std":                  "Std",
		"defect.mcol.min":                  "Min",
		"defect.mcol.max":                  "Max",
		"defect.mcol.range":                "Range",
		"defect.mcol.bright":               "Bright",
		"defect.mcol.dark":                 "Dark",
		"defect.col.material":              "Material",
		"defect.col.polarity":              "Polarity",
		"defect.col.area":                  "Area",
		"defect.col.size":                  "W×H",
		"defect.col.peakGv":                "Peak GV",
		"defect.col.excess":                "Excess",
		"defect.col.meanGv":                "Mean GV",
		"defect.col.fragments":             "Fragments",
		"defect.polarity.bright":           "Bright",
		"defect.polarity.dark":             "Dark",
		"defect.kind.bright":               "Bright",
		"defect.kind.dark":                 "Dark",
		"defect.probe.empty":               "No probe yet.",
		"defect.probe.summary":             "Box {w}×{h}, {defects} intersecting defects",
		"defect.probe.row":                 "{m}: {pixels} pixels, mean {mean} [{min}–{max}], range [{low}, {high}], bright {bright} / dark {dark}",
		"defect.probe.disabled":            "{m}: inspection is disabled for this material, so its pixels in the box are never flagged.",
		"defect.probe.allInside":           "Every pixel in the box is inside the accepted interval, so nothing was flagged: a too-wide range misses defects.",
		"defect.probe.filtered":            "{pixels} out-of-range pixels were flagged, but their group is smaller than MinArea and was dropped, so there is no defect.",
		"defect.probe.detected":            "{count} defects in the box.",
		"defect.insp.channel":              "Inspection parameters (channel {name}; one set per channel)",
		"defect.insp.explain":              "Inspection only looks at the test image brightness: a pixel of a material outside [Low, High] is flagged (a good product can be flagged too, i.e. over-kill); a too-wide range misses defects; a small Size yields many small defects while a large Size merges neighbours into one big defect; groups smaller than MinArea are dropped.",
		"defect.insp.enabled":              "Enabled",
		"defect.insp.low":                  "Low",
		"defect.insp.high":                 "High",
		"defect.insp.size":                 "Size",
		"defect.insp.minArea":              "MinArea",
		"defect.insp.stats":                "Current image: {pixels} px, mean {mean}, σ {std}, range [{min}–{max}]",
		"defect.insp.learnK":               "Learn factor k",
		"defect.insp.relearn":              "Re-learn ranges",
		"defect.insp.reset":                "Reset and re-learn",
		"defect.insp.learnNote":            "Learning uses a defect-free good sample (different noise seed than the test image); range = mean ± k·σ. A larger k widens the range and misses more; a smaller k over-kills good product.",
		"defect.stroke.foreign":            "Foreign {material} @{x},{y}",
		"defect.stroke.contamination":      "Contamination ×{factor} @{x},{y}",
		"defect.stroke.spot":               "{kind} spot r{r} {amp} @{x},{y}",
		"defect.stroke.line":               "{kind} line r{r} {amp} @{x},{y}→{x2},{y2}",
		"defect.stroke.rect":               "{kind} rect {amp} @{x},{y}–{x2},{y2}",
		"defect.currentGray":               "Defect {kind}",
		"defect.tool.brightRect":           "Bright rect",
		"defect.tool.darkRect":             "Dark rect",
		"defect.section.amplitude":         "Amplitude Mode",
		"defect.amplitude.absolute":        "Absolute ΔGV",
		"defect.amplitude.relative":        "Relative % (scales with light)",
		"defect.amplitude.target":          "Target GV",
		"defect.brush.relativePct":         "Relative amplitude %",
		"defect.brush.targetBright":        "Bright target GV",
		"defect.brush.targetDark":          "Dark target GV",
		"defect.section.profile":           "Edge Profile",
		"defect.profile.linear":            "Linear",
		"defect.profile.gaussian":          "Gaussian",
		"defect.brush.grain":               "Grain",
		"defect.brush.enabled":             "Enabled",
		"defect.section.random":            "Random Batch",
		"defect.random.count":              "Count",
		"defect.random.seed":               "Seed",
		"defect.random.kind":               "Kind",
		"defect.random.kind.bright":        "Bright",
		"defect.random.kind.dark":          "Dark",
		"defect.random.kind.mixed":         "Mixed",
		"defect.random.shape":              "Shape",
		"defect.random.shape.spot":         "Spot",
		"defect.random.shape.line":         "Line",
		"defect.random.shape.mixed":        "Mixed",
		"defect.random.radiusMin":          "Radius min",
		"defect.random.radiusMax":          "Radius max",
		"defect.random.valueMin":           "Value min",
		"defect.random.valueMax":           "Value max",
		"defect.random.material":           "Placement material",
		"defect.random.material.all":       "All",
		"defect.random.generate":           "Generate",
		"defect.section.noise":             "Imaging Noise",
		"defect.noise.readSigma":           "Read noise σ",
		"defect.noise.shotGain":            "Shot noise gain",
		"defect.noise.seed":                "Seed",
		"defect.noise.note":                "Noise is added to the test image only; the golden image is treated as a multi-board median with no noise; σ² = read² + shotGain×GV, so higher GV means more noise.",
		"defect.result.truncated":          "Showing the largest {shown} of {total}.",
	}, map[string]string{
		"defect.section.channel":           "렌더링 채널",
		"defect.section.brush":             "브러시",
		"defect.section.frame":             "프레임 도구",
		"defect.section.color":             "결함 색상",
		"defect.color.enable":              "색상으로 시뮬레이션 (채널별 밝기 차이)",
		"defect.color.r":                   "빨강 R",
		"defect.color.g":                   "초록 G",
		"defect.color.b":                   "파랑 B",
		"defect.color.preset.red":          "빨강",
		"defect.color.preset.green":        "초록",
		"defect.color.preset.blue":         "파랑",
		"defect.color.preset.white":        "흰색",
		"defect.color.preset.yellow":       "노랑",
		"defect.color.preset.cyan":         "청록",
		"defect.color.preset.magenta":      "자홍",
		"defect.color.weights":             "채널 가중치 R {r} · G {g} · B {b}",
		"defect.color.note":                "렌더링 채널 뷰는 계속 흑백입니다. 결함의 채널별 밝기 변화 = 진폭 × 해당 채널 색상 성분. 설정은 이후에 그리는 결함에만 적용됩니다.",
		"defect.stroke.colorSuffix":        " RGB {r}/{g}/{b}",
		"defect.frame.rect":                "사각형",
		"defect.frame.ellipse":             "타원",
		"defect.frame.pen":                 "펜",
		"defect.frame.kind":                "극성",
		"defect.frame.hint.rect":           "드래그하여 사각형을 그립니다",
		"defect.frame.hint.ellipse":        "드래그하여 타원을 그립니다",
		"defect.frame.hint.pen":            "왼쪽 클릭으로 꼭짓점 추가, 오른쪽 클릭으로 다각형 닫기, Esc 취소, Backspace 마지막 꼭짓점 취소",
		"defect.frame.note":                "진폭, 가장자리 프로파일, 부드러움은 위 섹션 설정을 따릅니다. 펜은 오른쪽 클릭으로 닫고, Esc 취소, Backspace 마지막 꼭짓점 취소.",
		"defect.stroke.ellipse":            "{kind} 타원 {amp} @{x},{y}–{x2},{y2}",
		"defect.stroke.polygon":            "{kind} 다각형 {amp} {n}점 @{x},{y}",
		"defect.tool.brightSpot":           "밝은 점",
		"defect.tool.darkSpot":             "어두운 점",
		"defect.tool.brightLine":           "밝은 선",
		"defect.tool.darkLine":             "어두운 선",
		"defect.tool.foreign":              "이물",
		"defect.tool.contamination":        "오염",
		"defect.tool.hint":                 "도구: {name}",
		"defect.brush.radius":              "반경",
		"defect.brush.deltaGv":             "ΔGV",
		"defect.brush.softness":            "부드러움",
		"defect.brush.foreignMetal":        "이물 재질 metal",
		"defect.brush.foreignSr":           "이물 재질 sr",
		"defect.brush.foreignOpen":         "이물 재질 open",
		"defect.brush.contaminationFactor": "오염 계수",
		"defect.brush.undo":                "실행 취소",
		"defect.brush.clear":               "비우기",
		"defect.brush.empty":               "아직 획이 없습니다.",
		"defect.brush.delete":              "삭제",
		"defect.section.params":            "검출 파라미터",
		"defect.params.title":              "검출 파라미터",
		"defect.params.open":               "검출 파라미터… (F1)",
		"defect.params.f1hint":             "F1 키로 언제든지 검출 파라미터 창을 열거나 닫을 수 있으며, Esc 로 닫습니다.",
		"defect.section.tools":             "도구",
		"defect.tool.probe":                "탐색 (드래그 박스)",
		"defect.tool.probeHint":            "뷰에서 박스를 드래그하여 탐색합니다",
		"defect.view.test":                 "테스트 이미지",
		"defect.view.golden":               "기준 이미지",
		"defect.view.diff":                 "차이 이미지",
		"defect.section.result":            "검출 결과",
		"defect.section.probe":             "탐색",
		"defect.result.empty":              "아직 검출을 실행하지 않았습니다.",
		"defect.result.defectCount":        "검출 수",
		"defect.result.filteredCount":      "MinArea 미만 제외 그룹",
		"defect.result.flaggedPixels":      "범위 밖 픽셀 수",
		"defect.mcol.material":             "재질",
		"defect.mcol.pixels":               "픽셀",
		"defect.mcol.mean":                 "평균",
		"defect.mcol.std":                  "표준편차",
		"defect.mcol.min":                  "최소",
		"defect.mcol.max":                  "최대",
		"defect.mcol.range":                "구간",
		"defect.mcol.bright":               "과밝음",
		"defect.mcol.dark":                 "과어두움",
		"defect.col.material":              "재질",
		"defect.col.polarity":              "극성",
		"defect.col.area":                  "면적",
		"defect.col.size":                  "너비×높이",
		"defect.col.peakGv":                "피크 GV",
		"defect.col.excess":                "초과량",
		"defect.col.meanGv":                "평균 GV",
		"defect.col.fragments":             "조각 수",
		"defect.polarity.bright":           "과밝음",
		"defect.polarity.dark":             "과어두움",
		"defect.kind.bright":               "밝음",
		"defect.kind.dark":                 "어두움",
		"defect.probe.empty":               "아직 탐색하지 않았습니다.",
		"defect.probe.summary":             "박스 {w}×{h}, 교차 결함 {defects}개",
		"defect.probe.row":                 "{m}: {pixels} 픽셀, 평균 {mean} [{min}–{max}], 구간 [{low}, {high}], 과밝음 {bright} / 과어두움 {dark}",
		"defect.probe.disabled":            "{m}: 이 재질의 검출이 꺼져 있어 박스 안 픽셀은 표시되지 않습니다.",
		"defect.probe.allInside":           "박스 안 모든 픽셀이 허용 구간 안에 있어 아무것도 표시되지 않았습니다: 구간이 너무 넓으면 결함을 놓칩니다.",
		"defect.probe.filtered":            "{pixels}개의 범위 밖 픽셀이 표시되었지만 그 그룹이 MinArea보다 작아 제외되어 결함이 없습니다.",
		"defect.probe.detected":            "박스 안에 결함이 {count}개 있습니다.",
		"defect.insp.channel":              "검출 파라미터 (채널 {name}; 채널마다 별도)",
		"defect.insp.explain":              "검출은 테스트 이미지의 밝기만 봅니다: 어떤 재질의 픽셀이 [Low, High] 밖이면 표시됩니다(양품도 표시될 수 있으며 이를 과검출이라 합니다). 구간이 너무 넓으면 결함을 놓칩니다. Size가 작으면 작은 결함이 많이 생기고, Size가 크면 인접한 것들이 하나의 큰 결함으로 합쳐집니다. 그룹 면적이 MinArea보다 작으면 제외됩니다.",
		"defect.insp.enabled":              "사용",
		"defect.insp.low":                  "하한 Low",
		"defect.insp.high":                 "상한 High",
		"defect.insp.size":                 "그룹 크기 Size",
		"defect.insp.minArea":              "최소 면적 MinArea",
		"defect.insp.stats":                "현재 이미지: {pixels} 픽셀, 평균 {mean}, σ {std}, 범위 [{min}–{max}]",
		"defect.insp.learnK":               "학습 계수 k",
		"defect.insp.relearn":              "구간 다시 학습",
		"defect.insp.reset":                "기본값 복원 후 재학습",
		"defect.insp.learnNote":            "학습은 불량이 없는 양품 샘플을 사용합니다(테스트 이미지와 다른 노이즈 시드). 구간 = 평균 ± k·σ. k가 클수록 구간이 넓어져 미검이 늘고, 작을수록 양품이 검출되는 과검이 늘어납니다.",
		"defect.stroke.foreign":            "이물 {material} @{x},{y}",
		"defect.stroke.contamination":      "오염 ×{factor} @{x},{y}",
		"defect.stroke.spot":               "{kind} 점 r{r} {amp} @{x},{y}",
		"defect.stroke.line":               "{kind} 선 r{r} {amp} @{x},{y}→{x2},{y2}",
		"defect.stroke.rect":               "{kind} 사각 {amp} @{x},{y}–{x2},{y2}",
		"defect.currentGray":               "결함 {kind}",
		"defect.tool.brightRect":           "밝은 사각",
		"defect.tool.darkRect":             "어두운 사각",
		"defect.section.amplitude":         "진폭 모드",
		"defect.amplitude.absolute":        "절대 ΔGV",
		"defect.amplitude.relative":        "상대 % (조명에 비례)",
		"defect.amplitude.target":          "목표 GV",
		"defect.brush.relativePct":         "상대 진폭 %",
		"defect.brush.targetBright":        "밝은 목표 GV",
		"defect.brush.targetDark":          "어두운 목표 GV",
		"defect.section.profile":           "가장자리 프로파일",
		"defect.profile.linear":            "선형",
		"defect.profile.gaussian":          "가우시안",
		"defect.brush.grain":               "그레인",
		"defect.brush.enabled":             "사용",
		"defect.section.random":            "랜덤 배치",
		"defect.random.count":              "개수",
		"defect.random.seed":               "시드",
		"defect.random.kind":               "유형",
		"defect.random.kind.bright":        "밝음",
		"defect.random.kind.dark":          "어두움",
		"defect.random.kind.mixed":         "혼합",
		"defect.random.shape":              "모양",
		"defect.random.shape.spot":         "점",
		"defect.random.shape.line":         "선",
		"defect.random.shape.mixed":        "혼합",
		"defect.random.radiusMin":          "반경 최소",
		"defect.random.radiusMax":          "반경 최대",
		"defect.random.valueMin":           "진폭 최소",
		"defect.random.valueMax":           "진폭 최대",
		"defect.random.material":           "배치 재질",
		"defect.random.material.all":       "전체",
		"defect.random.generate":           "생성",
		"defect.section.noise":             "이미징 노이즈",
		"defect.noise.readSigma":           "읽기 노이즈 σ",
		"defect.noise.shotGain":            "샷 노이즈 계수",
		"defect.noise.seed":                "시드",
		"defect.noise.note":                "노이즈는 테스트 이미지에만 추가됩니다. 기준 이미지는 다중 보드 중앙값으로 노이즈가 없다고 간주합니다. σ² = 읽기² + 샷 계수×GV 이므로 GV가 높을수록 노이즈가 큽니다.",
		"defect.result.truncated":          "면적이 큰 상위 {shown}개만 표시합니다. 전체 {total}개.",
	})
}

const maxOverlayLabels = 200

// maxOverlayBoxes 限制每帧绘制的缺陷框数量。
const maxOverlayBoxes = 3000

// maxResultRows 限制结果表显示的行数（r.Defects 已按面积降序）。
const maxResultRows = 500

// overlayBrightCol / overlayDarkCol 是缺陷框的过亮/过暗颜色。
var (
	overlayBrightCol = rgba(255, 140, 0, 255)
	overlayDarkCol   = rgba(80, 160, 255, 255)
)

// defectTool 是当前笔刷工具。
type defectTool int

// 笔刷工具常量。
const (
	toolNone defectTool = iota
	toolBrightSpot
	toolDarkSpot
	toolBrightLine
	toolDarkLine
	toolBrightRect
	toolDarkRect
	toolForeign
	toolContamination
	toolProbe
	toolFrame
)

// strokeEntry 是一条笔画（GV 或场景）及其启用状态。
type strokeEntry struct {
	gv      *core.DefectStroke
	scene   *core.SceneStroke
	enabled bool
	auto    bool // default tiny defect added on model load
}

// DefectPanel 移植 teach-app/src/defect-panel.ts 为原生 ImGui 面板。
type DefectPanel struct {
	a    *App
	mode defectMode

	tool              defectTool
	radius            float64
	deltaGv           float64
	softness          float64
	amplitude         core.DefectAmplitude
	relativePct       float64
	targetBright      float64
	targetDark        float64
	profile           core.DefectProfile
	grain             float64
	foreignMaterial   uint8
	contaminationFact float64
	useColor          bool
	color             [3]float64
	strokes           []strokeEntry
	viewKind          string // "test" | "golden" | "diff"
	channel           core.CameraChannel
	noise             core.SensorNoise

	inspectParams [3]inspect.Params // 每个相机通道一套检出参数
	learned       [3]bool           // 该通道是否已学习过区间
	relearn       bool              // 下一次运行强制重新学习当前通道
	learnK        float64           // 学习区间 = 均值 ± k·σ

	randomCount     int
	randomSeed      uint32
	randomKind      core.RandomKind
	randomShape     core.RandomShape
	randomRadiusMin float64
	randomRadiusMax float64
	randomValueMin  float64
	randomValueMax  float64
	randomMaterial  int // -1 = all, else material index

	strokeSeed uint32

	autoCount       int
	autoSeed        uint32
	productOpts     core.ProductOptions
	productLenScale float64
	productDraw     int

	golden *core.GrayImage
	test   *core.GrayImage

	lastReport *inspect.Report
	lastProbe  *inspect.ProbeReport
	probeRect  *core.Rect

	frame     *FrameTool
	frameKind core.DefectKind

	sel       selectionState
	showMarks bool // 绘制模拟笔画轮廓（关闭可对学生隐藏答案）

	snapSrc snapSource
	studio  studioOutput

	pub          livePublisher // studio: when the scenario was last published over the live link
	liveFollow   bool          // teach: follow the studio's scenario while it is connected
	liveSt       liveState     // teach: connection state as of the current frame
	liveMismatch string        // teach: model of the studio's scenario when it differs from the loaded one
	inspMsg      statusMsg     // teach: result of the last save/load of the inspection parameter file

	manual     bool      // teach: no automatic learning or inspection (the Inspect button runs it)
	inspectNow bool      // teach: the next requestRun learns and inspects even in manual mode
	user       userImage // teach: test image loaded from a file (Load Image)
	imgMsg     statusMsg // teach: result of the last Load Image

	paramsOpen bool

	active bool

	renderRev   uint64
	renderedRev uint64

	dragging  bool
	dragStart core.Point
	dragCur   core.Point
}

func newDefectPanel(a *App, mode defectMode) *DefectPanel {
	p := &DefectPanel{
		a:                 a,
		mode:              mode,
		tool:              toolNone,
		radius:            6,
		deltaGv:           60,
		softness:          0.3,
		amplitude:         core.AmpAbsolute,
		relativePct:       40,
		targetBright:      250,
		targetDark:        10,
		profile:           core.ProfileLinear,
		grain:             0,
		foreignMaterial:   core.MatMetal,
		contaminationFact: 1.5,
		useColor:          false,
		color:             [3]float64{1, 0, 0},
		viewKind:          "test",
		channel:           core.ChRed,
		noise:             core.SensorNoise{Seed: 1},
		randomCount:       10,
		randomSeed:        1,
		randomKind:        core.RandomMixedKind,
		randomShape:       core.RandomMixedShape,
		randomRadiusMin:   2,
		randomRadiusMax:   8,
		randomValueMin:    30,
		randomValueMax:    80,
		randomMaterial:    -1,
		strokeSeed:        1,
		renderRev:         1,
		frame:             NewFrameTool(),
		frameKind:         core.Bright,
		showMarks:         mode == modeAuthor,
		liveFollow:        mode == modeDetect,
		learnK:            5,
		autoCount:         defaultAutoCount,
		autoSeed:          defaultAutoSeed,
		productOpts:       core.DefaultProductOptions(),
		productLenScale:   1,
	}
	for i := range p.inspectParams {
		p.inspectParams[i] = newInspectParams()
	}
	return p
}

func (p *DefectPanel) ID() string { return "defect" }

func (p *DefectPanel) Title() string {
	if p.mode == modeAuthor {
		return T("tab.synth")
	}
	return T("tab.defect")
}

func (p *DefectPanel) Activate() {
	p.active = true
	if p.lastReport == nil || p.renderRev != p.renderedRev {
		p.requestRun()
	} else {
		p.requestView()
	}
}

func (p *DefectPanel) Deactivate() {
	p.active = false
	p.dragging = false
	p.paramsOpen = false
	p.sel.set = nil
	if p.sel.frame != nil {
		p.sel.frame.Reset()
	}
	if p.frame != nil {
		p.frame.Reset()
	}
	if p.a != nil && p.a.Viewer != nil {
		p.a.Viewer.SetDoubleClickFit(true)
	}
}

func (p *DefectPanel) ModelLoaded() {
	p.resetSnap()
	p.strokes = nil
	p.lastReport = nil
	p.lastProbe = nil
	p.probeRect = nil
	p.golden = nil
	p.test = nil
	p.strokeSeed = 1
	p.renderRev++
	p.sel.set = nil
	if p.sel.frame != nil {
		p.sel.frame.Reset()
	}
	// 换模型后已学到的区间失效，恢复默认并重新学习。
	p.learned = [3]bool{}
	p.relearn = false
	p.clearUserImage()
	p.imgMsg = statusMsg{}
	p.inspectNow = false
	for i := range p.inspectParams {
		p.inspectParams[i] = newInspectParams()
	}
	if p.frame != nil {
		p.frame.Reset()
	}
	if p.a != nil && p.a.Viewer != nil {
		p.a.Viewer.SetDoubleClickFit(true)
	}
	if p.a != nil && p.a.Scene != nil {
		p.productOpts = p.a.Scene.Product
		p.productLenScale = 1
	}
	p.addAutoDefects()
}

func (p *DefectPanel) LightChanged() {
	// 光照变化后刻意保留已学到的区间，让过检/漏检显现出来；教师需显式重新学习。
	p.renderRev++
	if p.active {
		p.requestRun()
	}
}

// resetInspect 把当前通道的检出参数恢复默认，并在下一次运行时重新学习区间。
func (p *DefectPanel) resetInspect() {
	p.inspectParams[p.channel] = newInspectParams()
	p.learned[p.channel] = false
}

// requestRelearn 保留当前通道的 Size/MinArea/启用设置，仅重新学习区间。
func (p *DefectPanel) requestRelearn() {
	p.relearn = true
	p.inspectNow = true
}

func (p *DefectPanel) Selftest() {
	if len(p.strokes) == 0 {
		cx, cy := p.findSrPoint()
		stroke := &core.DefectStroke{
			Kind:      core.Bright,
			Shape:     core.ShapeSpot,
			X:         float64(cx),
			Y:         float64(cy),
			Radius:    6,
			DeltaGv:   60,
			Softness:  0.3,
			Amplitude: core.AmpAbsolute,
			Profile:   core.ProfileLinear,
			Grain:     0,
			Seed:      1,
		}
		p.strokes = append(p.strokes, strokeEntry{gv: stroke, enabled: true})
		p.strokeSeed = 2
	}
	p.generateRandom(1, 10)
	p.renderRev++
	p.requestRun()
}

// findSrPoint 在材料图中寻找 SR 区域中心点（移植 TS findSrPoint）。
func (p *DefectPanel) findSrPoint() (int, int) {
	scene := p.a.Scene
	if scene == nil {
		return 0, 0
	}
	width, height := scene.W, scene.H
	cx := width / 2
	cy := height / 2
	material := scene.Material
	const rad = 3 + defaultEdgeMargin + 1
	isSr := func(x, y int) bool {
		if x < rad || y < rad || x >= width-rad || y >= height-rad {
			return false
		}
		for dy := -rad; dy <= rad; dy++ {
			for dx := -rad; dx <= rad; dx++ {
				if material[(y+dy)*width+(x+dx)] != core.MatSR {
					return false
				}
			}
		}
		return true
	}
	if isSr(cx, cy) {
		return cx, cy
	}
	maxRing := width
	if height > maxRing {
		maxRing = height
	}
	for r := 8; r <= maxRing; r += 8 {
		for d := -r; d <= r; d += 8 {
			candidates := [][2]int{
				{cx + d, cy - r},
				{cx + d, cy + r},
				{cx - r, cy + d},
				{cx + r, cy + d},
			}
			for _, c := range candidates {
				if isSr(c[0], c[1]) {
					return c[0], c[1]
				}
			}
		}
	}
	return cx, cy
}

// gvStrokes 返回启用的 GV 笔画。
func (p *DefectPanel) gvStrokes() []core.DefectStroke {
	var out []core.DefectStroke
	for _, e := range p.strokes {
		if e.gv != nil && e.enabled {
			out = append(out, *e.gv)
		}
	}
	return out
}

// sceneStrokes 返回启用的场景笔画。
func (p *DefectPanel) sceneStrokes() []core.SceneStroke {
	var out []core.SceneStroke
	for _, e := range p.strokes {
		if e.scene != nil && e.enabled {
			out = append(out, *e.scene)
		}
	}
	return out
}

// requestRun 提交渲染 + 检出任务（key "defect"）。
func (p *DefectPanel) requestRun() {
	a := p.a
	scene := a.Scene
	if scene == nil {
		return
	}
	if p.mode == modeAuthor {
		p.requestRender()
		return
	}
	inspectNow := p.inspectNow
	p.inspectNow = false
	doInspect := !p.manual || inspectNow
	userImg := p.user.img
	needRender := p.renderRev != p.renderedRev || p.golden == nil
	rev := p.renderRev
	light := a.Light.Clone()
	ch := p.channel
	noise := p.noise
	params := p.inspectParams[ch]
	needLearn := doInspect && (!p.learned[ch] || p.relearn)
	k := p.learnK
	viewKind := p.viewKind
	gv := p.gvStrokes()
	sc := p.sceneStrokes()
	golden, test := p.golden, p.test
	liveOn := userImg == nil && p.snapSrc.live && p.snapSrc.meta != nil
	snapOn := userImg == nil && p.snapSrc.on && !p.snapSrc.live && p.snapSrc.meta != nil
	liveSess, liveRev := p.snapSrc.liveSess, p.snapSrc.liveRev
	liveCl := p.liveSt.client
	exe := findCameraExe()
	srcDir := p.snapSrc.srcDir
	workDir := p.snapSrc.workDir
	snapMeta := p.snapSrc.meta
	have := make(map[snapKey]*core.GrayImage, len(p.snapSrc.frames))
	for k, v := range p.snapSrc.frames {
		have[k] = v
	}
	a.Jobs.Run("defect", func() func() {
		var res snapFetchResult
		if userImg != nil {
			test, golden = userImg, nil
		} else if liveOn {
			var err error
			res, err = liveFetch(liveCl, liveRev, have, ch, needLearn, viewKind != "test", scene.W, scene.H)
			if err != nil {
				return func() { p.liveFetchFailed(liveSess, liveRev, err) }
			}
			test, golden = res.test, res.golden
		} else if snapOn {
			var err error
			res, err = snapFetch(exe, srcDir, snapMeta, have, workDir, ch, needLearn, scene.W, scene.H)
			if err != nil {
				return func() {
					p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.snap.error", err.Error(), true
					a.SetHint(T("common.error", "msg", err.Error()))
				}
			}
			test = res.test
			golden = core.RenderChannel(scene.Light, &light, ch)
		} else if needRender {
			golden, test = core.RenderDefectPair(scene.Light, &light, ch, sc, gv, noise)
		}
		if needLearn {
			var good *core.GrayImage
			if userImg != nil {
				good = userImg
			} else if liveOn || snapOn {
				if res.good == nil {
					return func() {
						p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.snap.error", "good sample missing", true
						a.SetHint(T("common.error", "msg", "good sample missing"))
					}
				}
				good = res.good
			} else {
				good = core.RenderGoodSample(scene.Light, &light, ch, goodSampleNoise(noise))
			}
			learnedParams, err := inspect.Learn(inspectImage(good), inspectRegions(scene), k, params)
			if err != nil {
				return func() { a.SetHint(T("common.error", "msg", err.Error())) }
			}
			params = learnedParams
		}
		var report inspect.Report
		if doInspect {
			var err error
			report, err = inspect.Inspect(inspectImage(test), inspectRegions(scene), params)
			if err != nil {
				return func() { a.SetHint(T("common.error", "msg", err.Error())) }
			}
		}
		var pix []byte
		switch {
		case viewKind == "golden" && golden != nil:
			pix = PackGray(golden)
		case viewKind == "diff" && golden != nil:
			pix = PackSignedDiff(golden, test)
		default:
			pix = PackGray(test)
		}
		w, h := test.Width, test.Height
		return func() {
			if a.Scene != scene {
				PutRGBA(pix)
				return
			}
			if p.user.img != userImg {
				PutRGBA(pix)
				return
			}
			if liveOn && (!p.snapSrc.live || p.snapSrc.liveSess != liveSess || p.snapSrc.liveRev != liveRev) {
				// The studio moved on (or the user left the live source) while this was running;
				// the run for the newer revision is already queued.
				PutRGBA(pix)
				return
			}
			p.golden = golden
			p.test = test
			if liveOn {
				if p.snapSrc.frames == nil {
					p.snapSrc.frames = map[snapKey]*core.GrayImage{}
				}
				for k, v := range res.add {
					p.snapSrc.frames[k] = v
				}
				p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "", "", false
			}
			if snapOn {
				if p.snapSrc.frames == nil {
					p.snapSrc.frames = map[snapKey]*core.GrayImage{}
				}
				for k, v := range res.add {
					p.snapSrc.frames[k] = v
				}
				if res.workDir != "" {
					p.ownDir(res.workDir)
					p.snapSrc.workDir = res.workDir
				}
				if res.rendered && res.workDir == p.snapSrc.srcDir {
					p.snapSrc.meta = res.meta
				}
			}
			p.renderedRev = rev
			if doInspect {
				p.lastReport = &report
			} else {
				p.lastReport, p.lastProbe, p.probeRect = nil, nil, nil
			}
			if needLearn {
				p.inspectParams[ch] = params
				p.learned[ch] = true
				// 只在学习结果落地时清除请求，被替换掉的排队任务不会吞掉"重新学习"。
				p.relearn = false
			}
			if p.viewKind != viewKind {
				PutRGBA(pix)
				p.requestView()
				return
			}
			if p.active {
				a.Viewer.SetImage(w, h, pix)
			} else {
				PutRGBA(pix)
			}
		}
	})
}

// requestRender 渲染 golden/test 图对并显示当前视图（key "defect"）；不学习、不检出、
// 不使用相机抓拍。
func (p *DefectPanel) requestRender() {
	a := p.a
	scene := a.Scene
	if scene == nil {
		return
	}
	needRender := p.renderRev != p.renderedRev || p.golden == nil
	rev := p.renderRev
	light := a.Light.Clone()
	ch := p.channel
	noise := p.noise
	viewKind := p.viewKind
	gv := p.gvStrokes()
	sceneStr := p.sceneStrokes()
	golden, test := p.golden, p.test
	a.Jobs.Run("defect", func() func() {
		if needRender {
			golden, test = core.RenderDefectPair(scene.Light, &light, ch, sceneStr, gv, noise)
		}
		var pix []byte
		switch viewKind {
		case "golden":
			pix = PackGray(golden)
		case "diff":
			pix = PackSignedDiff(golden, test)
		default:
			pix = PackGray(test)
		}
		w, h := golden.Width, golden.Height
		return func() {
			if a.Scene != scene {
				PutRGBA(pix)
				return
			}
			p.golden = golden
			p.test = test
			p.renderedRev = rev
			if p.viewKind != viewKind {
				PutRGBA(pix)
				p.requestView()
				return
			}
			if p.active {
				a.Viewer.SetImage(w, h, pix)
			} else {
				PutRGBA(pix)
			}
		}
	})
}

// requestView 从已有 golden/test 重新打包当前视图（key "defect-view"）。
func (p *DefectPanel) requestView() {
	a := p.a
	scene := a.Scene
	if scene == nil || p.test == nil {
		return
	}
	if p.golden == nil && p.viewKind != "test" {
		// A live test image arrives without its golden image; fetch it for the other views.
		p.requestRun()
		return
	}
	viewKind := p.viewKind
	golden, test := p.golden, p.test
	a.Jobs.Run("defect-view", func() func() {
		var pix []byte
		switch viewKind {
		case "golden":
			pix = PackGray(golden)
		case "diff":
			pix = PackSignedDiff(golden, test)
		default:
			pix = PackGray(test)
		}
		w, h := test.Width, test.Height
		return func() {
			if a.Scene != scene || p.test != test {
				PutRGBA(pix)
				return
			}
			if p.active {
				a.Viewer.SetImage(w, h, pix)
			} else {
				PutRGBA(pix)
			}
		}
	})
}

// requestProbe 在矩形区域上执行探查（key "defect-probe"）。
func (p *DefectPanel) requestProbe(rect core.Rect) {
	a := p.a
	scene := a.Scene
	if scene == nil || p.test == nil || p.lastReport == nil {
		return
	}
	params := p.inspectParams[p.channel]
	test := p.test
	report := p.lastReport
	a.Jobs.Run("defect-probe", func() func() {
		pr, err := inspect.ProbeRect(inspectImage(test), inspectRegions(scene), params, *report, inspect.Rect{
			X: int(math.Floor(rect.X)),
			Y: int(math.Floor(rect.Y)),
			W: int(math.Ceil(rect.W)),
			H: int(math.Ceil(rect.H)),
		})
		if err != nil {
			return func() { a.SetHint(T("common.error", "msg", err.Error())) }
		}
		return func() {
			if a.Scene != scene || p.test != test {
				return
			}
			p.lastProbe = &pr
		}
	})
}

// generateRandom 在 UI 线程生成随机缺陷笔画。
func (p *DefectPanel) generateRandom(seed uint32, count int) {
	scene := p.a.Scene
	if scene == nil {
		return
	}
	var materials []uint8
	if p.randomMaterial >= 0 {
		materials = []uint8{uint8(p.randomMaterial)}
	}
	opts := core.RandomDefectOptions{
		Count:         count,
		Seed:          seed,
		Width:         scene.W,
		Height:        scene.H,
		Kind:          p.randomKind,
		Shape:         p.randomShape,
		RadiusMin:     math.Min(p.randomRadiusMin, p.randomRadiusMax),
		RadiusMax:     math.Max(p.randomRadiusMin, p.randomRadiusMax),
		ValueMin:      math.Min(p.randomValueMin, p.randomValueMax),
		ValueMax:      math.Max(p.randomValueMin, p.randomValueMax),
		Amplitude:     p.amplitude,
		Profile:       p.profile,
		Softness:      p.softness,
		Grain:         p.grain,
		LineLengthMax: 60,
		Material:      scene.Material,
		Materials:     materials,
		HasColor:      p.useColor,
		Color:         p.color,
	}
	strokes := core.GenerateRandomDefects(opts)
	for i := range strokes {
		s := strokes[i]
		p.strokes = append(p.strokes, strokeEntry{gv: &s, enabled: true})
	}
	p.renderRev++
}

// applyColor copies the panel colour onto a GV stroke when colour simulation is on.
func (p *DefectPanel) applyColor(s *core.DefectStroke) {
	if !p.useColor {
		return
	}
	s.HasColor = true
	for i := 0; i < 3; i++ {
		v := p.color[i]
		if v < 0 {
			v = 0
		} else if v > 1 {
			v = 1
		}
		s.Color[i] = v
	}
}

// colorOutline returns the packed outline colour for a defect colour, normalised so the
// largest component is 255; all-zero colours fall back to OverlayMuted.
func colorOutline(c [3]float64) uint32 {
	max := 0.0
	for _, v := range c {
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		return OverlayMuted
	}
	r := int(c[0]/max*255 + 0.5)
	g := int(c[1]/max*255 + 0.5)
	b := int(c[2]/max*255 + 0.5)
	return rgba(r, g, b, 255)
}

// strokeLabel 按当前语言生成笔画标签，带颜色时追加 RGB 后缀。
func (p *DefectPanel) strokeLabel(s *core.DefectStroke) string {
	label := p.strokeLabelBase(s)
	if s.HasColor {
		label += T("defect.stroke.colorSuffix", "r", F(s.Color[0], 2), "g", F(s.Color[1], 2), "b", F(s.Color[2], 2))
	}
	return label
}

// strokeLabelBase 按当前语言生成笔画基础标签。
func (p *DefectPanel) strokeLabelBase(s *core.DefectStroke) string {
	kind := T("defect.kind.dark")
	if s.Kind == core.Bright {
		kind = T("defect.kind.bright")
	}
	var amp string
	switch s.Amplitude {
	case core.AmpRelative:
		pct := int(math.Round(s.RelativePct))
		if s.Kind == core.Bright {
			amp = fmt.Sprintf("+%d%%", pct)
		} else {
			amp = fmt.Sprintf("−%d%%", pct)
		}
	case core.AmpTarget:
		tg := s.TargetGv
		if !s.HasTarget {
			if s.Kind == core.Bright {
				tg = 255
			} else {
				tg = 0
			}
		}
		amp = fmt.Sprintf("→%d", int(math.Round(tg)))
	default:
		amp = fmt.Sprintf("Δ%d", int(math.Round(s.DeltaGv)))
	}
	x := F(s.X, 0)
	y := F(s.Y, 0)
	switch s.Shape {
	case core.ShapeRect:
		return T("defect.stroke.rect", "kind", kind, "amp", amp, "x", x, "y", y, "x2", F(s.X2, 0), "y2", F(s.Y2, 0))
	case core.ShapeEllipse:
		return T("defect.stroke.ellipse", "kind", kind, "amp", amp, "x", x, "y", y, "x2", F(s.X2, 0), "y2", F(s.Y2, 0))
	case core.ShapePolygon:
		return T("defect.stroke.polygon", "kind", kind, "amp", amp, "n", fmt.Sprint(len(s.Points)), "x", x, "y", y)
	case core.ShapeLine:
		return T("defect.stroke.line", "kind", kind, "r", F(s.Radius, 1), "amp", amp, "x", x, "y", y, "x2", F(s.X2, 0), "y2", F(s.Y2, 0))
	}
	return T("defect.stroke.spot", "kind", kind, "r", F(s.Radius, 1), "amp", amp, "x", x, "y", y)
}

// gvAmplitudeFields 按当前幅度模式填充 GV 笔画字段。
func (p *DefectPanel) gvAmplitudeFields(kind core.DefectKind) (core.DefectAmplitude, float64, float64, float64, bool) {
	switch p.amplitude {
	case core.AmpRelative:
		return core.AmpRelative, 0, p.relativePct, 0, false
	case core.AmpTarget:
		tg := p.targetDark
		if kind == core.Bright {
			tg = p.targetBright
		}
		return core.AmpTarget, 0, 0, tg, true
	}
	return core.AmpAbsolute, p.deltaGv, 0, 0, false
}

func (p *DefectPanel) placeSpot(x, y, radius float64) {
	switch p.tool {
	case toolForeign:
		s := core.SceneStroke{
			Mode:     core.SceneMaterial,
			Shape:    core.ShapeSpot,
			X:        x,
			Y:        y,
			Radius:   radius,
			Material: p.foreignMaterial,
		}
		p.strokes = append(p.strokes, strokeEntry{scene: &s, enabled: true})
	case toolContamination:
		s := core.SceneStroke{
			Mode:   core.SceneTexture,
			Shape:  core.ShapeSpot,
			X:      x,
			Y:      y,
			Radius: radius,
			Factor: p.contaminationFact,
		}
		p.strokes = append(p.strokes, strokeEntry{scene: &s, enabled: true})
	default:
		kind := core.Dark
		if p.tool == toolBrightSpot {
			kind = core.Bright
		}
		amp, dg, pct, tg, hasTg := p.gvAmplitudeFields(kind)
		s := core.DefectStroke{
			Kind:        kind,
			Shape:       core.ShapeSpot,
			X:           x,
			Y:           y,
			Radius:      radius,
			DeltaGv:     dg,
			Softness:    p.softness,
			Amplitude:   amp,
			RelativePct: pct,
			TargetGv:    tg,
			HasTarget:   hasTg,
			Profile:     p.profile,
			Grain:       p.grain,
			Seed:        p.strokeSeed,
		}
		p.applyColor(&s)
		p.strokeSeed++
		p.strokes = append(p.strokes, strokeEntry{gv: &s, enabled: true})
	}
	p.renderRev++
	p.requestRun()
}

func (p *DefectPanel) placeLine(x, y, x2, y2 float64) {
	kind := core.Dark
	if p.tool == toolBrightLine {
		kind = core.Bright
	}
	amp, dg, pct, tg, hasTg := p.gvAmplitudeFields(kind)
	s := core.DefectStroke{
		Kind:        kind,
		Shape:       core.ShapeLine,
		X:           x,
		Y:           y,
		X2:          x2,
		Y2:          y2,
		Radius:      p.radius,
		DeltaGv:     dg,
		Softness:    p.softness,
		Amplitude:   amp,
		RelativePct: pct,
		TargetGv:    tg,
		HasTarget:   hasTg,
		Profile:     p.profile,
		Grain:       p.grain,
		Seed:        p.strokeSeed,
	}
	p.applyColor(&s)
	p.strokeSeed++
	p.strokes = append(p.strokes, strokeEntry{gv: &s, enabled: true})
	p.renderRev++
	p.requestRun()
}

func (p *DefectPanel) placeRect(x, y, x2, y2 float64) {
	kind := core.Dark
	if p.tool == toolBrightRect {
		kind = core.Bright
	}
	amp, dg, pct, tg, hasTg := p.gvAmplitudeFields(kind)
	s := core.DefectStroke{
		Kind:        kind,
		Shape:       core.ShapeRect,
		X:           x,
		Y:           y,
		X2:          x2,
		Y2:          y2,
		Radius:      0,
		DeltaGv:     dg,
		Softness:    p.softness,
		Amplitude:   amp,
		RelativePct: pct,
		TargetGv:    tg,
		HasTarget:   hasTg,
		Profile:     p.profile,
		Grain:       p.grain,
		Seed:        p.strokeSeed,
	}
	p.applyColor(&s)
	p.strokeSeed++
	p.strokes = append(p.strokes, strokeEntry{gv: &s, enabled: true})
	p.renderRev++
	p.requestRun()
}

// placeFrame commits a finished frame tool result as a GV stroke.
func (p *DefectPanel) placeFrame(res FrameResult) {
	kind := p.frameKind
	amp, dg, pct, tg, hasTg := p.gvAmplitudeFields(kind)
	s := core.DefectStroke{
		Kind:        kind,
		DeltaGv:     dg,
		Softness:    p.softness,
		Amplitude:   amp,
		RelativePct: pct,
		TargetGv:    tg,
		HasTarget:   hasTg,
		Profile:     p.profile,
		Grain:       p.grain,
		Seed:        p.strokeSeed,
	}
	p.strokeSeed++
	switch res.Kind {
	case FrameEllipse:
		s.Shape = core.ShapeEllipse
		s.X, s.Y = res.P0.X, res.P0.Y
		s.X2, s.Y2 = res.P1.X, res.P1.Y
	case FramePen:
		s.Shape = core.ShapePolygon
		s.Points = append([]core.Point(nil), res.Points...)
		if len(s.Points) > 0 {
			s.X, s.Y = s.Points[0].X, s.Points[0].Y
		}
		s.Radius = 0
	default:
		s.Shape = core.ShapeRect
		s.X, s.Y = res.P0.X, res.P0.Y
		s.X2, s.Y2 = res.P1.X, res.P1.Y
	}
	p.applyColor(&s)
	p.strokes = append(p.strokes, strokeEntry{gv: &s, enabled: true})
	p.renderRev++
	p.requestRun()
}

// handleKeys processes frame-tool keyboard shortcuts (Esc cancels, Backspace undoes).
func (p *DefectPanel) handleKeys() {
	if p.mode == modeDetect && imgui.IsKeyPressedBoolV(imgui.KeyF5, false) && !imgui.IsPopupOpenStrV("", imgui.PopupFlagsAnyPopupId) {
		p.requestInspect()
	}
	if p.paramsOpen {
		// Esc closes the parameter dialog first; the frame tool must not react to it.
		return
	}
	if p.tool != toolFrame || p.frame == nil {
		return
	}
	if imgui.IsKeyPressedBool(imgui.KeyEscape) {
		p.frame.Reset()
		p.tool = toolNone
		if p.a != nil && p.a.Viewer != nil {
			p.a.Viewer.SetDoubleClickFit(true)
		}
		p.a.SetHint("")
	}
	if imgui.IsKeyPressedBool(imgui.KeyBackspace) {
		p.frame.Undo()
	}
}

func (p *DefectPanel) DrawLeft() {
	p.handleKeys()
	if p.mode == modeAuthor {
		p.handleSelectKeys()
	}
	a := p.a
	changed := false

	if Section(T("defect.section.channel"), "defect-ch") {
		ch := int(p.channel)
		if Combo(T("defect.section.channel"), "defect-channel", &ch, []string{"Red", "Green", "Blue"}) {
			p.channel = core.CameraChannel(ch)
			p.renderRev++
			changed = true
		}
	}

	if p.mode == modeDetect {
		p.drawManualSection()
		p.drawLiveSection()
		if p.drawSnapSection() {
			changed = true
		}
	}

	if p.mode == modeAuthor && p.drawVirtualSection() {
		changed = true
	}

	if p.mode == modeAuthor && Section(T("defect.section.brush"), "defect-brush") {
		tools := []struct {
			tool  defectTool
			label string
		}{
			{toolBrightSpot, T("defect.tool.brightSpot")},
			{toolDarkSpot, T("defect.tool.darkSpot")},
			{toolBrightLine, T("defect.tool.brightLine")},
			{toolDarkLine, T("defect.tool.darkLine")},
			{toolBrightRect, T("defect.tool.brightRect")},
			{toolDarkRect, T("defect.tool.darkRect")},
			{toolForeign, T("defect.tool.foreign")},
			{toolContamination, T("defect.tool.contamination")},
		}
		for i, entry := range tools {
			if i > 0 {
				WrapSameLine(entry.label)
			}
			if ToolButton(entry.label, fmt.Sprintf("defect-tool-%d", i), p.tool == entry.tool) {
				p.tool = entry.tool
				a.SetHint(T("defect.tool.hint", "name", entry.label))
			}
		}
		SliderF64(T("defect.brush.radius"), "defect-radius", &p.radius, 0.5, 40, "%.1f")
		SliderF64(T("defect.brush.softness"), "defect-softness", &p.softness, 0, 1, "%.2f")
		// Combo order is metal, sr, open; map from the material index (metal=3, sr=1, open=2).
		fm := 0
		switch p.foreignMaterial {
		case core.MatSR:
			fm = 1
		case core.MatOpen:
			fm = 2
		}
		if Combo(T("defect.tool.foreign"), "defect-foreign", &fm, []string{
			T("defect.brush.foreignMetal"), T("defect.brush.foreignSr"), T("defect.brush.foreignOpen"),
		}) {
			switch fm {
			case 1:
				p.foreignMaterial = core.MatSR
			case 2:
				p.foreignMaterial = core.MatOpen
			default:
				p.foreignMaterial = core.MatMetal
			}
		}
		SliderF64(T("defect.brush.contaminationFactor"), "defect-contam", &p.contaminationFact, 0.2, 3, "%.2f")

		if len(p.strokes) == 0 {
			Note(T("defect.brush.empty"))
		} else {
			del := -1
			for i := range p.strokes {
				e := &p.strokes[i]
				imgui.PushIDInt(int32(i))
				if imgui.Checkbox("##en", &e.enabled) {
					p.renderRev++
					changed = true
				}
				imgui.SameLine()
				label := ""
				if e.gv != nil {
					label = p.strokeLabel(e.gv)
				} else if e.scene != nil {
					if e.scene.Mode == core.SceneMaterial {
						label = T("defect.stroke.foreign", "material", core.MaterialNames[e.scene.Material&3], "x", F(e.scene.X, 0), "y", F(e.scene.Y, 0))
					} else {
						label = T("defect.stroke.contamination", "factor", F(e.scene.Factor, 2), "x", F(e.scene.X, 0), "y", F(e.scene.Y, 0))
					}
				}
				if e.gv != nil && p.sel.set[e.gv] {
					label = "● " + label
				}
				imgui.AlignTextToFramePadding()
				imgui.TextUnformatted(label)
				imgui.SameLine()
				if Button(T("defect.brush.delete"), "defect-del") {
					del = i
				}
				imgui.PopID()
			}
			if del >= 0 {
				p.strokes = append(p.strokes[:del], p.strokes[del+1:]...)
				p.renderRev++
				changed = true
			}
		}
		if Button(T("defect.brush.undo"), "defect-undo") {
			if len(p.strokes) > 0 {
				p.strokes = p.strokes[:len(p.strokes)-1]
				p.renderRev++
				changed = true
			}
		}
		imgui.SameLine()
		if Button(T("defect.brush.clear"), "defect-clear") {
			p.strokes = nil
			p.renderRev++
			changed = true
		}
	}

	if p.mode == modeAuthor && Section(T("defect.section.frame"), "defect-frame") {
		frameTools := []struct {
			kind  FrameKind
			label string
			hint  string
		}{
			{FrameRect, T("defect.frame.rect"), T("defect.frame.hint.rect")},
			{FrameEllipse, T("defect.frame.ellipse"), T("defect.frame.hint.ellipse")},
			{FramePen, T("defect.frame.pen"), T("defect.frame.hint.pen")},
		}
		for i, entry := range frameTools {
			if i > 0 {
				WrapSameLine(entry.label)
			}
			active := p.tool == toolFrame && p.frame != nil && p.frame.Kind == entry.kind
			if ToolButton(entry.label, fmt.Sprintf("defect-frame-%d", i), active) {
				p.frame.Reset()
				p.frame.Kind = entry.kind
				p.tool = toolFrame
				a.SetHint(entry.hint)
			}
		}
		fk := int(p.frameKind)
		if Combo(T("defect.frame.kind"), "defect-frame-kind", &fk, []string{
			T("defect.kind.bright"), T("defect.kind.dark"),
		}) {
			p.frameKind = core.DefectKind(fk)
		}
		Note(T("defect.frame.note"))
	}

	if p.mode == modeAuthor && Section(T("defect.section.color"), "defect-color") {
		Check(T("defect.color.enable"), "defect-color-enable", &p.useColor)
		if p.useColor {
			SliderF64(T("defect.color.r"), "defect-color-r", &p.color[0], 0, 1, "%.2f")
			SliderF64(T("defect.color.g"), "defect-color-g", &p.color[1], 0, 1, "%.2f")
			SliderF64(T("defect.color.b"), "defect-color-b", &p.color[2], 0, 1, "%.2f")
			presets := []struct {
				label string
				col   [3]float64
			}{
				{T("defect.color.preset.red"), [3]float64{1, 0, 0}},
				{T("defect.color.preset.green"), [3]float64{0, 1, 0}},
				{T("defect.color.preset.blue"), [3]float64{0, 0, 1}},
				{T("defect.color.preset.white"), [3]float64{1, 1, 1}},
				{T("defect.color.preset.yellow"), [3]float64{1, 1, 0}},
				{T("defect.color.preset.cyan"), [3]float64{0, 1, 1}},
				{T("defect.color.preset.magenta"), [3]float64{1, 0, 1}},
			}
			for i, preset := range presets {
				if i > 0 {
					WrapSameLine(preset.label)
				}
				if Button(preset.label, fmt.Sprintf("defect-color-preset-%d", i)) {
					p.color = preset.col
				}
			}
			// Colour swatch with a sunken edge.
			sw := imgui.NewVec2(sc(56), imgui.FrameHeight())
			pos := imgui.CursorScreenPos()
			imgui.Dummy(sw)
			dl := imgui.WindowDrawList()
			FillRect(dl, pos, imgui.NewVec2(pos.X+sw.X, pos.Y+sw.Y),
				rgba(int(p.color[0]*255+0.5), int(p.color[1]*255+0.5), int(p.color[2]*255+0.5), 255))
			DrawEdge(dl, pos, imgui.NewVec2(pos.X+sw.X, pos.Y+sw.Y), EdgeSunken)
			Note(T("defect.color.weights", "r", F(p.color[0], 2), "g", F(p.color[1], 2), "b", F(p.color[2], 2)))
		}
		Note(T("defect.color.note"))
	}

	if p.mode == modeAuthor && Section(T("defect.section.amplitude"), "defect-amp") {
		amp := int(p.amplitude)
		if Combo(T("defect.section.amplitude"), "defect-ampmode", &amp, []string{
			T("defect.amplitude.absolute"), T("defect.amplitude.relative"), T("defect.amplitude.target"),
		}) {
			p.amplitude = core.DefectAmplitude(amp)
		}
		switch p.amplitude {
		case core.AmpRelative:
			SliderF64(T("defect.brush.relativePct"), "defect-relpct", &p.relativePct, 1, 300, "%.0f")
		case core.AmpTarget:
			SliderF64(T("defect.brush.targetBright"), "defect-tgb", &p.targetBright, 0, 255, "%.0f")
			SliderF64(T("defect.brush.targetDark"), "defect-tgd", &p.targetDark, 0, 255, "%.0f")
		default:
			SliderF64(T("defect.brush.deltaGv"), "defect-deltagv", &p.deltaGv, 1, 255, "%.0f")
		}
	}

	if p.mode == modeAuthor && Section(T("defect.section.profile"), "defect-profile") {
		pr := int(p.profile)
		if Combo(T("defect.section.profile"), "defect-profilemode", &pr, []string{
			T("defect.profile.linear"), T("defect.profile.gaussian"),
		}) {
			p.profile = core.DefectProfile(pr)
		}
		SliderF64(T("defect.brush.grain"), "defect-grain", &p.grain, 0, 1, "%.2f")
	}

	if p.mode == modeDetect && Section(T("defect.section.params"), "defect-params") {
		if Button(T("defect.params.open"), "defect-params-open") {
			p.paramsOpen = true
		}
		Note(T("defect.params.f1hint"))
	}

	if p.mode == modeAuthor && SectionClosed(T("defect.section.random"), "defect-random") {
		cnt := p.randomCount
		if SliderInt(T("defect.random.count"), "defect-rcount", &cnt, 1, 200) {
			p.randomCount = cnt
		}
		seed := int(p.randomSeed)
		if SliderInt(T("defect.random.seed"), "defect-rseed", &seed, 1, 9999) {
			p.randomSeed = uint32(seed)
		}
		rk := int(p.randomKind)
		if Combo(T("defect.random.kind"), "defect-rkind", &rk, []string{
			T("defect.random.kind.bright"), T("defect.random.kind.dark"), T("defect.random.kind.mixed"),
		}) {
			p.randomKind = core.RandomKind(rk)
		}
		rs := int(p.randomShape)
		if Combo(T("defect.random.shape"), "defect-rshape", &rs, []string{
			T("defect.random.shape.spot"), T("defect.random.shape.line"), T("defect.random.shape.mixed"),
		}) {
			p.randomShape = core.RandomShape(rs)
		}
		SliderF64(T("defect.random.radiusMin"), "defect-rrmin", &p.randomRadiusMin, 0.5, 40, "%.1f")
		SliderF64(T("defect.random.radiusMax"), "defect-rrmax", &p.randomRadiusMax, 0.5, 40, "%.1f")
		SliderF64(T("defect.random.valueMin"), "defect-rvmin", &p.randomValueMin, 1, 300, "%.0f")
		SliderF64(T("defect.random.valueMax"), "defect-rvmax", &p.randomValueMax, 1, 300, "%.0f")
		// Combo order is all, metal, sr, open, base; randomMaterial is a material index or -1.
		comboToMat := []int{-1, core.MatMetal, core.MatSR, core.MatOpen, core.MatBase}
		rm := 0
		for i, m := range comboToMat {
			if m == p.randomMaterial {
				rm = i
			}
		}
		if Combo(T("defect.random.material"), "defect-rmat", &rm, []string{
			T("defect.random.material.all"), "metal", "sr", "open", "base",
		}) {
			p.randomMaterial = comboToMat[rm]
		}
		if Button(T("defect.random.generate"), "defect-rgen") {
			p.generateRandom(p.randomSeed, p.randomCount)
			changed = true
		}
	}

	if p.mode == modeAuthor && Section(T("defect.section.noise"), "defect-noise") {
		rs := p.noise.ReadSigma
		if SliderF64(T("defect.noise.readSigma"), "defect-readsigma", &rs, 0, 20, "%.1f") {
			p.noise.ReadSigma = rs
			p.renderRev++
			changed = true
		}
		sg := p.noise.ShotGain
		if SliderF64(T("defect.noise.shotGain"), "defect-shotgain", &sg, 0, 2, "%.2f") {
			p.noise.ShotGain = sg
			p.renderRev++
			changed = true
		}
		ns := int(p.noise.Seed)
		if SliderInt(T("defect.noise.seed"), "defect-noiseseed", &ns, 1, 9999) {
			p.noise.Seed = uint32(ns)
			p.renderRev++
			changed = true
		}
		Note(T("defect.noise.note"))
	}

	if Section(T("defect.section.tools"), "defect-tools") {
		if p.mode == modeDetect && ToolButton(T("defect.tool.probe"), "defect-probe", p.tool == toolProbe) {
			p.tool = toolProbe
			a.SetHint(T("defect.tool.probeHint"))
		}
		vk := 0
		switch p.viewKind {
		case "golden":
			vk = 1
		case "diff":
			vk = 2
		}
		if Combo(T("defect.section.tools"), "defect-viewkind", &vk, []string{
			T("defect.view.test"), T("defect.view.golden"), T("defect.view.diff"),
		}) {
			switch vk {
			case 1:
				p.viewKind = "golden"
			case 2:
				p.viewKind = "diff"
			default:
				p.viewKind = "test"
			}
			if p.renderRev != p.renderedRev || p.golden == nil {
				p.requestRun()
			} else {
				p.requestView()
			}
		}
	}

	if changed {
		p.requestRun()
	}
}

// DrawDialogs draws the floating window of the mode: the selection editor in the
// studio (author mode) and the detection-parameter window (F1 toggles it) in the
// teaching program (detect mode).
func (p *DefectPanel) DrawDialogs() {
	if p.mode == modeAuthor {
		p.drawSelectionEditor()
		return
	}
	p.drawParamsDialog()
}

// drawParamsDialog draws the floating detection-parameter window (F1 toggles it).
func (p *DefectPanel) drawParamsDialog() {
	if imgui.IsKeyPressedBoolV(imgui.KeyF1, false) && !imgui.IsPopupOpenStrV("", imgui.PopupFlagsAnyPopupId) {
		p.paramsOpen = !p.paramsOpen
	}
	if p.paramsOpen && imgui.IsKeyPressedBool(imgui.KeyEscape) {
		p.paramsOpen = false
	}
	if !p.paramsOpen {
		return
	}
	vp := imgui.MainViewport()
	imgui.SetNextWindowPosV(imgui.NewVec2(vp.Pos().X+356, vp.Pos().Y+120), imgui.CondFirstUseEver, imgui.NewVec2(0, 0))
	imgui.SetNextWindowSizeV(imgui.NewVec2(sc(430), sc(560)), imgui.CondFirstUseEver)
	flags := imgui.WindowFlagsNoTitleBar | imgui.WindowFlagsNoCollapse | imgui.WindowFlagsNoSavedSettings |
		imgui.WindowFlagsNoScrollbar | imgui.WindowFlagsNoScrollWithMouse
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(10, 0))
	open := imgui.BeginV("##defect-params", nil, flags)
	imgui.PopStyleVar()
	if !open {
		imgui.End()
		return
	}
	wp, ws := imgui.WindowPos(), imgui.WindowSize()
	dl := imgui.WindowDrawList()
	DrawEdge(dl, wp, imgui.NewVec2(wp.X+ws.X, wp.Y+ws.Y), EdgeWindow)
	capMin := imgui.NewVec2(wp.X+3, wp.Y+3)
	capMax := imgui.NewVec2(wp.X+ws.X-3, wp.Y+3+sc(20))
	Caption(dl, capMin, capMax, T("defect.params.title"), true)
	imgui.SetCursorScreenPos(imgui.NewVec2(capMax.X-sc(18), capMin.Y+3))
	if ButtonSized("x", "defect-params-close", imgui.NewVec2(sc(16), sc(14)), true) {
		p.paramsOpen = false
	}

	imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+sc(10), wp.Y+sc(30)))
	imgui.BeginChildStrV("##defect-params-body", imgui.NewVec2(ws.X-sc(20), ws.Y-sc(40)), 0, 0)
	rerun := false
	ch := int(p.channel)
	params := &p.inspectParams[ch]
	Note(T("defect.insp.channel", "name", []string{"Red", "Green", "Blue"}[ch]))
	Note(T("defect.insp.explain"))
	SliderF64(T("defect.insp.learnK"), "insp-learnk", &p.learnK, 0.5, 12, "%.1f")
	if Button(T("defect.insp.relearn"), "insp-relearn") {
		p.requestRelearn()
		rerun = true
	}
	imgui.SameLine()
	if Button(T("defect.insp.reset"), "insp-reset") {
		p.resetInspect()
		rerun = true
	}
	Note(T("defect.insp.learnNote"))
	if Button(T("defect.insp.saveFile"), "insp-save") {
		p.saveParamsFile()
	}
	imgui.SameLine()
	if Button(T("defect.insp.loadFile"), "insp-load") && p.loadParamsFile() {
		rerun = true
	}
	Note(T("defect.insp.fileNote", "path", filepath.Join(p.sessionDir(), inspect.ParamsFileName)))
	p.inspMsg.draw()
	em := params.EdgeMargin
	if SliderInt(T("defect.insp.edgeMargin"), "insp-edge", &em, 0, inspect.MaxEdgeMargin) {
		for i := range p.inspectParams {
			p.inspectParams[i].EdgeMargin = em
		}
		p.learned = [3]bool{}
		rerun = true
	}
	Note(T("defect.insp.edgeNote"))
	if p.lastReport != nil && p.lastReport.EdgeIgnored > 0 {
		Note(T("defect.insp.edgeIgnored", "px", fmt.Sprint(p.lastReport.EdgeIgnored)))
	}
	bg := params.BackgroundRadius
	if SliderInt(T("defect.insp.bgRadius"), "insp-bg", &bg, 0, inspect.MaxBackgroundRadius) {
		for i := range p.inspectParams {
			p.inspectParams[i].BackgroundRadius = bg
		}
		p.learned = [3]bool{}
		rerun = true
	}
	Note(T("defect.insp.bgNote"))
	order := []inspect.Material{inspect.Metal, inspect.SR, inspect.Open, inspect.Base}
	for _, m := range order {
		mp := &params.Materials[m]
		if !Section(m.String(), fmt.Sprintf("insp-sec-%d", m)) {
			continue
		}
		if Check(T("defect.insp.enabled"), fmt.Sprintf("insp-en-%d", m), &mp.Enabled) {
			rerun = true
		}
		if SliderF64(T("defect.insp.low"), fmt.Sprintf("insp-low-%d", m), &mp.Low, 0, 255, "%.0f") {
			if mp.Low > mp.High {
				mp.High = mp.Low
			}
			rerun = true
		}
		if SliderF64(T("defect.insp.high"), fmt.Sprintf("insp-high-%d", m), &mp.High, 0, 255, "%.0f") {
			if mp.High < mp.Low {
				mp.Low = mp.High
			}
			rerun = true
		}
		sz := mp.Size
		if SliderInt(T("defect.insp.size"), fmt.Sprintf("insp-size-%d", m), &sz, 1, 64) {
			mp.Size = sz
			rerun = true
		}
		ma := mp.MinArea
		if SliderInt(T("defect.insp.minArea"), fmt.Sprintf("insp-minarea-%d", m), &ma, 1, 2000) {
			mp.MinArea = ma
			rerun = true
		}
		if Check(T("defect.insp.bgMaterial"), fmt.Sprintf("insp-bgm-%d", m), &mp.LocalBackground) {
			for i := range p.inspectParams {
				p.inspectParams[i].Materials[m].LocalBackground = mp.LocalBackground
			}
			p.learned = [3]bool{}
			rerun = true
		}
		if p.lastReport != nil {
			ms := p.lastReport.PerMaterial[m]
			if ms.Pixels > 0 {
				Note(T("defect.insp.stats",
					"pixels", fmt.Sprint(ms.Pixels),
					"mean", F(ms.Mean, 1),
					"std", F(ms.Std, 1),
					"min", fmt.Sprint(ms.Min),
					"max", fmt.Sprint(ms.Max)))
			}
		}
	}
	imgui.EndChild()
	if rerun {
		p.requestRun()
	}
	imgui.End()
}

func (p *DefectPanel) DrawRight() {
	if p.mode == modeAuthor {
		p.drawOutputPanel()
		return
	}
	a := p.a
	if Section(T("defect.section.result"), "defect-result") {
		if p.lastReport == nil {
			Note(T("defect.result.empty"))
		} else {
			r := p.lastReport
			rows := [][2]string{
				{T("defect.result.defectCount"), fmt.Sprint(len(r.Defects))},
				{T("defect.result.filteredCount"), fmt.Sprint(r.Filtered)},
				{T("defect.result.flaggedPixels"), fmt.Sprint(r.FlaggedPixels)},
			}
			KVTable("defect-kv", rows)

			params := p.inspectParams[p.channel]
			mHeaders := []string{
				T("defect.mcol.material"), T("defect.mcol.pixels"), T("defect.mcol.mean"), T("defect.mcol.std"),
				T("defect.mcol.min"), T("defect.mcol.max"), T("defect.mcol.range"),
				T("defect.mcol.bright"), T("defect.mcol.dark"),
			}
			mOrder := []inspect.Material{inspect.Metal, inspect.SR, inspect.Open, inspect.Base}
			DataTable("defect-mstats", mHeaders, len(mOrder), 5*imgui.FrameHeightWithSpacing()+8, func(row, col int) string {
				m := mOrder[row]
				ms := r.PerMaterial[m]
				mp := params.Materials[m]
				switch col {
				case 0:
					return m.String()
				case 1:
					return fmt.Sprint(ms.Pixels)
				case 2:
					return F(ms.Mean, 1)
				case 3:
					return F(ms.Std, 1)
				case 4:
					return fmt.Sprint(ms.Min)
				case 5:
					return fmt.Sprint(ms.Max)
				case 6:
					if !mp.Enabled {
						return "-"
					}
					return "[" + F(mp.Low, 0) + "," + F(mp.High, 0) + "]"
				case 7:
					if !mp.Enabled {
						return "-"
					}
					return fmt.Sprint(ms.Bright)
				}
				if !mp.Enabled {
					return "-"
				}
				return fmt.Sprint(ms.Dark)
			}, -1)

			headers := []string{
				T("defect.col.material"), T("defect.col.polarity"), T("defect.col.area"), T("defect.col.size"),
				T("defect.col.peakGv"), T("defect.col.excess"), T("defect.col.meanGv"), T("defect.col.fragments"),
			}
			defects := r.Defects
			n := len(defects)
			if n > maxResultRows {
				n = maxResultRows
			}
			clicked := DataTable("defect-blobs", headers, n, 12*imgui.FrameHeightWithSpacing()+8, func(row, col int) string {
				d := defects[row]
				switch col {
				case 0:
					return d.Material.String()
				case 1:
					if d.Polarity == inspect.Bright {
						return T("defect.polarity.bright")
					}
					return T("defect.polarity.dark")
				case 2:
					return fmt.Sprint(d.Area)
				case 3:
					return fmt.Sprintf("%dx%d", d.BBox.W, d.BBox.H)
				case 4:
					return fmt.Sprint(d.PeakGV)
				case 5:
					return F(d.Excess, 1)
				case 6:
					return F(d.MeanGV, 1)
				}
				return fmt.Sprint(d.Fragments)
			}, -1)
			if clicked >= 0 && clicked < n {
				a.Viewer.CenterOn(defects[clicked].CentroidX, defects[clicked].CentroidY)
			}
			if len(defects) > maxResultRows {
				Note(T("defect.result.truncated", "shown", maxResultRows, "total", len(defects)))
			}
		}
	}
	if Section(T("defect.section.probe"), "defect-probe-sec") {
		p.drawProbe()
	}
}

// drawProbe 渲染探查结果（逐材质行 + 教学解释）。
func (p *DefectPanel) drawProbe() {
	probe := p.lastProbe
	if probe == nil || probe.Rect.W == 0 {
		Note(T("defect.probe.empty"))
		return
	}
	Note(T("defect.probe.summary",
		"w", fmt.Sprint(probe.Rect.W),
		"h", fmt.Sprint(probe.Rect.H),
		"defects", fmt.Sprint(len(probe.Defects))))
	params := p.inspectParams[p.channel]
	flagged := 0
	for i := 0; i < inspect.NumMaterials; i++ {
		ps := probe.PerMaterial[i]
		if ps.Pixels == 0 {
			continue
		}
		m := inspect.Material(i)
		mp := params.Materials[i]
		if !mp.Enabled {
			Note(T("defect.probe.disabled", "m", m.String()))
			continue
		}
		flagged += ps.Bright + ps.Dark
		Note(T("defect.probe.row",
			"m", m.String(),
			"pixels", fmt.Sprint(ps.Pixels),
			"mean", F(ps.Mean, 1),
			"min", fmt.Sprint(ps.Min),
			"max", fmt.Sprint(ps.Max),
			"low", F(mp.Low, 0),
			"high", F(mp.High, 0),
			"bright", fmt.Sprint(ps.Bright),
			"dark", fmt.Sprint(ps.Dark)))
	}
	d := len(probe.Defects)
	switch {
	case flagged == 0:
		Note(T("defect.probe.allInside"))
	case d == 0:
		Note(T("defect.probe.filtered", "pixels", fmt.Sprint(flagged)))
	default:
		Note(T("defect.probe.detected", "count", fmt.Sprint(d)))
	}
	if p.lastProbe.Ignored > 0 {
		Note(T("defect.probe.ignored", "px", fmt.Sprint(p.lastProbe.Ignored)))
	}
}

func (p *DefectPanel) Overlay(o *Overlay) {
	if p.lastReport != nil {
		labels := 0
		boxes := 0
		for _, d := range p.lastReport.Defects {
			if boxes >= maxOverlayBoxes {
				break
			}
			if !o.Visible(float64(d.BBox.X), float64(d.BBox.Y), float64(d.BBox.W), float64(d.BBox.H)) {
				continue
			}
			col := overlayDarkCol
			if d.Polarity == inspect.Bright {
				col = overlayBrightCol
			}
			o.Rect(float64(d.BBox.X), float64(d.BBox.Y), float64(d.BBox.W), float64(d.BBox.H), col, 2)
			boxes++
			a, bb := o.ToScreen(float64(d.BBox.X), float64(d.BBox.Y)), o.ToScreen(float64(d.BBox.X+d.BBox.W), float64(d.BBox.Y+d.BBox.H))
			if bb.X-a.X >= 24 && labels < maxOverlayLabels {
				o.Text(float64(d.BBox.X), float64(d.BBox.Y), 2, -2, col, fmt.Sprintf("%s %d", d.Material.String(), d.Area))
				labels++
			}
		}
	}
	if p.showMarks {
		for _, e := range p.strokes {
			col := OverlayROI
			if !e.enabled {
				col = OverlayMuted
			} else if e.gv != nil && e.gv.HasColor {
				col = colorOutline(e.gv.Color)
			}
			if e.gv != nil {
				s := e.gv
				switch s.Shape {
				case core.ShapeSpot:
					o.Circle(s.X, s.Y, math.Max(s.Radius, 4/float64(o.Scale())), col, 1)
				case core.ShapeRect:
					o.Rect(math.Min(s.X, s.X2), math.Min(s.Y, s.Y2), math.Abs(s.X2-s.X), math.Abs(s.Y2-s.Y), col, 1)
				case core.ShapeEllipse:
					drawEllipseOutline(o, s.X, s.Y, s.X2, s.Y2, col, 1)
				case core.ShapePolygon:
					drawPolygonOutline(o, s.Points, col, 1)
				default:
					o.Line(s.X, s.Y, s.X2, s.Y2, col, 1)
				}
			} else if e.scene != nil {
				s := e.scene
				switch s.Shape {
				case core.ShapeSpot:
					o.Circle(s.X, s.Y, s.Radius, col, 1)
				case core.ShapeRect:
					o.Rect(math.Min(s.X, s.X2), math.Min(s.Y, s.Y2), math.Abs(s.X2-s.X), math.Abs(s.Y2-s.Y), col, 1)
				default:
					o.Line(s.X, s.Y, s.X2, s.Y2, col, 1)
				}
			}
		}
	}
	p.drawSelectionOverlay(o)
	if p.probeRect != nil {
		o.DashedRect(p.probeRect.X, p.probeRect.Y, p.probeRect.W, p.probeRect.H, OverlayProbe, 1.5)
	}
	if p.dragging {
		o.DragBox(p.dragStart.X, p.dragStart.Y, p.dragCur.X, p.dragCur.Y, OverlayROI)
	}
	if p.tool == toolFrame && p.frame != nil {
		mx, my, _ := p.a.Viewer.Mouse()
		p.frame.Draw(o, mx, my, OverlayROI)
	}
}

func (p *DefectPanel) Pointer(ev PointerEvent) bool {
	if p.tool == toolFrame && p.frame != nil {
		owned, res := p.frame.Pointer(ev)
		if res != nil {
			p.placeFrame(*res)
			p.tool = toolNone
		}
		if p.a != nil && p.a.Viewer != nil {
			p.a.Viewer.SetDoubleClickFit(!p.frame.Busy())
		}
		return owned
	}
	if p.tool == toolNone {
		if p.mode == modeDetect {
			return false
		}
		return p.selectPointer(ev)
	}
	switch ev.Kind {
	case PointerDown:
		p.dragging = true
		p.dragStart = core.Point{X: ev.X, Y: ev.Y}
		p.dragCur = p.dragStart
		return true
	case PointerMove:
		if p.dragging {
			p.dragCur = core.Point{X: ev.X, Y: ev.Y}
		}
		return true
	case PointerUp:
		if !p.dragging {
			return false
		}
		p.dragging = false
		x := math.Min(p.dragStart.X, p.dragCur.X)
		y := math.Min(p.dragStart.Y, p.dragCur.Y)
		w := math.Abs(p.dragCur.X - p.dragStart.X)
		h := math.Abs(p.dragCur.Y - p.dragStart.Y)
		dist := math.Hypot(p.dragCur.X-p.dragStart.X, p.dragCur.Y-p.dragStart.Y)
		switch p.tool {
		case toolProbe:
			if w >= 2 && h >= 2 {
				r := core.Rect{X: x, Y: y, W: w, H: h}
				p.probeRect = &r
				p.requestProbe(r)
			}
		case toolBrightRect, toolDarkRect:
			if w >= 2 && h >= 2 {
				p.placeRect(p.dragStart.X, p.dragStart.Y, p.dragCur.X, p.dragCur.Y)
			}
		case toolBrightSpot, toolDarkSpot, toolForeign, toolContamination:
			radius := p.radius
			if dist >= 2 {
				radius = dist
			}
			p.placeSpot(p.dragStart.X, p.dragStart.Y, radius)
		case toolBrightLine, toolDarkLine:
			if w >= 2 || h >= 2 {
				p.placeLine(p.dragStart.X, p.dragStart.Y, p.dragCur.X, p.dragCur.Y)
			}
		}
		p.tool = toolNone
		return true
	}
	return false
}

func (p *DefectPanel) Hover(x, y int) string {
	var img *core.GrayImage
	switch p.viewKind {
	case "golden":
		img = p.golden
	default:
		img = p.test
	}
	if img == nil || x < 0 || y < 0 || x >= img.Width || y >= img.Height {
		return ""
	}
	if p.viewKind == "diff" && p.golden != nil {
		d := int(img.Data[y*img.Width+x]) - int(p.golden.Data[y*p.golden.Width+x])
		return fmt.Sprintf("Δ %d", d)
	}
	return fmt.Sprintf("GV %d", img.Data[y*img.Width+x])
}
