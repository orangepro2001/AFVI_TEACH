package app

import (
	"fmt"
	"strconv"

	"afvi/teach-imgui/internal/golden"
)

// userMaskUI is the state of the UserMask section on the defect page. The target of a region
// is the sub-tab selection (DefectPanel.focus.target).
type userMaskUI struct {
	active bool // a drawing tool is selected: the frame tool owns left drags on the picture
	frame  *FrameTool
}

// newUserMaskUI returns the default state: no tool, a rectangle frame tool.
func newUserMaskUI() userMaskUI {
	return userMaskUI{frame: NewFrameTool()}
}

// userBoard returns the golden board of the app in detect mode, nil otherwise.
func (p *DefectPanel) userBoard() *goldenBoard {
	if p == nil || p.mode != modeDetect || p.a == nil {
		return nil
	}
	return p.a.goldenBoard()
}

// userMaskDrawable reports whether regions may be drawn: the board is built and, when the
// alignment is required, it is finished.
func (p *DefectPanel) userMaskDrawable() bool {
	b := p.userBoard()
	if b == nil || !b.Ready() {
		return false
	}
	flow := p.a.alignFlow()
	if flow == nil || !flow.Required() {
		return true
	}
	return p.AlignState() == alignDone
}

// userMaskStop drops the drawing tool and hands left drags back to the viewer.
func (p *DefectPanel) userMaskStop() {
	p.um.active = false
	p.um.frame.Reset()
	if p.a != nil && p.a.Viewer != nil {
		p.a.Viewer.SetDoubleClickFit(true)
	}
}

// userMaskPointer forwards a viewer event to the frame tool while it is active.
func (p *DefectPanel) userMaskPointer(ev PointerEvent) bool {
	if p == nil || !p.um.active || p.um.frame == nil {
		return false
	}
	b := p.userBoard()
	if b == nil {
		return false
	}
	if !p.userMaskDrawable() {
		p.um.frame.Reset()
		return false
	}
	switch ev.Kind {
	case PointerDown, PointerMove, PointerUp, PointerRightClick:
	default:
		return false
	}
	owned, done := p.um.frame.Pointer(ev)
	if done != nil {
		if len(b.UserMask().Regions) >= golden.MaxRegions {
			p.a.SetHint(T("defect.usermask.full", "n", strconv.Itoa(golden.MaxRegions)))
		} else {
			b.AddUserRegion(p.focus.target, goldenShapeFromFrame(done))
		}
	}
	if p.a != nil && p.a.Viewer != nil {
		p.a.Viewer.SetDoubleClickFit(!p.um.frame.Busy())
	}
	return owned
}

// userMaskTargetName returns the literal name of a target.
func userMaskTargetName(t golden.Target) string {
	if t == golden.TargetSR {
		return "SR"
	}
	return "Metal"
}

// userMaskShapeName returns the translated name of a shape kind.
func userMaskShapeName(k golden.ShapeKind) string {
	switch k {
	case golden.ShapeEllipse:
		return T("golden.shape.ellipse")
	case golden.ShapePolygon:
		return T("golden.shape.polygon")
	}
	return T("golden.shape.rect")
}

// userMaskRegionLabel returns "#N  Metal|SR  <shape name>".
func userMaskRegionLabel(i int, r golden.Region) string {
	return fmt.Sprintf("#%d  %s  %s", i+1, userMaskTargetName(r.Target), userMaskShapeName(r.Shape.Kind))
}

// drawUserMaskSection draws the UserMask section of the left column.
func (p *DefectPanel) drawUserMaskSection() {
	b := p.userBoard()
	if b == nil {
		return
	}
	if !Section("UserMask", "defect-usermask") {
		if p.um.active {
			p.userMaskStop()
		}
		return
	}
	Note(T("defect.usermask.note"))
	if !b.Ready() {
		Note(T("defect.usermask.wait"))
		return
	}
	drawable := p.userMaskDrawable()
	if !drawable {
		// The tool stays selected while the alignment is redone; drags pan the picture meanwhile.
		Note(T("defect.usermask.wait"))
	}
	if drawable {
		// The target follows the second-level sub-tab strip.
		Note(T("defect.usermask.target", "name", userMaskTargetName(p.focus.target)))

		// Shape buttons switch the drawing tool on and off.
		shapes := []struct {
			id    string
			label string
			kind  FrameKind
		}{
			{"defect-um-rect", T("golden.shape.rect"), FrameRect},
			{"defect-um-ellipse", T("golden.shape.ellipse"), FrameEllipse},
			{"defect-um-pen", T("golden.shape.polygon"), FramePen},
		}
		for i, s := range shapes {
			if i > 0 {
				WrapSameLine(s.label)
			}
			active := p.um.active && p.um.frame != nil && p.um.frame.Kind == s.kind
			if ToolButton(s.label, s.id, active) {
				if active {
					p.userMaskStop()
				} else {
					p.um.frame.Kind = s.kind
					p.um.frame.Reset()
					p.um.active = true
					p.a.SetHint(T("defect.usermask.hint", "name", userMaskTargetName(p.focus.target)))
				}
			}
		}
	}

	// Region list.
	um := b.UserMask()
	const maxRows = 16
	shown := len(um.Regions)
	if shown > maxRows {
		shown = maxRows
	}
	for i := 0; i < shown; i++ {
		r := um.Regions[i]
		Text(userMaskRegionLabel(i, r))
		SameLine()
		if Button(T("defect.usermask.delete"), fmt.Sprintf("defect-um-del-%d", i)) {
			b.RemoveUserRegion(i)
		}
	}
	if len(um.Regions) > maxRows {
		Note(T("defect.usermask.more", "n", strconv.Itoa(len(um.Regions)-maxRows)))
	}

	// Removed pixel counts.
	if d := b.Derived(); d != nil {
		rows := [][2]string{
			{"Metal", fmt.Sprintf("%d %s", d.userMetalPx, T("defect.usermask.removed"))},
			{"SR", fmt.Sprintf("%d %s", d.userSRPx, T("defect.usermask.removed"))},
		}
		KVTable("defect-um-kv", rows)
	}

	// Clear buttons: only drawn when there are regions of that kind; the first one starts a line.
	onLine := false
	if um.Count(golden.TargetMetal) > 0 {
		if Button(T("defect.usermask.clear.metal"), "defect-um-clear-metal") {
			b.ClearUserMask(false, golden.TargetMetal)
		}
		onLine = true
	}
	if um.Count(golden.TargetSR) > 0 {
		if onLine {
			WrapSameLine(T("defect.usermask.clear.sr"))
		}
		if Button(T("defect.usermask.clear.sr"), "defect-um-clear-sr") {
			b.ClearUserMask(false, golden.TargetSR)
		}
		onLine = true
	}
	if !um.Empty() {
		if onLine {
			WrapSameLine(T("defect.usermask.clear.all"))
		}
		if Button(T("defect.usermask.clear.all"), "defect-um-clear-all") {
			b.ClearUserMask(true, golden.TargetMetal)
		}
	}

	// Save: the golden board file also stores the UserMask.
	if Button(T("defect.usermask.save"), "defect-um-save") {
		_ = b.Save()
	}
	if b.Confirmed() {
		Note(T("defect.usermask.saved"))
	} else {
		Note(T("defect.usermask.unsaved"))
	}
	if b.Busy() {
		Note(T("defect.usermask.busy"))
	}
}

// userMaskTargetColor returns the outline colour of a target.
func userMaskTargetColor(t golden.Target) uint32 {
	if t == golden.TargetSR {
		return rgba(0, 200, 0, 255)
	}
	return rgba(255, 0, 255, 255)
}

// userMaskBounds returns the ordered bounding box of a region shape.
func userMaskBounds(s golden.Shape) (x, y, w, h float64) {
	if s.Kind == golden.ShapePolygon {
		if len(s.Points) == 0 {
			return 0, 0, 0, 0
		}
		minX, maxX := s.Points[0].X, s.Points[0].X
		minY, maxY := s.Points[0].Y, s.Points[0].Y
		for _, pt := range s.Points[1:] {
			if pt.X < minX {
				minX = pt.X
			}
			if pt.X > maxX {
				maxX = pt.X
			}
			if pt.Y < minY {
				minY = pt.Y
			}
			if pt.Y > maxY {
				maxY = pt.Y
			}
		}
		return minX, minY, maxX - minX, maxY - minY
	}
	x = s.P0.X
	if s.P1.X < x {
		x = s.P1.X
	}
	y = s.P0.Y
	if s.P1.Y < y {
		y = s.P1.Y
	}
	w = s.P1.X - s.P0.X
	if w < 0 {
		w = -w
	}
	h = s.P1.Y - s.P0.Y
	if h < 0 {
		h = -h
	}
	return x, y, w, h
}

// userMaskPolygonOutline draws the edges of a polygon without allocating.
func userMaskPolygonOutline(o *Overlay, pts []golden.Pt, col uint32, thickness float32) {
	n := len(pts)
	if n < 2 {
		return
	}
	for i := 0; i+1 < n; i++ {
		o.Line(pts[i].X, pts[i].Y, pts[i+1].X, pts[i+1].Y, col, thickness)
	}
	o.Line(pts[n-1].X, pts[n-1].Y, pts[0].X, pts[0].Y, col, thickness)
}

// userMaskOverlay draws the outline of every region and the frame in progress.
func (p *DefectPanel) userMaskOverlay(o *Overlay) {
	if o == nil {
		return
	}
	b := p.userBoard()
	if b != nil && b.Ready() {
		um := b.UserMask()
		for i, r := range um.Regions {
			col := userMaskTargetColor(r.Target)
			s := r.Shape
			if s.Kind == golden.ShapePolygon && len(s.Points) < 3 {
				continue
			}
			x0, y0, w, h := userMaskBounds(s)
			if !o.Visible(x0, y0, w, h) {
				continue
			}
			th := float32(1.5)
			if r.Target == p.focus.target {
				th = 3
			}
			switch s.Kind {
			case golden.ShapeRect:
				o.Rect(x0, y0, w, h, col, th)
			case golden.ShapeEllipse:
				drawEllipseOutline(o, s.P0.X, s.P0.Y, s.P1.X, s.P1.Y, col, th)
			case golden.ShapePolygon:
				userMaskPolygonOutline(o, s.Points, col, th)
			}
			userMaskLabel(o, x0, y0, w, h, col, r.Target, i)
		}
	}
	if p.um.active && p.um.frame != nil && p.a != nil && p.a.Viewer != nil && p.userMaskDrawable() {
		mx, my, _ := p.a.Viewer.Mouse()
		p.um.frame.Draw(o, mx, my, rgba(255, 255, 255, 255))
	}
}

// userMaskLabel draws "Metal N" / "SR N" at the top-left of the region box when it is wide
// enough on screen.
func userMaskLabel(o *Overlay, x, y, w, h float64, col uint32, t golden.Target, i int) {
	if w*float64(o.Scale()) < 24 {
		return
	}
	o.Text(x, y, 2, -2, col, fmt.Sprintf("%s %d", userMaskTargetName(t), i+1))
}
