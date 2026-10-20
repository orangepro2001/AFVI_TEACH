package app

import (
	"testing"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
)

// refinePicture builds a w*h grey picture filled with bg.
func refinePicture(w, h int, bg uint8) *core.GrayImage {
	img := &core.GrayImage{Width: w, Height: h, Data: make([]uint8, w*h)}
	for i := range img.Data {
		img.Data[i] = bg
	}
	return img
}

// refineFill sets the half-open pixel box to gv.
func refineFill(img *core.GrayImage, x0, y0, x1, y1 int, gv uint8) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.Data[y*img.Width+x] = gv
		}
	}
}

// refineBlock builds a w*h mask with the half-open pixel box set.
func refineBlock(w, h, x0, y0, x1, y1 int) *golden.Mask {
	m := golden.NewMask(w, h)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			m.Pix[y*w+x] = 1
		}
	}
	return m
}

func TestGoldenRefineSRSnapsToPicture(t *testing.T) {
	img := refinePicture(60, 40, 40)
	refineFill(img, 10, 8, 50, 32, 16)
	sr := refineBlock(60, 40, 13, 8, 53, 32)

	got := goldenRefineSR(img, sr)
	want := refineBlock(60, 40, 10, 8, 50, 32)
	if !got.Equal(want) {
		t.Fatalf("refined SR does not match the real block: got %d px, want %d px", got.Count(), want.Count())
	}
	if got.Equal(sr) {
		t.Fatalf("refined SR must not stay at the shifted layer")
	}
	if got == sr {
		t.Fatalf("refined SR must be a new mask")
	}
}

func TestGoldenRefineSRFlatPictureKeepsLayer(t *testing.T) {
	img := refinePicture(60, 40, 100)
	sr := refineBlock(60, 40, 13, 8, 53, 32)

	got := goldenRefineSR(img, sr)
	if !got.Equal(sr) {
		t.Fatalf("flat picture should keep the layer: got %d px, want %d px", got.Count(), sr.Count())
	}
}

func TestGoldenRefineSRSameAsLayer(t *testing.T) {
	img := refinePicture(60, 40, 40)
	refineFill(img, 13, 8, 53, 32, 16)
	sr := refineBlock(60, 40, 13, 8, 53, 32)

	got := goldenRefineSR(img, sr)
	if !got.Equal(sr) {
		t.Fatalf("aligned layer should be preserved: got %d px, want %d px", got.Count(), sr.Count())
	}
}

func TestGoldenRefineSRNilAndMismatch(t *testing.T) {
	sr := refineBlock(60, 40, 13, 8, 53, 32)
	got := goldenRefineSR(nil, sr)
	if got == nil || !got.Equal(sr) {
		t.Fatalf("nil picture should clone the layer")
	}
	if &got.Pix[0] == &sr.Pix[0] {
		t.Fatalf("nil picture must return a copy, not the layer itself")
	}

	small := refinePicture(30, 20, 40)
	got = goldenRefineSR(small, sr)
	if got == nil || !got.Equal(sr) {
		t.Fatalf("size mismatch should clone the layer")
	}
	if &got.Pix[0] == &sr.Pix[0] {
		t.Fatalf("size mismatch must return a copy, not the layer itself")
	}

	if got := goldenRefineSR(nil, nil); got != nil {
		t.Fatalf("nil layer should yield nil")
	}
}

func TestGoldenRefineSRSmallInteriorUnchanged(t *testing.T) {
	img := refinePicture(60, 40, 40)
	refineFill(img, 5, 5, 52, 33, 16)
	sr := refineBlock(60, 40, 5, 5, 15, 15)

	got := goldenRefineSR(img, sr)
	if !got.Equal(sr) {
		t.Fatalf("small layer should be unchanged")
	}
}
