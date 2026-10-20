package app

import (
	"bytes"
	"strings"
	"testing"

	"afvi/teach-imgui/internal/core"
	snapfmt "afvi/teach-imgui/internal/snap"
)

func TestWarpPresetsAreValidCaptureWarps(t *testing.T) {
	seen := map[string]bool{}
	for _, pr := range warpPresets() {
		if seen[pr.id] {
			t.Errorf("duplicate preset id %q", pr.id)
		}
		seen[pr.id] = true
		if err := core.ValidateCaptureWarp(pr.warp.Geometry()); err != nil {
			t.Errorf("preset %q is not a valid capture warp: %v", pr.id, err)
		}
		if identity := pr.warp.GeometryIdentity(); identity != (pr.id == "none") {
			t.Errorf("preset %q: GeometryIdentity() = %v", pr.id, identity)
		}
		for _, lang := range []Lang{LangZh, LangEn, LangKo} {
			if _, ok := dictionaries[lang]["warp.preset."+pr.id]; !ok {
				t.Errorf("preset %q has no %s translation", pr.id, lang)
			}
		}
	}
	for _, id := range []string{"none", "slight", "severe", "lift"} {
		if !seen[id] {
			t.Errorf("missing preset %q", id)
		}
	}
	if !warpPresetByID("nope").GeometryIdentity() {
		t.Error("an unknown preset id must be the identity")
	}
}

func TestSetWarpGeometryKeepsThePhotometricPart(t *testing.T) {
	dst := core.WarpParams{Scale: 1, Gain: 1.4, BlurSigma: 3, NoiseSigma: 5, Offset: 7, Gradient: 0.2, Tx: 99, TiltX: 0.05}
	setWarpGeometry(&dst, warpPresetByID("severe"))
	if dst.Gain != 1.4 || dst.BlurSigma != 3 || dst.NoiseSigma != 5 || dst.Offset != 7 || dst.Gradient != 0.2 {
		t.Errorf("photometric fields changed: %+v", dst)
	}
	if want := warpPresetByID("severe"); dst.Tx != want.Tx || dst.Ty != want.Ty || dst.ThetaDeg != want.ThetaDeg || dst.Scale != want.Scale || dst.TiltX != 0 {
		t.Errorf("geometry not replaced by the preset: %+v", dst)
	}
}

func TestWarpSummary(t *testing.T) {
	got := warpSummary(&snapfmt.Warp{Tx: 6, Ty: -4, ThetaDeg: 0.5, Scale: 1.01, TiltX: 0.01, BendAmplitude: 5, BendWavelength: 800})
	for _, want := range []string{"dx=6.0", "dy=-4.0", "rot=0.50°", "scale=1.010", "tilt=(0.010,0.000)", "bend=5.0@800"} {
		if !strings.Contains(got, want) {
			t.Errorf("summary %q lacks %q", got, want)
		}
	}
}

func TestStudioWarpFlowsIntoTheDescription(t *testing.T) {
	a := newTestApp(t, ProfileStudio)
	dp := defectPanelOf(a)
	if dp.captureWarp() != nil || dp.snapMeta().Recipe.Warp != nil {
		t.Fatal("a fresh studio has a warp in its description")
	}
	dp.warp = core.IdentityWarp
	setWarpGeometry(&dp.warp, warpPresetByID("severe"))
	w := dp.snapMeta().Recipe.Warp
	if w == nil || w.Tx != 40 || w.Ty != 25 {
		t.Fatalf("description warp %+v", w)
	}
	if dp.captureWarp() == nil {
		t.Error("captureWarp is nil for a misaligned capture")
	}
	if err := core.ValidateCaptureWarp(dp.snapMeta().Recipe.CoreWarp()); err != nil {
		t.Errorf("the description warp is not a valid capture warp: %v", err)
	}
	dp.warp = core.IdentityWarp
	if dp.snapMeta().Recipe.Warp != nil {
		t.Error("an identity warp must not appear in the description")
	}
}

func TestStudioRenderFollowsTheWarp(t *testing.T) {
	a := newTestApp(t, ProfileStudio)
	dp := defectPanelOf(a)
	dp.requestRun()
	pollJobs(t, a)
	nominal := append([]byte(nil), dp.test.Data...)

	dp.warp = core.IdentityWarp
	dp.warp.Tx, dp.warp.Ty = 6, 4
	dp.renderRev++
	dp.requestRun()
	pollJobs(t, a)
	light := a.Light.Clone()
	_, want := core.RenderDefectPairWarped(a.Scene.Light, &light, dp.channel, dp.sceneStrokes(), dp.gvStrokes(), dp.noise, dp.warp)
	if !bytes.Equal(dp.test.Data, want.Data) {
		t.Error("the studio test image is not the warped render")
	}
	if bytes.Equal(dp.test.Data, nominal) {
		t.Error("the warp did not change the rendered image")
	}
	w, h := dp.test.Width, dp.test.Height
	for y := 0; y < h-4; y++ {
		for x := 0; x < w-6; x++ {
			if dp.test.Data[(y+4)*w+x+6] != nominal[y*w+x] {
				t.Fatalf("pixel (%d,%d) did not move by (6,4)", x, y)
			}
		}
	}
}

func TestWarpedStudioLocksPaintingAndSelecting(t *testing.T) {
	a := newTestApp(t, ProfileStudio)
	dp := defectPanelOf(a)
	if !dp.PaletteSupports(PalettePick) {
		t.Fatal("author mode must support the pick tool at the nominal position")
	}
	dp.tool = toolBrightRect
	dp.warp = core.IdentityWarp
	dp.warp.Tx = 10
	if dp.PaletteSupports(PalettePick) || dp.PaletteSupports(PaletteRect) {
		t.Error("the palette must be greyed out while the capture is misaligned")
	}
	before := len(dp.strokes)
	for _, kind := range []PointerKind{PointerDown, PointerMove, PointerUp} {
		if dp.Pointer(PointerEvent{Kind: kind, X: 20, Y: 20}) {
			t.Errorf("pointer %v was claimed while misaligned", kind)
		}
	}
	if len(dp.strokes) != before {
		t.Error("a stroke was painted while misaligned")
	}
	dp.warp = core.IdentityWarp
	if !dp.PaletteSupports(PalettePick) {
		t.Error("the palette did not come back with the nominal position")
	}
}

func TestLiveRendererHonoursTheWarp(t *testing.T) {
	a := newTestApp(t, ProfileStudio)
	dp := defectPanelOf(a)
	dp.warp = core.IdentityWarp
	setWarpGeometry(&dp.warp, warpPresetByID("lift"))
	meta := dp.snapMeta()
	r := &liveRenderer{scene: a.Scene, cr: meta.Recipe.Core()}

	test, err := r.render("red", "test")
	if err != nil {
		t.Fatal(err)
	}
	light := a.Light.Clone()
	golden, want := core.RenderDefectPairWarped(a.Scene.Light, &light, core.ChRed, dp.sceneStrokes(), dp.gvStrokes(), dp.noise, dp.warp)
	if !bytes.Equal(test.Pix, want.Data) {
		t.Error("the live test frame is not the warped render")
	}
	got, err := r.render("red", "golden")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Pix, golden.Data) {
		t.Error("the live golden frame is not the warped golden")
	}
	// The good sample is the taught reference and stays at the nominal position.
	good, err := r.render("red", "good")
	if err != nil {
		t.Fatal(err)
	}
	wantGood := core.RenderGoodSample(a.Scene.Light, &light, core.ChRed, goodSampleNoise(dp.noise))
	if !bytes.Equal(good.Pix, wantGood.Data) {
		t.Error("the good sample moved with the warp")
	}
}

func TestSnapStaleSeesTheWarp(t *testing.T) {
	a := newTestApp(t, ProfileStudio)
	dp := defectPanelOf(a)
	dp.warp = core.IdentityWarp
	dp.warp.Tx = 12
	meta := dp.snapMeta()
	dp.snapSrc.on = true
	dp.snapSrc.meta = &meta
	dp.renderRev++
	if dp.snapStale() {
		t.Error("a capture of the current scenario is stale")
	}
	dp.warp.Tx = 13
	dp.renderRev++
	if !dp.snapStale() {
		t.Error("moving the product must make the adopted capture stale")
	}
}

func TestPackOverlayPutsTemplateInRedAndInputInCyan(t *testing.T) {
	tpl := &core.GrayImage{Width: 2, Height: 1, Data: []byte{10, 200}}
	in := &core.GrayImage{Width: 2, Height: 1, Data: []byte{50, 200}}
	pix := PackOverlay(tpl, in)
	defer PutRGBA(pix)
	want := []byte{10, 50, 50, 255, 200, 200, 200, 255}
	if !bytes.Equal(pix[:8], want) {
		t.Errorf("overlay pixels %v, want %v", pix[:8], want)
	}
}

func TestAlignDefaultsToTheOverlayView(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	var ap *AlignPanel
	for _, p := range a.panels {
		if x, ok := p.(*AlignPanel); ok {
			ap = x
		}
	}
	if ap == nil {
		t.Fatal("no align panel")
	}
	// The teach profile now also carries the forced-alignment panels; the bench is unchanged.
	if ap.viewKind != "overlay" {
		t.Errorf("default view %q, want overlay", ap.viewKind)
	}
	for i, v := range alignViews {
		if alignViewIndex(v) != i {
			t.Errorf("alignViewIndex(%q) = %d, want %d", v, alignViewIndex(v), i)
		}
	}
	if alignViewIndex("bogus") != 0 {
		t.Error("an unknown view must fall back to the first entry")
	}
}
