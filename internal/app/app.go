package app

import (
	"fmt"
	"runtime"
	"sort"
	"time"

	"afvi/teach-imgui/internal/access"
	"afvi/teach-imgui/internal/config"
	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/host"
	"afvi/teach-imgui/internal/ui"
)

// Panel is one mode tab (light / align / defect). All methods run on the UI thread;
// heavy work goes through App.Jobs and touches panel state only in the apply closure.
type Panel interface {
	ID() string
	Title() string
	// Activate is called when the tab becomes visible (and after a model load if visible);
	// it must put the panel's current image into the viewer.
	Activate()
	Deactivate()
	// ModelLoaded resets panel state for App.Scene (called for every panel, active or not).
	ModelLoaded()
	// LightChanged is called after App.Light, App.Display or App.ShowMaterial changed.
	LightChanged()
	DrawLeft()
	DrawRight()
	Overlay(o *Overlay)
	// Pointer receives left-button drags in image coordinates; return true on PointerDown
	// to own the drag (otherwise the viewer pans).
	Pointer(ev PointerEvent) bool
	// Hover returns the status-bar readout for image pixel (x, y), or "".
	Hover(x, y int) string
	// Selftest seeds demo content for automated screenshots (-selftest).
	Selftest()
}

// DialogPanel is implemented by panels that draw their own floating windows (for example
// the defect tab's detection-parameter dialog). DrawDialogs runs once per frame, after the
// main layout, for the active panel only while a model is loaded.
type DialogPanel interface{ DrawDialogs() }

type panelCtor struct {
	order    int
	make     func(*App) Panel
	profiles []Profile
	roles    []access.Role
}

var panelCtors []panelCtor

// registerPanel is called from each panel file's init(); order sets the tab position.
// It registers an ADMIN-only page. profiles restricts the panel to those profiles; an
// empty list allows every profile.
func registerPanel(order int, make func(*App) Panel, profiles ...Profile) {
	registerPanelFor([]access.Role{access.RoleAdmin}, order, make, profiles...)
}

// registerPanelFor registers a page visible to the listed roles only. An empty role list
// means no role at all (the page is never shown).
func registerPanelFor(roles []access.Role, order int, make func(*App) Panel, profiles ...Profile) {
	panelCtors = append(panelCtors, panelCtor{order: order, make: make, profiles: profiles, roles: roles})
}

// allows reports whether the panel is part of profile p.
func (c panelCtor) allows(p Profile) bool {
	if len(c.profiles) == 0 {
		return true
	}
	for _, prof := range c.profiles {
		if prof == p {
			return true
		}
	}
	return false
}

// visibleTo reports whether the page is reachable with role r.
func (c panelCtor) visibleTo(r access.Role) bool {
	for _, role := range c.roles {
		if role == r {
			return true
		}
	}
	return false
}

// DisplayChannel selects which rendered camera channel the light view shows.
type DisplayChannel int

const (
	DisplayRed DisplayChannel = iota
	DisplayGreen
	DisplayBlue
	DisplayRGB
)

// App owns the shared state: loaded scene, light setting, viewer, job runner and panels.
type App struct {
	Cfg        *config.Config
	CfgPath    string
	machineIdx int
	profile    Profile

	RefRoot  string
	Models   []ModelInfo
	modelIdx int
	Scene    *Scene
	loading  bool
	loadErr  string // raw detail; empty when there is no error
	errKey   string // i18n key rendered with {msg} = loadErr, so it follows language switches

	Light        core.LightSetting
	Display      DisplayChannel
	ShowMaterial bool

	Viewer *Viewer
	Jobs   *Jobs

	panels    []Panel     // the pages visible to the current role, in tab order
	allPanels []Panel     // every page constructed for the profile, any role, in tab order
	allCtors  []panelCtor // the registration of each page in allPanels, same order
	tab       int
	role      access.Role

	loginOpen   bool   // the login dialog is (or is about to be) open
	loginUser   string // user field of the login dialog
	loginPass   string // password field of the login dialog
	loginErr    string // i18n key of the last login error ("" = none)
	loginLocked bool   // the studio before its login: no page is visible and the dialog cannot be cancelled

	theme, appliedTheme string

	// focusView hides the left/right panels so the centre viewer fills the window.
	focusView bool

	hint      string
	quit      bool // the user chose Exit: the front end closes the window
	subNote   string
	elapsedMs float64
	start     time.Time
	frame     int
	heapMB    float64
	timings   map[string][]float64

	selftest  *Selftest
	exitAfter time.Duration

	cfgDlg     configDialog
	aboutOpen  bool
	aboutShown bool

	live *liveLink // the live link to the other program (nil = off)

	align  *AlignFlow   // taught alignment of the loaded model (teach program only, nil in the studio)
	golden *goldenBoard // golden board of the loaded model: the inspection area (teach program only, nil in the studio)
}

// NewApp builds the app state and panels; the window/backend is attached by main. The
// active machine comes from cfg and its models are scanned immediately (main decides
// which model to open first).
func NewApp(cfg *config.Config, cfgPath string, theme string, profile Profile) *App {
	a := &App{
		Cfg:      cfg,
		CfgPath:  cfgPath,
		modelIdx: -1,
		profile:  profile,
		Light:    core.DefaultCaptureLight(),
		Viewer:   NewViewer(),
		theme:    theme,
		start:    time.Now(),
		timings:  map[string][]float64{},
	}
	a.Jobs = NewJobs(func(err error) { a.SetHint(T("common.error", "msg", err.Error())) })
	// Both programs start with the light a real capture is taken with: red, green and blue
	// all lit, so every channel has contrast (alignment templates, inspection parameters).
	if profile == ProfileTeach {
		a.align = newAlignFlow(a)
		a.golden = newGoldenBoard(a)
	}
	sort.SliceStable(panelCtors, func(i, j int) bool { return panelCtors[i].order < panelCtors[j].order })
	for _, c := range panelCtors {
		if !c.allows(profile) {
			continue
		}
		a.allPanels = append(a.allPanels, c.make(a))
		a.allCtors = append(a.allCtors, c)
	}
	// Tests and the self-test expect the full interface; runProfile decides the start role.
	a.role = access.RoleAdmin
	a.rebuildPanels()
	a.machineIdx = cfg.ActiveIndex()
	m := a.CurrentMachine()
	a.RefRoot = m.Name
	if m.Address != "" {
		a.RefRoot = m.Name + " (" + m.Address + ")"
	}
	a.Models = ScanMachine(cfg, m)
	if len(a.Models) == 0 {
		a.loadErr, a.errKey = m.Name, "msg.noModels"
	}
	return a
}

// CurrentMachine returns the active machine (zero value when the config is empty).
func (a *App) CurrentMachine() config.Machine {
	if a.Cfg == nil || a.machineIdx < 0 || a.machineIdx >= len(a.Cfg.Machines) {
		return config.Machine{}
	}
	return a.Cfg.Machines[a.machineIdx]
}

// SelectMachine switches to machine i: it rescans models, clears the scene and viewer
// and auto-loads the first model. It is ignored while a model is loading or for any role
// other than the admin.
func (a *App) SelectMachine(i int) {
	if a.role != access.RoleAdmin {
		return
	}
	if a.Cfg == nil || i < 0 || i >= len(a.Cfg.Machines) || a.loading {
		return
	}
	a.machineIdx = i
	m := a.Cfg.Machines[i]
	a.Cfg.Active = m.Name
	a.RefRoot = m.Name
	if m.Address != "" {
		a.RefRoot = m.Name + " (" + m.Address + ")"
	}
	a.Models = ScanMachine(a.Cfg, m)
	if p := a.ActivePanel(); p != nil && a.Scene != nil {
		p.Deactivate()
	}
	a.Scene = nil
	a.modelIdx = -1
	a.Viewer.Clear()
	a.loadErr, a.errKey = "", ""
	if len(a.Models) > 0 {
		a.LoadModel(0)
	} else {
		a.loadErr, a.errKey = m.Name, "msg.noModels"
	}
}

// RescanModels re-scans the active machine (used after the configuration changed).
func (a *App) RescanModels() { a.SelectMachine(a.machineIdx) }

// Role returns the access level of the current user.
func (a *App) Role() access.Role { return a.role }

// rebuildPanels filters allPanels down to the pages visible to the current role.
func (a *App) rebuildPanels() {
	a.panels = make([]Panel, 0, len(a.allPanels))
	for i, p := range a.allPanels {
		if a.allCtors[i].visibleTo(a.role) {
			a.panels = append(a.panels, p)
		}
	}
}

// SetRole switches the access level: the active page is deactivated, the visible pages
// are rebuilt, the tab returns to the first page and admin-only dialogs are closed when
// the admin leaves.
func (a *App) SetRole(r access.Role) {
	if p := a.ActivePanel(); p != nil {
		p.Deactivate()
	}
	a.role = r
	a.rebuildPanels()
	a.tab = 0
	a.hint = ""
	if r != access.RoleAdmin {
		a.cfgDlg.needOpen = false
		a.cfgDlg.draft = nil
		a.cfgDlg.errMsg = ""
	}
	if r != access.RoleNone {
		a.loginLocked = false
	}
	if a.Scene != nil && len(a.panels) > 0 {
		a.panels[0].Activate()
	}
}

// Lock drops to no role at all: no page is visible and the login dialog cannot be
// cancelled (the studio before its login).
func (a *App) Lock() {
	a.SetRole(access.RoleNone)
	a.loginLocked = true
	a.loginOpen = true
}

// OpenLogin shows the login dialog.
func (a *App) OpenLogin() {
	a.loginOpen = true
	a.loginErr = ""
}

// TryLogin checks a login with internal/access. The studio accepts only an admin login.
// On success the role changes and the dialog closes; on failure the role is kept and
// loginErr is set to the "login.failed" key.
func (a *App) TryLogin(user, password string) bool {
	role, ok := access.Authenticate(user, password)
	if ok && a.profile == ProfileStudio && role != access.RoleAdmin {
		ok = false
	}
	if !ok {
		a.loginErr = "login.failed"
		a.loginPass = ""
		return false
	}
	a.SetRole(role)
	a.loginOpen = false
	a.loginLocked = false
	a.loginErr = ""
	a.loginUser = ""
	a.loginPass = ""
	return true
}

// Logout leaves the admin: the teaching program goes back to the operator pages, the
// studio locks itself again.
func (a *App) Logout() {
	if a.profile == ProfileStudio {
		a.Lock()
		return
	}
	a.SetRole(access.RoleOperator)
}

// panelByID returns the panel with the given id among all constructed pages (any role),
// or nil.
func (a *App) panelByID(id string) Panel {
	for _, p := range a.allPanels {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// PageVisible reports whether the page with that id is visible to the current role.
func (a *App) PageVisible(id string) bool {
	for _, p := range a.panels {
		if p.ID() == id {
			return true
		}
	}
	return false
}

// ShowPage switches to the visible page with that id. It returns false and changes
// nothing when the page is not visible to the current role.
func (a *App) ShowPage(id string) bool {
	for i, p := range a.panels {
		if p.ID() == id {
			a.switchTab(i)
			return a.ActivePanel() != nil && a.ActivePanel().ID() == id
		}
	}
	return false
}

// ApplyConfig validates c, saves it to CfgPath, installs it and switches to its active
// machine.
func (a *App) ApplyConfig(c *config.Config) error {
	if a.role != access.RoleAdmin {
		return nil
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if a.CfgPath != "" {
		if err := c.Save(a.CfgPath); err != nil {
			return err
		}
	}
	a.Cfg = c
	a.SelectMachine(c.ActiveIndex())
	return nil
}

// SetHint shows a message in the status bar.
func (a *App) SetHint(s string) { a.hint = s }

// SetElapsed shows a computation time in the status bar.
func (a *App) SetElapsed(ms float64) { a.elapsedMs = ms }

// CameraChannel maps the display selection to a render channel (RGB → Red for
// single-channel consumers such as alignment and estimation).
func (a *App) CameraChannel() core.CameraChannel {
	switch a.Display {
	case DisplayGreen:
		return core.ChGreen
	case DisplayBlue:
		return core.ChBlue
	}
	return core.ChRed
}

// NotifyLightChanged tells every panel the shared light setting or display changed.
func (a *App) NotifyLightChanged() {
	for _, p := range a.allPanels {
		p.LightChanged()
	}
}

// ActivePanel returns the visible panel.
func (a *App) ActivePanel() Panel {
	if a.tab < 0 || a.tab >= len(a.panels) {
		return nil
	}
	return a.panels[a.tab]
}

// IsActive reports whether p is the visible panel.
func (a *App) IsActive(p Panel) bool { return a.ActivePanel() == p }

func (a *App) switchTab(i int) {
	if i == a.tab || i < 0 || i >= len(a.panels) {
		return
	}
	if p := a.ActivePanel(); p != nil {
		p.Deactivate()
	}
	a.tab = i
	a.hint = ""
	if a.Scene != nil {
		a.panels[i].Activate()
	}
}

// LoadModel loads models[i] in the background; panels are reset when it completes.
func (a *App) LoadModel(i int) {
	if i < 0 || i >= len(a.Models) || a.loading {
		return
	}
	a.loading = true
	a.loadErr = ""
	info := a.Models[i]
	a.Jobs.Run("load", func() func() {
		scene, err := LoadScene(info)
		return func() {
			a.loading = false
			if err != nil {
				// Keep the previous scene and its model selection; only report the failure.
				a.loadErr, a.errKey = err.Error(), "status.loadFailed"
				return
			}
			a.modelIdx = i
			if p := a.ActivePanel(); p != nil && a.Scene != nil {
				p.Deactivate()
			}
			a.Scene = scene
			a.Viewer.Clear()
			a.align.modelLoaded(scene)
			a.golden.modelLoaded(scene)
			for _, p := range a.allPanels {
				p.ModelLoaded()
			}
			if p := a.ActivePanel(); p != nil {
				p.Activate()
			}
		}
	})
}

// SharedViewer implements host.Program.
func (a *App) SharedViewer() *ui.Viewer { return a.Viewer }

// Quit asks the front end to close the window.
func (a *App) Quit() { a.quit = true }

// Build implements host.Program: it runs one UI frame and returns the slots the front
// end renders (menu, toolbar, tabs, sub, left, right, status, palette, dialogs).
func (a *App) Build(b *ui.Builder) *host.Frame {
	a.frame++
	if a.frame%60 == 0 {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		a.heapMB = float64(ms.HeapAlloc) / (1 << 20)
	}
	for _, t := range a.Jobs.Poll() {
		if a.selftest != nil {
			// Only the self-test report consumes timings; do not grow without bound otherwise.
			a.timings[t.Key] = append(a.timings[t.Key], t.Ms)
		}
		if t.Key != "load" {
			a.elapsedMs = t.Ms
		}
	}
	a.tickPanels()
	if a.appliedTheme != a.theme {
		ApplyTheme(a.theme)
		a.appliedTheme = a.theme
	}
	slots := map[string]*ui.Node{}

	b.Begin()
	a.buildMenu()
	slots["menu"] = b.Finish()

	// keyboard shortcuts that work on every page
	if a.role == access.RoleAdmin && ui.KeyPressed("F11") { // same as the View menu item: admin only
		a.toggleFocus()
	}

	b.Begin()
	a.buildToolbar()
	slots["toolbar"] = b.Finish()

	b.Begin()
	titles := make([]string, len(a.panels))
	for i, p := range a.panels {
		titles[i] = p.Title()
	}
	if len(titles) > 0 {
		if idx := TabStrip("tabs", titles, a.tab); idx >= 0 {
			a.switchTab(idx)
		}
	}
	slots["tabs"] = b.Finish()

	panel := a.ActivePanel()
	ready := a.Scene != nil && panel != nil

	if ready {
		if s, ok := panel.(SubTabPanel); ok {
			if items, active := s.SubTabs(); len(items) > 0 {
				b.Begin()
				if idx := SubTabStrip("subtabs", items, active, s.SubTabNote()); idx >= 0 {
					s.SelectSubTab(idx)
				}
				slots["sub"] = b.Finish()
			}
		}
	}

	// pointer events from the canvas go to the active panel
	for _, p := range a.Viewer.TakePointers() {
		if ev, ok := pointerEvent(p); ok && ready && a.Viewer.HasImage() {
			panel.Pointer(ev)
		}
	}
	a.Viewer.Claim = false
	if ready {
		if c, ok := panel.(interface{ ClaimsDrag() bool }); ok {
			a.Viewer.Claim = c.ClaimsDrag()
		}
	}

	if ready {
		b.Begin()
		panel.DrawLeft()
		slots["left"] = b.Finish()
		b.Begin()
		panel.DrawRight()
		slots["right"] = b.Finish()
		panel.Overlay(a.Viewer.Overlay())
		b.Begin()
		a.buildPalette(panel)
		slots["palette"] = b.Finish()
		a.Viewer.SetHover(func(x, y int) string {
			s := "XY " + fmt.Sprint(x) + ", " + fmt.Sprint(y)
			if g := panel.Hover(x, y); g != "" {
				s += "   " + g
			}
			return s
		})
	} else {
		a.Viewer.Shapes = a.Viewer.Shapes[:0]
		b.Begin()
		ui.Note(a.centerMessage())
		slots["message"] = b.Finish()
	}

	b.Begin()
	a.buildStatus()
	slots["status"] = b.Finish()

	b.Begin()
	if ready {
		if d, ok := panel.(DialogPanel); ok {
			d.DrawDialogs()
		}
	}
	a.buildConfigDialog()
	a.buildAbout()
	a.buildLogin()
	slots["dialogs"] = b.Finish()

	return &host.Frame{
		Title: a.profile.WindowTitle(),
		Slots: slots,
		Meta: map[string]any{
			"profile": string(a.profile), "theme": a.theme, "lang": string(CurrentLang()),
			"fontPx": fontPx, "focusView": a.focusView, "quit": a.quit, "role": int(a.role),
			"ready": ready,
		},
	}
}

func (a *App) centerMessage() string {
	msg := T("status.loading")
	if a.loadErr != "" {
		msg = T(a.errKey, "msg", a.loadErr)
	}
	if a.role == access.RoleNone {
		msg = T("login.locked")
	}
	return msg
}

// buildMenu records the main menu (File / View / Help).
func (a *App) buildMenu() {
	ui.Menu(T("menu.file"), "m-file", func() {
		if a.role == access.RoleAdmin {
			if ui.MenuItem(T("menu.machineConfig"), "menu:config", "", -1, true) {
				a.OpenConfigDialog()
			}
			if ui.MenuItem(T("menu.rescan"), "menu:rescan", "", -1, true) {
				a.RescanModels()
			}
			ui.MenuSeparator()
			if ui.MenuItem(T("menu.logout"), "menu:logout", "", -1, true) {
				a.Logout()
			}
			ui.MenuSeparator()
		} else if a.role == access.RoleOperator {
			if ui.MenuItem(T("menu.login"), "menu:login", "", -1, true) {
				a.OpenLogin()
			}
			ui.MenuSeparator()
		}
		if ui.MenuItem(T("menu.exit"), "menu:exit", "Alt+F4", -1, true) {
			a.Quit()
		}
	})
	if a.role != access.RoleNone {
		ui.Menu(T("menu.view"), "m-view", func() {
			if a.role == access.RoleAdmin {
				c := 0
				if a.focusView {
					c = 1
				}
				if ui.MenuItem(T("menu.focus"), "menu:focus", "F11", c, true) {
					a.toggleFocus()
				}
				ui.MenuSeparator()
				for _, name := range []string{ThemeLight, ThemeDark} {
					c := 0
					if a.theme == name {
						c = 1
					}
					if ui.MenuItem(T("menu.theme")+": "+T("theme."+name), "menu:theme-"+name, "", c, true) {
						a.theme = name
					}
				}
			}
			for _, l := range Langs {
				c := 0
				if l.Code == CurrentLang() {
					c = 1
				}
				if ui.MenuItem(T("menu.lang")+": "+l.Label, "menu:lang-"+string(l.Code), "", c, true) {
					SetLang(l.Code)
				}
			}
			ui.MenuSeparator()
			if ui.MenuItem(T("menu.fit"), "menu:fit", "", -1, true) {
				a.Viewer.Fit()
			}
		})
	}
	ui.Menu(T("menu.help"), "m-help", func() {
		if ui.MenuItem(T("menu.about"), "menu:about", "", -1, true) {
			a.aboutOpen = true
		}
	})
}

// buildToolbar records the machine / model selectors and the login button.
func (a *App) buildToolbar() {
	if a.role == access.RoleAdmin {
		ui.Text(T("top.machine"))
		ui.SameLine()
		var names []string
		sel := -1
		if a.Cfg != nil {
			for i, m := range a.Cfg.Machines {
				label := m.Name
				if m.Address != "" {
					label = m.Name + "  (" + m.Address + ")"
				}
				names = append(names, label)
				if i == a.machineIdx {
					sel = i
				}
			}
		}
		if Combo("", "machine", &sel, names) {
			a.SelectMachine(sel)
		}
		ui.SameLine()
		if Button("…", "cfg-open") {
			a.OpenConfigDialog()
		}
		ui.SameLine()
	}
	ui.Text(T("top.model"))
	ui.SameLine()
	var ids []string
	for _, m := range a.Models {
		ids = append(ids, m.ID+" · "+m.Version)
	}
	if sel := a.modelIdx; Combo("", "model", &sel, ids) && sel != a.modelIdx {
		a.LoadModel(sel)
	}
	ui.SameLine()
	status := ""
	switch {
	case a.loading:
		status = T("status.loading")
	case a.Scene != nil:
		status = T("status.loaded", "w", a.Scene.W, "h", a.Scene.H, "ms", F(a.Scene.LoadMs, 0))
	}
	ui.Text(status)

	// right end: who is logged in plus the login / logout button
	if a.role != access.RoleNone {
		ui.SameLine()
		roleLabel := T("login.user.operator")
		roleID, roleText := "login-open", T("login.open")
		if a.role == access.RoleAdmin {
			roleLabel, roleID, roleText = T("login.user.admin"), "login-logout", T("login.logout")
		}
		ui.Text(roleLabel).Prop("right", true)
		ui.SameLine()
		if Button(roleText, roleID) {
			if a.role == access.RoleAdmin {
				a.Logout()
			} else {
				a.OpenLogin()
			}
		}
	}
}

// buildStatus records the status bar panes.
func (a *App) buildStatus() *ui.Node {
	elapsed := T("status.elapsedNone")
	if a.elapsedMs > 0 {
		elapsed = T("status.elapsed", "ms", F(a.elapsedMs, 1))
	}
	hint := a.hint
	if a.Jobs.AnyBusy() {
		hint = T("status.busy") + "  " + hint
	}
	zoom := T("status.zoom", "pct", F(float64(a.Viewer.Zoom())*100, 0))
	n := ui.Add(&ui.Node{K: "status", L: hint})
	n.C = []*ui.Node{
		{K: ui.KText, L: zoom, P: map[string]any{"w": 100}},
		{K: ui.KText, L: elapsed, P: map[string]any{"w": 130}},
		{K: ui.KText, L: fmt.Sprintf("heap %.0f MB", a.heapMB), P: map[string]any{"w": 110}},
	}
	return n
}

// buildAbout records the modal About dialog.
func (a *App) buildAbout() {
	if !a.aboutShown {
		if !a.aboutOpen {
			return
		}
		a.aboutShown, a.aboutOpen = true, false
	}
	titleKey, textKey := "about.title", "about.text"
	if a.profile == ProfileStudio {
		titleKey, textKey = "about.title.studio", "about.text.studio"
	}
	closed := ui.Dialog(T(titleKey), "about", 380, func() {
		ui.Text(a.profile.WindowTitle())
		ui.Note(T(textKey))
		ui.Text("Wails + Vue 3 · win32-vue")
		ui.Spacing()
		ui.SameLine()
		if Button(T("cfg.ok"), "about-ok") {
			a.aboutShown = false
		}
	})
	if closed || ui.KeyPressed("Escape") {
		a.aboutShown = false
	}
}
