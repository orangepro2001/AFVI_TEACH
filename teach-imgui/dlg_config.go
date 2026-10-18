package main

import (
	"strconv"
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/config"
)

// hostStat caches the result of CheckHostPath for one host of the selected machine.
type hostStat struct {
	checked bool
	models  int
	err     error
}

// configDialog is the modal "Machine Configuration" dialog state. All edits happen on
// draft; Cancel discards it.
type configDialog struct {
	needOpen bool
	draft    *config.Config
	sel      int
	stat     [3]hostStat // FM1, FM2, BM of the selected machine
	errMsg   string
}

// cloneConfig deep-copies c so the dialog can edit without touching the live config.
func cloneConfig(c *config.Config) *config.Config {
	if c == nil {
		return &config.Config{}
	}
	out := &config.Config{
		Active:  c.Active,
		Theme:   c.Theme,
		Lang:    c.Lang,
		BaseDir: c.BaseDir,
	}
	out.Machines = make([]config.Machine, len(c.Machines))
	copy(out.Machines, c.Machines)
	return out
}

// OpenConfigDialog prepares a fresh draft from the live config and requests the popup.
func (a *App) OpenConfigDialog() {
	d := &a.cfgDlg
	d.draft = cloneConfig(a.Cfg)
	d.sel = a.machineIdx
	if d.sel < 0 || d.sel >= len(d.draft.Machines) {
		d.sel = 0
	}
	d.errMsg = ""
	d.recheck(a)
	d.needOpen = true
}

// recheck re-evaluates the three host paths of the selected machine.
func (d *configDialog) recheck(a *App) {
	if d.draft == nil || d.sel < 0 || d.sel >= len(d.draft.Machines) {
		return
	}
	m := d.draft.Machines[d.sel]
	paths := [3]string{m.FM1, m.FM2, m.BM}
	for i, host := range [3]string{"FM1", "FM2", "BM"} {
		n, err := CheckHostPath(d.draft, host, paths[i])
		d.stat[i] = hostStat{checked: true, models: n, err: err}
	}
}

// statusText renders one host status line as (text, color).
func statusText(st hostStat) (string, imgui.Vec4) {
	p := CurrentPalette()
	if !st.checked {
		return "—", p.TextDim
	}
	switch {
	case st.err == nil:
		return T("cfg.status.ok", "n", st.models), p.OK
	case st.err == ErrPathNotSet:
		return T("cfg.status.notSet"), p.TextDim
	case st.err == ErrPathNotFound:
		return T("cfg.status.notFound"), p.NG
	case st.err == ErrNoRepository:
		return T("cfg.status.noRepo"), p.NG
	}
	return st.err.Error(), p.NG
}

// paintSelected repaints the just-drawn selectable row with the selection color and
// white text, matching HotText's text origin.
func paintSelected(label string) {
	p := CurrentPalette()
	min, max := imgui.ItemRectMin(), imgui.ItemRectMax()
	dl := imgui.WindowDrawList()
	FillRect(dl, min, max, p.Select)
	pad := imgui.CurrentStyle().FramePadding().X
	ts := imgui.CalcTextSize(label)
	y := min.Y + (max.Y-min.Y-ts.Y)/2
	dl.AddTextVec2(imgui.NewVec2(min.X+pad+2, y), p.SelectText, label)
}

// uniqueName returns a machine name from T("cfg.newName") with the smallest unused N.
func uniqueName(c *config.Config) string {
	used := map[string]bool{}
	for _, m := range c.Machines {
		used[strings.ToLower(m.Name)] = true
	}
	for n := 1; ; n++ {
		name := T("cfg.newName", "n", n)
		if !used[strings.ToLower(name)] {
			return name
		}
	}
}

// uniqueCopyName returns "<name> (2)" (or higher) that is not already used.
func uniqueCopyName(c *config.Config, base string) string {
	used := map[string]bool{}
	for _, m := range c.Machines {
		used[strings.ToLower(m.Name)] = true
	}
	for n := 2; ; n++ {
		name := base + " (" + strconv.Itoa(n) + ")"
		if !used[strings.ToLower(name)] {
			return name
		}
	}
}

// drawConfigDialog draws the modal dialog; it is cheap when the popup is closed.
func (a *App) drawConfigDialog() {
	d := &a.cfgDlg
	if d.needOpen {
		imgui.OpenPopupStr("machine-config")
		d.needOpen = false
	}
	vp := imgui.MainViewport()
	center := imgui.NewVec2(vp.Pos().X+vp.Size().X/2, vp.Pos().Y+vp.Size().Y/2)
	imgui.SetNextWindowPosV(center, imgui.CondAlways, imgui.NewVec2(0.5, 0.5))
	imgui.SetNextWindowSizeV(imgui.NewVec2(640, 430), imgui.CondAlways)
	flags := imgui.WindowFlagsNoTitleBar | imgui.WindowFlagsNoResize | imgui.WindowFlagsNoMove |
		imgui.WindowFlagsNoSavedSettings | imgui.WindowFlagsNoScrollbar | imgui.WindowFlagsNoScrollWithMouse
	// 10 px right padding keeps the group boxes (full available width) off the window frame.
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(10, 0))
	open := imgui.BeginPopupModalV("machine-config", nil, flags)
	imgui.PopStyleVar()
	if !open {
		return
	}
	if d.draft == nil {
		d.draft = cloneConfig(a.Cfg)
		d.sel = 0
	}
	if d.sel < 0 || d.sel >= len(d.draft.Machines) {
		d.sel = 0
	}

	// Chrome: window frame + caption + close box.
	wp, ws := imgui.WindowPos(), imgui.WindowSize()
	dl := imgui.WindowDrawList()
	DrawEdge(dl, wp, imgui.NewVec2(wp.X+ws.X, wp.Y+ws.Y), EdgeWindow)
	capMin := imgui.NewVec2(wp.X+3, wp.Y+3)
	capMax := imgui.NewVec2(wp.X+ws.X-3, wp.Y+23)
	Caption(dl, capMin, capMax, T("cfg.title"), true)

	cancel := false
	imgui.SetCursorScreenPos(imgui.NewVec2(capMax.X-18, capMin.Y+3))
	if ButtonSized("x", "cfg-close", imgui.NewVec2(16, 14), true) {
		cancel = true
	}
	if imgui.IsKeyPressedBool(imgui.KeyEscape) {
		cancel = true
	}

	// Body.
	const bodyY = 32
	const leftX = 10
	const leftW = 176
	const rightX = 196
	const rightW = 434

	// Left column: machine list.
	imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+leftX, wp.Y+bodyY))
	imgui.TextUnformatted(T("cfg.machines"))
	listTop := imgui.CursorScreenPos().Y + imgui.TextLineHeight() + 4
	listMin := imgui.NewVec2(wp.X+leftX, listTop)
	listMax := imgui.NewVec2(wp.X+leftX+leftW, listTop+230)
	FillRect(dl, listMin, listMax, CurrentPalette().Field)
	DrawEdge(dl, listMin, listMax, EdgeSunken)
	imgui.SetCursorScreenPos(imgui.NewVec2(listMin.X+2, listMin.Y+2))
	imgui.PushStyleColorVec4(imgui.ColChildBg, PaletteVec4(CurrentPalette().Field))
	imgui.BeginChildStrV("##cfg-list", imgui.NewVec2(leftW-4, 226), 0, 0)
	for i := range d.draft.Machines {
		name := d.draft.Machines[i].Name
		if imgui.SelectableBoolV(name+"##m"+strconv.Itoa(i), i == d.sel, 0, imgui.NewVec2(0, 0)) && i != d.sel {
			d.sel = i
			d.recheck(a)
		}
		if i == d.sel {
			paintSelected(name)
		} else {
			HotText(name, "")
		}
	}
	imgui.EndChild()
	imgui.PopStyleColor()

	// Left column: [Add][Remove][Duplicate].
	btnY := listMax.Y + 6
	imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+leftX, btnY))
	addLabel, remLabel, dupLabel := T("cfg.add"), T("cfg.remove"), T("cfg.duplicate")
	addW, remW, dupW := ButtonWidth(addLabel), ButtonWidth(remLabel), ButtonWidth(dupLabel)
	if addW < 52 {
		addW = 52
	}
	if remW < 52 {
		remW = 52
	}
	if dupW < 52 {
		dupW = 52
	}
	spacing := imgui.CurrentStyle().ItemSpacing().X
	rowW := addW + remW + dupW + 2*spacing
	// Two rows when the three buttons do not fit in the column.
	if rowW > leftW {
		imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+leftX, btnY))
		if ButtonSized(addLabel, "cfg-add", imgui.NewVec2(addW, 23), true) {
			d.draft.Machines = append(d.draft.Machines, config.Machine{Name: uniqueName(d.draft)})
			d.sel = len(d.draft.Machines) - 1
			d.recheck(a)
		}
		imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+leftX, btnY+27))
		if ButtonSized(remLabel, "cfg-remove", imgui.NewVec2(remW, 23), len(d.draft.Machines) > 1) {
			d.draft.Machines = append(d.draft.Machines[:d.sel], d.draft.Machines[d.sel+1:]...)
			if d.sel >= len(d.draft.Machines) {
				d.sel = len(d.draft.Machines) - 1
			}
			d.recheck(a)
		}
		imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+leftX+remW+spacing, btnY+27))
		if ButtonSized(dupLabel, "cfg-duplicate", imgui.NewVec2(dupW, 23), true) {
			cp := d.draft.Machines[d.sel]
			cp.Name = uniqueCopyName(d.draft, cp.Name)
			d.draft.Machines = append(d.draft.Machines, cp)
			d.sel = len(d.draft.Machines) - 1
			d.recheck(a)
		}
	} else {
		imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+leftX, btnY))
		if ButtonSized(addLabel, "cfg-add", imgui.NewVec2(addW, 23), true) {
			d.draft.Machines = append(d.draft.Machines, config.Machine{Name: uniqueName(d.draft)})
			d.sel = len(d.draft.Machines) - 1
			d.recheck(a)
		}
		imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+leftX+addW+spacing, btnY))
		if ButtonSized(remLabel, "cfg-remove", imgui.NewVec2(remW, 23), len(d.draft.Machines) > 1) {
			d.draft.Machines = append(d.draft.Machines[:d.sel], d.draft.Machines[d.sel+1:]...)
			if d.sel >= len(d.draft.Machines) {
				d.sel = len(d.draft.Machines) - 1
			}
			d.recheck(a)
		}
		imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+leftX+addW+remW+2*spacing, btnY))
		if ButtonSized(dupLabel, "cfg-duplicate", imgui.NewVec2(dupW, 23), true) {
			cp := d.draft.Machines[d.sel]
			cp.Name = uniqueCopyName(d.draft, cp.Name)
			d.draft.Machines = append(d.draft.Machines, cp)
			d.sel = len(d.draft.Machines) - 1
			d.recheck(a)
		}
	}

	// Right column.
	imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+rightX, wp.Y+bodyY))
	imgui.PushItemWidth(rightW)
	m := &d.draft.Machines[d.sel]
	GroupBox(T("cfg.machine"), "cfg-machine", func() {
		InputText(T("cfg.name"), "cfg-name", &m.Name, "")
		InputText(T("cfg.address"), "cfg-addr", &m.Address, T("cfg.hintAddress"))
	})
	GroupBox(T("cfg.paths"), "cfg-paths", func() {
		hostInput := func(label, id string, buf *string, idx int) {
			if InputText(label, id, buf, T("cfg.hintPath")) {
				d.stat[idx].checked = false
			}
			imgui.SetCursorPosX(imgui.CursorPosX() + labelWidth)
			txt, col := statusText(d.stat[idx])
			imgui.TextColored(col, txt)
		}
		hostInput(T("cfg.fm1"), "cfg-fm1", &m.FM1, 0)
		hostInput(T("cfg.fm2"), "cfg-fm2", &m.FM2, 1)
		hostInput(T("cfg.bm"), "cfg-bm", &m.BM, 2)
		// [Check] at the bottom-right of the group.
		checkLabel := T("cfg.check")
		cw := ButtonWidth(checkLabel)
		avail := imgui.ContentRegionAvail().X
		if avail > cw {
			imgui.SetCursorPosX(imgui.CursorPosX() + avail - cw)
		}
		if ButtonSized(checkLabel, "cfg-check", imgui.NewVec2(cw, 23), true) {
			d.recheck(a)
		}
	})
	imgui.PopItemWidth()

	// Bottom area.
	const bottomY = 430 - 62
	imgui.SetCursorScreenPos(imgui.NewVec2(wp.X+leftX, wp.Y+bottomY))
	Note(T("cfg.file") + a.CfgPath)
	verr := d.draft.Validate()
	if verr != nil {
		imgui.TextColored(CurrentPalette().NG, verr.Error())
	} else if d.errMsg != "" {
		imgui.TextColored(CurrentPalette().NG, d.errMsg)
	}

	okLabel, cancelLabel, applyLabel := T("cfg.ok"), T("cfg.cancel"), T("cfg.apply")
	okW, cancelW, applyW := ButtonWidth(okLabel), ButtonWidth(cancelLabel), ButtonWidth(applyLabel)
	if okW < 75 {
		okW = 75
	}
	if cancelW < 75 {
		cancelW = 75
	}
	if applyW < 75 {
		applyW = 75
	}
	btnRowW := okW + cancelW + applyW + 2*spacing
	btnX := wp.X + ws.X - 10 - btnRowW
	btnY2 := wp.Y + 430 - 30
	imgui.SetCursorScreenPos(imgui.NewVec2(btnX, btnY2))
	enabled := verr == nil
	if ButtonSized(okLabel, "cfg-ok", imgui.NewVec2(okW, 23), enabled) {
		d.draft.Active = d.draft.Machines[d.sel].Name
		d.draft.Theme = a.theme
		d.draft.Lang = string(CurrentLang())
		if err := a.ApplyConfig(cloneConfig(d.draft)); err != nil {
			d.errMsg = err.Error()
		} else {
			a.SetHint(T("cfg.applied"))
			imgui.CloseCurrentPopup()
		}
	}
	imgui.SetCursorScreenPos(imgui.NewVec2(btnX+okW+spacing, btnY2))
	if ButtonSized(cancelLabel, "cfg-cancel", imgui.NewVec2(cancelW, 23), true) {
		cancel = true
	}
	imgui.SetCursorScreenPos(imgui.NewVec2(btnX+okW+cancelW+2*spacing, btnY2))
	if ButtonSized(applyLabel, "cfg-apply", imgui.NewVec2(applyW, 23), enabled) {
		d.draft.Active = d.draft.Machines[d.sel].Name
		d.draft.Theme = a.theme
		d.draft.Lang = string(CurrentLang())
		if err := a.ApplyConfig(cloneConfig(d.draft)); err != nil {
			d.errMsg = err.Error()
		} else {
			a.SetHint(T("cfg.applied"))
			d.recheck(a)
		}
	}

	if cancel {
		imgui.CloseCurrentPopup()
	}
	imgui.EndPopup()
}

func init() {
	Messages(
		map[string]string{
			"cfg.open":            "机台配置…",
			"cfg.title":           "机台配置",
			"cfg.machines":        "机台",
			"cfg.add":             "添加",
			"cfg.remove":          "删除",
			"cfg.duplicate":       "复制",
			"cfg.machine":         "机台",
			"cfg.name":            "名称",
			"cfg.address":         "地址",
			"cfg.paths":           "数据路径",
			"cfg.fm1":             "FM1 路径",
			"cfg.fm2":             "FM2 路径",
			"cfg.bm":              "BM 路径",
			"cfg.check":           "检查",
			"cfg.ok":              "确定",
			"cfg.cancel":          "取消",
			"cfg.apply":           "应用",
			"cfg.file":            "配置文件：",
			"cfg.hintAddress":     "IP 或主机名",
			"cfg.hintPath":        "目录，内含 PxRepository",
			"cfg.newName":         "机台 {n}",
			"cfg.status.ok":       "正常 — {n} 个型号",
			"cfg.status.notSet":   "未设置",
			"cfg.status.notFound": "路径不存在",
			"cfg.status.noRepo":   "缺少 PxRepository",
			"cfg.applied":         "机台配置已保存",
		},
		map[string]string{
			"cfg.open":            "Machines…",
			"cfg.title":           "Machine Configuration",
			"cfg.machines":        "Machines",
			"cfg.add":             "Add",
			"cfg.remove":          "Remove",
			"cfg.duplicate":       "Duplicate",
			"cfg.machine":         "Machine",
			"cfg.name":            "Name",
			"cfg.address":         "Address",
			"cfg.paths":           "Data paths",
			"cfg.fm1":             "FM1 path",
			"cfg.fm2":             "FM2 path",
			"cfg.bm":              "BM path",
			"cfg.check":           "Check",
			"cfg.ok":              "OK",
			"cfg.cancel":          "Cancel",
			"cfg.apply":           "Apply",
			"cfg.file":            "Config file: ",
			"cfg.hintAddress":     "IP or host name",
			"cfg.hintPath":        "directory containing PxRepository",
			"cfg.newName":         "Machine {n}",
			"cfg.status.ok":       "OK — {n} models",
			"cfg.status.notSet":   "Not set",
			"cfg.status.notFound": "Path not found",
			"cfg.status.noRepo":   "PxRepository missing",
			"cfg.applied":         "Machine configuration saved",
		},
		map[string]string{
			"cfg.open":            "기계 설정…",
			"cfg.title":           "기계 설정",
			"cfg.machines":        "기계",
			"cfg.add":             "추가",
			"cfg.remove":          "삭제",
			"cfg.duplicate":       "복제",
			"cfg.machine":         "기계",
			"cfg.name":            "이름",
			"cfg.address":         "주소",
			"cfg.paths":           "데이터 경로",
			"cfg.fm1":             "FM1 경로",
			"cfg.fm2":             "FM2 경로",
			"cfg.bm":              "BM 경로",
			"cfg.check":           "확인",
			"cfg.ok":              "확인",
			"cfg.cancel":          "취소",
			"cfg.apply":           "적용",
			"cfg.file":            "설정 파일: ",
			"cfg.hintAddress":     "IP 또는 호스트 이름",
			"cfg.hintPath":        "PxRepository 가 있는 디렉터리",
			"cfg.newName":         "기계 {n}",
			"cfg.status.ok":       "정상 — 모델 {n}개",
			"cfg.status.notSet":   "설정되지 않음",
			"cfg.status.notFound": "경로를 찾을 수 없음",
			"cfg.status.noRepo":   "PxRepository 없음",
			"cfg.applied":         "기계 설정이 저장되었습니다",
		},
	)
}
