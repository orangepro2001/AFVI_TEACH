package app

import (
	"fmt"
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/access"
	"afvi/teach-imgui/internal/core"
)

// insertBefore returns steps with extra inserted in front of the step called name (appended
// when there is no such step).
func insertBefore(steps []selftestStep, name string, extra ...selftestStep) []selftestStep {
	out := make([]selftestStep, 0, len(steps)+len(extra))
	inserted := false
	for _, st := range steps {
		if !inserted && st.name == name {
			out = append(out, extra...)
			inserted = true
		}
		out = append(out, st)
	}
	if !inserted {
		out = append(out, extra...)
	}
	return out
}

// gotoPage switches to the visible page with the id and lets it seed its self-test state.
func gotoPage(a *App, id string) bool {
	for i, p := range a.panels {
		if p.ID() == id {
			a.switchTab(i)
			p.Selftest()
			return true
		}
	}
	return false
}

// pageIDs returns the ids of the visible pages joined by ",".
func pageIDs(a *App) string {
	ids := make([]string, 0, len(a.panels))
	for _, p := range a.panels {
		ids = append(ids, p.ID())
	}
	return strings.Join(ids, ",")
}

// legacyStep detaches the forced alignment and restores the nominal geometry, so the original
// steps run on the same code paths as before the flow existed.
func (s *Selftest) legacyStep() selftestStep {
	return selftestStep{
		name: "teach-legacy",
		enter: func(a *App) {
			s.flow = s.a.align
			s.a.align = nil
			// The golden board is detached too: the older steps inspect against the design layers.
			s.golden = s.a.golden
			s.a.golden = nil
			if dp := defectPanelOf(s.a); dp != nil {
				dp.warp = core.IdentityWarp
				dp.renderRev++
				dp.golden, dp.test = nil, nil
			}
		},
	}
}

// flowSteps exercises the forced alignment on the admin pages: the design ROIs, arming the flow,
// inspecting the aligned capture and blocking detection after an edit.
func (s *Selftest) flowSteps() []selftestStep {
	var n0 int
	var newRoi core.AlignRoi
	return []selftestStep{
		{
			name: "flow-attach",
			enter: func(a *App) {
				a.theme = ThemeDark
				SetLang(LangZh)
				a.align = s.flow
				a.golden = s.golden
				a.align.ResetRois()
				dp := defectPanelOf(a)
				dp.warp = core.DefaultCaptureWarp()
				dp.renderRev++
				dp.golden, dp.test = nil, nil
				dp.lastReport = nil
				gotoPage(a, "roi")
				dp.RequestCapture()
			},
			shot: "native-roi-dark-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				fl := s.a.alignFlow()
				if fl == nil {
					return nil, fmt.Errorf("flow-attach: no taught alignment on the app")
				}
				if !fl.Required() {
					return nil, fmt.Errorf("flow-attach: the alignment is not required")
				}
				if fl.Armed {
					return nil, fmt.Errorf("flow-attach: the flow is armed right after the reset")
				}
				if len(fl.Rois) < 2 {
					return nil, fmt.Errorf("flow-attach: %d ROIs, want at least 2", len(fl.Rois))
				}
				dp := defectPanelOf(s.a)
				if dp.Raw() == nil {
					return nil, fmt.Errorf("flow-attach: no raw capture")
				}
				if !dp.InspectBlocked() {
					return nil, fmt.Errorf("flow-attach: detection is not blocked while the flow is disarmed")
				}
				if st := dp.AlignState(); st != alignOff {
					return nil, fmt.Errorf("flow-attach: align state %s, want off", st.key())
				}
				if dp.Report() != nil {
					return nil, fmt.Errorf("flow-attach: a report exists although detection is blocked")
				}
				if !s.a.Viewer.HasImage() {
					return nil, fmt.Errorf("flow-attach: the viewer shows no image")
				}
				light := s.a.Light.Clone()
				nominal := core.RenderChannel(s.a.Scene.Light, &light, dp.Channel())
				rawDiff, _ := core.MeanAbsDiff(dp.Raw(), nominal, nil)
				if rawDiff <= 0.5 {
					return nil, fmt.Errorf("flow-attach: raw capture differs from the nominal render by %.3f, want the default misalignment", rawDiff)
				}
				return map[string]any{"rois": len(fl.Rois), "rawDiff": rawDiff}, nil
			},
		},
		{
			name: "flow-draw",
			enter: func(a *App) {
				n0 = len(a.align.Rois)
				gotoPage(a, "roi")
			},
			input: func(s *Selftest) []viewerInput {
				c := s.canvasCenter()
				return dragInput(btnLeft, imgui.NewVec2(c.X-30, c.Y-30), imgui.NewVec2(60, 50), 4)
			},
			check: func(s *Selftest) (map[string]any, error) {
				fl := s.a.align
				if len(fl.Rois) != n0+1 {
					return nil, fmt.Errorf("flow-draw: %d ROIs, want %d", len(fl.Rois), n0+1)
				}
				if fl.Armed {
					return nil, fmt.Errorf("flow-draw: drawing a ROI armed the flow")
				}
				newRoi = fl.Rois[n0]
				fl.SetRois(fl.Rois[:n0])
				defectPanelOf(s.a).RequestCapture()
				return map[string]any{"newRoi": fmt.Sprintf("%.0f,%.0f %.0f×%.0f", newRoi.X, newRoi.Y, newRoi.W, newRoi.H)}, nil
			},
		},
		{
			name: "flow-confirm",
			enter: func(a *App) {
				fp := s.panel("roi").(*AlignFlowPanel)
				fp.confirm()
				fp.view = flowViewAligned
				defectPanelOf(a).RequestCapture()
			},
			shot: "native-roi-aligned-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				fl := s.a.align
				if !fl.Armed {
					return nil, fmt.Errorf("flow-confirm: the flow is not armed after Confirm")
				}
				dp := defectPanelOf(s.a)
				if st := dp.AlignState(); st != alignDone {
					reason := ""
					if fp, ok := s.panel("roi").(*AlignFlowPanel); ok {
						reason = fp.reason()
					}
					if out := dp.AlignOutcome(); out != nil && out.Err != "" {
						reason = out.Err
					}
					return nil, fmt.Errorf("flow-confirm: align state %s, want done (%s)", st.key(), reason)
				}
				out := dp.AlignOutcome()
				if out == nil || !out.Usable {
					return nil, fmt.Errorf("flow-confirm: no usable alignment outcome")
				}
				aligned := dp.AlignedImage()
				if aligned == nil {
					return nil, fmt.Errorf("flow-confirm: no aligned image")
				}
				if dp.test != aligned {
					return nil, fmt.Errorf("flow-confirm: the test image is not the aligned image")
				}
				if dp.Report() == nil {
					return nil, fmt.Errorf("flow-confirm: detection did not run on the aligned capture")
				}
				if dp.InspectBlocked() {
					return nil, fmt.Errorf("flow-confirm: detection still blocked after a good alignment")
				}
				light := s.a.Light.Clone()
				nominal := core.RenderChannel(s.a.Scene.Light, &light, dp.Channel())
				rawDiff, _ := core.MeanAbsDiff(dp.Raw(), nominal, nil)
				alignedDiff, _ := core.MeanAbsDiff(aligned, nominal, nil)
				if !(alignedDiff < rawDiff) {
					return nil, fmt.Errorf("flow-confirm: aligned diff %.3f is not below the raw diff %.3f", alignedDiff, rawDiff)
				}
				return map[string]any{
					"rawDiff":     rawDiff,
					"alignedDiff": alignedDiff,
					"elapsedMs":   out.ElapsedMs,
					"defects":     len(dp.Report().Defects),
				}, nil
			},
		},
		{
			name: "flow-channel-green",
			enter: func(a *App) {
				dp := defectPanelOf(a)
				dp.SetChannel(core.ChGreen)
				dp.RequestCapture()
			},
			shot: "native-roi-aligned-green-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				dp := defectPanelOf(s.a)
				out := dp.AlignOutcome()
				if st := dp.AlignState(); st != alignDone {
					return nil, fmt.Errorf("flow-channel-green: align state %s on channel green, want done: %s", st.key(), explainAlign(out, len(s.a.align.Rois)))
				}
				if out == nil || !out.Usable {
					return nil, fmt.Errorf("flow-channel-green: no usable alignment outcome on channel green: %s", explainAlign(out, len(s.a.align.Rois)))
				}
				if out.Channel != core.ChGreen {
					return nil, fmt.Errorf("flow-channel-green: outcome channel %v, want green: %s", out.Channel, explainAlign(out, len(s.a.align.Rois)))
				}
				if out.TplFlat {
					return nil, fmt.Errorf("flow-channel-green: the design template of channel green is flat: %s", explainAlign(out, len(s.a.align.Rois)))
				}
				if out.RawFlat {
					return nil, fmt.Errorf("flow-channel-green: the capture of channel green is flat: %s", explainAlign(out, len(s.a.align.Rois)))
				}
				if dp.Report() == nil {
					return nil, fmt.Errorf("flow-channel-green: no report on channel green: %s", explainAlign(out, len(s.a.align.Rois)))
				}
				located := 0
				for _, m := range out.Report.Matches {
					if m.Located {
						located++
					}
				}
				return map[string]any{"channel": "green", "located": located, "elapsedMs": out.ElapsedMs}, nil
			},
		},
		{
			name: "flow-channel-blue",
			enter: func(a *App) {
				dp := defectPanelOf(a)
				dp.SetChannel(core.ChBlue)
				dp.RequestCapture()
			},
			shot: "native-roi-aligned-blue-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				dp := defectPanelOf(s.a)
				out := dp.AlignOutcome()
				if st := dp.AlignState(); st != alignDone {
					return nil, fmt.Errorf("flow-channel-blue: align state %s on channel blue, want done: %s", st.key(), explainAlign(out, len(s.a.align.Rois)))
				}
				if out == nil || !out.Usable {
					return nil, fmt.Errorf("flow-channel-blue: no usable alignment outcome on channel blue: %s", explainAlign(out, len(s.a.align.Rois)))
				}
				if out.Channel != core.ChBlue {
					return nil, fmt.Errorf("flow-channel-blue: outcome channel %v, want blue: %s", out.Channel, explainAlign(out, len(s.a.align.Rois)))
				}
				if out.TplFlat {
					return nil, fmt.Errorf("flow-channel-blue: the design template of channel blue is flat: %s", explainAlign(out, len(s.a.align.Rois)))
				}
				if out.RawFlat {
					return nil, fmt.Errorf("flow-channel-blue: the capture of channel blue is flat: %s", explainAlign(out, len(s.a.align.Rois)))
				}
				if dp.Report() == nil {
					return nil, fmt.Errorf("flow-channel-blue: no report on channel blue: %s", explainAlign(out, len(s.a.align.Rois)))
				}
				located := 0
				for _, m := range out.Report.Matches {
					if m.Located {
						located++
					}
				}
				dp.SetChannel(core.ChRed)
				dp.RequestCapture()
				return map[string]any{"channel": "blue", "located": located, "elapsedMs": out.ElapsedMs}, nil
			},
		},
		{
			name: "flow-edit",
			enter: func(a *App) {
				a.align.SetRois(a.align.Rois[1:])
				defectPanelOf(a).RequestCapture()
			},
			check: func(s *Selftest) (map[string]any, error) {
				fl := s.a.align
				if fl.Armed {
					return nil, fmt.Errorf("flow-edit: editing the ROIs did not disarm the flow")
				}
				dp := defectPanelOf(s.a)
				if st := dp.AlignState(); st != alignOff {
					return nil, fmt.Errorf("flow-edit: align state %s, want off", st.key())
				}
				if !dp.InspectBlocked() {
					return nil, fmt.Errorf("flow-edit: detection is not blocked after the edit")
				}
				if dp.Report() != nil {
					return nil, fmt.Errorf("flow-edit: a report exists after the edit")
				}
				if dp.AlignedImage() != nil {
					return nil, fmt.Errorf("flow-edit: an aligned image is still offered")
				}
				fl.SetRois(fl.Rois[:1])
				if fl.Arm() {
					return nil, fmt.Errorf("flow-edit: Arm accepted a single ROI")
				}
				if fl.Armed {
					return nil, fmt.Errorf("flow-edit: a refused Arm armed the flow")
				}
				fl.ResetRois()
				defectPanelOf(s.a).RequestCapture()
				return nil, nil
			},
		},
	}
}

// roleSteps exercises the roles and the strict page management of the teaching program.
func (s *Selftest) roleSteps() []selftestStep {
	var idsOperator string
	var pageShown bool
	var okBad, okUser, okAdmin bool
	var roleAfterBad access.Role
	return []selftestStep{
		{
			name: "roles-operator",
			enter: func(a *App) {
				a.theme = ThemeDark
				SetLang(LangZh)
				a.SetRole(access.RoleOperator)
			},
			shot: "native-operator-align-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				idsOperator = pageIDs(s.a)
				if idsOperator != "op-align,golden,op-inspect" {
					return nil, fmt.Errorf("roles-operator: visible pages %q, want op-align,golden,op-inspect", idsOperator)
				}
				if s.a.Role() != access.RoleOperator {
					return nil, fmt.Errorf("roles-operator: role %v, want operator", s.a.Role())
				}
				before := ""
				if p := s.a.ActivePanel(); p != nil {
					before = p.ID()
				}
				for _, id := range []string{"roi", "align", "defect", "light"} {
					if s.a.PageVisible(id) {
						return nil, fmt.Errorf("roles-operator: page %q is visible to the operator", id)
					}
					if s.a.ShowPage(id) {
						return nil, fmt.Errorf("roles-operator: ShowPage(%q) succeeded for the operator", id)
					}
					if p := s.a.ActivePanel(); p != nil && p.ID() != before {
						return nil, fmt.Errorf("roles-operator: a refused ShowPage(%q) switched to %q", id, p.ID())
					}
				}
				if s.a.panelByID("roi") == nil {
					return nil, fmt.Errorf("roles-operator: the roi page was not constructed")
				}
				if s.panel("roi") != nil {
					return nil, fmt.Errorf("roles-operator: the roi page is reachable through the visible pages")
				}
				p := s.a.ActivePanel()
				if p == nil || p.ID() != "op-align" {
					got := ""
					if p != nil {
						got = p.ID()
					}
					return nil, fmt.Errorf("roles-operator: active page %q, want op-align", got)
				}
				if !defectPanelOf(s.a).InspectBlocked() {
					return nil, fmt.Errorf("roles-operator: detection is not blocked for the operator")
				}
				return map[string]any{"pages": idsOperator, "role": s.a.Role().String()}, nil
			},
		},
		{
			name: "roles-confirm",
			enter: func(a *App) {
				op := a.panelByID("op-align").(*AlignFlowPanel)
				op.confirm()
			},
			shot: "native-operator-aligned-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				if !s.a.align.Armed {
					return nil, fmt.Errorf("roles-confirm: the operator Confirm did not arm the flow")
				}
				dp := defectPanelOf(s.a)
				if st := dp.AlignState(); st != alignDone {
					return nil, fmt.Errorf("roles-confirm: align state %s, want done", st.key())
				}
				if dp.InspectBlocked() {
					return nil, fmt.Errorf("roles-confirm: detection is still blocked after the Confirm")
				}
				return nil, nil
			},
		},
		s.goldenOperatorStep(),
		{
			name: "roles-flat-channel",
			enter: func(a *App) {
				a.Light = core.DefaultLightSetting()
				dp := defectPanelOf(a)
				dp.SetChannel(core.ChGreen)
				dp.RequestCapture()
			},
			shot: "native-operator-flat-channel-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				defer func() {
					s.a.Light = core.DefaultCaptureLight()
					defectPanelOf(s.a).SetChannel(core.ChRed)
					defectPanelOf(s.a).RequestCapture()
				}()
				dp := defectPanelOf(s.a)
				out := dp.AlignOutcome()
				if st := dp.AlignState(); st != alignFailed {
					return nil, fmt.Errorf("roles-flat-channel: align state %s, want failed: %s", st.key(), explainAlign(out, len(s.a.align.Rois)))
				}
				if out == nil {
					return nil, fmt.Errorf("roles-flat-channel: no alignment outcome")
				}
				if out.Channel != core.ChGreen {
					return nil, fmt.Errorf("roles-flat-channel: outcome channel %v, want green", out.Channel)
				}
				if !out.TplFlat {
					return nil, fmt.Errorf("roles-flat-channel: the design template of channel green is not flat: %s", explainAlign(out, len(s.a.align.Rois)))
				}
				for _, m := range out.Report.Matches {
					if m.Located {
						return nil, fmt.Errorf("roles-flat-channel: ROI %q was located although the channel has no light", m.Name)
					}
				}
				if !dp.InspectBlocked() {
					return nil, fmt.Errorf("roles-flat-channel: detection is not blocked after the failed alignment: %s", explainAlign(out, len(s.a.align.Rois)))
				}
				fp, _ := s.a.panelByID("op-align").(*AlignFlowPanel)
				if fp == nil {
					return nil, fmt.Errorf("roles-flat-channel: the op-align page is not an alignment panel")
				}
				want := explainAlign(out, len(s.a.align.Rois))
				text := fp.reason()
				if text != want {
					return nil, fmt.Errorf("roles-flat-channel: reason %q, want %q", text, want)
				}
				if !strings.Contains(text, T("alignx.ch.green")) {
					return nil, fmt.Errorf("roles-flat-channel: reason %q does not name the green channel", text)
				}
				if !strings.Contains(text, T("alignx.hintLight")) {
					return nil, fmt.Errorf("roles-flat-channel: reason %q does not carry the light hint", text)
				}
				if strings.Contains(text, "alignx.") {
					return nil, fmt.Errorf("roles-flat-channel: reason %q contains a raw translation key", text)
				}
				return map[string]any{"channel": "green", "boxes": len(s.a.align.Rois), "text": text}, nil
			},
		},
		{
			name: "roles-inspect",
			enter: func(a *App) {
				pageShown = a.ShowPage("op-inspect")
				defectPanelOf(a).RunInspect()
			},
			shot: "native-operator-inspect-zh.png",
			check: func(s *Selftest) (map[string]any, error) {
				if !pageShown {
					return nil, fmt.Errorf("roles-inspect: ShowPage(op-inspect) was refused")
				}
				dp := defectPanelOf(s.a)
				if dp.Report() == nil {
					return nil, fmt.Errorf("roles-inspect: no report")
				}
				p := s.a.ActivePanel()
				if p == nil || p.ID() != "op-inspect" {
					got := ""
					if p != nil {
						got = p.ID()
					}
					return nil, fmt.Errorf("roles-inspect: active page %q, want op-inspect", got)
				}
				return map[string]any{"defects": len(dp.Report().Defects)}, nil
			},
		},
		{
			name: "roles-inspect-en",
			enter: func(a *App) {
				a.theme = ThemeLight
				SetLang(LangEn)
			},
			shot: "native-operator-inspect-light-en.png",
			check: func(s *Selftest) (map[string]any, error) {
				if defectPanelOf(s.a).Report() == nil {
					return nil, fmt.Errorf("roles-inspect-en: no report")
				}
				return nil, nil
			},
		},
		{
			name: "roles-login",
			enter: func(a *App) {
				okBad = a.TryLogin("admin", "wrong")
				roleAfterBad = a.Role()
				okUser = a.TryLogin("nobody", "1")
				okAdmin = a.TryLogin("admin", "1")
				a.theme = ThemeDark
				SetLang(LangZh)
			},
			check: func(s *Selftest) (map[string]any, error) {
				if okBad {
					return nil, fmt.Errorf("roles-login: a wrong admin password was accepted")
				}
				if okUser {
					return nil, fmt.Errorf("roles-login: an unknown user was accepted")
				}
				if roleAfterBad != access.RoleOperator {
					return nil, fmt.Errorf("roles-login: role after the failed login %v, want operator", roleAfterBad)
				}
				if !okAdmin {
					return nil, fmt.Errorf("roles-login: the admin login was refused")
				}
				if s.a.Role() != access.RoleAdmin {
					return nil, fmt.Errorf("roles-login: role %v, want admin", s.a.Role())
				}
				if ids := pageIDs(s.a); ids != "roi,align,golden,defect" {
					return nil, fmt.Errorf("roles-login: admin pages %q, want roi,align,golden,defect", ids)
				}
				if s.a.PageVisible("op-align") || s.a.PageVisible("op-inspect") {
					return nil, fmt.Errorf("roles-login: an operator page is visible to the admin")
				}
				if s.a.ShowPage("op-align") {
					return nil, fmt.Errorf("roles-login: ShowPage(op-align) succeeded for the admin")
				}
				return map[string]any{"pages": pageIDs(s.a), "role": s.a.Role().String()}, nil
			},
		},
	}
}

// studioRoleSteps exercises the lock and the login of the camera studio.
func (s *Selftest) studioRoleSteps() []selftestStep {
	var idsLocked int
	var okOperator, okWrong, lockedStill, okAdmin bool
	return []selftestStep{
		{
			name: "studio-login",
			enter: func(a *App) {
				a.Lock()
				idsLocked = len(a.panels)
				okOperator = a.TryLogin("operator", "")
				okWrong = a.TryLogin("admin", "x")
				lockedStill = a.Role() == access.RoleNone
				okAdmin = a.TryLogin("admin", "1")
			},
			check: func(s *Selftest) (map[string]any, error) {
				if idsLocked != 0 {
					return nil, fmt.Errorf("studio-login: %d pages visible while locked", idsLocked)
				}
				if okOperator {
					return nil, fmt.Errorf("studio-login: the studio accepted the operator login")
				}
				if okWrong {
					return nil, fmt.Errorf("studio-login: a wrong admin password was accepted")
				}
				if !lockedStill {
					return nil, fmt.Errorf("studio-login: a failed login left the studio unlocked")
				}
				if !okAdmin {
					return nil, fmt.Errorf("studio-login: the admin login was refused")
				}
				if s.a.Role() != access.RoleAdmin {
					return nil, fmt.Errorf("studio-login: role %v, want admin", s.a.Role())
				}
				if len(s.a.panels) == 0 {
					return nil, fmt.Errorf("studio-login: no pages after the login")
				}
				if s.a.ActivePanel() == nil {
					return nil, fmt.Errorf("studio-login: no active page after the login")
				}
				return map[string]any{"pages": len(s.a.panels), "role": s.a.Role().String()}, nil
			},
		},
	}
}
