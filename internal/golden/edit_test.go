package golden

import "testing"

func TestShapeContainsRect(t *testing.T) {
	s := Shape{Kind: ShapeRect, P0: Pt{4, 2}, P1: Pt{0, 6}}
	if !s.Contains(0.5, 2.5) || !s.Contains(3.5, 5.5) {
		t.Fatalf("rect should contain its pixels")
	}
	if s.Contains(4.5, 2.5) || s.Contains(0.5, 6.5) || s.Contains(-0.1, 3) {
		t.Fatalf("rect should exclude the exclusive edges")
	}
}

func TestShapeContainsEllipseDegenerate(t *testing.T) {
	s := Shape{Kind: ShapeEllipse, P0: Pt{0, 0}, P1: Pt{0, 4}}
	if s.Contains(0, 2) {
		t.Fatalf("degenerate ellipse contains nothing")
	}
	s = Shape{Kind: ShapeEllipse, P0: Pt{0, 0}, P1: Pt{4, 4}}
	if !s.Contains(2, 2) || s.Contains(3.9, 3.9) {
		t.Fatalf("ellipse containment wrong")
	}
}

func TestShapeContainsConcavePolygon(t *testing.T) {
	// U shape: notch at the top middle.
	s := Shape{Kind: ShapePolygon, Points: []Pt{
		{0, 0}, {1, 0}, {1, 1}, {3, 1}, {3, 0}, {4, 0}, {4, 3}, {0, 3},
	}}
	if !s.Contains(0.5, 2.5) || !s.Contains(3.5, 1.5) {
		t.Fatalf("U shape should contain the arms")
	}
	if s.Contains(2, 0.5) {
		t.Fatalf("U shape should exclude the notch")
	}
}

func TestBoundsClipped(t *testing.T) {
	s := Shape{Kind: ShapeRect, P0: Pt{-5, -5}, P1: Pt{3, 3}}
	x0, y0, x1, y1 := s.Bounds(4, 4)
	if x0 != 0 || y0 != 0 || x1 != 3 || y1 != 3 {
		t.Fatalf("rect bounds = %d,%d,%d,%d", x0, y0, x1, y1)
	}
	s = Shape{Kind: ShapePolygon, Points: []Pt{{1.2, 1.3}, {5, 1}, {1, 4.9}}}
	x0, y0, x1, y1 = s.Bounds(4, 4)
	if x0 != 1 || y0 != 1 || x1 != 4 || y1 != 4 {
		t.Fatalf("polygon bounds = %d,%d,%d,%d", x0, y0, x1, y1)
	}
	s = Shape{Kind: ShapeRect, P0: Pt{10, 10}, P1: Pt{20, 20}}
	if x0, y0, x1, y1 = s.Bounds(4, 4); x0 != 0 || y0 != 0 || x1 != 0 || y1 != 0 {
		t.Fatalf("outside shape should be empty")
	}
}

func TestAddWithAndWithoutMinGV(t *testing.T) {
	m := NewMask(4, 1)
	pix := []uint8{10, 200, 20, 210}
	s := Shape{Kind: ShapeRect, P0: Pt{0, 0}, P1: Pt{4, 1}}

	Add(m, s, nil, 0)
	if m.Count() != 4 {
		t.Fatalf("add without minGV set %d pixels", m.Count())
	}

	m = NewMask(4, 1)
	Add(m, s, pix, 100)
	if m.Pix[0] != 0 || m.Pix[1] != 1 || m.Pix[2] != 0 || m.Pix[3] != 1 {
		t.Fatalf("add with minGV = %v", m.Pix)
	}
}

func TestErase(t *testing.T) {
	m := NewMask(4, 4)
	for i := range m.Pix {
		m.Pix[i] = 1
	}
	Erase(m, Shape{Kind: ShapeRect, P0: Pt{1, 1}, P1: Pt{3, 3}})
	if m.Count() != 16-4 {
		t.Fatalf("erase left %d pixels", m.Count())
	}
	if m.At(1, 1) || m.At(2, 2) {
		t.Fatalf("erased pixels still set")
	}
	if !m.At(0, 0) || !m.At(3, 3) {
		t.Fatalf("erase cleared outside pixels")
	}
}

func TestAreaFilterSpeck(t *testing.T) {
	m := NewMask(12, 10)
	m.Pix[2*12+2] = 1
	for i := 0; i < 2; i++ {
		m.Pix[5*12+5+i] = 1
	}
	in := m.Clone()
	out := AreaFilter(m, 2, false)
	if !m.Equal(in) {
		t.Fatalf("input mask was modified")
	}
	if out.At(2, 2) {
		t.Fatalf("speck should be removed")
	}
	if !out.At(5, 5) || !out.At(6, 5) {
		t.Fatalf("2-pixel blob should stay")
	}
}

func TestAreaFilterDiagonalForeground(t *testing.T) {
	m := NewMask(12, 10)
	m.Pix[2*12+2] = 1
	m.Pix[3*12+3] = 1
	out := AreaFilter(m, 2, false)
	if !out.At(2, 2) || !out.At(3, 3) {
		t.Fatalf("diagonal pixels are one 8-connected component")
	}
}

func TestAreaFilterFillHole(t *testing.T) {
	m := NewMask(12, 10)
	// 5x5 filled square with a single hole in the middle.
	for y := 2; y <= 6; y++ {
		for x := 2; x <= 6; x++ {
			m.Pix[y*12+x] = 1
		}
	}
	m.Pix[4*12+4] = 0
	out := AreaFilter(m, 2, true)
	if !out.At(4, 4) {
		t.Fatalf("enclosed 1-pixel hole should be filled")
	}
}

func TestAreaFilterHoleTouchingBorder(t *testing.T) {
	m := NewMask(12, 10)
	for y := 0; y <= 4; y++ {
		for x := 0; x <= 4; x++ {
			m.Pix[y*12+x] = 1
		}
	}
	m.Pix[0] = 0
	out := AreaFilter(m, 2, true)
	if out.At(0, 0) {
		t.Fatalf("hole touching the border must not be filled")
	}
}

func TestAreaFilterDiagonalBackground(t *testing.T) {
	m := NewMask(12, 10)
	// Blob ring around two diagonal background pixels.
	for y := 1; y <= 4; y++ {
		for x := 1; x <= 4; x++ {
			m.Pix[y*12+x] = 1
		}
	}
	// Diagonal gap: (2,2) and (3,3) are background, not 4-connected.
	m.Pix[2*12+2] = 0
	m.Pix[3*12+3] = 0
	out := AreaFilter(m, 2, true)
	if !out.At(2, 2) || !out.At(3, 3) {
		t.Fatalf("two diagonal 1-pixel background components should each be filled")
	}
}

func TestAreaFilterMinAreaOne(t *testing.T) {
	m := NewMask(6, 4)
	m.Pix[0] = 1
	out := AreaFilter(m, 1, true)
	if !out.Equal(m) {
		t.Fatalf("minArea <= 1 should clone")
	}
	if out == m {
		t.Fatalf("clone must be a new mask")
	}
}
