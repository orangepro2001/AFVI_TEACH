package app

import (
	"testing"

	"afvi/teach-imgui/internal/core"
)

// legacyDetect turns the forced alignment off on a teach app, restoring the behaviour of the
// pipeline before AlignFlow and the golden board existed (the detect run renders and inspects
// the raw capture against the design layers).
func legacyDetect(a *App) {
	a.align = nil
	a.golden = nil
}

// pipelineRois returns two ROIs on the corners of the synthetic model's SR rectangle (x 36..92,
// y 8..56 of the 96x64 picture): a corner has a gradient in both directions, so each window is
// unique, and the two sit on opposite corners so a shift and a rotation are measurable.
func pipelineRois(scene *Scene) []core.AlignRoi {
	return []core.AlignRoi{
		{Name: "R1", Rect: core.Rect{X: 24, Y: 0, W: 24, H: 24}},
		{Name: "R2", Rect: core.Rect{X: 72, Y: 40, W: 24, H: 24}},
	}
}

func TestDetectBlockedWithoutAlignment(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	if !dp.InspectBlocked() {
		t.Fatal("a teach panel without an armed flow must block the inspection")
	}
	dp.requestRun()
	pollJobs(t, a)
	if dp.lastReport != nil {
		t.Error("lastReport is set although the alignment is not confirmed")
	}
	if got := dp.AlignState(); got != alignOff {
		t.Errorf("AlignState() = %v, want alignOff", got)
	}
	if dp.test != dp.raw {
		t.Error("test image is not the raw capture before the alignment is confirmed")
	}
	if dp.raw == nil {
		t.Error("raw capture is nil")
	}
	if dp.InspectBlocked() != true {
		t.Error("InspectBlocked() = false after the run")
	}
}

func TestDetectRunsOnlyWhenAligned(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if a.align == nil {
		t.Fatal("teach app has no AlignFlow")
	}
	a.align.SetRois(pipelineRois(a.Scene))
	if !a.align.Arm() {
		t.Fatal("Arm() failed with two valid ROIs")
	}
	// The default capture shift (14, -10 px) is huge for the 96x64 test model; use a small one.
	dp.warp = core.WarpParams{Tx: 3, Ty: -2, ThetaDeg: 0.3, Scale: 1, Gain: 1}
	dp.renderRev++
	dp.requestRun()
	pollJobs(t, a)

	// The synthetic model is small but not degenerate: two 16x16 boxes on the design picture
	// are enough for the working-resolution alignment (scale 2) to find a fit whose verdict is
	// not NG, so the panel aligns and inspects. If this ever regresses the assertions below
	// fail rather than silently pass.
	if got := dp.AlignState(); got != alignDone {
		t.Fatalf("AlignState() = %v, want alignDone (outcome %+v)", got, dp.alignOut)
	}
	if dp.lastReport == nil {
		t.Fatal("lastReport is nil after a successful alignment")
	}
	if dp.test == nil || dp.aligned == nil || dp.test != dp.aligned {
		t.Error("the test image is not the aligned image")
	}
	if dp.raw == nil || dp.test == dp.raw {
		t.Error("the test image still is the raw capture")
	}
	if dp.InspectBlocked() {
		t.Error("InspectBlocked() = true after a successful alignment")
	}
	// The aligned capture is inspected with its no-data border skipped (the replicated border is
	// not product); the raw capture has no such mask.
	reg := dp.testRegions(a.Scene)
	if len(reg.Skip) != a.Scene.W*a.Scene.H {
		t.Fatalf("Skip mask has %d entries, want %d", len(reg.Skip), a.Scene.W*a.Scene.H)
	}
	if !reg.Skip[0] || reg.Skip[(a.Scene.H/2)*a.Scene.W+a.Scene.W/2] {
		t.Errorf("Skip mask: corner %v (want true), centre %v (want false)", reg.Skip[0], reg.Skip[(a.Scene.H/2)*a.Scene.W+a.Scene.W/2])
	}
	dp.test = dp.raw
	if dp.testRegions(a.Scene).Skip != nil {
		t.Error("the raw capture must be inspected without a Skip mask")
	}
}

func TestEditingRoisDisarms(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	a.align.SetRois(pipelineRois(a.Scene))
	if !a.align.Arm() {
		t.Fatal("Arm() failed")
	}
	dp.requestRun()
	pollJobs(t, a)
	if dp.AlignState() == alignOff {
		t.Fatal("the flow is not armed after the run")
	}
	rois := append([]core.AlignRoi(nil), a.align.Rois...)
	rois[0].X += 4
	a.align.SetRois(rois)
	if got := dp.AlignState(); got != alignOff {
		t.Errorf("AlignState() = %v after editing the ROIs, want alignOff", got)
	}
	if !dp.InspectBlocked() {
		t.Error("InspectBlocked() = false after the ROIs were edited")
	}
	dp.requestRun()
	pollJobs(t, a)
	if dp.AlignState() != alignOff || dp.lastReport != nil {
		t.Errorf("state %v report %v after an edit-and-run, want alignOff/nil", dp.AlignState(), dp.lastReport)
	}
	if dp.test != dp.raw {
		t.Error("test image is not the raw capture after the flow was disarmed")
	}
}

func TestStudioNeverAligns(t *testing.T) {
	a := newTestApp(t, ProfileStudio)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	if dp.a.alignFlow() != nil {
		t.Fatal("studio app has an AlignFlow")
	}
	if got := dp.AlignState(); got != alignOff {
		t.Errorf("AlignState() = %v in the studio, want alignOff", got)
	}
	if dp.InspectBlocked() {
		t.Error("InspectBlocked() = true in the studio")
	}
	dp.requestRun()
	pollJobs(t, a)
	if dp.lastReport != nil {
		t.Error("the studio inspected; author mode must never inspect")
	}
	if dp.raw != nil {
		t.Error("the studio aligned its render (raw capture) - author mode must not align")
	}
}
