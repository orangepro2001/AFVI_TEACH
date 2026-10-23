package app

import (
	"fmt"

	"afvi/teach-imgui/internal/access"
	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/inspect"
	"afvi/teach-imgui/internal/ui"
)

func init() {
	registerPanelFor([]access.Role{access.RoleOperator}, 4, func(a *App) Panel {
		return &OpInspectPanel{a: a, selected: -1}
	}, ProfileTeach)
	Messages(map[string]string{
		"tab.opInspect":           "检测",
		"opInspect.title":         "检测",
		"opInspect.blocked":       "还没有完成对位，不能检测。请先到“对位”页确认对位。",
		"opInspect.toAlign":       "去对位…",
		"opInspect.goldenBusy":    "金板还在生成，请稍候，或到“金板”页查看。",
		"opInspect.toGolden":      "去金板…",
		"opInspect.goldenUnsaved": "金板还没有保存确认，当前检查区域来自自动生成或未保存的金板。",
		"opInspect.goldenEmpty":   "金板的检查区域是空的（没有任何要检查的像素），不能检测。请到“金板”页重新生成或补画遮罩。",
		"opInspect.run":           "检测",
		"opInspect.ok":            "OK 合格",
		"opInspect.ng":            "NG 不合格",
		"opInspect.press":         "按“检测”开始。",
		"opInspect.bright":        "过亮",
		"opInspect.dark":          "过暗",
		"opInspect.noPanel":       "缺陷检出没有初始化。",
		"opInspect.hint":          "只检测对位后的图片；检出框显示在视图上。",
		"opInspect.count":         "缺陷数",
		"opInspect.flagged":       "越界像素",
		"opInspect.elapsed":       "对位耗时",
		"opInspect.channel":       "通道",
		"opInspect.results":       "最大的缺陷",
		"opInspect.emptyList":     "没有检出缺陷。",
		"opInspect.col.mat":       "材质",
		"opInspect.col.pol":       "极性",
		"opInspect.col.area":      "面积",
	}, map[string]string{
		"tab.opInspect":           "Inspect",
		"opInspect.title":         "Inspection",
		"opInspect.blocked":       "Alignment is not confirmed yet, so inspection is disabled. Confirm the alignment on the Align page first.",
		"opInspect.toAlign":       "Go to Align…",
		"opInspect.goldenBusy":    "The golden board is still being built. Wait a moment or open the Golden Board page.",
		"opInspect.toGolden":      "Go to Golden Board…",
		"opInspect.goldenUnsaved": "The golden board is not saved yet; the inspection area is the automatic or unsaved board.",
		"opInspect.goldenEmpty":   "The inspection area of the golden board is empty (no pixel to inspect), so detection is blocked. Regenerate or redraw the mask on the Golden Board page.",
		"opInspect.run":           "Inspect",
		"opInspect.ok":            "OK PASS",
		"opInspect.ng":            "NG FAIL",
		"opInspect.press":         "Press \"Inspect\" to start.",
		"opInspect.bright":        "Bright",
		"opInspect.dark":          "Dark",
		"opInspect.noPanel":       "The defect panel is not available.",
		"opInspect.hint":          "Only the aligned picture is inspected; detected boxes are drawn on the view.",
		"opInspect.count":         "Defects",
		"opInspect.flagged":       "Flagged pixels",
		"opInspect.elapsed":       "Align time",
		"opInspect.channel":       "Channel",
		"opInspect.results":       "Largest defects",
		"opInspect.emptyList":     "No defects detected.",
		"opInspect.col.mat":       "Material",
		"opInspect.col.pol":       "Polarity",
		"opInspect.col.area":      "Area",
	}, map[string]string{
		"tab.opInspect":           "검사",
		"opInspect.title":         "검사",
		"opInspect.blocked":       "정렬이 확인되지 않아 검사할 수 없습니다. 먼저 정렬 페이지에서 정렬을 확인하세요.",
		"opInspect.toAlign":       "정렬로 이동…",
		"opInspect.goldenBusy":    "골든 보드를 만드는 중입니다. 잠시 기다리거나 골든 보드 페이지를 확인하세요.",
		"opInspect.toGolden":      "골든 보드로 이동…",
		"opInspect.goldenUnsaved": "골든 보드가 아직 저장되지 않았습니다. 검사 영역은 자동 생성되었거나 저장되지 않은 보드입니다.",
		"opInspect.goldenEmpty":   "골든 보드의 검사 영역이 비어 있어(검사할 픽셀 없음) 검출할 수 없습니다. 골든 보드 페이지에서 마스크를 다시 생성하거나 그리세요.",
		"opInspect.run":           "검사",
		"opInspect.ok":            "OK 합격",
		"opInspect.ng":            "NG 불합격",
		"opInspect.press":         "\"검사\"를 눌러 시작하세요.",
		"opInspect.bright":        "과밝음",
		"opInspect.dark":          "과어두움",
		"opInspect.noPanel":       "결함 패널을 사용할 수 없습니다.",
		"opInspect.hint":          "정렬된 이미지만 검사하며, 검출 상자는 뷰에 표시됩니다.",
		"opInspect.count":         "결함 수",
		"opInspect.flagged":       "범위 밖 픽셀",
		"opInspect.elapsed":       "정렬 소요",
		"opInspect.channel":       "채널",
		"opInspect.results":       "가장 큰 결함",
		"opInspect.emptyList":     "검출된 결함이 없습니다.",
		"opInspect.col.mat":       "재질",
		"opInspect.col.pol":       "극성",
		"opInspect.col.area":      "면적",
	})
}

// maxOpInspectRows 限制操作员页面列出的缺陷数（rep.Defects 已按面积降序）。
const maxOpInspectRows = 12

// OpInspectPanel 是操作员的引导式检测页：只显示通道、检测按钮、OK/NG 结论和最大的几个缺陷。
// 它包装应用中唯一的 DefectPanel（缺陷检出的状态与图像都在那里）：
// 视图显示缺陷面板的图片和检出框，但参数窗口、学习、探查等工程界面永不出现。
type OpInspectPanel struct {
	a        *App
	active   bool
	msg      string // 预留的状态文字（当前只用于清空后的显示）
	selected int    // 右侧列表中被点击的缺陷索引，-1 = 无
}

// 编译期断言：OpInspectPanel 实现 Panel 和 HidePalette。
var (
	_ Panel                           = (*OpInspectPanel)(nil)
	_ interface{ HidePalette() bool } = (*OpInspectPanel)(nil)
)

// dp 返回应用中的缺陷检出面板（可能为 nil）。
func (p *OpInspectPanel) dp() *DefectPanel { return defectPanelOf(p.a) }

func (p *OpInspectPanel) ID() string { return "op-inspect" }

func (p *OpInspectPanel) Title() string { return T("tab.opInspect") }

// HidePalette 让操作员页面不显示浮动工具调色板。
func (p *OpInspectPanel) HidePalette() bool { return true }

// Activate 让缺陷检出面板把它的图片放进视图（并设置它自己的 active 标志）。
func (p *OpInspectPanel) Activate() {
	p.active = true
	if dp := p.dp(); dp != nil {
		dp.Activate()
	}
}

// Deactivate 释放视图（缺陷面板的拖动/弹窗也在这里收尾）。
func (p *OpInspectPanel) Deactivate() {
	p.active = false
	if dp := p.dp(); dp != nil {
		dp.Deactivate()
	}
}

// ModelLoaded 清空列表选择；缺陷检出面板会收到自己的 ModelLoaded 广播。
func (p *OpInspectPanel) ModelLoaded() { p.selected = -1 }

// LightChanged 清空列表选择；缺陷检出面板会收到自己的 LightChanged 广播。
func (p *OpInspectPanel) LightChanged() { p.selected = -1 }

// Overlay 画出缺陷检出框（教学软件的缺陷面板不画模拟笔画轮廓）。
func (p *OpInspectPanel) Overlay(o *Overlay) {
	if dp := p.dp(); dp != nil {
		dp.Overlay(o)
	}
}

// Pointer 不拥有任何拖动：操作员只能用中键/右键平移，没有工具栏。
func (p *OpInspectPanel) Pointer(ev PointerEvent) bool { return false }

// Hover 复用缺陷面板的状态栏读数。
func (p *OpInspectPanel) Hover(x, y int) string {
	if dp := p.dp(); dp != nil {
		return dp.Hover(x, y)
	}
	return ""
}

// Selftest 不注入演示内容。
func (p *OpInspectPanel) Selftest() {}

func (p *OpInspectPanel) DrawLeft() {
	dp := p.dp()
	if dp == nil {
		Warn(T("opInspect.noPanel"))
		return
	}
	Text(T("opInspect.title"))

	if dp.InspectBlocked() {
		if g := p.a.goldenBoard(); g != nil && !g.Ready() && dp.AlignState() == alignDone {
			Warn(T("opInspect.goldenBusy"))
			if ui.ButtonEx(T("opInspect.toGolden"), "op-togolden", true, true) {
				p.a.ShowPage(goldenPageID)
			}
			return
		}
		if g := p.a.goldenBoard(); g.Empty() && dp.AlignState() == alignDone {
			Warn(T("opInspect.goldenEmpty"))
			if ui.ButtonEx(T("opInspect.toGolden"), "op-togolden", true, true) {
				p.a.ShowPage(goldenPageID)
			}
			return
		}
		Warn(T("opInspect.blocked"))
		if ui.ButtonEx(T("opInspect.toAlign"), "op-toalign", true, true) {
			p.a.ShowPage("op-align")
		}
		return
	}

	if Section(T("opInspect.channel"), "op-channel") {
		labels := []string{"R", "G", "B"}
		chans := []core.CameraChannel{core.ChRed, core.ChGreen, core.ChBlue}
		for i, label := range labels {
			if i > 0 {
				SameLine()
			}
			if ToolButton(label, fmt.Sprintf("op-ch-%d", i), dp.Channel() == chans[i]) {
				dp.SetChannel(chans[i])
				p.selected = -1
			}
		}
	}

	runLabel := T("opInspect.run")
	if ui.ButtonEx(runLabel, "op-run", true, true) {
		dp.RunInspect()
		p.msg = ""
	}

	rep := dp.Report()
	if rep == nil {
		Note(T("opInspect.press"))
		return
	}
	drawVerdictBanner(len(rep.Defects) == 0)
	rows := [][2]string{
		{T("opInspect.count"), fmt.Sprint(len(rep.Defects))},
		{T("opInspect.flagged"), fmt.Sprint(rep.FlaggedPixels)},
	}
	if out := dp.AlignOutcome(); out != nil {
		rows = append(rows, [2]string{T("opInspect.elapsed"), T("status.elapsed", "ms", F(out.ElapsedMs, 1))})
	}
	KVTable("op-kv", rows)
	Note(T("opInspect.hint"))
	if g := p.a.goldenBoard(); g != nil && !g.Confirmed() {
		Warn(T("opInspect.goldenUnsaved"))
	}
}

// drawVerdictBanner draws the big OK/NG verdict badge across the panel width.
func drawVerdictBanner(ok bool) {
	text, level := T("opInspect.ok"), "ok"
	if !ok {
		text, level = T("opInspect.ng"), "ng"
	}
	ui.Badge(text, level).Prop("big", true)
}

func (p *OpInspectPanel) DrawRight() {
	dp := p.dp()
	if dp == nil {
		return
	}
	rep := dp.Report()
	if rep == nil {
		return
	}
	Text(T("opInspect.results"))
	if len(rep.Defects) == 0 {
		Note(T("opInspect.emptyList"))
		return
	}
	defects := rep.Defects
	n := len(defects)
	if n > maxOpInspectRows {
		n = maxOpInspectRows
	}
	headers := []string{T("opInspect.col.mat"), T("opInspect.col.pol"), T("opInspect.col.area")}
	clicked := DataTable("op-defects", headers, n, maxOpInspectRows, func(row, col int) string {
		d := defects[row]
		switch col {
		case 0:
			return d.Material.String()
		case 1:
			if d.Polarity == inspect.Bright {
				return T("opInspect.bright")
			}
			return T("opInspect.dark")
		}
		return fmt.Sprint(d.Area)
	}, p.selected)
	if clicked >= 0 && clicked < n {
		p.selected = clicked
		d := defects[clicked]
		p.a.Viewer.ZoomOn(d.CentroidX, d.CentroidY, 6)
	}
}
