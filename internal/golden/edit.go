// Package golden: shapes drawn on the design grid, Add / Erase and the area
// filter. Standard library only.
package golden

import (
	"math"
	"sort"
)

// ShapeKind selects how a Shape is interpreted.
type ShapeKind int

const (
	ShapeRect ShapeKind = iota
	ShapeEllipse
	ShapePolygon
)

// Pt is a point in image coordinates (pixel units, pixel (x,y) has its centre
// at (x+0.5, y+0.5)).
type Pt struct{ X, Y float64 }

// Shape is an area drawn on the design grid. Rect and Ellipse use P0 and P1
// as opposite corners of the bounding box (any order); Polygon uses Points.
type Shape struct {
	Kind   ShapeKind
	P0, P1 Pt
	Points []Pt
}

// rectBox returns the ordered bounding box of P0 and P1.
func (s Shape) rectBox() (x0, y0, x1, y1 float64) {
	x0, x1 = s.P0.X, s.P1.X
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	y0, y1 = s.P0.Y, s.P1.Y
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	return
}

// Contains reports whether the point is inside the shape.
func (s Shape) Contains(px, py float64) bool {
	switch s.Kind {
	case ShapeRect:
		x0, y0, x1, y1 := s.rectBox()
		return x0 <= px && px < x1 && y0 <= py && py < y1
	case ShapeEllipse:
		x0, y0, x1, y1 := s.rectBox()
		cx := (x0 + x1) / 2
		cy := (y0 + y1) / 2
		rx := (x1 - x0) / 2
		ry := (y1 - y0) / 2
		if rx <= 0 || ry <= 0 {
			return false
		}
		dx := (px - cx) / rx
		dy := (py - cy) / ry
		return dx*dx+dy*dy <= 1
	case ShapePolygon:
		return s.polygonContains(px, py)
	}
	return false
}

// polygonContains applies the even-odd rule.
func (s Shape) polygonContains(px, py float64) bool {
	n := len(s.Points)
	if n < 3 {
		return false
	}
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		pi, pj := s.Points[i], s.Points[j]
		if (pi.Y > py) != (pj.Y > py) {
			x := pi.X + (py-pi.Y)*(pj.X-pi.X)/(pj.Y-pi.Y)
			if px < x {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}

// polygonBounds returns the clipped integer bounding box of the points:
// floor of the minimum, ceil of the maximum. Empty when the result is
// outside or degenerate.
func (s Shape) polygonBounds(w, h int) (x0, y0, x1, y1 int) {
	if len(s.Points) < 3 {
		return 0, 0, 0, 0
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, p := range s.Points {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	x0 = int(math.Floor(minX))
	y0 = int(math.Floor(minY))
	x1 = int(math.Ceil(maxX))
	y1 = int(math.Ceil(maxY))
	return clipBounds(x0, y0, x1, y1, w, h)
}

// clipBounds clamps the half-open box to the w*h image.
func clipBounds(x0, y0, x1, y1, w, h int) (int, int, int, int) {
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > w {
		x1 = w
	}
	if y1 > h {
		y1 = h
	}
	if x1 <= x0 || y1 <= y0 {
		return 0, 0, 0, 0
	}
	return x0, y0, x1, y1
}

// Bounds returns the pixel bounds of the shape clipped to a w*h image, with
// x1 and y1 exclusive. It returns (0,0,0,0) when the result is empty.
func (s Shape) Bounds(w, h int) (x0, y0, x1, y1 int) {
	if w <= 0 || h <= 0 {
		return 0, 0, 0, 0
	}
	switch s.Kind {
	case ShapeRect, ShapeEllipse:
		bx0, by0, bx1, by1 := s.rectBox()
		if bx1 <= bx0 || by1 <= by0 {
			return 0, 0, 0, 0
		}
		x0 = int(math.Floor(bx0))
		y0 = int(math.Floor(by0))
		x1 = int(math.Ceil(bx1))
		y1 = int(math.Ceil(by1))
		return clipBounds(x0, y0, x1, y1, w, h)
	case ShapePolygon:
		return s.polygonBounds(w, h)
	}
	return 0, 0, 0, 0
}

// Add sets, in place, every pixel inside s. When minGV > 0 and pix has the
// mask size only pixels with pix[i] >= minGV are set. A nil mask is a no-op.
func Add(m *Mask, s Shape, pix []uint8, minGV int) {
	editShape(m, s, pix, minGV, true)
}

// Erase clears, in place, every pixel inside s. A nil mask is a no-op.
func Erase(m *Mask, s Shape) {
	editShape(m, s, nil, 0, false)
}

// editShape walks the clipped bounds and tests the pixel centres.
func editShape(m *Mask, s Shape, pix []uint8, minGV int, set bool) {
	if !m.Valid() {
		return
	}
	useGV := set && minGV > 0 && len(pix) == m.W*m.H
	x0, y0, x1, y1 := s.Bounds(m.W, m.H)
	if x1 <= x0 || y1 <= y0 {
		return
	}
	if set && s.Kind == ShapePolygon && len(s.Points) >= 3 {
		fillPolygon(m, s, x0, y0, x1, y1, pix, useGV, minGV)
		return
	}
	for y := y0; y < y1; y++ {
		cy := float64(y) + 0.5
		row := y * m.W
		for x := x0; x < x1; x++ {
			if !s.Contains(float64(x)+0.5, cy) {
				continue
			}
			if set {
				if useGV && pix[row+x] < uint8(minGV) {
					continue
				}
				m.Pix[row+x] = 1
			} else {
				m.Pix[row+x] = 0
			}
		}
	}
}

// fillPolygon is the scanline fill of a polygon: crossings of the row centre
// with the edges are sorted and the spans between pairs are filled.
func fillPolygon(m *Mask, s Shape, x0, y0, x1, y1 int, pix []uint8, useGV bool, minGV int) {
	n := len(s.Points)
	xs := make([]float64, 0, n)
	for y := y0; y < y1; y++ {
		cy := float64(y) + 0.5
		xs = xs[:0]
		j := n - 1
		for i := 0; i < n; i++ {
			pi, pj := s.Points[i], s.Points[j]
			if (pi.Y > cy) != (pj.Y > cy) {
				xs = append(xs, pi.X+(cy-pi.Y)*(pj.X-pi.X)/(pj.Y-pi.Y))
			}
			j = i
		}
		if len(xs) < 2 {
			continue
		}
		sort.Float64s(xs)
		row := y * m.W
		for k := 0; k+1 < len(xs); k += 2 {
			xa, xb := xs[k], xs[k+1]
			lo := int(math.Ceil(xa - 0.5))
			if lo < x0 {
				lo = x0
			}
			hi := int(math.Ceil(xb - 0.5))
			if hi > x1 {
				hi = x1
			}
			for x := lo; x < hi; x++ {
				if useGV && pix[row+x] < uint8(minGV) {
					continue
				}
				m.Pix[row+x] = 1
			}
		}
	}
}

// AreaFilter returns a new mask without foreground components (8-connected)
// smaller than minArea; with fillHoles it also fills background components
// (4-connected) smaller than minArea that do not touch the image border.
// minArea <= 1 clones the mask.
func AreaFilter(m *Mask, minArea int, fillHoles bool) *Mask {
	if !m.Valid() || minArea <= 1 {
		return m.Clone()
	}
	w, h := m.W, m.H
	out := m.Clone()
	visited := make([]bool, w*h)
	stack := make([]int32, 0, 1024)
	comp := make([]int32, 0, 1024)

	// Foreground: 8-connected, drop small components.
	for start := 0; start < w*h; start++ {
		if visited[start] || m.Pix[start] == 0 {
			continue
		}
		stack = stack[:0]
		comp = comp[:0]
		visited[start] = true
		stack = append(stack, int32(start))
		touchBorder := false
		for len(stack) > 0 {
			idx := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			comp = append(comp, idx)
			x, y := int(idx)%w, int(idx)/w
			if x == 0 || y == 0 || x == w-1 || y == h-1 {
				touchBorder = true
			}
			for dy := -1; dy <= 1; dy++ {
				ny := y + dy
				if ny < 0 || ny >= h {
					continue
				}
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx := x + dx
					if nx < 0 || nx >= w {
						continue
					}
					ni := ny*w + nx
					if visited[ni] || m.Pix[ni] == 0 {
						continue
					}
					visited[ni] = true
					stack = append(stack, int32(ni))
				}
			}
		}
		_ = touchBorder
		if len(comp) < minArea {
			for _, idx := range comp {
				out.Pix[idx] = 0
			}
		}
	}

	if !fillHoles {
		return out
	}

	// Background: 4-connected on the filtered mask, fill small components
	// that do not touch the border.
	visited = make([]bool, w*h)
	for start := 0; start < w*h; start++ {
		if visited[start] || out.Pix[start] != 0 {
			continue
		}
		stack = stack[:0]
		comp = comp[:0]
		visited[start] = true
		stack = append(stack, int32(start))
		touchBorder := false
		for len(stack) > 0 {
			idx := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			comp = append(comp, idx)
			x, y := int(idx)%w, int(idx)/w
			if x == 0 || y == 0 || x == w-1 || y == h-1 {
				touchBorder = true
			}
			if x > 0 {
				ni := idx - 1
				if !visited[ni] && out.Pix[ni] == 0 {
					visited[ni] = true
					stack = append(stack, ni)
				}
			}
			if x < w-1 {
				ni := idx + 1
				if !visited[ni] && out.Pix[ni] == 0 {
					visited[ni] = true
					stack = append(stack, ni)
				}
			}
			if y > 0 {
				ni := idx - int32(w)
				if !visited[ni] && out.Pix[ni] == 0 {
					visited[ni] = true
					stack = append(stack, ni)
				}
			}
			if y < h-1 {
				ni := idx + int32(w)
				if !visited[ni] && out.Pix[ni] == 0 {
					visited[ni] = true
					stack = append(stack, ni)
				}
			}
		}
		if touchBorder || len(comp) >= minArea {
			continue
		}
		for _, idx := range comp {
			out.Pix[idx] = 1
		}
	}
	return out
}
