package app

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
	"afvi/teach-imgui/internal/inspect"
)

// userMaskSteps exercises the UserMask regions of the defect page: drawing a Metal and an SR
// region, clearing them and the effect on the inspection (an SR region never removes Metal
// pixels, a Metal region hides the Metal defects it covers, removing the regions restores them).
func (s *Selftest) userMaskSteps() []selftestStep {
	var (
		metalPx   int       // userMetalPx after step 1
		base      int       // defect count of the unedited board
		baseSR    int       // SR defects of the unedited board
		rect      core.Rect // area that hides the first SR defect
		rectShape golden.Shape
	)
	board := func() *goldenBoard { return s.a.golden }
	dp := func() *DefectPanel { return defectPanelOf(s.a) }
	canvasDrag := func(s *Selftest) []viewerInput {
		c := s.canvasCenter()
		return dragInput(btnLeft, imgui.NewVec2(c.X-40, c.Y-30), imgui.NewVec2(80, 60), 4)
	}
	// inRect reports whether the point is inside the rectangle.
	inRect := func(r core.Rect, x, y float64) bool {
		return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
	}
	// srDefectsInRect returns the SR defects whose centroid lies inside rect.
	srDefectsInRect := func(rep *inspect.Report) int {
		if rep == nil {
			return 0
		}
		n := 0
		for _, d := range rep.Defects {
			if d.Material == inspect.SR && inRect(rect, d.CentroidX, d.CentroidY) {
				n++
			}
		}
		return n
	}
	// countMaterial returns the number of defects of material m.
	countMaterial := func(rep *inspect.Report, m inspect.Material) int {
		if rep == nil {
			return 0
		}
		n := 0
		for _, d := range rep.Defects {
			if d.Material == m {
				n++
			}
		}
		return n
	}
	return []selftestStep{
		{
			name: "defect-usermask-ready",
			enter: func(a *App) {
				a.theme = ThemeDark
				SetLang(LangZh)
				a.golden.Auto()
				a.golden.SetPadding(golden.DefaultPadding)
				a.golden.SetSRInset(golden.DefaultSRInset)
				a.golden.ClearUserMask(true, golden.TargetMetal)
				s.panel("roi").(*AlignFlowPanel).confirm()
				gotoPage(a, "defect")
				// The page seeds its own strokes; they stay for the following steps.
				dp().paramsOpen = false
				dp().RunInspect()
			},
			check: func(s *Selftest) (map[string]any, error) {
				if p := s.a.ActivePanel(); p == nil || p.ID() != "defect" {
					return nil, fmt.Errorf("defect-usermask-ready: the defect page is not shown")
				}
				// Drags only draw once the board is built and the alignment is done.
				if !dp().userMaskDrawable() {
					return nil, fmt.Errorf("defect-usermask-ready: regions cannot be drawn yet (align state %s)", dp().AlignState().key())
				}
				return map[string]any{"drawable": true}, nil
			},
		},
		{
			name: "defect-usermask-draw",
			enter: func(a *App) {
				// Select the tool exactly as a click on the Metal / rectangle buttons would.
				dp().um.active = true
				dp().focus.target = golden.TargetMetal
				dp().um.frame.Kind = FrameRect
				dp().um.frame.Reset()
			},
			input: canvasDrag,
			shot:  "native-defect-usermask-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				if p := s.a.ActivePanel(); p == nil || p.ID() != "defect" {
					return nil, fmt.Errorf("defect-usermask-draw: the defect page is not shown")
				}
				b := board()
				if b == nil || !b.Ready() {
					return nil, fmt.Errorf("defect-usermask-draw: the golden board is not ready")
				}
				um := b.UserMask()
				if len(um.Regions) != 1 {
					return nil, fmt.Errorf("defect-usermask-draw: %d regions, want 1", len(um.Regions))
				}
				r := um.Regions[0]
				if r.Target != golden.TargetMetal {
					return nil, fmt.Errorf("defect-usermask-draw: region target %v, want metal", r.Target)
				}
				if r.Shape.Kind != golden.ShapeRect {
					return nil, fmt.Errorf("defect-usermask-draw: region kind %v, want rect", r.Shape.Kind)
				}
				d := b.Derived()
				if d == nil {
					return nil, fmt.Errorf("defect-usermask-draw: no derived masks")
				}
				if d.userMetalPx <= 0 {
					return nil, fmt.Errorf("defect-usermask-draw: %d Metal pixels removed, want > 0", d.userMetalPx)
				}
				if d.userSRPx != 0 {
					return nil, fmt.Errorf("defect-usermask-draw: %d SR pixels removed by a Metal region, want 0", d.userSRPx)
				}
				metalPx = d.userMetalPx
				return map[string]any{
					"regions":     len(um.Regions),
					"userMetalPx": d.userMetalPx,
					"userSRPx":    d.userSRPx,
				}, nil
			},
		},
		{
			name: "golden-user-sr-region",
			enter: func(a *App) {
				// The picture centre is the chip (Metal), so the SR region goes around an SR point.
				px, py := dp().findSrPoint()
				x, y := float64(px), float64(py)
				dp().um.active = true
				dp().focus.target = golden.TargetSR
				dp().um.frame.Reset()
				a.golden.AddUserRegion(golden.TargetSR, golden.Shape{
					Kind: golden.ShapeRect,
					P0:   golden.Pt{X: x - 60, Y: y - 60},
					P1:   golden.Pt{X: x + 60, Y: y + 60},
				})
			},
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				um := b.UserMask()
				if len(um.Regions) != 2 {
					return nil, fmt.Errorf("golden-user-sr-region: %d regions, want 2", len(um.Regions))
				}
				if r := um.Regions[1]; r.Target != golden.TargetSR {
					return nil, fmt.Errorf("golden-user-sr-region: second region target %v, want sr", r.Target)
				}
				d := b.Derived()
				if d == nil {
					return nil, fmt.Errorf("golden-user-sr-region: no derived masks")
				}
				if d.userSRPx <= 0 {
					return nil, fmt.Errorf("golden-user-sr-region: %d SR pixels removed, want > 0", d.userSRPx)
				}
				if d.userMetalPx != metalPx {
					return nil, fmt.Errorf("golden-user-sr-region: %d Metal pixels removed, was %d after the Metal region", d.userMetalPx, metalPx)
				}
				dp().userMaskStop()
				return map[string]any{
					"regions":     len(um.Regions),
					"userMetalPx": d.userMetalPx,
					"userSRPx":    d.userSRPx,
				}, nil
			},
		},
		{
			name: "golden-user-clear",
			enter: func(a *App) {
				a.golden.ClearUserMask(true, golden.TargetMetal)
			},
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				if !b.UserMask().Empty() {
					return nil, fmt.Errorf("golden-user-clear: %d regions left, want 0", len(b.UserMask().Regions))
				}
				d := b.Derived()
				if d == nil {
					return nil, fmt.Errorf("golden-user-clear: no derived masks")
				}
				if d.userMetalPx != 0 || d.userSRPx != 0 {
					return nil, fmt.Errorf("golden-user-clear: %d Metal / %d SR pixels removed, want 0 / 0", d.userMetalPx, d.userSRPx)
				}
				return map[string]any{"userMetalPx": d.userMetalPx, "userSRPx": d.userSRPx}, nil
			},
		},
		{
			name: "golden-user-baseline",
			enter: func(a *App) {
				a.theme = ThemeDark
				SetLang(LangZh)
				dp().channel = core.ChRed
				// Only the default tiny defects (all deep inside SR) take part in the following steps.
				dp().strokes = nil
				dp().addAutoDefects()
				s.panel("roi").(*AlignFlowPanel).confirm()
				dp().resetInspect()
				dp().renderRev++
				dp().RunInspect()
			},
			check: func(s *Selftest) (map[string]any, error) {
				rep := dp().Report()
				if rep == nil {
					return nil, fmt.Errorf("golden-user-baseline: no report")
				}
				var d0 *inspect.Defect
				for i := range rep.Defects {
					if rep.Defects[i].Material == inspect.SR {
						d0 = &rep.Defects[i]
						break
					}
				}
				if d0 == nil {
					return nil, fmt.Errorf("golden-user-baseline: no SR defect in the report (Metal %d, SR %d, Open %d, Base %d), cannot place the region",
						countMaterial(rep, inspect.Metal), countMaterial(rep, inspect.SR),
						countMaterial(rep, inspect.Open), countMaterial(rep, inspect.Base))
				}
				w, h := s.a.Scene.W, s.a.Scene.H
				x0 := float64(d0.BBox.X - 12)
				y0 := float64(d0.BBox.Y - 12)
				x1 := float64(d0.BBox.X + d0.BBox.W + 12)
				y1 := float64(d0.BBox.Y + d0.BBox.H + 12)
				if x0 < 0 {
					x0 = 0
				}
				if y0 < 0 {
					y0 = 0
				}
				if x1 > float64(w) {
					x1 = float64(w)
				}
				if y1 > float64(h) {
					y1 = float64(h)
				}
				rect = core.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
				rectShape = golden.Shape{Kind: golden.ShapeRect, P0: golden.Pt{X: x0, Y: y0}, P1: golden.Pt{X: x1, Y: y1}}
				base = len(rep.Defects)
				baseSR = countMaterial(rep, inspect.SR)
				return map[string]any{
					"defects":   base,
					"sr":        baseSR,
					"rect":      fmt.Sprintf("%.0f,%.0f %.0f×%.0f", rect.X, rect.Y, rect.W, rect.H),
					"centroidX": d0.CentroidX,
					"centroidY": d0.CentroidY,
				}, nil
			},
		},
		{
			name: "golden-user-metal-keeps-sr",
			enter: func(a *App) {
				a.golden.AddUserRegion(golden.TargetMetal, rectShape)
				dp().RunInspect()
			},
			check: func(s *Selftest) (map[string]any, error) {
				rep := dp().Report()
				if rep == nil {
					return nil, fmt.Errorf("golden-user-metal-keeps-sr: no report")
				}
				if srDefectsInRect(rep) == 0 {
					return nil, fmt.Errorf("golden-user-metal-keeps-sr: no SR defect inside the Metal region any more")
				}
				if got := countMaterial(rep, inspect.SR); got != baseSR {
					return nil, fmt.Errorf("golden-user-metal-keeps-sr: %d SR defects, want %d", got, baseSR)
				}
				d := board().Derived()
				userMetalPx := 0
				if d != nil {
					userMetalPx = d.userMetalPx
				}
				return map[string]any{
					"srInRect":    srDefectsInRect(rep),
					"defects":     len(rep.Defects),
					"userMetalPx": userMetalPx,
				}, nil
			},
		},
		{
			name: "golden-user-sr-removes",
			enter: func(a *App) {
				a.golden.AddUserRegion(golden.TargetSR, rectShape)
				dp().RunInspect()
			},
			shot: "native-defect-usermask-result-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				rep := dp().Report()
				if rep == nil {
					return nil, fmt.Errorf("golden-user-sr-removes: no report")
				}
				if n := srDefectsInRect(rep); n != 0 {
					return nil, fmt.Errorf("golden-user-sr-removes: %d SR defects still inside the region", n)
				}
				if len(rep.Defects) > base-1 {
					return nil, fmt.Errorf("golden-user-sr-removes: %d defects, want at most %d", len(rep.Defects), base-1)
				}
				b := board()
				d := b.Derived()
				if d == nil || d.userSRPx <= 0 {
					return nil, fmt.Errorf("golden-user-sr-removes: no SR pixels removed by the region")
				}
				if !dp().learned[core.ChRed] {
					return nil, fmt.Errorf("golden-user-sr-removes: the UserMask edit made the detection forget its intervals")
				}
				if dp().goldenRev != b.LearnRev() {
					return nil, fmt.Errorf("golden-user-sr-removes: detection learns revision %d, board at %d", dp().goldenRev, b.LearnRev())
				}
				if st := dp().AlignState(); st != alignDone {
					return nil, fmt.Errorf("golden-user-sr-removes: align state %s, want done", st.key())
				}
				return map[string]any{
					"defects":  len(rep.Defects),
					"base":     base,
					"userSRPx": d.userSRPx,
					"learnRev": b.LearnRev(),
				}, nil
			},
		},
		{
			name: "golden-user-restore",
			enter: func(a *App) {
				a.golden.ClearUserMask(true, golden.TargetMetal)
				dp().RunInspect()
			},
			check: func(s *Selftest) (map[string]any, error) {
				rep := dp().Report()
				if rep == nil {
					return nil, fmt.Errorf("golden-user-restore: no report")
				}
				got := len(rep.Defects)
				// Leave the alignment disarmed again, like the golden steps do.
				s.a.align.SetRois(s.a.align.Rois)
				dp().RequestCapture()
				if got != base {
					return nil, fmt.Errorf("golden-user-restore: %d defects, want %d again", got, base)
				}
				return map[string]any{"defects": got, "base": base}, nil
			},
		},
	}
}
