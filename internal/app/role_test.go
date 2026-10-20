package app

import (
	"strings"
	"testing"

	"afvi/teach-imgui/internal/access"
)

// registerTestPanel registers a page for the given roles and removes it again after the
// test. panelCtors is re-sorted by NewApp, so the append order does not matter.
func registerTestPanel(t *testing.T, role access.Role, id string) {
	t.Helper()
	before := len(panelCtors)
	registerPanelFor([]access.Role{role}, 50, func(a *App) Panel { return &fakePanel{id: id} }, ProfileTeach)
	t.Cleanup(func() { panelCtors = panelCtors[:before] })
}

// fakePanel is a page with no behaviour, only an id.
type fakePanel struct{ id string }

func (p *fakePanel) ID() string                   { return p.id }
func (p *fakePanel) Title() string                { return p.id }
func (p *fakePanel) Activate()                    {}
func (p *fakePanel) Deactivate()                  {}
func (p *fakePanel) ModelLoaded()                 {}
func (p *fakePanel) LightChanged()                {}
func (p *fakePanel) DrawLeft()                    {}
func (p *fakePanel) DrawRight()                   {}
func (p *fakePanel) Overlay(o *Overlay)           {}
func (p *fakePanel) Pointer(ev PointerEvent) bool { return false }
func (p *fakePanel) Hover(x, y int) string        { return "" }
func (p *fakePanel) Selftest()                    {}

func TestDefaultRoleIsAdmin(t *testing.T) {
	registerTestPanel(t, access.RoleOperator, "op-test")
	a := newTestApp(t, ProfileTeach)
	if a.Role() != access.RoleAdmin {
		t.Errorf("NewApp role = %v, want admin", a.Role())
	}
	// The admin sees every page except the ones registered for the operator only.
	var hidden []string
	for _, p := range a.allPanels {
		if !a.PageVisible(p.ID()) {
			hidden = append(hidden, p.ID())
		}
	}
	if got := strings.Join(hidden, ","); got != "op-align,op-inspect,op-test" {
		t.Errorf("pages hidden from the admin = %q, want op-align,op-inspect,op-test", got)
	}
	if len(a.panels)+len(hidden) != len(a.allPanels) {
		t.Errorf("panels %d + hidden %d != allPanels %d", len(a.panels), len(hidden), len(a.allPanels))
	}
	if a.PageVisible("op-test") {
		t.Error("an operator-only page is visible to the admin")
	}
	if a.panelByID("op-test") == nil {
		t.Error("panelByID does not find a hidden page")
	}
}

func TestSetRoleFiltersPages(t *testing.T) {
	registerTestPanel(t, access.RoleOperator, "op-test")
	a := newTestApp(t, ProfileTeach)
	a.SetRole(access.RoleOperator)
	if a.PageVisible("align") || a.PageVisible("defect") {
		t.Errorf("admin pages still visible: %v", a.panels)
	}
	if !a.PageVisible("op-test") {
		t.Errorf("operator page not visible: %v", a.panels)
	}
	// switchTab cannot reach a hidden page and ShowPage refuses it.
	a.switchTab(len(a.panels) - 1)
	for _, p := range a.panels {
		if p.ID() == "align" || p.ID() == "defect" {
			t.Errorf("switchTab reached hidden page %q", p.ID())
		}
	}
	if a.ShowPage("defect") {
		t.Error("ShowPage(defect) succeeded for the operator")
	}
	if !a.ShowPage("op-test") || a.ActivePanel() == nil || a.ActivePanel().ID() != "op-test" {
		t.Error("ShowPage(op-test) failed for the operator")
	}
	a.SetRole(access.RoleAdmin)
	if !a.PageVisible("align") || a.PageVisible("op-test") {
		t.Errorf("after SetRole(admin) panels = %v", a.panels)
	}
}

func TestTryLogin(t *testing.T) {
	registerTestPanel(t, access.RoleOperator, "op-test")
	a := newTestApp(t, ProfileTeach)
	a.SetRole(access.RoleOperator)
	if !a.TryLogin("admin", "1") {
		t.Fatal("TryLogin(admin, 1) failed")
	}
	if a.Role() != access.RoleAdmin {
		t.Errorf("role after a good login = %v, want admin", a.Role())
	}
	if a.loginOpen {
		t.Error("loginOpen still true after a good login")
	}
	a.SetRole(access.RoleOperator)
	if a.TryLogin("admin", "wrong") {
		t.Error("TryLogin with a wrong password succeeded")
	}
	if a.Role() != access.RoleOperator {
		t.Errorf("role after a failed login = %v, want operator", a.Role())
	}
	if a.loginErr != "login.failed" {
		t.Errorf("loginErr = %q, want login.failed", a.loginErr)
	}
}

func TestStudioRefusesOperatorLogin(t *testing.T) {
	a := newTestApp(t, ProfileStudio)
	if a.TryLogin("operator", "") {
		t.Error("the studio accepted an operator login")
	}
	if a.Role() == access.RoleOperator {
		t.Error("the studio role became operator")
	}
	if a.TryLogin("admin", "1") && a.Role() != access.RoleAdmin {
		t.Errorf("studio admin login left role %v", a.Role())
	}
}

func TestLockAndLogout(t *testing.T) {
	registerTestPanel(t, access.RoleOperator, "op-test")
	a := newTestApp(t, ProfileTeach)
	a.Lock()
	if len(a.panels) != 0 {
		t.Errorf("Lock left %d pages visible", len(a.panels))
	}
	if a.ActivePanel() != nil {
		t.Error("ActivePanel is not nil while locked")
	}
	if !a.loginOpen || !a.loginLocked {
		t.Error("Lock did not open the locked login dialog")
	}
	if !a.TryLogin("admin", "1") {
		t.Fatal("TryLogin while locked failed")
	}
	a.Logout()
	if a.Role() != access.RoleOperator {
		t.Errorf("Logout in the teaching program left role %v, want operator", a.Role())
	}
}

func TestAdminOperationsAreNoOpsForOperator(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	load := a.loading
	scene := a.Scene
	a.SetRole(access.RoleOperator)
	a.SelectMachine(0)
	if a.loading != load || a.Scene != scene {
		t.Error("SelectMachine changed state for the operator")
	}
	if err := a.ApplyConfig(cloneConfig(a.Cfg)); err != nil {
		t.Errorf("ApplyConfig for the operator returned %v, want nil", err)
	}
	path := a.CfgPath
	a.CfgPath = ""
	if err := a.ApplyConfig(cloneConfig(a.Cfg)); err != nil || a.CfgPath != "" {
		t.Error("ApplyConfig wrote the configuration for the operator")
	}
	a.CfgPath = path
}
