package app

import (
	"testing"

	"afvi/teach-imgui/internal/golden"
)

func TestGoldenEmptyBoardBlocksInspection(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	dp := defectPanelOf(a)
	if g.Empty() {
		t.Fatal("a fresh board is empty")
	}
	g.EraseShape(goldenFull(a.Scene.W, a.Scene.H))
	g.SetSRInset(golden.MaxPad) // the SR layer of the small test model is gone with the largest inset
	pollJobs(t, a)
	if !g.Ready() || !g.Empty() {
		t.Fatalf("after erasing everything: Ready() = %v, Empty() = %v", g.Ready(), g.Empty())
	}
	if !dp.InspectBlocked() {
		t.Error("InspectBlocked() = false although the board inspects nothing")
	}
	g.SetSRInset(golden.DefaultSRInset)
	g.Auto()
	pollJobs(t, a)
	if g.Empty() {
		t.Error("the board is still empty after Auto")
	}
}

func TestGoldenNotBusyWhenSceneGoesAway(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	g.SetPadding(g.Padding() + 1)
	a.Scene = nil // what SelectMachine does before the next model is loaded
	pollJobs(t, a)
	if g.Busy() {
		t.Error("the board stays busy after its scene went away")
	}
}
