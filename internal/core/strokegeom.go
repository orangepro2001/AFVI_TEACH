package core

import "math"

// StrokeBounds 返回笔画可见形状的包围盒 (x0, y0, x1, y1)。
// Spot 为 (X±Radius, Y±Radius)；Line 为线段 (X,Y)-(X2,Y2) 的包围盒再按 Radius 外扩；
// Rect 与 Ellipse 为角点 (X,Y)、(X2,Y2) 的 min/max；
// Polygon 为 Points 的 min/max（Points 为空时退化为单点 (X,Y)）。
func StrokeBounds(s *DefectStroke) (x0, y0, x1, y1 float64) {
	switch s.Shape {
	case ShapeSpot:
		return s.X - s.Radius, s.Y - s.Radius, s.X + s.Radius, s.Y + s.Radius
	case ShapeLine:
		minX := math.Min(s.X, s.X2) - s.Radius
		maxX := math.Max(s.X, s.X2) + s.Radius
		minY := math.Min(s.Y, s.Y2) - s.Radius
		maxY := math.Max(s.Y, s.Y2) + s.Radius
		return minX, minY, maxX, maxY
	case ShapeRect, ShapeEllipse:
		return math.Min(s.X, s.X2), math.Min(s.Y, s.Y2), math.Max(s.X, s.X2), math.Max(s.Y, s.Y2)
	case ShapePolygon:
		if len(s.Points) == 0 {
			return s.X, s.Y, s.X, s.Y
		}
		minX := s.Points[0].X
		maxX := s.Points[0].X
		minY := s.Points[0].Y
		maxY := s.Points[0].Y
		for _, p := range s.Points[1:] {
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
		return minX, minY, maxX, maxY
	}
	return s.X, s.Y, s.X, s.Y
}

// StrokeCenter 返回 StrokeBounds 的中心。
func StrokeCenter(s *DefectStroke) (cx, cy float64) {
	x0, y0, x1, y1 := StrokeBounds(s)
	return (x0 + x1) / 2, (y0 + y1) / 2
}

// PointInPolygon 使用奇偶射线法判断点是否在多边形内；顶点少于 3 个时返回 false。
func PointInPolygon(pts []Point, x, y float64) bool {
	n := len(pts)
	if n < 3 {
		return false
	}
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		xi := pts[i].X
		yi := pts[i].Y
		xj := pts[j].X
		yj := pts[j].Y
		if (yi > y) != (yj > y) {
			xCross := (xj-xi)*(y-yi)/(yj-yi) + xi
			if x < xCross {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}

// PointInEllipse 判断点是否落在由角点 (x0,y0)、(x1,y1) 定义的内切椭圆内（角点顺序任意）。
// 退化盒（宽或高为 0）不包含任何点。
func PointInEllipse(x0, y0, x1, y1, px, py float64) bool {
	minX := math.Min(x0, x1)
	maxX := math.Max(x0, x1)
	minY := math.Min(y0, y1)
	maxY := math.Max(y0, y1)
	rx := (maxX - minX) / 2
	ry := (maxY - minY) / 2
	if rx <= 0 || ry <= 0 {
		return false
	}
	cx := (minX + maxX) / 2
	cy := (minY + maxY) / 2
	dx := (px - cx) / rx
	dy := (py - cy) / ry
	return dx*dx+dy*dy <= 1
}

// StrokeDistance 返回点到笔画可见形状的距离，点在形状内时为 0。
// Spot：max(0, hypot - Radius)；Line：到线段的距离减去 Radius（最小 0）；
// Rect：到矩形的距离（内部为 0）；Ellipse：内部为 0，否则按包围盒中心近似缩放；
// Polygon：内部为 0，否则到各边的最小距离。半径为 0 的退化情形不会返回 NaN。
func StrokeDistance(s *DefectStroke, px, py float64) float64 {
	switch s.Shape {
	case ShapeSpot:
		d := math.Hypot(px-s.X, py-s.Y) - s.Radius
		if d < 0 {
			return 0
		}
		return d
	case ShapeLine:
		d := distanceToSegment(px, py, s.X, s.Y, s.X2, s.Y2) - s.Radius
		if d < 0 {
			return 0
		}
		return d
	case ShapeRect:
		minX := math.Min(s.X, s.X2)
		maxX := math.Max(s.X, s.X2)
		minY := math.Min(s.Y, s.Y2)
		maxY := math.Max(s.Y, s.Y2)
		dx := 0.0
		if px < minX {
			dx = minX - px
		} else if px > maxX {
			dx = px - maxX
		}
		dy := 0.0
		if py < minY {
			dy = minY - py
		} else if py > maxY {
			dy = py - maxY
		}
		return math.Hypot(dx, dy)
	case ShapeEllipse:
		if PointInEllipse(s.X, s.Y, s.X2, s.Y2, px, py) {
			return 0
		}
		minX := math.Min(s.X, s.X2)
		maxX := math.Max(s.X, s.X2)
		minY := math.Min(s.Y, s.Y2)
		maxY := math.Max(s.Y, s.Y2)
		rx := (maxX - minX) / 2
		ry := (maxY - minY) / 2
		if rx <= 0 || ry <= 0 {
			return math.Hypot(px-(minX+maxX)/2, py-(minY+maxY)/2)
		}
		cx := (minX + maxX) / 2
		cy := (minY + maxY) / 2
		q := math.Hypot((px-cx)/rx, (py-cy)/ry) - 1
		if q < 0 {
			q = 0
		}
		return q * math.Min(rx, ry)
	case ShapePolygon:
		if len(s.Points) < 3 {
			if len(s.Points) == 0 {
				return math.Hypot(px-s.X, py-s.Y)
			}
			return math.Hypot(px-s.Points[0].X, py-s.Points[0].Y)
		}
		if PointInPolygon(s.Points, px, py) {
			return 0
		}
		best := math.Inf(1)
		n := len(s.Points)
		j := n - 1
		for i := 0; i < n; i++ {
			d := distanceToSegment(px, py, s.Points[j].X, s.Points[j].Y, s.Points[i].X, s.Points[i].Y)
			if d < best {
				best = d
			}
			j = i
		}
		return best
	}
	return math.Hypot(px-s.X, py-s.Y)
}
