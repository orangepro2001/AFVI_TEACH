package inspect

import (
	"reflect"
	"testing"
)

// naiveSkipExclusion is the reference for Regions.Skip: a skipped pixel is excluded, and so is
// every pixel within r (Chebyshev) of a pixel whose material or skip flag differs.
func naiveSkipExclusion(reg Regions, r int) []uint8 {
	w, h := reg.Width, reg.Height
	out := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if reg.Skip[i] {
				out[i] = 1
				continue
			}
			for dy := -r; dy <= r && out[i] == 0; dy++ {
				for dx := -r; dx <= r; dx++ {
					nx, ny := x+dx, y+dy
					if nx < 0 || ny < 0 || nx >= w || ny >= h {
						continue
					}
					j := ny*w + nx
					if reg.Material[j] != reg.Material[i] || reg.Skip[j] != reg.Skip[i] {
						out[i] = 1
						break
					}
				}
			}
		}
	}
	return out
}

func TestEdgeExclusionWithSkip(t *testing.T) {
	w, h := 40, 30
	reg := Regions{Width: w, Height: h, Material: make([]uint8, w*h), Skip: make([]bool, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			reg.Material[y*w+x] = uint8(x / 20)
			// A no-data band along the top and the left edge, like the border an alignment shift opens.
			reg.Skip[y*w+x] = y < 4 || x < 3
		}
	}
	for _, r := range []int{1, 2, 5} {
		got := edgeExclusion(reg, r)
		if want := naiveSkipExclusion(reg, r); !reflect.DeepEqual(got, want) {
			t.Errorf("r=%d: exclusion differs from the reference", r)
		}
	}
	// Without an edge margin only the skipped pixels themselves are excluded.
	got := edgeExclusion(reg, 0)
	for i, s := range reg.Skip {
		if (got[i] == 1) != s {
			t.Fatalf("r=0: pixel %d excluded=%v, skip=%v", i, got[i] == 1, s)
		}
	}
	// No Skip and no margin: nothing is excluded (nil), as before.
	if edgeExclusion(Regions{Width: w, Height: h, Material: reg.Material}, 0) != nil {
		t.Error("r=0 without Skip must return nil")
	}
}

// A flat unit whose no-data border was filled with a wrong value must be OK when the border
// is skipped, and NG when it is not.
func TestInspectIgnoresSkippedPixels(t *testing.T) {
	w, h := 60, 40
	reg := Regions{Width: w, Height: h, Material: make([]uint8, w*h)}
	for i := range reg.Material {
		reg.Material[i] = uint8(SR)
	}
	pix := make([]uint8, w*h)
	skip := make([]bool, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			pix[y*w+x] = 100
			if y < 5 {
				pix[y*w+x] = 10 // the replicated border, far outside the accepted interval
				skip[y*w+x] = true
			}
		}
	}
	p := DefaultParams()
	p.Materials[SR].Low, p.Materials[SR].High = 80, 120
	img := Image{Width: w, Height: h, Pix: pix}

	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) == 0 {
		t.Fatal("the dark border was not flagged without Skip")
	}
	reg.Skip = skip
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Errorf("%d defects although the dark border is skipped: %+v", len(rep.Defects), rep.Defects)
	}
	if rep.EdgeIgnored != 5*w {
		t.Errorf("EdgeIgnored = %d, want %d skipped pixels", rep.EdgeIgnored, 5*w)
	}
	// A wrongly sized mask is refused.
	reg.Skip = skip[:len(skip)-1]
	if _, err := Inspect(img, reg, p); err == nil {
		t.Error("a Skip mask of the wrong size was accepted")
	}
}
