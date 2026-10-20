package app

import (
	"afvi/teach-imgui/internal/core"
)

// alignFlowScale is the working-resolution divisor of the real-capture alignment.
const alignFlowScale = 2

// AlignFlow is the taught alignment of the loaded model: the ROIs drawn on the imagined
// (design) picture and whether the user confirmed them. One per App in the teach program
// (nil in the studio): field `align *AlignFlow` of App.
// It holds NO image: the template that a capture is matched against is built by the
// detection pipeline (see the pipeline rule in the job contract).
// All fields are touched on the UI thread only.
type AlignFlow struct {
	a *App

	scene  *Scene
	Rois   []core.AlignRoi // full-resolution design coordinates
	Scale  int             // working-resolution divisor (1..4), default alignFlowScale
	Config core.AlignConfig
	Armed  bool
	Rev    int
}

// newAlignFlow builds an empty flow for a. The caller calls modelLoaded once a scene is
// available; until then the flow is disarmed with no ROIs.
func newAlignFlow(a *App) *AlignFlow {
	return &AlignFlow{
		a:      a,
		Scale:  alignFlowScale,
		Config: core.DefaultAlignConfig,
	}
}

// modelLoaded resets the flow for a new model: the ROIs become a clone of the machine
// ROIs of scene, the config returns to the default, the flow is disarmed and Rev grows.
func (f *AlignFlow) modelLoaded(scene *Scene) {
	if f == nil {
		return
	}
	f.scene = scene
	if scene != nil {
		f.Rois = cloneRois(scene.MachineRois)
	} else {
		f.Rois = nil
	}
	f.Config = core.DefaultAlignConfig
	f.Scale = alignFlowScale
	f.Armed = false
	f.Rev++
}

// Arm is called when the user pressed Confirm: it needs at least two ROIs and a loaded
// scene, otherwise it does nothing and returns false. A successful arm grows Rev.
func (f *AlignFlow) Arm() bool {
	if f == nil || f.scene == nil || len(f.Rois) < 2 {
		return false
	}
	f.Armed = true
	f.Rev++
	return true
}

// Disarm drops the taught alignment. It does nothing (and grows no Rev) when the flow is
// already disarmed.
func (f *AlignFlow) Disarm() {
	if f == nil || !f.Armed {
		return
	}
	f.Armed = false
	f.Rev++
}

// SetRois replaces the ROIs (the slice is cloned) and disarms.
func (f *AlignFlow) SetRois(rois []core.AlignRoi) {
	if f == nil {
		return
	}
	f.Rois = cloneRois(rois)
	f.Disarm()
}

// ResetRois restores a clone of the machine ROIs of the loaded scene and disarms.
func (f *AlignFlow) ResetRois() {
	if f == nil {
		return
	}
	if f.scene != nil {
		f.Rois = cloneRois(f.scene.MachineRois)
	} else {
		f.Rois = nil
	}
	f.Disarm()
}

// SetConfig replaces Config and disarms.
func (f *AlignFlow) SetConfig(cfg core.AlignConfig) {
	if f == nil {
		return
	}
	f.Config = cfg
	f.Disarm()
}

// SetScale sets the working-resolution divisor, clamped to 1..4, and disarms.
func (f *AlignFlow) SetScale(scale int) {
	if f == nil {
		return
	}
	if scale < 1 {
		scale = 1
	} else if scale > 4 {
		scale = 4
	}
	f.Scale = scale
	f.Disarm()
}

// AlignSnapshot is an immutable copy of the flow for job goroutines.
type AlignSnapshot struct {
	Rev    int
	Armed  bool
	Rois   []core.AlignRoi // cloned
	Scale  int
	Config core.AlignConfig
}

// Snapshot returns an immutable copy of the flow.
func (f *AlignFlow) Snapshot() AlignSnapshot {
	if f == nil {
		return AlignSnapshot{}
	}
	return AlignSnapshot{
		Rev:    f.Rev,
		Armed:  f.Armed,
		Rois:   cloneRois(f.Rois),
		Scale:  f.Scale,
		Config: f.Config,
	}
}

// Required reports whether detection needs a usable alignment: true in the teach program.
func (f *AlignFlow) Required() bool {
	if f == nil || f.a == nil {
		return false
	}
	return f.a.profile == ProfileTeach
}

// alignFlow returns a.align (nil in the studio).
func (a *App) alignFlow() *AlignFlow {
	if a == nil {
		return nil
	}
	return a.align
}
