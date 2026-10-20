package golden

import "testing"

// fillRect sets a rectangle of m to 1.
func fillRect(m *Mask, x0, y0, x1, y1 int) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			m.Pix[y*m.W+x] = 1
		}
	}
}

func TestClampPad(t *testing.T) {
	cases := []struct{ in, want int }{
		{-5, 0}, {0, 0}, {2, 2}, {MaxPad, MaxPad}, {MaxPad + 10, MaxPad},
	}
	for _, c := range cases {
		if got := ClampPad(c.in); got != c.want {
			t.Fatalf("ClampPad(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestInspectMetal(t *testing.T) {
	m := NewMask(40, 40)
	fillRect(m, 10, 10, 30, 30)
	got := InspectMetal(m, 2)
	if got.Count() != 16*16 {
		t.Fatalf("InspectMetal count = %d, want %d", got.Count(), 16*16)
	}
	same := InspectMetal(m, 0)
	if !same.Equal(m) {
		t.Fatal("InspectMetal with padding 0 must equal the mask")
	}
	if InspectMetal(nil, 2) != nil {
		t.Fatal("InspectMetal(nil) must be nil")
	}
}

func TestAutoSR(t *testing.T) {
	sr := NewMask(30, 30)
	fillRect(sr, 0, 0, 30, 30)
	metal := NewMask(30, 30)
	fillRect(metal, 10, 10, 20, 20)

	got := AutoSR(sr, metal, 2)
	if got == nil || !got.Valid() {
		t.Fatal("AutoSR returned an invalid mask")
	}
	// The image border does not shrink the SR (outside counts as inside for
	// erosion), but the metal block plus its inset is removed.
	if !got.At(0, 0) || !got.At(29, 29) {
		t.Fatal("AutoSR must keep the image border pixels")
	}
	if got.At(10, 10) || got.At(19, 19) || got.At(8, 15) {
		t.Fatal("AutoSR must remove the dilated metal region")
	}
	if !got.At(7, 7) {
		t.Fatal("AutoSR must keep pixels outside the dilated metal region")
	}

	only := AutoSR(sr, nil, 2)
	if only == nil || only.Count() != sr.Count() {
		t.Fatal("AutoSR with nil metal must be the plain erosion")
	}
}

func TestMaterialAndMaskFromMaterial(t *testing.T) {
	metal := NewMask(4, 4)
	metal.Pix[0] = 1
	sr := NewMask(4, 4)
	sr.Pix[1] = 1
	sr.Pix[0] = 1

	mat := Material(metal, sr)
	if len(mat) != 16 {
		t.Fatalf("Material len = %d, want 16", len(mat))
	}
	if mat[0] != 3 || mat[1] != 1 || mat[2] != 0 {
		t.Fatalf("Material values = %d,%d,%d, want 3,1,0", mat[0], mat[1], mat[2])
	}

	m := MaskFromMaterial(mat, 4, 4, 3)
	if m.Count() != 1 || !m.At(0, 0) {
		t.Fatal("MaskFromMaterial(3) wrong")
	}
	one := MaskFromMaterial(mat, 4, 4, 1)
	if one.Count() != 1 || !one.At(1, 0) {
		t.Fatal("MaskFromMaterial(1) wrong")
	}
	if short := MaskFromMaterial(mat, 5, 5, 3); short.Valid() != true || short.Count() != 0 {
		t.Fatal("MaskFromMaterial with a wrong length must be an empty mask")
	}

	if Material(nil, nil) != nil {
		t.Fatal("Material(nil, nil) must be nil")
	}
	if Material(NewMask(4, 4), NewMask(5, 5)) != nil {
		t.Fatal("Material with different sizes must be nil")
	}
}

func TestAutoRange(t *testing.T) {
	const w, h = 20, 20
	pix := make([]uint8, w*h)
	metal := NewMask(w, h)
	other := NewMask(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < w/2 {
				pix[y*w+x] = 100
				other.Pix[y*w+x] = 1
			} else {
				pix[y*w+x] = 200
				metal.Pix[y*w+x] = 1
			}
		}
	}
	lo, ok := AutoRange(pix, w, h, metal, other)
	if !ok {
		t.Fatal("AutoRange must succeed on a two-class picture")
	}
	if lo != 150 {
		t.Fatalf("AutoRange lo = %d, want 150", lo)
	}

	flat := make([]uint8, w*h)
	for i := range flat {
		flat[i] = 120
	}
	lo, ok = AutoRange(flat, w, h, metal, other)
	if ok || lo != 128 {
		t.Fatalf("AutoRange on a flat picture = %d,%v, want 128,false", lo, ok)
	}
}
