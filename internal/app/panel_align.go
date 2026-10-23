package app

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/ui"
)

func init() {
	registerPanel(2, func(a *App) Panel {
		return &AlignPanel{
			a:         a,
			scale:     2,
			source:    "median",
			viewKind:  "overlay",
			tool:      "none",
			selected:  -1,
			autoRun:   true,
			dirty:     true,
			params:    defaultWarpParams(),
			config:    core.DefaultAlignConfig,
			occValue:  0,
			lastScale: 2,
			tplSource: tplDesign,
		}
	}, ProfileTeach)
	Messages(map[string]string{
		"align.section.source":         "源图",
		"align.source.median":          "MEDIAN 原图",
		"align.source.light":           "当前光照渲染图",
		"align.template":               "模板图",
		"align.template.design":        "设计假想图（仅形状）",
		"align.template.image":         "与源图相同（旧）",
		"align.template.designNote":    "ROI 的模板从设计材料层生成的假想图上切取：没有真实纹理和噪声，只能靠形状去匹配，更接近实机。",
		"align.template.imageNote":     "模板和输入是同一张图（只是扭曲了），ROI 画在哪里都能完美匹配，仅用于对照。",
		"align.template.registered":    "假想图已与源图配准：整体偏移 ({dx}, {dy}) px，旋转 {deg}°（设计层与实拍图的固有差异）。",
		"align.template.unregistered":  "假想图没能配准到源图（可用的匹配窗口不足），ROI 的位置会带着设计层与实拍图的固有偏差。",
		"align.section.warp":           "扭曲参数",
		"align.param.tx":               "tx",
		"align.param.ty":               "ty",
		"align.param.theta":            "旋转°",
		"align.param.scale":            "缩放",
		"align.param.shearX":           "剪切 X",
		"align.param.shearY":           "剪切 Y",
		"align.param.bendAmplitude":    "弯曲幅度",
		"align.param.bendWavelength":   "弯曲波长",
		"align.param.blurSigma":        "模糊 σ",
		"align.param.noiseSigma":       "噪声 σ",
		"align.param.gain":             "增益",
		"align.param.offset":           "偏移",
		"align.param.gradient":         "光照梯度",
		"align.param.seed":             "随机种子",
		"align.resetAll":               "全部归零",
		"align.section.occlusion":      "遮挡",
		"align.occlusion.draw":         "画遮挡（拖框）",
		"align.occlusion.hint":         "在模板视图拖框添加遮挡",
		"align.occlusion.fill":         "填充值",
		"align.occlusion.empty":        "尚无遮挡。",
		"align.occlusion.delete":       "删除",
		"align.section.match":          "匹配参数",
		"align.match.searchRadius":     "搜索半径",
		"align.match.pyramid1":         "金字塔 1",
		"align.match.pyramid2":         "金字塔 2",
		"align.match.pyramid4":         "金字塔 4",
		"align.match.pyramid8":         "金字塔 8",
		"align.match.minNcc":           "最小 NCC",
		"align.match.minUniqueness":    "最小唯一性",
		"align.match.ransac":           "RANSAC 阈值",
		"align.match.okResidual":       "OK 残差",
		"align.match.ngResidual":       "NG 残差",
		"align.section.roi":            "ROI",
		"align.roi.restore":            "恢复机台定位点",
		"align.roi.new":                "新建 ROI（拖框）",
		"align.roi.newHint":            "在模板视图拖框新建 ROI",
		"align.roi.select":             "选择/移动",
		"align.roi.selectHint":         "拖动 ROI 移动，Delete 删除选中",
		"align.roi.empty":              "尚无 ROI。",
		"align.section.action":         "执行",
		"align.action.generate":        "生成输入图",
		"align.action.run":             "执行对齐",
		"align.action.autoRun":         "参数变化后自动执行",
		"align.view.template":          "模板",
		"align.view.input":             "输入",
		"align.view.diff":              "差异",
		"align.section.verdict":        "判定",
		"align.section.ncc":            "NCC 曲面",
		"align.result.empty":           "尚未执行对齐。",
		"align.result.fitFailed":       "失败",
		"align.result.fit":             "拟合",
		"align.result.truthMean":       "真值误差 mean",
		"align.result.truthMax":        "真值误差 max",
		"align.result.residualRms":     "内点残差 RMS",
		"align.result.meanNcc":         "平均 NCC",
		"align.result.elapsed":         "耗时",
		"align.col.roi":                "ROI",
		"align.col.ncc":                "NCC",
		"align.col.uniqueness":         "唯一性",
		"align.col.located":            "定位",
		"align.col.residual":           "残差",
		"align.col.reason":             "原因",
		"align.yes":                    "是",
		"align.no":                     "否",
		"align.needRois":               "至少需要 2 个 ROI",
		"align.currentGray":            "对齐 {kind}",
		"align.reason.too-few-located": "定位成功的 ROI 只有 {n} 个，至少需要 2 个",
		"align.reason.no-fit":          "无法拟合相似变换",
		"align.reason.residual-high":   "内点残差 RMS {rms}px 超过 NG 阈值 {th}px",
		"align.reason.outlier":         "存在定位成功但被判定为外点的 ROI",
		"align.reason.ambiguous":       "存在唯一性不足的 ROI，可能是重复纹理",
		"align.reason.residual-warn":   "内点残差 RMS {rms}px 超过 OK 阈值 {th}px",
		"align.reason.roi-missing":     "存在未定位的 ROI",
		"align.match.out-of-range":     "超出范围",
		"align.match.flat":             "纹理平坦",
		"align.match.low-ncc":          "NCC 过低",
		"align.run":                    "运行对齐",
		"align.autoRun":                "自动运行",
		"align.view":                   "视图",
		"align.workScale":              "工作分辨率",
		"align.match.pyramid":          "金字塔",
		"tab.align":                    "对位试验台",
	}, map[string]string{
		"align.section.source":         "Source",
		"align.source.median":          "MEDIAN original",
		"align.source.light":           "Current lighting render",
		"align.template":               "Template",
		"align.template.design":        "Design picture (shape only)",
		"align.template.image":         "Same as source (old)",
		"align.template.designNote":    "ROI templates are cut from an imagined picture made from the design material layers: no real texture or noise, matching can only use shapes, like on the machine.",
		"align.template.imageNote":     "Template and input are the same picture (only warped), so any ROI matches perfectly; for comparison only.",
		"align.template.registered":    "The design picture was registered to the source: overall shift ({dx}, {dy}) px, rotation {deg}° (the inherent difference between design layers and the real picture).",
		"align.template.unregistered":  "The design picture could not be registered to the source (too few usable windows), so ROI positions carry the inherent offset between the design layers and the real picture.",
		"align.section.warp":           "Warp Parameters",
		"align.param.tx":               "tx",
		"align.param.ty":               "ty",
		"align.param.theta":            "Rotation°",
		"align.param.scale":            "Scale",
		"align.param.shearX":           "Shear X",
		"align.param.shearY":           "Shear Y",
		"align.param.bendAmplitude":    "Bend amplitude",
		"align.param.bendWavelength":   "Bend wavelength",
		"align.param.blurSigma":        "Blur σ",
		"align.param.noiseSigma":       "Noise σ",
		"align.param.gain":             "Gain",
		"align.param.offset":           "Offset",
		"align.param.gradient":         "Lighting gradient",
		"align.param.seed":             "Random seed",
		"align.resetAll":               "Reset All",
		"align.section.occlusion":      "Occlusion",
		"align.occlusion.draw":         "Draw occlusion (drag box)",
		"align.occlusion.hint":         "Drag a box on the template view to add an occlusion",
		"align.occlusion.fill":         "Fill value",
		"align.occlusion.empty":        "No occlusions yet.",
		"align.occlusion.delete":       "Delete",
		"align.section.match":          "Matching Parameters",
		"align.match.searchRadius":     "Search range",
		"align.match.pyramid1":         "Pyramid 1",
		"align.match.pyramid2":         "Pyramid 2",
		"align.match.pyramid4":         "Pyramid 4",
		"align.match.pyramid8":         "Pyramid 8",
		"align.match.minNcc":           "Min NCC",
		"align.match.minUniqueness":    "Min uniqueness",
		"align.match.ransac":           "RANSAC threshold",
		"align.match.okResidual":       "OK residual",
		"align.match.ngResidual":       "NG residual",
		"align.section.roi":            "ROI",
		"align.roi.restore":            "Restore machine anchors",
		"align.roi.new":                "New ROI (drag box)",
		"align.roi.newHint":            "Drag a box on the template view to create an ROI",
		"align.roi.select":             "Select/Move",
		"align.roi.selectHint":         "Drag an ROI to move it, press Delete to remove the selected one",
		"align.roi.empty":              "No ROIs yet.",
		"align.section.action":         "Run",
		"align.action.generate":        "Generate input image",
		"align.action.run":             "Run alignment",
		"align.action.autoRun":         "Auto run on parameter change",
		"align.view.template":          "Template",
		"align.view.input":             "Input",
		"align.view.diff":              "Difference",
		"align.section.verdict":        "Verdict",
		"align.section.ncc":            "NCC Surface",
		"align.result.empty":           "Alignment has not been run yet.",
		"align.result.fitFailed":       "Failed",
		"align.result.fit":             "Fit",
		"align.result.truthMean":       "Truth error mean",
		"align.result.truthMax":        "Truth error max",
		"align.result.residualRms":     "Inlier residual RMS",
		"align.result.meanNcc":         "Mean NCC",
		"align.result.elapsed":         "Elapsed",
		"align.col.roi":                "ROI",
		"align.col.ncc":                "NCC",
		"align.col.uniqueness":         "Uniqueness",
		"align.col.located":            "Located",
		"align.col.residual":           "Residual",
		"align.col.reason":             "Reason",
		"align.yes":                    "Yes",
		"align.no":                     "No",
		"align.needRois":               "At least 2 ROIs are required",
		"align.currentGray":            "Align {kind}",
		"align.reason.too-few-located": "Only {n} ROIs were located, at least 2 are required",
		"align.reason.no-fit":          "Could not fit a similarity transform",
		"align.reason.residual-high":   "Inlier residual RMS {rms}px exceeds the NG threshold {th}px",
		"align.reason.outlier":         "A located ROI was classified as an outlier",
		"align.reason.ambiguous":       "An ROI has insufficient uniqueness, possibly a repeated texture",
		"align.reason.residual-warn":   "Inlier residual RMS {rms}px exceeds the OK threshold {th}px",
		"align.reason.roi-missing":     "Some ROIs were not located",
		"align.match.out-of-range":     "Out of range",
		"align.match.flat":             "Flat texture",
		"align.match.low-ncc":          "NCC too low",
		"align.run":                    "Run alignment",
		"align.autoRun":                "Auto run",
		"align.view":                   "View",
		"align.workScale":              "Working scale",
		"align.match.pyramid":          "Pyramid",
		"tab.align":                    "Align bench",
	}, map[string]string{
		"align.section.source":         "소스",
		"align.source.median":          "MEDIAN 원본",
		"align.source.light":           "현재 조명 렌더링",
		"align.template":               "템플릿",
		"align.template.design":        "설계 가상 이미지 (형상만)",
		"align.template.image":         "소스와 동일 (이전)",
		"align.template.designNote":    "ROI 템플릿은 설계 재료 레이어로 생성한 가상 이미지에서 잘라냅니다. 실제 텍스처나 노이즈가 없어 형상만으로 매칭하므로 실기기에 더 가깝습니다.",
		"align.template.imageNote":     "템플릿과 입력이 같은 이미지(왜곡만 적용)이므로 어떤 ROI 도 완벽하게 매칭됩니다. 비교용으로만 사용하십시오.",
		"align.template.registered":    "가상 이미지가 소스에 정합되었습니다. 전체 이동 ({dx}, {dy}) px, 회전 {deg}° (설계 레이어와 실사 이미지의 고유 차이).",
		"align.template.unregistered":  "가상 이미지를 소스에 정합하지 못했습니다(사용 가능한 매칭 창이 부족함). 따라서 ROI 위치에는 설계 레이어와 실사 이미지 사이의 고유한 어긋남이 남습니다.",
		"align.section.warp":           "왜곡 파라미터",
		"align.param.tx":               "tx",
		"align.param.ty":               "ty",
		"align.param.theta":            "회전°",
		"align.param.scale":            "스케일",
		"align.param.shearX":           "전단 X",
		"align.param.shearY":           "전단 Y",
		"align.param.bendAmplitude":    "굽힘 진폭",
		"align.param.bendWavelength":   "굽힘 파장",
		"align.param.blurSigma":        "블러 σ",
		"align.param.noiseSigma":       "노이즈 σ",
		"align.param.gain":             "게인",
		"align.param.offset":           "오프셋",
		"align.param.gradient":         "조명 그라디언트",
		"align.param.seed":             "랜덤 시드",
		"align.resetAll":               "모두 초기화",
		"align.section.occlusion":      "가림",
		"align.occlusion.draw":         "가림 그리기 (드래그 박스)",
		"align.occlusion.hint":         "템플릿 뷰에서 박스를 드래그하여 가림을 추가합니다",
		"align.occlusion.fill":         "채움 값",
		"align.occlusion.empty":        "아직 가림이 없습니다.",
		"align.occlusion.delete":       "삭제",
		"align.section.match":          "매칭 파라미터",
		"align.match.searchRadius":     "탐색 범위",
		"align.match.pyramid1":         "피라미드 1",
		"align.match.pyramid2":         "피라미드 2",
		"align.match.pyramid4":         "피라미드 4",
		"align.match.pyramid8":         "피라미드 8",
		"align.match.minNcc":           "최소 NCC",
		"align.match.minUniqueness":    "최소 고유성",
		"align.match.ransac":           "RANSAC 임계값",
		"align.match.okResidual":       "OK 잔차",
		"align.match.ngResidual":       "NG 잔차",
		"align.section.roi":            "ROI",
		"align.roi.restore":            "기계 기준점 복원",
		"align.roi.new":                "ROI 생성 (드래그 박스)",
		"align.roi.newHint":            "템플릿 뷰에서 박스를 드래그하여 ROI 를 생성합니다",
		"align.roi.select":             "선택/이동",
		"align.roi.selectHint":         "ROI 를 드래그하여 이동, Delete 로 선택 항목 삭제",
		"align.roi.empty":              "아직 ROI 가 없습니다.",
		"align.section.action":         "실행",
		"align.action.generate":        "입력 이미지 생성",
		"align.action.run":             "정렬 실행",
		"align.action.autoRun":         "파라미터 변경 시 자동 실행",
		"align.view.template":          "템플릿",
		"align.view.input":             "입력",
		"align.view.diff":              "차이",
		"align.section.verdict":        "판정",
		"align.section.ncc":            "NCC 곡면",
		"align.result.empty":           "아직 정렬을 실행하지 않았습니다.",
		"align.result.fitFailed":       "실패",
		"align.result.fit":             "피팅",
		"align.result.truthMean":       "정답 오차 mean",
		"align.result.truthMax":        "정답 오차 max",
		"align.result.residualRms":     "인라이어 잔차 RMS",
		"align.result.meanNcc":         "평균 NCC",
		"align.result.elapsed":         "소요",
		"align.col.roi":                "ROI",
		"align.col.ncc":                "NCC",
		"align.col.uniqueness":         "고유성",
		"align.col.located":            "위치",
		"align.col.residual":           "잔차",
		"align.col.reason":             "원인",
		"align.yes":                    "예",
		"align.no":                     "아니오",
		"align.needRois":               "ROI 가 최소 2 개 필요합니다",
		"align.currentGray":            "정렬 {kind}",
		"align.reason.too-few-located": "위치를 찾은 ROI 가 {n} 개뿐이며 최소 2 개가 필요합니다",
		"align.reason.no-fit":          "유사 변환을 피팅할 수 없습니다",
		"align.reason.residual-high":   "인라이어 잔차 RMS {rms}px 가 NG 임계값 {th}px 를 초과했습니다",
		"align.reason.outlier":         "위치를 찾았지만 외점으로 판정된 ROI 가 있습니다",
		"align.reason.ambiguous":       "고유성이 부족한 ROI 가 있습니다. 반복 텍스처일 수 있습니다",
		"align.reason.residual-warn":   "인라이어 잔차 RMS {rms}px 가 OK 임계값 {th}px 를 초과했습니다",
		"align.reason.roi-missing":     "위치를 찾지 못한 ROI 가 있습니다",
		"align.match.out-of-range":     "범위 초과",
		"align.match.flat":             "평탄한 텍스처",
		"align.match.low-ncc":          "NCC 부족",
		"align.run":                    "정렬 실행",
		"align.autoRun":                "자동 실행",
		"align.view":                   "뷰",
		"align.workScale":              "작업 해상도",
		"align.match.pyramid":          "피라미드",
		"tab.align":                    "정렬 시험대",
	})
}

// defaultWarpParams mirrors the TS initial WarpParams (Scale 1, Gain 1, rest 0).
// matchOpsPerSecond calibrates core.MatchCost to seconds: pyramid 1 with a search radius of 230 on the
// reference model (8e10 operations) took 34 s on a 16-thread machine. Only used for the warning below.
const (
	matchOpsPerSecond = 2e9
	slowRunSeconds    = 5.0
)

const (
	tplDesign = "design" // imagined picture built from the design material layers only (shape, no texture): the default
	tplImage  = "image"  // the source picture itself: every ROI trivially matches (the old behaviour)
)

// designTemplate is the cached imagined picture for one (scene, source, scale, light, channel).
type designTemplate struct {
	scene   *Scene
	source  string
	scale   int
	light   core.LightSetting  // only with source "light"
	channel core.CameraChannel // only with source "light"
	img     *core.GrayImage
	info    core.DesignInfo
}

func defaultWarpParams() core.WarpParams {
	return core.WarpParams{Scale: 1, Gain: 1}
}

// AlignPanel ports teach-app/src/align-panel.ts to native ImGui.
type AlignPanel struct {
	a *App

	rois      []core.AlignRoi
	occs      []core.Occlusion
	selected  int
	tool      string // "none" | "roi" | "select" | "occlusion"
	viewKind  string // "overlay" | "template" | "input" | "diff" | "aligned" | "alignedOverlay" | "alignedDiff"
	source    string // "median" | "light"
	tplSource string // tplDesign | tplImage
	scale     int
	params    core.WarpParams
	config    core.AlignConfig
	autoRun   bool
	dirty     bool
	occValue  float64
	active    bool
	lastScale int

	template   *core.GrayImage
	input      *core.GrayImage
	reference  *core.GrayImage // the real un-warped picture (working resolution) the input was warped from; the whole-picture checks measure against it. The template itself with the "image" template
	design     *designTemplate // cached imagined picture for the current (scene, source, scale, light)
	designInfo core.DesignInfo // info of the design the shown template was built from

	report       *core.AlignReport
	truth        *core.TruthComparison
	elapsedMs    float64
	heat         *core.NccSurface
	reportConfig core.AlignConfig
	reportParams core.WarpParams // warp (working resolution) and template size the report was computed for
	reportW      int
	reportH      int
	reportKey    alignKey            // everything the report was computed from; stale when it differs from inputKey()
	diag         []core.RoiDiagnosis // per ROI: match error, deviation from the fit, border overhang, occlusion
	fitInput     *core.GrayImage     // the input image the report's fit was computed for; alignFit is nil once input differs
	aligned      *alignedResult      // the input brought onto the template grid with report.Fit

	hoverRoi bool // the cursor is over a ROI body (select tool claims the drag only there)

	dragging  bool
	dragStart core.Point
	dragCur   core.Point
	moveIdx   int
	moveOrig  core.Point
}

func (p *AlignPanel) ID() string    { return "align" }
func (p *AlignPanel) Title() string { return T("tab.align") }

// PaletteSupports: the align panel uses the pick (select/move) and rect (ROI/occlusion) tools.
func (p *AlignPanel) PaletteSupports(t PaletteTool) bool {
	return t == PalettePick || t == PaletteRect
}

// PaletteTool maps the panel's internal tool string to a palette tool.
func (p *AlignPanel) PaletteTool() PaletteTool {
	switch p.tool {
	case "select":
		return PalettePick
	case "roi", "occlusion":
		return PaletteRect
	}
	return PaletteNone
}

// SetPaletteTool mirrors the left-panel tool buttons: pick selects/moves ROIs, rect keeps
// the current ROI/occlusion mode (defaulting to ROI), none returns to pan.
func (p *AlignPanel) SetPaletteTool(t PaletteTool) {
	switch t {
	case PalettePick:
		p.tool = "select"
		p.a.SetHint(T("align.roi.selectHint"))
	case PaletteRect:
		if p.tool != "roi" && p.tool != "occlusion" {
			p.tool = "roi"
			p.a.SetHint(T("align.roi.newHint"))
		}
	case PaletteNone:
		p.tool = "none"
	}
}

func (p *AlignPanel) Activate() {
	p.active = true
	if p.dirty || p.template == nil {
		p.requestRun(true)
	} else {
		p.requestView()
	}
}

func (p *AlignPanel) Deactivate() {
	p.active = false
	p.dragging = false
}

func (p *AlignPanel) ModelLoaded() {
	p.rois = cloneRois(p.a.Scene.MachineRois)
	p.occs = nil
	p.selected = -1
	p.report = nil
	p.truth = nil
	p.heat = nil
	p.template = nil
	p.input = nil
	p.reference = nil
	p.design = nil
	p.designInfo = core.DesignInfo{}
	p.dirty = true
}

func (p *AlignPanel) LightChanged() {
	if p.source == "light" {
		p.dirty = true
		if p.active && p.autoRun {
			p.requestRun(true)
		}
	}
}

func (p *AlignPanel) Selftest() {
	if len(p.rois) < 2 {
		p.rois = cloneRois(p.a.Scene.MachineRois)
	}
	p.requestRun(true)
}

func cloneRois(in []core.AlignRoi) []core.AlignRoi {
	out := make([]core.AlignRoi, len(in))
	copy(out, in)
	return out
}

// scaledParams divides the full-resolution parameters by the working scale.
func (p *AlignPanel) scaledParams() core.WarpParams {
	s := float64(p.scale)
	out := p.params
	out.Tx /= s
	out.Ty /= s
	out.BendAmplitude /= s
	out.BendWavelength /= s
	out.BlurSigma /= s
	out.Occlusions = make([]core.Occlusion, len(p.occs))
	for i, o := range p.occs {
		out.Occlusions[i] = core.Occlusion{
			Rect:  core.Rect{X: o.X / s, Y: o.Y / s, W: o.W / s, H: o.H / s},
			Value: o.Value,
		}
	}
	return out
}

// scaledRois divides the full-resolution ROI coordinates by the working scale.
func (p *AlignPanel) scaledRois() []core.AlignRoi {
	s := float64(p.scale)
	out := make([]core.AlignRoi, len(p.rois))
	for i, r := range p.rois {
		out[i] = core.AlignRoi{
			Name: r.Name,
			Rect: core.Rect{X: r.X / s, Y: r.Y / s, W: r.W / s, H: r.H / s},
		}
	}
	return out
}

// alignKey is everything a report depends on. A report is stale when the panel's current inputs
// differ from the ones it was computed from; a finished job must never mark newer edits clean.
type alignKey struct {
	params  core.WarpParams
	occs    []core.Occlusion
	rois    []core.AlignRoi
	scale   int
	source  string
	tpl     string
	config  core.AlignConfig
	light   core.LightSetting  // only with source "light"
	channel core.CameraChannel // only with source "light"
}

// inputKey snapshots the current inputs (the slices are copied, so it can be kept).
func (p *AlignPanel) inputKey() alignKey {
	k := alignKey{
		params: p.params,
		occs:   append([]core.Occlusion(nil), p.occs...),
		rois:   cloneRois(p.rois),
		scale:  p.scale,
		source: p.source,
		tpl:    p.tplSource,
		config: p.config,
	}
	k.params.Occlusions = nil
	if p.source == "light" {
		k.light = p.a.Light.Clone()
		k.channel = p.a.CameraChannel()
	}
	return k
}

// reportStale reports whether the shown report does not belong to the current inputs.
func (p *AlignPanel) reportStale() bool {
	return p.report != nil && !reflect.DeepEqual(p.reportKey, p.inputKey())
}

// requestRun schedules the full pipeline (warp + align + view) under key "align".
func (p *AlignPanel) requestRun(force bool) {
	a := p.a
	scene := a.Scene
	if scene == nil {
		return
	}
	if len(p.rois) < 2 {
		// Still generate and show the warped image; only the matching step needs ROIs.
		a.SetHint(T("align.needRois"))
	}
	scale := p.scale
	// Snapshot panel state: the job goroutine must not read panel fields.
	template, input := p.template, p.input
	reference := p.reference
	source := p.source
	tplSource := p.tplSource
	cache := p.design
	viewKind := p.viewKind
	params := p.scaledParams()
	rois := p.scaledRois()
	config := p.config
	light := a.Light.Clone()
	ch := a.CameraChannel()
	needWarp := p.dirty || p.template == nil || force
	runAlign := p.autoRun || force
	key := p.inputKey()
	curFit, curAligned := p.alignFit(), p.aligned
	regenerated := needWarp || template == nil
	a.Jobs.Run("align", func() func() {
		start := time.Now()
		var newDesign *designTemplate
		if regenerated {
			var base *core.GrayImage
			if source == "light" {
				base = core.RenderChannel(scene.Light, &light, ch)
			} else {
				base = scene.Median
			}
			var inputBase *core.GrayImage
			if scale > 1 {
				down, err := core.DownsampleArea(base, scale)
				if err != nil {
					return func() { a.SetHint(T("common.error", "msg", err.Error())) }
				}
				inputBase = down
			} else {
				inputBase = base.Clone()
			}
			reference = inputBase
			if tplSource == tplImage {
				template = inputBase
			} else {
				reuse := cache != nil && cache.scene == scene && cache.source == source && cache.scale == scale &&
					(source != "light" || (cache.channel == ch && reflect.DeepEqual(cache.light, light)))
				if reuse {
					template = cache.img
				} else {
					img, info, err := core.DesignTemplate(scene.Material, base, scale)
					if err != nil {
						return func() { a.SetHint(T("common.error", "msg", err.Error())) }
					}
					newDesign = &designTemplate{scene: scene, source: source, scale: scale, img: img, info: info}
					if source == "light" {
						newDesign.light = light
						newDesign.channel = ch
					}
					template = img
				}
			}
			input = core.ApplyWarp(inputBase, &params, 1)
		}
		var report *core.AlignReport
		var truth *core.TruthComparison
		var diag []core.RoiDiagnosis
		if runAlign && len(rois) >= 2 {
			r := core.AlignWithRois(template, input, rois, config, core.Similarity{Scale: 1})
			report = &r
			t := core.CompareToTruth(r.Fit, &params, rois, template.Width, template.Height)
			truth = &t
			diag = core.DiagnoseRois(&r, rois, &params, template.Width, template.Height)
		}
		// The image the alignment result produces: only from the fit this run found, or from the
		// earlier fit while the input is still the image that fit was computed for.
		var fit *core.Similarity
		if report != nil {
			fit = report.Fit
		} else if !regenerated {
			fit = curFit
		}
		aligned := alignedFor(curAligned, reference, input, fit, params.Occlusions, config.PyramidFactor)
		pix := packAlignView(viewKind, template, input, aligned)
		elapsed := float64(time.Since(start).Microseconds()) / 1000
		w, h := template.Width, template.Height
		return func() {
			if a.Scene != scene {
				PutRGBA(pix)
				return
			}
			p.template = template
			p.input = input
			p.reference = reference
			p.aligned = aligned
			if newDesign != nil {
				p.design = newDesign
			}
			if tplSource == tplDesign && p.design != nil {
				p.designInfo = p.design.info
			} else {
				p.designInfo = core.DesignInfo{}
			}
			// Only a job that saw the latest edits may mark the panel clean: a long run (pyramid 1)
			// can finish after the user has already changed something.
			if reflect.DeepEqual(key, p.inputKey()) {
				p.dirty = false
			}
			if report != nil {
				p.report = report
				p.truth = truth
				p.diag = diag
				p.reportKey = key
				p.fitInput = input
				p.elapsedMs = elapsed
				p.lastScale = scale
				p.reportConfig = config
				p.reportParams = params
				p.reportW, p.reportH = w, h
				if p.selected < 0 && len(report.Matches) > 0 {
					p.selected = 0
					p.heat = report.Matches[0].Surface
				}
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

// requestView repacks the current view from the cached template/input under key "align-view".
func (p *AlignPanel) requestView() {
	a := p.a
	scene := a.Scene
	if scene == nil || p.template == nil || p.input == nil {
		return
	}
	viewKind := p.viewKind
	template := p.template
	input := p.input
	reference := p.alignReference()
	fit, cached := p.alignFit(), p.aligned
	occs := p.scaledParams().Occlusions
	pyramid := p.config.PyramidFactor
	a.Jobs.Run("align-view", func() func() {
		aligned := cached
		if isAlignedView(viewKind) {
			aligned = alignedFor(cached, reference, input, fit, occs, pyramid)
		}
		pix := packAlignView(viewKind, template, input, aligned)
		w, h := template.Width, template.Height
		return func() {
			if a.Scene != scene || p.template != template {
				PutRGBA(pix)
				return
			}
			if p.input == input && aligned != nil {
				p.aligned = aligned
			}
			if p.active {
				a.Viewer.SetImage(w, h, pix)
			} else {
				PutRGBA(pix)
			}
		}
	})
}

func (p *AlignPanel) DrawLeft() {
	a := p.a
	changed := false

	if Section(T("align.section.source"), "align-src") {
		src := 0
		if p.source == "light" {
			src = 1
		}
		if Combo(T("align.section.source"), "align-source", &src, []string{T("align.source.median"), T("align.source.light")}) {
			if src == 1 {
				p.source = "light"
			} else {
				p.source = "median"
			}
			p.dirty = true
			changed = true
		}
		tsel := 0
		if p.tplSource == tplImage {
			tsel = 1
		}
		if Combo(T("align.template"), "align-template", &tsel, []string{T("align.template.design"), T("align.template.image")}) {
			if tsel == 1 {
				p.tplSource = tplImage
			} else {
				p.tplSource = tplDesign
			}
			p.dirty = true
			changed = true
		}
		if p.tplSource == tplDesign {
			Note(T("align.template.designNote"))
			if p.design != nil && p.template == p.design.img {
				if p.designInfo.Registered {
					pose := core.FitPose(p.designInfo.Fit, p.template.Width, p.template.Height)
					workScale := p.lastScale
					if workScale == 0 {
						workScale = p.scale
					}
					Note(T("align.template.registered",
						"dx", F(pose.ShiftX*float64(workScale), 1),
						"dy", F(pose.ShiftY*float64(workScale), 1),
						"deg", F(pose.ThetaDeg, 2)))
				} else {
					Note(T("align.template.unregistered"))
				}
			}
		} else {
			Note(T("align.template.imageNote"))
		}
		sc := 0
		switch p.scale {
		case 1:
			sc = 0
		case 4:
			sc = 2
		default:
			sc = 1
		}
		if Combo(T("align.workScale"), "align-scale", &sc, []string{"1/1", "1/2", "1/4"}) {
			switch sc {
			case 0:
				p.scale = 1
			case 2:
				p.scale = 4
			default:
				p.scale = 2
			}
			p.dirty = true
			changed = true
		}
	}

	if Section(T("align.section.misalign"), "align-misalign") {
		for i, pr := range warpPresets() {
			label := T("warp.preset." + pr.id)
			if i > 0 {
				WrapSameLine(label)
			}
			if Button(label, "align-preset-"+pr.id) {
				setWarpGeometry(&p.params, pr.warp)
				changed = true
			}
		}
		changed = SliderF64(T("align.param.tx"), "align-tx", &p.params.Tx, -300, 300, "%.0f") || changed
		changed = SliderF64(T("align.param.ty"), "align-ty", &p.params.Ty, -300, 300, "%.0f") || changed
		changed = SliderF64(T("align.param.theta"), "align-theta", &p.params.ThetaDeg, -5, 5, "%.2f") || changed
		changed = SliderF64(T("align.param.scale"), "align-scale-p", &p.params.Scale, 0.95, 1.05, "%.3f") || changed
		if Button(T("align.resetAll"), "align-reset") {
			p.params = defaultWarpParams()
			p.occs = nil
			p.dirty = true
			changed = true
		}
	}

	if Section(T("align.section.lift"), "align-lift") {
		changed = SliderF64(T("align.param.tiltX"), "align-tiltx", &p.params.TiltX, -0.1, 0.1, "%.4f") || changed
		changed = SliderF64(T("align.param.tiltY"), "align-tilty", &p.params.TiltY, -0.1, 0.1, "%.4f") || changed
		changed = SliderF64(T("align.param.bendAmplitude"), "align-benda", &p.params.BendAmplitude, -30, 30, "%.1f") || changed
		changed = SliderF64(T("align.param.bendWavelength"), "align-bendw", &p.params.BendWavelength, 200, 6000, "%.0f") || changed
		changed = SliderF64(T("align.param.shearX"), "align-shearx", &p.params.ShearX, -0.05, 0.05, "%.3f") || changed
		changed = SliderF64(T("align.param.shearY"), "align-sheary", &p.params.ShearY, -0.05, 0.05, "%.3f") || changed
	}

	if SectionClosed(T("align.section.photo"), "align-photo") {
		changed = SliderF64(T("align.param.blurSigma"), "align-blur", &p.params.BlurSigma, 0, 6, "%.1f") || changed
		changed = SliderF64(T("align.param.noiseSigma"), "align-noise", &p.params.NoiseSigma, 0, 40, "%.1f") || changed
		changed = SliderF64(T("align.param.gain"), "align-gain", &p.params.Gain, 0.3, 2, "%.2f") || changed
		changed = SliderF64(T("align.param.offset"), "align-offset", &p.params.Offset, -80, 80, "%.0f") || changed
		changed = SliderF64(T("align.param.gradient"), "align-grad", &p.params.Gradient, -0.8, 0.8, "%.2f") || changed
	}

	if Section(T("align.section.occlusion"), "align-occ") {
		if ToolButton(T("align.occlusion.draw"), "align-occ-draw", p.tool == "occlusion") {
			p.tool = "occlusion"
			a.SetHint(T("align.occlusion.hint"))
		}
		SliderF64(T("align.occlusion.fill"), "align-occ-fill", &p.occValue, 0, 255, "%.0f")
		if len(p.occs) == 0 {
			Note(T("align.occlusion.empty"))
		} else {
			del := -1
			for i, o := range p.occs {
				Text(fmt.Sprintf("x%.0f y%.0f w%.0f h%.0f v%.0f", o.X, o.Y, o.W, o.H, o.Value))
				SameLine()
				if Button(T("align.occlusion.delete"), fmt.Sprintf("align-occ-del-%d", i)) {
					del = i
				}
			}
			if del >= 0 {
				p.occs = append(p.occs[:del], p.occs[del+1:]...)
				p.dirty = true
				changed = true
			}
		}
	}

	// Matching parameters only re-run the alignment; they never invalidate the warped pair.
	rerun := false
	if Section(T("align.section.match"), "align-match") {
		rerun = SliderF64(T("align.match.searchRadius"), "align-search", &p.config.SearchRadius, 10, 400, "%.0f") || rerun
		py := 0
		switch p.config.PyramidFactor {
		case 1:
			py = 0
		case 2:
			py = 1
		case 8:
			py = 3
		default:
			py = 2
		}
		if Combo(T("align.match.pyramid"), "align-pyramid", &py, []string{
			T("align.match.pyramid1"), T("align.match.pyramid2"), T("align.match.pyramid4"), T("align.match.pyramid8"),
		}) {
			switch py {
			case 0:
				p.config.PyramidFactor = 1
			case 1:
				p.config.PyramidFactor = 2
			case 3:
				p.config.PyramidFactor = 8
			default:
				p.config.PyramidFactor = 4
			}
			rerun = true
		}
		rerun = SliderF64(T("align.match.minNcc"), "align-minncc", &p.config.MinNcc, 0, 1, "%.2f") || rerun
		rerun = SliderF64(T("align.match.minUniqueness"), "align-minuniq", &p.config.MinUniqueness, 0, 1, "%.2f") || rerun
		rerun = SliderF64(T("align.match.minOverlap"), "align-minoverlap", &p.config.MinOverlap, 0.5, 1, "%.2f") || rerun
		rerun = SliderF64(T("align.match.ransac"), "align-ransac", &p.config.RansacThresholdPx, 0.5, 20, "%.1f") || rerun
		rerun = SliderF64(T("align.match.okResidual"), "align-okres", &p.config.OkResidualPx, 0.1, 10, "%.1f") || rerun
		rerun = SliderF64(T("align.match.ngResidual"), "align-ngres", &p.config.NgResidualPx, 0.1, 20, "%.1f") || rerun
		Note(T("align.match.minOverlapNote"))
		if est := core.MatchCost(p.scaledRois(), p.config) / matchOpsPerSecond; est >= slowRunSeconds {
			Warn(T("align.match.slow", "sec", F(est, 0)))
		}
	}

	if Section(T("align.section.roi"), "align-roi") {
		if Button(T("align.roi.restore"), "align-roi-restore") {
			p.rois = cloneRois(a.Scene.MachineRois)
			p.selected = -1
			changed = true
		}
		WrapSameLine(T("align.roi.new"))
		if ToolButton(T("align.roi.new"), "align-roi-new", p.tool == "roi") {
			p.tool = "roi"
			a.SetHint(T("align.roi.newHint"))
		}
		WrapSameLine(T("align.roi.select"))
		if ToolButton(T("align.roi.select"), "align-roi-select", p.tool == "select") {
			p.tool = "select"
			a.SetHint(T("align.roi.selectHint"))
		}
		if len(p.rois) == 0 {
			Note(T("align.roi.empty"))
		} else {
			del := -1
			for i, r := range p.rois {
				// A latching tool button stands in for the ImGui selectable row.
				sel := i == p.selected
				if ToolButton(fmt.Sprintf("%s  x%.0f y%.0f w%.0f h%.0f", r.Name, r.X, r.Y, r.W, r.H), fmt.Sprintf("align-roi-sel-%d", i), sel) {
					p.selected = i
					a.Viewer.CenterOn(r.X/float64(p.scale)+r.W/float64(p.scale)/2, r.Y/float64(p.scale)+r.H/float64(p.scale)/2)
				}
				SameLine()
				if Button(T("align.occlusion.delete"), fmt.Sprintf("align-roi-del-%d", i)) {
					del = i
				}
			}
			if del >= 0 {
				p.rois = append(p.rois[:del], p.rois[del+1:]...)
				if p.selected >= len(p.rois) {
					p.selected = -1
				}
				changed = true
			}
		}
		if ui.KeyPressed("Delete") && p.selected >= 0 && p.selected < len(p.rois) {
			p.rois = append(p.rois[:p.selected], p.rois[p.selected+1:]...)
			p.selected = -1
			changed = true
		}
	}

	if Section(T("align.section.action"), "align-action") {
		if Check(T("align.action.autoRun"), "align-autorun", &p.autoRun) {
			rerun = true
		}
		if Button(T("align.action.run"), "align-run") {
			p.requestRun(true)
		}
		WrapSameLine(T("align.action.generate"))
		if Button(T("align.action.generate"), "align-gen") {
			p.dirty = true
			p.requestRun(true)
		}
		vk := alignViewIndex(p.viewKind)
		if Combo(T("align.view"), "align-viewkind", &vk, []string{
			T("align.view.overlay"), T("align.view.template"), T("align.view.input"), T("align.view.diff"),
			T("align.view.aligned"), T("align.view.alignedOverlay"), T("align.view.alignedDiff"),
		}) {
			p.viewKind = alignViews[vk]
			if p.dirty || p.template == nil {
				p.requestRun(true)
			} else {
				p.requestView()
			}
		}
		if isAlignedView(p.viewKind) {
			if al := p.alignedNow(); al == nil {
				Warn(T("align.view.noFit"))
			} else {
				Note(T("align.view.alignedNote"))
				if al.check.Windows > 0 && Button(T("align.view.gotoWorst"), "align-goto-worst") {
					a.Viewer.ZoomOn(al.check.Worst.X, al.check.Worst.Y, 4)
				}
			}
		}
	}

	if changed {
		p.dirty = true
	}
	if (changed || rerun) && p.autoRun {
		p.requestRun(false)
	}
}

func (p *AlignPanel) DrawRight() {
	if entries := p.legendEntries(); len(entries) > 0 {
		if Section(T("align.legend.title"), "align-legend") {
			drawLegend(entries)
		}
	}
	if Section(T("align.section.verdict"), "align-verdict") {
		p.drawVerdict()
	}
	if Section(T("align.section.roiResult"), "align-roires") {
		p.drawRoiResults()
	}
	if Section(T("align.section.ncc"), "align-ncc") {
		if p.heat == nil {
			Note(T("align.result.empty"))
		} else {
			Heatmap("align", p.heat, 320)
		}
	}
}

// reasonText renders a verdict reason using the translated code and computed values.
func (p *AlignPanel) reasonText(r core.AlignVerdictReason) string {
	switch r.Code {
	case "too-few-located":
		n := 0
		if p.report != nil {
			for _, m := range p.report.Matches {
				if m.Located {
					n++
				}
			}
		}
		return T("align.reason.too-few-located", "n", n)
	case "no-fit":
		return T("align.reason.no-fit")
	case "residual-high":
		return T("align.reason.residual-high",
			"rms", F(p.fullPx(p.report.ResidualRms), 3),
			"th", F(p.fullPx(p.reportConfig.NgResidualPx), 3))
	case "outlier":
		return p.reasonWithRois(T("align.reason.outlier"), func(i int, m core.RoiMatch) (string, bool) {
			if !m.Located || m.Found == nil || i >= len(p.report.Residuals) || p.report.Residuals[i] >= 0 || i >= len(p.diag) {
				return "", false
			}
			return T("align.detail.outlier", "name", m.Name, "dev", F(p.fullPx(p.diag[i].FitDevPx), 1),
				"th", F(p.fullPx(p.reportConfig.RansacThresholdPx), 1)), true
		})
	case "ambiguous":
		return p.reasonWithRois(T("align.reason.ambiguous"), func(i int, m core.RoiMatch) (string, bool) {
			if !m.Located || m.Uniqueness >= p.reportConfig.MinUniqueness {
				return "", false
			}
			return T("align.detail.ambiguous", "name", m.Name, "u", F(m.Uniqueness, 3), "min", F(p.reportConfig.MinUniqueness, 2)), true
		})
	case "residual-warn":
		return T("align.reason.residual-warn",
			"rms", F(p.fullPx(p.report.ResidualRms), 3),
			"th", F(p.fullPx(p.reportConfig.OkResidualPx), 3))
	case "roi-missing":
		return p.reasonWithRois(T("align.reason.roi-missing"), func(i int, m core.RoiMatch) (string, bool) {
			if m.Located {
				return "", false
			}
			return T("align.detail.missing", "name", m.Name, "why", p.matchReasonText(m.Reason)), true
		})
	}
	return r.Code
}

// reasonWithRois appends to a verdict reason the ROIs it is about, each with its numbers.
func (p *AlignPanel) reasonWithRois(base string, detail func(i int, m core.RoiMatch) (string, bool)) string {
	if p.report == nil {
		return base
	}
	var items []string
	for i, m := range p.report.Matches {
		if text, ok := detail(i, m); ok {
			items = append(items, text)
		}
	}
	if len(items) == 0 {
		return base
	}
	return base + T("align.colon") + strings.Join(items, T("align.sep"))
}

// matchReasonText translates a per-ROI reason code.
func (p *AlignPanel) matchReasonText(code string) string {
	switch code {
	case "out-of-range":
		return T("align.match.out-of-range")
	case "flat":
		return T("align.match.flat")
	case "low-ncc":
		return T("align.match.low-ncc")
	}
	return code
}

func (p *AlignPanel) Overlay(o *Overlay) {
	s := float64(p.scale)
	if p.viewKind == "template" {
		for i, r := range p.rois {
			th := float32(1.5)
			if i == p.selected {
				th = 3
			}
			o.Rect(r.X/s, r.Y/s, r.W/s, r.H/s, OverlayWarn, th)
			o.Text(r.X/s, r.Y/s, 2, 12, OverlayWarn, r.Name)
		}
	} else if isAlignedView(p.viewKind) {
		p.drawAlignedOverlay(o)
	} else {
		p.drawAlignOverlay(o)
	}
	// Occlusions are painted onto the warped input, so they sit at the same place in every view
	// of the input; in the aligned views they move with the input, outlined where the fit puts them.
	aligned := isAlignedView(p.viewKind)
	var back core.Similarity
	fit := p.alignFit()
	if fit != nil {
		back = core.InvertSimilarity(*fit)
	}
	for _, occ := range p.occs {
		x, y, w, h := occ.X/s, occ.Y/s, occ.W/s, occ.H/s
		if !aligned {
			o.FillRect(x, y, w, h, OverlayNGFill)
			o.Rect(x, y, w, h, OverlayNG, 1.5)
			continue
		}
		if fit == nil {
			continue
		}
		c := [4]core.Point{
			core.ApplySimilarity(back, core.Point{X: x, Y: y}),
			core.ApplySimilarity(back, core.Point{X: x + w, Y: y}),
			core.ApplySimilarity(back, core.Point{X: x + w, Y: y + h}),
			core.ApplySimilarity(back, core.Point{X: x, Y: y + h}),
		}
		for k := range c {
			n := c[(k+1)%4]
			o.Line(c[k].X, c[k].Y, n.X, n.Y, OverlayNG, 1.5)
		}
	}
	if p.dragging {
		o.DragBox(p.dragStart.X, p.dragStart.Y, p.dragCur.X, p.dragCur.Y, OverlayROI)
	}
}

func (p *AlignPanel) Pointer(ev PointerEvent) bool {
	s := float64(p.scale)
	switch p.tool {
	case "roi", "occlusion":
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
			if w >= 4 && h >= 4 {
				if p.tool == "roi" {
					p.rois = append(p.rois, core.AlignRoi{
						Name: fmt.Sprintf("R%d", len(p.rois)+1),
						Rect: core.Rect{X: x * s, Y: y * s, W: w * s, H: h * s},
					})
				} else {
					p.occs = append(p.occs, core.Occlusion{
						Rect:  core.Rect{X: x * s, Y: y * s, W: w * s, H: h * s},
						Value: p.occValue,
					})
				}
				p.dirty = true
				if p.autoRun {
					p.requestRun(false)
				}
			}
			p.tool = "none"
			return true
		}
		return false
	case "select":
		switch ev.Kind {
		case PointerDown:
			idx := p.hitTest(ev.X, ev.Y)
			if idx >= 0 {
				p.selected = idx
				p.moveIdx = idx
				p.moveOrig = core.Point{X: ev.X, Y: ev.Y}
				p.dragging = true
				return true
			}
			p.selected = -1
			return false
		case PointerMove:
			if p.dragging && p.moveIdx >= 0 && p.moveIdx < len(p.rois) {
				r := &p.rois[p.moveIdx]
				r.X += (ev.X - p.moveOrig.X) * s
				r.Y += (ev.Y - p.moveOrig.Y) * s
				p.moveOrig = core.Point{X: ev.X, Y: ev.Y}
				return true
			}
			return false
		case PointerUp:
			if p.dragging {
				p.dragging = false
				p.moveIdx = -1
				p.dirty = true
				if p.autoRun {
					p.requestRun(false)
				}
				return true
			}
			return false
		}
	}
	return false
}

// hitTest returns the topmost ROI containing the working-resolution point.
func (p *AlignPanel) hitTest(x, y float64) int {
	s := float64(p.scale)
	fx := x * s
	fy := y * s
	for i := len(p.rois) - 1; i >= 0; i-- {
		r := p.rois[i]
		if fx >= r.X && fy >= r.Y && fx <= r.X+r.W && fy <= r.Y+r.H {
			return i
		}
	}
	return -1
}

// ClaimsDrag: the ROI / occlusion tools take every left drag, the select tool only over a ROI
// (or while one is being moved); otherwise the viewer pans.
func (p *AlignPanel) ClaimsDrag() bool {
	switch p.tool {
	case "roi", "occlusion":
		return true
	case "select":
		return p.dragging || p.hoverRoi
	}
	return false
}

func (p *AlignPanel) Hover(x, y int) string {
	p.hoverRoi = p.tool == "select" && p.template != nil && p.hitTest(float64(x)+0.5, float64(y)+0.5) >= 0
	if isAlignedView(p.viewKind) {
		return p.hoverAligned(x, y)
	}
	var img *core.GrayImage
	switch p.viewKind {
	case "input", "diff", "overlay":
		img = p.input
	default:
		img = p.template
	}
	if img == nil || x < 0 || y < 0 || x >= img.Width || y >= img.Height {
		return ""
	}
	if p.viewKind == "overlay" && p.template != nil {
		return fmt.Sprintf("T %d  I %d", p.template.Data[y*p.template.Width+x], img.Data[y*img.Width+x])
	}
	if p.viewKind == "diff" && p.template != nil {
		d := int(img.Data[y*img.Width+x]) - int(p.template.Data[y*p.template.Width+x])
		if d < 0 {
			d = -d
		}
		return fmt.Sprintf("GV %d", d)
	}
	return fmt.Sprintf("GV %d", img.Data[y*img.Width+x])
}
