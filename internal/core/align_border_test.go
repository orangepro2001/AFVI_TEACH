package core

import (
	"math"
	"testing"
)

// cornerScene is a textured image with a ROI flush against the bottom-right corner and a warp
// that pushes the product toward that corner, so the true window sticks out of the image.
func cornerScene() (template, input *GrayImage, roi AlignRoi, warp WarpParams) {
	const width, height = 400, 300
	template = makeTexture(width, height, 31)
	warp = warpParams(func(p *WarpParams) {
		p.Tx = 5
		p.Ty = 4
	})
	input = ApplyWarp(template, &warp, 31)
	roi = AlignRoi{Name: "corner", Rect: Rect{X: width - 100, Y: height - 100, W: 100, H: 100}}
	return template, input, roi, warp
}

func matchError(m RoiMatch, warp *WarpParams, width, height int) float64 {
	if m.Found == nil {
		return math.Inf(1)
	}
	truth := WarpPoint(warp, m.Template, width, height)
	return math.Hypot(m.Found.X-truth.X, m.Found.Y-truth.Y)
}

func TestMatchRoiFollowsAWindowThatLeavesTheImage(t *testing.T) {
	template, input, roi, warp := cornerScene()
	prior := Similarity{Scale: 1}

	cfg := DefaultAlignConfig
	if cfg.MinOverlap <= 0 || cfg.MinOverlap >= 1 {
		t.Fatalf("the default MinOverlap %v must allow partial windows", cfg.MinOverlap)
	}
	m := MatchRoi(template, input, roi, prior, cfg)
	if !m.Located || m.Found == nil {
		t.Fatalf("corner ROI not located: %+v", m)
	}
	if err := matchError(m, &warp, template.Width, template.Height); err > 0.7 {
		t.Errorf("match error %.2f px, want < 0.7", err)
	}
	if m.Overlap >= 1 || m.Overlap < cfg.MinOverlap {
		t.Errorf("overlap %.3f, want in [%.2f, 1)", m.Overlap, cfg.MinOverlap)
	}
	if m.Ncc < 0.95 {
		t.Errorf("ncc %.3f, want > 0.95", m.Ncc)
	}

	// MinOverlap 0 is the old behaviour: only windows fully inside the image count, so the match
	// is stuck at the border, short of the true position.
	strict := cfg
	strict.MinOverlap = 0
	s := MatchRoi(template, input, roi, prior, strict)
	if err := matchError(s, &warp, template.Width, template.Height); err < 3 {
		t.Errorf("strict match error %.2f px, want >= 3 (clipped at the border)", err)
	}
	if s.Found != nil && s.Overlap != 1 {
		t.Errorf("strict overlap %.3f, want 1", s.Overlap)
	}
}

func TestMatchRoiRefusesAWindowMostlyOutside(t *testing.T) {
	template, _, roi, _ := cornerScene()
	warp := warpParams(func(p *WarpParams) { p.Tx = 30; p.Ty = 30 })
	input := ApplyWarp(template, &warp, 31)
	m := MatchRoi(template, input, roi, Similarity{Scale: 1}, DefaultAlignConfig)
	// Half of the true window is outside the image (overlap 25%), far below the 90% limit.
	if err := matchError(m, &warp, template.Width, template.Height); err < 10 {
		t.Errorf("a window 75%% outside was matched to within %.2f px", err)
	}
}

func TestNccAtPartialWindowNeedsEnoughOverlap(t *testing.T) {
	img := makeTexture(100, 80, 3)
	pw, ph := 20, 20
	patch, ok := extractPatch(img, 80, 60, pw, ph)
	if !ok {
		t.Fatal("patch outside the image")
	}
	if got, want := nccAt(img, patch, pw, ph, 80, 60, 0), Ncc(patch, patch); math.Abs(got-want) > 1e-12 {
		t.Errorf("inside window: ncc %v, want %v", got, want)
	}
	// 2 px out on the right and 3 px out on the bottom: overlap 18*17/400 = 0.765.
	if got := nccAt(img, patch, pw, ph, 82, 63, 0); got != -2 {
		t.Errorf("strict partial window scored %v, want -2", got)
	}
	if got := nccAt(img, patch, pw, ph, 82, 63, 1); got != -2 {
		t.Errorf("MinOverlap 1 partial window scored %v, want -2", got)
	}
	if got := nccAt(img, patch, pw, ph, 82, 63, 0.9); got != -2 {
		t.Errorf("overlap 0.765 < 0.9 scored %v, want -2", got)
	}
	if got := nccAt(img, patch, pw, ph, 82, 63, 0.7); got <= -2 {
		t.Errorf("overlap 0.765 >= 0.7 not scored: %v", got)
	}
	// A window with no overlap at all is never valid.
	if got := nccAt(img, patch, pw, ph, 150, 150, 0.1); got != -2 {
		t.Errorf("window outside scored %v, want -2", got)
	}
	// The visible part of the window is compared with the same pixels of the patch.
	partial := nccAt(img, patch, pw, ph, 80, 60, 0.5)
	if math.Abs(partial-nccAt(img, patch, pw, ph, 80, 60, 0)) > 1e-12 {
		t.Errorf("a fully inside window must not depend on MinOverlap")
	}
}

func TestAlignWithRoisIsOKWhenACornerRoiLeavesTheImage(t *testing.T) {
	const width, height = 400, 300
	template := makeTexture(width, height, 41)
	warp := warpParams(func(p *WarpParams) {
		p.Tx = 5
		p.Ty = 4
		p.ThetaDeg = 0.1
	})
	input := ApplyWarp(template, &warp, 41)
	rois := []AlignRoi{
		{Name: "TL", Rect: Rect{X: 20, Y: 20, W: 60, H: 60}},
		{Name: "TR", Rect: Rect{X: width - 80, Y: 20, W: 60, H: 60}},
		{Name: "BL", Rect: Rect{X: 20, Y: height - 80, W: 60, H: 60}},
		{Name: "BR", Rect: Rect{X: width - 100, Y: height - 100, W: 100, H: 100}},
	}
	rep := AlignWithRois(template, input, rois, DefaultAlignConfig, Similarity{Scale: 1})
	if rep.Level != VerdictOK {
		t.Fatalf("level %s, reasons %+v", rep.Level, rep.Reasons)
	}
	strict := DefaultAlignConfig
	strict.MinOverlap = 0
	old := AlignWithRois(template, input, rois, strict, Similarity{Scale: 1})
	if old.Level == VerdictOK {
		t.Errorf("the strict matcher unexpectedly passed; the scene no longer shows the border problem")
	}
}

func TestDiagnoseRois(t *testing.T) {
	const width, height = 400, 300
	template := makeTexture(width, height, 51)
	warp := warpParams(func(p *WarpParams) {
		p.Tx = 5
		p.Ty = 4
		// Covers the left half of the true window of "mid" (true window x 195..255).
		p.Occlusions = []Occlusion{{Rect: Rect{X: 195, Y: 100, W: 30, H: 80}, Value: 0}}
	})
	rois := []AlignRoi{
		{Name: "TL", Rect: Rect{X: 20, Y: 20, W: 60, H: 60}},
		{Name: "TR", Rect: Rect{X: width - 80, Y: 20, W: 60, H: 60}},
		{Name: "BL", Rect: Rect{X: 20, Y: height - 80, W: 60, H: 60}},
		{Name: "BR", Rect: Rect{X: width - 100, Y: height - 100, W: 100, H: 100}},
		{Name: "mid", Rect: Rect{X: 190, Y: 110, W: 60, H: 60}},
		{Name: "flat", Rect: Rect{X: 300, Y: 150, W: 40, H: 40}},
	}
	// Make the "flat" ROI featureless so that it is not found at all.
	for y := 150; y < 190; y++ {
		for x := 300; x < 340; x++ {
			template.Data[y*width+x] = 90
		}
	}
	input := ApplyWarp(template, &warp, 51)
	rep := AlignWithRois(template, input, rois, DefaultAlignConfig, Similarity{Scale: 1})
	diag := DiagnoseRois(&rep, rois, &warp, width, height)
	if len(diag) != len(rois) {
		t.Fatalf("%d diagnoses for %d ROIs", len(diag), len(rois))
	}

	br := diag[3]
	if math.Abs(br.OverhangPx-5) > 1e-9 {
		t.Errorf("BR overhang %.3f, want 5 (the larger of the 5 px right and 4 px down)", br.OverhangPx)
	}
	if br.MatchErrPx > 0.7 {
		t.Errorf("BR match error %.2f px", br.MatchErrPx)
	}
	if want := (95.0 / 100) * (96.0 / 100); math.Abs(br.InsideShare-want) > 1e-9 {
		t.Errorf("BR inside share %.4f, want %.4f", br.InsideShare, want)
	}
	if tl := diag[0]; tl.OverhangPx != 0 || tl.Occluded != 0 || tl.InsideShare != 1 {
		t.Errorf("TL should be inside and uncovered: %+v", tl)
	}
	mid := diag[4]
	if math.Abs(mid.Occluded-0.5) > 1e-9 {
		t.Errorf("mid occluded share %.3f, want 0.5", mid.Occluded)
	}
	if rep.Matches[5].Found != nil || rep.Matches[5].Reason != "flat" {
		t.Errorf("the featureless ROI was not reported flat: %+v", rep.Matches[5])
	}
	for i, d := range diag {
		if rep.Matches[i].Found == nil && (!math.IsNaN(d.MatchErrPx) || !math.IsNaN(d.FitDevPx)) {
			t.Errorf("%s has no match but the diagnosis has numbers: %+v", rois[i].Name, d)
		}
		if rep.Matches[i].Found != nil && rep.Fit != nil && math.IsNaN(d.FitDevPx) {
			t.Errorf("%s has a match and a fit but no deviation", rois[i].Name)
		}
	}

	// Without a fit there is no deviation from it.
	noFit := rep
	noFit.Fit = nil
	for i, d := range DiagnoseRois(&noFit, rois, &warp, width, height) {
		if !math.IsNaN(d.FitDevPx) {
			t.Errorf("ROI %d: deviation %v without a fit", i, d.FitDevPx)
		}
	}
}

func TestMatchCostGrowsWithTheSearch(t *testing.T) {
	rois := []AlignRoi{{Name: "a", Rect: Rect{X: 0, Y: 0, W: 280, H: 250}}}
	cfg := DefaultAlignConfig
	cfg.SearchRadius = 230
	cfg.PyramidFactor = 4
	fast := MatchCost(rois, cfg)
	cfg.PyramidFactor = 1
	slow := MatchCost(rois, cfg)
	if !(slow > 100*fast) {
		t.Errorf("pyramid 1 cost %.3g vs pyramid 4 cost %.3g: the difference must be large", slow, fast)
	}
	if MatchCost(nil, cfg) != 0 {
		t.Error("no ROIs must cost nothing")
	}
}
