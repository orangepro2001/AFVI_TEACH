package core

import (
	"math"
	"testing"
)

func TestRealignInputIdentityKeepsTheInput(t *testing.T) {
	img := makeTexture(120, 90, 5)
	re := RealignInput(img, Similarity{Scale: 1}, nil)
	for i, v := range img.Data {
		if re.Image.Data[i] != v || !re.Valid[i] {
			t.Fatalf("pixel %d: %d valid=%v, want %d valid", i, re.Image.Data[i], re.Valid[i], v)
		}
	}
}

// An integer shift is undone exactly: the pixel (x, y) of the result is the input at fit(x, y), so
// with the similarity template -> input the template comes back, apart from the strip that has no data.
func TestRealignInputDirection(t *testing.T) {
	const width, height = 160, 120
	template := makeTexture(width, height, 9)
	warp := warpParams(func(p *WarpParams) {
		p.Tx = 5
		p.Ty = 4
	})
	input := ApplyWarp(template, &warp, 9)
	fit := WarpToSimilarity(&warp, width, height)
	re := RealignInput(input, fit, nil)
	used := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			inside := x+5 <= width-1 && y+4 <= height-1
			if re.Valid[i] != inside {
				t.Fatalf("(%d,%d): valid=%v, want %v", x, y, re.Valid[i], inside)
			}
			if !inside {
				if re.Image.Data[i] != 0 {
					t.Fatalf("(%d,%d): no-data pixel is %d, want 0", x, y, re.Image.Data[i])
				}
				continue
			}
			used++
			if re.Image.Data[i] != template.Data[i] {
				t.Fatalf("(%d,%d): %d, want the template's %d", x, y, re.Image.Data[i], template.Data[i])
			}
		}
	}
	if mean, n := MeanAbsDiff(template, re.Image, re.Valid); mean != 0 || n != used {
		t.Errorf("mean difference %v over %d pixels, want 0 over %d", mean, n, used)
	}
}

// The fit found by the matcher (not the true warp) brings a rotated and shifted input back onto the
// template: the difference over the valid pixels drops a lot.
func TestRealignInputWithTheFoundFit(t *testing.T) {
	const width, height = 400, 300
	template := makeTexture(width, height, 21)
	warp := warpParams(func(p *WarpParams) {
		p.Tx = 7
		p.Ty = -5
		p.ThetaDeg = 0.3
	})
	input := ApplyWarp(template, &warp, 21)
	rois := []AlignRoi{
		{Name: "TL", Rect: Rect{X: 40, Y: 40, W: 70, H: 70}},
		{Name: "TR", Rect: Rect{X: width - 120, Y: 40, W: 70, H: 70}},
		{Name: "BL", Rect: Rect{X: 40, Y: height - 120, W: 70, H: 70}},
		{Name: "BR", Rect: Rect{X: width - 120, Y: height - 120, W: 70, H: 70}},
	}
	rep := AlignWithRois(template, input, rois, DefaultAlignConfig, Similarity{Scale: 1})
	if rep.Fit == nil || rep.Level != VerdictOK {
		t.Fatalf("no usable fit: level %s reasons %+v", rep.Level, rep.Reasons)
	}
	re := RealignInput(input, *rep.Fit, nil)
	before, nb := MeanAbsDiff(template, input, re.Valid)
	after, na := MeanAbsDiff(template, re.Image, re.Valid)
	if nb != na || na < width*height/2 {
		t.Fatalf("pixels used: before %d after %d", nb, na)
	}
	if !(after < before/4) {
		t.Errorf("mean difference %.2f -> %.2f, want at least a four-fold drop", before, after)
	}
	// The realigned image must be about as close to the template as the one made with the true warp.
	oracle := RealignInput(input, WarpToSimilarity(&warp, width, height), nil)
	best, _ := MeanAbsDiff(template, oracle.Image, re.Valid)
	if after > best+1.5 {
		t.Errorf("found-fit difference %.2f is far from the true-warp difference %.2f", after, best)
	}
}

func TestRealignInputExcludesRectangles(t *testing.T) {
	img := makeTexture(100, 80, 3)
	re := RealignInput(img, Similarity{Scale: 1}, []Rect{{X: 20, Y: 10, W: 30, H: 20}})
	for _, c := range []struct {
		x, y  int
		valid bool
	}{{35, 20, false}, {20, 10, false}, {19, 9, false}, {18, 20, true}, {35, 33, true}, {52, 20, true}, {70, 60, true}} {
		if got := re.Valid[c.y*100+c.x]; got != c.valid {
			t.Errorf("(%d,%d): valid=%v, want %v", c.x, c.y, got, c.valid)
		}
		if !c.valid && re.Image.Data[c.y*100+c.x] != 0 {
			t.Errorf("(%d,%d): excluded pixel is %d, want 0", c.x, c.y, re.Image.Data[c.y*100+c.x])
		}
	}
}

func TestMeanAbsDiff(t *testing.T) {
	a := &GrayImage{Width: 2, Height: 2, Data: []uint8{10, 20, 30, 40}}
	b := &GrayImage{Width: 2, Height: 2, Data: []uint8{12, 17, 30, 50}}
	if m, n := MeanAbsDiff(a, b, nil); n != 4 || math.Abs(m-3.75) > 1e-12 {
		t.Errorf("all pixels: %v over %d, want 3.75 over 4", m, n)
	}
	if m, n := MeanAbsDiff(a, b, []bool{true, false, true, false}); n != 2 || math.Abs(m-1) > 1e-12 {
		t.Errorf("masked: %v over %d, want 1 over 2", m, n)
	}
	if m, n := MeanAbsDiff(a, b, []bool{false, false, false, false}); n != 0 || !math.IsNaN(m) {
		t.Errorf("empty mask: %v over %d, want NaN over 0", m, n)
	}
}

func TestCheckRealignedMeasuresTheLeftoverShift(t *testing.T) {
	const width, height = 800, 600
	template := makeTexture(width, height, 61)
	warp := warpParams(func(p *WarpParams) {
		p.Tx = 5
		p.Ty = 4
	})
	input := ApplyWarp(template, &warp, 61)
	exact := WarpToSimilarity(&warp, width, height)

	perfect := CheckRealigned(template, RealignInput(input, exact, nil), 2)
	if perfect.Total < 10 || perfect.Windows < perfect.Total-2 {
		t.Fatalf("perfect fit: %d of %d windows found", perfect.Windows, perfect.Total)
	}
	if perfect.MaxPx > 0.2 {
		t.Errorf("perfect fit leaves %.2f px", perfect.MaxPx)
	}

	// A fit that is 3 px off in x brings the whole picture 3 px off, wherever the ROIs were.
	off := exact
	off.Tx += 3
	bad := CheckRealigned(template, RealignInput(input, off, nil), 2)
	if bad.Windows < bad.Total-2 {
		t.Fatalf("shifted fit: %d of %d windows found", bad.Windows, bad.Total)
	}
	if math.Abs(bad.MaxPx-3) > 0.4 || math.Abs(bad.RmsPx-3) > 0.4 {
		t.Errorf("shifted fit: max %.2f rms %.2f px, want about 3", bad.MaxPx, bad.RmsPx)
	}
	if bad.Worst.X <= 0 || bad.Worst.Y <= 0 || bad.Win != 40 {
		t.Errorf("worst window not reported: %+v (window %d)", bad.Worst, bad.Win)
	}

	// An image that is too small for the grid, and one that has no data at all, give an empty check.
	if c := CheckRealigned(makeTexture(100, 80, 1), RealignInput(makeTexture(100, 80, 1), Similarity{Scale: 1}, nil), 4); c.Total != 0 {
		t.Errorf("tiny image: %+v", c)
	}
	none := RealignInput(input, Similarity{Tx: 5000, Scale: 1}, nil)
	if c := CheckRealigned(template, none, 2); c.Total != 0 || c.Windows != 0 {
		t.Errorf("no data: %+v", c)
	}
}
