package app

import (
	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/inspect"
)

const (
	defaultEdgeMargin       = 6 // pixels next to another material that Inspect ignores by default
	defaultBackgroundRadius = 2 // local background radius (pixels) of the default Inspect configuration; applies to Metal only
	defaultAutoCount        = core.DefaultTinyCount
	defaultAutoSeed         = core.DefaultTinySeed
)

// newInspectParams is the app's default Inspect configuration (library defaults plus the
// edge margin and the local background radius). The local background is enabled for Metal
// only: its brightness is bimodal, while SR/Open/Base are unimodal and work well with the
// plain per-material interval (the median follows hard-edged plateaus, so a drawn block on
// SR would be absorbed as background).
func newInspectParams() inspect.Params {
	p := inspect.DefaultParams()
	p.EdgeMargin = defaultEdgeMargin
	p.BackgroundRadius = defaultBackgroundRadius
	for i := range p.Materials {
		p.Materials[i].LocalBackground = inspect.Material(i) == inspect.Metal
	}
	return p
}

// freshParams is the default Inspect configuration of this panel. With a golden board the mask
// itself keeps the inspection away from the edges (its padding), so the edge margin is 0.
func (p *DefectPanel) freshParams() inspect.Params {
	prm := newInspectParams()
	if p.a != nil && p.a.golden != nil {
		prm.EdgeMargin = 0
	}
	return prm
}

// interiorSR reports whether the whole (2r+1)×(2r+1) window around (x,y) lies inside the
// image and every material value in it equals core.MatSR.
func interiorSR(scene *Scene, x, y, r int) bool {
	if scene == nil {
		return false
	}
	return core.InteriorMaterial(scene.Material, scene.W, scene.H, x, y, r, core.MatSR)
}

// addAutoDefects replaces the default tiny defects with a fresh batch placed deep inside SR.
func (p *DefectPanel) addAutoDefects() {
	p.removeAutoDefects()
	scene := p.a.Scene
	if scene == nil || p.autoCount <= 0 {
		return
	}
	strokes := core.DefaultTinyDefects(scene.Material, scene.W, scene.H, p.autoCount, p.autoSeed, defaultEdgeMargin)
	for i := range strokes {
		stroke := strokes[i]
		p.strokes = append(p.strokes, strokeEntry{gv: &stroke, enabled: true, auto: true})
	}
	p.renderRev++
}

// removeAutoDefects drops every default tiny defect from the stroke list and the selection.
func (p *DefectPanel) removeAutoDefects() {
	if len(p.strokes) == 0 {
		return
	}
	kept := p.strokes[:0]
	for i := range p.strokes {
		if p.strokes[i].auto {
			continue
		}
		kept = append(kept, p.strokes[i])
	}
	p.strokes = kept
	p.pruneSelection()
	p.renderRev++
}

// drawVirtualSection draws the virtual-product section; it reports whether anything changed.
func (p *DefectPanel) drawVirtualSection() bool {
	if !Section(T("defect.section.virtual"), "defect-virtual") {
		return false
	}
	changed := false
	Note(T("defect.virtual.note"))

	if Check(T("defect.virtual.enable"), "virt-enable", &p.productOpts.Enabled) {
		changed = true
	}
	if SliderF64(T("defect.virtual.strength"), "virt-strength", &p.productOpts.Strength, 0, 3, "%.2f") {
		changed = true
	}
	seed := int(p.productOpts.Seed)
	if SliderInt(T("defect.virtual.seed"), "virt-seed", &seed, 1, 9999) {
		p.productOpts.Seed = uint32(seed)
		changed = true
	}
	if SliderF64(T("defect.virtual.length"), "virt-length", &p.productLenScale, 0.25, 4, "%.2f") {
		defaults := core.DefaultProductOptions()
		for i := 0; i < 4; i++ {
			p.productOpts.Materials[i].Length = defaults.Materials[i].Length * p.productLenScale
		}
		changed = true
	}
	if Button(T("defect.virtual.newBoard"), "virt-newboard") {
		p.productDraw++
		p.productOpts.Seed = uint32(1 + (p.productDraw*7919+int(p.productOpts.Seed)*31)%9999)
		changed = true
	}
	SameLine()
	if Button(T("defect.virtual.reset"), "virt-reset") {
		p.productOpts = core.DefaultProductOptions()
		p.productLenScale = 1
		changed = true
	}
	if changed {
		p.a.ApplyProduct(p.productOpts)
	}

	autoCount := p.autoCount
	if SliderInt(T("defect.virtual.autoCount"), "virt-autocount", &autoCount, 0, 50) {
		p.autoCount = autoCount
	}
	autoSeed := int(p.autoSeed)
	if SliderInt(T("defect.virtual.autoSeed"), "virt-autoseed", &autoSeed, 1, 9999) {
		p.autoSeed = uint32(autoSeed)
	}
	if Button(T("defect.virtual.autoRegen"), "virt-autoregen") {
		p.addAutoDefects()
		changed = true
	}
	SameLine()
	if Button(T("defect.virtual.autoRemove"), "virt-autoremove") {
		p.removeAutoDefects()
		changed = true
	}
	Check(T("defect.virtual.marks"), "virt-marks", &p.showMarks)
	Note(T("defect.virtual.autoNote"))
	return changed
}

func init() {
	Messages(map[string]string{
		"defect.section.virtual":    "虚拟产品",
		"defect.virtual.note":       "MEDIAN 是多块良品的中值，过于平滑。虚拟良品在其上叠加像素级表面不均匀（约 2 µm/像素，再好的表面工艺也不可能完全一致），所有页面共用。「纹理强度」为 0 即理想均匀样品。",
		"defect.virtual.enable":     "启用虚拟良品",
		"defect.virtual.strength":   "纹理强度",
		"defect.virtual.seed":       "纹理种子",
		"defect.virtual.length":     "相关长度倍率",
		"defect.virtual.newBoard":   "换一块板",
		"defect.virtual.reset":      "恢复默认",
		"defect.virtual.autoCount":  "默认不良数量",
		"defect.virtual.autoSeed":   "默认不良种子",
		"defect.virtual.autoRegen":  "重新生成默认不良",
		"defect.virtual.autoRemove": "移除默认不良",
		"defect.virtual.marks":      "显示模拟笔画轮廓",
		"defect.virtual.autoNote":   "加载模型时会自动加入约 10 个不大的亮点（SR 内、半径约 1.5–3 像素），肉眼几乎看不到，但 Inspect 能检出；它们是普通的模拟不良，可用右侧悬浮工具选中后调参。",
		"defect.insp.edgeMargin":    "边缘忽略（像素）",
		"defect.insp.edgeNote":      "与不同材质相距不超过该像素数的像素不参与统计、学习与检出（机台的 Margine）。修改后会重新学习区间；设为 0 可看到材质交界处的大量过检。",
		"defect.insp.edgeIgnored":   "被忽略的边缘像素：{px}",
		"defect.insp.bgRadius":      "局部背景半径（像素）",
		"defect.insp.bgNote":        "0 = 关闭，区间作用于原始 GV。大于 0 时，勾选了\"使用局部背景\"的材质先减去同材质 (2r+1)² 邻域的中位数（局部背景）再套区间：亮度不均匀的材质（金属）不会整块报成巨大缺陷，过检呈零散小点；但只检出比窗口小的局部异常，大块平整的缺陷会被当作背景，所以默认只对金属开启。修改后会重新学习区间。",
		"defect.insp.bgMaterial":    "使用局部背景",
		"defect.probe.ignored":      "框内被边缘忽略的像素：{px}",
	}, map[string]string{
		"defect.section.virtual":    "Virtual product",
		"defect.virtual.note":       "MEDIAN is the median of several good boards and is too smooth. The virtual good unit adds pixel-scale surface non-uniformity on top of it (about 2 µm/pixel; even the best surface process cannot be perfectly uniform), shared by every page. A texture strength of 0 is an ideal uniform sample.",
		"defect.virtual.enable":     "Enable virtual good unit",
		"defect.virtual.strength":   "Texture strength",
		"defect.virtual.seed":       "Texture seed",
		"defect.virtual.length":     "Correlation length scale",
		"defect.virtual.newBoard":   "New board",
		"defect.virtual.reset":      "Reset",
		"defect.virtual.autoCount":  "Default defect count",
		"defect.virtual.autoSeed":   "Default defect seed",
		"defect.virtual.autoRegen":  "Regenerate default defects",
		"defect.virtual.autoRemove": "Remove default defects",
		"defect.virtual.marks":      "Show simulated stroke outlines",
		"defect.virtual.autoNote":   "Loading a model adds about 10 small bright specks (inside SR, radius about 1.5–3 pixels) that are almost invisible to the eye but detectable by Inspect; they are ordinary simulated defects and can be selected with the floating tool on the right to tune their parameters.",
		"defect.insp.edgeMargin":    "Edge margin (pixels)",
		"defect.insp.edgeNote":      "Pixels within this distance of a different material take no part in statistics, learning or detection (the machine's Margine). Changing it re-learns the ranges; set it to 0 to see the many false defects at material boundaries.",
		"defect.insp.edgeIgnored":   "Ignored edge pixels: {px}",
		"defect.insp.bgRadius":      "Local background radius (pixels)",
		"defect.insp.bgNote":        "0 = off, the interval applies to the raw GV. Above 0, materials that have \"Use local background\" ticked first lose the median of their same-material (2r+1)² neighbourhood (local background) before the interval is applied: a non-uniform material (metal) is no longer reported as one giant defect and over-kill shows up as scattered dots, but only anomalies smaller than the window are found and large flat defects count as background, so it is on for metal only by default. Changing it re-learns the ranges.",
		"defect.insp.bgMaterial":    "Use local background",
		"defect.probe.ignored":      "Pixels ignored by the edge margin in the box: {px}",
	}, map[string]string{
		"defect.section.virtual":    "가상 제품",
		"defect.virtual.note":       "MEDIAN은 여러 양품 보드의 중앙값이라 너무 매끄럽습니다. 가상 양품은 그 위에 픽셀 단위의 표면 불균일을 더합니다(약 2 µm/픽셀, 아무리 좋은 표면 공정도 완전히 균일할 수 없습니다). 모든 페이지가 공유합니다. 텍스처 강도가 0이면 이상적으로 균일한 샘플입니다.",
		"defect.virtual.enable":     "가상 양품 사용",
		"defect.virtual.strength":   "텍스처 강도",
		"defect.virtual.seed":       "텍스처 시드",
		"defect.virtual.length":     "상관 길이 배율",
		"defect.virtual.newBoard":   "다른 보드",
		"defect.virtual.reset":      "기본값 복원",
		"defect.virtual.autoCount":  "기본 불량 개수",
		"defect.virtual.autoSeed":   "기본 불량 시드",
		"defect.virtual.autoRegen":  "기본 불량 다시 생성",
		"defect.virtual.autoRemove": "기본 불량 제거",
		"defect.virtual.marks":      "모의 획 윤곽 표시",
		"defect.virtual.autoNote":   "모델을 로드하면 약 10개의 작은 밝은 점(SR 내부, 반경 약 1.5–3 픽셀)이 자동으로 추가됩니다. 육안으로는 거의 보이지 않지만 Inspect는 검출할 수 있습니다. 일반적인 모의 불량이며 오른쪽 떠 있는 도구로 선택해 파라미터를 조정할 수 있습니다.",
		"defect.insp.edgeMargin":    "가장자리 무시(픽셀)",
		"defect.insp.edgeNote":      "다른 재질과의 거리가 이 픽셀 수 이내인 픽셀은 통계, 학습, 검출에 참여하지 않습니다(기계의 Margine). 변경하면 구간을 다시 학습합니다. 0으로 설정하면 재질 경계에서 많은 과검출을 볼 수 있습니다.",
		"defect.insp.edgeIgnored":   "무시된 가장자리 픽셀: {px}",
		"defect.insp.bgRadius":      "국소 배경 반경(픽셀)",
		"defect.insp.bgNote":        "0 = 끔, 구간이 원본 GV에 적용됩니다. 0보다 크면 \"국소 배경 사용\"을 켠 재질은 같은 재질의 (2r+1)² 이웃의 중앙값(국소 배경)을 먼저 뺀 뒤 구간을 적용합니다. 밝기가 불균일한 재질(금속)이 하나의 거대한 결함으로 보고되지 않고 과검출은 흩어진 점으로 나타나지만, 창보다 작은 이상만 찾으며 크고 평평한 결함은 배경으로 취급되므로 기본적으로 금속에만 켜져 있습니다. 변경하면 구간을 다시 학습합니다.",
		"defect.insp.bgMaterial":    "국소 배경 사용",
		"defect.probe.ignored":      "박스 안에서 가장자리로 무시된 픽셀: {px}",
	})
}
