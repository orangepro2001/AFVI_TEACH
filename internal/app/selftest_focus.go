package app

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/golden"
	"afvi/teach-imgui/internal/inspect"
)

// focusSteps exercises the Metal / SR sub-tab of the defect page: the tint of the viewer
// picture, the result filter and the UserMask target that follows the selected tab.
func (s *Selftest) focusSteps() []selftestStep {
	dp := func() *DefectPanel { return defectPanelOf(s.a) }
	board := func() *goldenBoard { return s.a.golden }
	canvasDrag := func(s *Selftest) []viewerInput {
		c := s.canvasCenter()
		return dragInput(btnLeft, imgui.NewVec2(c.X-40, c.Y-30), imgui.NewVec2(80, 60), 4)
	}
	// materialRows returns how many of the focus rows are defects of material m.
	materialRows := func(rep *inspect.Report, rows []int, m inspect.Material) int {
		if rep == nil {
			return 0
		}
		n := 0
		for _, i := range rows {
			if rep.Defects[i].Material == m {
				n++
			}
		}
		return n
	}
	// polarityRows returns how many of the focus rows are defects of polarity pol.
	polarityRows := func(rep *inspect.Report, rows []int, pol inspect.Polarity) int {
		if rep == nil {
			return 0
		}
		n := 0
		for _, i := range rows {
			if rep.Defects[i].Polarity == pol {
				n++
			}
		}
		return n
	}
	return []selftestStep{
		{
			name: "defect-focus-metal",
			enter: func(a *App) {
				a.theme = ThemeDark
				SetLang(LangZh)
				// The earlier steps leave the alignment disarmed: confirm the ROIs again, as the UserMask steps do.
				s.panel("roi").(*AlignFlowPanel).confirm()
				gotoPage(a, "defect")
				dp().paramsOpen = false
				dp().SelectSubTab(0)
				dp().focus.polarity = focusAll
				dp().RunInspect()
			},
			shot: "native-defect-focus-metal-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				if p := s.a.ActivePanel(); p == nil || p.ID() != "defect" {
					return nil, fmt.Errorf("defect-focus-metal: the defect page is not shown")
				}
				p := dp()
				if !p.userMaskDrawable() {
					return nil, fmt.Errorf("defect-focus-metal: regions cannot be drawn yet (align state %s)", p.AlignState().key())
				}
				want := p.focusTintKey()
				if !want.on || want.target != golden.TargetMetal {
					return nil, fmt.Errorf("defect-focus-metal: focus tint key %+v, want on and metal", want)
				}
				if p.tintShown != want {
					return nil, fmt.Errorf("defect-focus-metal: shown tint %+v, want %+v", p.tintShown, want)
				}
				items, active := p.SubTabs()
				if len(items) != 2 || active != 0 {
					return nil, fmt.Errorf("defect-focus-metal: %d sub-tabs, active %d, want 2 and 0", len(items), active)
				}
				if p.SubTabNote() == "" {
					return nil, fmt.Errorf("defect-focus-metal: empty sub-tab note")
				}
				rows := p.focusRows()
				all, _, _ := focusCounts(p.Report(), inspect.Metal)
				if metal := materialRows(p.Report(), rows, inspect.Metal); metal != len(rows) {
					return nil, fmt.Errorf("defect-focus-metal: %d of %d rows are Metal", metal, len(rows))
				}
				if len(rows) != all {
					return nil, fmt.Errorf("defect-focus-metal: %d rows, want %d Metal defects", len(rows), all)
				}
				return map[string]any{"rows": len(rows), "metal": all, "tint": want.on}, nil
			},
		},
		{
			name: "defect-focus-sr",
			enter: func(a *App) {
				dp().SelectSubTab(1)
			},
			shot: "native-defect-focus-sr-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				p := dp()
				rev := 0
				if b := board(); b != nil {
					rev = b.Rev()
				}
				want := tintKey{on: true, target: golden.TargetSR, rev: rev}
				if p.tintShown != want {
					return nil, fmt.Errorf("defect-focus-sr: shown tint %+v, want %+v", p.tintShown, want)
				}
				if _, active := p.SubTabs(); active != 1 {
					return nil, fmt.Errorf("defect-focus-sr: active sub-tab %d, want 1", active)
				}
				rows := p.focusRows()
				all, _, _ := focusCounts(p.Report(), inspect.SR)
				if sr := materialRows(p.Report(), rows, inspect.SR); sr != len(rows) {
					return nil, fmt.Errorf("defect-focus-sr: %d of %d rows are SR", sr, len(rows))
				}
				if len(rows) != all {
					return nil, fmt.Errorf("defect-focus-sr: %d rows, want %d SR defects", len(rows), all)
				}
				return map[string]any{"rows": len(rows), "sr": all}, nil
			},
		},
		{
			name: "defect-focus-bright",
			enter: func(a *App) {
				dp().focus.polarity = focusBright
			},
			shot: "native-defect-focus-bright-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				p := dp()
				rows := p.focusRows()
				all, bright, _ := focusCounts(p.Report(), inspect.SR)
				if sr := materialRows(p.Report(), rows, inspect.SR); sr != len(rows) {
					return nil, fmt.Errorf("defect-focus-bright: %d of %d rows are SR", sr, len(rows))
				}
				if b := polarityRows(p.Report(), rows, inspect.Bright); b != len(rows) {
					return nil, fmt.Errorf("defect-focus-bright: %d of %d rows are bright", b, len(rows))
				}
				if len(rows) != bright {
					return nil, fmt.Errorf("defect-focus-bright: %d rows, want %d bright SR defects (of %d)", len(rows), bright, all)
				}
				return map[string]any{"rows": len(rows), "bright": bright, "all": all}, nil
			},
		},
		{
			name: "defect-focus-dark",
			enter: func(a *App) {
				dp().focus.polarity = focusDark
			},
			shot: "native-defect-focus-dark-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				p := dp()
				rows := p.focusRows()
				all, bright, dark := focusCounts(p.Report(), inspect.SR)
				if sr := materialRows(p.Report(), rows, inspect.SR); sr != len(rows) {
					return nil, fmt.Errorf("defect-focus-dark: %d of %d rows are SR", sr, len(rows))
				}
				if d := polarityRows(p.Report(), rows, inspect.Dark); d != len(rows) {
					return nil, fmt.Errorf("defect-focus-dark: %d of %d rows are dark", d, len(rows))
				}
				if len(rows) != dark {
					return nil, fmt.Errorf("defect-focus-dark: %d rows, want %d dark SR defects", len(rows), dark)
				}
				if bright+dark != all {
					return nil, fmt.Errorf("defect-focus-dark: bright %d + dark %d != all %d", bright, dark, all)
				}
				return map[string]any{"rows": len(rows), "dark": dark, "bright": bright, "all": all}, nil
			},
		},
		{
			name: "defect-focus-usermask-target",
			enter: func(a *App) {
				p := dp()
				p.focus.polarity = focusAll
				p.SelectSubTab(1)
				// Select the tool exactly as a click on the Metal / rectangle buttons would,
				// without touching a target field: the target is the selected sub-tab.
				p.um.active = true
				p.um.frame.Kind = FrameRect
				p.um.frame.Reset()
			},
			input: canvasDrag,
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				if b == nil || !b.Ready() {
					return nil, fmt.Errorf("defect-focus-usermask-target: the golden board is not ready")
				}
				um := b.UserMask()
				if len(um.Regions) != 1 {
					return nil, fmt.Errorf("defect-focus-usermask-target: %d regions, want 1", len(um.Regions))
				}
				if r := um.Regions[0]; r.Target != golden.TargetSR {
					return nil, fmt.Errorf("defect-focus-usermask-target: region target %v, want sr", r.Target)
				}
				d := b.Derived()
				if d == nil {
					return nil, fmt.Errorf("defect-focus-usermask-target: no derived masks")
				}
				if d.userMetalPx != 0 {
					return nil, fmt.Errorf("defect-focus-usermask-target: %d Metal pixels removed by an SR region, want 0", d.userMetalPx)
				}
				return map[string]any{"regions": len(um.Regions), "userSRPx": d.userSRPx}, nil
			},
		},
		{
			name: "defect-focus-reset",
			enter: func(a *App) {
				a.golden.ClearUserMask(true, golden.TargetMetal)
				p := dp()
				p.userMaskStop()
				p.SelectSubTab(0)
				p.focus.polarity = focusAll
				p.RunInspect()
			},
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				if b == nil {
					return nil, fmt.Errorf("defect-focus-reset: no golden board")
				}
				if !b.UserMask().Empty() {
					return nil, fmt.Errorf("defect-focus-reset: %d regions left, want 0", len(b.UserMask().Regions))
				}
				p := dp()
				if p.focus.target != golden.TargetMetal || p.focus.polarity != focusAll {
					return nil, fmt.Errorf("defect-focus-reset: focus %+v, want metal / all", p.focus)
				}
				want := p.focusTintKey()
				if !want.on || want.target != golden.TargetMetal {
					return nil, fmt.Errorf("defect-focus-reset: focus tint key %+v, want on and metal", want)
				}
				if p.tintShown != want {
					return nil, fmt.Errorf("defect-focus-reset: shown tint %+v, want %+v", p.tintShown, want)
				}
				// Leave the alignment disarmed again, like the UserMask steps do (the role steps expect it).
				s.a.align.SetRois(s.a.align.Rois)
				dp().RequestCapture()
				return map[string]any{"tint": want.on, "target": int(want.target)}, nil
			},
		},
	}
}
