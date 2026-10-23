package app

import (
	"strconv"
	"strings"

	"afvi/teach-imgui/internal/access"
	"afvi/teach-imgui/internal/config"
	"afvi/teach-imgui/internal/ui"
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
	needOpen bool // the dialog is open (drawn every frame until closed)
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
	if a.role != access.RoleAdmin {
		return
	}
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

// statusText renders one host status line as (text, tone).
func statusText(st hostStat) (string, string) {
	if !st.checked {
		return "—", "dim"
	}
	switch {
	case st.err == nil:
		return T("cfg.status.ok", "n", st.models), "good"
	case st.err == ErrPathNotSet:
		return T("cfg.status.notSet"), "dim"
	case st.err == ErrPathNotFound:
		return T("cfg.status.notFound"), "bad"
	case st.err == ErrNoRepository:
		return T("cfg.status.noRepo"), "bad"
	}
	return st.err.Error(), "bad"
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

// buildConfigDialog records the modal dialog while it is open; it is free when closed.
func (a *App) buildConfigDialog() {
	if a.role != access.RoleAdmin {
		return
	}
	d := &a.cfgDlg
	if !d.needOpen {
		return
	}
	if d.draft == nil {
		d.draft = cloneConfig(a.Cfg)
		d.sel = 0
	}
	if d.sel < 0 || d.sel >= len(d.draft.Machines) {
		d.sel = 0
	}
	closeDlg := func() {
		d.needOpen = false
		d.draft = nil
		d.errMsg = ""
	}

	cancel := false
	if ui.Dialog(T("cfg.title"), "cfg", 640, func() {
		// Machine list with Add / Remove / Duplicate.
		GroupBox(T("cfg.machines"), "cfg-machines", func() {
			if len(d.draft.Machines) > 0 {
				names := make([]string, len(d.draft.Machines))
				for i := range d.draft.Machines {
					names[i] = d.draft.Machines[i].Name
				}
				clicked := DataTable("cfg-list", []string{T("cfg.machines")}, len(names), 6, func(row, col int) string {
					return names[row]
				}, d.sel)
				if clicked >= 0 && clicked < len(names) && clicked != d.sel {
					d.sel = clicked
					d.recheck(a)
				}
			}
			if Button(T("cfg.add"), "cfg-add") {
				d.draft.Machines = append(d.draft.Machines, config.Machine{Name: uniqueName(d.draft)})
				d.sel = len(d.draft.Machines) - 1
				d.recheck(a)
			}
			SameLine()
			if ButtonOn(T("cfg.remove"), "cfg-remove", len(d.draft.Machines) > 1) {
				d.draft.Machines = append(d.draft.Machines[:d.sel], d.draft.Machines[d.sel+1:]...)
				if d.sel >= len(d.draft.Machines) {
					d.sel = len(d.draft.Machines) - 1
				}
				d.recheck(a)
			}
			SameLine()
			if ButtonOn(T("cfg.duplicate"), "cfg-duplicate", len(d.draft.Machines) > 0) {
				cp := d.draft.Machines[d.sel]
				cp.Name = uniqueCopyName(d.draft, cp.Name)
				d.draft.Machines = append(d.draft.Machines, cp)
				d.sel = len(d.draft.Machines) - 1
				d.recheck(a)
			}
		})
		if len(d.draft.Machines) == 0 {
			return
		}

		// Selected machine.
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
				txt, tone := statusText(d.stat[idx])
				ui.TextTone(txt, tone)
			}
			hostInput(T("cfg.fm1"), "cfg-fm1", &m.FM1, 0)
			hostInput(T("cfg.fm2"), "cfg-fm2", &m.FM2, 1)
			hostInput(T("cfg.bm"), "cfg-bm", &m.BM, 2)
			if Button(T("cfg.check"), "cfg-check") {
				d.recheck(a)
			}
		})

		Note(T("cfg.file") + a.CfgPath)
		verr := d.draft.Validate()
		if verr != nil {
			ui.TextTone(verr.Error(), "bad")
		} else if d.errMsg != "" {
			ui.TextTone(d.errMsg, "bad")
		}

		apply := func() bool {
			d.draft.Active = d.draft.Machines[d.sel].Name
			d.draft.Theme = a.theme
			d.draft.Lang = string(CurrentLang())
			if err := a.ApplyConfig(cloneConfig(d.draft)); err != nil {
				d.errMsg = err.Error()
				return false
			}
			a.SetHint(T("cfg.applied"))
			return true
		}
		enabled := verr == nil
		if ui.ButtonEx(T("cfg.ok"), "cfg-ok", enabled, true) {
			if apply() {
				cancel = true
			}
		}
		SameLine()
		if Button(T("cfg.cancel"), "cfg-cancel") {
			cancel = true
		}
		SameLine()
		if ButtonOn(T("cfg.apply"), "cfg-apply", enabled) {
			if apply() {
				d.recheck(a)
			}
		}
	}) {
		cancel = true
	}
	if ui.KeyPressed("Escape") {
		cancel = true
	}
	if cancel {
		closeDlg()
	}
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
