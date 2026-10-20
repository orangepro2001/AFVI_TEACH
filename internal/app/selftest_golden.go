package app

import (
	"fmt"
	"path/filepath"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
)

// goldenSteps exercises the golden board page of the teaching program: the automatic board, the
// binary range, morphology with undo, padding and SR inset, drawing with the viewer (rectangle,
// ellipse, polygon, minimum grey value), the area threshold, saving and loading and the inspection
// through the mask.
func (s *Selftest) goldenSteps() []selftestStep {
	var (
		before   *golden.Mask
		saved    *golden.Mask
		srBefore int
		countAdd int
		dirSaved string
		rangeLo  int
	)
	board := func() *goldenBoard { return s.a.golden }
	panel := func() *GoldenPanel { return s.panel("golden").(*GoldenPanel) }
	canvasDrag := func(s *Selftest) []viewerInput {
		c := s.canvasCenter()
		return dragInput(btnLeft, imgui.NewVec2(c.X-40, c.Y-30), imgui.NewVec2(80, 60), 4)
	}
	return []selftestStep{
		{
			name: "golden-attach",
			enter: func(a *App) {
				a.theme = ThemeDark
				SetLang(LangZh)
				if a.golden == nil {
					a.golden = s.golden
				}
				a.golden.modelLoaded(a.Scene)
				// The older steps left parameters made without a board (edge margin 6): start from the defaults.
				if dp := defectPanelOf(a); dp != nil {
					for i := range dp.inspectParams {
						dp.inspectParams[i] = dp.freshParams()
					}
					dp.learned = [3]bool{}
				}
				gotoPage(a, "golden")
			},
			shot: "native-golden-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				if b == nil || !b.Ready() {
					return nil, fmt.Errorf("golden-attach: the golden board is not ready")
				}
				if p := s.a.ActivePanel(); p == nil || p.ID() != "golden" {
					return nil, fmt.Errorf("golden-attach: the golden page is not shown")
				}
				if !s.a.Viewer.HasImage() {
					return nil, fmt.Errorf("golden-attach: the viewer shows no picture")
				}
				if w, h := s.a.Viewer.ImageSize(); w != s.a.Scene.W || h != s.a.Scene.H {
					return nil, fmt.Errorf("golden-attach: picture %dx%d, want %dx%d", w, h, s.a.Scene.W, s.a.Scene.H)
				}
				d := b.Derived()
				if d == nil || d.metalPx <= 0 || d.srPx <= 0 {
					return nil, fmt.Errorf("golden-attach: empty masks (%+v)", d)
				}
				if d.metalPx+d.bandPx != b.Metal().Count() {
					return nil, fmt.Errorf("golden-attach: metal %d + band %d != drawn %d", d.metalPx, d.bandPx, b.Metal().Count())
				}
				if b.Padding() != golden.DefaultPadding || b.SRInset() != golden.DefaultSRInset {
					return nil, fmt.Errorf("golden-attach: padding %d inset %d, want the defaults", b.Padding(), b.SRInset())
				}
				key, _, isErr := b.Message()
				if isErr || (key != "golden.msg.auto" && key != "golden.msg.fallback") {
					return nil, fmt.Errorf("golden-attach: message %q (error %v)", key, isErr)
				}
				lo, hi, out := b.Range()
				rangeLo = lo
				return map[string]any{
					"metalPx": d.metalPx, "bandPx": d.bandPx, "srPx": d.srPx,
					"range": fmt.Sprintf("%d..%d out=%v", lo, hi, out), "message": key,
					"registered": b.Geometry().registered,
				}, nil
			},
		},
		{
			name: "golden-range",
			enter: func(a *App) {
				lo, hi, out := a.golden.Range()
				lo = clampInt(lo+10, 0, 255)
				if lo > hi {
					hi = lo
				}
				a.golden.SetRange(lo, hi, out)
			},
			shot: "native-golden-range-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				lo, hi, out := b.Range()
				if lo != clampInt(rangeLo+10, 0, 255) {
					return nil, fmt.Errorf("golden-range: low %d, want %d", lo, clampInt(rangeLo+10, 0, 255))
				}
				src := b.Source()
				want := golden.Threshold(src.Data, src.Width, src.Height, uint8(lo), uint8(hi), out)
				if !b.Metal().Equal(want) {
					return nil, fmt.Errorf("golden-range: the mask is not the threshold of the picture")
				}
				return map[string]any{"count": b.Metal().Count()}, nil
			},
		},
		{
			name: "golden-morph",
			enter: func(a *App) {
				before = a.golden.Metal()
				a.golden.Morph(morphErode, 2)
			},
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				if !b.CanUndo() {
					return nil, fmt.Errorf("golden-morph: nothing to undo after Erode")
				}
				if before.Count() > 0 && b.Metal().Count() >= before.Count() {
					return nil, fmt.Errorf("golden-morph: Erode kept %d of %d pixels", b.Metal().Count(), before.Count())
				}
				return map[string]any{"before": before.Count(), "after": b.Metal().Count()}, nil
			},
		},
		{
			name: "golden-undo",
			enter: func(a *App) {
				a.golden.Undo()
			},
			check: func(s *Selftest) (map[string]any, error) {
				if !board().Metal().Equal(before) {
					return nil, fmt.Errorf("golden-undo: Undo did not bring the earlier mask back")
				}
				return map[string]any{"count": board().Metal().Count()}, nil
			},
		},
		{
			name: "golden-padding",
			enter: func(a *App) {
				a.golden.SetPadding(6)
			},
			shot: "native-golden-padding-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				d := b.Derived()
				if !d.metalInspect.Equal(golden.InspectMetal(b.Metal(), 6)) {
					return nil, fmt.Errorf("golden-padding: the inspected mask is not the drawn mask eroded by 6")
				}
				if b.Metal().Count() > 0 && d.bandPx <= 0 {
					return nil, fmt.Errorf("golden-padding: no padding band")
				}
				return map[string]any{"metalPx": d.metalPx, "bandPx": d.bandPx}, nil
			},
		},
		{
			name: "golden-padding-zero",
			enter: func(a *App) {
				a.golden.SetPadding(0)
			},
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				d := b.Derived()
				if d.bandPx != 0 || d.metalPx != b.Metal().Count() {
					return nil, fmt.Errorf("golden-padding-zero: band %d, inspected %d of %d", d.bandPx, d.metalPx, b.Metal().Count())
				}
				return map[string]any{"metalPx": d.metalPx}, nil
			},
		},
		{
			name: "golden-srinset",
			enter: func(a *App) {
				a.golden.SetPadding(golden.DefaultPadding)
				srBefore = a.golden.Derived().srPx
				a.golden.SetSRInset(12)
			},
			check: func(s *Selftest) (map[string]any, error) {
				d := board().Derived()
				if d.srPx >= srBefore {
					return nil, fmt.Errorf("golden-srinset: SR %d px with inset 12, %d px with the default", d.srPx, srBefore)
				}
				return map[string]any{"srPx": d.srPx, "before": srBefore}, nil
			},
		},
		{
			name: "golden-add-rect",
			enter: func(a *App) {
				a.golden.SetSRInset(golden.DefaultSRInset)
				a.golden.EraseShape(golden.Shape{Kind: golden.ShapeRect, P1: golden.Pt{X: float64(a.Scene.W), Y: float64(a.Scene.H)}})
				p := panel()
				p.minGV = 0
				p.setTool(goldenToolAdd)
				p.setShape(FrameRect)
			},
			input: canvasDrag,
			shot:  "native-golden-add-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				n := b.Metal().Count()
				if n <= 0 || n >= s.a.Scene.W*s.a.Scene.H {
					return nil, fmt.Errorf("golden-add-rect: %d pixels after dragging a rectangle", n)
				}
				countAdd = n
				return map[string]any{"count": n}, nil
			},
		},
		{
			name: "golden-erase-ellipse",
			enter: func(a *App) {
				p := panel()
				p.setTool(goldenToolErase)
				p.setShape(FrameEllipse)
			},
			input: func(s *Selftest) []viewerInput {
				c := s.canvasCenter()
				return dragInput(btnLeft, imgui.NewVec2(c.X-30, c.Y-20), imgui.NewVec2(60, 40), 4)
			},
			check: func(s *Selftest) (map[string]any, error) {
				n := board().Metal().Count()
				if n >= countAdd {
					return nil, fmt.Errorf("golden-erase-ellipse: %d pixels, was %d before erasing", n, countAdd)
				}
				return map[string]any{"count": n, "before": countAdd}, nil
			},
		},
		{
			name: "golden-add-polygon",
			enter: func(a *App) {
				p := panel()
				p.minGV = 0
				p.setTool(goldenToolAdd)
				p.setShape(FramePen)
				countAdd = a.golden.Metal().Count()
			},
			input: func(s *Selftest) []viewerInput {
				c := s.canvasCenter()
				verts := []imgui.Vec2{
					imgui.NewVec2(c.X+20, c.Y+30),
					imgui.NewVec2(c.X+120, c.Y+30),
					imgui.NewVec2(c.X+70, c.Y+110),
				}
				var seq []viewerInput
				for _, v := range verts {
					press := viewerInput{mouse: v, hovered: true}
					press.clicked[btnLeft], press.down[btnLeft] = true, true
					release := viewerInput{mouse: v, hovered: true}
					release.released[btnLeft] = true
					seq = append(seq, press, release)
				}
				last := verts[len(verts)-1]
				rpress := viewerInput{mouse: last, hovered: true}
				rpress.clicked[btnRight], rpress.down[btnRight] = true, true
				rrelease := viewerInput{mouse: last, hovered: true}
				rrelease.released[btnRight] = true
				return append(seq, rpress, rrelease, viewerInput{mouse: last, hovered: true})
			},
			check: func(s *Selftest) (map[string]any, error) {
				n := board().Metal().Count()
				if n <= countAdd {
					return nil, fmt.Errorf("golden-add-polygon: %d pixels, was %d before the polygon", n, countAdd)
				}
				return map[string]any{"count": n, "before": countAdd}, nil
			},
		},
		{
			name: "golden-add-mingv",
			enter: func(a *App) {
				a.golden.EraseShape(golden.Shape{Kind: golden.ShapeRect, P1: golden.Pt{X: float64(a.Scene.W), Y: float64(a.Scene.H)}})
				p := panel()
				p.minGV = 100
				p.setTool(goldenToolAdd)
				p.setShape(FrameRect)
			},
			input: canvasDrag,
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				m, src := b.Metal(), b.Source()
				dark := 0
				for i, v := range m.Pix {
					if v != 0 && src.Data[i] < 100 {
						dark++
					}
				}
				if dark != 0 {
					return nil, fmt.Errorf("golden-add-mingv: %d added pixels are darker than the minimum grey value", dark)
				}
				panel().minGV = 0
				return map[string]any{"count": m.Count()}, nil
			},
		},
		{
			name: "golden-area",
			enter: func(a *App) {
				p := panel()
				p.setTool(goldenToolPoint)
				a.golden.Auto()
				a.golden.AreaFilter(50, true)
			},
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				if !b.Metal().Valid() || !b.Ready() {
					return nil, fmt.Errorf("golden-area: the board is not ready after the area threshold")
				}
				return map[string]any{"count": b.Metal().Count()}, nil
			},
		},
		{
			name: "golden-save",
			enter: func(a *App) {
				dirSaved = filepath.Join(s.out, "golden-board")
				a.golden.dirOver = dirSaved
				saved = a.golden.Metal()
				_ = a.golden.Save()
			},
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				if !b.Confirmed() {
					return nil, fmt.Errorf("golden-save: the board is not confirmed after Save")
				}
				got, err := golden.Load(dirSaved)
				if err != nil {
					return nil, fmt.Errorf("golden-save: %w", err)
				}
				if !got.Metal.Equal(saved) {
					return nil, fmt.Errorf("golden-save: the saved mask differs from the board")
				}
				if got.Settings.Padding != b.Padding() || got.Settings.SRInset != b.SRInset() {
					return nil, fmt.Errorf("golden-save: settings %+v", got.Settings)
				}
				return map[string]any{"dir": dirSaved, "count": got.Metal.Count()}, nil
			},
		},
		{
			name: "golden-load",
			enter: func(a *App) {
				a.golden.EraseShape(golden.Shape{Kind: golden.ShapeRect, P1: golden.Pt{X: float64(a.Scene.W), Y: float64(a.Scene.H)}})
				_ = a.golden.Load()
			},
			shot: "native-golden-saved-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				if !b.Metal().Equal(saved) {
					return nil, fmt.Errorf("golden-load: the loaded mask differs from the saved one")
				}
				if !b.Confirmed() {
					return nil, fmt.Errorf("golden-load: not confirmed after Load")
				}
				b.dirOver = ""
				return map[string]any{"count": b.Metal().Count()}, nil
			},
		},
		{
			name: "golden-i18n-en",
			enter: func(a *App) {
				a.theme = ThemeLight
				SetLang(LangEn)
			},
			shot: "native-golden-light-en.png",
			check: func(s *Selftest) (map[string]any, error) {
				if got := panel().Title(); got != "Golden Board" {
					return nil, fmt.Errorf("golden-i18n-en: title %q", got)
				}
				return nil, nil
			},
		},
		{
			name: "golden-i18n-ko",
			enter: func(a *App) {
				a.theme = ThemeDark
				SetLang(LangKo)
			},
			shot: "native-golden-dark-ko.png",
			check: func(s *Selftest) (map[string]any, error) {
				if t := panel().Title(); t == "tab.golden" || t == "Golden Board" || t == "金板" {
					return nil, fmt.Errorf("golden-i18n-ko: untranslated title %q", t)
				}
				return nil, nil
			},
		},
		{
			name: "golden-inspect",
			enter: func(a *App) {
				SetLang(LangZh)
				a.golden.Auto()
				fp := s.panel("roi").(*AlignFlowPanel)
				fp.confirm()
				gotoPage(a, "defect")
				defectPanelOf(a).RunInspect()
			},
			shot: "native-golden-inspect-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				b := board()
				dp := defectPanelOf(s.a)
				if !b.Ready() {
					return nil, fmt.Errorf("golden-inspect: the board is not ready")
				}
				if dp.InspectBlocked() {
					return nil, fmt.Errorf("golden-inspect: detection is blocked")
				}
				rep := dp.Report()
				if rep == nil {
					return nil, fmt.Errorf("golden-inspect: no report")
				}
				if em := dp.inspectParams[dp.channel].EdgeMargin; em != 0 {
					return nil, fmt.Errorf("golden-inspect: edge margin %d with a golden board, want 0 (the padding does that job)", em)
				}
				if dp.goldenRev != b.LearnRev() {
					return nil, fmt.Errorf("golden-inspect: the detection used board revision %d, the board is at %d", dp.goldenRev, b.LearnRev())
				}
				mat := b.Derived().material
				reg := s.a.inspectRegionsFor(s.a.Scene)
				if len(reg.Material) != len(mat) || (len(mat) > 0 && &reg.Material[0] != &mat[0]) {
					return nil, fmt.Errorf("golden-inspect: the detection does not use the golden material map")
				}
				w, h := s.a.Scene.W, s.a.Scene.H
				outside := 0
				for _, d := range rep.Defects {
					inside := false
					for y := d.BBox.Y; y < d.BBox.Y+d.BBox.H && !inside; y++ {
						for x := d.BBox.X; x < d.BBox.X+d.BBox.W; x++ {
							if x >= 0 && y >= 0 && x < w && y < h && mat[y*w+x] != 0 {
								inside = true
								break
							}
						}
					}
					if !inside {
						outside++
					}
				}
				if outside != 0 {
					return nil, fmt.Errorf("golden-inspect: %d defects lie completely outside the inspected mask", outside)
				}
				res := map[string]any{
					"defects": len(rep.Defects), "metalPx": b.Derived().metalPx, "srPx": b.Derived().srPx,
					"edgeMargin": dp.inspectParams[dp.channel].EdgeMargin,
				}
				// Leave the alignment disarmed again, like the flow steps do.
				s.a.align.SetRois(s.a.align.Rois)
				dp.RequestCapture()
				return res, nil
			},
		},
	}
}

// goldenEndToEnd checks the whole teach path on the aligned capture: no overkill on a defect-free
// unit and the detection of the default tiny defects with the golden mask as inspection area.
func (s *Selftest) goldenEndToEnd() []selftestStep {
	return []selftestStep{
		{
			name: "golden-e2e-good",
			enter: func(a *App) {
				a.golden.Auto()
				a.golden.SetPadding(golden.DefaultPadding)
				a.golden.SetSRInset(golden.DefaultSRInset)
				dp := defectPanelOf(a)
				// gotoPage lets the page seed its own test strokes, so clear them afterwards.
				gotoPage(a, "defect")
				dp.strokes = nil
				dp.channel = core.ChRed
				dp.noise = core.SensorNoise{Seed: 1}
				dp.paramsOpen = false
				dp.resetInspect()
				dp.renderRev++
				s.panel("roi").(*AlignFlowPanel).confirm()
				dp.RunInspect()
			},
			shot: "native-golden-good-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				dp := defectPanelOf(s.a)
				rep := dp.Report()
				if rep == nil || !dp.learned[core.ChRed] {
					return nil, fmt.Errorf("golden-e2e-good: report or learned flag missing")
				}
				if dp.InspectBlocked() {
					return nil, fmt.Errorf("golden-e2e-good: detection is blocked")
				}
				if em := dp.inspectParams[core.ChRed].EdgeMargin; em != 0 {
					return nil, fmt.Errorf("golden-e2e-good: edge margin %d, want 0", em)
				}
				enabled := 0
				for i := range rep.PerMaterial {
					enabled += rep.PerMaterial[i].Pixels
				}
				if len(rep.Defects) != 0 {
					detail := ""
					for i, d := range rep.Defects {
						if i == 6 {
							break
						}
						detail += fmt.Sprintf(" [mat %v pol %v area %d box %d,%d %dx%d peak %d]", d.Material, d.Polarity, d.Area, d.BBox.X, d.BBox.Y, d.BBox.W, d.BBox.H, d.PeakGV)
					}
					return nil, fmt.Errorf("golden-e2e-good: %d defects (%d flagged pixels) on a defect-free aligned unit: overkill%s", len(rep.Defects), rep.FlaggedPixels, detail)
				}
				return map[string]any{"defects": 0, "flagged": rep.FlaggedPixels, "inspectedPx": enabled}, nil
			},
		},
		{
			name: "golden-e2e-defects",
			enter: func(a *App) {
				dp := defectPanelOf(a)
				dp.addAutoDefects()
				dp.RunInspect()
			},
			shot: "native-golden-defects-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				dp := defectPanelOf(s.a)
				rep := dp.Report()
				if rep == nil {
					return nil, fmt.Errorf("golden-e2e-defects: no report")
				}
				autoStrokes, detected := 0, 0
				for i := range dp.strokes {
					e := &dp.strokes[i]
					if !e.auto || e.gv == nil {
						continue
					}
					autoStrokes++
					if dp.strokeDetected(e.gv) {
						detected++
					}
				}
				extra := len(rep.Defects) - detected
				res := map[string]any{"autoStrokes": autoStrokes, "detected": detected, "defects": len(rep.Defects), "extra": extra}
				// Leave the alignment disarmed again, like the flow steps do.
				s.a.align.SetRois(s.a.align.Rois)
				dp.RequestCapture()
				if autoStrokes != defaultAutoCount {
					return nil, fmt.Errorf("golden-e2e-defects: %d auto strokes, want %d", autoStrokes, defaultAutoCount)
				}
				if detected < 7 {
					return res, fmt.Errorf("golden-e2e-defects: %d of %d tiny defects detected, want >= 7", detected, autoStrokes)
				}
				if extra > 6 {
					return res, fmt.Errorf("golden-e2e-defects: %d extra defects, want <= 6", extra)
				}
				return res, nil
			},
		},
	}
}

// goldenOperatorStep shows the golden page to the operator.
func (s *Selftest) goldenOperatorStep() selftestStep {
	return selftestStep{
		name: "roles-golden",
		enter: func(a *App) {
			a.theme = ThemeDark
			SetLang(LangZh)
			gotoPage(a, "golden")
		},
		shot: "native-operator-golden-zh.png",
		check: func(s *Selftest) (map[string]any, error) {
			if s.a.Role().String() != "operator" {
				return nil, fmt.Errorf("roles-golden: role %s, want operator", s.a.Role())
			}
			if p := s.a.ActivePanel(); p == nil || p.ID() != "golden" {
				return nil, fmt.Errorf("roles-golden: the golden page is not shown to the operator")
			}
			b := s.a.golden
			if b == nil || !b.Ready() {
				return nil, fmt.Errorf("roles-golden: the golden board is not ready")
			}
			if !s.a.Viewer.HasImage() {
				return nil, fmt.Errorf("roles-golden: the viewer shows no picture")
			}
			return map[string]any{"pages": pageIDs(s.a), "metalPx": b.Derived().metalPx}, nil
		},
	}
}
