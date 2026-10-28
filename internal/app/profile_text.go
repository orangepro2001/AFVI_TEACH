package app

// Texts of the program switch (profile_ui.go / split.go). Every key must exist in all three
// languages.
func init() {
	Messages(map[string]string{
		"profile.teach":          "教学程序",
		"profile.studio":         "相机工作室",
		"profile.split":          "⇔ 并排",
		"profile.split.hint":     "把另一个界面作为独立窗口打开，左右并排（两个窗口可各自操作）",
		"profile.pick.title":     "启动哪个界面？",
		"profile.pick.text":      "本程序一个窗口里同时包含「教学程序」和「相机工作室」，启动时选一个，之后随时可以在工具栏左侧切换。相机工作室需要管理员登录。",
		"profile.pick.remember":  "记住我的选择",
		"profile.split.started":  "已打开另一个窗口（{profile}），两个窗口左右并排。",
		"profile.split.failed":   "无法打开另一个窗口：{msg}",
		"profile.split.selftest": "自测模式下不打开第二个窗口。",
	}, map[string]string{
		"profile.teach":          "Teaching",
		"profile.studio":         "Camera studio",
		"profile.split":          "⇔ Side by side",
		"profile.split.hint":     "Open the other window next to this one and tile both",
		"profile.pick.title":     "Which window?",
		"profile.pick.text":      "This program contains both the teaching program and the camera studio. Pick one to start with; the switch at the left of the toolbar changes it at any time. The studio needs an admin login.",
		"profile.pick.remember":  "Remember my choice",
		"profile.split.started":  "Opened the other window ({profile}); both are placed side by side.",
		"profile.split.failed":   "Cannot open the other window: {msg}",
		"profile.split.selftest": "The self-test does not open a second window.",
	}, map[string]string{
		"profile.teach":          "교육 프로그램",
		"profile.studio":         "카메라 스튜디오",
		"profile.split":          "⇔ 나란히",
		"profile.split.hint":     "다른 화면을 별도 창으로 열고 좌우로 나란히 배치합니다",
		"profile.pick.title":     "어떤 화면으로 시작할까요?",
		"profile.pick.text":      "이 프로그램은 교육 프로그램과 카메라 스튜디오를 하나의 창에 담고 있습니다. 시작할 화면을 고르십시오. 이후에는 도구 모음 왼쪽에서 언제든 바꿀 수 있습니다. 카메라 스튜디오는 관리자 로그인이 필요합니다.",
		"profile.pick.remember":  "내 선택 기억하기",
		"profile.split.started":  "다른 창({profile})을 열었고 두 창을 나란히 배치했습니다.",
		"profile.split.failed":   "다른 창을 열 수 없습니다: {msg}",
		"profile.split.selftest": "자체 테스트에서는 두 번째 창을 열지 않습니다.",
	})
}
