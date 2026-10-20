package app

import (
	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/inspect"
)

// alignTemplate is the cached design template of the real-capture alignment together with
// the key it was built for.
type alignTemplate struct {
	scene *Scene
	ch    core.CameraChannel
	light core.LightSetting
	scale int
	img   *core.GrayImage
}

// AlignOutcome is the result of aligning the CURRENT capture with the taught ROIs.
type AlignOutcome struct {
	Rev       int // AlignSnapshot.Rev it was computed for
	Usable    bool
	Report    core.AlignReport
	Image     *core.GrayImage // the aligned capture (full resolution) when Usable, else nil
	NoData    []bool          // pixels of Image the capture had no data for (filled by replication); the inspection skips them
	ElapsedMs float64
	Err       string // non-empty when core.AlignCapture returned an error (then Usable is false)
}

// alignState is the alignment state of the detection as the pages show it.
type alignState int

const (
	alignOff     alignState = iota // not armed: nothing taught yet (detection blocked)
	alignWorking                   // armed, the result for this Rev is not there yet
	alignDone                      // armed and the capture was aligned (Usable)
	alignFailed                    // armed but the capture could not be aligned (detection blocked)
)

// key returns the i18n-style suffix of the state ("off" | "working" | "done" | "failed").
func (s alignState) key() string {
	switch s {
	case alignWorking:
		return "working"
	case alignDone:
		return "done"
	case alignFailed:
		return "failed"
	}
	return "off"
}

// defectAlignState is the alignment state of a DefectPanel. DefectPanel embeds it, so the
// pages read the raw capture, the outcome and the aligned image through the facade below.
type defectAlignState struct {
	raw        *core.GrayImage // the capture as acquired, before any alignment (nil before the first run)
	alignOut   *AlignOutcome   // outcome of aligning `raw` (nil = never aligned for this raw)
	aligned    *core.GrayImage // alignOut.Image when alignOut.Usable (then it is the same pointer as DefectPanel.test)
	alignedRev int             // AlignFlow.Rev the last finished run was made with (0 = none)
	tpl        *alignTemplate  // cached design template, see the pipeline rule
}

// defectPanelOf returns the DefectPanel of the app, in either mode (nil if none).
func defectPanelOf(a *App) *DefectPanel {
	if a == nil {
		return nil
	}
	p, _ := a.panelByID("defect").(*DefectPanel)
	return p
}

// Raw returns the capture as acquired, before any alignment.
func (p *DefectPanel) Raw() *core.GrayImage {
	if p == nil {
		return nil
	}
	return p.raw
}

// AlignOutcome returns the outcome of aligning Raw (nil when never aligned).
func (p *DefectPanel) AlignOutcome() *AlignOutcome {
	if p == nil {
		return nil
	}
	return p.alignOut
}

// AlignState reports the alignment state of the panel as described by the contract.
func (p *DefectPanel) AlignState() alignState {
	if p == nil {
		return alignOff
	}
	flow := p.a.alignFlow()
	if flow == nil || !flow.Armed {
		return alignOff
	}
	if p.alignOut == nil || p.alignOut.Rev != flow.Rev {
		return alignWorking
	}
	if p.alignOut.Usable {
		return alignDone
	}
	return alignFailed
}

// AlignedImage returns the aligned capture when the alignment succeeded, else nil.
func (p *DefectPanel) AlignedImage() *core.GrayImage {
	if p == nil || p.AlignState() != alignDone {
		return nil
	}
	return p.aligned
}

// InspectBlocked reports whether the taught alignment is required but not usable, so the
// inspection must not run.
func (p *DefectPanel) InspectBlocked() bool {
	if p == nil {
		return false
	}
	flow := p.a.alignFlow()
	return flow != nil && flow.Required() && p.AlignState() != alignDone
}

// Report returns the last inspection report.
func (p *DefectPanel) Report() *inspect.Report {
	if p == nil {
		return nil
	}
	return p.lastReport
}

// Channel returns the render channel of the panel.
func (p *DefectPanel) Channel() core.CameraChannel {
	if p == nil {
		return core.ChRed
	}
	return p.channel
}

// SetChannel switches the render channel exactly like the channel combo does.
func (p *DefectPanel) SetChannel(ch core.CameraChannel) {
	if p == nil || p.channel == ch {
		return
	}
	p.channel = ch
	p.renderRev++
	p.requestRun()
}

// RunInspect learns (when needed) and inspects the current test image now.
func (p *DefectPanel) RunInspect() {
	if p == nil {
		return
	}
	p.requestInspect()
}

// RequestCapture refreshes the test image (and aligns it when the flow is armed).
func (p *DefectPanel) RequestCapture() {
	if p == nil {
		return
	}
	p.requestRun()
}

// testRegions returns the design material map for inspecting p.test. When p.test is the aligned
// capture, the pixels the alignment had no data for are marked so the inspection ignores them
// instead of judging the replicated border.
func (p *DefectPanel) testRegions(scene *Scene) inspect.Regions {
	reg := inspectRegions(scene)
	if out := p.alignOut; out != nil && out.Usable && p.aligned != nil && p.test == p.aligned {
		reg.Skip = out.NoData
	}
	return reg
}
