package app

import (
	"fmt"
	"strings"
)

// Lang is a UI language code.
type Lang string

const (
	LangZh Lang = "zh"
	LangEn Lang = "en"
	LangKo Lang = "ko"
)

// Langs lists the selectable languages in display order; labels stay in their own script.
var Langs = []struct {
	Code  Lang
	Label string
}{
	{LangZh, "中文"},
	{LangEn, "English"},
	{LangKo, "한국어"},
}

var (
	currentLang  = LangZh
	dictionaries = map[Lang]map[string]string{LangZh: {}, LangEn: {}, LangKo: {}}
)

// Messages registers translations; every panel calls it from init() for its own keys.
func Messages(zh, en, ko map[string]string) {
	for lang, dict := range map[Lang]map[string]string{LangZh: zh, LangEn: en, LangKo: ko} {
		for k, v := range dict {
			dictionaries[lang][k] = v
		}
	}
}

// SetLang switches the UI language at runtime; no reload is needed in immediate mode.
func SetLang(l Lang) {
	if _, ok := dictionaries[l]; ok {
		currentLang = l
	}
}

// CurrentLang returns the active UI language.
func CurrentLang() Lang { return currentLang }

// ParseLang maps a flag value to a language, defaulting to Chinese.
func ParseLang(s string) Lang {
	switch Lang(strings.ToLower(s)) {
	case LangEn:
		return LangEn
	case LangKo:
		return LangKo
	}
	return LangZh
}

// T looks up key in the current language (falling back to zh, then the key itself) and
// replaces {name} placeholders from alternating name/value arguments.
func T(key string, args ...any) string {
	s, ok := dictionaries[currentLang][key]
	if !ok {
		s, ok = dictionaries[LangZh][key]
	}
	if !ok {
		s = key
	}
	for i := 0; i+1 < len(args); i += 2 {
		name, _ := args[i].(string)
		var val string
		switch v := args[i+1].(type) {
		case string:
			val = v
		default:
			val = fmt.Sprint(v)
		}
		s = strings.ReplaceAll(s, "{"+name+"}", val)
	}
	return s
}

func init() {
	Messages(map[string]string{
		"app.title":          "AFVI 教学核心",
		"top.machine":        "机台",
		"top.model":          "型号",
		"top.theme":          "主题",
		"top.lang":           "语言",
		"theme.dark":         "深色（夜间）",
		"theme.light":        "经典（Windows）",
		"menu.file":          "文件(F)",
		"menu.view":          "视图(V)",
		"menu.help":          "帮助(H)",
		"menu.machineConfig": "机台配置…",
		"menu.rescan":        "重新扫描型号",
		"menu.exit":          "退出",
		"menu.theme":         "主题",
		"menu.lang":          "语言",
		"menu.fit":           "适应窗口",
		"menu.about":         "关于…",
		"about.title":        "关于 Color AFVI Master Teach",
		"about.text":         "AFVI 彩色检查机教学软件（原生 Win32 风格界面）",
		"tab.light":          "光照与 GV",
		"tab.align":          "对齐",
		"tab.defect":         "缺陷检出",
		"status.zoom":        "缩放 {pct}%",
		"status.elapsed":     "耗时 {ms} ms",
		"status.elapsedNone": "耗时 —",
		"status.loading":     "加载中…",
		"status.loaded":      "已加载 {w}×{h} · 用时 {ms} ms",
		"status.loadFailed":  "加载失败：{msg}",
		"status.noModels":    "无可用型号",
		"status.busy":        "计算中…",
		"msg.noModels":       "机台 {msg} 没有找到可用型号。\n\n期望的目录结构：\n<FM1|FM2|BM 路径>/PxRepository/<型号>/\n  Metal.tif, SR.tif, Open.tif\n  <版本>/Line/<Top|Bottom>/Layer/MEDIAN/Median_0_1_0.tif\n\n请打开 文件 ▸ 机台配置 检查路径，或直接编辑 afvi.toml。",
		"msg.noReference":    "找不到 reference 目录。可设置环境变量 AFVI_REFERENCE_DIR，或用 -ref 指定，或在 afvi.toml / -config 中配置机台路径。",
		"common.error":       "错误：{msg}",
		"common.none":        "—",
		"viewer.hintPan":     "滚轮缩放 · 中键/右键拖动平移 · 双击适应窗口",
	}, map[string]string{
		"app.title":          "AFVI Teaching Core",
		"top.machine":        "Machine",
		"top.model":          "Model",
		"top.theme":          "Theme",
		"top.lang":           "Language",
		"theme.dark":         "Dark (night)",
		"theme.light":        "Classic (Windows)",
		"menu.file":          "File",
		"menu.view":          "View",
		"menu.help":          "Help",
		"menu.machineConfig": "Machine Configuration…",
		"menu.rescan":        "Rescan models",
		"menu.exit":          "Exit",
		"menu.theme":         "Theme",
		"menu.lang":          "Language",
		"menu.fit":           "Fit view",
		"menu.about":         "About…",
		"about.title":        "About Color AFVI Master Teach",
		"about.text":         "AFVI colour inspection machine teaching tool (native Win32-style UI)",
		"tab.light":          "Light & GV",
		"tab.align":          "Alignment",
		"tab.defect":         "Defect Detection",
		"status.zoom":        "Zoom {pct}%",
		"status.elapsed":     "Elapsed {ms} ms",
		"status.elapsedNone": "Elapsed —",
		"status.loading":     "Loading…",
		"status.loaded":      "Loaded {w}×{h} · {ms} ms",
		"status.loadFailed":  "Load failed: {msg}",
		"status.noModels":    "No models available",
		"status.busy":        "Computing…",
		"msg.noModels":       "No usable models found for machine {msg}.\n\nExpected directory layout:\n<FM1|FM2|BM path>/PxRepository/<model>/\n  Metal.tif, SR.tif, Open.tif\n  <version>/Line/<Top|Bottom>/Layer/MEDIAN/Median_0_1_0.tif\n\nOpen File ▸ Machine Configuration to check the paths, or edit afvi.toml.",
		"msg.noReference":    "Reference directory not found. Set AFVI_REFERENCE_DIR, pass -ref, or configure machine paths in afvi.toml / -config.",
		"common.error":       "Error: {msg}",
		"common.none":        "—",
		"viewer.hintPan":     "Wheel to zoom · middle/right drag to pan · double-click to fit",
	}, map[string]string{
		"app.title":          "AFVI 교육 코어",
		"top.machine":        "기계",
		"top.model":          "모델",
		"top.theme":          "테마",
		"top.lang":           "언어",
		"theme.dark":         "다크(야간)",
		"theme.light":        "클래식(Windows)",
		"menu.file":          "파일(F)",
		"menu.view":          "보기(V)",
		"menu.help":          "도움말(H)",
		"menu.machineConfig": "기계 설정…",
		"menu.rescan":        "모델 다시 검색",
		"menu.exit":          "종료",
		"menu.theme":         "테마",
		"menu.lang":          "언어",
		"menu.fit":           "화면 맞춤",
		"menu.about":         "정보…",
		"about.title":        "Color AFVI Master Teach 정보",
		"about.text":         "AFVI 컬러 검사기 교육 소프트웨어 (네이티브 Win32 스타일 UI)",
		"tab.light":          "조명 및 GV",
		"tab.align":          "정렬",
		"tab.defect":         "결함 검출",
		"status.zoom":        "확대 {pct}%",
		"status.elapsed":     "소요 {ms} ms",
		"status.elapsedNone": "소요 —",
		"status.loading":     "로딩 중…",
		"status.loaded":      "로드됨 {w}×{h} · {ms} ms",
		"status.loadFailed":  "로드 실패: {msg}",
		"status.noModels":    "사용 가능한 모델 없음",
		"status.busy":        "계산 중…",
		"msg.noModels":       "기계 {msg} 에 사용 가능한 모델이 없습니다.\n\n기대 디렉터리 구조:\n<FM1|FM2|BM 경로>/PxRepository/<모델>/\n  Metal.tif, SR.tif, Open.tif\n  <버전>/Line/<Top|Bottom>/Layer/MEDIAN/Median_0_1_0.tif\n\n파일 ▸ 기계 설정 을 열어 경로를 확인하거나 afvi.toml 을 편집하세요.",
		"msg.noReference":    "reference 디렉터리를 찾을 수 없습니다. AFVI_REFERENCE_DIR 을 설정하거나 -ref 로 지정하거나 afvi.toml / -config 에서 기계 경로를 설정하세요.",
		"common.error":       "오류: {msg}",
		"common.none":        "—",
		"viewer.hintPan":     "휠로 확대 · 가운데/오른쪽 드래그로 이동 · 더블클릭으로 화면 맞춤",
	})
}
