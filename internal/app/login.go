package app

import (
	"github.com/AllenDang/cimgui-go/imgui"
)

// drawLogin draws the modal login dialog. It is shown from the menu and the toolbar of
// the operator and at start-up in the studio (locked: the dialog cannot be cancelled).
func (a *App) drawLogin() {
	if a.loginOpen {
		imgui.OpenPopupStr("login")
		a.loginOpen = false
	}
	vp := imgui.MainViewport()
	center := imgui.NewVec2(vp.Pos().X+vp.Size().X/2, vp.Pos().Y+vp.Size().Y/2)
	imgui.SetNextWindowPosV(center, imgui.CondAlways, imgui.NewVec2(0.5, 0.5))
	imgui.SetNextWindowSizeV(imgui.NewVec2(sc(400), sc(210)), imgui.CondAlways)
	flags := imgui.WindowFlagsNoTitleBar | imgui.WindowFlagsNoResize | imgui.WindowFlagsNoMove |
		imgui.WindowFlagsNoSavedSettings | imgui.WindowFlagsNoScrollbar
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(0, 0))
	open := imgui.BeginPopupModalV("login", nil, flags)
	imgui.PopStyleVar()
	if !open {
		return
	}
	wp, ws := imgui.WindowPos(), imgui.WindowSize()
	dl := imgui.WindowDrawList()
	DrawEdge(dl, wp, imgui.NewVec2(wp.X+ws.X, wp.Y+ws.Y), EdgeWindow)
	capMin := imgui.NewVec2(wp.X+3, wp.Y+3)
	capMax := imgui.NewVec2(wp.X+ws.X-3, wp.Y+3+sc(20))
	Caption(dl, capMin, capMax, T("login.title"), true)
	imgui.SetCursorScreenPos(imgui.NewVec2(capMax.X-sc(18), capMin.Y+3))
	if ButtonSized("x", "login-close", imgui.NewVec2(sc(16), sc(14)), !a.loginLocked) {
		a.closeLogin()
	}

	imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+sc(16), wp.Y+sc(44)))
	labelColumn(T("login.user"))
	imgui.InputTextWithHint("##login-user", "", &a.loginUser, imgui.InputTextFlagsNone, nil)
	ItemEdge(EdgeSunken)

	imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+sc(16), wp.Y+sc(72)))
	labelColumn(T("login.password"))
	imgui.InputTextWithHint("##login-pass", "", &a.loginPass, imgui.InputTextFlagsPassword, nil)
	ItemEdge(EdgeSunken)

	if a.loginErr != "" {
		imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+sc(16), wp.Y+sc(98)))
		imgui.TextColored(CurrentPalette().NG, T(a.loginErr))
	} else {
		imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+sc(16), wp.Y+sc(98)))
		Note(T("login.hint"))
	}

	okLabel, cancelLabel := T("login.ok"), T("login.cancel")
	if a.loginLocked {
		cancelLabel = T("login.exit")
	}
	okW, cancelW := ButtonWidth(okLabel), ButtonWidth(cancelLabel)
	if okW < sc(75) {
		okW = sc(75)
	}
	if cancelW < sc(75) {
		cancelW = sc(75)
	}
	spacing := imgui.CurrentStyle().ItemSpacing().X
	btnX := wp.X + ws.X - sc(16) - okW - cancelW - spacing
	btnY := wp.Y + sc(210) - sc(30)
	imgui.SetCursorScreenPos(imgui.NewVec2(btnX, btnY))
	if ButtonSized(okLabel, "login-ok", imgui.NewVec2(okW, imgui.FrameHeight()+2), true) {
		if a.TryLogin(a.loginUser, a.loginPass) {
			imgui.CloseCurrentPopup()
		}
	}
	imgui.SetCursorScreenPos(imgui.NewVec2(btnX+okW+spacing, btnY))
	if ButtonSized(cancelLabel, "login-cancel", imgui.NewVec2(cancelW, imgui.FrameHeight()+2), true) {
		if a.loginLocked {
			a.be.SetShouldClose(true)
		} else {
			a.closeLogin()
		}
	}

	if imgui.IsKeyPressedBool(imgui.KeyEnter) {
		if a.TryLogin(a.loginUser, a.loginPass) {
			imgui.CloseCurrentPopup()
		}
	}
	if imgui.IsKeyPressedBool(imgui.KeyEscape) && !a.loginLocked {
		a.closeLogin()
	}
	imgui.EndPopup()
}

// closeLogin closes the dialog without logging in and forgets the password.
func (a *App) closeLogin() {
	a.loginOpen = false
	a.loginPass = ""
	a.loginErr = ""
	imgui.CloseCurrentPopup()
}

func init() {
	Messages(map[string]string{
		"login.title":         "登录",
		"login.user":          "用户名",
		"login.password":      "密码",
		"login.ok":            "登录",
		"login.cancel":        "取消",
		"login.exit":          "退出",
		"login.failed":        "用户名或密码不正确。",
		"login.locked":        "请先登录。",
		"login.open":          "登录…",
		"login.logout":        "注销",
		"login.user.operator": "操作员",
		"login.user.admin":    "管理员",
		"menu.login":          "管理员登录…",
		"menu.logout":         "注销",
		"login.hint":          "操作员无需密码；工程师请使用管理员账号。",
	}, map[string]string{
		"login.title":         "Login",
		"login.user":          "User",
		"login.password":      "Password",
		"login.ok":            "Login",
		"login.cancel":        "Cancel",
		"login.exit":          "Exit",
		"login.failed":        "Wrong user name or password.",
		"login.locked":        "Please log in first.",
		"login.open":          "Log in…",
		"login.logout":        "Log out",
		"login.user.operator": "Operator",
		"login.user.admin":    "Administrator",
		"menu.login":          "Admin login…",
		"menu.logout":         "Log out",
		"login.hint":          "Operators need no password; engineers use the administrator account.",
	}, map[string]string{
		"login.title":         "로그인",
		"login.user":          "사용자",
		"login.password":      "비밀번호",
		"login.ok":            "로그인",
		"login.cancel":        "취소",
		"login.exit":          "종료",
		"login.failed":        "사용자 이름 또는 비밀번호가 올바르지 않습니다.",
		"login.locked":        "먼저 로그인하세요.",
		"login.open":          "로그인…",
		"login.logout":        "로그아웃",
		"login.user.operator": "오퍼레이터",
		"login.user.admin":    "관리자",
		"menu.login":          "관리자 로그인…",
		"menu.logout":         "로그아웃",
		"login.hint":          "오퍼레이터는 비밀번호가 필요 없습니다. 엔지니어는 관리자 계정을 사용하세요.",
	})
}
