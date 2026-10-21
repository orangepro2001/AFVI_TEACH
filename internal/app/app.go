package app

import (
	"fmt"
	"runtime"
	"sort"
	"time"

	"github.com/AllenDang/cimgui-go/backend"
	"github.com/AllenDang/cimgui-go/backend/glfwbackend"
	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/access"
	"afvi/teach-imgui/internal/config"
	"afvi/teach-imgui/internal/core"
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
	be backend.Backend[glfwbackend.GLFWWindowFlags]

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
	elapsedMs float64
	start     time.Time
	frame     int
	heapMB    float64
	timings   map[string][]float64

	selftest  *Selftest
	exitAfter time.Duration

	cfgDlg    configDialog
	aboutOpen bool

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

// Frame builds one UI frame.
func (a *App) Frame() {
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
	a.drawMenuBar()
	if a.role == access.RoleAdmin && imgui.IsKeyPressedBoolV(imgui.KeyF11, false) { // same as the View menu item: admin only
		a.toggleFocus()
	}

	vp := imgui.MainViewport()
	pos, size := vp.Pos(), vp.Size()
	menuH := imgui.FrameHeight()
	toolH, statusH := sc(34), sc(24)
	leftW, rightW := sc(340), sc(380)
	if a.focusView {
		leftW, rightW = 0, 0
	}
	tabH := imgui.FrameHeight() + 4

	y := pos.Y + menuH
	toolbarY := y
	y += toolH
	tabsTop := y + 4
	pageTop := tabsTop + 2 + tabH
	pageBottom := pos.Y + size.Y - statusH - 2

	// the page tabs only need tabsTop and tabH: the active panel below decides the rectangles
	place(pos.X, tabsTop, size.X, tabH+8)
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(0, 0))
	imgui.BeginV("##tabs", nil, fixedWindowFlags()|imgui.WindowFlagsNoBackground|imgui.WindowFlagsNoScrollbar)
	imgui.PopStyleVar()
	imgui.SetCursorPos(imgui.NewVec2(10, 2))
	titles := make([]string, len(a.panels))
	for i, p := range a.panels {
		titles[i] = p.Title()
	}
	if idx := TabStrip("tabs", titles, a.tab); idx >= 0 {
		a.switchTab(idx)
	}
	imgui.End()

	panel := a.ActivePanel()
	ready := a.Scene != nil && panel != nil

	// second-level strip: the page reserves a band under the tabs when it provides entries
	var st SubTabPanel
	var subItems []SubTabItem
	subActive := 0
	subNote := ""
	if ready {
		if s, ok := panel.(SubTabPanel); ok {
			if items, active := s.SubTabs(); len(items) > 0 {
				st, subItems, subActive, subNote = s, items, active, s.SubTabNote()
			}
		}
	}
	subH := float32(0)
	if st != nil {
		subH = imgui.FrameHeight() + sc(10)
	}

	pageMin := imgui.NewVec2(pos.X+4, pageTop)
	pageMax := imgui.NewVec2(pos.X+size.X-4, pageBottom)
	innerMin := imgui.NewVec2(pageMin.X+4, pageMin.Y+4+subH)
	innerMax := imgui.NewVec2(pageMax.X-4, pageMax.Y-4)
	innerH := innerMax.Y - innerMin.Y

	leftMin := imgui.NewVec2(innerMin.X, innerMin.Y)
	leftMax := imgui.NewVec2(innerMin.X+leftW, innerMax.Y)
	rightMin := imgui.NewVec2(innerMax.X-rightW, innerMin.Y)
	rightMax := imgui.NewVec2(innerMax.X, innerMax.Y)
	viewMin := imgui.NewVec2(innerMin.X+leftW, innerMin.Y)
	viewMax := imgui.NewVec2(innerMax.X-rightW, innerMax.Y)
	if leftW > 0 {
		viewMin.X += 4
	}
	if rightW > 0 {
		viewMax.X -= 4
	}

	bg := imgui.BackgroundDrawList()
	FillRect(bg, pos, imgui.NewVec2(pos.X+size.X, pos.Y+size.Y), CurrentPalette().Face)
	DrawEdge(bg, pageMin, pageMax, EdgeWindow)
	if leftW > 0 {
		DrawEdge(bg, imgui.NewVec2(leftMin.X-2, leftMin.Y-2), imgui.NewVec2(leftMax.X+2, leftMax.Y+2), EdgeSunken)
	}
	if rightW > 0 {
		DrawEdge(bg, imgui.NewVec2(rightMin.X-2, rightMin.Y-2), imgui.NewVec2(rightMax.X+2, rightMax.Y+2), EdgeSunken)
	}
	DrawEdge(bg, imgui.NewVec2(viewMin.X-2, viewMin.Y-2), imgui.NewVec2(viewMax.X+2, viewMax.Y+2), EdgeSunken)

	a.drawToolbar(pos.X, toolbarY, size.X, toolH)

	if st != nil {
		place(pageMin.X+8, pageMin.Y+4, pageMax.X-pageMin.X-16, subH)
		imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(0, 0))
		imgui.BeginV("##subtabs", nil, fixedWindowFlags()|imgui.WindowFlagsNoBackground|imgui.WindowFlagsNoScrollbar)
		imgui.PopStyleVar()
		imgui.SetCursorPos(imgui.NewVec2(sc(4), (subH-(imgui.FrameHeight()+2))/2))
		if idx := SubTabStrip("subtabs", subItems, subActive, subNote); idx >= 0 {
			st.SelectSubTab(idx)
		}
		imgui.End()
	}

	if leftW > 0 {
		place(leftMin.X, leftMin.Y, leftW, innerH)
		imgui.BeginV("##left", nil, fixedWindowFlags())
		if ready {
			panel.DrawLeft()
		}
		imgui.End()
	}

	if rightW > 0 {
		place(rightMin.X, rightMin.Y, rightW, innerH)
		imgui.BeginV("##right", nil, fixedWindowFlags())
		if ready {
			panel.DrawRight()
		}
		imgui.End()
	}

	place(viewMin.X, viewMin.Y, viewMax.X-viewMin.X, innerH)
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(0, 0))
	imgui.BeginV("##viewer", nil, fixedWindowFlags()|imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoScrollWithMouse)
	imgui.PopStyleVar()
	if ready {
		a.Viewer.Draw(panel.Overlay, panel.Pointer)
		a.drawPalette(panel)
	} else {
		a.drawCenterMessage()
	}
	imgui.End()

	a.drawStatus(pos.X, pos.Y+size.Y-statusH, size.X, statusH)
	if ready {
		if d, ok := panel.(DialogPanel); ok {
			d.DrawDialogs()
		}
	}
	a.drawConfigDialog()
	a.drawAbout()
	a.drawLogin()
}

// drawMenuBar draws the Win32 main menu bar (File / View / Help).
func (a *App) drawMenuBar() {
	if !imgui.BeginMainMenuBar() {
		return
	}
	if imgui.BeginMenu(T("menu.file")) {
		HotText(T("menu.file"), "")
		if a.role == access.RoleAdmin {
			if imgui.MenuItemBoolV(T("menu.machineConfig"), "", false, true) {
				a.OpenConfigDialog()
			}
			HotText(T("menu.machineConfig"), "")
			if imgui.MenuItemBoolV(T("menu.rescan"), "", false, true) {
				a.RescanModels()
			}
			HotText(T("menu.rescan"), "")
			imgui.Separator()
			if imgui.MenuItemBoolV(T("menu.logout"), "", false, true) {
				a.Logout()
			}
			HotText(T("menu.logout"), "")
			imgui.Separator()
		} else if a.role == access.RoleOperator {
			if imgui.MenuItemBoolV(T("menu.login"), "", false, true) {
				a.OpenLogin()
			}
			HotText(T("menu.login"), "")
			imgui.Separator()
		}
		if imgui.MenuItemBoolV(T("menu.exit"), "Alt+F4", false, true) {
			a.be.SetShouldClose(true)
		}
		HotText(T("menu.exit"), "Alt+F4")
		imgui.EndMenu()
	}
	if a.role != access.RoleNone && imgui.BeginMenu(T("menu.view")) {
		HotText(T("menu.view"), "")
		if a.role == access.RoleAdmin {
			if imgui.MenuItemBoolV(T("menu.focus"), "F11", a.focusView, true) {
				a.toggleFocus()
			}
			HotText(T("menu.focus"), "F11")
			imgui.Separator()
			if imgui.BeginMenu(T("menu.theme")) {
				HotText(T("menu.theme"), "")
				for _, name := range []string{ThemeLight, ThemeDark} {
					if imgui.MenuItemBoolV(T("theme."+name), "", a.theme == name, true) {
						a.theme = name
					}
					HotText(T("theme."+name), "")
				}
				imgui.EndMenu()
			}
		}
		if imgui.BeginMenu(T("menu.lang")) {
			HotText(T("menu.lang"), "")
			for _, l := range Langs {
				if imgui.MenuItemBoolV(l.Label, "", l.Code == CurrentLang(), true) {
					SetLang(l.Code)
				}
				HotText(l.Label, "")
			}
			imgui.EndMenu()
		}
		imgui.Separator()
		if imgui.MenuItemBoolV(T("menu.fit"), "", false, true) {
			a.Viewer.Fit()
		}
		HotText(T("menu.fit"), "")
		imgui.EndMenu()
	}
	if imgui.BeginMenu(T("menu.help")) {
		HotText(T("menu.help"), "")
		if imgui.MenuItemBoolV(T("menu.about"), "", false, true) {
			a.aboutOpen = true
		}
		HotText(T("menu.about"), "")
		imgui.EndMenu()
	}
	imgui.EndMainMenuBar()
}

// drawAbout draws the modal About dialog.
func (a *App) drawAbout() {
	if a.aboutOpen {
		imgui.OpenPopupStr("about")
		a.aboutOpen = false
	}
	vp := imgui.MainViewport()
	center := imgui.NewVec2(vp.Pos().X+vp.Size().X/2, vp.Pos().Y+vp.Size().Y/2)
	imgui.SetNextWindowPosV(center, imgui.CondAlways, imgui.NewVec2(0.5, 0.5))
	imgui.SetNextWindowSizeV(imgui.NewVec2(sc(360), sc(170)), imgui.CondAlways)
	flags := imgui.WindowFlagsNoTitleBar | imgui.WindowFlagsNoResize | imgui.WindowFlagsNoMove |
		imgui.WindowFlagsNoSavedSettings | imgui.WindowFlagsNoScrollbar
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(0, 0))
	open := imgui.BeginPopupModalV("about", nil, flags)
	imgui.PopStyleVar()
	if !open {
		return
	}
	wp, ws := imgui.WindowPos(), imgui.WindowSize()
	dl := imgui.WindowDrawList()
	DrawEdge(dl, wp, imgui.NewVec2(wp.X+ws.X, wp.Y+ws.Y), EdgeWindow)
	capMin := imgui.NewVec2(wp.X+3, wp.Y+3)
	capMax := imgui.NewVec2(wp.X+ws.X-3, wp.Y+3+sc(20))
	titleKey, textKey := "about.title", "about.text"
	if a.profile == ProfileStudio {
		titleKey, textKey = "about.title.studio", "about.text.studio"
	}
	Caption(dl, capMin, capMax, T(titleKey), true)

	imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+sc(16), wp.Y+sc(40)))
	imgui.TextUnformatted(a.profile.WindowTitle())
	imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+sc(16), wp.Y+sc(62)))
	imgui.TextUnformatted(T(textKey))
	imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+sc(16), wp.Y+sc(84)))
	imgui.TextUnformatted("Dear ImGui " + imgui.Version())

	okLabel := T("cfg.ok")
	okW := ButtonWidth(okLabel)
	if okW < sc(75) {
		okW = sc(75)
	}
	imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+ws.X-sc(16)-okW, wp.Y+sc(170)-sc(30)))
	if ButtonSized(okLabel, "about-ok", imgui.NewVec2(okW, imgui.FrameHeight()+2), true) {
		imgui.CloseCurrentPopup()
	}
	if imgui.IsKeyPressedBool(imgui.KeyEscape) {
		imgui.CloseCurrentPopup()
	}
	imgui.EndPopup()
}

func fixedWindowFlags() imgui.WindowFlags {
	return imgui.WindowFlagsNoTitleBar | imgui.WindowFlagsNoResize | imgui.WindowFlagsNoMove |
		imgui.WindowFlagsNoCollapse | imgui.WindowFlagsNoSavedSettings | imgui.WindowFlagsNoBringToFrontOnFocus
}

func place(x, y, w, h float32) {
	imgui.SetNextWindowPosV(imgui.NewVec2(x, y), imgui.CondAlways, imgui.NewVec2(0, 0))
	imgui.SetNextWindowSizeV(imgui.NewVec2(w, h), imgui.CondAlways)
}

func (a *App) drawCenterMessage() {
	msg := T("status.loading")
	if a.loadErr != "" {
		msg = T(a.errKey, "msg", a.loadErr)
	}
	if a.role == access.RoleNone {
		msg = T("login.locked")
	}
	pos := imgui.CursorScreenPos()
	avail := imgui.ContentRegionAvail()
	dl := imgui.WindowDrawList()
	dl.AddRectFilled(pos, imgui.NewVec2(pos.X+avail.X, pos.Y+avail.Y), CurrentPalette().Workspace)
	imgui.SetCursorScreenPos(imgui.NewVec2(pos.X+24, pos.Y+24))
	imgui.PushTextWrapPosV(avail.X - 24)
	imgui.PushStyleColorVec4(imgui.ColText, rgb4(255, 255, 255))
	imgui.TextUnformatted(msg)
	imgui.PopStyleColor()
	imgui.PopTextWrapPos()
}

func (a *App) drawToolbar(x, y, w, h float32) {
	place(x, y, w, h)
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(8, 4))
	imgui.BeginV("##toolbar", nil, fixedWindowFlags()|imgui.WindowFlagsNoScrollbar)
	imgui.PopStyleVar()

	if a.role == access.RoleAdmin {
		imgui.AlignTextToFramePadding()
		imgui.TextUnformatted(T("top.machine"))
		imgui.SameLine()
		imgui.SetNextItemWidth(sc(190))
		machinePreview := ""
		if a.Cfg != nil && a.machineIdx >= 0 && a.machineIdx < len(a.Cfg.Machines) {
			m := a.Cfg.Machines[a.machineIdx]
			machinePreview = m.Name
			if m.Address != "" {
				machinePreview = m.Name + "  (" + m.Address + ")"
			}
		}
		if imgui.BeginCombo("##machine", machinePreview) {
			if a.Cfg != nil {
				for i, m := range a.Cfg.Machines {
					label := m.Name
					if m.Address != "" {
						label = m.Name + "  (" + m.Address + ")"
					}
					if imgui.SelectableBoolV(label+"##mc"+fmt.Sprint(i), i == a.machineIdx, 0, imgui.NewVec2(0, 0)) && i != a.machineIdx {
						a.SelectMachine(i)
					}
					HotText(label, "")
				}
			}
			imgui.EndCombo()
		}
		ItemEdge(EdgeSunken)
		imgui.SameLine()
		if ButtonSized("…", "cfg-open", imgui.NewVec2(sc(26), imgui.FrameHeight()), true) {
			a.OpenConfigDialog()
		}
		ToolbarSeparator()
		imgui.SameLine()
	}

	imgui.AlignTextToFramePadding()
	imgui.TextUnformatted(T("top.model"))
	imgui.SameLine()
	imgui.SetNextItemWidth(sc(220))
	preview := ""
	if a.modelIdx >= 0 {
		preview = a.Models[a.modelIdx].ID + " · " + a.Models[a.modelIdx].Version
	}
	if imgui.BeginCombo("##model", preview) {
		for i, m := range a.Models {
			label := m.ID + " · " + m.Version
			if imgui.SelectableBoolV(label+"##m"+fmt.Sprint(i), i == a.modelIdx, 0, imgui.NewVec2(0, 0)) && i != a.modelIdx {
				a.LoadModel(i)
			}
			HotText(label, "")
		}
		imgui.EndCombo()
	}
	ItemEdge(EdgeSunken)
	ToolbarSeparator()
	imgui.SameLine()

	imgui.AlignTextToFramePadding()
	status := ""
	switch {
	case a.loading:
		status = T("status.loading")
	case a.Scene != nil:
		status = T("status.loaded", "w", a.Scene.W, "h", a.Scene.H, "ms", F(a.Scene.LoadMs, 0))
	}
	imgui.TextUnformatted(status)

	// Right end: who is logged in plus the login / logout button.
	roleLabel := T("login.user.operator")
	roleID, roleText := "login-open", T("login.open")
	if a.role == access.RoleAdmin {
		roleLabel, roleID, roleText = T("login.user.admin"), "login-logout", T("login.logout")
	}
	roleW := ButtonWidth(roleText)
	avail := imgui.ContentRegionAvail().X
	if avail > roleW+imgui.CalcTextSize(roleLabel).X+sc(16) {
		imgui.SameLine()
		imgui.SetCursorPosX(imgui.CursorPosX() + avail - roleW - imgui.CalcTextSize(roleLabel).X - imgui.CurrentStyle().ItemSpacing().X)
		imgui.AlignTextToFramePadding()
		imgui.TextUnformatted(roleLabel)
		imgui.SameLine()
		if ButtonSized(roleText, roleID, imgui.NewVec2(roleW, imgui.FrameHeight()), true) {
			if a.role == access.RoleAdmin {
				a.Logout()
			} else {
				a.OpenLogin()
			}
		}
	}

	dl := imgui.WindowDrawList()
	wp, ws := imgui.WindowPos(), imgui.WindowSize()
	strip(dl, wp.X, wp.Y+ws.Y-2, wp.X+ws.X, wp.Y+ws.Y, CurrentPalette().Shadow)
	strip(dl, wp.X, wp.Y+ws.Y-1, wp.X+ws.X, wp.Y+ws.Y, CurrentPalette().Highlight)
	imgui.End()
}

func (a *App) drawStatus(x, y, w, h float32) {
	place(x, y, w, h)
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(0, 0))
	imgui.BeginV("##status", nil, fixedWindowFlags()|imgui.WindowFlagsNoScrollbar)
	imgui.PopStyleVar()

	xy, gv := "XY —", "GV —"
	if mx, my, in := a.Viewer.Mouse(); in && a.Scene != nil {
		ix, iy := int(mx), int(my)
		xy = fmt.Sprintf("XY %d, %d", ix, iy)
		if p := a.ActivePanel(); p != nil {
			if s := p.Hover(ix, iy); s != "" {
				gv = s
			}
		}
	}
	elapsed := T("status.elapsedNone")
	if a.elapsedMs > 0 {
		elapsed = T("status.elapsed", "ms", F(a.elapsedMs, 1))
	}
	hint := a.hint
	if a.Jobs.AnyBusy() {
		hint = T("status.busy") + "  " + hint
	}
	zoom := T("status.zoom", "pct", F(float64(a.Viewer.Zoom())*100, 0))
	fpsHeap := fmt.Sprintf("%.0f fps | heap %.0f MB", imgui.CurrentIO().Framerate(), a.heapMB)

	xyW := imgui.CalcTextSize("XY 00000, 00000").X + 16
	gvW := imgui.CalcTextSize(gv).X + 16
	if gvW < sc(130) {
		gvW = sc(130)
	}
	zoomW := sc(100)
	elapsedW := sc(130)
	fpsW := imgui.CalcTextSize("999 fps | heap 9999 MB").X + 16
	const gap = 2
	hintW := w - xyW - gvW - zoomW - elapsedW - fpsW - 5*gap
	if hintW < 0 {
		hintW = 0
	}

	dl := imgui.WindowDrawList()
	col := TextColor()
	py := y + 2
	ph := h - 4
	drawPane := func(px, pw float32, text string, clip bool) {
		min := imgui.NewVec2(px, py)
		max := imgui.NewVec2(px+pw, py+ph)
		DrawEdge(dl, min, max, EdgeStatus)
		ts := imgui.CalcTextSize(text)
		ty := min.Y + (ph-ts.Y)/2
		if clip {
			dl.PushClipRect(imgui.NewVec2(min.X+2, min.Y), imgui.NewVec2(max.X-2, max.Y))
		}
		dl.AddTextVec2(imgui.NewVec2(min.X+6, ty), col, text)
		if clip {
			dl.PopClipRect()
		}
	}
	px := x
	drawPane(px, hintW, hint, true)
	px += hintW + gap
	drawPane(px, xyW, xy, false)
	px += xyW + gap
	drawPane(px, gvW, gv, false)
	px += gvW + gap
	drawPane(px, zoomW, zoom, false)
	px += zoomW + gap
	drawPane(px, elapsedW, elapsed, false)
	px += elapsedW + gap
	drawPane(px, fpsW, fpsHeap, false)

	// Size grip in the bottom-right corner.
	gx := x + w - 14
	gy := y + h - 14
	p := CurrentPalette()
	for i := 0; i < 3; i++ {
		off := float32(i * 4)
		strip(dl, gx+2+off, gy+12, gx+12, gy+13, p.Shadow)
		strip(dl, gx+3+off, gy+13, gx+13, gy+14, p.Highlight)
	}
	imgui.End()
}
