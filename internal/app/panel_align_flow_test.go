package app

import (
	"testing"

	"afvi/teach-imgui/internal/access"
)

// newFlowPanel returns the page with the given id (nil if absent).
func newFlowPanel(a *App, id string) *AlignFlowPanel {
	p, _ := a.panelByID(id).(*AlignFlowPanel)
	return p
}

func TestAlignFlowPagesRegistered(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	roi := a.panelByID("roi")
	if roi == nil {
		t.Fatal("admin panel \"roi\" is missing")
	}
	op := a.panelByID("op-align")
	if op == nil {
		t.Fatal("operator panel \"op-align\" is missing")
	}
	if hr, ok := roi.(interface{ HidePalette() bool }); !ok || !hr.HidePalette() {
		t.Error("roi page must hide the palette")
	}
	if ho, ok := op.(interface{ HidePalette() bool }); !ok || !ho.HidePalette() {
		t.Error("op-align page must hide the palette")
	}
}

func TestAlignFlowPageRoles(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	if !a.PageVisible("roi") || a.PageVisible("op-align") {
		t.Fatalf("admin visibility wrong: roi=%v op-align=%v", a.PageVisible("roi"), a.PageVisible("op-align"))
	}
	a.SetRole(access.RoleOperator)
	if a.PageVisible("roi") {
		t.Error("roi visible to operator")
	}
	if !a.PageVisible("op-align") {
		t.Error("op-align not visible to operator")
	}
	if a.ShowPage("roi") {
		t.Error("ShowPage(roi) succeeded for the operator")
	}
}

func TestAlignFlowOperatorDrawAndDelete(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	a.SetRole(access.RoleOperator)
	op := newFlowPanel(a, "op-align")
	if op == nil {
		t.Fatal("no op-align panel")
	}
	op.Activate()
	// The machine ROIs are loaded into the flow on modelLoaded; grab a starting count.
	if a.alignFlow() == nil {
		t.Fatal("alignFlow is nil in the teach program")
	}
	before := len(a.alignFlow().Rois)
	scene := a.Scene
	x0 := float64(scene.W) * 0.2
	y0 := float64(scene.H) * 0.2
	x1 := x0 + 40
	y1 := y0 + 40
	if !op.Pointer(PointerEvent{Kind: PointerDown, X: x0, Y: y0}) {
		t.Fatal("PointerDown not owned")
	}
	op.Pointer(PointerEvent{Kind: PointerMove, X: x1, Y: y1})
	op.Pointer(PointerEvent{Kind: PointerUp, X: x1, Y: y1})
	after := len(a.alignFlow().Rois)
	if after != before+1 {
		t.Fatalf("ROIs after drawing = %d, want %d", after, before+1)
	}
	if a.alignFlow().Armed {
		t.Error("drawing a ROI must disarm the flow")
	}
	// Select the newest one and delete it.
	op.editor.sel = after - 1
	op.deleteSelected()
	if got := len(a.alignFlow().Rois); got != before {
		t.Errorf("ROIs after delete = %d, want %d", got, before)
	}
}

func TestAlignFlowConfirmNeedsTwo(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	op := newFlowPanel(a, "op-align")
	if op == nil {
		t.Fatal("no op-align panel")
	}
	a.alignFlow().SetRois(nil)
	if a.alignFlow().Armed {
		t.Fatal("flow armed with no ROIs")
	}
	op.confirm()
	if op.msg != "needTwo" {
		t.Errorf("msg = %q, want needTwo", op.msg)
	}
	if a.alignFlow().Armed {
		t.Error("flow armed with fewer than two ROIs")
	}
}

func TestAlignFlowResetBoxes(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	op := newFlowPanel(a, "op-align")
	if op == nil {
		t.Fatal("no op-align panel")
	}
	a.alignFlow().SetRois(nil)
	op.resetBoxes()
	if len(a.alignFlow().Rois) != len(a.Scene.MachineRois) {
		t.Errorf("ROIs after reset = %d, want %d", len(a.alignFlow().Rois), len(a.Scene.MachineRois))
	}
}
