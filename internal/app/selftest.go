package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/inspect"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// Selftest drives the app through every tab, theme and language, waits until background
// jobs settle, saves screenshots and a metrics JSON, then closes the window.
type Selftest struct {
	a           *App
	out         string
	steps       []selftestStep
	idx         int
	frames      int
	quiet       int
	stepStart   time.Time
	frameStart  time.Time
	frameMs     []float64
	results     map[string]any
	shots       []string
	errors      []string
	pendingShot string
	capturing   bool
	finished    bool
	deadline    time.Time
	seq         []viewerInput
	seqIdx      int
}

type selftestStep struct {
	name  string
	enter func(a *App)
	shot  string
	// input is fed to the viewer one frame at a time after enter; check runs once jobs settle.
	input func(s *Selftest) []viewerInput
	check func(s *Selftest) (map[string]any, error)
}

func (s *Selftest) canvasCenter() imgui.Vec2 {
	o, sz := s.a.Viewer.Canvas()
	return imgui.NewVec2(o.X+sz.X/2, o.Y+sz.Y/2)
}

// dragInput builds a press-move-release sequence for button btn starting at from.
func dragInput(btn int, from, d imgui.Vec2, steps int) []viewerInput {
	seq := []viewerInput{{mouse: from, hovered: true}}
	press := viewerInput{mouse: from, hovered: true}
	press.clicked[btn], press.down[btn] = true, true
	seq = append(seq, press)
	for k := 1; k <= steps; k++ {
		f := float32(k) / float32(steps)
		in := viewerInput{mouse: imgui.NewVec2(from.X+d.X*f, from.Y+d.Y*f), delta: imgui.NewVec2(d.X/float32(steps), d.Y/float32(steps)), hovered: true}
		in.down[btn] = true
		seq = append(seq, in)
	}
	release := viewerInput{mouse: imgui.NewVec2(from.X+d.X, from.Y+d.Y), hovered: true}
	release.released[btn] = true
	return append(seq, release, viewerInput{mouse: imgui.NewVec2(from.X+d.X, from.Y+d.Y), hovered: true})
}

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func (s *Selftest) panel(id string) Panel {
	for _, p := range s.a.panels {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// NewSelftest prepares the scripted run.
func NewSelftest(a *App, out string) *Selftest {
	s := &Selftest{a: a, out: out, results: map[string]any{}, deadline: time.Now().Add(300 * time.Second)}
	tab := func(id string) func(a *App) {
		return func(a *App) {
			for i, p := range a.panels {
				if p.ID() == id {
					a.switchTab(i)
					p.Selftest()
				}
			}
		}
	}
	var zoom0, panX0, panY0 float32
	var shiftIn0, shiftOut0 float32
	var rois0, strokes0 int
	s.steps = []selftestStep{
		{name: "load"},
		{name: "light-dark-zh", enter: func(a *App) { a.theme = ThemeDark; SetLang(LangZh); tab("light")(a) }, shot: "native-light-dark-zh.png"},
		{name: "teach-start", enter: func(a *App) { a.theme = ThemeDark; SetLang(LangZh); tab("align")(a) }},
		{name: "interact-viewer", input: func(s *Selftest) []viewerInput {
			v := s.a.Viewer
			zoom0 = v.zoom
			c := s.canvasCenter()
			seq := []viewerInput{{mouse: c, hovered: true}, {mouse: c, wheel: 1, hovered: true}, {mouse: c, hovered: true}}
			return seq
		}, check: func(s *Selftest) (map[string]any, error) {
			v := s.a.Viewer
			ratio := float64(v.zoom / zoom0)
			panX0, panY0 = v.panX, v.panY
			if !near(ratio, 1.2, 1e-4) {
				return nil, fmt.Errorf("wheel zoom ratio %.5f, want 1.2", ratio)
			}
			return map[string]any{"wheelZoomRatio": ratio}, nil
		}},
		{name: "interact-pan", input: func(s *Selftest) []viewerInput {
			return dragInput(btnMiddle, s.canvasCenter(), imgui.NewVec2(80, -40), 4)
		}, check: func(s *Selftest) (map[string]any, error) {
			v := s.a.Viewer
			dx, dy := float64(v.panX-panX0), float64(v.panY-panY0)
			if !near(dx, 80, 0.01) || !near(dy, -40, 0.01) {
				return nil, fmt.Errorf("middle-drag pan (%.2f, %.2f), want (80, -40)", dx, dy)
			}
			return map[string]any{"panDx": dx, "panDy": dy}, nil
		}},
		{name: "interact-shift-zoom-in", input: func(s *Selftest) []viewerInput {
			v := s.a.Viewer
			shiftIn0 = v.zoom
			c := s.canvasCenter()
			press := viewerInput{mouse: c, hovered: true, shift: true}
			press.clicked[btnLeft], press.down[btnLeft] = true, true
			release := viewerInput{mouse: c, hovered: true, shift: true}
			release.released[btnLeft] = true
			return []viewerInput{{mouse: c, hovered: true}, press, release, {mouse: c, hovered: true}}
		}, check: func(s *Selftest) (map[string]any, error) {
			v := s.a.Viewer
			ratio := float64(v.zoom / shiftIn0)
			if !near(ratio, 1.5, 1e-4) {
				return nil, fmt.Errorf("shift+left zoom ratio %.5f, want 1.5", ratio)
			}
			return map[string]any{"shiftZoomInRatio": ratio}, nil
		}},
		{name: "interact-shift-zoom-out", input: func(s *Selftest) []viewerInput {
			v := s.a.Viewer
			shiftOut0 = v.zoom
			c := s.canvasCenter()
			press := viewerInput{mouse: c, hovered: true, shift: true}
			press.clicked[btnRight], press.down[btnRight] = true, true
			release := viewerInput{mouse: c, hovered: true, shift: true}
			release.released[btnRight] = true
			return []viewerInput{{mouse: c, hovered: true}, press, release, {mouse: c, hovered: true}}
		}, check: func(s *Selftest) (map[string]any, error) {
			v := s.a.Viewer
			ratio := float64(v.zoom / shiftOut0)
			if !near(ratio, 1.0/1.5, 1e-4) {
				return nil, fmt.Errorf("shift+right zoom ratio %.5f, want %.5f", ratio, 1.0/1.5)
			}
			return map[string]any{"shiftZoomOutRatio": ratio}, nil
		}},
		{name: "interact-measure", input: func(s *Selftest) []viewerInput {
			return dragInput(btnLeft, s.canvasCenter(), imgui.NewVec2(100, 80), 5)
		}, check: func(s *Selftest) (map[string]any, error) {
			lp := s.panel("light").(*LightPanel)
			z := float64(s.a.Viewer.zoom)
			if lp.measure == nil {
				return nil, fmt.Errorf("left drag did not create a measure box")
			}
			if !near(lp.measure.W, 100/z, 0.01) || !near(lp.measure.H, 80/z, 0.01) {
				return nil, fmt.Errorf("measure box %.2f×%.2f, want %.2f×%.2f", lp.measure.W, lp.measure.H, 100/z, 80/z)
			}
			if len(lp.stats) == 0 || lp.stats[0].stats.Roi.W != math.Ceil(lp.measure.X+lp.measure.W)-math.Floor(lp.measure.X) {
				return nil, fmt.Errorf("measure statistics not updated for the dragged box")
			}
			return map[string]any{"measureW": lp.measure.W, "measureH": lp.measure.H, "pixels": lp.stats[0].stats.Pixels}, nil
		}},
		{name: "align-dark-zh", enter: tab("align"), shot: "native-align-dark-zh.png"},
		{name: "interact-align-roi", enter: func(a *App) {
			ap := s.panel("align").(*AlignPanel)
			rois0 = len(ap.rois)
			ap.tool = "roi"
		}, input: func(s *Selftest) []viewerInput {
			c := s.canvasCenter()
			return dragInput(btnLeft, imgui.NewVec2(c.X-30, c.Y-30), imgui.NewVec2(60, 50), 4)
		}, check: func(s *Selftest) (map[string]any, error) {
			ap := s.panel("align").(*AlignPanel)
			if len(ap.rois) != rois0+1 || ap.tool != "none" {
				return nil, fmt.Errorf("ROI tool drag: rois %d→%d, tool %q", rois0, len(ap.rois), ap.tool)
			}
			last := ap.rois[len(ap.rois)-1]
			if ap.report == nil || len(ap.report.Matches) != len(ap.rois) {
				return nil, fmt.Errorf("alignment did not re-run with the new ROI")
			}
			// Remove the probe ROI so the screenshots show the machine ROIs only.
			ap.rois = ap.rois[:rois0]
			ap.dirty = true
			ap.requestRun(true)
			return map[string]any{"newRoi": fmt.Sprintf("%.0f,%.0f %.0f×%.0f", last.X, last.Y, last.W, last.H)}, nil
		}},
		{name: "align-overlay", enter: func(a *App) {
			tab("align")(a)
			ap := s.panel("align").(*AlignPanel)
			ap.params = defaultWarpParams()
			ap.occs = nil
			ap.dirty = true
			ap.requestRun(true)
		}, check: func(s *Selftest) (map[string]any, error) {
			ap := s.panel("align").(*AlignPanel)
			if ap.viewKind != "overlay" {
				return nil, fmt.Errorf("align-overlay: default view is %q, want overlay", ap.viewKind)
			}
			if ap.report == nil || ap.truth == nil || ap.template == nil || ap.input == nil {
				return nil, fmt.Errorf("align-overlay: no report, truth or images")
			}
			// Overlay texture: template in red, warped input in green and blue.
			pix := PackOverlay(ap.template, ap.input)
			defer PutRGBA(pix)
			n := ap.template.Width * ap.template.Height
			for _, i := range []int{0, n / 3, n / 2, n - 1} {
				if pix[4*i] != ap.template.Data[i] || pix[4*i+1] != ap.input.Data[i] || pix[4*i+2] != ap.input.Data[i] || pix[4*i+3] != 255 {
					return nil, fmt.Errorf("align-overlay: pixel %d is %v, want template %d / input %d", i, pix[4*i:4*i+4], ap.template.Data[i], ap.input.Data[i])
				}
			}
			truthLevel := core.TruthLevel(ap.truth.MaxErrorPx, ap.reportConfig)
			judge := core.JudgeAgainstTruth(ap.report.Level, truthLevel)
			if ap.report.Level != core.VerdictOK || judge != core.JudgeCorrect {
				return nil, fmt.Errorf("align-overlay: unwarped input judged %s (%s), truth %s", ap.report.Level, judge, truthLevel)
			}
			return map[string]any{"view": ap.viewKind, "verdict": string(ap.report.Level), "truthMaxPx": ap.truth.MaxErrorPx}, nil
		}},
		{name: "align-offset", enter: func(a *App) {
			ap := s.panel("align").(*AlignPanel)
			ap.tool = "none"
			setWarpGeometry(&ap.params, warpPresetByID("severe"))
			ap.dirty = true
			ap.requestRun(true)
		}, shot: "native-align-offset.png", check: func(s *Selftest) (map[string]any, error) {
			ap := s.panel("align").(*AlignPanel)
			if ap.report == nil || ap.truth == nil || ap.report.Fit == nil {
				return nil, fmt.Errorf("align-offset: no report, truth or fit")
			}
			want := warpPresetByID("severe")
			if got := ap.reportParams.Tx * float64(ap.lastScale); math.Abs(got-want.Tx) > 1e-9 {
				return nil, fmt.Errorf("align-offset: report used tx %.3f, want %.3f", got, want.Tx)
			}
			// The offset must be visible: a large share of the input differs from the template.
			differing := 0
			for i := range ap.template.Data {
				d := int(ap.template.Data[i]) - int(ap.input.Data[i])
				if d > 8 || d < -8 {
					differing++
				}
			}
			if differing*20 < len(ap.template.Data) {
				return nil, fmt.Errorf("align-offset: only %d of %d pixels differ, the offset did not reach the image", differing, len(ap.template.Data))
			}
			w, h := ap.reportW, ap.reportH
			tp := core.TruthPose(&ap.reportParams, w, h)
			fp := core.FitPose(*ap.report.Fit, w, h)
			if math.Hypot(tp.ShiftX, tp.ShiftY) < 10 {
				return nil, fmt.Errorf("align-offset: true centre shift only (%.2f, %.2f) working px", tp.ShiftX, tp.ShiftY)
			}
			if math.Abs(fp.ShiftX-tp.ShiftX) > 1.5 || math.Abs(fp.ShiftY-tp.ShiftY) > 1.5 || math.Abs(fp.ThetaDeg-tp.ThetaDeg) > 0.3 {
				return nil, fmt.Errorf("align-offset: measured pose %+v is far from the set pose %+v", fp, tp)
			}
			truthLevel := core.TruthLevel(ap.truth.MaxErrorPx, ap.reportConfig)
			judge := core.JudgeAgainstTruth(ap.report.Level, truthLevel)
			if judge != core.JudgeCorrect {
				return nil, fmt.Errorf("align-offset: verdict %s but truth %s (%s), max truth error %.3f px", ap.report.Level, truthLevel, judge, ap.truth.MaxErrorPx)
			}
			return map[string]any{
				"verdict": string(ap.report.Level), "judgement": string(judge), "truthMaxPx": ap.truth.MaxErrorPx,
				"setShift": []float64{tp.ShiftX, tp.ShiftY}, "measuredShift": []float64{fp.ShiftX, fp.ShiftY},
				"differingPixels": differing,
			}, nil
		}},
		{name: "align-lift", enter: func(a *App) {
			ap := s.panel("align").(*AlignPanel)
			ap.params = defaultWarpParams()
			setWarpGeometry(&ap.params, warpPresetByID("lift"))
			ap.dirty = true
			ap.requestRun(true)
		}, shot: "native-align-lift.png", check: func(s *Selftest) (map[string]any, error) {
			ap := s.panel("align").(*AlignPanel)
			if ap.report == nil || ap.truth == nil || math.IsNaN(ap.truth.MaxErrorPx) {
				return nil, fmt.Errorf("align-lift: no report or truth comparison")
			}
			want := warpPresetByID("lift")
			if ap.reportParams.TiltX != want.TiltX || ap.reportParams.TiltY != want.TiltY {
				return nil, fmt.Errorf("align-lift: report used tilt (%g, %g), want (%g, %g)", ap.reportParams.TiltX, ap.reportParams.TiltY, want.TiltX, want.TiltY)
			}
			differing := 0
			for i := range ap.template.Data {
				d := int(ap.template.Data[i]) - int(ap.input.Data[i])
				if d > 8 || d < -8 {
					differing++
				}
			}
			if differing*50 < len(ap.template.Data) {
				return nil, fmt.Errorf("align-lift: only %d of %d pixels differ, the lift did not reach the image", differing, len(ap.template.Data))
			}
			truthLevel := core.TruthLevel(ap.truth.MaxErrorPx, ap.reportConfig)
			judge := core.JudgeAgainstTruth(ap.report.Level, truthLevel)
			out := map[string]any{
				"verdict": string(ap.report.Level), "truth": string(truthLevel), "judgement": string(judge),
				"truthMaxPx": ap.truth.MaxErrorPx, "differingPixels": differing,
			}
			return out, nil
		}},
		// Leave the tab as it was found for the screenshots that follow.
		{name: "align-reset", enter: func(a *App) {
			ap := s.panel("align").(*AlignPanel)
			ap.params = defaultWarpParams()
			ap.dirty = true
			ap.requestRun(true)
		}},
		{name: "defect-dark-zh", enter: tab("defect"), shot: "native-defect-dark-zh.png"},
		{name: "interact-defect-rect", enter: func(a *App) {
			dp := s.panel("defect").(*DefectPanel)
			strokes0 = len(dp.strokes)
			dp.tool = toolBrightRect
		}, input: func(s *Selftest) []viewerInput {
			c := s.canvasCenter()
			return dragInput(btnLeft, imgui.NewVec2(c.X+40, c.Y+40), imgui.NewVec2(30, 20), 3)
		}, check: func(s *Selftest) (map[string]any, error) {
			dp := s.panel("defect").(*DefectPanel)
			if len(dp.strokes) != strokes0+1 || dp.tool != toolNone {
				return nil, fmt.Errorf("rect tool drag: strokes %d→%d", strokes0, len(dp.strokes))
			}
			st := dp.strokes[len(dp.strokes)-1].gv
			if st == nil || st.Shape != core.ShapeRect || dp.renderRev != dp.renderedRev {
				return nil, fmt.Errorf("rect stroke not rendered")
			}
			return map[string]any{"rect": fmt.Sprintf("%.0f,%.0f→%.0f,%.0f", st.X, st.Y, st.X2, st.Y2)}, nil
		}},
		{name: "interact-defect-ellipse", enter: func(a *App) {
			dp := s.panel("defect").(*DefectPanel)
			strokes0 = len(dp.strokes)
			dp.tool = toolFrame
			dp.frame.Reset()
			dp.frame.Kind = FrameEllipse
		}, input: func(s *Selftest) []viewerInput {
			c := s.canvasCenter()
			return dragInput(btnLeft, imgui.NewVec2(c.X-60, c.Y-40), imgui.NewVec2(50, 30), 3)
		}, check: func(s *Selftest) (map[string]any, error) {
			dp := s.panel("defect").(*DefectPanel)
			if len(dp.strokes) != strokes0+1 || dp.tool != toolNone {
				return nil, fmt.Errorf("ellipse tool drag: strokes %d→%d, tool %v", strokes0, len(dp.strokes), dp.tool)
			}
			st := dp.strokes[len(dp.strokes)-1].gv
			if st == nil || st.Shape != core.ShapeEllipse || dp.renderRev != dp.renderedRev {
				return nil, fmt.Errorf("ellipse stroke not rendered")
			}
			return map[string]any{"ellipse": fmt.Sprintf("%.0f,%.0f→%.0f,%.0f", st.X, st.Y, st.X2, st.Y2)}, nil
		}},
		{name: "interact-defect-polygon", enter: func(a *App) {
			dp := s.panel("defect").(*DefectPanel)
			strokes0 = len(dp.strokes)
			dp.tool = toolFrame
			dp.frame.Reset()
			dp.frame.Kind = FramePen
		}, input: func(s *Selftest) []viewerInput {
			c := s.canvasCenter()
			verts := []imgui.Vec2{
				imgui.NewVec2(c.X-60, c.Y+40),
				imgui.NewVec2(c.X-10, c.Y+40),
				imgui.NewVec2(c.X-35, c.Y+90),
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
			seq = append(seq, rpress, rrelease, viewerInput{mouse: last, hovered: true})
			return seq
		}, check: func(s *Selftest) (map[string]any, error) {
			dp := s.panel("defect").(*DefectPanel)
			if len(dp.strokes) != strokes0+1 || dp.tool != toolNone {
				return nil, fmt.Errorf("pen tool: strokes %d→%d, tool %v", strokes0, len(dp.strokes), dp.tool)
			}
			st := dp.strokes[len(dp.strokes)-1].gv
			if st == nil || st.Shape != core.ShapePolygon || len(st.Points) != 3 {
				return nil, fmt.Errorf("polygon stroke not committed with 3 points")
			}
			return map[string]any{"polygonPoints": len(st.Points)}, nil
		}},
		{name: "interact-defect-probe", enter: func(a *App) {
			s.panel("defect").(*DefectPanel).tool = toolProbe
		}, input: func(s *Selftest) []viewerInput {
			c := s.canvasCenter()
			return dragInput(btnLeft, imgui.NewVec2(c.X+30, c.Y+30), imgui.NewVec2(50, 40), 3)
		}, check: func(s *Selftest) (map[string]any, error) {
			dp := s.panel("defect").(*DefectPanel)
			if dp.lastProbe == nil || dp.probeRect == nil || dp.lastProbe.Rect.W == 0 {
				return nil, fmt.Errorf("probe drag produced no probe report")
			}
			pixels := 0
			for _, ps := range dp.lastProbe.PerMaterial {
				pixels += ps.Pixels
			}
			return map[string]any{"probePixels": pixels, "probeDefects": len(dp.lastProbe.Defects)}, nil
		}},
		{name: "defect-light-en", enter: func(a *App) { a.theme = ThemeLight; SetLang(LangEn) }, shot: "native-defect-light-en.png"},
		{name: "align-light-ko", enter: func(a *App) { SetLang(LangKo); tab("align")(a) }, shot: "native-align-light-ko.png"},
		{name: "light-light-ko", enter: tab("light"), shot: "native-light-light-ko.png"},
	}
	// Coloured-defect steps: four 50×50 bright rects (red, green, blue, white) laid out
	// horizontally; each render channel must show only the blocks whose colour has a
	// non-zero component in that channel.
	var colorRects [4]core.Rect
	colorStrokes := [][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 1, 1}}
	enterColor := func(a *App) {
		for i, p := range a.panels {
			if p.ID() == "defect" {
				a.switchTab(i)
			}
		}
		for i := 0; i < 3; i++ {
			a.Light.Channels[i].Enable = true
			a.Light.Channels[i].Value = 512
		}
		a.NotifyLightChanged()
		dp := s.panel("defect").(*DefectPanel)
		dp.strokes = nil
		dp.channel = core.ChRed
		scene := a.Scene
		for i, cs := range colorStrokes {
			x := float64(scene.W/2 + (i-2)*120)
			y := float64(scene.H / 2)
			colorRects[i] = core.Rect{X: x, Y: y, W: 50, H: 50}
			st := &core.DefectStroke{
				Kind:      core.Bright,
				Shape:     core.ShapeRect,
				X:         x,
				Y:         y,
				X2:        x + 50,
				Y2:        y + 50,
				DeltaGv:   100,
				Softness:  0,
				Amplitude: core.AmpAbsolute,
				HasColor:  true,
				Color:     cs,
			}
			dp.strokes = append(dp.strokes, strokeEntry{gv: st, enabled: true})
		}
		dp.renderRev++
		dp.requestRun()
	}
	colorCheck := func(s *Selftest, want [4]bool) (map[string]any, error) {
		dp := s.panel("defect").(*DefectPanel)
		if dp.golden == nil || dp.test == nil {
			return nil, fmt.Errorf("colour step: golden/test not rendered")
		}
		var sums [4]float64
		for i, r := range colorRects {
			x0 := int(r.X)
			y0 := int(r.Y)
			x1 := int(r.X + r.W)
			y1 := int(r.Y + r.H)
			sum := 0.0
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					idx := y*dp.golden.Width + x
					sum += float64(dp.test.Data[idx]) - float64(dp.golden.Data[idx])
				}
			}
			sums[i] = sum
		}
		for i, w := range want {
			if w && sums[i] <= 0 {
				return nil, fmt.Errorf("colour block %d sum %.1f, want > 0", i, sums[i])
			}
			if !w && sums[i] != 0 {
				return nil, fmt.Errorf("colour block %d sum %.1f, want 0", i, sums[i])
			}
		}
		return map[string]any{
			"redSum": sums[0], "greenSum": sums[1], "blueSum": sums[2], "whiteSum": sums[3],
		}, nil
	}
	s.steps = append(s.steps,
		selftestStep{
			name: "defect-color-red", enter: enterColor, shot: "native-defect-color-red.png",
			check: func(s *Selftest) (map[string]any, error) {
				return colorCheck(s, [4]bool{true, false, false, true})
			},
		},
		selftestStep{
			name: "defect-color-green", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.channel = core.ChGreen
				dp.renderRev++
				dp.requestRun()
			}, shot: "native-defect-color-green.png",
			check: func(s *Selftest) (map[string]any, error) {
				return colorCheck(s, [4]bool{false, true, false, true})
			},
		},
		selftestStep{
			name: "defect-color-blue", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.channel = core.ChBlue
				dp.renderRev++
				dp.requestRun()
			}, shot: "native-defect-color-blue.png",
			check: func(s *Selftest) (map[string]any, error) {
				return colorCheck(s, [4]bool{false, false, true, true})
			},
		},
		selftestStep{
			name: "defect-params-dialog", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.paramsOpen = true
			}, shot: "native-defect-params-dialog.png",
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if !dp.paramsOpen {
					return nil, fmt.Errorf("params dialog closed unexpectedly")
				}
				return map[string]any{"paramsDialog": true}, nil
			},
		},
	)

	var goodFlagged, overkillDefects, overkillFlagged int
	var missRect core.Rect
	var snapRedTest *core.GrayImage
	var snapDir string
	enterDefect := func(a *App) {
		for i, p := range a.panels {
			if p.ID() == "defect" {
				a.switchTab(i)
			}
		}
	}
	s.steps = append(s.steps,
		selftestStep{
			name: "inspect-good", enter: func(a *App) {
				enterDefect(a)
				for i := 0; i < 3; i++ {
					a.Light.Channels[i].Enable = true
					a.Light.Channels[i].Value = 512
				}
				a.NotifyLightChanged()
				dp := s.panel("defect").(*DefectPanel)
				dp.strokes = nil
				dp.channel = core.ChRed
				dp.noise = core.SensorNoise{Seed: 1}
				dp.paramsOpen = false
				dp.resetInspect()
				dp.renderRev++
				dp.requestRun()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.lastReport == nil || !dp.learned[core.ChRed] {
					return nil, fmt.Errorf("inspect-good: report or learned flag missing")
				}
				params := dp.inspectParams[core.ChRed]
				enabledPx := 0
				for i := 0; i < inspect.NumMaterials; i++ {
					if params.Materials[i].Enabled {
						enabledPx += dp.lastReport.PerMaterial[i].Pixels
					}
				}
				if dp.lastReport.FlaggedPixels*20 > enabledPx {
					return nil, fmt.Errorf("inspect-good: %d flagged of %d enabled pixels (>5%%)", dp.lastReport.FlaggedPixels, enabledPx)
				}
				goodFlagged = dp.lastReport.FlaggedPixels
				metal := params.Materials[inspect.Metal]
				sr := params.Materials[inspect.SR]
				return map[string]any{
					"goodFlagged": goodFlagged,
					"goodDefects": len(dp.lastReport.Defects),
					"metalRange":  fmt.Sprintf("[%.0f,%.0f]", metal.Low, metal.High),
					"srRange":     fmt.Sprintf("[%.0f,%.0f]", sr.Low, sr.High),
				}, nil
			},
		},
		selftestStep{
			name: "inspect-overkill", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				for _, m := range []inspect.Material{inspect.Metal, inspect.SR} {
					ms := dp.lastReport.PerMaterial[m]
					mp := &dp.inspectParams[core.ChRed].Materials[m]
					mp.Low = math.Max(0, math.Floor(ms.Mean-0.5*ms.Std))
					mp.High = math.Min(255, math.Ceil(ms.Mean+0.5*ms.Std))
					mp.Size = 1
					mp.MinArea = 1
				}
				dp.paramsOpen = true
				dp.requestRun()
			}, shot: "native-defect-inspect-overkill.png",
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.lastReport == nil {
					return nil, fmt.Errorf("inspect-overkill: no report")
				}
				if len(dp.lastReport.Defects) < 2 || dp.lastReport.FlaggedPixels <= 3*goodFlagged {
					return nil, fmt.Errorf("inspect-overkill: defects %d, flagged %d (good %d)", len(dp.lastReport.Defects), dp.lastReport.FlaggedPixels, goodFlagged)
				}
				overkillDefects = len(dp.lastReport.Defects)
				overkillFlagged = dp.lastReport.FlaggedPixels
				return map[string]any{
					"overkillDefects": overkillDefects,
					"overkillFlagged": overkillFlagged,
					"goodFlagged":     goodFlagged,
				}, nil
			},
		},
		selftestStep{
			name: "inspect-size", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				for _, m := range []inspect.Material{inspect.Metal, inspect.SR} {
					mp := &dp.inspectParams[core.ChRed].Materials[m]
					mp.Size = 33
				}
				dp.paramsOpen = false
				dp.requestRun()
			}, shot: "native-defect-inspect-size.png",
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.lastReport == nil {
					return nil, fmt.Errorf("inspect-size: no report")
				}
				rep := dp.lastReport
				ok := rep.Filtered > 0 &&
					len(rep.Defects) < overkillDefects &&
					len(rep.Defects)+rep.Filtered == overkillDefects &&
					rep.FlaggedPixels == overkillFlagged
				if ok {
					for _, d := range rep.Defects {
						if d.Area < 33 {
							ok = false
							break
						}
					}
				}
				if !ok {
					return nil, fmt.Errorf("inspect-size: defects %d (was %d), filtered %d, flagged %d (was %d)", len(rep.Defects), overkillDefects, rep.Filtered, rep.FlaggedPixels, overkillFlagged)
				}
				return map[string]any{
					"smallSizeDefects": overkillDefects,
					"largeSizeDefects": len(rep.Defects),
					"filtered":         rep.Filtered,
				}, nil
			},
		},
		selftestStep{
			name: "inspect-miss", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				cx, cy := dp.findSrPoint()
				missRect = core.Rect{X: float64(cx - 3), Y: float64(cy - 3), W: 7, H: 7}
				dp.strokes = []strokeEntry{{gv: &core.DefectStroke{
					Kind:      core.Bright,
					Shape:     core.ShapeRect,
					X:         float64(cx - 3),
					Y:         float64(cy - 3),
					X2:        float64(cx + 3),
					Y2:        float64(cy + 3),
					DeltaGv:   80,
					Softness:  0,
					Amplitude: core.AmpAbsolute,
				}, enabled: true}}
				for _, m := range []inspect.Material{inspect.Metal, inspect.SR} {
					mp := &dp.inspectParams[core.ChRed].Materials[m]
					mp.Low = 0
					mp.High = 255
					mp.Size = 3
					mp.MinArea = 1
				}
				dp.renderRev++
				dp.requestRun()
			}, shot: "native-defect-inspect-miss.png",
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.golden == nil || dp.test == nil {
					return nil, fmt.Errorf("inspect-miss: golden/test not rendered")
				}
				diff := 0
				x0 := int(missRect.X)
				y0 := int(missRect.Y)
				for y := y0; y < y0+int(missRect.H); y++ {
					for x := x0; x < x0+int(missRect.W); x++ {
						idx := y*dp.test.Width + x
						diff += int(dp.test.Data[idx]) - int(dp.golden.Data[idx])
					}
				}
				if diff <= 0 {
					return nil, fmt.Errorf("inspect-miss: drawn defect diff %d, want > 0", diff)
				}
				if dp.lastReport == nil || dp.lastReport.FlaggedPixels != 0 || len(dp.lastReport.Defects) != 0 {
					return nil, fmt.Errorf("inspect-miss: over-wide range still flagged pixels")
				}
				return map[string]any{"drawnDiffSum": diff, "flagged": dp.lastReport.FlaggedPixels}, nil
			},
		},
		selftestStep{
			name: "inspect-relearn", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.requestRelearn()
				dp.requestRun()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if !dp.learned[core.ChRed] {
					return nil, fmt.Errorf("inspect-relearn: not learned")
				}
				srHigh := dp.inspectParams[core.ChRed].Materials[inspect.SR].High
				if srHigh >= 255 {
					return nil, fmt.Errorf("inspect-relearn: SR High %.0f not narrowed", srHigh)
				}
				if dp.lastReport == nil {
					return nil, fmt.Errorf("inspect-relearn: no report")
				}
				hits := 0
				for _, d := range dp.lastReport.Defects {
					if d.Material != inspect.SR {
						continue
					}
					bx := float64(d.BBox.X)
					by := float64(d.BBox.Y)
					bw := float64(d.BBox.W)
					bh := float64(d.BBox.H)
					if bx < missRect.X+missRect.W && missRect.X < bx+bw &&
						by < missRect.Y+missRect.H && missRect.Y < by+bh {
						hits++
					}
				}
				if hits == 0 {
					return nil, fmt.Errorf("inspect-relearn: no SR defect intersects the drawn rect")
				}
				return map[string]any{"srHigh": srHigh, "hitDefects": hits}, nil
			},
		},
	)

	var widthBefore, widthFocus float64
	s.steps = append(s.steps,
		selftestStep{
			name: "palette-caps", enter: func(a *App) {
				if a.profile == ProfileStudio {
					tab("light")(a)
				} else {
					tab("align")(a)
				}
			}, shot: "native-palette-caps.png",
			check: func(s *Selftest) (map[string]any, error) {
				if s.a.profile == ProfileStudio {
					lp, ok := s.panel("light").(PaletteHost)
					if !ok {
						return nil, fmt.Errorf("palette-caps: light panel does not implement PaletteHost")
					}
					if !lp.PaletteSupports(PaletteRect) || lp.PaletteSupports(PalettePick) ||
						lp.PaletteSupports(PaletteEllipse) || lp.PaletteSupports(PalettePolygon) {
						return nil, fmt.Errorf("palette-caps: light panel supports the wrong tools")
					}
					if lp.PaletteTool() != PaletteRect {
						return nil, fmt.Errorf("palette-caps: light panel active tool %v, want rect", lp.PaletteTool())
					}
					dp, ok := s.panel("defect").(PaletteHost)
					if !ok {
						return nil, fmt.Errorf("palette-caps: defect panel does not implement PaletteHost")
					}
					for _, t := range []PaletteTool{PalettePick, PaletteRect, PaletteEllipse, PalettePolygon} {
						if !dp.PaletteSupports(t) {
							return nil, fmt.Errorf("palette-caps: defect panel does not support tool %v", t)
						}
					}
					return map[string]any{"profile": string(s.a.profile), "light": "rect", "defect": "pick,rect,ellipse,polygon"}, nil
				}
				ap, ok := s.panel("align").(PaletteHost)
				if !ok {
					return nil, fmt.Errorf("palette-caps: align panel does not implement PaletteHost")
				}
				if !ap.PaletteSupports(PalettePick) || !ap.PaletteSupports(PaletteRect) ||
					ap.PaletteSupports(PaletteEllipse) || ap.PaletteSupports(PalettePolygon) {
					return nil, fmt.Errorf("palette-caps: align panel supports the wrong tools")
				}
				ap.SetPaletteTool(PalettePick)
				if ap.(*AlignPanel).tool != "select" {
					return nil, fmt.Errorf("palette-caps: pick set tool %q, want select", ap.(*AlignPanel).tool)
				}
				ap.SetPaletteTool(PaletteRect)
				if ap.(*AlignPanel).tool != "roi" {
					return nil, fmt.Errorf("palette-caps: rect set tool %q, want roi", ap.(*AlignPanel).tool)
				}
				ap.SetPaletteTool(PaletteNone)
				if ap.(*AlignPanel).tool != "none" {
					return nil, fmt.Errorf("palette-caps: none set tool %q, want none", ap.(*AlignPanel).tool)
				}
				dp, ok := s.panel("defect").(PaletteHost)
				if !ok {
					return nil, fmt.Errorf("palette-caps: defect panel does not implement PaletteHost")
				}
				for _, t := range []PaletteTool{PalettePick, PaletteRect, PaletteEllipse, PalettePolygon} {
					if dp.PaletteSupports(t) {
						return nil, fmt.Errorf("palette-caps: defect panel supports tool %v in detect mode", t)
					}
				}
				return map[string]any{"profile": string(s.a.profile), "align": "pick,rect"}, nil
			},
		},
		selftestStep{
			name: "palette-focus", enter: func(a *App) {
				_, sz := a.Viewer.Canvas()
				widthBefore = float64(sz.X)
				a.toggleFocus()
			}, shot: "native-palette-focus.png",
			check: func(s *Selftest) (map[string]any, error) {
				_, sz := s.a.Viewer.Canvas()
				widthFocus = float64(sz.X)
				if !s.a.focusView {
					return nil, fmt.Errorf("palette-focus: focusView is false")
				}
				if widthFocus <= widthBefore+600 {
					return nil, fmt.Errorf("palette-focus: canvas width %.0f, want > %.0f", widthFocus, widthBefore+600)
				}
				return map[string]any{"widthBefore": widthBefore, "widthFocus": widthFocus}, nil
			},
		},
		selftestStep{
			name: "palette-focus-off", enter: func(a *App) {
				if a.focusView {
					a.toggleFocus()
				}
			},
			check: func(s *Selftest) (map[string]any, error) {
				if s.a.focusView {
					return nil, fmt.Errorf("palette-focus-off: focusView is still true")
				}
				return map[string]any{"focusView": false}, nil
			},
		},
	)

	var selBefore, selAfter int
	s.steps = append(s.steps,
		selftestStep{
			name: "select-setup", enter: func(a *App) {
				enterDefect(a)
				dp := s.panel("defect").(*DefectPanel)
				dp.strokes = nil
				dp.sel.set = nil
				dp.noise = core.SensorNoise{Seed: 1}
				dp.paramsOpen = false
				dp.inspectParams[core.ChRed] = newInspectParams()
				dp.learned[core.ChRed] = false
				dp.relearn = false
				cx, cy := dp.findSrPoint()
				spotA := &core.DefectStroke{
					Kind: core.Bright, Shape: core.ShapeSpot,
					X: float64(cx), Y: float64(cy), Radius: 3, DeltaGv: 80,
					Softness: 0.3, Amplitude: core.AmpAbsolute, Profile: core.ProfileLinear, Seed: 11,
				}
				bx := cx + 60
				for dx := 60; dx <= 400; dx += 8 {
					if interiorSR(s.a.Scene, cx+dx, cy, defaultEdgeMargin+3+3) {
						bx = cx + dx
						break
					}
				}
				spotB := &core.DefectStroke{
					Kind: core.Bright, Shape: core.ShapeSpot,
					X: float64(bx), Y: float64(cy), Radius: 3, DeltaGv: 80,
					Softness: 0.3, Amplitude: core.AmpAbsolute, Profile: core.ProfileLinear, Seed: 12,
				}
				rectC := &core.DefectStroke{
					Kind: core.Bright, Shape: core.ShapeRect,
					X: float64(cx - 200), Y: float64(cy), X2: float64(cx - 150), Y2: float64(cy + 30),
					DeltaGv: 80, Softness: 0.3, Amplitude: core.AmpAbsolute, Profile: core.ProfileLinear, Seed: 13,
				}
				dp.strokes = []strokeEntry{
					{gv: spotA, enabled: true},
					{gv: spotB, enabled: true},
					{gv: rectC, enabled: true},
				}
				dp.renderRev++
				dp.requestRun()
				dp.SetPaletteTool(PalettePick)
				s.a.Viewer.zoom = 4
				s.a.Viewer.CenterOn(float64(cx)+30, float64(cy))
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if len(dp.strokes) != 3 {
					return nil, fmt.Errorf("select-setup: %d strokes, want 3", len(dp.strokes))
				}
				if dp.PaletteTool() != PalettePick {
					return nil, fmt.Errorf("select-setup: tool %v, want pick", dp.PaletteTool())
				}
				if !dp.PaletteSupports(PaletteEllipse) {
					return nil, fmt.Errorf("select-setup: ellipse not supported")
				}
				if dp.test == nil {
					return nil, fmt.Errorf("select-setup: no test image")
				}
				return map[string]any{"strokes": len(dp.strokes)}, nil
			},
		},
		selftestStep{
			name: "select-pick", input: func(s *Selftest) []viewerInput {
				dp := s.panel("defect").(*DefectPanel)
				pt := imageToScreen(s, dp.strokes[0].gv.X, dp.strokes[0].gv.Y)
				return dragInput(btnLeft, pt, imgui.NewVec2(0, 0), 1)
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if len(dp.sel.set) != 1 || !dp.sel.set[dp.strokes[0].gv] {
					return nil, fmt.Errorf("select-pick: selection %d, want stroke A only", len(dp.sel.set))
				}
				return map[string]any{"selected": len(dp.sel.set)}, nil
			},
		},
		selftestStep{
			name: "select-ctrl", input: func(s *Selftest) []viewerInput {
				dp := s.panel("defect").(*DefectPanel)
				pt := imageToScreen(s, dp.strokes[1].gv.X, dp.strokes[1].gv.Y)
				return withCtrl(dragInput(btnLeft, pt, imgui.NewVec2(0, 0), 1))
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if len(dp.sel.set) != 2 || !dp.sel.set[dp.strokes[0].gv] || !dp.sel.set[dp.strokes[1].gv] {
					return nil, fmt.Errorf("select-ctrl: selection %d, want A and B", len(dp.sel.set))
				}
				return map[string]any{"selected": len(dp.sel.set)}, nil
			},
		},
		selftestStep{
			name: "select-rect", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.SetPaletteTool(PaletteRect)
			},
			input: func(s *Selftest) []viewerInput {
				dp := s.panel("defect").(*DefectPanel)
				a := dp.strokes[0].gv
				z := float64(s.a.Viewer.zoom)
				from := imageToScreen(s, a.X-8, a.Y-8)
				return dragInput(btnLeft, from, imgui.NewVec2(float32(16*z), float32(16*z)), 4)
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if len(dp.sel.set) != 1 || !dp.sel.set[dp.strokes[0].gv] {
					return nil, fmt.Errorf("select-rect: selection %d, want stroke A only", len(dp.sel.set))
				}
				return map[string]any{"selected": len(dp.sel.set)}, nil
			},
		},
		selftestStep{
			name: "select-edit", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				selBefore = len(dp.selectedStrokes())
				dp.applyToSelection(func(st *core.DefectStroke) {
					st.DeltaGv = 4
					st.Radius = 1
				})
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if selBefore != 1 {
					return nil, fmt.Errorf("select-edit: selected before %d, want 1", selBefore)
				}
				selAfter = 0
				for _, st := range dp.selectedStrokes() {
					if st.DeltaGv == 4 && st.Radius == 1 {
						selAfter++
					}
				}
				if selAfter != 1 {
					return nil, fmt.Errorf("select-edit: edited %d selected strokes, want 1", selAfter)
				}
				if b := dp.strokes[1].gv; b.DeltaGv != 80 || b.Radius != 3 {
					return nil, fmt.Errorf("select-edit: stroke B changed (dGV %.0f, radius %.0f)", b.DeltaGv, b.Radius)
				}
				if len(dp.sel.set) != 1 || !dp.sel.set[dp.strokes[0].gv] {
					return nil, fmt.Errorf("select-edit: selection changed")
				}
				return map[string]any{"selectedBefore": selBefore, "editedAfter": selAfter}, nil
			},
		},
		selftestStep{
			name: "select-caps", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.clearSelection()
				dp.SetPaletteTool(PaletteNone)
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				var host PaletteHost = dp
				for _, t := range []PaletteTool{PalettePick, PaletteRect, PaletteEllipse, PalettePolygon} {
					if !host.PaletteSupports(t) {
						return nil, fmt.Errorf("select-caps: tool %v not supported", t)
					}
				}
				return map[string]any{"defect": "pick,rect,ellipse,polygon"}, nil
			},
		},
	)

	s.steps = append(s.steps,
		selftestStep{
			name: "product-variation", enter: tab("light"),
			check: func(s *Selftest) (map[string]any, error) {
				a := s.a
				if a.Scene == nil || !a.Scene.Product.Enabled {
					return nil, fmt.Errorf("product-variation: product not enabled")
				}
				rms := srRmsDelta(a)
				if rms < 0.05 {
					return nil, fmt.Errorf("product-variation: srRmsDelta %.4f, want >= 0.05", rms)
				}
				return map[string]any{"srRmsDelta": rms, "seed": a.Scene.Product.Seed}, nil
			},
		},
		selftestStep{
			name: "product-off", enter: func(a *App) {
				opts := a.Scene.Product
				opts.Enabled = false
				a.ApplyProduct(opts)
			},
			check: func(s *Selftest) (map[string]any, error) {
				a := s.a
				if a.Scene.Product.Enabled {
					return nil, fmt.Errorf("product-off: product still enabled")
				}
				rms := srRmsDelta(a)
				if rms >= 1e-9 {
					return nil, fmt.Errorf("product-off: srRmsDelta %.6f, want < 1e-9", rms)
				}
				return map[string]any{"srRmsDelta": rms}, nil
			},
		},
		selftestStep{
			name: "product-on", enter: func(a *App) {
				a.ApplyProduct(core.DefaultProductOptions())
				dp := s.panel("defect").(*DefectPanel)
				dp.productOpts = core.DefaultProductOptions()
			},
			check: func(s *Selftest) (map[string]any, error) {
				a := s.a
				if !a.Scene.Product.Enabled {
					return nil, fmt.Errorf("product-on: product not enabled")
				}
				rms := srRmsDelta(a)
				if rms < 0.05 {
					return nil, fmt.Errorf("product-on: srRmsDelta %.4f, want >= 0.05", rms)
				}
				if a.Scene.Product.Seed != 1 {
					return nil, fmt.Errorf("product-on: seed %d, want 1", a.Scene.Product.Seed)
				}
				return map[string]any{"srRmsDelta": rms, "seed": a.Scene.Product.Seed}, nil
			},
		},
		selftestStep{
			name: "defaults-detected", enter: func(a *App) {
				enterDefect(a)
				for i := 0; i < 3; i++ {
					a.Light.Channels[i].Enable = true
					a.Light.Channels[i].Value = 512
				}
				a.NotifyLightChanged()
				dp := s.panel("defect").(*DefectPanel)
				dp.channel = core.ChRed
				dp.noise = core.SensorNoise{Seed: 1}
				dp.paramsOpen = false
				dp.clearSelection()
				dp.SetPaletteTool(PaletteNone)
				dp.ModelLoaded()
				dp.Activate()
				a.Viewer.Fit() // the earlier select-* steps zoomed in; the screenshot shows the whole board
			}, shot: "native-defect-virtual.png",
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.lastReport == nil || !dp.learned[core.ChRed] {
					return nil, fmt.Errorf("defaults-detected: report or learned flag missing")
				}
				autoStrokes := 0
				detected := 0
				for i := range dp.strokes {
					e := &dp.strokes[i]
					if !e.auto || e.gv == nil {
						continue
					}
					autoStrokes++
					if e.gv.Radius < 1.5 || e.gv.Radius > 3 {
						return nil, fmt.Errorf("defaults-detected: auto radius %.2f out of [1.5,3]", e.gv.Radius)
					}
					if e.gv.DeltaGv < 45 || e.gv.DeltaGv > 90 {
						return nil, fmt.Errorf("defaults-detected: auto deltaGv %.1f out of [45,90]", e.gv.DeltaGv)
					}
					if dp.strokeDetected(e.gv) {
						detected++
					}
				}
				if autoStrokes != defaultAutoCount {
					return nil, fmt.Errorf("defaults-detected: %d auto strokes, want %d", autoStrokes, defaultAutoCount)
				}
				if detected < 9 {
					return nil, fmt.Errorf("defaults-detected: %d auto strokes detected, want >= 9", detected)
				}
				if dp.golden == nil || dp.test == nil {
					return nil, fmt.Errorf("defaults-detected: golden/test not rendered")
				}
				changed := 0
				for i := range dp.test.Data {
					if dp.test.Data[i] != dp.golden.Data[i] {
						changed++
					}
				}
				if changed > 600 {
					return nil, fmt.Errorf("defaults-detected: %d changed pixels, want <= 600", changed)
				}
				extra := len(dp.lastReport.Defects) - detected
				if extra > 6 {
					return nil, fmt.Errorf("defaults-detected: %d extra defects, want <= 6", extra)
				}
				sr := dp.inspectParams[core.ChRed].Materials[inspect.SR]
				return map[string]any{
					"autoStrokes":   autoStrokes,
					"detected":      detected,
					"changedPixels": changed,
					"extraDefects":  extra,
					"srRange":       fmt.Sprintf("[%.0f,%.0f]", sr.Low, sr.High),
				}, nil
			},
		},
		selftestStep{
			name: "edge-margin-off", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				for i := range dp.inspectParams {
					dp.inspectParams[i].EdgeMargin = 0
				}
				dp.learned = [3]bool{}
				dp.renderRev++
				dp.requestRun()
			}, shot: "native-defect-edge-off.png",
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.lastReport == nil {
					return nil, fmt.Errorf("edge-margin-off: no report")
				}
				if dp.lastReport.EdgeIgnored != 0 {
					return nil, fmt.Errorf("edge-margin-off: EdgeIgnored %d, want 0", dp.lastReport.EdgeIgnored)
				}
				if len(dp.lastReport.Defects) <= 20 {
					return nil, fmt.Errorf("edge-margin-off: %d defects, want > 20", len(dp.lastReport.Defects))
				}
				return map[string]any{"defectsWithoutMargin": len(dp.lastReport.Defects)}, nil
			},
		},
		selftestStep{
			name: "edge-margin-on", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				for i := range dp.inspectParams {
					dp.inspectParams[i].EdgeMargin = defaultEdgeMargin
				}
				dp.learned = [3]bool{}
				dp.renderRev++
				dp.requestRun()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.lastReport == nil {
					return nil, fmt.Errorf("edge-margin-on: no report")
				}
				if dp.lastReport.EdgeIgnored <= 0 {
					return nil, fmt.Errorf("edge-margin-on: EdgeIgnored %d, want > 0", dp.lastReport.EdgeIgnored)
				}
				if len(dp.lastReport.Defects) > 20 {
					return nil, fmt.Errorf("edge-margin-on: %d defects, want <= 20", len(dp.lastReport.Defects))
				}
				return map[string]any{"defectsWithMargin": len(dp.lastReport.Defects)}, nil
			},
		},
	)

	s.steps = append(s.steps,
		selftestStep{
			name: "snap-camera", enter: func(a *App) {
				enterDefect(a)
				for i := 0; i < 3; i++ {
					a.Light.Channels[i].Enable = true
					a.Light.Channels[i].Value = 512
				}
				a.NotifyLightChanged()
				dp := s.panel("defect").(*DefectPanel)
				dp.channel = core.ChRed
				dp.noise = core.SensorNoise{Seed: 1}
				dp.paramsOpen = false
				dp.clearSelection()
				dp.SetPaletteTool(PaletteNone)
				dp.ModelLoaded()
				dp.Activate()
				a.Viewer.Fit()
				dp.requestSnap()
			}, shot: "native-defect-snap.png",
			check: func(s *Selftest) (map[string]any, error) {
				if findCameraExe() == "" {
					return nil, fmt.Errorf("snap-camera: camera program not found (build bin/camera.exe next to the app)")
				}
				dp := s.panel("defect").(*DefectPanel)
				if !dp.snapSrc.on || dp.snapSrc.meta == nil {
					return nil, fmt.Errorf("snap-camera: capture not adopted")
				}
				if dp.lastReport == nil || dp.test == nil {
					return nil, fmt.Errorf("snap-camera: report or test image missing")
				}
				for _, name := range []string{snapfmt.MetaFile, "red.png", "red-good.png"} {
					if _, err := os.Stat(filepath.Join(dp.snapSrc.srcDir, name)); err != nil {
						return nil, fmt.Errorf("snap-camera: %s missing: %v", name, err)
					}
				}
				m := dp.snapSrc.meta
				if !strings.HasPrefix(m.Producer, "afvi-camera") {
					return nil, fmt.Errorf("snap-camera: producer %q, want afvi-camera*", m.Producer)
				}
				if m.Model.ID != s.a.Scene.Info.ID {
					return nil, fmt.Errorf("snap-camera: model %q, want %q", m.Model.ID, s.a.Scene.Info.ID)
				}
				if len(m.Recipe.Defects) != defaultAutoCount {
					return nil, fmt.Errorf("snap-camera: %d recipe defects, want %d", len(m.Recipe.Defects), defaultAutoCount)
				}
				if dp.snapStale() {
					return nil, fmt.Errorf("snap-camera: capture reported stale right after the snap")
				}
				if dp.snapSrc.frames[snapKey{ch: core.ChRed, kind: snapfmt.KindTest}] == nil ||
					dp.snapSrc.frames[snapKey{ch: core.ChRed, kind: snapfmt.KindGood}] == nil {
					return nil, fmt.Errorf("snap-camera: red test/good images not cached")
				}
				if !bytes.Equal(dp.test.Data, simulate(s.a, dp, core.ChRed).Data) {
					return nil, fmt.Errorf("snap-camera: snapped image differs from the in-process render")
				}
				detected := 0
				for i := range dp.strokes {
					if dp.strokes[i].gv != nil && dp.strokeDetected(dp.strokes[i].gv) {
						detected++
					}
				}
				if detected < 9 {
					return nil, fmt.Errorf("snap-camera: %d auto strokes detected, want >= 9", detected)
				}
				extra := len(dp.lastReport.Defects) - detected
				if extra > 6 {
					return nil, fmt.Errorf("snap-camera: %d extra defects, want <= 6", extra)
				}
				snapRedTest = dp.test
				snapDir = dp.snapSrc.srcDir
				return map[string]any{
					"defects":           len(dp.lastReport.Defects),
					"detected":          detected,
					"extraDefects":      extra,
					"equalToSimulation": true,
					"dir":               snapDir,
				}, nil
			},
		},
		selftestStep{
			name: "snap-stale", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				if len(dp.strokes) > 0 && dp.strokes[0].gv != nil {
					dp.strokes[0].gv.Radius += 2
				}
				dp.renderRev++
				dp.requestRun()
			}, shot: "native-defect-snap-stale.png",
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if !dp.snapStale() {
					return nil, fmt.Errorf("snap-stale: capture not reported stale after the edit")
				}
				if dp.test != snapRedTest {
					return nil, fmt.Errorf("snap-stale: test image changed before the next snap")
				}
				return map[string]any{"stale": true}, nil
			},
		},
		selftestStep{
			name: "snap-resnap", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.requestSnap()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.snapStale() {
					return nil, fmt.Errorf("snap-resnap: capture still stale after the snap")
				}
				if dp.test == snapRedTest {
					return nil, fmt.Errorf("snap-resnap: test image pointer unchanged")
				}
				if bytes.Equal(dp.test.Data, snapRedTest.Data) {
					return nil, fmt.Errorf("snap-resnap: snapped image unchanged after the edit")
				}
				if !bytes.Equal(dp.test.Data, simulate(s.a, dp, core.ChRed).Data) {
					return nil, fmt.Errorf("snap-resnap: snapped image differs from the in-process render")
				}
				snapRedTest = dp.test
				snapDir = dp.snapSrc.srcDir
				return map[string]any{"equalToSimulation": true}, nil
			},
		},
		selftestStep{
			name: "snap-channel", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.channel = core.ChGreen
				dp.renderRev++
				dp.requestRun()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				green := dp.snapSrc.frames[snapKey{ch: core.ChGreen, kind: snapfmt.KindTest}]
				if green == nil {
					return nil, fmt.Errorf("snap-channel: green test image not cached")
				}
				if _, err := os.Stat(filepath.Join(dp.snapSrc.srcDir, "green.png")); err != nil {
					return nil, fmt.Errorf("snap-channel: green.png missing: %v", err)
				}
				if dp.snapStale() {
					return nil, fmt.Errorf("snap-channel: channel switch reported as an edit")
				}
				if !bytes.Equal(dp.test.Data, simulate(s.a, dp, core.ChGreen).Data) {
					return nil, fmt.Errorf("snap-channel: snapped image differs from the in-process render")
				}
				if bytes.Equal(dp.test.Data, snapRedTest.Data) {
					return nil, fmt.Errorf("snap-channel: green image equals the red image")
				}
				return map[string]any{"equalToSimulation": true}, nil
			},
		},
		selftestStep{
			name: "snap-load", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.channel = core.ChRed
				dp.strokes = nil
				dp.snapSrc.path, dp.snapSrc.pathAuto = snapDir, false
				dp.requestSnapLoad(snapDir)
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if !dp.snapSrc.on {
					return nil, fmt.Errorf("snap-load: capture not adopted")
				}
				if dp.snapSrc.srcDir != snapDir {
					return nil, fmt.Errorf("snap-load: srcDir %q, want %q", dp.snapSrc.srcDir, snapDir)
				}
				if dp.snapSrc.workDir != snapDir {
					return nil, fmt.Errorf("snap-load: a new work folder was created")
				}
				if len(dp.strokes) != defaultAutoCount {
					return nil, fmt.Errorf("snap-load: %d strokes, want %d", len(dp.strokes), defaultAutoCount)
				}
				for i := range dp.strokes {
					if dp.strokes[i].gv == nil || dp.strokes[i].auto {
						return nil, fmt.Errorf("snap-load: stroke %d not imported from the capture", i)
					}
				}
				if dp.snapStale() {
					return nil, fmt.Errorf("snap-load: loaded capture reported stale")
				}
				if dp.lastReport == nil {
					return nil, fmt.Errorf("snap-load: no report")
				}
				if !bytes.Equal(dp.test.Data, snapRedTest.Data) {
					return nil, fmt.Errorf("snap-load: loaded image differs from the snapped image")
				}
				return map[string]any{"strokes": len(dp.strokes)}, nil
			},
		},
		selftestStep{
			name: "snap-off", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.setSnapOff()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.snapSrc.on {
					return nil, fmt.Errorf("snap-off: capture still on")
				}
				if !bytes.Equal(dp.test.Data, simulate(s.a, dp, core.ChRed).Data) {
					return nil, fmt.Errorf("snap-off: in-process render differs from the simulation")
				}
				dp.resetSnap()
				if len(dp.snapSrc.owned) != 0 {
					return nil, fmt.Errorf("snap-off: owned folders left")
				}
				if _, err := os.Stat(snapDir); err == nil {
					return nil, fmt.Errorf("snap-off: capture folder %q still exists", snapDir)
				}
				return map[string]any{"cleaned": true}, nil
			},
		},
	)

	var bgGlobalFlagged int
	s.steps = append(s.steps,
		selftestStep{
			name: "bg-global-learn", enter: func(a *App) {
				enterDefect(a)
				for i := 0; i < 3; i++ {
					a.Light.Channels[i].Enable = true
					a.Light.Channels[i].Value = 512
				}
				a.NotifyLightChanged()
				dp := s.panel("defect").(*DefectPanel)
				dp.strokes = nil
				dp.channel = core.ChRed
				dp.noise = core.SensorNoise{Seed: 1}
				dp.paramsOpen = false
				dp.resetInspect()
				for i := range dp.inspectParams {
					dp.inspectParams[i].BackgroundRadius = 0
				}
				dp.learned = [3]bool{}
				dp.renderRev++
				dp.requestRun()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.lastReport == nil || !dp.learned[core.ChRed] {
					return nil, fmt.Errorf("bg-global-learn: report or learned flag missing")
				}
				metal := dp.inspectParams[core.ChRed].Materials[inspect.Metal]
				return map[string]any{
					"metalRange": fmt.Sprintf("[%.0f,%.0f]", metal.Low, metal.High),
				}, nil
			},
		},
		selftestStep{
			name: "bg-global-blobs", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				ms := dp.lastReport.PerMaterial[inspect.Metal]
				mp := &dp.inspectParams[core.ChRed].Materials[inspect.Metal]
				mp.Low = 0
				mp.High = math.Min(255, math.Ceil(ms.Mean+22))
				mp.Size = 3
				mp.MinArea = 1
				dp.requestRun()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.lastReport == nil {
					return nil, fmt.Errorf("bg-global-blobs: no report")
				}
				n := 0
				maxArea := 0
				for _, d := range dp.lastReport.Defects {
					if d.Material != inspect.Metal {
						continue
					}
					n++
					if d.Area > maxArea {
						maxArea = d.Area
					}
				}
				if maxArea < 5000 {
					return nil, fmt.Errorf("bg-global-blobs: largest metal defect %d px, want >= 5000", maxArea)
				}
				bgGlobalFlagged = dp.lastReport.PerMaterial[inspect.Metal].Bright
				return map[string]any{
					"metalDefects": n,
					"metalMaxArea": maxArea,
					"metalFlagged": bgGlobalFlagged,
				}, nil
			},
		},
		selftestStep{
			name: "bg-local-learn", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				for i := range dp.inspectParams {
					dp.inspectParams[i].BackgroundRadius = defaultBackgroundRadius
				}
				dp.learned = [3]bool{}
				dp.requestRun()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.lastReport == nil || !dp.learned[core.ChRed] {
					return nil, fmt.Errorf("bg-local-learn: report or learned flag missing")
				}
				if dp.inspectParams[core.ChRed].BackgroundRadius != defaultBackgroundRadius {
					return nil, fmt.Errorf("bg-local-learn: radius %d, want %d", dp.inspectParams[core.ChRed].BackgroundRadius, defaultBackgroundRadius)
				}
				metal := dp.inspectParams[core.ChRed].Materials[inspect.Metal]
				if metal.Low < 30 {
					return nil, fmt.Errorf("bg-local-learn: metal Low %.0f, want >= 30", metal.Low)
				}
				params := dp.inspectParams[core.ChRed]
				enabledPx := 0
				for i := 0; i < inspect.NumMaterials; i++ {
					if params.Materials[i].Enabled {
						enabledPx += dp.lastReport.PerMaterial[i].Pixels
					}
				}
				if dp.lastReport.FlaggedPixels*20 > enabledPx {
					return nil, fmt.Errorf("bg-local-learn: %d flagged of %d enabled pixels (>5%%)", dp.lastReport.FlaggedPixels, enabledPx)
				}
				if !params.Materials[inspect.Metal].LocalBackground || params.Materials[inspect.SR].LocalBackground {
					return nil, fmt.Errorf("bg-local-learn: local background flags metal=%v sr=%v",
						params.Materials[inspect.Metal].LocalBackground, params.Materials[inspect.SR].LocalBackground)
				}
				return map[string]any{
					"metalRange":  fmt.Sprintf("[%.0f,%.0f]", metal.Low, metal.High),
					"goodFlagged": dp.lastReport.FlaggedPixels,
				}, nil
			},
		},
		selftestStep{
			name: "bg-local-dots", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				ms := dp.lastReport.PerMaterial[inspect.Metal]
				mp := &dp.inspectParams[core.ChRed].Materials[inspect.Metal]
				mp.Low = 0
				mp.High = math.Min(255, math.Ceil(ms.Mean+22))
				mp.Size = 1
				mp.MinArea = 1
				dp.requestRun()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.lastReport == nil {
					return nil, fmt.Errorf("bg-local-dots: no report")
				}
				n := 0
				maxArea := 0
				for _, d := range dp.lastReport.Defects {
					if d.Material != inspect.Metal {
						continue
					}
					n++
					if d.Area > maxArea {
						maxArea = d.Area
					}
				}
				if n < 20 {
					return nil, fmt.Errorf("bg-local-dots: %d metal defects, want >= 20", n)
				}
				if maxArea > 400 {
					return nil, fmt.Errorf("bg-local-dots: largest metal defect %d px, want <= 400", maxArea)
				}
				flagged := dp.lastReport.PerMaterial[inspect.Metal].Bright
				if flagged*10 >= bgGlobalFlagged {
					return nil, fmt.Errorf("bg-local-dots: %d flagged vs %d with the global interval", flagged, bgGlobalFlagged)
				}
				return map[string]any{
					"metalDefects":  n,
					"metalMaxArea":  maxArea,
					"metalFlagged":  flagged,
					"globalFlagged": bgGlobalFlagged,
				}, nil
			},
		},
		selftestStep{
			name: "studio-save", enter: func(a *App) {
				tab("defect")(a)
				dp := s.panel("defect").(*DefectPanel)
				dp.studio.chInit = true
				dp.studio.channels = [3]bool{true, false, false}
				dp.studio.good = true
				dp.studio.recipe = false
				dp.studio.dir = filepath.Join(s.out, "studio-capture")
				dp.requestSaveCapture()
			}, shot: "native-studio-save.png",
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.studio.last == nil || dp.studio.msgErr {
					return nil, fmt.Errorf("studio-save: %s %s", dp.studio.msgKey, dp.studio.msgText)
				}
				meta, dir, err := snapfmt.ReadMeta(dp.studio.dir)
				if err != nil {
					return nil, fmt.Errorf("studio-save: %v", err)
				}
				if !strings.HasPrefix(meta.Producer, "afvi-camera") {
					return nil, fmt.Errorf("studio-save: producer %q, want the afvi-camera prefix (camera.exe renders the images)", meta.Producer)
				}
				if meta.Width != s.a.Scene.W || meta.Height != s.a.Scene.H {
					return nil, fmt.Errorf("studio-save: size %dx%d, want %dx%d", meta.Width, meta.Height, s.a.Scene.W, s.a.Scene.H)
				}
				if len(meta.Recipe.Defects) != len(dp.gvStrokes()) {
					return nil, fmt.Errorf("studio-save: %d recipe defects, want %d", len(meta.Recipe.Defects), len(dp.gvStrokes()))
				}
				for _, kind := range []string{snapfmt.KindTest, snapfmt.KindGood} {
					f, err := snapfmt.ReadImage(dir, meta, "red", kind)
					if err != nil {
						return nil, fmt.Errorf("studio-save: %v", err)
					}
					if f.Width != s.a.Scene.W || f.Height != s.a.Scene.H {
						return nil, fmt.Errorf("studio-save: %s image %dx%d, want %dx%d", kind, f.Width, f.Height, s.a.Scene.W, s.a.Scene.H)
					}
				}
				if _, ok := meta.FindImage("green", snapfmt.KindTest); ok {
					return nil, fmt.Errorf("studio-save: green test image was written")
				}
				return map[string]any{"dir": dir, "images": len(meta.Images), "defects": len(meta.Recipe.Defects)}, nil
			},
		},
		selftestStep{
			name: "studio-warp", enter: func(a *App) {
				tab("defect")(a)
				dp := s.panel("defect").(*DefectPanel)
				dp.warp = core.IdentityWarp
				setWarpGeometry(&dp.warp, warpPresetByID("severe"))
				lift := warpPresetByID("lift")
				dp.warp.TiltX, dp.warp.TiltY = lift.TiltX, lift.TiltY
				dp.warp.BendAmplitude, dp.warp.BendWavelength = lift.BendAmplitude, lift.BendWavelength
				dp.renderRev++
				dp.scrollToWarp = true
				a.Viewer.Fit()
				dp.studio.chInit = true
				dp.studio.channels = [3]bool{true, false, false}
				dp.studio.good = true
				dp.studio.recipe = false
				dp.studio.dir = filepath.Join(s.out, "studio-warp-capture")
				dp.requestSaveCapture()
			}, shot: "native-studio-warp.png",
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.studio.last == nil || dp.studio.msgErr {
					return nil, fmt.Errorf("studio-warp: %s %s", dp.studio.msgKey, dp.studio.msgText)
				}
				meta, dir, err := snapfmt.ReadMeta(dp.studio.dir)
				if err != nil {
					return nil, fmt.Errorf("studio-warp: %v", err)
				}
				w := meta.Recipe.Warp
				lift := warpPresetByID("lift")
				if w == nil || w.Tx != 40 || w.Ty != 25 || w.TiltX != lift.TiltX || w.BendAmplitude != lift.BendAmplitude {
					return nil, fmt.Errorf("studio-warp: capture description carries warp %+v", w)
				}
				// camera.exe and the in-process simulation render the same misaligned product.
				test, err := snapfmt.ReadImage(dir, meta, "red", snapfmt.KindTest)
				if err != nil {
					return nil, fmt.Errorf("studio-warp: %v", err)
				}
				if want := simulate(s.a, dp, core.ChRed); !bytes.Equal(test.Pix, want.Data) {
					return nil, fmt.Errorf("studio-warp: camera.exe test image differs from the in-process render")
				}
				light := s.a.Light.Clone()
				_, nominal := core.RenderDefectPair(s.a.Scene.Light, &light, core.ChRed, dp.sceneStrokes(), dp.gvStrokes(), dp.noise)
				if bytes.Equal(test.Pix, nominal.Data) {
					return nil, fmt.Errorf("studio-warp: the warped capture equals the nominal one")
				}
				// The good sample is the taught reference and stays at the nominal position.
				good, err := snapfmt.ReadImage(dir, meta, "red", snapfmt.KindGood)
				if err != nil {
					return nil, fmt.Errorf("studio-warp: %v", err)
				}
				wantGood := core.RenderGoodSample(s.a.Scene.Light, &light, core.ChRed, goodSampleNoise(dp.noise))
				if !bytes.Equal(good.Pix, wantGood.Data) {
					return nil, fmt.Errorf("studio-warp: the good sample moved with the warp")
				}
				return map[string]any{"dir": dir, "warp": warpSummary(w)}, nil
			},
		},
		// Back to the nominal position for the steps that follow.
		selftestStep{
			name: "studio-warp-reset", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.warp = core.IdentityWarp
				dp.renderRev++
			},
		},
	)
	var manualWant *core.GrayImage
	var manualX, manualY int
	s.steps = append(s.steps,
		selftestStep{
			name: "manual-on", enter: func(a *App) {
				enterDefect(a)
				dp := s.panel("defect").(*DefectPanel)
				dp.liveFollow = false
				dp.clearUserImage()
				dp.snapSrc.on = false
				dp.channel = core.ChRed
				dp.strokes = nil
				dp.noise = core.SensorNoise{Seed: 1}
				dp.paramsOpen = false
				dp.resetInspect()
				dp.manual = true
				dp.renderRev++
				dp.requestRun()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.test == nil {
					return nil, fmt.Errorf("manual-on: no test image")
				}
				if dp.lastReport != nil {
					return nil, fmt.Errorf("manual-on: report present in manual mode")
				}
				if dp.learned[dp.channel] {
					return nil, fmt.Errorf("manual-on: learned[%v] = true, want false", dp.channel)
				}
				return map[string]any{"manual": true}, nil
			},
		},
		selftestStep{
			name: "manual-inspect", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.requestInspect()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.lastReport == nil {
					return nil, fmt.Errorf("manual-inspect: no report")
				}
				if !dp.learned[dp.channel] {
					return nil, fmt.Errorf("manual-inspect: not learned")
				}
				if !dp.manual {
					return nil, fmt.Errorf("manual-inspect: manual mode turned off")
				}
				if !bytes.Equal(dp.test.Data, simulate(s.a, dp, core.ChRed).Data) {
					return nil, fmt.Errorf("manual-inspect: test image differs from the simulation")
				}
				return map[string]any{"defects": len(dp.lastReport.Defects)}, nil
			},
		},
		selftestStep{
			name: "manual-load", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				manualWant = simulate(s.a, dp, core.ChRed).Clone()
				manualX, manualY = dp.findSrPoint()
				for y := manualY - 7; y <= manualY+7; y++ {
					if y < 0 || y >= manualWant.Height {
						continue
					}
					for x := manualX - 7; x <= manualX+7; x++ {
						if x < 0 || x >= manualWant.Width {
							continue
						}
						manualWant.Data[y*manualWant.Width+x] = 255
					}
				}
				if err := os.MkdirAll(s.out, 0o755); err != nil {
					return
				}
				path := filepath.Join(s.out, "manual-image.png")
				f, err := os.Create(path)
				if err != nil {
					return
				}
				g := image.NewGray(image.Rect(0, 0, manualWant.Width, manualWant.Height))
				copy(g.Pix, manualWant.Data)
				err = png.Encode(f, g)
				f.Close()
				if err != nil {
					return
				}
				dp.requestLoadImage(path)
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.user.img == nil {
					return nil, fmt.Errorf("manual-load: no loaded image")
				}
				if !bytes.Equal(dp.user.img.Data, manualWant.Data) {
					return nil, fmt.Errorf("manual-load: loaded pixels differ from the written file")
				}
				if dp.test != dp.user.img {
					return nil, fmt.Errorf("manual-load: test image is not the loaded image")
				}
				if dp.lastReport != nil {
					return nil, fmt.Errorf("manual-load: report present in manual mode")
				}
				if dp.liveFollow {
					return nil, fmt.Errorf("manual-load: live follow still on")
				}
				if dp.snapSrc.on {
					return nil, fmt.Errorf("manual-load: snap source still on")
				}
				return map[string]any{"loaded": true}, nil
			},
		},
		selftestStep{
			name: "manual-load-inspect", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.requestInspect()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.lastReport == nil {
					return nil, fmt.Errorf("manual-load-inspect: no report")
				}
				hits := 0
				for _, d := range dp.lastReport.Defects {
					if d.BBox.X <= manualX && manualX < d.BBox.X+d.BBox.W &&
						d.BBox.Y <= manualY && manualY < d.BBox.Y+d.BBox.H {
						hits++
					}
				}
				if hits == 0 {
					return nil, fmt.Errorf("manual-load-inspect: no defect contains (%d,%d)", manualX, manualY)
				}
				return map[string]any{"defects": len(dp.lastReport.Defects), "hits": hits}, nil
			},
		},
		selftestStep{
			name: "manual-restore", enter: func(a *App) {
				dp := s.panel("defect").(*DefectPanel)
				dp.clearUserImage()
				dp.manual = false
				dp.renderRev++
				dp.golden, dp.test = nil, nil
				dp.requestRun()
			},
			check: func(s *Selftest) (map[string]any, error) {
				dp := s.panel("defect").(*DefectPanel)
				if dp.user.img != nil {
					return nil, fmt.Errorf("manual-restore: loaded image still present")
				}
				if dp.lastReport == nil {
					return nil, fmt.Errorf("manual-restore: no report")
				}
				if !bytes.Equal(dp.test.Data, simulate(s.a, dp, core.ChRed).Data) {
					return nil, fmt.Errorf("manual-restore: test image differs from the simulation")
				}
				return map[string]any{"restored": true}, nil
			},
		},
	)
	s.steps = filterSteps(s.steps, a.profile)
	if a.live != nil && a.live.cli != nil {
		// -link was given to the teaching program: the other steps run on the program's own
		// simulation, and the last step follows the studio that is expected on that address.
		if dp, ok := s.panel("defect").(*DefectPanel); ok {
			dp.liveFollow = false
		}
		s.steps = append(s.steps, liveSelftestStep(s))
	}
	return s
}

// studioOnlySteps run only in the camera studio; teachOnlySteps only in the teaching program.
// A step in neither set runs in both.
var studioOnlySteps = map[string]bool{
	"light-dark-zh":           true,
	"interact-measure":        true,
	"interact-defect-rect":    true,
	"interact-defect-ellipse": true,
	"interact-defect-polygon": true,
	"light-light-ko":          true,
	"defect-color-red":        true,
	"defect-color-green":      true,
	"defect-color-blue":       true,
	"select-setup":            true,
	"select-pick":             true,
	"select-ctrl":             true,
	"select-rect":             true,
	"select-edit":             true,
	"select-caps":             true,
	"product-variation":       true,
	"product-off":             true,
	"product-on":              true,
	"studio-save":             true,
	"studio-warp":             true,
	"studio-warp-reset":       true,
}

var teachOnlySteps = map[string]bool{
	"teach-start":           true,
	"align-dark-zh":         true,
	"interact-align-roi":    true,
	"align-overlay":         true,
	"align-offset":          true,
	"align-lift":            true,
	"align-reset":           true,
	"interact-defect-probe": true,
	"align-light-ko":        true,
	"defect-params-dialog":  true,
	"inspect-good":          true,
	"inspect-overkill":      true,
	"inspect-size":          true,
	"inspect-miss":          true,
	"inspect-relearn":       true,
	"defaults-detected":     true,
	"edge-margin-off":       true,
	"edge-margin-on":        true,
	"snap-camera":           true,
	"snap-stale":            true,
	"snap-resnap":           true,
	"snap-channel":          true,
	"snap-load":             true,
	"snap-off":              true,
	"bg-global-learn":       true,
	"bg-global-blobs":       true,
	"bg-local-learn":        true,
	"bg-local-dots":         true,
	"manual-on":             true,
	"manual-inspect":        true,
	"manual-load":           true,
	"manual-load-inspect":   true,
	"manual-restore":        true,
}

// filterSteps keeps the steps that belong to the profile, in order.
func filterSteps(steps []selftestStep, p Profile) []selftestStep {
	out := make([]selftestStep, 0, len(steps))
	for _, st := range steps {
		if p == ProfileStudio && teachOnlySteps[st.name] {
			continue
		}
		if p != ProfileStudio && studioOnlySteps[st.name] {
			continue
		}
		out = append(out, st)
	}
	return out
}

// srRmsDelta returns the RMS of the SR-material texture difference between the live scene
// and a pristine light scene built from the same MEDIAN and material map.
func srRmsDelta(a *App) float64 {
	if a.Scene == nil {
		return 0
	}
	pristine, err := core.NewLightScene(a.Scene.Median, a.Scene.Material)
	if err != nil {
		return 0
	}
	live := a.Scene.Light
	if live == nil {
		return 0
	}
	sum := 0.0
	count := 0
	for i := range a.Scene.Material {
		if a.Scene.Material[i] != core.MatSR {
			continue
		}
		d := float64(live.Texture[i]) - float64(pristine.Texture[i])
		sum += d * d
		count++
	}
	if count == 0 {
		return 0
	}
	return math.Sqrt(sum / float64(count))
}

// simulate renders the red/green/blue test image of the defect panel's current scenario in-process.
func simulate(a *App, dp *DefectPanel, ch core.CameraChannel) *core.GrayImage {
	light := a.Light.Clone()
	_, test := core.RenderDefectPairWarped(a.Scene.Light, &light, ch, dp.sceneStrokes(), dp.gvStrokes(), dp.noise, dp.warp)
	return test
}

// imageToScreen 把图像坐标转换为屏幕坐标。
func imageToScreen(s *Selftest, x, y float64) imgui.Vec2 {
	o, _ := s.a.Viewer.Canvas()
	return imgui.NewVec2(o.X+s.a.Viewer.panX+float32(x)*s.a.Viewer.zoom, o.Y+s.a.Viewer.panY+float32(y)*s.a.Viewer.zoom)
}

// withCtrl 把序列中每个输入的 ctrl 置为 true。
func withCtrl(seq []viewerInput) []viewerInput {
	for i := range seq {
		seq[i].ctrl = true
	}
	return seq
}

// Step advances the script; called once per frame after the UI is built.
func (s *Selftest) Step() {
	if s.finished || s.pendingShot != "" || s.capturing {
		return
	}
	now := time.Now()
	if !s.frameStart.IsZero() {
		s.frameMs = append(s.frameMs, float64(now.Sub(s.frameStart).Microseconds())/1000)
	}
	s.frameStart = now
	if now.After(s.deadline) {
		s.errors = append(s.errors, "timeout in step "+s.steps[s.idx].name)
		s.finish()
		return
	}
	st := s.steps[s.idx]
	if s.frames == 0 {
		s.stepStart = now
		s.frameMs = s.frameMs[:0]
		if st.enter != nil {
			st.enter(s.a)
		}
		s.seq, s.seqIdx = nil, 0
		if st.input != nil {
			s.seq = st.input(s)
		}
	}
	s.frames++
	if s.seqIdx < len(s.seq) {
		// Injected input is consumed by the viewer on the next frame.
		s.a.Viewer.Inject(s.seq[s.seqIdx])
		s.seqIdx++
		s.quiet = 0
		return
	}
	if st.name == "load" && s.a.Scene == nil {
		if s.a.loadErr != "" && !s.a.loading {
			s.errors = append(s.errors, s.a.loadErr)
			s.finish()
		}
		return
	}
	if s.a.Jobs.AnyBusy() || s.a.loading {
		s.quiet = 0
		return
	}
	s.quiet++
	if s.quiet < 15 || s.frames < 20 {
		return
	}
	res := map[string]any{
		"settleMs":     float64(time.Since(s.stepStart).Microseconds()) / 1000,
		"frames":       s.frames,
		"frameMeanMs":  mean(s.frameMs),
		"frameP95Ms":   percentile(s.frameMs, 0.95),
		"viewerHasImg": s.a.Viewer.HasImage(),
	}
	if st.check != nil {
		detail, err := st.check(s)
		if err != nil {
			s.errors = append(s.errors, st.name+": "+err.Error())
			res["pass"] = false
		} else {
			res["pass"] = true
		}
		for k, v := range detail {
			res[k] = v
		}
	}
	s.results[st.name] = res
	if st.shot != "" {
		s.pendingShot = filepath.Join(s.out, st.shot)
	}
	s.idx++
	s.frames, s.quiet = 0, 0
	if s.idx >= len(s.steps) && s.pendingShot == "" {
		s.finish()
	}
}

// AfterRender takes a pending screenshot between frames (PrintWindow re-enters the GLFW
// refresh callback, so it must not run inside a frame).
func (s *Selftest) AfterRender() {
	if s.pendingShot == "" || s.capturing {
		return
	}
	path := s.pendingShot
	s.capturing = true
	if err := captureWindow(s.a.profile.WindowTitle(), path); err != nil {
		s.errors = append(s.errors, filepath.Base(path)+": "+err.Error())
	} else {
		s.shots = append(s.shots, path)
	}
	s.capturing = false
	s.pendingShot = ""
	if s.idx >= len(s.steps) && !s.finished {
		s.finish()
	}
}

func (s *Selftest) finish() {
	s.finished = true
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	jobs := map[string]any{}
	for k, v := range s.a.timings {
		jobs[k] = map[string]any{"count": len(v), "meanMs": mean(v), "maxMs": percentile(v, 1)}
	}
	report := map[string]any{
		"imgui":        imgui.Version(),
		"go":           runtime.Version(),
		"cpus":         runtime.NumCPU(),
		"reference":    s.a.RefRoot,
		"models":       len(s.a.Models),
		"steps":        s.results,
		"jobs":         jobs,
		"heapAllocMB":  float64(ms.HeapAlloc) / (1 << 20),
		"sysMB":        float64(ms.Sys) / (1 << 20),
		"screenshots":  s.shots,
		"errors":       s.errors,
		"totalSeconds": time.Since(s.a.start).Seconds(),
	}
	if s.a.Scene != nil {
		report["model"] = s.a.Scene.Info.ID
		report["image"] = fmt.Sprintf("%dx%d", s.a.Scene.W, s.a.Scene.H)
		report["loadMs"] = s.a.Scene.LoadMs
		report["machineRois"] = len(s.a.Scene.MachineRois)
	}
	data, _ := json.MarshalIndent(report, "", "  ")
	_ = os.WriteFile(filepath.Join(s.out, "native-metrics.json"), data, 0o644)
	fmt.Println(string(data))
	s.a.be.SetShouldClose(true)
}

func mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range v {
		sum += x
	}
	return sum / float64(len(v))
}

func percentile(v []float64, q float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	i := int(float64(len(s)-1) * q)
	return s[i]
}
