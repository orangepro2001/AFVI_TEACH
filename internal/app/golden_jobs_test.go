package app

import (
	"testing"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
)

// goldenBlockPicture builds a w*h picture: vBlock inside the block rect, vBack elsewhere.
func goldenBlockPicture(w, h int, x0, y0, x1, y1 int, vBlock, vBack uint8) *core.GrayImage {
	img := &core.GrayImage{Width: w, Height: h, Data: make([]uint8, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := vBack
			if x >= x0 && x < x1 && y >= y0 && y < y1 {
				v = vBlock
			}
			img.Data[y*w+x] = v
		}
	}
	return img
}

// goldenBlockMask builds a mask that is set inside the block rect.
func goldenBlockMask(w, h int, x0, y0, x1, y1 int) *golden.Mask {
	m := golden.NewMask(w, h)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x >= 0 && x < w && y >= 0 && y < h {
				m.Pix[y*w+x] = 1
			}
		}
	}
	return m
}

// goldenNotMask returns the complement of m.
func goldenNotMask(m *golden.Mask) *golden.Mask {
	out := golden.NewMask(m.W, m.H)
	for i, v := range m.Pix {
		if v == 0 {
			out.Pix[i] = 1
		}
	}
	return out
}

// goldenBlockEqual reports whether m is set exactly inside the block rect.
func goldenBlockEqual(m *golden.Mask, w, h, x0, y0, x1, y1 int) bool {
	return m != nil && m.Equal(goldenBlockMask(w, h, x0, y0, x1, y1))
}

func TestGoldenAutoMetalBrightBlock(t *testing.T) {
	const w, h = 40, 30
	src := goldenBlockPicture(w, h, 10, 8, 28, 22, 200, 40)
	geo := &goldenGeometry{
		srReal:    goldenNotMask(goldenBlockMask(w, h, 10, 8, 28, 22)),
		metalReal: goldenBlockMask(w, h, 10, 8, 28, 22),
	}
	auto := goldenAutoMetal(src, geo)
	if auto.Fallback {
		t.Fatalf("unexpected fallback: %+v", auto)
	}
	if auto.Low < 115 || auto.Low > 125 {
		t.Fatalf("low = %d, want near 120", auto.Low)
	}
	if auto.High != 255 || auto.OutRange {
		t.Fatalf("range = [%d,%d] out=%v, want [lo,255] in range", auto.Low, auto.High, auto.OutRange)
	}
	if !goldenBlockEqual(auto.Metal, w, h, 10, 8, 28, 22) {
		t.Fatalf("metal differs from the block after Open(1)")
	}
}

func TestGoldenAutoMetalDarkBlock(t *testing.T) {
	const w, h = 40, 30
	src := goldenBlockPicture(w, h, 10, 8, 28, 22, 40, 200)
	geo := &goldenGeometry{
		srReal:    goldenNotMask(goldenBlockMask(w, h, 10, 8, 28, 22)),
		metalReal: goldenBlockMask(w, h, 10, 8, 28, 22),
	}
	auto := goldenAutoMetal(src, geo)
	if auto.Fallback {
		t.Fatalf("unexpected fallback: %+v", auto)
	}
	if auto.Low != 0 {
		t.Fatalf("low = %d, want 0", auto.Low)
	}
	if auto.High < 115 || auto.High > 125 {
		t.Fatalf("high = %d, want near 119", auto.High)
	}
	if auto.OutRange {
		t.Fatalf("out range should be false")
	}
	if !goldenBlockEqual(auto.Metal, w, h, 10, 8, 28, 22) {
		t.Fatalf("metal differs from the block after Open(1)")
	}
}

func TestGoldenAutoMetalFlatFallsBack(t *testing.T) {
	const w, h = 40, 30
	src := goldenTestPicture(w, h, 120)
	metalReal := goldenBlockMask(w, h, 10, 8, 28, 22)
	geo := &goldenGeometry{
		srReal:    goldenBlockMask(w, h, 0, 0, w, h),
		metalReal: metalReal,
	}
	auto := goldenAutoMetal(src, geo)
	if !auto.Fallback {
		t.Fatalf("flat picture should fall back: %+v", auto)
	}
	if auto.Low != 128 || auto.High != 255 || auto.OutRange {
		t.Fatalf("fallback range = [%d,%d] out=%v, want [128,255] in range", auto.Low, auto.High, auto.OutRange)
	}
	if auto.Metal == nil || !auto.Metal.Equal(metalReal) {
		t.Fatalf("fallback metal should equal metalReal")
	}
}

func TestGoldenAutoMetalNilGeometry(t *testing.T) {
	const w, h = 40, 30
	src := goldenTestPicture(w, h, 120)
	auto := goldenAutoMetal(src, nil)
	if !auto.Fallback {
		t.Fatalf("nil geometry should fall back")
	}
	if auto.Low != 128 || auto.High != 255 || auto.OutRange {
		t.Fatalf("fallback range = [%d,%d] out=%v, want [128,255] in range", auto.Low, auto.High, auto.OutRange)
	}
	if auto.Metal == nil || !auto.Metal.Valid() || auto.Metal.W != w || auto.Metal.H != h || auto.Metal.Count() != 0 {
		t.Fatalf("nil geometry metal should be empty %dx%d", w, h)
	}
}

func TestGoldenBuildGeometryErrors(t *testing.T) {
	if _, err := goldenBuildGeometry(nil, nil); err == nil {
		t.Fatalf("nil scene should error")
	}
	scene := &Scene{W: 8, H: 6, Material: make([]uint8, 48)}
	ref := goldenTestPicture(4, 4, 10)
	if _, err := goldenBuildGeometry(scene, nil); err == nil {
		t.Fatalf("nil reference should error")
	}
	if _, err := goldenBuildGeometry(scene, ref); err == nil {
		t.Fatalf("size mismatch should error")
	}
}

func TestGoldenBuildGeometrySyntheticScene(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	if a.Scene == nil {
		t.Fatalf("test app has no scene")
	}
	scene := a.Scene
	if scene.W != 96 || scene.H != 64 {
		t.Fatalf("scene size = %dx%d, want 96x64", scene.W, scene.H)
	}
	ref := goldenSourceImage(scene, a.Light, core.SensorNoise{}, core.ChRed)
	if ref == nil || ref.Width != scene.W || ref.Height != scene.H {
		t.Fatalf("source image has the wrong size")
	}
	geo, err := goldenBuildGeometry(scene, ref)
	if err != nil {
		t.Fatalf("goldenBuildGeometry: %v", err)
	}
	if geo.registered {
		t.Fatalf("the synthetic picture should not register")
	}
	var srN, metalN int
	for _, v := range scene.Material {
		switch v {
		case core.MatSR:
			srN++
		case core.MatMetal:
			metalN++
		}
	}
	if geo.srReal == nil || geo.srReal.Count() != srN {
		t.Fatalf("srReal count = %d, want %d", geo.srReal.Count(), srN)
	}
	if geo.metalReal == nil || geo.metalReal.Count() != metalN {
		t.Fatalf("metalReal count = %d, want %d", geo.metalReal.Count(), metalN)
	}
}
