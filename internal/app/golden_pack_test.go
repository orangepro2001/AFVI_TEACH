package app

import (
	"testing"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
)

// goldenTestPicture returns a w*h grey picture filled with v.
func goldenTestPicture(w, h int, v uint8) *core.GrayImage {
	img := &core.GrayImage{Width: w, Height: h, Data: make([]uint8, w*h)}
	for i := range img.Data {
		img.Data[i] = v
	}
	return img
}

// goldenTestMask returns a w*h mask with the given pixels set to 1.
func goldenTestMask(w, h int, set ...[2]int) *golden.Mask {
	m := golden.NewMask(w, h)
	for _, p := range set {
		if p[0] >= 0 && p[0] < w && p[1] >= 0 && p[1] < h {
			m.Pix[p[1]*w+p[0]] = 1
		}
	}
	return m
}

// goldenPixel returns the RGB of pixel (x, y) of an RGBA buffer.
func goldenPixel(buf []byte, w, x, y int) [3]uint8 {
	o := 4 * (y*w + x)
	return [3]uint8{buf[o], buf[o+1], buf[o+2]}
}

func TestDefaultGoldenView(t *testing.T) {
	v := defaultGoldenView()
	if v.Color != [3]uint8{255, 0, 255} {
		t.Fatalf("color = %v, want magenta", v.Color)
	}
	if v.Opacity != 50 {
		t.Fatalf("opacity = %d, want 50", v.Opacity)
	}
	if !v.ShowMask || !v.ShowSR {
		t.Fatalf("default shows = %+v, want mask and SR on", v)
	}
	if v.ShowBand {
		t.Fatalf("default shows band, want it off")
	}
}

func TestPackGoldenNilSource(t *testing.T) {
	if got := PackGolden(nil, nil, nil, nil, defaultGoldenView()); got != nil {
		t.Fatalf("PackGolden(nil) = %v, want nil", got)
	}
}

func TestPackGoldenOpacity(t *testing.T) {
	const w, h = 6, 4
	src := goldenTestPicture(w, h, 100)
	metal := goldenTestMask(w, h, [2]int{1, 1}, [2]int{2, 1})
	view := defaultGoldenView()
	buf := PackGolden(src, metal, metal, nil, view)
	PutRGBA(buf)
	if got := goldenPixel(buf, w, 1, 1); got != [3]uint8{178, 50, 178} {
		t.Fatalf("metal pixel = %v, want (178,50,178)", got)
	}
	if got := goldenPixel(buf, w, 0, 0); got != [3]uint8{100, 100, 100} {
		t.Fatalf("plain pixel = %v, want grey 100", got)
	}

	view.Opacity = 0
	buf = PackGolden(src, metal, metal, nil, view)
	if got := goldenPixel(buf, w, 1, 1); got != [3]uint8{100, 100, 100} {
		t.Fatalf("opacity 0 pixel = %v, want grey 100", got)
	}
	PutRGBA(buf)

	view.Opacity = 100
	buf = PackGolden(src, metal, metal, nil, view)
	if got := goldenPixel(buf, w, 1, 1); got != [3]uint8{255, 0, 255} {
		t.Fatalf("opacity 100 pixel = %v, want pure magenta", got)
	}
	PutRGBA(buf)

	view.Opacity = 50
	view.Color = [3]uint8{0, 0, 255}
	buf = PackGolden(src, metal, metal, nil, view)
	if got := goldenPixel(buf, w, 1, 1); got != [3]uint8{50, 50, 178} {
		t.Fatalf("custom colour pixel = %v, want (50,50,178)", got)
	}
	PutRGBA(buf)
}

func TestPackGoldenFlagsOff(t *testing.T) {
	const w, h = 6, 4
	src := goldenTestPicture(w, h, 100)
	metal := goldenTestMask(w, h, [2]int{1, 1})
	sr := goldenTestMask(w, h, [2]int{2, 2})
	view := goldenView{Color: [3]uint8{255, 0, 0}, Opacity: 50}
	buf := PackGolden(src, metal, metal, sr, view)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if got := goldenPixel(buf, w, x, y); got != [3]uint8{100, 100, 100} {
				t.Fatalf("pixel (%d,%d) = %v, want grey 100", x, y, got)
			}
		}
	}
	PutRGBA(buf)
}

func TestPackGoldenSRAndBand(t *testing.T) {
	const w, h = 6, 4
	src := goldenTestPicture(w, h, 100)
	metal := goldenTestMask(w, h, [2]int{1, 1}, [2]int{2, 1})
	inspect := goldenTestMask(w, h, [2]int{1, 1})
	sr := goldenTestMask(w, h, [2]int{3, 2})
	view := defaultGoldenView()

	// Band off by default: the band pixel stays the plain grey picture.
	buf := PackGolden(src, metal, inspect, sr, view)
	if got := goldenPixel(buf, w, 2, 1); got != [3]uint8{100, 100, 100} {
		t.Fatalf("band pixel (band off) = %v, want grey 100", got)
	}
	if got := goldenPixel(buf, w, 3, 2); got != [3]uint8{50, 150, 50} {
		t.Fatalf("sr pixel = %v, want (50,150,50)", got)
	}
	if got := goldenPixel(buf, w, 1, 1); got != [3]uint8{178, 50, 178} {
		t.Fatalf("inspect pixel = %v, want (178,50,178)", got)
	}
	PutRGBA(buf)

	// Band on: azure {70,140,255} at 50% over grey 100 -> (85,120,178).
	view.ShowBand = true
	buf = PackGolden(src, metal, inspect, sr, view)
	if want := goldenBlend([3]uint8{100, 100, 100}, goldenBandColor, 50); want != ([3]uint8{85, 120, 178}) {
		t.Fatalf("blend formula = %v, want (85,120,178)", want)
	}
	if got := goldenPixel(buf, w, 2, 1); got != [3]uint8{85, 120, 178} {
		t.Fatalf("band pixel (band on) = %v, want (85,120,178)", got)
	}
	PutRGBA(buf)
}

func TestPackGoldenOrder(t *testing.T) {
	const w, h = 6, 4
	src := goldenTestPicture(w, h, 100)
	metal := goldenTestMask(w, h, [2]int{2, 2})
	sr := goldenTestMask(w, h, [2]int{2, 2})
	view := defaultGoldenView()
	buf := PackGolden(src, metal, metal, sr, view)
	// Magenta at 50% over grey 100 is (178,50,178); green at 50% over that is (89,125,89).
	if got := goldenPixel(buf, w, 2, 2); got != [3]uint8{89, 125, 89} {
		t.Fatalf("overlapping pixel = %v, want SR painted last (89,125,89)", got)
	}
	PutRGBA(buf)
}

func TestPackGoldenNoInspectFallsBackToMetal(t *testing.T) {
	const w, h = 6, 4
	src := goldenTestPicture(w, h, 100)
	metal := goldenTestMask(w, h, [2]int{1, 1})
	view := defaultGoldenView()
	buf := PackGolden(src, metal, nil, nil, view)
	if got := goldenPixel(buf, w, 1, 1); got != [3]uint8{178, 50, 178} {
		t.Fatalf("metal pixel without inspect = %v, want (178,50,178)", got)
	}
	PutRGBA(buf)
}

func TestPackGoldenSizeMismatchIgnored(t *testing.T) {
	const w, h = 6, 4
	src := goldenTestPicture(w, h, 100)
	metal := goldenTestMask(3, 2, [2]int{1, 1})
	sr := goldenTestMask(3, 2, [2]int{1, 1})
	view := defaultGoldenView()
	buf := PackGolden(src, metal, metal, sr, view)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if got := goldenPixel(buf, w, x, y); got != [3]uint8{100, 100, 100} {
				t.Fatalf("pixel (%d,%d) = %v, want grey 100", x, y, got)
			}
		}
	}
	PutRGBA(buf)
}
