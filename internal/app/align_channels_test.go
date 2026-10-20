package app

import (
	"strings"
	"testing"

	"afvi/teach-imgui/internal/core"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// smallWarp is the capture warp TestDetectRunsOnlyWhenAligned uses: the default shift is
// huge for the 96x64 test model, so the tests below misalign the product only slightly.
func smallWarp() core.WarpParams {
	return core.WarpParams{Tx: 3, Ty: -2, ThetaDeg: 0.3, Scale: 1, Gain: 1}
}

// locatedMatches counts the ROIs the alignment report says were located.
func locatedMatches(out *AlignOutcome) int {
	if out == nil {
		return 0
	}
	n := 0
	for _, m := range out.Report.Matches {
		if m.Located {
			n++
		}
	}
	return n
}

func TestAlignEveryChannelWithTeachLight(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	rois := pipelineRois(a.Scene)
	a.align.SetRois(rois)
	if !a.align.Arm() {
		t.Fatal("Arm() failed with two valid ROIs")
	}
	dp.warp = smallWarp()

	for _, ch := range []core.CameraChannel{core.ChRed, core.ChGreen, core.ChBlue} {
		if dp.channel == ch {
			// SetChannel does nothing when the channel is unchanged.
			dp.renderRev++
			dp.requestRun()
		} else {
			dp.SetChannel(ch)
		}
		pollJobs(t, a)

		out := dp.AlignOutcome()
		if got := dp.AlignState(); got != alignDone {
			t.Fatalf("channel %v: AlignState() = %v, want alignDone (%+v)\n%s",
				ch, got, out, explainAlign(out, len(rois)))
		}
		if out == nil {
			t.Fatalf("channel %v: AlignOutcome() is nil", ch)
		}
		if !out.Usable {
			t.Errorf("channel %v: Usable = false\n%s", ch, explainAlign(out, len(rois)))
		}
		if out.Channel != ch {
			t.Errorf("channel %v: out.Channel = %v\n%s", ch, out.Channel, explainAlign(out, len(rois)))
		}
		if out.TplFlat {
			t.Errorf("channel %v: TplFlat = true, the design template must have contrast\n%s", ch, explainAlign(out, len(rois)))
		}
		if out.RawFlat {
			t.Errorf("channel %v: RawFlat = true, the capture must have contrast\n%s", ch, explainAlign(out, len(rois)))
		}
		if n := locatedMatches(out); n < 2 {
			t.Errorf("channel %v: %d located matches, want at least 2\n%s", ch, n, explainAlign(out, len(rois)))
		}
	}
}

func TestAlignRedOnlyLightExplainsFlatChannel(t *testing.T) {
	prev := currentLang
	SetLang(LangZh)
	defer SetLang(prev)

	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	a.Light = core.DefaultLightSetting() // red LED only: green and blue are dark
	a.align.SetRois(pipelineRois(a.Scene))
	if !a.align.Arm() {
		t.Fatal("Arm() failed with two valid ROIs")
	}
	dp.warp = smallWarp()
	dp.SetChannel(core.ChGreen)
	pollJobs(t, a)

	out := dp.AlignOutcome()
	if got := dp.AlignState(); got != alignFailed {
		t.Fatalf("AlignState() = %v, want alignFailed (%+v)", got, out)
	}
	if out == nil {
		t.Fatal("AlignOutcome() is nil")
	}
	if out.Channel != core.ChGreen {
		t.Errorf("out.Channel = %v, want ChGreen", out.Channel)
	}
	if !out.TplFlat {
		t.Error("TplFlat = false, the green design template must be flat")
	}
	if !out.RawFlat {
		t.Error("RawFlat = false, the green capture must be flat")
	}
	if n := locatedMatches(out); n != 0 {
		t.Errorf("%d located matches, want 0", n)
	}
	if !dp.InspectBlocked() {
		t.Error("InspectBlocked() = false, want true")
	}
	text := explainAlign(out, 2)
	if !strings.Contains(text, T("alignx.ch.green")) {
		t.Errorf("explanation does not name the green channel:\n%s", text)
	}
	if !strings.Contains(text, T("alignx.hintLight")) {
		t.Errorf("explanation does not carry the light hint:\n%s", text)
	}
	if strings.Contains(text, "alignx.") {
		t.Errorf("explanation contains an untranslated key:\n%s", text)
	}
	if got := dp.alignReason(); got != text {
		t.Errorf("alignReason() =\n%s\nwant\n%s", got, text)
	}
}

func TestAlignTemplateFollowsSnapLight(t *testing.T) {
	t.Setenv("AFVI_CAMERA_EXE", "") // render the missing frames in-process

	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	a.Light = core.DefaultLightSetting() // red only: the template must not take this light
	a.align.SetRois(pipelineRois(a.Scene))
	if !a.align.Arm() {
		t.Fatal("Arm() failed with two valid ROIs")
	}
	dp.warp = smallWarp()

	r := snapfmt.NewRecipe(core.DefaultProductOptions(), core.DefaultCaptureLight(), core.SensorNoise{}, nil, nil).WithWarp(dp.warp)
	meta := &snapfmt.Meta{
		Format:  snapfmt.Format,
		Version: snapfmt.Version,
		Model:   a.Scene.Info.Ref(),
		Width:   a.Scene.W,
		Height:  a.Scene.H,
		Recipe:  r,
		Images:  []snapfmt.Image{},
	}
	dp.snapSrc.on = true
	dp.snapSrc.meta = meta

	dp.SetChannel(core.ChGreen)
	pollJobs(t, a)

	out := dp.AlignOutcome()
	if got := dp.AlignState(); got != alignDone {
		t.Fatalf("AlignState() = %v, want alignDone (%+v)", got, out)
	}
	if out == nil {
		t.Fatal("AlignOutcome() is nil")
	}
	if !out.Usable {
		t.Errorf("Usable = false, the template must take the description's light (%+v)", out)
	}
	if out.TplFlat {
		t.Error("TplFlat = true, the green template rendered with the description's RGB light must have contrast")
	}
}
