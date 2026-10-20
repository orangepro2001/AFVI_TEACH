package app

import (
	"reflect"
	"testing"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
)

// goldenUserRect is a rectangle shape from corners in any order.
func goldenUserRect(x0, y0, x1, y1 float64) golden.Shape {
	return golden.Shape{Kind: golden.ShapeRect, P0: golden.Pt{X: x0, Y: y0}, P1: golden.Pt{X: x1, Y: y1}}
}

// goldenUserBothWindow searches the current derived masks for a window that contains
// both an inspected Metal pixel and an inspected SR pixel. When none is found the
// whole image is returned.
func goldenUserBothWindow(t *testing.T, a *App) golden.Shape {
	t.Helper()
	g := a.golden
	d := g.Derived()
	w, h := a.Scene.W, a.Scene.H
	full := golden.Shape{Kind: golden.ShapeRect, P0: golden.Pt{X: 0, Y: 0}, P1: golden.Pt{X: float64(w), Y: float64(h)}}
	if d == nil || d.metalInspect == nil || d.sr == nil {
		return full
	}
	// Walk small candidate windows and pick the smallest that holds a metal and an SR pixel.
	best := full
	bestArea := w * h
	for y0 := 0; y0 < h; y0 += 4 {
		for x0 := 0; x0 < w; x0 += 4 {
			for size := 8; size <= w && size <= h; size += 4 {
				x1 := x0 + size
				y1 := y0 + size
				if x1 > w || y1 > h {
					break
				}
				hasMetal, hasSR := false, false
				for y := y0; y < y1 && !(hasMetal && hasSR); y++ {
					for x := x0; x < x1; x++ {
						if d.metalInspect.At(x, y) {
							hasMetal = true
						}
						if d.sr.At(x, y) {
							hasSR = true
						}
						if hasMetal && hasSR {
							break
						}
					}
				}
				if hasMetal && hasSR && size*size < bestArea {
					best = goldenUserRect(float64(x0), float64(y0), float64(x1), float64(y1))
					bestArea = size * size
					break
				}
			}
		}
	}
	return best
}

// goldenUserInside reports whether the pixel centre lies inside the shape.
func goldenUserInside(sh golden.Shape, x, y int) bool {
	return sh.Contains(float64(x)+0.5, float64(y)+0.5)
}

// goldenUserCountInside counts set pixels of maybe that lie inside sh on a w*h grid.
func goldenUserCountInside(maybe *golden.Mask, sh golden.Shape, w, h int) int {
	if maybe == nil {
		return 0
	}
	n := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if maybe.At(x, y) && goldenUserInside(sh, x, y) {
				n++
			}
		}
	}
	return n
}

// goldenUserAnyInside reports whether any pixel of maybe lies inside sh.
func goldenUserAnyInside(maybe *golden.Mask, sh golden.Shape, w, h int) bool {
	return goldenUserCountInside(maybe, sh, w, h) > 0
}

func TestGoldenUserMetalOnlyRemovesMetal(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	w, h := a.Scene.W, a.Scene.H
	shape := goldenUserBothWindow(t, a)

	oldD := g.Derived()
	oldInspect := oldD.metalInspect.Clone()
	oldSR := oldD.sr.Clone()
	oldMetalPx := oldD.metalPx
	oldSRPx := oldD.srPx
	if !goldenUserAnyInside(oldInspect, shape, w, h) {
		t.Fatalf("chosen window holds no inspected Metal pixel: %+v", shape)
	}

	g.AddUserRegion(golden.TargetMetal, shape)
	pollJobs(t, a)

	d := g.Derived()
	if d == nil {
		t.Fatal("Derived() is nil after AddUserRegion")
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			inside := goldenUserInside(shape, x, y)
			oldSet := oldInspect.At(x, y)
			newSet := d.metalInspect.At(x, y)
			if inside && oldSet && newSet {
				t.Fatalf("pixel (%d,%d) inside the region is still inspected Metal", x, y)
			}
			if !inside && oldSet != newSet {
				t.Fatalf("pixel (%d,%d) outside the region changed: %v -> %v", x, y, oldSet, newSet)
			}
		}
	}
	if !d.sr.Equal(oldSR) {
		t.Error("the SR mask changed on a Metal region")
	}
	if d.srPx != oldSRPx {
		t.Errorf("srPx = %d, want %d", d.srPx, oldSRPx)
	}
	wantRemoved := oldMetalPx - d.metalPx
	if wantRemoved <= 0 {
		t.Fatalf("metalPx = %d, before %d, want smaller", d.metalPx, oldMetalPx)
	}
	if d.userMetalPx != wantRemoved {
		t.Errorf("userMetalPx = %d, want %d", d.userMetalPx, wantRemoved)
	}
	if d.userSRPx != 0 {
		t.Errorf("userSRPx = %d, want 0", d.userSRPx)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if goldenUserInside(shape, x, y) && d.material[i] == 3 {
				t.Fatalf("material at (%d,%d) inside the region is still 3", x, y)
			}
			if oldSR.At(x, y) && d.material[i] != 1 {
				t.Fatalf("old SR pixel (%d,%d) is material %d, want 1", x, y, d.material[i])
			}
		}
	}
}

func TestGoldenUserSROnlyRemovesSR(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	w, h := a.Scene.W, a.Scene.H
	shape := goldenUserBothWindow(t, a)

	oldD := g.Derived()
	oldInspect := oldD.metalInspect.Clone()
	oldSR := oldD.sr.Clone()
	oldMetalPx := oldD.metalPx
	oldSRPx := oldD.srPx
	if !goldenUserAnyInside(oldSR, shape, w, h) {
		t.Fatalf("chosen window holds no inspected SR pixel: %+v", shape)
	}

	g.AddUserRegion(golden.TargetSR, shape)
	pollJobs(t, a)

	d := g.Derived()
	if d == nil {
		t.Fatal("Derived() is nil after AddUserRegion")
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			inside := goldenUserInside(shape, x, y)
			oldSet := oldSR.At(x, y)
			newSet := d.sr.At(x, y)
			if inside && oldSet && newSet {
				t.Fatalf("pixel (%d,%d) inside the region is still inspected SR", x, y)
			}
			if !inside && oldSet != newSet {
				t.Fatalf("pixel (%d,%d) outside the region changed: %v -> %v", x, y, oldSet, newSet)
			}
		}
	}
	if !d.metalInspect.Equal(oldInspect) {
		t.Error("the inspected Metal mask changed on an SR region")
	}
	if d.metalPx != oldMetalPx {
		t.Errorf("metalPx = %d, want %d", d.metalPx, oldMetalPx)
	}
	wantRemoved := oldSRPx - d.srPx
	if wantRemoved <= 0 {
		t.Fatalf("srPx = %d, before %d, want smaller", d.srPx, oldSRPx)
	}
	if d.userSRPx != wantRemoved {
		t.Errorf("userSRPx = %d, want %d", d.userSRPx, wantRemoved)
	}
	if d.userMetalPx != 0 {
		t.Errorf("userMetalPx = %d, want 0", d.userMetalPx)
	}
}

func TestGoldenUserBothTargetsIndependent(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	w, h := a.Scene.W, a.Scene.H
	shape := goldenUserBothWindow(t, a)

	base := g.Derived()
	baseMetal, baseSR := base.metalPx, base.srPx

	g.AddUserRegion(golden.TargetMetal, shape)
	pollJobs(t, a)
	afterMetal := g.Derived()
	metalRemoved := baseMetal - afterMetal.metalPx
	srRemovedByMetal := baseSR - afterMetal.srPx
	if metalRemoved <= 0 {
		t.Fatalf("the Metal region removed nothing: metalPx %d -> %d", baseMetal, afterMetal.metalPx)
	}
	if srRemovedByMetal != 0 {
		t.Errorf("the Metal region removed %d SR pixels, want 0", srRemovedByMetal)
	}

	g.AddUserRegion(golden.TargetSR, shape)
	pollJobs(t, a)
	both := g.Derived()
	srRemovedTotal := baseSR - both.srPx
	if srRemovedTotal <= 0 {
		t.Fatalf("the SR region removed nothing: srPx %d -> %d", baseSR, both.srPx)
	}
	if both.metalPx != afterMetal.metalPx {
		t.Errorf("the SR region changed metalPx: %d -> %d", afterMetal.metalPx, both.metalPx)
	}
	if both.userMetalPx != metalRemoved {
		t.Errorf("userMetalPx = %d, want %d", both.userMetalPx, metalRemoved)
	}
	if both.userSRPx != srRemovedTotal {
		t.Errorf("userSRPx = %d, want %d", both.userSRPx, srRemovedTotal)
	}

	// Remove region 0 (the Metal one): Metal comes back, the SR exclusion stays.
	g.RemoveUserRegion(0)
	pollJobs(t, a)
	withoutMetal := g.Derived()
	if withoutMetal.metalPx != baseMetal {
		t.Errorf("metalPx = %d after removing the Metal region, want %d", withoutMetal.metalPx, baseMetal)
	}
	if withoutMetal.userMetalPx != 0 {
		t.Errorf("userMetalPx = %d after removing the Metal region, want 0", withoutMetal.userMetalPx)
	}
	if withoutMetal.srPx != both.srPx {
		t.Errorf("srPx = %d after removing the Metal region, want %d", withoutMetal.srPx, both.srPx)
	}
	if withoutMetal.userSRPx != srRemovedTotal {
		t.Errorf("userSRPx = %d after removing the Metal region, want %d", withoutMetal.userSRPx, srRemovedTotal)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !goldenUserInside(shape, x, y) {
				continue
			}
			if base.metalInspect.At(x, y) != withoutMetal.metalInspect.At(x, y) {
				t.Fatalf("Metal pixel (%d,%d) did not come back", x, y)
			}
			if base.sr.At(x, y) && withoutMetal.sr.At(x, y) {
				t.Fatalf("SR pixel (%d,%d) came back after removing the Metal region", x, y)
			}
		}
	}
}

func TestGoldenUserRevisions(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	if !g.Ready() {
		t.Fatal("Ready() = false on a fresh board")
	}
	check := func(step string) {
		t.Helper()
		if !g.Ready() {
			t.Fatalf("Ready() = false after %s", step)
		}
		if d := g.Derived(); d.rev != g.Rev() {
			t.Fatalf("derived.rev = %d, Rev() = %d after %s", d.rev, g.Rev(), step)
		}
	}

	rev0, learn0, align0 := g.Rev(), g.LearnRev(), g.AlignRev()
	shape := goldenUserBothWindow(t, a)
	g.AddUserRegion(golden.TargetMetal, shape)
	pollJobs(t, a)
	check("AddUserRegion")
	if g.Rev() < rev0+1 {
		t.Errorf("Rev() = %d after AddUserRegion, want at least %d", g.Rev(), rev0+1)
	}
	if g.LearnRev() != learn0 {
		t.Errorf("LearnRev() = %d after AddUserRegion, want %d", g.LearnRev(), learn0)
	}
	if g.AlignRev() != align0 {
		t.Errorf("AlignRev() = %d after AddUserRegion, want %d", g.AlignRev(), align0)
	}

	rev1, learn1 := g.Rev(), g.LearnRev()
	g.SetPadding(g.Padding() + 1)
	pollJobs(t, a)
	check("SetPadding")
	if g.Rev() <= rev1 {
		t.Errorf("Rev() = %d after SetPadding, want more than %d", g.Rev(), rev1)
	}
	if g.LearnRev() <= learn1 {
		t.Errorf("LearnRev() = %d after SetPadding, want more than %d", g.LearnRev(), learn1)
	}
	if g.AlignRev() != align0 {
		t.Errorf("AlignRev() = %d after SetPadding, want %d", g.AlignRev(), align0)
	}

	g.Morph(morphDilate, 1)
	pollJobs(t, a)
	check("Morph")
	if g.AlignRev() <= align0 {
		t.Errorf("AlignRev() = %d after Morph, want more than %d", g.AlignRev(), align0)
	}
}

func TestGoldenUserKeepsAlignmentAndLearning(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	dp.learned = [3]bool{true, true, true}
	dp.goldenRev = g.LearnRev()
	before := dp.AlignState()

	shape := goldenUserBothWindow(t, a)
	learnRev := g.LearnRev()
	g.AddUserRegion(golden.TargetMetal, shape)
	pollJobs(t, a)

	if g.LearnRev() != learnRev {
		t.Fatalf("LearnRev() = %d after the UserMask, want %d", g.LearnRev(), learnRev)
	}
	if dp.learned != ([3]bool{true, true, true}) {
		t.Fatalf("learned = %v after the UserMask, want all true", dp.learned)
	}
	if dp.goldenRev != g.LearnRev() {
		t.Fatalf("goldenRev = %d, LearnRev() = %d", dp.goldenRev, g.LearnRev())
	}
	if before != alignWorking && dp.AlignState() == alignWorking {
		t.Errorf("AlignState() = alignWorking because of the UserMask (was %v)", before)
	}

	dp.requestRun()
	pollJobs(t, a)
	if dp.learned != ([3]bool{true, true, true}) {
		t.Errorf("learned = %v after requestRun, want all true", dp.learned)
	}
	if dp.goldenRev != g.LearnRev() {
		t.Errorf("goldenRev = %d after requestRun, LearnRev() = %d", dp.goldenRev, g.LearnRev())
	}
}

func TestGoldenUserInspectRegions(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	w, h := a.Scene.W, a.Scene.H
	shape := goldenUserBothWindow(t, a)
	oldInspect := g.Derived().metalInspect.Clone()

	g.AddUserRegion(golden.TargetMetal, shape)
	pollJobs(t, a)

	d := g.Derived()
	reg := a.inspectRegionsFor(a.Scene)
	if len(reg.Material) != len(d.material) {
		t.Fatalf("region material has %d entries, want %d", len(reg.Material), len(d.material))
	}
	for i := range reg.Material {
		if reg.Material[i] != d.material[i] {
			t.Fatalf("region material differs at %d: %d, want %d", i, reg.Material[i], d.material[i])
		}
	}
	removed := 0
	for y := 0; y < h && removed == 0; y++ {
		for x := 0; x < w; x++ {
			if !goldenUserInside(shape, x, y) || !oldInspect.At(x, y) {
				continue
			}
			if reg.Material[y*w+x] != 0 {
				t.Fatalf("pixel (%d,%d) inside the region is material %d, want 0", x, y, reg.Material[y*w+x])
			}
			if !d.metalInspect.At(x, y) {
				removed++
				break
			}
		}
	}
	if removed == 0 {
		t.Error("no inspected Metal pixel inside the region to check")
	}
}

func TestGoldenUserConfirmedAndSave(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	g.dirOver = t.TempDir()

	g.Save()
	pollJobs(t, a)
	if !g.Confirmed() {
		t.Fatal("Confirmed() = false after a save")
	}

	shape := goldenUserBothWindow(t, a)
	g.AddUserRegion(golden.TargetMetal, shape)
	pollJobs(t, a)
	if g.Confirmed() {
		t.Error("Confirmed() = true after an AddUserRegion")
	}
	if g.UserMask().Count(golden.TargetMetal) != 1 {
		t.Fatalf("UserMask().Count(metal) = %d, want 1", g.UserMask().Count(golden.TargetMetal))
	}
	userMetalPx := g.Derived().userMetalPx

	g.Save()
	pollJobs(t, a)
	if !g.Confirmed() {
		t.Fatal("Confirmed() = false after the second save")
	}
	want := g.UserMask()

	g.modelLoaded(a.Scene)
	pollJobs(t, a)
	if !g.Ready() {
		t.Fatal("Ready() = false after the reload")
	}
	got := g.UserMask()
	if !reflect.DeepEqual(got.Regions, want.Regions) {
		t.Errorf("UserMask().Regions after the load = %+v, want %+v", got.Regions, want.Regions)
	}
	if g.Derived().userMetalPx != userMetalPx {
		t.Errorf("userMetalPx = %d after the load, want %d", g.Derived().userMetalPx, userMetalPx)
	}
}

func TestGoldenUserResetOnModelLoaded(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	g.dirOver = t.TempDir()
	shape := goldenUserBothWindow(t, a)
	g.AddUserRegion(golden.TargetSR, shape)
	pollJobs(t, a)
	if g.UserMask().Empty() {
		t.Fatal("UserMask() is empty after AddUserRegion")
	}

	g.modelLoaded(a.Scene)
	pollJobs(t, a)
	if !g.UserMask().Empty() {
		t.Errorf("UserMask() = %+v after modelLoaded with an empty folder, want empty", g.UserMask().Regions)
	}
	if d := g.Derived(); d != nil && (d.userMetalPx != 0 || d.userSRPx != 0) {
		t.Errorf("userMask pixels after the reload: metal %d, sr %d, want 0", d.userMetalPx, d.userSRPx)
	}
}

func TestGoldenUserInvalidAndNoop(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	rev := g.Rev()

	bad := golden.Shape{Kind: golden.ShapeRect, P0: golden.Pt{X: 5, Y: 5}, P1: golden.Pt{X: 5, Y: 5}}
	g.AddUserRegion(golden.TargetMetal, bad)
	pollJobs(t, a)
	if !g.UserMask().Empty() {
		t.Errorf("UserMask() = %+v after an invalid shape, want empty", g.UserMask().Regions)
	}
	if g.Rev() != rev {
		t.Errorf("Rev() = %d after an invalid shape, want %d", g.Rev(), rev)
	}

	g.RemoveUserRegion(5)
	g.ClearUserMask(true, golden.TargetMetal)
	pollJobs(t, a)
	if !g.UserMask().Empty() {
		t.Errorf("UserMask() = %+v after RemoveUserRegion(5) / ClearUserMask, want empty", g.UserMask().Regions)
	}
	if g.Rev() != rev {
		t.Errorf("Rev() = %d after no-ops, want %d", g.Rev(), rev)
	}
	if g.Busy() {
		t.Error("Busy() = true after the poll")
	}
}

func TestGoldenUserEmptyBoard(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	w, h := a.Scene.W, a.Scene.H
	full := golden.Shape{Kind: golden.ShapeRect, P0: golden.Pt{X: 0, Y: 0}, P1: golden.Pt{X: float64(w), Y: float64(h)}}
	if g.Empty() {
		t.Fatal("a fresh board is empty")
	}

	g.AddUserRegion(golden.TargetMetal, full)
	g.AddUserRegion(golden.TargetSR, full)
	pollJobs(t, a)

	if !g.Ready() {
		t.Fatal("Ready() = false after full-image regions")
	}
	d := g.Derived()
	if d.metalPx != 0 || d.srPx != 0 {
		t.Fatalf("metalPx = %d, srPx = %d after full-image regions, want 0 and 0", d.metalPx, d.srPx)
	}
	if !g.Empty() {
		t.Error("Empty() = false although both areas are fully excluded")
	}
	if dp := defectPanelOf(a); dp != nil && !dp.InspectBlocked() {
		t.Error("InspectBlocked() = false although the board inspects nothing")
	}
	_ = core.MatMetal
}

func TestGoldenUserRemoveByValue(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	first := goldenUserRect(2, 2, 20, 20)
	second := goldenUserRect(30, 30, 50, 50)
	g.AddUserRegion(golden.TargetMetal, first)
	pollJobs(t, a)
	g.AddUserRegion(golden.TargetSR, second)
	pollJobs(t, a)
	if n := len(g.UserMask().Regions); n != 2 {
		t.Fatalf("%d regions, want 2", n)
	}
	// Two clicks on the same row before the board has applied the first removal must remove one
	// region only, not the one that moved into its place.
	g.RemoveUserRegion(0)
	g.RemoveUserRegion(0)
	pollJobs(t, a)
	um := g.UserMask()
	if len(um.Regions) != 1 || um.Regions[0].Target != golden.TargetSR {
		t.Fatalf("regions after a double removal = %+v, want only the SR region", um.Regions)
	}
}
