package app

import (
	"fmt"
	"math"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/core"
)

// selectionState 是缺陷面板的选择工具状态。
type selectionState struct {
	tool  PaletteTool                 // PaletteNone、PalettePick、PaletteRect、PaletteEllipse、PalettePolygon
	frame *FrameTool                  // 三个画框工具使用的橡皮筋/钢笔
	set   map[*core.DefectStroke]bool // 选中的 GV 笔画（以稳定的笔画指针为键）
}

// PaletteSupports 报告面板支持的选择工具。
func (p *DefectPanel) PaletteSupports(t PaletteTool) bool {
	switch t {
	case PalettePick, PaletteRect, PaletteEllipse, PalettePolygon:
		return true
	}
	return false
}

// PaletteTool 返回当前激活的选择工具；笔刷/画框放置工具优先，此时选择工具处于休眠。
func (p *DefectPanel) PaletteTool() PaletteTool {
	if p.tool != toolNone {
		return PaletteNone
	}
	return p.sel.tool
}

// SetPaletteTool 切换选择工具。
func (p *DefectPanel) SetPaletteTool(t PaletteTool) {
	p.sel.tool = t
	if p.sel.frame == nil {
		p.sel.frame = NewFrameTool()
	}
	p.sel.frame.Reset()
	switch t {
	case PaletteRect:
		p.sel.frame.Kind = FrameRect
	case PaletteEllipse:
		p.sel.frame.Kind = FrameEllipse
	case PalettePolygon:
		p.sel.frame.Kind = FramePen
	}
	if t != PaletteNone {
		p.tool = toolNone
		p.dragging = false
		if p.frame != nil {
			p.frame.Reset()
		}
	}
	if p.a != nil && p.a.Viewer != nil {
		p.a.Viewer.SetDoubleClickFit(true)
	}
	switch t {
	case PalettePick:
		p.a.SetHint(T("defect.sel.hint.pick"))
	case PaletteRect:
		p.a.SetHint(T("defect.sel.hint.rect"))
	case PaletteEllipse:
		p.a.SetHint(T("defect.sel.hint.ellipse"))
	case PalettePolygon:
		p.a.SetHint(T("defect.sel.hint.polygon"))
	default:
		p.a.SetHint("")
	}
}

// selectPointer 处理选择工具的指针事件。
func (p *DefectPanel) selectPointer(ev PointerEvent) bool {
	switch p.sel.tool {
	case PalettePick:
		if ev.Kind == PointerDown {
			p.pickAt(ev.X, ev.Y, ev.Ctrl)
		}
		return false
	case PaletteRect, PaletteEllipse, PalettePolygon:
		if p.sel.frame == nil {
			return false
		}
		owned, res := p.sel.frame.Pointer(ev)
		if res != nil {
			p.selectByFrame(*res, ev.Ctrl)
		} else if ev.Kind == PointerUp && p.sel.tool != PalettePolygon {
			p.pickAt(ev.X, ev.Y, ev.Ctrl)
		}
		if p.a != nil && p.a.Viewer != nil {
			p.a.Viewer.SetDoubleClickFit(!p.sel.frame.Busy())
		}
		return owned
	}
	return false
}

// pickAt 选中离 (x,y) 最近的笔画；容差为 6 屏幕像素。
func (p *DefectPanel) pickAt(x, y float64, ctrl bool) {
	zoom := float64(1)
	if p.a != nil && p.a.Viewer != nil {
		zoom = float64(p.a.Viewer.Zoom())
	}
	if zoom <= 0 {
		zoom = 1
	}
	tol := 6 / zoom
	var best *core.DefectStroke
	bestDist := math.Inf(1)
	bestArea := math.Inf(1)
	for i := range p.strokes {
		s := p.strokes[i].gv
		if s == nil {
			continue
		}
		d := core.StrokeDistance(s, x, y)
		if d > tol {
			continue
		}
		x0, y0, x1, y1 := core.StrokeBounds(s)
		area := (x1 - x0) * (y1 - y0)
		if d < bestDist-1e-9 || (math.Abs(d-bestDist) <= 1e-9 && area < bestArea) {
			best = s
			bestDist = d
			bestArea = area
		}
	}
	if p.sel.set == nil {
		p.sel.set = map[*core.DefectStroke]bool{}
	}
	if ctrl {
		if best != nil {
			if p.sel.set[best] {
				delete(p.sel.set, best)
			} else {
				p.sel.set[best] = true
			}
		}
		return
	}
	p.sel.set = map[*core.DefectStroke]bool{}
	if best != nil {
		p.sel.set[best] = true
	}
}

// selectByFrame 选中中心落在画框内的笔画。
func (p *DefectPanel) selectByFrame(res FrameResult, ctrl bool) {
	if p.sel.set == nil {
		p.sel.set = map[*core.DefectStroke]bool{}
	}
	if !ctrl {
		p.sel.set = map[*core.DefectStroke]bool{}
	}
	for i := range p.strokes {
		s := p.strokes[i].gv
		if s == nil {
			continue
		}
		cx, cy := core.StrokeCenter(s)
		hit := false
		switch res.Kind {
		case FrameRect:
			minX := math.Min(res.P0.X, res.P1.X)
			maxX := math.Max(res.P0.X, res.P1.X)
			minY := math.Min(res.P0.Y, res.P1.Y)
			maxY := math.Max(res.P0.Y, res.P1.Y)
			hit = cx >= minX && cx <= maxX && cy >= minY && cy <= maxY
		case FrameEllipse:
			hit = core.PointInEllipse(res.P0.X, res.P0.Y, res.P1.X, res.P1.Y, cx, cy)
		case FramePen:
			hit = core.PointInPolygon(res.Points, cx, cy)
		}
		if hit {
			p.sel.set[s] = true
		}
	}
}

// pruneSelection 丢弃已不在 p.strokes 中的选中指针。
func (p *DefectPanel) pruneSelection() {
	if len(p.sel.set) == 0 {
		return
	}
	live := make(map[*core.DefectStroke]bool, len(p.strokes))
	for i := range p.strokes {
		if p.strokes[i].gv != nil {
			live[p.strokes[i].gv] = true
		}
	}
	for s := range p.sel.set {
		if !live[s] {
			delete(p.sel.set, s)
		}
	}
}

// clearSelection 清空选择。
func (p *DefectPanel) clearSelection() {
	p.sel.set = nil
}

// selectedStrokes 按 p.strokes 顺序返回选中的笔画。
func (p *DefectPanel) selectedStrokes() []*core.DefectStroke {
	var out []*core.DefectStroke
	for i := range p.strokes {
		s := p.strokes[i].gv
		if s != nil && p.sel.set[s] {
			out = append(out, s)
		}
	}
	return out
}

// applyToSelection 对每个选中笔画执行 fn，然后重新渲染并检出。
func (p *DefectPanel) applyToSelection(fn func(s *core.DefectStroke)) {
	sel := p.selectedStrokes()
	if len(sel) == 0 {
		return
	}
	for _, s := range sel {
		fn(s)
	}
	p.renderRev++
	p.requestRun()
}

// strokeDetected 报告 Inspect 是否检出了该笔画（仅用于显示）。
func (p *DefectPanel) strokeDetected(s *core.DefectStroke) bool {
	if p.lastReport == nil {
		return false
	}
	x0, y0, x1, y1 := core.StrokeBounds(s)
	x0 -= 2
	y0 -= 2
	x1 += 2
	y1 += 2
	for _, d := range p.lastReport.Defects {
		bx0 := float64(d.BBox.X)
		by0 := float64(d.BBox.Y)
		bx1 := float64(d.BBox.X + d.BBox.W)
		by1 := float64(d.BBox.Y + d.BBox.H)
		if bx0 < x1 && x0 < bx1 && by0 < y1 && y0 < by1 {
			return true
		}
	}
	return false
}

// deleteSelected 从 p.strokes 中删除选中的笔画。
func (p *DefectPanel) deleteSelected() {
	if len(p.sel.set) == 0 {
		return
	}
	kept := p.strokes[:0]
	for i := range p.strokes {
		s := p.strokes[i].gv
		if s != nil && p.sel.set[s] {
			continue
		}
		kept = append(kept, p.strokes[i])
	}
	p.strokes = kept
	p.clearSelection()
	p.renderRev++
	p.requestRun()
}

// handleSelectKeys 处理选择工具的键盘快捷键。
func (p *DefectPanel) handleSelectKeys() {
	if p.paramsOpen {
		return
	}
	if imgui.CurrentIO().WantTextInput() {
		return
	}
	if imgui.IsKeyPressedBool(imgui.KeyEscape) {
		if p.sel.frame != nil && p.sel.frame.Busy() {
			p.sel.frame.Reset()
		} else {
			p.clearSelection()
		}
	}
	if imgui.IsKeyPressedBool(imgui.KeyBackspace) {
		if p.sel.frame != nil {
			p.sel.frame.Undo()
		}
	}
	if imgui.IsKeyPressedBool(imgui.KeyDelete) {
		p.deleteSelected()
	}
}

// drawSelectionOverlay 绘制选中笔画的轮廓与进行中的画框。
func (p *DefectPanel) drawSelectionOverlay(o *Overlay) {
	for _, s := range p.selectedStrokes() {
		switch s.Shape {
		case core.ShapeSpot:
			o.Circle(s.X, s.Y, math.Max(s.Radius, 7/float64(o.Scale())), OverlayWarn, 2)
		case core.ShapeRect:
			o.Rect(math.Min(s.X, s.X2), math.Min(s.Y, s.Y2), math.Abs(s.X2-s.X), math.Abs(s.Y2-s.Y), OverlayWarn, 2)
		case core.ShapeEllipse:
			drawEllipseOutline(o, s.X, s.Y, s.X2, s.Y2, OverlayWarn, 2)
		case core.ShapePolygon:
			drawPolygonOutline(o, s.Points, OverlayWarn, 2)
		default:
			o.Line(s.X, s.Y, s.X2, s.Y2, OverlayWarn, 2)
		}
		x0, y0, x1, y1 := core.StrokeBounds(s)
		minSize := 14 / float64(o.Scale())
		if x1-x0 < minSize {
			cx := (x0 + x1) / 2
			x0 = cx - minSize/2
			x1 = cx + minSize/2
		}
		if y1-y0 < minSize {
			cy := (y0 + y1) / 2
			y0 = cy - minSize/2
			y1 = cy + minSize/2
		}
		o.Rect(x0, y0, x1-x0, y1-y0, OverlayWarn, 1)
	}
	if p.tool == toolNone && p.sel.frame != nil {
		switch p.sel.tool {
		case PaletteRect, PaletteEllipse, PalettePolygon:
			mx, my, _ := p.a.Viewer.Mouse()
			p.sel.frame.Draw(o, mx, my, OverlayWarn)
		}
	}
}

// drawSelectionEditor 绘制浮动选择编辑器。
func (p *DefectPanel) drawSelectionEditor() {
	p.pruneSelection()
	sel := p.selectedStrokes()
	if len(sel) == 0 {
		return
	}
	origin, size := p.a.Viewer.Canvas()
	imgui.SetNextWindowPosV(imgui.NewVec2(origin.X+size.X-330-56, origin.Y+size.Y-372), imgui.CondFirstUseEver, imgui.NewVec2(0, 0))
	imgui.SetNextWindowSizeV(imgui.NewVec2(320, 360), imgui.CondFirstUseEver)
	flags := imgui.WindowFlagsNoTitleBar | imgui.WindowFlagsNoCollapse | imgui.WindowFlagsNoSavedSettings |
		imgui.WindowFlagsNoScrollbar | imgui.WindowFlagsNoScrollWithMouse
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(10, 0))
	open := imgui.BeginV("##defect-selection", nil, flags)
	imgui.PopStyleVar()
	if !open {
		imgui.End()
		return
	}
	wp, ws := imgui.WindowPos(), imgui.WindowSize()
	dl := imgui.WindowDrawList()
	DrawEdge(dl, wp, imgui.NewVec2(wp.X+ws.X, wp.Y+ws.Y), EdgeWindow)
	capMin := imgui.NewVec2(wp.X+3, wp.Y+3)
	capMax := imgui.NewVec2(wp.X+ws.X-3, wp.Y+23)
	Caption(dl, capMin, capMax, T("defect.sel.title"), true)
	imgui.SetCursorScreenPos(imgui.NewVec2(capMax.X-18, capMin.Y+3))
	if ButtonSized("x", "sel-close", imgui.NewVec2(16, 14), true) {
		p.clearSelection()
		imgui.End()
		return
	}

	imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+10, wp.Y+30))
	imgui.BeginChildStrV("##defect-selection-body", imgui.NewVec2(ws.X-20, ws.Y-40), 0, 0)

	Note(T("defect.sel.count", "n", fmt.Sprint(len(sel))))
	if p.renderRev != p.renderedRev {
		Note(T("defect.sel.stale"))
	} else {
		found := 0
		total := 0
		for _, s := range sel {
			if !p.strokeEnabled(s) {
				continue
			}
			total++
			if p.strokeDetected(s) {
				found++
			}
		}
		Note(T("defect.sel.detected", "found", fmt.Sprint(found), "total", fmt.Sprint(total)))
	}

	if Button(T("defect.sel.hide"), "sel-hide") {
		p.applyToSelection(func(s *core.DefectStroke) {
			scaleAmplitude(s, 0.8)
			if s.Shape == core.ShapeSpot || s.Shape == core.ShapeLine {
				s.Radius = clampF(s.Radius*0.9, 0.5, 40)
			}
		})
	}
	imgui.SameLine()
	if Button(T("defect.sel.clear"), "sel-clear") {
		p.applyToSelection(func(s *core.DefectStroke) {
			scaleAmplitude(s, 1.25)
			if s.Shape == core.ShapeSpot || s.Shape == core.ShapeLine {
				s.Radius = clampF(s.Radius*1.1, 0.5, 40)
			}
		})
	}
	imgui.SameLine()
	if Button(T("defect.sel.delete"), "sel-delete") {
		p.deleteSelected()
		imgui.EndChild()
		imgui.End()
		return
	}
	imgui.SameLine()
	if Button(T("defect.sel.deselect"), "sel-deselect") {
		p.clearSelection()
		imgui.EndChild()
		imgui.End()
		return
	}

	first := sel[0]
	kind := int(first.Kind)
	if Combo(T("defect.sel.kind"), "sel-kind", &kind, []string{
		T("defect.kind.bright"), T("defect.kind.dark"),
	}) {
		k := core.DefectKind(kind)
		p.applyToSelection(func(s *core.DefectStroke) { s.Kind = k })
	}

	amp := int(first.Amplitude)
	if Combo(T("defect.sel.ampMode"), "sel-ampmode", &amp, []string{
		T("defect.amplitude.absolute"), T("defect.amplitude.relative"), T("defect.amplitude.target"),
	}) {
		mode := core.DefectAmplitude(amp)
		p.applyToSelection(func(s *core.DefectStroke) {
			switch mode {
			case core.AmpRelative:
				if s.RelativePct == 0 {
					s.RelativePct = 40
				}
			case core.AmpTarget:
				s.HasTarget = true
				if s.TargetGv == 0 {
					if s.Kind == core.Bright {
						s.TargetGv = 250
					} else {
						s.TargetGv = 10
					}
				}
			default:
				if s.DeltaGv == 0 {
					s.DeltaGv = 60
				}
			}
			s.Amplitude = mode
		})
	}

	switch first.Amplitude {
	case core.AmpRelative:
		v := first.RelativePct
		if SliderF64(T("defect.sel.value.pct"), "sel-value-pct", &v, 1, 300, "%.0f") {
			p.applyToSelection(func(s *core.DefectStroke) { s.RelativePct = v })
		}
	case core.AmpTarget:
		v := first.TargetGv
		if SliderF64(T("defect.sel.value.target"), "sel-value-target", &v, 0, 255, "%.0f") {
			p.applyToSelection(func(s *core.DefectStroke) {
				s.TargetGv = v
				s.HasTarget = true
			})
		}
	default:
		v := first.DeltaGv
		if SliderF64(T("defect.sel.value.delta"), "sel-value-delta", &v, 1, 255, "%.0f") {
			p.applyToSelection(func(s *core.DefectStroke) { s.DeltaGv = v })
		}
	}

	r := first.Radius
	if SliderF64(T("defect.sel.radius"), "sel-radius", &r, 0.5, 40, "%.1f") {
		p.applyToSelection(func(s *core.DefectStroke) {
			if s.Shape == core.ShapeSpot || s.Shape == core.ShapeLine {
				s.Radius = r
			}
		})
	}

	soft := first.Softness
	if SliderF64(T("defect.sel.softness"), "sel-softness", &soft, 0, 1, "%.2f") {
		p.applyToSelection(func(s *core.DefectStroke) { s.Softness = soft })
	}

	grain := first.Grain
	if SliderF64(T("defect.sel.grain"), "sel-grain", &grain, 0, 1, "%.2f") {
		p.applyToSelection(func(s *core.DefectStroke) { s.Grain = grain })
	}

	prof := int(first.Profile)
	if Combo(T("defect.sel.profile"), "sel-profile", &prof, []string{
		T("defect.profile.linear"), T("defect.profile.gaussian"),
	}) {
		pr := core.DefectProfile(prof)
		p.applyToSelection(func(s *core.DefectStroke) { s.Profile = pr })
	}

	enabled := p.strokeEnabled(first)
	if Check(T("defect.sel.enabled"), "sel-enabled", &enabled) {
		for i := range p.strokes {
			s := p.strokes[i].gv
			if s != nil && p.sel.set[s] {
				p.strokes[i].enabled = enabled
			}
		}
		p.renderRev++
		p.requestRun()
	}

	imgui.EndChild()
	imgui.End()
}

// strokeEnabled 返回笔画在 p.strokes 中的启用状态。
func (p *DefectPanel) strokeEnabled(s *core.DefectStroke) bool {
	for i := range p.strokes {
		if p.strokes[i].gv == s {
			return p.strokes[i].enabled
		}
	}
	return false
}

// scaleAmplitude 按因子缩放笔画的幅度（目标模式保持不变）。
func scaleAmplitude(s *core.DefectStroke, f float64) {
	switch s.Amplitude {
	case core.AmpRelative:
		s.RelativePct = clampF(s.RelativePct*f, 1, 300)
	case core.AmpTarget:
		// 目标模式保持目标值不变。
	default:
		s.DeltaGv = clampF(s.DeltaGv*f, 1, 255)
	}
}

// clampF 将 v 限制在 [lo, hi]。
func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func init() {
	Messages(map[string]string{
		"defect.sel.hint.pick":    "点选：单击选中最近的笔画，Ctrl+单击加入/移出选择",
		"defect.sel.hint.rect":    "矩形框选：拖动框选中心在框内的笔画，Ctrl+拖动加入选择",
		"defect.sel.hint.ellipse": "椭圆框选：拖动框选中心在椭圆内的笔画，Ctrl+拖动加入选择",
		"defect.sel.hint.polygon": "多边形框选：左键添加顶点，右键闭合，Ctrl+拖动加入选择",
		"defect.sel.title":        "选中的不良",
		"defect.sel.count":        "已选 {n} 个不良",
		"defect.sel.detected":     "检出：{found} / {total} 个已被 Inspect 检出",
		"defect.sel.stale":        "检出状态更新中…",
		"defect.sel.hide":         "更隐蔽",
		"defect.sel.clear":        "更清晰",
		"defect.sel.delete":       "删除",
		"defect.sel.deselect":     "取消选择",
		"defect.sel.kind":         "类型",
		"defect.sel.ampMode":      "幅度模式",
		"defect.sel.radius":       "半径",
		"defect.sel.softness":     "柔和度",
		"defect.sel.grain":        "颗粒",
		"defect.sel.profile":      "边缘轮廓",
		"defect.sel.enabled":      "启用",
		"defect.sel.value.delta":  "ΔGV",
		"defect.sel.value.pct":    "相对幅度 %",
		"defect.sel.value.target": "目标 GV",
	}, map[string]string{
		"defect.sel.hint.pick":    "Pick: click the nearest stroke, Ctrl+click toggles it in the selection",
		"defect.sel.hint.rect":    "Rectangle frame: drag to select strokes whose centre is inside, Ctrl+drag adds",
		"defect.sel.hint.ellipse": "Ellipse frame: drag to select strokes whose centre is inside, Ctrl+drag adds",
		"defect.sel.hint.polygon": "Polygon frame: left click adds a vertex, right click closes, Ctrl+drag adds",
		"defect.sel.title":        "Selected defects",
		"defect.sel.count":        "{n} defects selected",
		"defect.sel.detected":     "Detected: {found} / {total} found by Inspect",
		"defect.sel.stale":        "Detection status updating…",
		"defect.sel.hide":         "More hidden",
		"defect.sel.clear":        "Clearer",
		"defect.sel.delete":       "Delete",
		"defect.sel.deselect":     "Deselect",
		"defect.sel.kind":         "Kind",
		"defect.sel.ampMode":      "Amplitude mode",
		"defect.sel.radius":       "Radius",
		"defect.sel.softness":     "Softness",
		"defect.sel.grain":        "Grain",
		"defect.sel.profile":      "Edge profile",
		"defect.sel.enabled":      "Enabled",
		"defect.sel.value.delta":  "ΔGV",
		"defect.sel.value.pct":    "Relative amplitude %",
		"defect.sel.value.target": "Target GV",
	}, map[string]string{
		"defect.sel.hint.pick":    "선택: 가장 가까운 획을 클릭, Ctrl+클릭으로 선택에 추가/제거",
		"defect.sel.hint.rect":    "사각형 프레임: 중심이 안에 있는 획을 드래그로 선택, Ctrl+드래그로 추가",
		"defect.sel.hint.ellipse": "타원 프레임: 중심이 안에 있는 획을 드래그로 선택, Ctrl+드래그로 추가",
		"defect.sel.hint.polygon": "다각형 프레임: 왼쪽 클릭으로 꼭짓점 추가, 오른쪽 클릭으로 닫기, Ctrl+드래그로 추가",
		"defect.sel.title":        "선택된 불량",
		"defect.sel.count":        "{n}개 불량 선택됨",
		"defect.sel.detected":     "검출: {found} / {total}개가 Inspect에서 검출됨",
		"defect.sel.stale":        "검출 상태 갱신 중…",
		"defect.sel.hide":         "더 숨기기",
		"defect.sel.clear":        "더 선명하게",
		"defect.sel.delete":       "삭제",
		"defect.sel.deselect":     "선택 해제",
		"defect.sel.kind":         "유형",
		"defect.sel.ampMode":      "진폭 모드",
		"defect.sel.radius":       "반경",
		"defect.sel.softness":     "부드러움",
		"defect.sel.grain":        "그레인",
		"defect.sel.profile":      "가장자리 프로파일",
		"defect.sel.enabled":      "사용",
		"defect.sel.value.delta":  "ΔGV",
		"defect.sel.value.pct":    "상대 진폭 %",
		"defect.sel.value.target": "목표 GV",
	})
}
