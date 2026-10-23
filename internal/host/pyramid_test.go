package host

import "testing"

func rgbaOf(w, h int, f func(x, y int) [3]byte) []byte {
	b := make([]byte, 4*w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := f(x, y)
			o := 4 * (y*w + x)
			b[o], b[o+1], b[o+2], b[o+3] = c[0], c[1], c[2], 255
		}
	}
	return b
}

func TestPyramidGreyRGBAIsServedAsOneByte(t *testing.T) {
	src := rgbaOf(8, 6, func(x, y int) [3]byte { v := byte(10 * x); return [3]byte{v, v, v} })
	p := newPyramid(src, 8, 6, false)
	pix, w, h, gray := p.get(0)
	if !gray || w != 8 || h != 6 || len(pix) != 48 || pix[3] != 30 {
		t.Fatalf("level 0: gray=%v %dx%d len=%d", gray, w, h, len(pix))
	}
	pix, w, h, gray = p.get(1)
	if !gray || w != 4 || h != 3 || pix[0] != 5 { // (0+10+0+10)/4
		t.Fatalf("level 1: gray=%v %dx%d pix0=%d", gray, w, h, pix[0])
	}
}

func TestPyramidColourStaysRGBAAndOddSizesReplicateEdges(t *testing.T) {
	src := rgbaOf(5, 3, func(x, y int) [3]byte { return [3]byte{200, 100, 0} })
	p := newPyramid(src, 5, 3, false)
	pix, w, h, gray := p.get(1)
	if gray || w != 3 || h != 2 || len(pix) != 4*6 || pix[0] != 200 || pix[1] != 100 || pix[2] != 0 || pix[3] != 255 {
		t.Fatalf("colour level 1: gray=%v %dx%d len=%d %v", gray, w, h, len(pix), pix[:4])
	}
}

func TestPyramidClampsLevels(t *testing.T) {
	p := newPyramid(make([]byte, 16), 4, 4, true)
	_, w, h, _ := p.get(99)
	if w != 1 || h != 1 {
		t.Fatalf("coarsest level %dx%d, want 1x1", w, h)
	}
	if _, w, _, _ := p.get(-3); w != 4 {
		t.Fatalf("negative level width %d", w)
	}
}
