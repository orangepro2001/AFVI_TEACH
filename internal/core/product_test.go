package core

import (
	"math"
	"testing"
)

// productTestScene builds a synthetic scene: Texture all 1.0, left half MatSR, right half MatMetal.
func productTestScene(w, h int) *LightScene {
	material := make([]uint8, w*h)
	texture := make([]float32, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			if x < w/2 {
				material[idx] = MatSR
			} else {
				material[idx] = MatMetal
			}
			texture[idx] = 1
		}
	}
	return &LightScene{Width: w, Height: h, Material: material, Texture: texture}
}

func TestProductDeterministic(t *testing.T) {
	opts := DefaultProductOptions()
	a := productTestScene(256, 256)
	b := productTestScene(256, 256)
	ApplyProductVariation(a, opts)
	ApplyProductVariation(b, opts)
	for i := range a.Texture {
		if a.Texture[i] != b.Texture[i] {
			t.Fatalf("texture differs at %d: %v vs %v", i, a.Texture[i], b.Texture[i])
		}
	}
	other := opts
	other.Seed = 42
	c := productTestScene(256, 256)
	ApplyProductVariation(c, other)
	diff := 0
	for i := range a.Texture {
		if a.Texture[i] != c.Texture[i] {
			diff++
		}
	}
	if diff*10 < len(a.Texture) {
		t.Fatalf("different seed changed only %d/%d pixels", diff, len(a.Texture))
	}
}

func TestProductDisabledOrZeroStrength(t *testing.T) {
	opts := DefaultProductOptions()
	opts.Enabled = false
	a := productTestScene(64, 64)
	ApplyProductVariation(a, opts)
	for i, v := range a.Texture {
		if v != 1 {
			t.Fatalf("disabled changed texture at %d: %v", i, v)
		}
	}
	opts = DefaultProductOptions()
	opts.Strength = 0
	b := productTestScene(64, 64)
	ApplyProductVariation(b, opts)
	for i, v := range b.Texture {
		if v != 1 {
			t.Fatalf("zero strength changed texture at %d: %v", i, v)
		}
	}
	ApplyProductVariation(nil, DefaultProductOptions())
}

func TestProductStatistics(t *testing.T) {
	opts := DefaultProductOptions()
	scene := productTestScene(256, 256)
	ApplyProductVariation(scene, opts)
	w := scene.Width
	h := scene.Height
	var srSum, srSumSq, mtSum, mtSumSq float64
	var srN, mtN int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := float64(scene.Texture[y*w+x])
			if v < 0 || v > 4 {
				t.Fatalf("value out of range at %d,%d: %v", x, y, v)
			}
			if x < w/2 {
				srSum += v
				srSumSq += v * v
				srN++
			} else {
				mtSum += v
				mtSumSq += v * v
				mtN++
			}
		}
	}
	srMean := srSum / float64(srN)
	srStd := math.Sqrt(srSumSq/float64(srN) - srMean*srMean)
	mtMean := mtSum / float64(mtN)
	mtStd := math.Sqrt(mtSumSq/float64(mtN) - mtMean*mtMean)
	srExp := math.Sqrt(0.10*0.10 + 0.06*0.06)
	mtExp := math.Sqrt(0.04*0.04 + 0.03*0.03)
	if math.Abs(srMean-1) > 0.02 {
		t.Fatalf("SR mean %v not within 0.02 of 1", srMean)
	}
	if math.Abs(mtMean-1) > 0.02 {
		t.Fatalf("Metal mean %v not within 0.02 of 1", mtMean)
	}
	if math.Abs(srStd-srExp) > 0.35*srExp {
		t.Fatalf("SR std %v not within 35%% of %v", srStd, srExp)
	}
	if math.Abs(mtStd-mtExp) > 0.35*mtExp {
		t.Fatalf("Metal std %v not within 35%% of %v", mtStd, mtExp)
	}
}

func TestProductCorrelation(t *testing.T) {
	opts := DefaultProductOptions()
	opts.Materials[MatSR] = ProductMaterial{Sigma: 0.1, Length: 24, Grain: 0}
	scene := productTestScene(256, 256)
	ApplyProductVariation(scene, opts)
	w := scene.Width
	h := scene.Height
	half := w / 2
	var sum, sumSq, sumProd float64
	var n int
	for y := 0; y < h; y++ {
		for x := 0; x < half-1; x++ {
			a := float64(scene.Texture[y*w+x])
			b := float64(scene.Texture[y*w+x+1])
			sum += a
			sumSq += a * a
			sumProd += a * b
			n++
		}
	}
	mean := sum / float64(n)
	variance := sumSq/float64(n) - mean*mean
	cov := sumProd/float64(n) - mean*mean
	corr := cov / variance
	if corr <= 0.8 {
		t.Fatalf("correlated lag-1 correlation %v not > 0.8", corr)
	}

	opts2 := DefaultProductOptions()
	opts2.Materials[MatSR] = ProductMaterial{Sigma: 0, Length: 24, Grain: 0.1}
	scene2 := productTestScene(256, 256)
	ApplyProductVariation(scene2, opts2)
	sum, sumSq, sumProd, n = 0, 0, 0, 0
	for y := 0; y < h; y++ {
		for x := 0; x < half-1; x++ {
			a := float64(scene2.Texture[y*w+x])
			b := float64(scene2.Texture[y*w+x+1])
			sum += a
			sumSq += a * a
			sumProd += a * b
			n++
		}
	}
	mean = sum / float64(n)
	variance = sumSq/float64(n) - mean*mean
	cov = sumProd/float64(n) - mean*mean
	corr = cov / variance
	if corr >= 0.1 {
		t.Fatalf("grain lag-1 correlation %v not < 0.1", corr)
	}
}

func TestProductAppliedToRender(t *testing.T) {
	w, h := 128, 128
	data := make([]uint8, w*h)
	for i := range data {
		data[i] = 100
	}
	median := &GrayImage{Width: w, Height: h, Data: data}
	material := make([]uint8, w*h)
	for i := range material {
		material[i] = MatMetal
	}
	setting := DefaultLightSetting()

	scene, err := NewProductLightScene(median, material, DefaultProductOptions())
	if err != nil {
		t.Fatalf("NewProductLightScene: %v", err)
	}
	img := RenderChannel(scene, &setting, ChRed)
	var sum, sumSq float64
	for _, v := range img.Data {
		sum += float64(v)
		sumSq += float64(v) * float64(v)
	}
	n := float64(len(img.Data))
	mean := sum / n
	std := math.Sqrt(sumSq/n - mean*mean)
	if std <= 0.5 {
		t.Fatalf("rendered std %v not > 0.5", std)
	}

	off := DefaultProductOptions()
	off.Enabled = false
	scene2, err := NewProductLightScene(median, material, off)
	if err != nil {
		t.Fatalf("NewProductLightScene: %v", err)
	}
	img2 := RenderChannel(scene2, &setting, ChRed)
	first := img2.Data[0]
	for i, v := range img2.Data {
		if v != first {
			t.Fatalf("disabled render not constant at %d: %v vs %v", i, v, first)
		}
	}
}

func TestProductScaleFullSize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-size product test in short mode")
	}
	w, h := 5860, 4020
	material := make([]uint8, w*h)
	texture := make([]float32, w*h)
	for i := range texture {
		texture[i] = 1
	}
	scene := &LightScene{Width: w, Height: h, Material: material, Texture: texture}
	ApplyProductVariation(scene, DefaultProductOptions())
}
