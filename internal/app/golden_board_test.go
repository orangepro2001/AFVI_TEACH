package app

import (
	"testing"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
)

// goldenFull returns a rectangle over the whole w x h image.
func goldenFull(w, h int) golden.Shape {
	return golden.Shape{Kind: golden.ShapeRect, P0: golden.Pt{X: 0, Y: 0}, P1: golden.Pt{X: float64(w), Y: float64(h)}}
}

// goldenAutoRange reads the range of a fresh board the way the first build leaves it.
func goldenAutoRange(t *testing.T, a *App) (int, int, bool) {
	t.Helper()
	g := a.golden
	if g == nil || !g.Ready() {
		t.Fatal("golden board is not ready")
	}
	return g.Range()
}

func TestGoldenReady(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	if g == nil {
		t.Fatal("teach app has no golden board")
	}
	if !g.Ready() {
		t.Fatal("Ready() = false after the first build")
	}
	if !g.Metal().Valid() {
		t.Fatal("Metal() is not valid")
	}
	d := g.Derived()
	if d == nil {
		t.Fatal("Derived() is nil")
	}
	if d.rev != g.Rev() {
		t.Errorf("derived.rev = %d, Rev() = %d, want equal", d.rev, g.Rev())
	}
	w, h := a.Scene.W, a.Scene.H
	if len(d.material) != w*h {
		t.Fatalf("material has %d entries, want %d", len(d.material), w*h)
	}
	var n3, n1 int
	for _, v := range d.material {
		switch v {
		case 3:
			n3++
		case 1:
			n1++
		case 0:
		default:
			t.Fatalf("unexpected material value %d", v)
		}
	}
	if n3 != d.metalPx {
		t.Errorf("material 3 count = %d, metalPx = %d", n3, d.metalPx)
	}
	if n1 != d.srPx {
		t.Errorf("material 1 count = %d, srPx = %d", n1, d.srPx)
	}
	src := g.Source()
	if src == nil || src.Width != w || src.Height != h {
		t.Fatalf("Source() = %+v, want %dx%d", src, w, h)
	}
	key, _, _ := g.Message()
	if key != "golden.msg.auto" && key != "golden.msg.fallback" {
		t.Errorf("Message key = %q, want golden.msg.auto or golden.msg.fallback", key)
	}
	if g.Confirmed() {
		t.Error("Confirmed() = true on a fresh board")
	}
}

func TestGoldenSR(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	if g.SRInset() != golden.DefaultSRInset {
		t.Fatalf("SRInset() = %d, want %d", g.SRInset(), golden.DefaultSRInset)
	}
	rev := g.Rev()
	d := g.Derived()
	if d.srPx != 1920 {
		t.Errorf("srPx = %d with inset %d, want 1920", d.srPx, golden.DefaultSRInset)
	}
	w := a.Scene.W
	for i, v := range d.sr.Pix {
		if v == 0 {
			continue
		}
		if a.Scene.Material[i] != core.MatSR {
			t.Fatalf("SR pixel %d lies outside the design SR layer", i)
		}
	}

	g.SetSRInset(0)
	pollJobs(t, a)
	if g.Rev() <= rev {
		t.Errorf("Rev() = %d did not grow after the inset change", g.Rev())
	}
	rev = g.Rev()
	if got := g.Derived().srPx; got != 2688 {
		t.Errorf("srPx = %d with inset 0, want 2688", got)
	}
	if x := 40; !g.Derived().sr.At(x, 12) {
		t.Errorf("pixel (%d, 12) is not SR with inset 0", x)
	}
	if x := 12; a.Scene.Material[(x+w*12)] != core.MatBase && g.Derived().sr.At(x, 12) {
		t.Errorf("pixel (%d, 12) is SR although it is not design SR", x)
	}
}

func TestGoldenPadding(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	g.SetPadding(0)
	pollJobs(t, a)
	d := g.Derived()
	if d.metalPx != g.Metal().Count() {
		t.Errorf("metalPx = %d with padding 0, want %d", d.metalPx, g.Metal().Count())
	}
	if d.bandPx != 0 {
		t.Errorf("bandPx = %d with padding 0, want 0", d.bandPx)
	}

	g.SetPadding(2)
	if g.Padding() != 2 {
		t.Errorf("Padding() = %d right after the request, want 2", g.Padding())
	}
	pollJobs(t, a)
	d = g.Derived()
	want := golden.InspectMetal(g.Metal(), 2)
	if !d.metalInspect.Equal(want) {
		t.Error("metalInspect does not equal InspectMetal(Metal(), 2)")
	}
	if d.bandPx != g.Metal().Count()-d.metalPx {
		t.Errorf("bandPx = %d, want %d", d.bandPx, g.Metal().Count()-d.metalPx)
	}

	g.SetPadding(500)
	if g.Padding() != golden.MaxPad {
		t.Errorf("Padding() = %d after SetPadding(500), want %d", g.Padding(), golden.MaxPad)
	}
	pollJobs(t, a)
}

func TestGoldenMorphAndUndo(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	before := g.Metal()

	g.Morph(morphErode, 1)
	pollJobs(t, a)
	if got := g.Metal().Count(); got >= before.Count() {
		t.Errorf("erode count = %d, before %d, want smaller", got, before.Count())
	}
	if !g.CanUndo() {
		t.Fatal("CanUndo() = false after a morph")
	}
	if !g.Undo() {
		t.Fatal("Undo() = false")
	}
	pollJobs(t, a)
	if !g.Metal().Equal(before) {
		t.Error("Metal() after undo does not equal the mask before the morph")
	}

	for i := 0; i < 8; i++ {
		if i%2 == 0 {
			g.Morph(morphDilate, 1)
		} else {
			g.Morph(morphErode, 1)
		}
		pollJobs(t, a)
	}
	n := 0
	for g.Undo() {
		pollJobs(t, a)
		n++
	}
	if n != goldenHistory {
		t.Errorf("%d undos available, want %d", n, goldenHistory)
	}

	queued := len(g.queue)
	g.Morph(morphOpen, 0)
	if len(g.queue) != queued || g.running {
		t.Errorf("Morph(morphOpen, 0) queued work: queue %d -> %d, running %v", queued, len(g.queue), g.running)
	}
}

func TestGoldenRange(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	lo, hi := 200, 255
	g.SetRange(lo, hi, false)
	if rl, rh, rout := g.Range(); rl != lo || rh != hi || rout {
		t.Errorf("Range() = %d, %d, %v right after the request, want %d, %d, false", rl, rh, rout, lo, hi)
	}
	pollJobs(t, a)
	want := golden.Threshold(g.Source().Data, a.Scene.W, a.Scene.H, uint8(lo), uint8(hi), false)
	if !g.Metal().Equal(want) {
		t.Error("Metal() does not equal the threshold of the source")
	}

	histBefore := len(g.history)
	for v := 100; v < 105; v++ {
		g.SetRange(v, 255, false)
	}
	if len(g.history) != histBefore {
		t.Errorf("history grew to %d before the poll, want %d", len(g.history), histBefore)
	}
	pollJobs(t, a)
	// The first range after another edit pushed one entry; a run of ranges shares it.
	if got := len(g.history); got != histBefore || got != 1 {
		t.Errorf("history has %d entries after six ranges, want 1 (before the run: %d)", got, histBefore)
	}
	if rl, _, _ := g.Range(); rl != 104 {
		t.Errorf("Range() low = %d, want the last request 104", rl)
	}

	g.SetRange(250, 10, false)
	if rl, rh, _ := g.Range(); rl != 250 || rh != 250 {
		t.Errorf("Range() = %d, %d after SetRange(250, 10), want 250, 250", rl, rh)
	}
	pollJobs(t, a)
}

func TestGoldenShapes(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	w, h := a.Scene.W, a.Scene.H

	g.EraseShape(goldenFull(w, h))
	pollJobs(t, a)
	if got := g.Metal().Count(); got != 0 {
		t.Fatalf("count after erasing everything = %d, want 0", got)
	}

	g.AddShape(goldenFull(w, h), 0)
	pollJobs(t, a)
	if got := g.Metal().Count(); got != w*h {
		t.Fatalf("count after adding everything = %d, want %d", got, w*h)
	}

	g.EraseShape(goldenFull(w, h))
	pollJobs(t, a)
	want := 0
	for _, v := range g.Source().Data {
		if v >= 150 {
			want++
		}
	}
	g.AddShape(goldenFull(w, h), 150)
	pollJobs(t, a)
	if got := g.Metal().Count(); got != want {
		t.Errorf("count with minGV 150 = %d, want %d", got, want)
	}

	prev := g.Metal().Count()
	ell := golden.Shape{Kind: golden.ShapeEllipse, P0: golden.Pt{X: 0, Y: 0}, P1: golden.Pt{X: float64(w) / 2, Y: float64(h)}}
	g.AddShape(ell, 0)
	pollJobs(t, a)
	added := g.Metal().Count()
	if added <= prev {
		t.Errorf("count after the ellipse = %d, want more than %d", added, prev)
	}
	g.EraseShape(ell)
	pollJobs(t, a)
	if got := g.Metal().Count(); got >= added {
		t.Errorf("count after erasing the ellipse = %d, want less than %d", got, added)
	}

	tri := golden.Shape{Kind: golden.ShapePolygon, Points: []golden.Pt{
		{X: 0, Y: 0}, {X: float64(w), Y: 0}, {X: 0, Y: float64(h)},
	}}
	prev = g.Metal().Count()
	g.AddShape(tri, 0)
	pollJobs(t, a)
	added = g.Metal().Count()
	if added <= prev {
		t.Errorf("count after the triangle = %d, want more than %d", added, prev)
	}
	g.EraseShape(tri)
	pollJobs(t, a)
	if got := g.Metal().Count(); got >= added {
		t.Errorf("count after erasing the triangle = %d, want less than %d", got, added)
	}
}

func TestGoldenAuto(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	lo, hi, out := goldenAutoRange(t, a)

	g.EraseShape(goldenFull(a.Scene.W, a.Scene.H))
	pollJobs(t, a)
	if got := g.Metal().Count(); got != 0 {
		t.Fatalf("count after erasing everything = %d, want 0", got)
	}

	g.Auto()
	pollJobs(t, a)
	if got := g.Metal().Count(); got == 0 {
		t.Error("Auto() left an empty mask")
	}
	if rl, rh, rout := g.Range(); rl != lo || rh != hi || rout != out {
		t.Errorf("Range() = %d, %d, %v after Auto(), want %d, %d, %v", rl, rh, rout, lo, hi, out)
	}
}

func TestGoldenPersistence(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	dir := t.TempDir()
	g.dirOver = dir

	g.Save()
	pollJobs(t, a)
	if !g.Confirmed() {
		t.Error("Confirmed() = false after a save")
	}
	if key, _, err := g.Message(); key != "golden.msg.saved" || err {
		t.Errorf("Message() = %q, err %v after a save, want golden.msg.saved", key, err)
	}
	board, err := golden.Load(dir)
	if err != nil {
		t.Fatalf("golden.Load: %v", err)
	}
	if !board.Metal.Equal(g.Metal()) {
		t.Error("the saved metal mask does not equal Metal()")
	}
	lo, hi, out := g.Range()
	if board.Settings.Channel != int(g.Channel()) || board.Settings.Padding != g.Padding() ||
		board.Settings.SRInset != g.SRInset() || board.Settings.Low != lo ||
		board.Settings.High != hi || board.Settings.OutRange != out {
		t.Errorf("saved settings %+v do not match the board", board.Settings)
	}

	g.EraseShape(goldenFull(a.Scene.W, a.Scene.H))
	pollJobs(t, a)
	g.Load()
	pollJobs(t, a)
	if !g.Metal().Equal(board.Metal) {
		t.Error("Metal() after Load() does not equal the saved mask")
	}
	if !g.Confirmed() {
		t.Error("Confirmed() = false after a load")
	}

	g.Morph(morphErode, 1)
	pollJobs(t, a)
	if g.Confirmed() {
		t.Error("Confirmed() = true after an edit")
	}

	empty := t.TempDir()
	g.dirOver = empty
	saved := g.Metal()
	g.Load()
	pollJobs(t, a)
	if key, _, isErr := g.Message(); !isErr || key == "" {
		t.Errorf("Message() = %q, err %v after loading from an empty folder, want an error", key, isErr)
	}
	if !g.Metal().Equal(saved) {
		t.Error("the mask changed after a failed load")
	}

	// Default folder: the saved board is loaded again when the model is reloaded.
	g.dirOver = ""
	g.Save()
	pollJobs(t, a)
	if !g.Confirmed() {
		t.Fatal("Confirmed() = false after the second save")
	}
	savedMask := g.Metal()
	g.modelLoaded(a.Scene)
	pollJobs(t, a)
	if key, _, _ := g.Message(); key != "golden.msg.loaded" {
		t.Errorf("Message() = %q after reloading the model, want golden.msg.loaded", key)
	}
	if !g.Confirmed() {
		t.Error("Confirmed() = false after the automatic load")
	}
	if !g.Metal().Equal(savedMask) {
		t.Error("the mask after the automatic load does not equal the saved one")
	}
}

func TestGoldenChannel(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	red := g.Source()
	metal := g.Metal()

	g.SetChannel(core.ChGreen)
	pollJobs(t, a)
	if g.Channel() != core.ChGreen {
		t.Errorf("Channel() = %v, want ChGreen", g.Channel())
	}
	grn := g.Source()
	if grn == nil || grn.Width != red.Width || grn.Height != red.Height {
		t.Fatalf("Source() = %+v after the channel change", grn)
	}
	same := true
	for i := range red.Data {
		if red.Data[i] != grn.Data[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("the green source equals the red one")
	}
	if g.Metal() != metal {
		t.Error("the mask pointer changed on a channel switch")
	}

	g.dirOver = t.TempDir()
	g.Save()
	pollJobs(t, a)
	g.modelLoaded(a.Scene)
	pollJobs(t, a)
	if g.Channel() != core.ChGreen {
		t.Errorf("Channel() = %v after reloading the saved board, want ChGreen", g.Channel())
	}
}

func TestGoldenInspectRegions(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
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
	same := len(reg.Material) == len(a.Scene.Material)
	if same {
		for i := range reg.Material {
			if reg.Material[i] != a.Scene.Material[i] {
				same = false
				break
			}
		}
	}
	if same {
		t.Error("the golden material map equals the design map")
	}

	g.modelLoaded(a.Scene)
	if g.Ready() {
		t.Error("Ready() = true right after modelLoaded")
	}
	if got := a.inspectRegionsFor(a.Scene).Material; len(got) != len(a.Scene.Material) || &got[0] != &a.Scene.Material[0] {
		t.Error("inspectRegionsFor did not fall back to the design map before the board is ready")
	}
	if dp := defectPanelOf(a); dp == nil || !dp.InspectBlocked() {
		t.Error("InspectBlocked() = false while the board is not ready")
	}
	pollJobs(t, a)
	if !g.Ready() {
		t.Fatal("Ready() = false after the rebuild")
	}

	dp := defectPanelOf(a)
	a.align = nil
	if dp.InspectBlocked() {
		t.Error("InspectBlocked() = true with the alignment off and a ready board")
	}

	dp.RunInspect()
	pollJobs(t, a)
	if !dp.learned[dp.channel] {
		t.Error("the channel was not learned")
	}
	if dp.goldenRev != g.Rev() {
		t.Errorf("goldenRev = %d, Rev() = %d", dp.goldenRev, g.Rev())
	}

	g.SetPadding(4)
	pollJobs(t, a)
	pollJobs(t, a)
	if dp.goldenRev != g.Rev() {
		t.Errorf("goldenRev = %d after the padding change, Rev() = %d", dp.goldenRev, g.Rev())
	}
	if !dp.learned[dp.channel] {
		t.Error("the channel was not learned again")
	}
	if dp.lastReport == nil {
		t.Fatal("lastReport is nil after the re-run")
	}

	mat := g.Derived().material
	w := a.Scene.W
	for _, df := range dp.lastReport.Defects {
		inside := false
		for y := df.BBox.Y; y < df.BBox.Y+df.BBox.H && !inside; y++ {
			for x := df.BBox.X; x < df.BBox.X+df.BBox.W; x++ {
				if x < 0 || y < 0 || x >= w || y >= a.Scene.H {
					continue
				}
				if mat[y*w+x] != 0 {
					inside = true
					break
				}
			}
		}
		if !inside {
			t.Errorf("defect %+v lies completely outside the inspected mask", df)
		}
	}
}

func TestGoldenStaleResults(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	g.modelLoaded(a.Scene)
	g.modelLoaded(a.Scene)
	g.SetPadding(3)
	pollJobs(t, a)
	if !g.Ready() {
		t.Fatal("Ready() = false after the stale results")
	}
	if g.Busy() {
		t.Error("Busy() = true after the poll")
	}
	if g.Padding() != 3 {
		t.Errorf("Padding() = %d, want 3", g.Padding())
	}
}

func TestGoldenShapeFromFrame(t *testing.T) {
	rect := goldenShapeFromFrame(&FrameResult{
		Kind: FrameRect,
		P0:   core.Point{X: 1, Y: 2},
		P1:   core.Point{X: 5, Y: 7},
	})
	if rect.Kind != golden.ShapeRect || rect.P0 != (golden.Pt{X: 1, Y: 2}) || rect.P1 != (golden.Pt{X: 5, Y: 7}) {
		t.Errorf("rect = %+v", rect)
	}
	ell := goldenShapeFromFrame(&FrameResult{
		Kind: FrameEllipse,
		P0:   core.Point{X: 1, Y: 2},
		P1:   core.Point{X: 5, Y: 7},
	})
	if ell.Kind != golden.ShapeEllipse || ell.P0 != (golden.Pt{X: 1, Y: 2}) || ell.P1 != (golden.Pt{X: 5, Y: 7}) {
		t.Errorf("ellipse = %+v", ell)
	}
	pen := goldenShapeFromFrame(&FrameResult{
		Kind:   FramePen,
		Points: []core.Point{{X: 1, Y: 2}, {X: 3, Y: 4}, {X: 5, Y: 6}},
	})
	if pen.Kind != golden.ShapePolygon || len(pen.Points) != 3 {
		t.Fatalf("polygon = %+v", pen)
	}
	for i, pt := range pen.Points {
		if pt != (golden.Pt{X: float64(i*2 + 1), Y: float64(i*2 + 2)}) {
			t.Errorf("point %d = %+v", i, pt)
		}
	}
	if zero := goldenShapeFromFrame(nil); zero.Kind != golden.ShapeRect || zero.P0 != (golden.Pt{}) || zero.P1 != (golden.Pt{}) {
		t.Errorf("nil result = %+v, want the zero shape", zero)
	}
}

func TestGoldenPixelInfo(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	g := a.golden
	w, h := a.Scene.W, a.Scene.H
	if _, _, _, _, ok := g.PixelInfo(-1, 0); ok {
		t.Error("PixelInfo(-1, 0) reports ok")
	}
	if _, _, _, _, ok := g.PixelInfo(w, 0); ok {
		t.Error("PixelInfo(W, 0) reports ok")
	}
	gv, _, _, _, ok := g.PixelInfo(5, 5)
	if !ok {
		t.Fatal("PixelInfo(5, 5) reports not ok")
	}
	if want := int(g.Source().Data[5*w+5]); gv != want {
		t.Errorf("PixelInfo(5, 5) gv = %d, want %d", gv, want)
	}
	if h <= 0 {
		t.Fatal("no image")
	}
}
