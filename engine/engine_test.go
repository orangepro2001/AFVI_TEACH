package engine

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// synth makes a grey plate with a material map: a metal pad on a resist body.
func synth(w, h int) (*Gray, []uint8) {
	g := &Gray{W: w, H: h, Data: make([]uint8, w*h)}
	mat := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if x > w/3 && x < 2*w/3 && y > h/3 && y < 2*h/3 {
				mat[i] = 3 // metal
				g.Data[i] = 196
			} else {
				mat[i] = 1 // SR
				g.Data[i] = 96
			}
		}
	}
	return g, mat
}

func TestLearnAndInspectClean(t *testing.T) {
	good, mat := synth(120, 120)
	p, err := Learn(good, mat, 4)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.Materials["metal"]; !ok {
		t.Fatalf("learn did not produce a metal range: %+v", p.Materials)
	}
	if p.Materials["base"].Enabled {
		t.Fatal("base (the plate margin) must not be inspected")
	}
	// a clean plate passes
	res, err := Inspect(good, mat, p)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != "ok" || res.DefectCount != 0 {
		t.Fatalf("a clean plate reported %s with %d defects", res.Verdict, res.DefectCount)
	}
	if len(res.Materials) != 4 {
		t.Fatalf("materials = %d, want 4", len(res.Materials))
	}
}

func TestInspectFindsAPlantedDefect(t *testing.T) {
	good, mat := synth(120, 120)
	p, err := Learn(good, mat, 4)
	if err != nil {
		t.Fatal(err)
	}
	// a bright speck on the metal pad
	bad := &Gray{W: good.W, H: good.H, Data: append([]uint8(nil), good.Data...)}
	for y := 50; y < 70; y++ {
		for x := 50; x < 70; x++ {
			bad.Data[y*bad.W+x] = 250
		}
	}
	res, err := Inspect(bad, mat, p)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != "ng" || res.DefectCount == 0 {
		t.Fatalf("the speck was not found: %+v", res)
	}
	d := res.Defects[0]
	if d.Material != "metal" || d.Polarity != "bright" || d.Area < 100 {
		t.Fatalf("unexpected defect %+v", d)
	}
	if d.X < 45 || d.Y < 45 || d.X+d.W > 75 {
		t.Fatalf("the bounding box is off: %+v", d)
	}
}

func TestDecodePNG(t *testing.T) {
	src := image.NewGray(image.Rect(0, 0, 8, 6))
	for i := range src.Pix {
		src.Pix[i] = uint8(i * 5)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	g, err := Decode(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if g.W != 8 || g.H != 6 {
		t.Fatalf("decoded %dx%d", g.W, g.H)
	}
	// grey input must survive unchanged
	for i := range src.Pix {
		if g.Data[i] != src.Pix[i] {
			t.Fatalf("pixel %d changed: %d -> %d", i, src.Pix[i], g.Data[i])
		}
	}
	if _, err := Decode(nil); err != ErrNoImage {
		t.Fatalf("empty input: %v", err)
	}
	if _, err := Decode([]byte("not an image")); err == nil {
		t.Fatal("garbage decoded")
	}
}

func TestColorToGrey(t *testing.T) {
	// pure red becomes the luma of red, which is what color.GrayModel gives
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.Set(0, 0, color.RGBA{255, 0, 0, 255})
	g := FromImage(src)
	want := color.GrayModel.Convert(color.RGBA{255, 0, 0, 255}).(color.Gray).Y
	if g.Data[0] != want {
		t.Fatalf("red -> %d, want %d", g.Data[0], want)
	}
}

func TestSizeMismatch(t *testing.T) {
	g, mat := synth(20, 20)
	if _, err := Inspect(g, mat[:10], Params{}); err != ErrSizeMatch {
		t.Fatalf("mismatched material map: %v", err)
	}
	if _, err := Learn(g, mat[:10], 4); err != ErrSizeMatch {
		t.Fatalf("mismatched learn: %v", err)
	}
}
