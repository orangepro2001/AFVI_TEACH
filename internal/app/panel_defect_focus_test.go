package app

import (
	"testing"

	"afvi/teach-imgui/internal/golden"
	"afvi/teach-imgui/internal/inspect"
)

// focusTestReport builds a report with two Metal (bright, dark) and three SR defects
// (two bright, one dark).
func focusTestReport() *inspect.Report {
	return &inspect.Report{Defects: []inspect.Defect{
		{Material: inspect.Metal, Polarity: inspect.Bright},
		{Material: inspect.Metal, Polarity: inspect.Dark},
		{Material: inspect.SR, Polarity: inspect.Bright},
		{Material: inspect.SR, Polarity: inspect.Bright},
		{Material: inspect.SR, Polarity: inspect.Dark},
	}}
}

// focusTestPanel returns a detect-mode panel with no app attached; it is enough for the
// pure state methods (SubTabs, SelectSubTab, focusRows, focusTint) that must not touch
// the jobs or the viewer.
func focusTestPanel() *DefectPanel {
	return newDefectPanel(nil, modeDetect)
}

func TestFocusSubTabsAuthorNil(t *testing.T) {
	p := newDefectPanel(nil, modeAuthor)
	items, active := p.SubTabs()
	if items != nil {
		t.Errorf("author SubTabs() = %+v, want nil", items)
	}
	if active != 0 {
		t.Errorf("author SubTabs() active = %d, want 0", active)
	}
}

func TestFocusSubTabsLabels(t *testing.T) {
	p := focusTestPanel()
	items, active := p.SubTabs()
	if len(items) != 2 {
		t.Fatalf("%d sub-tabs, want 2", len(items))
	}
	if items[0].Label != "Metal" || items[1].Label != "SR" {
		t.Errorf("labels without a report = %q, %q, want Metal, SR", items[0].Label, items[1].Label)
	}
	if active != 0 {
		t.Errorf("active = %d, want 0", active)
	}
	if items[0].Color != rgba(255, 0, 255, 255) {
		t.Errorf("Metal chip colour %08x, want magenta", items[0].Color)
	}
	if items[1].Color != rgba(0, 200, 0, 255) {
		t.Errorf("SR chip colour %08x, want green", items[1].Color)
	}

	p.lastReport = focusTestReport()
	items, active = p.SubTabs()
	if items[0].Label != "Metal (2)" {
		t.Errorf("Metal label = %q, want \"Metal (2)\"", items[0].Label)
	}
	if items[1].Label != "SR (3)" {
		t.Errorf("SR label = %q, want \"SR (3)\"", items[1].Label)
	}
	if active != 0 {
		t.Errorf("active = %d, want 0", active)
	}

	p.focus.target = golden.TargetSR
	if _, active = p.SubTabs(); active != 1 {
		t.Errorf("active with SR target = %d, want 1", active)
	}
}

func TestFocusSelectSubTab(t *testing.T) {
	p := focusTestPanel()
	if p.focus.target != golden.TargetMetal {
		t.Fatalf("default target %v, want metal", p.focus.target)
	}
	p.lastReport = focusTestReport()

	metalRows := len(p.focusRows())
	if metalRows != 2 {
		t.Fatalf("Metal rows = %d, want 2", metalRows)
	}

	p.SelectSubTab(1)
	if p.focus.target != golden.TargetSR {
		t.Errorf("target after SelectSubTab(1) = %v, want sr", p.focus.target)
	}
	srRows := len(p.focusRows())
	if srRows != 3 {
		t.Errorf("SR rows = %d, want 3", srRows)
	}

	p.SelectSubTab(0)
	if p.focus.target != golden.TargetMetal {
		t.Errorf("target after SelectSubTab(0) = %v, want metal", p.focus.target)
	}
	if n := len(p.focusRows()); n != 2 {
		t.Errorf("Metal rows after switching back = %d, want 2", n)
	}

	// Out-of-range and same-index selections change nothing.
	p.SelectSubTab(0)
	p.SelectSubTab(-1)
	p.SelectSubTab(2)
	if p.focus.target != golden.TargetMetal {
		t.Errorf("target after no-op selections = %v, want metal", p.focus.target)
	}
}

func TestFocusSelectSubTabAuthorNoop(t *testing.T) {
	p := newDefectPanel(nil, modeAuthor)
	p.SelectSubTab(1)
	if p.focus.target != golden.TargetMetal {
		t.Errorf("author SelectSubTab changed target to %v", p.focus.target)
	}
}

func TestFocusRowsFollowsFilter(t *testing.T) {
	p := focusTestPanel()
	p.lastReport = focusTestReport()

	p.focus.target = golden.TargetMetal
	p.focus.polarity = focusAll
	if n := len(p.focusRows()); n != 2 {
		t.Errorf("Metal / all rows = %d, want 2", n)
	}
	p.focus.polarity = focusBright
	if n := len(p.focusRows()); n != 1 {
		t.Errorf("Metal / bright rows = %d, want 1", n)
	}
	p.focus.polarity = focusDark
	if n := len(p.focusRows()); n != 1 {
		t.Errorf("Metal / dark rows = %d, want 1", n)
	}

	p.focus.target = golden.TargetSR
	p.focus.polarity = focusAll
	if n := len(p.focusRows()); n != 3 {
		t.Errorf("SR / all rows = %d, want 3", n)
	}
	p.focus.polarity = focusBright
	if n := len(p.focusRows()); n != 2 {
		t.Errorf("SR / bright rows = %d, want 2", n)
	}
	p.focus.polarity = focusDark
	if n := len(p.focusRows()); n != 1 {
		t.Errorf("SR / dark rows = %d, want 1", n)
	}
}

func TestFocusRowsCache(t *testing.T) {
	p := focusTestPanel()
	p.lastReport = focusTestReport()

	first := p.focusRows()
	second := p.focusRows()
	if len(first) == 0 || &first[0] != &second[0] {
		t.Errorf("focusRows() did not return the cached slice for an unchanged report")
	}

	p.lastReport = focusTestReport()
	third := p.focusRows()
	if len(third) != len(first) {
		t.Errorf("rows for a new report = %d, want %d", len(third), len(first))
	}
	if len(third) > 0 && &third[0] == &first[0] {
		t.Errorf("focusRows() returned the stale cache for a new report pointer")
	}

	// A changed focus recomputes even with the same report.
	before := p.focusRows()
	p.focus.polarity = focusBright
	after := p.focusRows()
	if len(after) == len(before) && len(after) > 0 && &after[0] == &before[0] {
		t.Errorf("focusRows() kept the cache after a polarity change")
	}
}

func TestFocusRowsNilReport(t *testing.T) {
	p := focusTestPanel()
	if rows := p.focusRows(); rows != nil {
		t.Errorf("focusRows() with no report = %v, want nil", rows)
	}
}

func TestFocusTintKeyOffHasNoRev(t *testing.T) {
	p := focusTestPanel()
	if k := p.focusTintKey(); k != (tintKey{on: false, target: golden.TargetMetal, rev: 0}) {
		t.Errorf("focusTintKey() = %+v, want {false metal 0}", k)
	}
	p.focus.target = golden.TargetSR
	if k := p.focusTintKey(); k != (tintKey{on: false, target: golden.TargetSR, rev: 0}) {
		t.Errorf("focusTintKey() with SR = %+v, want {false sr 0}", k)
	}
}

func TestFocusTintOff(t *testing.T) {
	// No board: no tint.
	p := focusTestPanel()
	if p.focusTint().on() {
		t.Errorf("focusTint() without a board is on")
	}
	if k := p.focusTintKey(); k.on || k.target != golden.TargetMetal {
		t.Errorf("focusTintKey() without a board = %+v, want off and metal", k)
	}

	// A board that is not ready still gives no tint.
	a := &App{}
	p = newDefectPanel(a, modeDetect)
	if p.focusTint().on() {
		t.Errorf("focusTint() with an unready board is on")
	}
	if k := p.focusTintKey(); k.on {
		t.Errorf("focusTintKey() with an unready board = %+v, want off", k)
	}

	// Author mode never tints.
	p = newDefectPanel(nil, modeAuthor)
	if p.focusTint().on() {
		t.Errorf("focusTint() in author mode is on")
	}
}
