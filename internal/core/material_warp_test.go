package core

import (
	"math"
	"math/rand"
	"testing"
)

func TestWarpMaterialToReferenceIdentity(t *testing.T) {
	w, h := 37, 23
	material := make([]uint8, w*h)
	for i := range material {
		material[i] = uint8(i % 5)
	}
	out := WarpMaterialToReference(material, w, h, Similarity{Scale: 1}, 1)
	if out == nil {
		t.Fatal("identity returned nil")
	}
	if &out[0] == &material[0] {
		t.Fatal("identity returned the same slice, want a copy")
	}
	if len(out) != len(material) {
		t.Fatalf("len = %d, want %d", len(out), len(material))
	}
	for i := range material {
		if out[i] != material[i] {
			t.Fatalf("pixel %d = %d, want %d", i, out[i], material[i])
		}
	}
}

func TestWarpMaterialToReferenceShift(t *testing.T) {
	w, h := 40, 30
	material := make([]uint8, w*h)
	// A 5x4 block of material 3 at x 10..14, y 10..13.
	for y := 10; y < 14; y++ {
		for x := 10; x < 15; x++ {
			material[y*w+x] = 3
		}
	}
	// A 2x2 block in the corner, to exercise clamping.
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			material[y*w+x] = 3
		}
	}
	// out(x, y) = material(x+Tx, y+Ty): the fit maps a reference point to the design point,
	// so Tx = -3, Ty = -2 moves the picture content by (+3, +2).
	out := WarpMaterialToReference(material, w, h, Similarity{Tx: -3, Ty: -2, Scale: 1}, 1)
	want := make([]uint8, w*h)
	for y := 12; y < 16; y++ {
		for x := 13; x < 18; x++ {
			want[y*w+x] = 3
		}
	}
	// The corner block is stretched by the clamp: sx = clamp(x-3) is 0 for x <= 3 and 1 for
	// x = 4, sy = clamp(y-2) is 0 for y <= 2 and 1 for y = 3.
	for y := 0; y < 4; y++ {
		for x := 0; x < 5; x++ {
			want[y*w+x] = 3
		}
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("pixel (%d,%d) = %d, want %d", i%w, i/w, out[i], want[i])
		}
	}
}

func TestWarpMaterialToReferenceFormula(t *testing.T) {
	w, h := 50, 40
	material := make([]uint8, w*h)
	rng := rand.New(rand.NewSource(12345))
	for i := range material {
		material[i] = uint8(rng.Intn(4))
	}
	fit := Similarity{Tx: 1.5, Ty: 0.5, Scale: 2}
	scale := 2
	out := WarpMaterialToReference(material, w, h, fit, scale)
	if out == nil {
		t.Fatal("got nil")
	}
	full := FitToFullRes(fit, scale)
	for i := 0; i < 2000; i++ {
		x := rng.Intn(w)
		y := rng.Intn(h)
		p := ApplySimilarity(full, Point{X: float64(x), Y: float64(y)})
		sx := int(math.Floor(p.X + 0.5))
		sy := int(math.Floor(p.Y + 0.5))
		if sx < 0 {
			sx = 0
		} else if sx > w-1 {
			sx = w - 1
		}
		if sy < 0 {
			sy = 0
		} else if sy > h-1 {
			sy = h - 1
		}
		got := out[y*w+x]
		want := material[sy*w+sx]
		if got != want {
			t.Fatalf("pixel (%d,%d) = %d, want %d", x, y, got, want)
		}
	}
}

func TestWarpMaterialToReferenceRotationKeepsCount(t *testing.T) {
	w, h := 200, 200
	material := make([]uint8, w*h)
	count := 0
	for y := 60; y < 140; y++ {
		for x := 60; x < 140; x++ {
			material[y*w+x] = 3
			count++
		}
	}
	out := WarpMaterialToReference(material, w, h, Similarity{ThetaDeg: 0.3, Scale: 1}, 1)
	if out == nil {
		t.Fatal("got nil")
	}
	got := 0
	for _, v := range out {
		if v == 3 {
			got++
		}
	}
	if diff := got - count; diff < 0 {
		diff = -diff
	} else if diff > count/50 {
		t.Fatalf("material count changed too much: got %d, want about %d", got, count)
	}
}

func TestWarpMaterialToReferenceInvalid(t *testing.T) {
	if out := WarpMaterialToReference(make([]uint8, 10), 4, 3, Similarity{Scale: 1}, 1); out != nil {
		t.Fatal("wrong length should return nil")
	}
	if out := WarpMaterialToReference(nil, 0, 5, Similarity{Scale: 1}, 1); out != nil {
		t.Fatal("w<=0 should return nil")
	}
	if out := WarpMaterialToReference(nil, 5, 0, Similarity{Scale: 1}, 1); out != nil {
		t.Fatal("h<=0 should return nil")
	}
}
