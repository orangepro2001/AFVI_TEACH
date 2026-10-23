package app

import (
	"testing"

	"afvi/teach-imgui/internal/ui"
)

// drawIDs records the controls of fn into a frame tree and returns every node id found.
func drawIDs(fn func()) map[string]int {
	b := ui.NewBuilder()
	ui.Use(b)
	b.Begin()
	fn()
	root := b.Finish()
	b.Drop()
	ids := map[string]int{}
	var walk func(n *ui.Node)
	walk = func(n *ui.Node) {
		if n.ID != "" {
			ids[n.ID]++
		}
		for _, c := range n.C {
			walk(c)
		}
	}
	walk(root)
	return ids
}

// TestAlignPagesRecord draws the three align pages into the frame tree: nothing may panic
// and no control id may repeat (the front end addresses controls by id).
func TestAlignPagesRecord(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	for _, id := range []string{"align", "roi", "op-align"} {
		p := a.panelByID(id)
		if p == nil {
			t.Fatalf("panel %s missing", id)
		}
		p.Activate()
		pollJobs(t, a)
		for _, side := range []struct {
			name string
			fn   func()
		}{{"left", p.DrawLeft}, {"right", p.DrawRight}} {
			for want, n := range drawIDs(side.fn) {
				if n > 1 {
					t.Errorf("%s %s: id %q appears %d times", id, side.name, want, n)
				}
			}
		}
		p.Overlay(a.Viewer.Overlay())
		if c, ok := p.(interface{ ClaimsDrag() bool }); ok {
			_ = c.ClaimsDrag()
		}
		p.Deactivate()
	}
}
