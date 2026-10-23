package app

import (
	"math"

	"afvi/teach-imgui/internal/core"
)

// FrameKind selects the shape a FrameTool draws.
type FrameKind int

// Frame tool kinds.
const (
	FrameRect FrameKind = iota
	FrameEllipse
	FramePen
)

// FrameResult is a finished frame in image coordinates.
type FrameResult struct {
	Kind   FrameKind
	P0, P1 core.Point   // FrameRect/FrameEllipse: opposite corners as dragged
	Points []core.Point // FramePen: closed polygon vertices (a copy, ≥3)
}

// FrameTool is a reusable, panel-independent frame tool: a rubber-band rectangle or
// ellipse, or a Photoshop-style pen polygon closed with a right click.
type FrameTool struct {
	Kind FrameKind

	dragging bool
	start    core.Point
	cur      core.Point

	points []core.Point
}

// NewFrameTool returns a frame tool in rectangle mode.
func NewFrameTool() *FrameTool { return &FrameTool{Kind: FrameRect} }

// Reset cancels any drag and drops an open pen polygon.
func (t *FrameTool) Reset() {
	t.dragging = false
	t.points = nil
}

// Busy reports whether a rubber-band drag is active or a pen polygon has vertices.
func (t *FrameTool) Busy() bool { return t.dragging || len(t.points) > 0 }

// Undo removes the last pen vertex (no-op for other kinds or an empty polygon).
func (t *FrameTool) Undo() {
	if t.Kind != FramePen || len(t.points) == 0 {
		return
	}
	t.points = t.points[:len(t.points)-1]
}

// Pointer handles one viewer event. It returns whether the tool owns the drag and, when a
// frame is finished, the result.
func (t *FrameTool) Pointer(ev PointerEvent) (owned bool, done *FrameResult) {
	switch t.Kind {
	case FrameRect, FrameEllipse:
		switch ev.Kind {
		case PointerDown:
			t.dragging = true
			t.start = core.Point{X: ev.X, Y: ev.Y}
			t.cur = t.start
			return true, nil
		case PointerMove:
			if t.dragging {
				t.cur = core.Point{X: ev.X, Y: ev.Y}
			}
			return true, nil
		case PointerUp:
			if !t.dragging {
				return false, nil
			}
			t.dragging = false
			dx := math.Abs(t.cur.X - t.start.X)
			dy := math.Abs(t.cur.Y - t.start.Y)
			if dx >= 2 && dy >= 2 {
				return true, &FrameResult{Kind: t.Kind, P0: t.start, P1: t.cur}
			}
			return true, nil
		}
		return false, nil
	case FramePen:
		switch ev.Kind {
		case PointerDown:
			p := core.Point{X: ev.X, Y: ev.Y}
			if len(t.points) == 0 || math.Hypot(p.X-t.points[len(t.points)-1].X, p.Y-t.points[len(t.points)-1].Y) > 0.5 {
				t.points = append(t.points, p)
			}
			return true, nil
		case PointerUp:
			return true, nil
		case PointerRightClick:
			if len(t.points) < 3 {
				return true, nil
			}
			pts := make([]core.Point, len(t.points))
			copy(pts, t.points)
			t.points = nil
			return true, &FrameResult{Kind: FramePen, Points: pts}
		}
		return true, nil
	}
	return false, nil
}

// Draw renders the in-progress frame in image space through o.
func (t *FrameTool) Draw(o *Overlay, mouseX, mouseY float64, col uint32) {
	switch t.Kind {
	case FrameRect:
		if t.dragging {
			o.DashedRect(math.Min(t.start.X, t.cur.X), math.Min(t.start.Y, t.cur.Y),
				math.Abs(t.cur.X-t.start.X), math.Abs(t.cur.Y-t.start.Y), col, 1)
		}
	case FrameEllipse:
		if t.dragging {
			drawEllipseOutline(o, t.start.X, t.start.Y, t.cur.X, t.cur.Y, col, 1)
		}
	case FramePen:
		if len(t.points) == 0 {
			return
		}
		for i := 1; i < len(t.points); i++ {
			o.Line(t.points[i-1].X, t.points[i-1].Y, t.points[i].X, t.points[i].Y, col, 1)
		}
		last := t.points[len(t.points)-1]
		o.Line(last.X, last.Y, mouseX, mouseY, col, 1)
		if len(t.points) >= 2 {
			o.Line(mouseX, mouseY, t.points[0].X, t.points[0].Y, col, 1)
		}
		// Vertex markers: small circles (the front end has no screen-space squares).
		r := 2.5 / math.Max(float64(o.Scale()), 1e-6)
		for _, p := range t.points {
			o.Circle(p.X, p.Y, r, col, 2)
		}
	}
}

// drawEllipseOutline draws a closed ellipse inscribed in the corner box (x0,y0)-(x1,y1)
// as a 64-segment polyline.
func drawEllipseOutline(o *Overlay, x0, y0, x1, y1 float64, col uint32, thickness float32) {
	cx := (x0 + x1) / 2
	cy := (y0 + y1) / 2
	rx := math.Abs(x1-x0) / 2
	ry := math.Abs(y1-y0) / 2
	const segments = 64
	prevX := cx + rx
	prevY := cy
	for i := 1; i <= segments; i++ {
		a := 2 * math.Pi * float64(i) / segments
		x := cx + rx*math.Cos(a)
		y := cy + ry*math.Sin(a)
		o.Line(prevX, prevY, x, y, col, thickness)
		prevX, prevY = x, y
	}
}

// drawPolygonOutline draws a closed polygon through pts.
func drawPolygonOutline(o *Overlay, pts []core.Point, col uint32, thickness float32) {
	if len(pts) < 2 {
		return
	}
	for i := 1; i < len(pts); i++ {
		o.Line(pts[i-1].X, pts[i-1].Y, pts[i].X, pts[i].Y, col, thickness)
	}
	o.Line(pts[len(pts)-1].X, pts[len(pts)-1].Y, pts[0].X, pts[0].Y, col, thickness)
}
