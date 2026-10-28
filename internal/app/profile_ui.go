package app

import (
	"github.com/AllenDang/cimgui-go/imgui"
)

// The window shows one of two faces; this file draws the controls that say so: the segment in
// the toolbar, the "side by side" button and the picker shown at startup when the configuration
// does not name a face.

// drawProfileSwitch draws the 教学程序 / 相机工作室 latch buttons and the side-by-side button at
// the left end of the toolbar, on the toolbar's own row (the second-level tab strip is a row of
// its own and would push the toolbar apart). It is drawn for every role: which program the
// window is must always be visible, and the studio itself asks for the admin login.
func (a *App) drawProfileSwitch() {
	if ToolButton(T("profile.teach"), "profile-teach", a.profile == ProfileTeach) {
		a.SwitchProfile(ProfileTeach)
	}
	imgui.SameLine()
	if ToolButton(T("profile.studio"), "profile-studio", a.profile == ProfileStudio) {
		a.SwitchProfile(ProfileStudio)
	}
	imgui.SameLine()
	split := T("profile.split")
	if ButtonSized(split, "profile-split", imgui.NewVec2(ButtonWidth(split), imgui.FrameHeight()+2), true) {
		a.SplitWithPeer()
	}
	HotText(split, T("profile.split.hint"))
	imgui.SameLine()
	ToolbarSeparator()
	imgui.SameLine()
}

// drawProfilePicker is the startup dialog: which face does this window open with? It is shown
// when the configuration says "ask" (-profile ask, start_profile = "ask", or nothing at all),
// and it offers to remember the answer in afvi.toml.
func (a *App) drawProfilePicker() {
	if a.pickProfile {
		imgui.OpenPopupStr("profile-pick")
		a.pickProfile = false
	}
	vp := imgui.MainViewport()
	center := imgui.NewVec2(vp.Pos().X+vp.Size().X/2, vp.Pos().Y+vp.Size().Y/2)
	imgui.SetNextWindowPosV(center, imgui.CondAlways, imgui.NewVec2(0.5, 0.5))
	imgui.SetNextWindowSizeV(imgui.NewVec2(sc(430), sc(210)), imgui.CondAlways)
	flags := imgui.WindowFlagsNoTitleBar | imgui.WindowFlagsNoResize | imgui.WindowFlagsNoMove |
		imgui.WindowFlagsNoSavedSettings | imgui.WindowFlagsNoScrollbar
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(0, 0))
	open := imgui.BeginPopupModalV("profile-pick", nil, flags)
	imgui.PopStyleVar()
	if !open {
		return
	}
	wp, ws := imgui.WindowPos(), imgui.WindowSize()
	dl := imgui.WindowDrawList()
	DrawEdge(dl, wp, imgui.NewVec2(wp.X+ws.X, wp.Y+ws.Y), EdgeWindow)
	Caption(dl, imgui.NewVec2(wp.X+3, wp.Y+3), imgui.NewVec2(wp.X+ws.X-3, wp.Y+3+sc(20)),
		T("profile.pick.title"), true)

	x, y := wp.X+sc(16), wp.Y+sc(40)
	imgui.SetCursorScreenPos(imgui.NewVec2(x, y))
	imgui.PushTextWrapPosV(ws.X - sc(16))
	imgui.TextUnformatted(T("profile.pick.text"))
	imgui.PopTextWrapPos()

	// bottom row: [x] 记住我的选择            [教学程序] [相机工作室]
	btnY := wp.Y + ws.Y - sc(34)
	bw := sc(112)
	teachW, studioW := ButtonWidth(T("profile.teach")), ButtonWidth(T("profile.studio"))
	if teachW > bw {
		bw = teachW
	}
	if studioW > bw {
		bw = studioW
	}
	studioX := wp.X + ws.X - sc(16) - bw
	teachX := studioX - sc(8) - bw
	imgui.SetCursorScreenPos(imgui.NewVec2(teachX, btnY))
	if ButtonSized(T("profile.teach"), "pick-teach", imgui.NewVec2(bw, imgui.FrameHeight()+4), a.profile == ProfileTeach) {
		a.finishPick(ProfileTeach)
	}
	imgui.SetCursorScreenPos(imgui.NewVec2(studioX, btnY))
	if ButtonSized(T("profile.studio"), "pick-studio", imgui.NewVec2(bw, imgui.FrameHeight()+4), a.profile == ProfileStudio) {
		a.finishPick(ProfileStudio)
	}
	imgui.SetCursorScreenPos(imgui.NewVec2(x, btnY+sc(4)))
	remember := a.Cfg != nil && a.Cfg.RememberLast
	if imgui.Checkbox(T("profile.pick.remember"), &remember) && a.Cfg != nil {
		a.Cfg.RememberLast = remember
		if a.CfgPath != "" {
			_ = a.Cfg.Save(a.CfgPath)
		}
	}
	if imgui.IsKeyPressedBool(imgui.KeyEscape) {
		// Esc keeps the face the window already has instead of trapping the user
		imgui.CloseCurrentPopup()
	}
	imgui.EndPopup()
}

// finishPick applies the startup choice and closes the picker.
func (a *App) finishPick(p Profile) {
	imgui.CloseCurrentPopup()
	if p != a.profile {
		a.SwitchProfile(p)
	}
	a.rememberProfile()
}
