package app

import (
	"fmt"
	"math"

	"afvi/teach-imgui/internal/core"
)

const (
	// minRoiSize is the smallest new ROI a rubber band may create (full-resolution design px).
	minRoiSize = 16.0
	// roiHitSlop is the extra pick margin around an existing ROI (image px).
	roiHitSlop = 6.0
)

// roi drag modes.
const (
	roiDragNone = iota
	roiDragNew
	roiDragMove
)

// roiEditor is the pointer interaction on the design picture. ROIs are full-resolution
// design coordinates (core.AlignRoi{Name, X, Y, W, H}).
type roiEditor struct {
	sel int // selected ROI index, -1 = none

	mode int // roiDragNone | roiDragNew | roiDragMove
	idx  int // ROI under the move drag
	// start is where the drag began, cur where the pointer is now (image coordinates).
	start, cur core.Point
	// grab is the offset from the ROI origin to the pointer at the start of a move drag.
	grab core.Point
	// moving is the committed-free preview rectangle of a move drag.
	moving core.Rect
}

// newRoiEditor returns an editor with no selection and no drag.
func newRoiEditor() *roiEditor { return &roiEditor{sel: -1} }

// hitSlopRoi returns the index of the topmost ROI that contains (x, y) within the slop.
func (e *roiEditor) hitSlopRoi(rois []core.AlignRoi, x, y float64) int {
	for i := len(rois) - 1; i >= 0; i-- {
		r := rois[i]
		if x >= r.X-roiHitSlop && y >= r.Y-roiHitSlop &&
			x <= r.X+r.W+roiHitSlop && y <= r.Y+r.H+roiHitSlop {
			return i
		}
	}
	return -1
}

// clampMovedRoi keeps a rectangle of the given size inside [0, w) x [0, h).
func clampMovedRoi(r core.Rect, x, y float64, w, h float64) core.Rect {
	out := core.Rect{X: x, Y: y, W: r.W, H: r.H}
	if out.W > w {
		out.W = w
	}
	if out.H > h {
		out.H = h
	}
	if out.X < 0 {
		out.X = 0
	}
	if out.X+out.W > w {
		out.X = w - out.W
	}
	if out.Y < 0 {
		out.Y = 0
	}
	if out.Y+out.H > h {
		out.Y = h - out.H
	}
	return out
}

// fresh clones the ROI list so a change never aliases the caller's slice.
func freshRois(rois []core.AlignRoi) []core.AlignRoi {
	out := make([]core.AlignRoi, len(rois))
	copy(out, rois)
	return out
}

// firstName returns the first R<n> (n >= 1) not used by any name in rois.
func (e *roiEditor) firstName(rois []core.AlignRoi) string {
	used := make(map[string]bool, len(rois))
	for _, r := range rois {
		used[r.Name] = true
	}
	for n := 1; ; n++ {
		name := fmt.Sprintf("R%d", n)
		if !used[name] {
			return name
		}
	}
}

// Pointer handles a left-button event in image coordinates. It returns the possibly
// changed list, whether the list changed (only on drag END, never per mouse-move frame)
// and whether the event is owned.
func (e *roiEditor) Pointer(ev PointerEvent, rois []core.AlignRoi, w, h int) (out []core.AlignRoi, changed, owned bool) {
	if e.sel >= len(rois) {
		e.sel = -1
	}
	fw, fh := float64(w), float64(h)
	inside := ev.X >= 0 && ev.Y >= 0 && ev.X < fw && ev.Y < fh
	switch ev.Kind {
	case PointerDown:
		if !inside {
			e.mode = roiDragNone
			return rois, false, false
		}
		if idx := e.hitSlopRoi(rois, ev.X, ev.Y); idx >= 0 {
			e.sel = idx
			e.mode = roiDragMove
			e.idx = idx
			e.moving = rois[idx].Rect
			e.start = core.Point{X: ev.X, Y: ev.Y}
			e.cur = e.start
			e.grab = core.Point{X: ev.X - rois[idx].X, Y: ev.Y - rois[idx].Y}
			return rois, false, true
		}
		e.sel = -1
		e.mode = roiDragNew
		e.start = core.Point{X: ev.X, Y: ev.Y}
		e.cur = e.start
		return rois, false, true
	case PointerMove:
		if e.mode == roiDragNone {
			return rois, false, false
		}
		e.cur = core.Point{X: ev.X, Y: ev.Y}
		if e.mode == roiDragMove && e.idx >= 0 && e.idx < len(rois) {
			r := rois[e.idx]
			e.moving = clampMovedRoi(r.Rect, ev.X-e.grab.X, ev.Y-e.grab.Y, fw, fh)
		}
		return rois, false, true
	case PointerUp:
		mode := e.mode
		if mode == roiDragNone {
			return rois, false, false
		}
		e.mode = roiDragNone
		e.cur = core.Point{X: ev.X, Y: ev.Y}
		switch mode {
		case roiDragMove:
			if e.idx < 0 || e.idx >= len(rois) {
				return rois, false, true
			}
			r := rois[e.idx]
			moved := clampMovedRoi(r.Rect, ev.X-e.grab.X, ev.Y-e.grab.Y, fw, fh)
			if math.Abs(moved.X-r.X) < 1 && math.Abs(moved.Y-r.Y) < 1 {
				return rois, false, true
			}
			out = freshRois(rois)
			out[e.idx] = core.AlignRoi{Name: r.Name, Rect: moved}
			return out, true, true
		case roiDragNew:
			x := math.Min(e.start.X, e.cur.X)
			y := math.Min(e.start.Y, e.cur.Y)
			bw := math.Abs(e.cur.X - e.start.X)
			bh := math.Abs(e.cur.Y - e.start.Y)
			if bw < minRoiSize || bh < minRoiSize {
				e.sel = -1
				return rois, false, true
			}
			// Clamp the rectangle into the image; a drag that left the image shrinks it.
			if x < 0 {
				bw += x
				x = 0
			}
			if y < 0 {
				bh += y
				y = 0
			}
			if x+bw > fw {
				bw = fw - x
			}
			if y+bh > fh {
				bh = fh - y
			}
			if bw < minRoiSize || bh < minRoiSize {
				e.sel = -1
				return rois, false, true
			}
			out = freshRois(rois)
			out = append(out, core.AlignRoi{
				Name: e.firstName(rois),
				Rect: core.Rect{X: x, Y: y, W: bw, H: bh},
			})
			e.sel = len(out) - 1
			return out, true, true
		}
		return rois, false, true
	}
	return rois, false, false
}

// DeleteSelected removes the selected ROI (no-op when none) and clears the selection.
func (e *roiEditor) DeleteSelected(rois []core.AlignRoi) (out []core.AlignRoi, changed bool) {
	if e.sel < 0 || e.sel >= len(rois) {
		e.sel = -1
		return rois, false
	}
	out = make([]core.AlignRoi, 0, len(rois)-1)
	out = append(out, rois[:e.sel]...)
	out = append(out, rois[e.sel+1:]...)
	e.sel = -1
	return out, true
}

// Overlay draws every ROI (selected one highlighted) with its name, plus the preview of
// the open drag, into o.
func (e *roiEditor) Overlay(o *Overlay, rois []core.AlignRoi) {
	if e.sel >= len(rois) {
		e.sel = -1
	}
	for i, r := range rois {
		col := OverlayROI
		th := float32(2)
		if i == e.sel {
			col = OverlayWarn
			th = 3
		}
		o.Rect(r.X, r.Y, r.W, r.H, col, th)
		o.Text(r.X, r.Y, 2, 2, col, r.Name)
	}
	switch e.mode {
	case roiDragMove:
		if e.idx >= 0 && e.idx < len(rois) {
			m := e.moving
			o.DashedRect(m.X, m.Y, m.W, m.H, OverlayWarn, 1.5)
			o.Text(m.X, m.Y, 2, 2, OverlayWarn, rois[e.idx].Name)
		}
	case roiDragNew:
		x := math.Min(e.start.X, e.cur.X)
		y := math.Min(e.start.Y, e.cur.Y)
		bw := math.Abs(e.cur.X - e.start.X)
		bh := math.Abs(e.cur.Y - e.start.Y)
		o.DragBox(e.start.X, e.start.Y, e.cur.X, e.cur.Y, OverlayProbe)
		o.DashedRect(x, y, bw, bh, OverlayProbe, 1)
	}
}

// Reset clears the selection and drops any drag.
func (e *roiEditor) Reset() {
	e.sel = -1
	e.mode = roiDragNone
	e.idx = -1
	e.moving = core.Rect{}
}
