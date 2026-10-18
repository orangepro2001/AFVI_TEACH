package inspect

import (
	"math"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

// allMaterials 是"所有材质都启用局部背景"的开关，供直接调用 computeBackground 的测试使用。
var allMaterials = [NumMaterials]bool{true, true, true, true}

// bruteResid 用朴素实现计算像素 (x,y) 的残差 d。
func bruteResid(img Image, reg Regions, excl []uint8, r, x, y int) int16 {
	w, h := img.Width, img.Height
	m := reg.Material[y*w+x]
	if excl != nil && excl[y*w+x] != 0 {
		return 0
	}
	var vals []int
	x0 := x - r
	if x0 < 0 {
		x0 = 0
	}
	x1 := x + r
	if x1 > w-1 {
		x1 = w - 1
	}
	y0 := y - r
	if y0 < 0 {
		y0 = 0
	}
	y1 := y + r
	if y1 > h-1 {
		y1 = h - 1
	}
	for yy := y0; yy <= y1; yy++ {
		for xx := x0; xx <= x1; xx++ {
			idx := yy*w + xx
			if reg.Material[idx] != m {
				continue
			}
			if excl != nil && excl[idx] != 0 {
				continue
			}
			vals = append(vals, int(img.Pix[idx]))
		}
	}
	if len(vals) == 0 {
		return 0
	}
	sort.Ints(vals)
	b := vals[len(vals)/2]
	return int16(int(img.Pix[y*w+x]) - b)
}

func TestBackgroundMatchesBruteForce(t *testing.T) {
	w, h := 41, 33
	rng := rand.New(rand.NewSource(7))
	img := Image{Width: w, Height: h, Pix: make([]uint8, w*h)}
	reg := Regions{Width: w, Height: h, Material: make([]uint8, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			img.Pix[idx] = uint8(rng.Intn(256))
			reg.Material[idx] = uint8((x/9 + y/7) % 4)
		}
	}

	for _, useExcl := range []bool{false, true} {
		var excl []uint8
		if useExcl {
			excl = edgeExclusion(reg, 2)
		}
		for _, r := range []int{1, 2, 3, 6} {
			lb := computeBackground(img, reg, excl, r, allMaterials)
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					idx := y*w + x
					want := bruteResid(img, reg, excl, r, x, y)
					if lb.resid[idx] != want {
						t.Fatalf("useExcl=%v r=%d 像素(%d,%d) 残差不一致：got=%d want=%d",
							useExcl, r, x, y, lb.resid[idx], want)
					}
				}
			}
			// 逐材质校验 anchor/mean/std。
			for m := 0; m < NumMaterials; m++ {
				var sumV, sumD, sumDD float64
				var cnt int
				for i := 0; i < w*h; i++ {
					if reg.Material[i] != uint8(m) {
						continue
					}
					if excl != nil && excl[i] != 0 {
						continue
					}
					cnt++
					v := float64(img.Pix[i])
					d := float64(lb.resid[i])
					sumV += v
					sumD += d
					sumDD += d * d
				}
				if cnt == 0 {
					continue
				}
				wantAnchor := (sumV - sumD) / float64(cnt)
				if math.Abs(lb.anchor[m]-wantAnchor) > 1e-9 {
					t.Fatalf("useExcl=%v r=%d 材质 %d anchor 不一致：got=%v want=%v",
						useExcl, r, m, lb.anchor[m], wantAnchor)
				}
				wantMean := sumV / float64(cnt)
				if math.Abs(lb.mean(m)-wantMean) > 1e-9 {
					t.Fatalf("useExcl=%v r=%d 材质 %d mean 不一致：got=%v want=%v",
						useExcl, r, m, lb.mean(m), wantMean)
				}
				meanD := sumD / float64(cnt)
				variance := sumDD/float64(cnt) - meanD*meanD
				if variance < 0 {
					variance = 0
				}
				wantStd := math.Sqrt(variance)
				if math.Abs(lb.std(m)-wantStd) > 1e-9 {
					t.Fatalf("useExcl=%v r=%d 材质 %d std 不一致：got=%v want=%v",
						useExcl, r, m, lb.std(m), wantStd)
				}
			}
		}
	}
}

func TestBackgroundRadiusZeroIsLegacy(t *testing.T) {
	w, h := 60, 40
	rng := rand.New(rand.NewSource(11))
	img := Image{Width: w, Height: h, Pix: make([]uint8, w*h)}
	reg := Regions{Width: w, Height: h, Material: make([]uint8, w*h)}
	for i := 0; i < w*h; i++ {
		img.Pix[i] = uint8(rng.Intn(256))
		reg.Material[i] = uint8(rng.Intn(NumMaterials))
	}

	p := DefaultParams()
	p.Materials[Metal].Enabled = true
	p.Materials[SR].Enabled = true
	p.Materials[Open].Enabled = true
	p.Materials[Metal].Low = 40
	p.Materials[Metal].High = 200
	p.Materials[SR].Low = 30
	p.Materials[SR].High = 210
	p.Materials[Open].Low = 20
	p.Materials[Open].High = 220
	p.BackgroundRadius = 0
	p.EdgeMargin = 0

	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if !sameDefectMultiset(naiveInspect(img, reg, p), rep.Defects) {
		t.Fatalf("BackgroundRadius=0 与朴素实现不一致")
	}
	rep2, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep, rep2) {
		t.Fatalf("两次 Inspect 结果不一致")
	}
}

func TestBackgroundMaterialsDoNotMix(t *testing.T) {
	w, h := 40, 20
	img, reg := buildImage(w, h, func(x, y int) (uint8, Material) {
		if x < 20 {
			return 200, Metal
		}
		return 20, SR
	})
	lb := computeBackground(img, reg, nil, 3, allMaterials)
	for i := range lb.resid {
		if lb.resid[i] != 0 {
			t.Fatalf("像素 %d 残差应为 0，得到 %d", i, lb.resid[i])
		}
	}
	if lb.anchor[Metal] != 200 {
		t.Fatalf("Metal anchor 期望 200，得到 %v", lb.anchor[Metal])
	}
	if lb.anchor[SR] != 20 {
		t.Fatalf("SR anchor 期望 20，得到 %v", lb.anchor[SR])
	}
}

func TestBackgroundSmoothRamp(t *testing.T) {
	w, h := 120, 60
	ramp, reg := buildImage(w, h, func(x, y int) (uint8, Material) {
		return uint8(40 + x), Metal
	})
	specks := [][2]int{{20, 10}, {45, 40}, {70, 20}, {95, 45}, {105, 15}}
	testImg, _ := buildImage(w, h, func(x, y int) (uint8, Material) {
		v := 40 + x
		for _, s := range specks {
			if x >= s[0] && x < s[0]+3 && y >= s[1] && y < s[1]+3 {
				v += 70
			}
		}
		if v > 255 {
			v = 255
		}
		return uint8(v), Metal
	})

	base := DefaultParams()
	base.Materials[Metal].Enabled = true
	base.Materials[SR].Enabled = false
	base.Materials[Metal].Size = 3
	base.Materials[Metal].MinArea = 4
	base.EdgeMargin = 0

	// a) r = 0：学习后对测试图无缺陷（盲区）。
	p0 := base
	p0.BackgroundRadius = 0
	learned0, err := Learn(ramp, reg, 5, p0)
	if err != nil {
		t.Fatal(err)
	}
	rep0, err := Inspect(testImg, reg, learned0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep0.Defects) != 0 {
		t.Fatalf("r=0 期望 0 个缺陷，得到 %d", len(rep0.Defects))
	}

	// b) r = 3：学习后能检出所有斑点。
	p3 := base
	p3.BackgroundRadius = 3
	learned3, err := Learn(ramp, reg, 5, p3)
	if err != nil {
		t.Fatal(err)
	}
	rep3, err := Inspect(testImg, reg, learned3)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep3.Defects) < 5 {
		t.Fatalf("r=3 期望至少 5 个缺陷，得到 %d", len(rep3.Defects))
	}
	for _, s := range specks {
		cx := s[0] + 1
		cy := s[1] + 1
		found := false
		for _, d := range rep3.Defects {
			if cx >= d.BBox.X && cx < d.BBox.X+d.BBox.W && cy >= d.BBox.Y && cy < d.BBox.Y+d.BBox.H {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("斑点中心 (%d,%d) 未被任何缺陷 BBox 覆盖", cx, cy)
		}
	}
	repClean, err := Inspect(ramp, reg, learned3)
	if err != nil {
		t.Fatal(err)
	}
	if len(repClean.Defects) > 2 {
		t.Fatalf("r=3 干净斜坡期望至多 2 个缺陷，得到 %d", len(repClean.Defects))
	}
}

// stripeImage 构建双峰条纹图：宽 20 的竖条纹，偶数条 40、奇数条 170，加 sigma=5 噪声。
func stripeImage(t *testing.T) (Image, Regions) {
	t.Helper()
	w, h := 160, 80
	rng := rand.New(rand.NewSource(3))
	img := Image{Width: w, Height: h, Pix: make([]uint8, w*h)}
	reg := Regions{Width: w, Height: h, Material: make([]uint8, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			base := 40.0
			if (x/20)%2 == 1 {
				base = 170.0
			}
			v := base + rng.NormFloat64()*5
			if v < 0 {
				v = 0
			}
			if v > 255 {
				v = 255
			}
			img.Pix[idx] = uint8(math.Round(v))
			reg.Material[idx] = uint8(Metal)
		}
	}
	return img, reg
}

func TestBackgroundBimodalPlateauDots(t *testing.T) {
	img, reg := stripeImage(t)

	base := DefaultParams()
	base.Materials[Metal].Enabled = true
	base.Materials[SR].Enabled = false
	base.Materials[Metal].Size = 1
	base.Materials[Metal].MinArea = 1
	base.EdgeMargin = 0

	// a) r = 0：整条亮条纹被标记。
	p0 := base
	p0.BackgroundRadius = 0
	p0.Materials[Metal].Low = 0
	p0.Materials[Metal].High = 148
	rep0, err := Inspect(img, reg, p0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep0.Defects) > 6 {
		t.Fatalf("r=0 期望至多 6 个缺陷，得到 %d", len(rep0.Defects))
	}
	maxArea := 0
	for _, d := range rep0.Defects {
		if d.Area > maxArea {
			maxArea = d.Area
		}
	}
	if maxArea < 1000 {
		t.Fatalf("r=0 期望最大 Area >= 1000，得到 %d", maxArea)
	}

	// b) r = 3：散点。
	p3 := base
	p3.BackgroundRadius = 3
	repStats, err := Inspect(img, reg, p3)
	if err != nil {
		t.Fatal(err)
	}
	high := math.Round(repStats.PerMaterial[Metal].Mean + 10)
	p3.Materials[Metal].Low = 0
	p3.Materials[Metal].High = high
	rep3, err := Inspect(img, reg, p3)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep3.Defects) < 100 {
		t.Fatalf("r=3 期望至少 100 个缺陷，得到 %d", len(rep3.Defects))
	}
	maxArea3 := 0
	for _, d := range rep3.Defects {
		if d.Area > maxArea3 {
			maxArea3 = d.Area
		}
	}
	if maxArea3 > 12 {
		t.Fatalf("r=3 期望最大 Area <= 12，得到 %d", maxArea3)
	}
}

func TestBackgroundLearnIsTight(t *testing.T) {
	img, reg := stripeImage(t)

	base := DefaultParams()
	base.Materials[Metal].Enabled = true
	base.Materials[SR].Enabled = false
	base.EdgeMargin = 0

	p3 := base
	p3.BackgroundRadius = 3
	learned3, err := Learn(img, reg, 5, p3)
	if err != nil {
		t.Fatal(err)
	}
	if learned3.Materials[Metal].High-learned3.Materials[Metal].Low >= 70 {
		t.Fatalf("r=3 期望 High-Low < 70，得到 %v", learned3.Materials[Metal].High-learned3.Materials[Metal].Low)
	}

	p0 := base
	p0.BackgroundRadius = 0
	learned0, err := Learn(img, reg, 5, p0)
	if err != nil {
		t.Fatal(err)
	}
	if learned0.Materials[Metal].High-learned0.Materials[Metal].Low < 200 {
		t.Fatalf("r=0 期望 High-Low >= 200，得到 %v", learned0.Materials[Metal].High-learned0.Materials[Metal].Low)
	}
}

func TestBackgroundProbeRect(t *testing.T) {
	img, reg := stripeImage(t)

	base := DefaultParams()
	base.Materials[Metal].Enabled = true
	base.Materials[SR].Enabled = false
	base.Materials[Metal].Size = 1
	base.Materials[Metal].MinArea = 1
	base.EdgeMargin = 0

	rect := Rect{X: 20, Y: 10, W: 20, H: 60}

	// r = 0：整条亮条纹被标记。
	p0 := base
	p0.BackgroundRadius = 0
	p0.Materials[Metal].Low = 0
	p0.Materials[Metal].High = 148
	rep0, err := Inspect(img, reg, p0)
	if err != nil {
		t.Fatal(err)
	}
	pr0, err := ProbeRect(img, reg, p0, rep0, rect)
	if err != nil {
		t.Fatal(err)
	}
	if pr0.PerMaterial[Metal].Bright < 1100 {
		t.Fatalf("r=0 期望 Metal Bright >= 1100，得到 %d", pr0.PerMaterial[Metal].Bright)
	}

	// r = 3：散点。
	p3 := base
	p3.BackgroundRadius = 3
	repStats, err := Inspect(img, reg, p3)
	if err != nil {
		t.Fatal(err)
	}
	high := math.Round(repStats.PerMaterial[Metal].Mean + 10)
	p3.Materials[Metal].Low = 0
	p3.Materials[Metal].High = high
	rep3, err := Inspect(img, reg, p3)
	if err != nil {
		t.Fatal(err)
	}
	pr3, err := ProbeRect(img, reg, p3, rep3, rect)
	if err != nil {
		t.Fatal(err)
	}
	if pr3.PerMaterial[Metal].Bright > 120 {
		t.Fatalf("r=3 期望 Metal Bright <= 120，得到 %d", pr3.PerMaterial[Metal].Bright)
	}
}

func TestBackgroundValidation(t *testing.T) {
	img, reg := uniform(8, 8, 100, Metal)

	p := DefaultParams()
	p.BackgroundRadius = -1
	if err := p.Validate(); err == nil {
		t.Fatalf("BackgroundRadius=-1 期望 Validate 错误")
	}
	if _, err := Inspect(img, reg, p); err == nil {
		t.Fatalf("BackgroundRadius=-1 期望 Inspect 错误")
	}
	if _, err := Learn(img, reg, 3, p); err == nil {
		t.Fatalf("BackgroundRadius=-1 期望 Learn 错误")
	}

	p.BackgroundRadius = MaxBackgroundRadius + 1
	if err := p.Validate(); err == nil {
		t.Fatalf("BackgroundRadius=MaxBackgroundRadius+1 期望 Validate 错误")
	}
	if _, err := Inspect(img, reg, p); err == nil {
		t.Fatalf("BackgroundRadius=MaxBackgroundRadius+1 期望 Inspect 错误")
	}
	if _, err := Learn(img, reg, 3, p); err == nil {
		t.Fatalf("BackgroundRadius=MaxBackgroundRadius+1 期望 Learn 错误")
	}

	p.BackgroundRadius = 0
	if err := p.Validate(); err != nil {
		t.Fatalf("BackgroundRadius=0 期望通过，得到 %v", err)
	}
	if _, err := Inspect(img, reg, p); err != nil {
		t.Fatalf("BackgroundRadius=0 期望 Inspect 通过，得到 %v", err)
	}

	p.BackgroundRadius = MaxBackgroundRadius
	if err := p.Validate(); err != nil {
		t.Fatalf("BackgroundRadius=MaxBackgroundRadius 期望通过，得到 %v", err)
	}
	if _, err := Inspect(img, reg, p); err != nil {
		t.Fatalf("BackgroundRadius=MaxBackgroundRadius 期望 Inspect 通过，得到 %v", err)
	}
}

func TestBackgroundPerMaterialSwitch(t *testing.T) {
	w, h := 80, 40
	rngMetal := rand.New(rand.NewSource(3))
	rngSR := rand.New(rand.NewSource(4))
	img, reg := buildImage(w, h, func(x, y int) (uint8, Material) {
		if x < 40 {
			base := 40.0
			if (x/10)%2 == 1 {
				base = 170.0
			}
			v := base + rngMetal.NormFloat64()*5
			if v < 0 {
				v = 0
			}
			if v > 255 {
				v = 255
			}
			return uint8(math.Round(v)), Metal
		}
		v := 20.0 + rngSR.NormFloat64()*1.5
		if x >= 60 && x < 65 && y >= 15 && y < 20 {
			v += 60
		}
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		return uint8(math.Round(v)), SR
	})

	base := DefaultParams()
	base.Materials[Metal].Enabled = true
	base.Materials[SR].Enabled = true
	base.Materials[Open].Enabled = false
	base.Materials[Base].Enabled = false
	base.Materials[Metal].Size = 1
	base.Materials[Metal].MinArea = 1
	base.Materials[SR].Size = 1
	base.Materials[SR].MinArea = 1
	base.EdgeMargin = 0

	// a) 混合：金属用局部背景，SR 用全局区间。
	pRaw := base
	pRaw.BackgroundRadius = 0
	learnedRaw, err := Learn(img, reg, 5, pRaw)
	if err != nil {
		t.Fatal(err)
	}
	repRaw, err := Inspect(img, reg, learnedRaw)
	if err != nil {
		t.Fatal(err)
	}

	pMixed := base
	pMixed.BackgroundRadius = 2
	pMixed.Materials[Metal].LocalBackground = true
	pMixed.Materials[SR].LocalBackground = false
	learnedMixed, err := Learn(img, reg, 5, pMixed)
	if err != nil {
		t.Fatal(err)
	}
	repMixed, err := Inspect(img, reg, learnedMixed)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(repRaw.PerMaterial[SR], repMixed.PerMaterial[SR]) {
		t.Fatalf("SR 统计在混合模式下应保持不变：raw=%+v mixed=%+v", repRaw.PerMaterial[SR], repMixed.PerMaterial[SR])
	}
	if repMixed.PerMaterial[Metal].Std*2 >= repRaw.PerMaterial[Metal].Std {
		t.Fatalf("混合模式下金属 Std 应小于全局的一半：mixed=%v raw=%v",
			repMixed.PerMaterial[Metal].Std, repRaw.PerMaterial[Metal].Std)
	}
	found := false
	for _, d := range repMixed.Defects {
		if d.Material != SR {
			continue
		}
		if 62 >= d.BBox.X && 62 < d.BBox.X+d.BBox.W && 17 >= d.BBox.Y && 17 < d.BBox.Y+d.BBox.H {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("混合模式下 SR 的全局区间应检出 (62,17) 处的方块")
	}

	// b) 全部启用局部背景：SR 的硬边方块被中位数吸收。
	pAll := base
	pAll.BackgroundRadius = 2
	pAll.Materials[Metal].LocalBackground = true
	pAll.Materials[SR].LocalBackground = true
	learnedAll, err := Learn(img, reg, 5, pAll)
	if err != nil {
		t.Fatal(err)
	}
	repAll, err := Inspect(img, reg, learnedAll)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range repAll.Defects {
		if d.Material != SR {
			continue
		}
		if 62 >= d.BBox.X && 62 < d.BBox.X+d.BBox.W && 17 >= d.BBox.Y && 17 < d.BBox.Y+d.BBox.H {
			t.Fatalf("SR 启用局部背景后不应检出 (62,17) 处的方块：%+v", d.BBox)
		}
	}
}

func TestBackgroundDeterministic(t *testing.T) {
	img, reg := stripeImage(t)

	p := DefaultParams()
	p.Materials[Metal].Enabled = true
	p.Materials[SR].Enabled = false
	p.Materials[Metal].Size = 1
	p.Materials[Metal].MinArea = 1
	p.Materials[Metal].Low = 0
	p.Materials[Metal].High = 200
	p.EdgeMargin = 0
	p.BackgroundRadius = 3

	rep1, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	rep2, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep1, rep2) {
		t.Fatalf("两次 Inspect 结果不一致")
	}
}
