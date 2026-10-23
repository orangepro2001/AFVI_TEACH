package ui

import "testing"

func run(b *Builder, cmds []Cmd, draw func()) *Node {
	Use(b)
	b.Feed(cmds)
	b.Begin()
	draw()
	n := b.Finish()
	b.Drop()
	return n
}

func TestSliderAppliesSetAndClamps(t *testing.T) {
	b := NewBuilder()
	v := 1.0
	run(b, []Cmd{{ID: "g", Op: "set", V: 9.0}}, func() {
		if !SliderF64("Gain", "g", &v, 0, 5, "%.1f") {
			t.Fatal("changed expected")
		}
	})
	if v != 5 {
		t.Fatalf("v=%v want clamp to 5", v)
	}
}

func TestButtonClickIsConsumedOnce(t *testing.T) {
	b := NewBuilder()
	n := 0
	draw := func() {
		if Button("Go", "go") {
			n++
		}
	}
	run(b, []Cmd{{ID: "go", Op: "click"}}, draw)
	run(b, nil, draw)
	if n != 1 {
		t.Fatalf("clicks=%d", n)
	}
}

func TestSameLineBecomesRow(t *testing.T) {
	b := NewBuilder()
	root := run(b, nil, func() {
		Button("A", "a")
		SameLine()
		Button("B", "b")
		SameLine()
		Button("C", "c")
		Button("D", "d")
	})
	if len(root.C) != 2 || root.C[0].K != KRow || len(root.C[0].C) != 3 || root.C[1].ID != "d" {
		t.Fatalf("bad tree: %+v", root.C)
	}
}

func TestSectionCollectsBodyAndToggles(t *testing.T) {
	b := NewBuilder()
	draw := func() {
		if Section("S1", "s1") {
			Text("in1")
		}
		if Section("S2", "s2") {
			Text("in2")
		}
	}
	root := run(b, nil, draw)
	if len(root.C) != 2 || len(root.C[0].C) != 1 || root.C[1].C[0].L != "in2" {
		t.Fatalf("bad sections: %+v", root.C)
	}
	run(b, []Cmd{{ID: "sec-s1", Op: "toggle"}}, draw)
	root = run(b, nil, draw)
	if root.C[0].V != false || len(root.C[0].C) != 0 {
		t.Fatalf("section 1 should be closed: %+v", root.C[0])
	}
}

func TestGroupBoxNestsAndComboRejectsOutOfRange(t *testing.T) {
	b := NewBuilder()
	i := 0
	root := run(b, []Cmd{{ID: "c", Op: "set", V: 7.0}}, func() {
		GroupBox("G", "g", func() {
			Combo("C", "c", &i, []string{"a", "b"})
		})
	})
	if i != 0 || len(root.C[0].C) != 1 {
		t.Fatalf("i=%d tree=%+v", i, root.C)
	}
}

func TestTabStripReturnsClickedIndex(t *testing.T) {
	b := NewBuilder()
	got := -2
	run(b, []Cmd{{ID: "tabs", Op: "set", V: 2.0}}, func() { got = TabStrip("tabs", []string{"a", "b", "c"}, 0) })
	if got != 2 {
		t.Fatalf("got %d", got)
	}
}
