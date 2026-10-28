package app

import (
	"afvi/teach-imgui/internal/access"
)

// The window is one program with two faces: the teaching program and the camera studio. Each
// face keeps its own page set and its own state (painted defects, alignment ROIs, the golden
// board) for as long as the window lives, so switching back and forth is instant and throws
// nothing away. Only the face the user asks for is built, so a program that is never opened
// costs nothing.

// profileSet is the page set of one face.
type profileSet struct {
	allPanels []Panel
	allCtors  []panelCtor
}

// buildProfilePanels constructs the pages a profile registers, in tab order.
func buildProfilePanels(a *App, p Profile) *profileSet {
	set := &profileSet{}
	for _, c := range panelCtors {
		if !c.allows(p) {
			continue
		}
		set.allPanels = append(set.allPanels, c.make(a))
		set.allCtors = append(set.allCtors, c)
	}
	return set
}

// Profile returns the face this window currently shows.
func (a *App) Profile() Profile { return a.profile }

// profileSet returns (and on first use builds) the page set of p. The teaching program owns the
// alignment flow and the golden board; they are created with its page set and kept afterwards,
// because the pages hold references to them.
func (a *App) profilePanels(p Profile) *profileSet {
	if a.sets == nil {
		a.sets = map[Profile]*profileSet{}
	}
	if set, ok := a.sets[p]; ok {
		return set
	}
	if p == ProfileTeach {
		if a.align == nil {
			a.align = newAlignFlow(a)
		}
		if a.golden == nil {
			a.golden = newGoldenBoard(a)
		}
		if a.Scene != nil {
			a.align.modelLoaded(a.Scene)
			a.golden.modelLoaded(a.Scene)
		}
	}
	set := buildProfilePanels(a, p)
	a.sets[p] = set
	if a.Scene != nil {
		// a face that is opened for the first time while a model is loaded must see it
		for _, page := range set.allPanels {
			page.ModelLoaded()
		}
	}
	return set
}

// SwitchProfile makes the window the teaching program or the camera studio, without a restart
// and without reloading the model. Switching to the studio needs an admin login: the switch then
// waits for the login dialog (pendingProfile) and happens when it succeeds.
//
// The live link is restarted, because its role depends on the face: the studio listens, the
// teaching program dials. Painted defects, ROIs and the golden board survive the switch.
func (a *App) SwitchProfile(p Profile) bool {
	if p != ProfileTeach && p != ProfileStudio {
		return false
	}
	if p == a.profile {
		return true
	}
	if p == ProfileStudio && a.role != access.RoleAdmin {
		a.pendingProfile = p
		a.loginErr = ""
		a.OpenLogin()
		return false
	}
	a.pendingProfile = ProfileNone
	if active := a.ActivePanel(); active != nil {
		active.Deactivate()
	}
	// the link's role follows the face: stop it, switch, start it again on the same address
	addr := a.liveAddr
	a.closeLive()
	if p == ProfileStudio {
		a.role = access.RoleAdmin // SwitchProfile only gets here with an admin
	}

	a.profile = p
	set := a.profilePanels(p)
	a.allPanels, a.allCtors = set.allPanels, set.allCtors
	a.rebuildPanels()
	a.tab = 0

	if a.be != nil {
		a.be.SetWindowTitle(p.WindowTitle())
	}
	if addr != "" && (a.selftest == nil || a.live != nil || a.linkOn) {
		a.StartLive(addr)
	}
	a.hint = ""
	if active := a.ActivePanel(); active != nil {
		active.Activate()
	}
	a.rememberProfile()
	return true
}

// ProfileSwitchAllowed reports whether the user may switch to p right now (the studio needs an
// admin login, which the login dialog provides).
func (a *App) ProfileSwitchAllowed(p Profile) bool {
	return p == a.profile || p == ProfileTeach || a.role == access.RoleAdmin
}

// rememberProfile stores the face the user ended up with, when the configuration asks for it.
func (a *App) rememberProfile() {
	if a.Cfg == nil || a.selftest != nil || !a.Cfg.RememberLast || a.Cfg.StartProfile == string(a.profile) {
		return
	}
	a.Cfg.StartProfile = string(a.profile)
	if a.CfgPath != "" {
		_ = a.Cfg.Save(a.CfgPath)
	}
}
