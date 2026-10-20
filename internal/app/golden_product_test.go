package app

import (
	"testing"

	"afvi/teach-imgui/internal/core"
)

func TestGoldenFollowsProductChange(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	if g == nil || !g.Ready() {
		t.Fatal("golden board is not ready")
	}
	rev := g.Rev()
	opts := core.DefaultProductOptions()
	opts.Seed = 7
	a.ApplyProduct(opts)
	pollJobs(t, a)
	if a.Scene.Product.Seed != 7 {
		t.Fatalf("product seed = %d, want 7", a.Scene.Product.Seed)
	}
	if !g.Ready() {
		t.Fatal("the board did not rebuild after the product change")
	}
	if g.Rev() == rev {
		t.Errorf("board revision %d did not change", rev)
	}
	if g.scene != a.Scene {
		t.Error("the board still holds the old scene")
	}
	// A request made after the change must still run (the queue is not stuck).
	g.SetPadding(g.Padding() + 1)
	pollJobs(t, a)
	if g.Busy() {
		t.Error("the board is still busy after the request")
	}
}
