package app

import (
	"strconv"

	"afvi/teach-imgui/internal/golden"
	"afvi/teach-imgui/internal/inspect"
)

// focusCache caches the indices of the defects that pass the sub-tab filter.
type focusCache struct {
	rep *inspect.Report
	f   defectFocus
	idx []int
	ok  bool
}

// SubTabs implements SubTabPanel: the Metal / SR strip of the detect page.
func (p *DefectPanel) SubTabs() ([]SubTabItem, int) {
	if p == nil || p.mode != modeDetect {
		return nil, 0
	}
	items := []SubTabItem{
		{Label: focusTabLabel("Metal", p.lastReport, inspect.Metal), Color: rgba(255, 0, 255, 255)},
		{Label: focusTabLabel("SR", p.lastReport, inspect.SR), Color: rgba(0, 200, 0, 255)},
	}
	return items, int(p.focus.target)
}

// focusTabLabel returns "name" or "name (n)" with n = defects of material m.
func focusTabLabel(name string, r *inspect.Report, m inspect.Material) string {
	if r == nil {
		return name
	}
	all, _, _ := focusCounts(r, m)
	return name + " (" + strconv.Itoa(all) + ")"
}

// SelectSubTab implements SubTabPanel: switch the sub-tab to item i.
func (p *DefectPanel) SelectSubTab(i int) {
	if p == nil || p.mode != modeDetect || i < 0 || i > 1 {
		return
	}
	t := golden.Target(i)
	if t == p.focus.target {
		return
	}
	p.focus.target = t
	p.focusCache = focusCache{}
	if p.um.frame != nil {
		p.um.frame.Reset()
	}
	if p.a != nil {
		p.a.SetHint(T("defect.focus.hint", "name", userMaskTargetName(t)))
	}
	if p.test != nil {
		p.requestView()
	}
}

// SubTabNote implements SubTabPanel: the dim status text right of the strip.
func (p *DefectPanel) SubTabNote() string {
	if p == nil {
		return ""
	}
	name := userMaskTargetName(p.focus.target)
	if p.focusTint().on() {
		key := "defect.focus.color.metal"
		count := 0
		if p.focus.target == golden.TargetSR {
			key = "defect.focus.color.sr"
		}
		if g := p.a.goldenBoard(); g != nil {
			if d := g.Derived(); d != nil {
				if p.focus.target == golden.TargetSR {
					count = d.srPx
				} else {
					count = d.metalPx
				}
			}
		}
		return T("defect.focus.note.on", "name", name, "color", T(key), "px", strconv.Itoa(count))
	}
	return T("defect.focus.note.off", "name", name)
}

// focusTint returns the tint the viewer picture must carry for the selected sub-tab.
func (p *DefectPanel) focusTint() tintSpec {
	if p == nil || p.a == nil || p.mode != modeDetect || p.viewKind != "test" {
		return tintSpec{}
	}
	g := p.a.goldenBoard()
	if g == nil || !g.Ready() || g.Empty() || !p.userMaskDrawable() {
		return tintSpec{}
	}
	return newTintSpec(g.Derived(), p.focus.target)
}

// focusTintKey identifies the tint of the picture that should be shown now.
func (p *DefectPanel) focusTintKey() tintKey {
	if p == nil {
		return tintKey{}
	}
	k := tintKey{on: p.focusTint().on(), target: p.focus.target}
	// rev is only meaningful when the tint is on; off keys keep rev 0.
	if k.on && p.a != nil {
		if g := p.a.goldenBoard(); g != nil {
			k.rev = g.Rev()
		}
	}
	return k
}

// focusRows returns the indices into p.lastReport.Defects that pass the sub-tab filter,
// cached so the frame loop does not allocate.
func (p *DefectPanel) focusRows() []int {
	if p == nil {
		return nil
	}
	c := &p.focusCache
	if c.ok && c.rep == p.lastReport && c.f == p.focus {
		return c.idx
	}
	c.rep = p.lastReport
	c.f = p.focus
	c.idx = p.focus.indices(p.lastReport)
	c.ok = true
	return c.idx
}

// focusPolName returns the translated name of a polarity filter.
func focusPolName(pol focusPolarity) string {
	switch pol {
	case focusBright:
		return T("defect.polarity.bright")
	case focusDark:
		return T("defect.polarity.dark")
	}
	return T("defect.filter.pol.all")
}

// drawFocusFilter draws the polarity filter row of the result column.
func (p *DefectPanel) drawFocusFilter(r *inspect.Report) {
	all, bright, dark := focusCounts(r, p.focus.material())
	Text(T("defect.filter.label"))
	SameLine()

	entries := []struct {
		id    string
		label string
		pol   focusPolarity
		n     int
	}{
		{"defect-flt-all", T("defect.filter.all", "n", strconv.Itoa(all)), focusAll, all},
		{"defect-flt-bright", T("defect.filter.bright", "n", strconv.Itoa(bright)), focusBright, bright},
		{"defect-flt-dark", T("defect.filter.dark", "n", strconv.Itoa(dark)), focusDark, dark},
	}
	for i, e := range entries {
		if i > 0 {
			WrapSameLine(e.label)
		}
		if ToolButton(e.label, e.id, p.focus.polarity == e.pol) {
			p.focus.polarity = e.pol
		}
	}
	name := userMaskTargetName(p.focus.target)
	Note(T("defect.filter.showing", "name", name, "pol", focusPolName(p.focus.polarity), "n", strconv.Itoa(len(p.focusRows()))))
}
