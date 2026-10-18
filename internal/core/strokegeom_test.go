package core

import (
	"math"
	"testing"
)

func TestStrokeBounds(t *testing.T) {
	cases := []struct {
		name           string
		s              DefectStroke
		x0, y0, x1, y1 float64
	}{
		{"spot", DefectStroke{Shape: ShapeSpot, X: 10, Y: 20, Radius: 3}, 7, 17, 13, 23},
		{"line", DefectStroke{Shape: ShapeLine, X: 0, Y: 0, X2: 10, Y2: 0, Radius: 2}, -2, -2, 12, 2},
		{"rect", DefectStroke{Shape: ShapeRect, X: 5, Y: 5, X2: 1, Y2: 2}, 1, 2, 5, 5},
		{"ellipse", DefectStroke{Shape: ShapeEllipse, X: 0, Y: 0, X2: 4, Y2: 6}, 0, 0, 4, 6},
		{"polygon", DefectStroke{Shape: ShapePolygon, Points: []Point{{X: 1, Y: 1}, {X: 5, Y: 2}, {X: 3, Y: 7}}}, 1, 1, 5, 7},
		{"polygon-empty", DefectStroke{Shape: ShapePolygon, X: 9, Y: 9}, 9, 9, 9, 9},
	}
	for _, c := range cases {
		x0, y0, x1, y1 := StrokeBounds(&c.s)
		if x0 != c.x0 || y0 != c.y0 || x1 != c.x1 || y1 != c.y1 {
			t.Errorf("%s: bounds (%.1f,%.1f,%.1f,%.1f), want (%.1f,%.1f,%.1f,%.1f)", c.name, x0, y0, x1, y1, c.x0, c.y0, c.x1, c.y1)
		}
	}
}

func TestStrokeCenter(t *testing.T) {
	s := DefectStroke{Shape: ShapeRect, X: 0, Y: 0, X2: 10, Y2: 4}
	cx, cy := StrokeCenter(&s)
	if cx != 5 || cy != 2 {
		t.Errorf("center (%.1f,%.1f), want (5,2)", cx, cy)
	}
}

func TestPointInPolygon(t *testing.T) {
	square := []Point{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}
	if !PointInPolygon(square, 5, 5) {
		t.Errorf("square: (5,5) should be inside")
	}
	if PointInPolygon(square, 15, 5) {
		t.Errorf("square: (15,5) should be outside")
	}
	l := []Point{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 4}, {X: 4, Y: 4}, {X: 4, Y: 10}, {X: 0, Y: 10}}
	if !PointInPolygon(l, 2, 2) {
		t.Errorf("L: (2,2) should be inside")
	}
	if PointInPolygon(l, 8, 8) {
		t.Errorf("L: (8,8) in the notch should be outside")
	}
	if PointInPolygon(l, 20, 20) {
		t.Errorf("L: (20,20) should be outside")
	}
	if PointInPolygon([]Point{{X: 0, Y: 0}, {X: 1, Y: 1}}, 0.5, 0.5) {
		t.Errorf("fewer than 3 points must be false")
	}
}

func TestPointInEllipse(t *testing.T) {
	if !PointInEllipse(0, 0, 10, 10, 5, 5) {
		t.Errorf("center should be inside")
	}
	if PointInEllipse(0, 0, 10, 10, 0, 0) {
		t.Errorf("corner should be outside")
	}
	if PointInEllipse(0, 0, 0, 10, 0, 5) {
		t.Errorf("degenerate box must contain nothing")
	}
	if !PointInEllipse(10, 10, 0, 0, 5, 5) {
		t.Errorf("swapped corners: center should be inside")
	}
}

func TestStrokeDistance(t *testing.T) {
	spot := DefectStroke{Shape: ShapeSpot, X: 0, Y: 0, Radius: 2}
	if d := StrokeDistance(&spot, 0, 0); d != 0 {
		t.Errorf("spot inside: %.2f, want 0", d)
	}
	if d := StrokeDistance(&spot, 5, 0); math.Abs(d-3) > 1e-9 {
		t.Errorf("spot 5px out: %.2f, want 3", d)
	}
	line := DefectStroke{Shape: ShapeLine, X: 0, Y: 0, X2: 10, Y2: 0, Radius: 0}
	if d := StrokeDistance(&line, 5, 3); math.Abs(d-3) > 1e-9 {
		t.Errorf("line perpendicular: %.2f, want 3", d)
	}
	if d := StrokeDistance(&line, 15, 0); math.Abs(d-5) > 1e-9 {
		t.Errorf("line beyond end: %.2f, want 5", d)
	}
	rect := DefectStroke{Shape: ShapeRect, X: 0, Y: 0, X2: 10, Y2: 10}
	if d := StrokeDistance(&rect, 5, 5); d != 0 {
		t.Errorf("rect inside: %.2f, want 0", d)
	}
	if d := StrokeDistance(&rect, 14, 5); math.Abs(d-4) > 1e-9 {
		t.Errorf("rect 4px right: %.2f, want 4", d)
	}
	poly := DefectStroke{Shape: ShapePolygon, Points: []Point{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}}
	if d := StrokeDistance(&poly, 5, 5); d != 0 {
		t.Errorf("polygon inside: %.2f, want 0", d)
	}
	if d := StrokeDistance(&poly, 13, 5); math.Abs(d-3) > 1e-9 {
		t.Errorf("polygon outside: %.2f, want 3", d)
	}
	zeroSpot := DefectStroke{Shape: ShapeSpot, X: 1, Y: 1, Radius: 0}
	if d := StrokeDistance(&zeroSpot, 1, 1); math.IsNaN(d) {
		t.Errorf("zero-radius spot returned NaN")
	}
	zeroLine := DefectStroke{Shape: ShapeLine, X: 1, Y: 1, X2: 1, Y2: 1, Radius: 0}
	if d := StrokeDistance(&zeroLine, 4, 5); math.IsNaN(d) {
		t.Errorf("zero-length line returned NaN")
	}
}
