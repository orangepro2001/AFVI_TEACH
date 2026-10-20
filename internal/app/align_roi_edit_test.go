package app

import (
	"reflect"
	"testing"

	"afvi/teach-imgui/internal/core"
)

// roiAt builds an ROI for the tests.
func roiAt(name string, x, y, w, h float64) core.AlignRoi {
	return core.AlignRoi{Name: name, Rect: core.Rect{X: x, Y: y, W: w, H: h}}
}

// drag drives a full down / move / up drag from (x0, y0) to (x1, y1).
func drag(e *roiEditor, rois []core.AlignRoi, w, h int, x0, y0, x1, y1 float64) ([]core.AlignRoi, bool) {
	out, ch, owned := e.Pointer(PointerEvent{Kind: PointerDown, X: x0, Y: y0}, rois, w, h)
	if !owned {
		return out, ch
	}
	out2, _, _ := e.Pointer(PointerEvent{Kind: PointerMove, X: x1, Y: y1}, out, w, h)
	out3, ch3, _ := e.Pointer(PointerEvent{Kind: PointerUp, X: x1, Y: y1}, out2, w, h)
	return out3, ch || ch3
}

func TestRoiEditorDrawDirections(t *testing.T) {
	const w, h = 400, 300
	cases := []struct {
		name           string
		x0, y0, x1, y1 float64
		want           core.Rect
	}{
		{"right-down", 10, 20, 60, 80, core.Rect{X: 10, Y: 20, W: 50, H: 60}},
		{"left-down", 60, 20, 10, 80, core.Rect{X: 10, Y: 20, W: 50, H: 60}},
		{"right-up", 10, 80, 60, 20, core.Rect{X: 10, Y: 20, W: 50, H: 60}},
		{"left-up", 60, 80, 10, 20, core.Rect{X: 10, Y: 20, W: 50, H: 60}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newRoiEditor()
			out, changed := drag(e, nil, w, h, c.x0, c.y0, c.x1, c.y1)
			if !changed {
				t.Fatal("a large rectangle was not committed")
			}
			if len(out) != 1 {
				t.Fatalf("ROIs = %d, want 1", len(out))
			}
			if out[0].Rect != c.want {
				t.Errorf("rect = %+v, want %+v", out[0].Rect, c.want)
			}
			if out[0].Name != "R1" {
				t.Errorf("name = %q, want R1", out[0].Name)
			}
		})
	}
}

func TestRoiEditorIgnoresSmallRectangle(t *testing.T) {
	e := newRoiEditor()
	out, changed := drag(e, nil, 400, 300, 10, 10, 20, 20)
	if changed {
		t.Error("a rectangle below minRoiSize was committed")
	}
	if len(out) != 0 {
		t.Errorf("ROIs = %d, want 0", len(out))
	}
}

func TestRoiEditorNamesFirstUnused(t *testing.T) {
	e := newRoiEditor()
	rois := []core.AlignRoi{roiAt("R1", 0, 0, 40, 40), roiAt("R2", 50, 0, 40, 40)}
	out, changed := drag(e, rois, 400, 300, 100, 100, 200, 200)
	if !changed || len(out) != 3 {
		t.Fatalf("append failed: changed=%v len=%d", changed, len(out))
	}
	if out[2].Name != "R3" {
		t.Errorf("new name = %q, want R3", out[2].Name)
	}
	// Select R2 and delete it: the next auto name reuses R2.
	e.sel = 1
	del, ok := e.DeleteSelected(out)
	if !ok || len(del) != 2 {
		t.Fatalf("delete failed: ok=%v len=%d", ok, len(del))
	}
	out2, changed := drag(e, del, 400, 300, 250, 50, 350, 150)
	if !changed || len(out2) != 3 {
		t.Fatalf("second append failed")
	}
	if out2[2].Name != "R2" {
		t.Errorf("reused name = %q, want R2", out2[2].Name)
	}
}

func TestRoiEditorSelectsTopmost(t *testing.T) {
	e := newRoiEditor()
	rois := []core.AlignRoi{roiAt("A", 0, 0, 100, 100), roiAt("B", 50, 50, 100, 100)}
	_, _, owned := e.Pointer(PointerEvent{Kind: PointerDown, X: 60, Y: 60}, rois, 400, 300)
	if !owned {
		t.Fatal("down inside an ROI was not owned")
	}
	if e.sel != 1 {
		t.Errorf("sel = %d, want 1 (topmost)", e.sel)
	}
	e.Pointer(PointerEvent{Kind: PointerUp, X: 60, Y: 60}, rois, 400, 300)
	e.sel = -1
	_, _, _ = e.Pointer(PointerEvent{Kind: PointerDown, X: 5, Y: 5}, rois, 400, 300)
	if e.sel != 0 {
		t.Errorf("sel = %d, want 0 (only A covers that point)", e.sel)
	}
}

func TestRoiEditorMoveClamps(t *testing.T) {
	const w, h = 200, 150
	// Move far past each border; the ROI keeps its size and stays inside.
	corners := []struct {
		name   string
		dx, dy float64
		wantX  float64
		wantY  float64
	}{
		{"left", -1000, 0, 0, 20},
		{"top", 0, -1000, 20, 0},
		{"right", 1000, 0, float64(w) - 40, 20},
		{"bottom", 0, 1000, 20, float64(h) - 40},
	}
	for _, c := range corners {
		t.Run(c.name, func(t *testing.T) {
			e := newRoiEditor()
			rois := []core.AlignRoi{roiAt("R1", 20, 20, 40, 40)}
			// Grab the ROI centre so the move follows the pointer.
			gx, gy := 40.0, 40.0
			tx, ty := gx+c.dx, gy+c.dy
			_, _, owned := e.Pointer(PointerEvent{Kind: PointerDown, X: gx, Y: gy}, rois, w, h)
			if !owned {
				t.Fatal("down on the ROI was not owned")
			}
			e.Pointer(PointerEvent{Kind: PointerMove, X: tx, Y: ty}, rois, w, h)
			out, changed, _ := e.Pointer(PointerEvent{Kind: PointerUp, X: tx, Y: ty}, rois, w, h)
			if !changed {
				t.Fatalf("a large move was not committed")
			}
			if out[0].W != 40 || out[0].H != 40 {
				t.Errorf("size changed: %+v", out[0].Rect)
			}
			if out[0].X != c.wantX || out[0].Y != c.wantY {
				t.Errorf("pos = (%.0f,%.0f), want (%.0f,%.0f)", out[0].X, out[0].Y, c.wantX, c.wantY)
			}
			if out[0].X < 0 || out[0].Y < 0 || out[0].X+out[0].W > w || out[0].Y+out[0].H > h {
				t.Errorf("ROI left the image: %+v", out[0].Rect)
			}
		})
	}
}

func TestRoiEditorMoveTinyIsNoChange(t *testing.T) {
	e := newRoiEditor()
	rois := []core.AlignRoi{roiAt("R1", 20, 20, 40, 40)}
	e.Pointer(PointerEvent{Kind: PointerDown, X: 40, Y: 40}, rois, 200, 150)
	out, changed, _ := e.Pointer(PointerEvent{Kind: PointerUp, X: 40.4, Y: 40.2}, rois, 200, 150)
	if changed {
		t.Errorf("a sub-pixel move was reported as a change: %+v", out)
	}
}

func TestRoiEditorDownOutsideNotOwned(t *testing.T) {
	e := newRoiEditor()
	rois := []core.AlignRoi{roiAt("R1", 10, 10, 40, 40)}
	for _, p := range []core.Point{{X: -1, Y: 10}, {X: 10, Y: -1}, {X: 400, Y: 10}, {X: 10, Y: 300}} {
		_, changed, owned := e.Pointer(PointerEvent{Kind: PointerDown, X: p.X, Y: p.Y}, rois, 400, 300)
		if owned || changed {
			t.Errorf("down at %+v was owned=%v changed=%v outside the image", p, owned, changed)
		}
	}
}

func TestRoiEditorMoveNeverChanges(t *testing.T) {
	e := newRoiEditor()
	rois := []core.AlignRoi{roiAt("R1", 20, 20, 40, 40)}
	e.Pointer(PointerEvent{Kind: PointerDown, X: 40, Y: 40}, rois, 200, 150)
	for _, x := range []float64{50, 60, 70} {
		_, changed, owned := e.Pointer(PointerEvent{Kind: PointerMove, X: x, Y: 40}, rois, 200, 150)
		if changed {
			t.Errorf("PointerMove reported changed at x=%.0f", x)
		}
		if !owned {
			t.Errorf("PointerMove not owned at x=%.0f", x)
		}
	}
}

func TestRoiEditorDeleteSelected(t *testing.T) {
	e := newRoiEditor()
	rois := []core.AlignRoi{roiAt("R1", 0, 0, 40, 40), roiAt("R2", 50, 0, 40, 40)}
	out, changed := e.DeleteSelected(rois)
	if changed || len(out) != 2 {
		t.Errorf("DeleteSelected without selection changed the list (changed=%v len=%d)", changed, len(out))
	}
	e.sel = 0
	out2, changed := e.DeleteSelected(out)
	if !changed || len(out2) != 1 || out2[0].Name != "R2" {
		t.Fatalf("delete failed: changed=%v %+v", changed, out2)
	}
	if e.sel != -1 {
		t.Errorf("sel = %d after delete, want -1", e.sel)
	}
}

func TestRoiEditorDoesNotMutateInput(t *testing.T) {
	e := newRoiEditor()
	rois := []core.AlignRoi{roiAt("R1", 20, 20, 40, 40)}
	orig := freshRois(rois)
	if _, changed := drag(e, rois, 200, 150, 100, 100, 160, 160); !changed {
		t.Fatal("append did not commit")
	}
	if !reflect.DeepEqual(rois, orig) {
		t.Errorf("the input slice was mutated: %+v", rois)
	}
	// A move also works on a copy.
	e2 := newRoiEditor()
	if _, changed := drag(e2, rois, 200, 150, 40, 40, 90, 90); !changed {
		t.Fatal("move did not commit")
	}
	if !reflect.DeepEqual(rois, orig) {
		t.Errorf("the input slice was mutated by a move: %+v", rois)
	}
}

func TestRoiEditorReset(t *testing.T) {
	e := newRoiEditor()
	rois := []core.AlignRoi{roiAt("R1", 20, 20, 40, 40)}
	e.Pointer(PointerEvent{Kind: PointerDown, X: 100, Y: 100}, rois, 200, 150)
	e.Reset()
	if e.sel != -1 || e.mode != roiDragNone {
		t.Errorf("Reset left sel=%d mode=%d", e.sel, e.mode)
	}
	// A move after Reset must start from the pointer, not from stale drag state.
	_, _, owned := e.Pointer(PointerEvent{Kind: PointerDown, X: 40, Y: 40}, rois, 200, 150)
	if !owned || e.sel != 0 {
		t.Errorf("pointer after Reset owned=%v sel=%d", owned, e.sel)
	}
}
