package app

import (
	"afvi/teach-imgui/internal/access"
	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/ui"
)

func init() {
	registerPanelFor([]access.Role{access.RoleAdmin}, 1,
		func(a *App) Panel { return newAlignFlowPanel(a, false) }, ProfileTeach)
	registerPanelFor([]access.Role{access.RoleOperator}, 1,
		func(a *App) Panel { return newAlignFlowPanel(a, true) }, ProfileTeach)
	Messages(map[string]string{
		"tab.roi":             "对位",
		"tab.opAlign":         "对位",
		"flow.state.off":      "未对位",
		"flow.state.working":  "对位中",
		"flow.state.done":     "对位完成",
		"flow.state.failed":   "对位失败",
		"flow.view.design":    "设计图",
		"flow.view.raw":       "拍摄图",
		"flow.view.aligned":   "对位后",
		"flow.material.base":  "基材",
		"flow.material.sr":    "SR",
		"flow.material.open":  "开口",
		"flow.material.metal": "金属",
		"flow.needTwo":        "至少需要两个框。",
		"flow.viewLabel":      "查看",

		"opAlign.title":        "对位",
		"opAlign.step.off":     "第 1 步：在假想图上确认框的位置（拖动画新框，拖动框移动，Delete 删除），然后点击“确定对位”。",
		"opAlign.step.working": "正在对位，请稍候……",
		"opAlign.step.done":    "对位完成。下一步：切换到“金板”页，确认检查区域。",
		"opAlign.step.failed":  "对位失败。\n{reason}",
		"opAlign.confirm":      "确定对位",
		"opAlign.reset":        "恢复默认框",
		"opAlign.delete":       "删除选中框",
		"opAlign.needTwo":      "至少需要两个框。",
		"opAlign.toGolden":     "下一步：金板",
		"opAlign.help.1":       "1. 这是按设计图画出的假想产品图，框要框住有明显特征的位置（四角、边缘、金属块）。",
		"opAlign.help.2":       "2. 在图上拖动画一个新框，框内拖动可移动，选中后按 Delete 删除。",
		"opAlign.help.3":       "3. 框好以后点击左侧“确定对位”。",
		"opAlign.help.4":       "4. 对位完成后切到“检测”页做检测。",

		"roi.section.view":     "视图",
		"roi.section.rois":     "ROI 列表",
		"roi.section.params":   "参数",
		"roi.section.taught":   "已教对位",
		"roi.restore":          "恢复机台定位点",
		"roi.clear":            "清空",
		"roi.delete":           "删除选中",
		"roi.confirm":          "确定对位",
		"roi.cancel":           "取消已教对位",
		"roi.empty":            "尚无 ROI。",
		"roi.noReport":         "尚未对齐当前拍摄图。",
		"roi.col.name":         "名称",
		"roi.col.ncc":          "NCC",
		"roi.col.uniq":         "唯一性",
		"roi.col.located":      "定位",
		"roi.col.error":        "误差 px",
		"roi.info.elapsed":     "耗时 ms",
		"roi.info.fitTx":       "拟合 Tx",
		"roi.info.fitTy":       "拟合 Ty",
		"roi.info.rot":         "旋转°",
		"roi.info.scale":       "缩放",
		"roi.info.meanNcc":     "平均 NCC",
		"roi.info.residual":    "内点残差 RMS px",
		"roi.yes":              "是",
		"roi.no":               "否",
		"roi.none":             "—",
		"roi.param.scale":      "工作分辨率",
		"roi.param.search":     "搜索半径",
		"roi.param.minNcc":     "最小 NCC",
		"roi.param.minUniq":    "最小唯一性",
		"roi.param.minOverlap": "最小重叠",
	}, map[string]string{
		"tab.roi":             "Align",
		"tab.opAlign":         "Align",
		"flow.state.off":      "Not aligned",
		"flow.state.working":  "Aligning",
		"flow.state.done":     "Aligned",
		"flow.state.failed":   "Align failed",
		"flow.view.design":    "Design",
		"flow.view.raw":       "Capture",
		"flow.view.aligned":   "Aligned",
		"flow.material.base":  "Base",
		"flow.material.sr":    "SR",
		"flow.material.open":  "Open",
		"flow.material.metal": "Metal",
		"flow.needTwo":        "At least two boxes are required.",
		"flow.viewLabel":      "View",

		"opAlign.title":        "Align",
		"opAlign.step.off":     "Step 1: confirm the box positions on the design picture (drag to draw a box, drag inside to move, Delete to remove), then press Confirm.",
		"opAlign.step.working": "Aligning, please wait…",
		"opAlign.step.done":    "Aligned. Next: switch to the Golden Board page and confirm the inspection area.",
		"opAlign.step.failed":  "Alignment failed.\n{reason}",
		"opAlign.confirm":      "Confirm",
		"opAlign.reset":        "Restore default boxes",
		"opAlign.delete":       "Delete selected box",
		"opAlign.needTwo":      "At least two boxes are required.",
		"opAlign.toGolden":     "Next: Golden Board",
		"opAlign.help.1":       "1. This is the imagined product picture drawn from the design; the boxes must cover distinctive features (corners, edges, metal pads).",
		"opAlign.help.2":       "2. Drag on the picture to draw a new box, drag inside a box to move it, Delete removes the selected box.",
		"opAlign.help.3":       "3. When the boxes are placed, press Confirm on the left.",
		"opAlign.help.4":       "4. After the alignment is done, switch to the Inspect page.",

		"roi.section.view":     "View",
		"roi.section.rois":     "ROI list",
		"roi.section.params":   "Parameters",
		"roi.section.taught":   "Taught alignment",
		"roi.restore":          "Restore machine anchors",
		"roi.clear":            "Clear",
		"roi.delete":           "Delete selected",
		"roi.confirm":          "Confirm",
		"roi.cancel":           "Cancel taught alignment",
		"roi.empty":            "No ROIs yet.",
		"roi.noReport":         "The current capture has not been aligned yet.",
		"roi.col.name":         "Name",
		"roi.col.ncc":          "NCC",
		"roi.col.uniq":         "Uniqueness",
		"roi.col.located":      "Located",
		"roi.col.error":        "Error px",
		"roi.info.elapsed":     "Elapsed ms",
		"roi.info.fitTx":       "Fit Tx",
		"roi.info.fitTy":       "Fit Ty",
		"roi.info.rot":         "Rotation°",
		"roi.info.scale":       "Scale",
		"roi.info.meanNcc":     "Mean NCC",
		"roi.info.residual":    "Inlier residual RMS px",
		"roi.yes":              "Yes",
		"roi.no":               "No",
		"roi.none":             "—",
		"roi.param.scale":      "Working scale",
		"roi.param.search":     "Search radius",
		"roi.param.minNcc":     "Min NCC",
		"roi.param.minUniq":    "Min uniqueness",
		"roi.param.minOverlap": "Min overlap",
	}, map[string]string{
		"tab.roi":             "정렬",
		"tab.opAlign":         "정렬",
		"flow.state.off":      "정렬 안 됨",
		"flow.state.working":  "정렬 중",
		"flow.state.done":     "정렬 완료",
		"flow.state.failed":   "정렬 실패",
		"flow.view.design":    "설계도",
		"flow.view.raw":       "촬영 이미지",
		"flow.view.aligned":   "정렬 후",
		"flow.material.base":  "베이스",
		"flow.material.sr":    "SR",
		"flow.material.open":  "오픈",
		"flow.material.metal": "메탈",
		"flow.needTwo":        "최소 두 개의 박스가 필요합니다.",
		"flow.viewLabel":      "보기",

		"opAlign.title":        "정렬",
		"opAlign.step.off":     "1 단계: 가상 이미지에서 박스 위치를 확인하세요. (드래그로 새 박스, 박스 안 드래그로 이동, Delete 로 삭제) 그리고 '정렬 확인' 을 누르세요.",
		"opAlign.step.working": "정렬 중입니다. 잠시 기다려 주세요…",
		"opAlign.step.done":    "정렬 완료. 다음: '골든 보드' 페이지로 이동해 검사 영역을 확인하세요.",
		"opAlign.step.failed":  "정렬 실패.\n{reason}",
		"opAlign.confirm":      "정렬 확인",
		"opAlign.reset":        "기본 박스 복원",
		"opAlign.delete":       "선택 박스 삭제",
		"opAlign.needTwo":      "최소 두 개의 박스가 필요합니다.",
		"opAlign.toGolden":     "다음: 골든 보드",
		"opAlign.help.1":       "1. 설계도로 그린 가상 제품도입니다. 뚜렷한 특징(모서리, 가장자리, 금속 패드)을 박스로 감싸세요.",
		"opAlign.help.2":       "2. 그림 위에서 드래그하여 새 박스를 그리고, 박스 안을 드래그하면 이동합니다. Delete 로 선택한 박스를 삭제합니다.",
		"opAlign.help.3":       "3. 박스를 배치한 뒤 왼쪽에서 '정렬 확인' 을 누르세요.",
		"opAlign.help.4":       "4. 정렬이 끝나면 '검사' 페이지로 이동하세요.",

		"roi.section.view":     "보기",
		"roi.section.rois":     "ROI 목록",
		"roi.section.params":   "파라미터",
		"roi.section.taught":   "학습된 정렬",
		"roi.restore":          "기계 기준점 복원",
		"roi.clear":            "비우기",
		"roi.delete":           "선택 삭제",
		"roi.confirm":          "정렬 확인",
		"roi.cancel":           "학습된 정렬 취소",
		"roi.empty":            "아직 ROI 가 없습니다.",
		"roi.noReport":         "현재 촬영 이미지는 아직 정렬되지 않았습니다.",
		"roi.col.name":         "이름",
		"roi.col.ncc":          "NCC",
		"roi.col.uniq":         "고유성",
		"roi.col.located":      "위치",
		"roi.col.error":        "오차 px",
		"roi.info.elapsed":     "소요 ms",
		"roi.info.fitTx":       "피팅 Tx",
		"roi.info.fitTy":       "피팅 Ty",
		"roi.info.rot":         "회전°",
		"roi.info.scale":       "스케일",
		"roi.info.meanNcc":     "평균 NCC",
		"roi.info.residual":    "인라이어 잔차 RMS px",
		"roi.yes":              "예",
		"roi.no":               "아니오",
		"roi.none":             "—",
		"roi.param.scale":      "작업 해상도",
		"roi.param.search":     "탐색 반경",
		"roi.param.minNcc":     "최소 NCC",
		"roi.param.minUniq":    "최소 고유성",
		"roi.param.minOverlap": "최소 겹침",
	})
}

// flowView selects which picture the alignment pages show.
type flowView int

const (
	flowViewDesign flowView = iota
	flowViewRaw
	flowViewAligned
)

// flowViews is the display order of the three views.
var flowViews = []flowView{flowViewDesign, flowViewRaw, flowViewAligned}

// viewKey returns the i18n suffix of a view ("design" | "raw" | "aligned").
func (v flowView) key() string {
	switch v {
	case flowViewRaw:
		return "raw"
	case flowViewAligned:
		return "aligned"
	}
	return "design"
}

// AlignFlowPanel is the ROI page of the forced alignment: the admin variant (`roi`) is the
// engineering view with sliders and tables, the operator variant (`op-align`) is the guided
// version. Both edit App.align (the AlignFlow) through the shared roiEditor.
type AlignFlowPanel struct {
	a        *App
	operator bool
	editor   *roiEditor
	active   bool
	view     flowView

	shownRaw     *core.GrayImage
	shownAligned *core.GrayImage
	shownView    flowView
	shownScene   *Scene

	msg string // "needTwo" after a failed confirm ("" = none)
}

func newAlignFlowPanel(a *App, operator bool) *AlignFlowPanel {
	p := &AlignFlowPanel{a: a, operator: operator, editor: newRoiEditor(), view: flowViewDesign}
	// Force the first refresh so the picture is uploaded on activation.
	p.shownView = flowView(-1)
	return p
}

// ID returns the page id of the variant.
func (p *AlignFlowPanel) ID() string {
	if p.operator {
		return "op-align"
	}
	return "roi"
}

// Title returns the translated tab title of the variant.
func (p *AlignFlowPanel) Title() string {
	if p.operator {
		return T("tab.opAlign")
	}
	return T("tab.roi")
}

// HidePalette: the ROI pages have no floating tool palette.
func (p *AlignFlowPanel) HidePalette() bool { return true }

// Activate opens the page: it resets the editor and ensures a capture exists.
func (p *AlignFlowPanel) Activate() {
	p.active = true
	p.editor.Reset()
	p.a.Viewer.SetDoubleClickFit(true)
	p.refreshImage(true)
	if dp := p.dp(); dp != nil && dp.Raw() == nil {
		dp.RequestCapture()
	}
}

// Deactivate closes the page.
func (p *AlignFlowPanel) Deactivate() {
	p.active = false
	p.editor.Reset()
}

// ModelLoaded resets the view for a new model.
func (p *AlignFlowPanel) ModelLoaded() {
	p.view = flowViewDesign
	p.editor.Reset()
	p.shownRaw = nil
	p.shownAligned = nil
	p.shownScene = nil
	p.shownView = flowView(-1)
	p.msg = ""
	if p.active {
		p.refreshImage(true)
	}
}

// LightChanged: the design picture and the capture do not depend on the display light.
func (p *AlignFlowPanel) LightChanged() {}

// Tick refreshes the shown picture once per frame while the page is active.
func (p *AlignFlowPanel) Tick() {
	if p.active {
		p.refreshImage(false)
	}
}

// Selftest has nothing to seed: the page derives everything from the flow and the capture.
func (p *AlignFlowPanel) Selftest() {}

// dp returns the detection panel of the app (nil if none).
func (p *AlignFlowPanel) dp() *DefectPanel { return defectPanelOf(p.a) }

// flow returns the taught alignment of the app (nil in the studio).
func (p *AlignFlowPanel) flow() *AlignFlow { return p.a.alignFlow() }

// currentRois returns the taught ROIs (nil when there is no flow).
func (p *AlignFlowPanel) currentRois() []core.AlignRoi {
	if f := p.flow(); f != nil {
		return f.Rois
	}
	return nil
}

// refreshImage uploads the picture of the current view; force ignores the change check.
func (p *AlignFlowPanel) refreshImage(force bool) {
	scene := p.a.Scene
	if scene == nil {
		return
	}
	dp := p.dp()
	var raw, aligned *core.GrayImage
	if dp != nil {
		raw = dp.Raw()
		aligned = dp.AlignedImage()
	}
	view := p.view
	if view == flowViewRaw && raw == nil {
		view = flowViewDesign
	}
	if view == flowViewAligned && aligned == nil {
		view = flowViewDesign
	}
	if !force && p.shownView == view && p.shownRaw == raw && p.shownAligned == aligned && p.shownScene == scene {
		return
	}
	var pix []byte
	var w, h int
	switch view {
	case flowViewRaw:
		w, h = raw.Width, raw.Height
		pix = PackGray(raw)
	case flowViewAligned:
		w, h = aligned.Width, aligned.Height
		pix = PackGray(aligned)
	default:
		w, h = scene.W, scene.H
		pix = PackMaterial(scene.Material)
	}
	p.a.Viewer.SetImage(w, h, pix)
	p.shownView = view
	p.shownRaw = raw
	p.shownAligned = aligned
	p.shownScene = scene
}

// Overlay draws the ROIs of the flow into o.
func (p *AlignFlowPanel) Overlay(o *Overlay) {
	p.editor.Overlay(o, p.currentRois())
}

// Pointer forwards a pointer event to the ROI editor.
func (p *AlignFlowPanel) Pointer(ev PointerEvent) bool {
	flow := p.flow()
	if flow == nil || p.a.Scene == nil {
		return false
	}
	switch ev.Kind {
	case PointerDown, PointerMove, PointerUp:
	default:
		return false
	}
	out, changed, owned := p.editor.Pointer(ev, flow.Rois, p.a.Scene.W, p.a.Scene.H)
	if changed {
		flow.SetRois(out)
		if dp := p.dp(); dp != nil {
			dp.RequestCapture()
		}
	}
	return owned
}

// ClaimsDrag: over the picture a left drag draws a new ROI or moves one (the editor owns
// every drag that starts inside the image); outside it, and for the Wails canvas that means
// everywhere else, the view pans. A drag in progress stays claimed.
func (p *AlignFlowPanel) ClaimsDrag() bool {
	if p.flow() == nil || p.a.Scene == nil {
		return false
	}
	if p.editor.mode != roiDragNone {
		return true
	}
	return p.mouseOverPicture()
}

// mouseOverPicture reports whether the cursor is over the design picture.
func (p *AlignFlowPanel) mouseOverPicture() bool {
	x, y, inside := p.a.Viewer.Mouse()
	s := p.a.Scene
	return inside && s != nil && x >= 0 && y >= 0 && x < float64(s.W) && y < float64(s.H)
}

// Hover returns the material name under the pixel.
func (p *AlignFlowPanel) Hover(x, y int) string {
	scene := p.a.Scene
	if scene == nil || x < 0 || y < 0 || x >= scene.W || y >= scene.H {
		return ""
	}
	if len(scene.Material) < scene.W*scene.H {
		return ""
	}
	switch scene.Material[y*scene.W+x] & 3 {
	case 0:
		return T("flow.material.base")
	case 1:
		return T("flow.material.sr")
	case 2:
		return T("flow.material.open")
	default:
		return T("flow.material.metal")
	}
}

// deleteSelected removes the selected ROI and re-runs the capture.
func (p *AlignFlowPanel) deleteSelected() {
	flow := p.flow()
	if flow == nil {
		return
	}
	out, changed := p.editor.DeleteSelected(flow.Rois)
	if changed {
		flow.SetRois(out)
		if dp := p.dp(); dp != nil {
			dp.RequestCapture()
		}
	}
}

// confirm arms the flow; fewer than two ROIs sets the needTwo message.
func (p *AlignFlowPanel) confirm() {
	flow := p.flow()
	if flow == nil {
		return
	}
	if flow.Arm() {
		p.msg = ""
		if dp := p.dp(); dp != nil {
			dp.RequestCapture()
		}
		return
	}
	p.msg = "needTwo"
}

// resetBoxes restores the machine ROIs and re-runs the capture.
func (p *AlignFlowPanel) resetBoxes() {
	flow := p.flow()
	if flow == nil {
		return
	}
	flow.ResetRois()
	p.editor.Reset()
	if dp := p.dp(); dp != nil {
		dp.RequestCapture()
	}
}

// alignStateText translates an alignment state.
func alignStateText(st alignState) string { return T("flow.state." + st.key()) }

// reason returns the human-readable failure reason of the last alignment ("" when none).
func (p *AlignFlowPanel) reason() string {
	dp := p.dp()
	if dp == nil {
		return ""
	}
	out := dp.AlignOutcome()
	if out == nil {
		return ""
	}
	drawn := 0
	if f := p.flow(); f != nil {
		drawn = len(f.Rois)
	}
	if text := explainAlign(out, drawn); text != "" {
		return text
	}
	if len(out.Report.Reasons) > 0 {
		return out.Report.Reasons[0].Message
	}
	return out.Err
}

// DrawLeft dispatches to the variant.
func (p *AlignFlowPanel) DrawLeft() {
	if p.operator {
		p.drawOperatorLeft()
	} else {
		p.drawAdminLeft()
	}
}

// DrawRight dispatches to the variant.
func (p *AlignFlowPanel) DrawRight() {
	if p.operator {
		p.drawOperatorRight()
	} else {
		p.drawAdminRight()
	}
}

// drawOperatorLeft is the big guided column of the operator page.
func (p *AlignFlowPanel) drawOperatorLeft() {
	Text(T("opAlign.title"))
	ui.Separator()

	dp := p.dp()
	st := alignOff
	if dp != nil {
		st = dp.AlignState()
	}
	switch st {
	case alignFailed:
		Warn(T("opAlign.step.failed", "reason", p.reason()))
	case alignWorking:
		Note(T("opAlign.step.working"))
	case alignDone:
		Note(T("opAlign.step.done"))
	default:
		Note(T("opAlign.step.off"))
	}

	p.drawBigBadge(st)

	if ui.ButtonEx(T("opAlign.confirm"), "op-align-confirm", st != alignWorking, true) {
		p.confirm()
	}
	if p.msg == "needTwo" {
		Warn(T("opAlign.needTwo"))
	}
	if Button(T("opAlign.reset"), "op-align-reset") {
		p.resetBoxes()
	}
	enabledDel := p.editor != nil && p.editor.sel >= 0
	if ButtonOn(T("opAlign.delete"), "op-align-delete", enabledDel) {
		p.deleteSelected()
	}
	if st == alignDone {
		if Button(T("opAlign.toGolden"), "op-align-to-golden") {
			p.a.ShowPage(goldenPageID)
		}
	}

	ui.Separator()
	Text(T("flow.viewLabel"))
	dp2 := p.dp()
	var raw, aligned *core.GrayImage
	if dp2 != nil {
		raw = dp2.Raw()
		aligned = dp2.AlignedImage()
	}
	for i, v := range flowViews {
		if i > 0 {
			WrapSameLine(T("flow.view." + v.key()))
		}
		enabled := true
		if v == flowViewRaw && raw == nil {
			enabled = false
		}
		if v == flowViewAligned && aligned == nil {
			enabled = false
		}
		if ToolButton(T("flow.view."+v.key()), "flow-view-"+v.key(), p.shownView == v) && enabled {
			p.view = v
			p.refreshImage(true)
		}
	}

	if p.mouseOverPicture() && ui.KeyPressed("Delete") {
		p.deleteSelected()
	}
}

// drawOperatorRight shows the short numbered help.
func (p *AlignFlowPanel) drawOperatorRight() {
	Note(T("opAlign.help.1"))
	Note(T("opAlign.help.2"))
	Note(T("opAlign.help.3"))
	Note(T("opAlign.help.4"))
}

// drawBigBadge draws a large status lamp in the operator page (colours as in Badge).
func (p *AlignFlowPanel) drawBigBadge(st alignState) {
	level := "manual"
	switch st {
	case alignDone:
		level = "ok"
	case alignFailed:
		level = "ng"
	}
	ui.AlignBanner(alignStateText(st), level)
}
