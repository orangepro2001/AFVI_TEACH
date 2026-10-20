package app

import (
	"testing"

	"afvi/teach-imgui/internal/access"
)

// TestOpInspectRegistered 确认操作员检测页存在，并且只对操作员可见。
func TestOpInspectRegistered(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	if a.panelByID("op-inspect") == nil {
		t.Fatal("panelByID(\"op-inspect\") = nil, want the operator page")
	}
	a.SetRole(access.RoleOperator)
	if !a.PageVisible("op-inspect") {
		t.Errorf("op-inspect is not visible to the operator: %v", a.panels)
	}
	if got := a.ActivePanel(); got != nil && got.ID() == "op-inspect" {
		t.Error("the operator page should not be the first visible page")
	}
}

// TestOpInspectAdminHidden 确认管理员看不到操作员检测页，也无法切换过去。
func TestOpInspectAdminHidden(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	a.SetRole(access.RoleAdmin)
	if a.PageVisible("op-inspect") {
		t.Errorf("op-inspect is visible to the admin: %v", a.panels)
	}
	if a.ShowPage("op-inspect") {
		t.Error("ShowPage(op-inspect) succeeded for the admin")
	}
	if p := a.panelByID("op-inspect"); p == nil {
		t.Fatal("panelByID does not find the hidden operator page")
	}
}

// TestOpInspectPaletteAndPointer 确认页面隐藏调色板，并且不拥有任何拖动。
func TestOpInspectPaletteAndPointer(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	p, ok := a.panelByID("op-inspect").(*OpInspectPanel)
	if !ok || p == nil {
		t.Fatal("panelByID(\"op-inspect\") is not an *OpInspectPanel")
	}
	if !p.HidePalette() {
		t.Error("HidePalette() = false, want true")
	}
	for _, kind := range []PointerKind{PointerDown, PointerMove, PointerUp, PointerRightClick} {
		if p.Pointer(PointerEvent{Kind: kind, X: 1, Y: 1}) {
			t.Errorf("Pointer(%v) = true, want false", kind)
		}
	}
	if got, want := p.ID(), "op-inspect"; got != want {
		t.Errorf("ID() = %q, want %q", got, want)
	}
	if got, want := p.Title(), T("tab.opInspect"); got != want {
		t.Errorf("Title() = %q, want %q", got, want)
	}
}

// TestOpInspectBlockedWithoutAlignment 确认教学程序刚加载时检测被对位流程挡住。
func TestOpInspectBlockedWithoutAlignment(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	if flow := a.alignFlow(); flow == nil || flow.Armed {
		t.Fatal("fresh teach app: the align flow is nil or already armed")
	}
	if !dp.InspectBlocked() {
		t.Error("InspectBlocked() = false on a fresh teach app, want true")
	}
	if dp.AlignState() != alignOff {
		t.Errorf("AlignState() = %v, want alignOff", dp.AlignState())
	}
}
