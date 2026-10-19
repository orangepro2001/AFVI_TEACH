package installer

import "strings"

// Languages are the installer's languages, the same three as the programs'.
var Languages = []string{"zh", "en", "ko"}

// LangFromLANGID maps a Windows language identifier (GetUserDefaultUILanguage) to one of
// Languages; anything that is not Chinese or Korean gets English.
func LangFromLANGID(id uint16) string {
	switch id & 0x3ff {
	case 0x04:
		return "zh"
	case 0x12:
		return "ko"
	}
	return "en"
}

// ValidLang returns lang when it is one of Languages and "" otherwise.
func ValidLang(lang string) string {
	for _, l := range Languages {
		if l == lang {
			return lang
		}
	}
	return ""
}

// T returns the installer text for key in lang with the {name} placeholders replaced by the
// key/value pairs in kv; an unknown key comes back as itself.
func T(lang, key string, kv ...string) string {
	s, ok := texts[lang][key]
	if !ok {
		s, ok = texts["en"][key]
	}
	if !ok {
		return key
	}
	if len(kv) >= 2 {
		pairs := make([]string, 0, len(kv))
		for i := 0; i+1 < len(kv); i += 2 {
			pairs = append(pairs, "{"+kv[i]+"}", kv[i+1])
		}
		s = strings.NewReplacer(pairs...).Replace(s)
	}
	return s
}

var texts = map[string]map[string]string{
	"zh": {
		"title":            "{product} 安装程序",
		"version":          "版本 {version}",
		"upgrade":          "检测到已安装的版本 {old}，将升级到 {version}。配置文件和相机输出会保留。",
		"location":         "安装位置",
		"browse":           "浏览…",
		"browseTitle":      "选择安装文件夹",
		"writableNote":     "程序把配置（afvi.toml）和相机输出（captures）写在安装文件夹里，请选择当前账户可以写入的文件夹。",
		"components":       "将安装：",
		"optMenu":          "创建开始菜单快捷方式",
		"optDesktop":       "创建桌面快捷方式",
		"optPath":          "把安装文件夹加入 PATH（可在命令行直接运行 camera.exe）",
		"optLaunch":        "安装完成后启动 {program}",
		"install":          "安装",
		"cancel":           "取消",
		"finish":           "完成",
		"installing":       "正在安装…",
		"copying":          "正在复制 {name}",
		"doneTitle":        "安装完成",
		"done":             "{product} 已安装到：\n{dir}",
		"failedTitle":      "安装失败",
		"failed":           "安装没有完成：\n{err}",
		"inUse":            "{path} 正在被使用。\n请关闭 {product} 的所有窗口，然后重试。",
		"nonEmpty":         "文件夹 {dir} 不是空的，也不是 {product} 的安装文件夹。\n仍然安装到这里吗？",
		"badDir":           "请输入完整的文件夹路径，例如 {example}",
		"notWritable":      "无法写入 {dir}：\n{err}\n请选择当前账户可以写入的文件夹。",
		"uninstallTitle":   "卸载 {product}",
		"uninstallConfirm": "确定要卸载 {product} 吗？",
		"uninstallData":    "同时删除配置文件和相机输出（afvi.toml 与 captures 文件夹）吗？\n选择“否”会保留它们。",
		"uninstalled":      "{product} 已卸载。",
		"uninstallPartial": "下列文件正在使用，没有删除：\n{files}\n请关闭 {product} 后再运行一次卸载。",
		"notInstalled":     "没有找到 {product} 的安装信息，无法卸载。",
	},
	"en": {
		"title":            "{product} Setup",
		"version":          "Version {version}",
		"upgrade":          "Version {old} is installed and will be upgraded to {version}. Your configuration and camera output are kept.",
		"location":         "Install location",
		"browse":           "Browse...",
		"browseTitle":      "Choose the install folder",
		"writableNote":     "The programs write their configuration (afvi.toml) and the camera output (captures) into the install folder, so choose a folder your account can write to.",
		"components":       "To be installed:",
		"optMenu":          "Create Start Menu shortcuts",
		"optDesktop":       "Create desktop shortcuts",
		"optPath":          "Add the install folder to PATH (run camera.exe from a command prompt)",
		"optLaunch":        "Start {program} when the setup is done",
		"install":          "Install",
		"cancel":           "Cancel",
		"finish":           "Finish",
		"installing":       "Installing...",
		"copying":          "Copying {name}",
		"doneTitle":        "Setup complete",
		"done":             "{product} was installed to:\n{dir}",
		"failedTitle":      "Setup failed",
		"failed":           "The installation did not finish:\n{err}",
		"inUse":            "{path} is in use.\nClose all {product} windows and try again.",
		"nonEmpty":         "The folder {dir} is not empty and is not an install folder of {product}.\nInstall here anyway?",
		"badDir":           "Enter a complete folder path, for example {example}",
		"notWritable":      "Cannot write to {dir}:\n{err}\nChoose a folder your account can write to.",
		"uninstallTitle":   "Uninstall {product}",
		"uninstallConfirm": "Uninstall {product}?",
		"uninstallData":    "Also delete the configuration and the camera output (afvi.toml and the captures folder)?\nChoose No to keep them.",
		"uninstalled":      "{product} was uninstalled.",
		"uninstallPartial": "These files are in use and were not removed:\n{files}\nClose {product} and run the uninstaller again.",
		"notInstalled":     "No installation of {product} was found, nothing to uninstall.",
	},
	"ko": {
		"title":            "{product} 설치 프로그램",
		"version":          "버전 {version}",
		"upgrade":          "버전 {old}이(가) 설치되어 있으며 {version}(으)로 업그레이드합니다. 설정 파일과 카메라 출력은 유지됩니다.",
		"location":         "설치 위치",
		"browse":           "찾아보기…",
		"browseTitle":      "설치 폴더 선택",
		"writableNote":     "프로그램은 설정(afvi.toml)과 카메라 출력(captures)을 설치 폴더에 기록하므로 현재 계정이 쓸 수 있는 폴더를 선택하세요.",
		"components":       "설치할 항목:",
		"optMenu":          "시작 메뉴 바로 가기 만들기",
		"optDesktop":       "바탕 화면 바로 가기 만들기",
		"optPath":          "설치 폴더를 PATH에 추가 (명령 프롬프트에서 camera.exe 실행)",
		"optLaunch":        "설치가 끝나면 {program} 실행",
		"install":          "설치",
		"cancel":           "취소",
		"finish":           "마침",
		"installing":       "설치하는 중…",
		"copying":          "{name} 복사 중",
		"doneTitle":        "설치 완료",
		"done":             "{product}을(를) 다음 위치에 설치했습니다:\n{dir}",
		"failedTitle":      "설치 실패",
		"failed":           "설치를 완료하지 못했습니다:\n{err}",
		"inUse":            "{path}이(가) 사용 중입니다.\n{product}의 모든 창을 닫고 다시 시도하세요.",
		"nonEmpty":         "{dir} 폴더가 비어 있지 않으며 {product}의 설치 폴더도 아닙니다.\n그래도 여기에 설치할까요?",
		"badDir":           "전체 폴더 경로를 입력하세요. 예: {example}",
		"notWritable":      "{dir}에 쓸 수 없습니다:\n{err}\n현재 계정이 쓸 수 있는 폴더를 선택하세요.",
		"uninstallTitle":   "{product} 제거",
		"uninstallConfirm": "{product}을(를) 제거할까요?",
		"uninstallData":    "설정 파일과 카메라 출력(afvi.toml과 captures 폴더)도 삭제할까요?\n‘아니요’를 선택하면 유지됩니다.",
		"uninstalled":      "{product}을(를) 제거했습니다.",
		"uninstallPartial": "다음 파일이 사용 중이어서 삭제하지 못했습니다:\n{files}\n{product}을(를) 닫고 제거 프로그램을 다시 실행하세요.",
		"notInstalled":     "{product}의 설치 정보를 찾을 수 없어 제거할 수 없습니다.",
	},
}
