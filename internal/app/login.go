package app

import "afvi/teach-imgui/internal/ui"

// buildLogin records the modal login dialog. It is shown from the menu and the toolbar of
// the operator and at start-up in the studio (locked: the dialog cannot be cancelled).
func (a *App) buildLogin() {
	if !a.loginOpen {
		return
	}
	submit := func() {
		a.TryLogin(a.loginUser, a.loginPass)
	}
	closed := ui.Dialog(T("login.title"), "login", 400, func() {
		InputText(T("login.user"), "login-user", &a.loginUser, "")
		ui.InputTextEx(T("login.password"), "login-pass", &a.loginPass, "", true)
		if a.loginErr != "" {
			ui.TextTone(T(a.loginErr), "bad")
		} else {
			Note(T("login.hint"))
		}
		ui.Spacing()
		if ui.ButtonEx(T("login.ok"), "login-ok", true, true) {
			submit()
		}
		ui.SameLine()
		cancel := T("login.cancel")
		if a.loginLocked {
			cancel = T("login.exit")
		}
		if Button(cancel, "login-cancel") {
			if a.loginLocked {
				a.Quit()
			} else {
				a.closeLogin()
			}
		}
	})
	if ui.KeyPressed("Enter") {
		submit()
	}
	if (closed || ui.KeyPressed("Escape")) && !a.loginLocked {
		a.closeLogin()
	}
}

// closeLogin closes the dialog without logging in and forgets the password.
func (a *App) closeLogin() {
	a.loginOpen = false
	a.loginPass = ""
	a.loginErr = ""
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
