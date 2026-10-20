package core

import (
	"math"
	"testing"
)

// designFixture returns a 640x480 material map with several asymmetric shapes: three metal
// rectangles, an SR L-shape and an open block on a base background.
func designFixture() ([]uint8, int, int) {
	const w, h = 640, 480
	m := make([]uint8, w*h)
	rect := func(x0, y0, x1, y1 int, mat uint8) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				m[y*w+x] = mat
			}
		}
	}
	rect(164, 150, 268, 240, MatMetal)
	rect(456, 310, 560, 400, MatMetal)
	rect(250, 50, 350, 130, MatMetal)
	rect(370, 100, 410, 330, MatSR)
	rect(370, 290, 480, 330, MatSR)
	rect(160, 320, 240, 430, MatOpen)
	return m, w, h
}

func designLevels() [4]uint8 {
	return [4]uint8{22, 30, 25, 80}
}

// designHashNoise is a deterministic integer hash of (x, y) in [-15, 15].
func designHashNoise(x, y int) int {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return int(h%31) - 15
}

// texturedRef paints the shapes of material moved by (shiftX, shiftY) pixels with the per-material
// levels plus deterministic noise, like a real picture of a slightly misaligned product.
func texturedRef(material []uint8, w, h, shiftX, shiftY int) *GrayImage {
	levels := designLevels()
	data := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx := x - shiftX
			sy := y - shiftY
			mat := uint8(MatBase)
			if sx >= 0 && sy >= 0 && sx < w && sy < h {
				mat = material[sy*w+sx]
			}
			v := int(levels[mat]) + designHashNoise(x, y)
			if v < 0 {
				v = 0
			} else if v > 255 {
				v = 255
			}
			data[y*w+x] = uint8(v)
		}
	}
	return &GrayImage{Width: w, Height: h, Data: data}
}

func designRegionStd(img *GrayImage, x0, y0, size int) float64 {
	sum := 0.0
	sumSq := 0.0
	for y := y0; y < y0+size; y++ {
		for x := x0; x < x0+size; x++ {
			v := float64(img.Data[y*img.Width+x])
			sum += v
			sumSq += v * v
		}
	}
	n := float64(size * size)
	mean := sum / n
	variance := sumSq/n - mean*mean
	if variance < 0 {
		variance = 0
	}
	return math.Sqrt(variance)
}

// designFirstCrossing returns the first column in row whose value is at least threshold.
func designFirstCrossing(img *GrayImage, row int, threshold float64) int {
	for x := 0; x < img.Width; x++ {
		if float64(img.Data[row*img.Width+x]) >= threshold {
			return x
		}
	}
	return -1
}

func TestDesignImageUsesMaterialMeans(t *testing.T) {
	m, w, h := designFixture()
	ref := texturedRef(m, w, h, 0, 0)
	img, err := DesignImage(m, ref)
	if err != nil {
		t.Fatalf("DesignImage: %v", err)
	}
	if img.Width != w || img.Height != h || len(img.Data) != w*h {
		t.Fatalf("size = %dx%d len %d, want %dx%d", img.Width, img.Height, len(img.Data), w, h)
	}
	stats, err := MaterialStats(ref, m)
	if err != nil {
		t.Fatalf("MaterialStats: %v", err)
	}
	var level [4]uint8
	for mat := 0; mat < 4; mat++ {
		level[mat] = ClampByte(stats[mat].Mean)
	}
	for i, mat := range m {
		if img.Data[i] != level[mat] {
			t.Fatalf("pixel %d (material %d) = %d, want %d", i, mat, img.Data[i], level[mat])
		}
	}

	baseOnly := make([]uint8, len(m))
	baseStats, err := MaterialStats(ref, baseOnly)
	if err != nil {
		t.Fatalf("MaterialStats base only: %v", err)
	}
	if baseStats[MatSR].Count != 0 || baseStats[MatOpen].Count != 0 || baseStats[MatMetal].Count != 0 {
		t.Fatalf("materials without pixels must have count 0")
	}
	for mat := MatSR; mat <= MatMetal; mat++ {
		if ClampByte(baseStats[mat].Mean) != 0 {
			t.Fatalf("material %d without pixels must give level 0", mat)
		}
	}
	flat, err := DesignImage(baseOnly, ref)
	if err != nil {
		t.Fatalf("DesignImage base only: %v", err)
	}
	wantBase := ClampByte(baseStats[MatBase].Mean)
	for i, v := range flat.Data {
		if v != wantBase {
			t.Fatalf("base-only pixel %d = %d, want %d", i, v, wantBase)
		}
	}

	if _, err := DesignImage(m[:len(m)-1], ref); err == nil {
		t.Fatalf("DesignImage with a size mismatch must fail")
	}
}

func TestDesignTemplateHasNoTexture(t *testing.T) {
	m, w, h := designFixture()
	ref := texturedRef(m, w, h, 0, 0)
	before := ref.Clone()
	tpl, _, err := DesignTemplate(m, ref, 1)
	if err != nil {
		t.Fatalf("DesignTemplate: %v", err)
	}
	if tpl.Width != w || tpl.Height != h {
		t.Fatalf("template size = %dx%d, want %dx%d", tpl.Width, tpl.Height, w, h)
	}
	refStd := designRegionStd(ref, 20, 20, 40)
	tplStd := designRegionStd(tpl, 20, 20, 40)
	if refStd <= 5 {
		t.Fatalf("reference flat region std = %.2f, want > 5", refStd)
	}
	if tplStd >= 1 {
		t.Fatalf("template flat region std = %.2f, want < 1", tplStd)
	}

	tpl2, _, err := DesignTemplate(m, ref, 2)
	if err != nil {
		t.Fatalf("DesignTemplate scale 2: %v", err)
	}
	if tpl2.Width != w/2 || tpl2.Height != h/2 {
		t.Fatalf("scale 2 template size = %dx%d, want %dx%d", tpl2.Width, tpl2.Height, w/2, h/2)
	}

	diff, used := MeanAbsDiff(ref, before, nil)
	if used != len(ref.Data) || diff != 0 {
		t.Fatalf("DesignTemplate modified ref (diff %.3f over %d pixels)", diff, used)
	}
}

func TestDesignTemplateRegistersToTheReference(t *testing.T) {
	m, w, h := designFixture()
	ref := texturedRef(m, w, h, 4, -3)
	tpl, info, err := DesignTemplate(m, ref, 1)
	if err != nil {
		t.Fatalf("DesignTemplate: %v", err)
	}
	if !info.Registered {
		t.Fatalf("template was not registered: %+v", info)
	}
	if math.Abs(info.Fit.Tx-(-4)) > 1 {
		t.Fatalf("Fit.Tx = %.3f, want about -4", info.Fit.Tx)
	}
	if math.Abs(info.Fit.Ty-3) > 1 {
		t.Fatalf("Fit.Ty = %.3f, want about +3", info.Fit.Ty)
	}
	if math.Abs(info.Fit.ThetaDeg) >= 0.3 {
		t.Fatalf("Fit.ThetaDeg = %.3f, want |theta| < 0.3", info.Fit.ThetaDeg)
	}
	// The fixture has few shapes, most of the 30 windows lie on flat base and cannot be located.
	if info.Windows < 6 {
		t.Fatalf("located windows = %d, want at least 6: %+v", info.Windows, info)
	}
	if info.Inliers < 4 {
		t.Fatalf("inliers = %d, want at least 4: %+v", info.Inliers, info)
	}

	design, err := DesignImage(m, ref)
	if err != nil {
		t.Fatalf("DesignImage: %v", err)
	}
	row := 200
	threshold := (float64(design.Data[row*w+50]) + float64(design.Data[row*w+200])) / 2
	refEdge := designFirstCrossing(ref, row, threshold)
	tplEdge := designFirstCrossing(tpl, row, threshold)
	designEdge := designFirstCrossing(design, row, threshold)
	if refEdge < 0 || tplEdge < 0 || designEdge < 0 {
		t.Fatalf("edge not found: ref %d tpl %d design %d", refEdge, tplEdge, designEdge)
	}
	if math.Abs(float64(tplEdge-refEdge)) > 1 {
		t.Fatalf("registered template edge at %d, reference edge at %d", tplEdge, refEdge)
	}
	if off := refEdge - designEdge; off < 3 || off > 5 {
		t.Fatalf("unregistered design edge is %d px from the reference edge, want about 4", off)
	}
}

func TestDesignTemplateAlreadyAligned(t *testing.T) {
	m, w, h := designFixture()
	ref := texturedRef(m, w, h, 0, 0)
	tpl, info, err := DesignTemplate(m, ref, 1)
	if err != nil {
		t.Fatalf("DesignTemplate: %v", err)
	}
	if math.Abs(info.Fit.Tx) >= 0.5 || math.Abs(info.Fit.Ty) >= 0.5 {
		t.Fatalf("Fit = %+v, want |Tx|, |Ty| < 0.5", info.Fit)
	}
	if !info.Registered {
		t.Fatalf("unshifted reference was not registered: %+v", info)
	}
	design, err := DesignImage(m, ref)
	if err != nil {
		t.Fatalf("DesignImage: %v", err)
	}
	blurred := GaussianBlur(design, DesignBlurPx)
	diff, used := MeanAbsDiff(tpl, blurred, nil)
	if used == 0 {
		t.Fatalf("MeanAbsDiff used no pixels")
	}
	if diff >= 1 {
		t.Fatalf("registered template differs from the unregistered design by %.3f, want < 1", diff)
	}
}

func TestDesignTemplateSkipsTinyAndFlat(t *testing.T) {
	const w, h = 40, 30
	m := make([]uint8, w*h)
	for y := 10; y < 20; y++ {
		for x := 10; x < 20; x++ {
			m[y*w+x] = MatMetal
		}
	}
	ref := texturedRef(m, w, h, 0, 0)
	tpl, info, err := DesignTemplate(m, ref, 1)
	if err != nil {
		t.Fatalf("DesignTemplate tiny: %v", err)
	}
	if info.Registered {
		t.Fatalf("tiny image must not register")
	}
	if tpl.Width != w || tpl.Height != h {
		t.Fatalf("tiny template size = %dx%d, want %dx%d", tpl.Width, tpl.Height, w, h)
	}

	const fw, fh = 640, 480
	flatMaterial := make([]uint8, fw*fh)
	flatRef := NewGray(fw, fh, 128)
	flatTpl, flatInfo, err := DesignTemplate(flatMaterial, flatRef, 1)
	if err != nil {
		t.Fatalf("DesignTemplate flat: %v", err)
	}
	if flatInfo.Registered {
		t.Fatalf("flat image must not register")
	}
	if flatTpl.Width != fw || flatTpl.Height != fh {
		t.Fatalf("flat template size = %dx%d, want %dx%d", flatTpl.Width, flatTpl.Height, fw, fh)
	}
}

func TestDesignTemplateScale2(t *testing.T) {
	m, w, h := designFixture()
	ref := texturedRef(m, w, h, 4, -3)
	tpl, info, err := DesignTemplate(m, ref, 2)
	if err != nil {
		t.Fatalf("DesignTemplate: %v", err)
	}
	if tpl.Width != w/2 || tpl.Height != h/2 {
		t.Fatalf("scale 2 template size = %dx%d, want %dx%d", tpl.Width, tpl.Height, w/2, h/2)
	}
	if !info.Registered {
		t.Fatalf("scale 2 template was not registered: %+v", info)
	}
	if math.Abs(info.Fit.Tx-(-2)) > 1 {
		t.Fatalf("scale 2 Fit.Tx = %.3f, want about -2", info.Fit.Tx)
	}
	if math.Abs(info.Fit.Ty-1.5) > 1 {
		t.Fatalf("scale 2 Fit.Ty = %.3f, want about +1.5", info.Fit.Ty)
	}
}
