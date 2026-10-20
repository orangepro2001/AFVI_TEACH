package app

import (
	"reflect"
	"testing"

	"afvi/teach-imgui/internal/core"
)

// testRois returns n machine ROIs for a synthetic scene.
func testRois(n int) []core.AlignRoi {
	out := make([]core.AlignRoi, n)
	for i := 0; i < n; i++ {
		out[i] = core.AlignRoi{
			Name: "P" + string(rune('1'+i)),
			Rect: core.Rect{X: float64(10 * i), Y: float64(5 * i), W: 8, H: 8},
		}
	}
	return out
}

// newFlowTestApp builds a minimal App with a teach profile and a flow.
func newFlowTestApp() *App {
	a := &App{profile: ProfileTeach}
	a.align = newAlignFlow(a)
	return a
}

func TestAlignFlowModelLoaded(t *testing.T) {
	a := newFlowTestApp()
	f := a.alignFlow()
	scene := &Scene{MachineRois: testRois(3)}
	f.SetScale(4)
	f.SetConfig(core.AlignConfig{SearchRadius: 7})
	f.Armed = true
	before := f.Rev

	f.modelLoaded(scene)
	if f.Rev != before+1 {
		t.Errorf("Rev = %d after modelLoaded, want %d", f.Rev, before+1)
	}
	if f.Armed {
		t.Error("Armed = true after modelLoaded, want false")
	}
	if f.Scale != alignFlowScale {
		t.Errorf("Scale = %d after modelLoaded, want %d", f.Scale, alignFlowScale)
	}
	if f.Config != core.DefaultAlignConfig {
		t.Errorf("Config = %+v after modelLoaded, want the default", f.Config)
	}
	if !reflect.DeepEqual(f.Rois, scene.MachineRois) {
		t.Errorf("Rois = %+v, want a clone of the machine ROIs", f.Rois)
	}
	// The clone must not alias the scene.
	f.Rois[0].Name = "changed"
	if scene.MachineRois[0].Name == "changed" {
		t.Error("modelLoaded did not clone the machine ROIs")
	}
}

func TestAlignFlowArmNeedsTwoRois(t *testing.T) {
	a := newFlowTestApp()
	f := a.alignFlow()
	f.modelLoaded(&Scene{MachineRois: testRois(1)})
	rev := f.Rev
	if f.Arm() {
		t.Error("Arm with one ROI = true, want false")
	}
	if f.Armed || f.Rev != rev {
		t.Errorf("Arm with one ROI changed the flow: Armed=%v Rev=%d", f.Armed, f.Rev)
	}

	f.SetRois(testRois(2))
	rev = f.Rev
	if !f.Arm() {
		t.Fatal("Arm with two ROIs = false, want true")
	}
	if !f.Armed || f.Rev != rev+1 {
		t.Errorf("Arm: Armed=%v Rev=%d, want Armed=true Rev=%d", f.Armed, f.Rev, rev+1)
	}
}

func TestAlignFlowArmNeedsScene(t *testing.T) {
	a := newFlowTestApp()
	f := a.alignFlow()
	f.Rois = testRois(2)
	rev := f.Rev
	if f.Arm() {
		t.Error("Arm without a scene = true, want false")
	}
	if f.Armed || f.Rev != rev {
		t.Errorf("Arm without a scene changed the flow: Armed=%v Rev=%d", f.Armed, f.Rev)
	}
}

func TestAlignFlowDisarmRules(t *testing.T) {
	a := newFlowTestApp()
	f := a.alignFlow()
	f.modelLoaded(&Scene{MachineRois: testRois(2)})

	// Disarm when already disarmed changes nothing.
	rev := f.Rev
	f.Disarm()
	if f.Armed || f.Rev != rev {
		t.Errorf("Disarm when disarmed changed the flow: Armed=%v Rev=%d", f.Armed, f.Rev)
	}

	if !f.Arm() {
		t.Fatal("Arm = false, want true")
	}
	rev = f.Rev
	f.Disarm()
	if f.Armed || f.Rev != rev+1 {
		t.Errorf("Disarm when armed: Armed=%v Rev=%d, want Armed=false Rev=%d", f.Armed, f.Rev, rev+1)
	}
}

func TestAlignFlowSettersAlwaysDisarm(t *testing.T) {
	a := newFlowTestApp()
	f := a.alignFlow()
	scene := &Scene{MachineRois: testRois(2)}
	f.modelLoaded(scene)

	armAndCheck := func(name string, apply func()) {
		t.Helper()
		if !f.Arm() {
			t.Fatalf("%s: Arm = false, want true", name)
		}
		rev := f.Rev
		apply()
		if f.Armed {
			t.Errorf("%s: Armed = true after the setter, want false", name)
		}
		if f.Rev != rev+1 {
			t.Errorf("%s: Rev = %d after the setter, want %d", name, f.Rev, rev+1)
		}
	}

	armAndCheck("SetRois", func() { f.SetRois(testRois(3)) })
	if len(f.Rois) != 3 {
		t.Errorf("SetRois: len(Rois) = %d, want 3", len(f.Rois))
	}

	armAndCheck("ResetRois", func() { f.ResetRois() })
	if !reflect.DeepEqual(f.Rois, scene.MachineRois) {
		t.Errorf("ResetRois: Rois = %+v, want the machine ROIs", f.Rois)
	}

	armAndCheck("SetConfig", func() { f.SetConfig(core.AlignConfig{SearchRadius: 42}) })
	if f.Config.SearchRadius != 42 {
		t.Errorf("SetConfig: SearchRadius = %v, want 42", f.Config.SearchRadius)
	}

	armAndCheck("SetScale", func() { f.SetScale(3) })
	if f.Scale != 3 {
		t.Errorf("SetScale: Scale = %d, want 3", f.Scale)
	}

	// Clamp 1..4.
	f.SetScale(0)
	if f.Scale != 1 {
		t.Errorf("SetScale(0): Scale = %d, want 1", f.Scale)
	}
	f.SetScale(9)
	if f.Scale != 4 {
		t.Errorf("SetScale(9): Scale = %d, want 4", f.Scale)
	}
}

func TestAlignFlowSetRoisClones(t *testing.T) {
	a := newFlowTestApp()
	f := a.alignFlow()
	in := testRois(2)
	f.SetRois(in)
	in[0].Name = "changed"
	if f.Rois[0].Name == "changed" {
		t.Error("SetRois did not clone the slice")
	}
}

func TestAlignFlowSnapshotIsDeepCopy(t *testing.T) {
	a := newFlowTestApp()
	f := a.alignFlow()
	f.modelLoaded(&Scene{MachineRois: testRois(2)})
	f.Arm()
	snap := f.Snapshot()
	if snap.Rev != f.Rev || snap.Armed != f.Armed || snap.Scale != f.Scale {
		t.Errorf("Snapshot = %+v, flow Rev=%d Armed=%v Scale=%d", snap, f.Rev, f.Armed, f.Scale)
	}
	if !reflect.DeepEqual(snap.Rois, f.Rois) {
		t.Errorf("Snapshot.Rois = %+v, want %+v", snap.Rois, f.Rois)
	}
	snap.Rois[0].Name = "changed"
	if f.Rois[0].Name == "changed" {
		t.Error("Snapshot aliases the flow's ROIs")
	}
	f.SetRois(testRois(4))
	if len(snap.Rois) != 2 {
		t.Error("the flow's SetRois changed an earlier snapshot")
	}
}

func TestAlignFlowRequired(t *testing.T) {
	teach := newFlowTestApp()
	if !teach.alignFlow().Required() {
		t.Error("Required in the teach profile = false, want true")
	}
	studio := &App{profile: ProfileStudio}
	studio.align = newAlignFlow(studio)
	if studio.alignFlow().Required() {
		t.Error("Required in the studio profile = true, want false")
	}
	var nilApp *App
	if nilApp.alignFlow() != nil {
		t.Error("alignFlow on a nil app returned a flow")
	}
}

func TestAlignFlowNilReceiver(t *testing.T) {
	var f *AlignFlow
	if snap := f.Snapshot(); !reflect.DeepEqual(snap, AlignSnapshot{}) {
		t.Errorf("nil Snapshot = %+v, want the zero value", snap)
	}
	if f.Required() {
		t.Error("nil Required = true, want false")
	}
	f.Disarm()
	if f.Arm() {
		t.Error("nil Arm = true, want false")
	}
	f.SetRois(testRois(2))
	f.ResetRois()
	f.SetConfig(core.AlignConfig{})
	f.SetScale(3)
	f.modelLoaded(nil)
}

func TestAlignStateKey(t *testing.T) {
	cases := []struct {
		state alignState
		want  string
	}{
		{alignOff, "off"},
		{alignWorking, "working"},
		{alignDone, "done"},
		{alignFailed, "failed"},
	}
	for _, c := range cases {
		if got := c.state.key(); got != c.want {
			t.Errorf("%d.key() = %q, want %q", c.state, got, c.want)
		}
	}
}

func TestFacadeAlignState(t *testing.T) {
	a := newFlowTestApp()
	f := a.alignFlow()
	f.modelLoaded(&Scene{MachineRois: testRois(2)})
	p := &DefectPanel{a: a}

	if got := p.AlignState(); got != alignOff {
		t.Errorf("disarmed: AlignState = %d, want alignOff", got)
	}
	if got := p.AlignedImage(); got != nil {
		t.Errorf("disarmed: AlignedImage = %v, want nil", got)
	}

	if !f.Arm() {
		t.Fatal("Arm = false, want true")
	}
	if got := p.AlignState(); got != alignWorking {
		t.Errorf("armed without an outcome: AlignState = %d, want alignWorking", got)
	}
	if !p.InspectBlocked() {
		t.Error("armed without an outcome: InspectBlocked = false, want true")
	}

	// An outcome of an older revision is stale.
	p.alignOut = &AlignOutcome{Rev: f.Rev - 1, Usable: true}
	if got := p.AlignState(); got != alignWorking {
		t.Errorf("stale outcome: AlignState = %d, want alignWorking", got)
	}

	img := &core.GrayImage{Width: 1, Height: 1, Data: []uint8{0}}
	p.alignOut = &AlignOutcome{Rev: f.Rev, Usable: true, Image: img}
	p.aligned = img
	if got := p.AlignState(); got != alignDone {
		t.Errorf("usable outcome: AlignState = %d, want alignDone", got)
	}
	if got := p.AlignedImage(); got != img {
		t.Errorf("usable outcome: AlignedImage = %v, want the aligned image", got)
	}
	if p.InspectBlocked() {
		t.Error("align done: InspectBlocked = true, want false")
	}

	p.alignOut = &AlignOutcome{Rev: f.Rev, Usable: false, Err: "boom"}
	p.aligned = nil
	if got := p.AlignState(); got != alignFailed {
		t.Errorf("unusable outcome: AlignState = %d, want alignFailed", got)
	}
	if !p.InspectBlocked() {
		t.Error("align failed: InspectBlocked = false, want true")
	}
	if got := p.AlignedImage(); got != nil {
		t.Errorf("align failed: AlignedImage = %v, want nil", got)
	}
}

func TestFacadeNilSafe(t *testing.T) {
	var p *DefectPanel
	if got := p.Raw(); got != nil {
		t.Errorf("Raw = %v, want nil", got)
	}
	if got := p.AlignOutcome(); got != nil {
		t.Errorf("AlignOutcome = %v, want nil", got)
	}
	if got := p.AlignState(); got != alignOff {
		t.Errorf("AlignState = %d, want alignOff", got)
	}
	if got := p.AlignedImage(); got != nil {
		t.Errorf("AlignedImage = %v, want nil", got)
	}
	if p.InspectBlocked() {
		t.Error("InspectBlocked = true, want false")
	}
	if got := p.Report(); got != nil {
		t.Errorf("Report = %v, want nil", got)
	}
	if got := p.Channel(); got != core.ChRed {
		t.Errorf("Channel = %v, want ChRed", got)
	}
	p.SetChannel(core.ChGreen)
	p.RunInspect()
	p.RequestCapture()
}

func TestDefectPanelOfNilWithoutPanel(t *testing.T) {
	a := newFlowTestApp()
	if got := defectPanelOf(a); got != nil {
		t.Errorf("defectPanelOf = %v, want nil", got)
	}
	if got := defectPanelOf(nil); got != nil {
		t.Errorf("defectPanelOf(nil) = %v, want nil", got)
	}
}
