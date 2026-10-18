package inspect

import (
	"math"
	"math/rand"
	"testing"
)

// buildImage 根据闭包构建图像与材质图。
func buildImage(w, h int, fn func(x, y int) (uint8, Material)) (Image, Regions) {
	img := Image{Width: w, Height: h, Pix: make([]uint8, w*h)}
	reg := Regions{Width: w, Height: h, Material: make([]uint8, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v, m := fn(x, y)
			idx := y*w + x
			img.Pix[idx] = v
			reg.Material[idx] = uint8(m)
		}
	}
	return img, reg
}

// uniform 构建全图同材质同亮度的图像。
func uniform(w, h int, v uint8, m Material) (Image, Regions) {
	return buildImage(w, h, func(x, y int) (uint8, Material) { return v, m })
}

func TestIntervalOverkill(t *testing.T) {
	w, h := 64, 48
	img, reg := uniform(w, h, 100, Metal)

	p := DefaultParams()
	p.Materials[Metal].Low = 90
	p.Materials[Metal].High = 110
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("期望 0 个缺陷，得到 %d", len(rep.Defects))
	}

	p.Materials[Metal].Low = 105
	p.Materials[Metal].High = 110
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	d := rep.Defects[0]
	if d.Polarity != Dark {
		t.Fatalf("期望 Dark，得到 %v", d.Polarity)
	}
	if d.Area != w*h {
		t.Fatalf("期望 Area=%d，得到 %d", w*h, d.Area)
	}

	p.Materials[Metal].Low = 0
	p.Materials[Metal].High = 95
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 || rep.Defects[0].Polarity != Bright {
		t.Fatalf("期望 1 个 Bright 缺陷，得到 %+v", rep.Defects)
	}
}

func TestMetalAndSRSeparate(t *testing.T) {
	w, h := 64, 48
	img, reg := buildImage(w, h, func(x, y int) (uint8, Material) {
		if x < w/2 {
			return 100, Metal
		}
		return 100, SR
	})

	p := DefaultParams()
	p.Materials[Metal].Low = 90
	p.Materials[Metal].High = 110
	p.Materials[SR].Low = 50
	p.Materials[SR].High = 80
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	if rep.Defects[0].Material != SR || rep.Defects[0].Polarity != Bright {
		t.Fatalf("期望 SR/Bright，得到 %v/%v", rep.Defects[0].Material, rep.Defects[0].Polarity)
	}

	// 交换区间。
	p.Materials[Metal].Low = 50
	p.Materials[Metal].High = 80
	p.Materials[SR].Low = 90
	p.Materials[SR].High = 110
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	if rep.Defects[0].Material != Metal || rep.Defects[0].Polarity != Bright {
		t.Fatalf("期望 Metal/Bright，得到 %v/%v", rep.Defects[0].Material, rep.Defects[0].Polarity)
	}

	// 关闭 SR，则 SR 不再被标记。
	p.Materials[SR].Enabled = false
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 || rep.Defects[0].Material != Metal {
		t.Fatalf("期望仅 Metal 缺陷，得到 %+v", rep.Defects)
	}
}

func TestTooWideRangeMisses(t *testing.T) {
	w, h := 64, 48
	img, reg := buildImage(w, h, func(x, y int) (uint8, Material) {
		if x >= 10 && x < 20 && y >= 10 && y < 20 {
			return 150, Metal
		}
		return 100, Metal
	})

	p := DefaultParams()
	p.Materials[Metal].Low = 50
	p.Materials[Metal].High = 200
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("期望 0 个缺陷，得到 %d", len(rep.Defects))
	}

	p.Materials[Metal].Low = 50
	p.Materials[Metal].High = 120
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	d := rep.Defects[0]
	if d.Area != 100 {
		t.Fatalf("期望 Area=100，得到 %d", d.Area)
	}
	if d.PeakGV != 150 {
		t.Fatalf("期望 PeakGV=150，得到 %d", d.PeakGV)
	}
	if math.Abs(d.Excess-30) > 1e-9 {
		t.Fatalf("期望 Excess=30，得到 %v", d.Excess)
	}
}

func TestSizeMergesSplits(t *testing.T) {
	w, h := 64, 48
	// 五个 2x2 亮块，间距 4（块间空隙 2）。
	patches := [][2]int{{5, 5}, {11, 5}, {17, 5}, {23, 5}, {29, 5}}
	img, reg := buildImage(w, h, func(x, y int) (uint8, Material) {
		for _, p := range patches {
			if x >= p[0] && x < p[0]+2 && y >= p[1] && y < p[1]+2 {
				return 200, Metal
			}
		}
		return 100, Metal
	})

	p := DefaultParams()
	p.Materials[Metal].Low = 0
	p.Materials[Metal].High = 150
	p.Materials[Metal].Size = 1
	p.Materials[Metal].MinArea = 1
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 5 {
		t.Fatalf("Size=1 期望 5 个缺陷，得到 %d", len(rep.Defects))
	}
	for _, d := range rep.Defects {
		if d.Fragments != 1 {
			t.Fatalf("期望 Fragments=1，得到 %d", d.Fragments)
		}
	}

	p.Materials[Metal].Size = 5
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("Size=5 期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	d := rep.Defects[0]
	if d.Fragments != 5 {
		t.Fatalf("期望 Fragments=5，得到 %d", d.Fragments)
	}
	if d.Area != 20 {
		t.Fatalf("期望 Area=20，得到 %d", d.Area)
	}
	if d.BBox.X != 5 || d.BBox.W != 26 {
		t.Fatalf("期望 BBox.X=5 W=26，得到 X=%d W=%d", d.BBox.X, d.BBox.W)
	}

	// 间距 10 时仍为 5 个。
	patches2 := [][2]int{{5, 5}, {17, 5}, {29, 5}, {41, 5}, {53, 5}}
	img2, reg2 := buildImage(w, h, func(x, y int) (uint8, Material) {
		for _, p := range patches2 {
			if x >= p[0] && x < p[0]+2 && y >= p[1] && y < p[1]+2 {
				return 200, Metal
			}
		}
		return 100, Metal
	})
	rep, err = Inspect(img2, reg2, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 5 {
		t.Fatalf("间距 10 期望 5 个缺陷，得到 %d", len(rep.Defects))
	}

	// 对角相邻：Size=1 时仍为一个缺陷。
	img3, reg3 := buildImage(w, h, func(x, y int) (uint8, Material) {
		if (x == 5 && y == 5) || (x == 6 && y == 6) {
			return 200, Metal
		}
		return 100, Metal
	})
	p.Materials[Metal].Size = 1
	rep, err = Inspect(img3, reg3, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("对角相邻期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	if rep.Defects[0].Fragments != 1 {
		t.Fatalf("期望 Fragments=1，得到 %d", rep.Defects[0].Fragments)
	}
}

func TestMinArea(t *testing.T) {
	w, h := 64, 48
	// 面积 1、3、8 的块。
	img, reg := buildImage(w, h, func(x, y int) (uint8, Material) {
		// 面积 1
		if x == 5 && y == 5 {
			return 200, Metal
		}
		// 面积 3
		if y == 10 && x >= 5 && x < 8 {
			return 200, Metal
		}
		// 面积 8
		if y >= 15 && y < 19 && x >= 5 && x < 7 {
			return 200, Metal
		}
		return 100, Metal
	})

	p := DefaultParams()
	p.Materials[Metal].Low = 0
	p.Materials[Metal].High = 150
	p.Materials[Metal].Size = 1
	p.Materials[Metal].MinArea = 4
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	if rep.Filtered != 2 {
		t.Fatalf("期望 Filtered=2，得到 %d", rep.Filtered)
	}

	p.Materials[Metal].MinArea = 1
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 3 {
		t.Fatalf("期望 3 个缺陷，得到 %d", len(rep.Defects))
	}
	if rep.Filtered != 0 {
		t.Fatalf("期望 Filtered=0，得到 %d", rep.Filtered)
	}
}

func TestPolarityAndMaterialSeparation(t *testing.T) {
	w, h := 64, 48
	img, reg := buildImage(w, h, func(x, y int) (uint8, Material) {
		if x == 5 && y == 5 {
			return 200, Metal // 亮
		}
		if x == 6 && y == 5 {
			return 50, Metal // 暗
		}
		if x == 7 && y == 5 {
			return 200, SR // 亮 SR
		}
		return 100, Metal
	})

	p := DefaultParams()
	p.Materials[Metal].Low = 90
	p.Materials[Metal].High = 110
	p.Materials[SR].Low = 90
	p.Materials[SR].High = 110
	p.Materials[Metal].Size = 9
	p.Materials[SR].Size = 9
	p.Materials[Metal].MinArea = 1
	p.Materials[SR].MinArea = 1
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	// 亮 Metal、暗 Metal、亮 SR 各一个。
	if len(rep.Defects) != 3 {
		t.Fatalf("期望 3 个缺陷，得到 %d: %+v", len(rep.Defects), rep.Defects)
	}
}

func TestStatistics(t *testing.T) {
	w, h := 4, 2
	img, reg := buildImage(w, h, func(x, y int) (uint8, Material) {
		if x < 2 {
			return uint8(10 + x + y*2), Metal
		}
		return uint8(100 + x), SR
	})

	p := DefaultParams()
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	ms := rep.PerMaterial[Metal]
	if ms.Pixels != 4 {
		t.Fatalf("Metal Pixels 期望 4，得到 %d", ms.Pixels)
	}
	// 值：10,11,12,13
	if ms.Min != 10 || ms.Max != 13 {
		t.Fatalf("Metal Min/Max 期望 10/13，得到 %d/%d", ms.Min, ms.Max)
	}
	if math.Abs(ms.Mean-11.5) > 1e-9 {
		t.Fatalf("Metal Mean 期望 11.5，得到 %v", ms.Mean)
	}
	// 总体标准差：sqrt(((10-11.5)^2+...)/4) = sqrt(1.25)
	if math.Abs(ms.Std-math.Sqrt(1.25)) > 1e-9 {
		t.Fatalf("Metal Std 期望 %v，得到 %v", math.Sqrt(1.25), ms.Std)
	}
	// 关闭的材质仍有统计。
	bs := rep.PerMaterial[Base]
	if bs.Pixels != 0 {
		t.Fatalf("Base Pixels 期望 0，得到 %d", bs.Pixels)
	}

	// SR 统计：值 102,103（两行各两个）。
	srs := rep.PerMaterial[SR]
	if srs.Pixels != 4 {
		t.Fatalf("SR Pixels 期望 4，得到 %d", srs.Pixels)
	}
	if srs.Min != 102 || srs.Max != 103 {
		t.Fatalf("SR Min/Max 期望 102/103，得到 %d/%d", srs.Min, srs.Max)
	}
	if math.Abs(srs.Mean-102.5) > 1e-9 {
		t.Fatalf("SR Mean 期望 102.5，得到 %v", srs.Mean)
	}
	if math.Abs(srs.Std-0.5) > 1e-9 {
		t.Fatalf("SR Std 期望 0.5，得到 %v", srs.Std)
	}

	// 默认参数下无标记。
	if ms.Bright != 0 || ms.Dark != 0 {
		t.Fatalf("默认参数下 Metal Bright/Dark 期望 0/0，得到 %d/%d", ms.Bright, ms.Dark)
	}

	// Metal Low=12：值 10、11 为暗。
	p2 := DefaultParams()
	p2.Materials[Metal].Low = 12
	p2.Materials[Metal].High = 255
	rep2, err := Inspect(img, reg, p2)
	if err != nil {
		t.Fatal(err)
	}
	if rep2.PerMaterial[Metal].Dark != 2 || rep2.PerMaterial[Metal].Bright != 0 {
		t.Fatalf("期望 Dark=2 Bright=0，得到 %d/%d", rep2.PerMaterial[Metal].Dark, rep2.PerMaterial[Metal].Bright)
	}

	// Metal High=11：值 12、13 为亮。
	p3 := DefaultParams()
	p3.Materials[Metal].Low = 0
	p3.Materials[Metal].High = 11
	rep3, err := Inspect(img, reg, p3)
	if err != nil {
		t.Fatal(err)
	}
	if rep3.PerMaterial[Metal].Bright != 2 || rep3.PerMaterial[Metal].Dark != 0 {
		t.Fatalf("期望 Bright=2 Dark=0，得到 %d/%d", rep3.PerMaterial[Metal].Bright, rep3.PerMaterial[Metal].Dark)
	}
}

func TestGrowMatchesNaiveDilation(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for iter := 0; iter < 40; iter++ {
		w := 1 + rng.Intn(30)
		h := 1 + rng.Intn(30)
		density := 0.02 + rng.Float64()*0.08
		mask := make([]uint8, w*h)
		for i := range mask {
			if rng.Float64() < density {
				mask[i] = 1
			}
		}
		r := rng.Intn(6)
		orig := make([]uint8, len(mask))
		copy(orig, mask)

		got := grow(mask, w, h, r)
		want := naiveDilate(mask, w, h, r)
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("iter %d w=%d h=%d r=%d 像素 %d 不一致：got=%d want=%d", iter, w, h, r, i, got[i], want[i])
			}
		}
		for i := range mask {
			if mask[i] != orig[i] {
				t.Fatalf("grow 修改了输入掩码")
			}
		}
	}
}

// naiveDilate 用 Chebyshev 距离做朴素膨胀。
func naiveDilate(mask []uint8, w, h, r int) []uint8 {
	out := make([]uint8, w*h)
	if r <= 0 {
		copy(out, mask)
		return out
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if mask[y*w+x] == 0 {
				continue
			}
			for dy := -r; dy <= r; dy++ {
				for dx := -r; dx <= r; dx++ {
					ny, nx := y+dy, x+dx
					if ny < 0 || ny >= h || nx < 0 || nx >= w {
						continue
					}
					out[ny*w+nx] = 1
				}
			}
		}
	}
	return out
}

func TestLabelRunsEightConnectivity(t *testing.T) {
	// 手工构造掩码：
	// 行 0：x=0 与 x=2 两个行程（对角相邻于行 1 的 x=1）。
	// 行 1：x=1 一个行程，与行 0 的两个行程都对角连通。
	// 行 2：x=5 一个行程，与上面隔开一列，独立分量。
	// 行 3：x=4 与 x=6 两个行程。
	// 行 4：x=4..6 一个行程，把行 3 的两个行程连成 U 形。
	w, h := 8, 5
	mask := make([]uint8, w*h)
	set := func(x, y int) { mask[y*w+x] = 1 }
	set(0, 0)
	set(2, 0)
	set(1, 1)
	set(5, 2)
	set(4, 3)
	set(6, 3)
	set(4, 4)
	set(5, 4)
	set(6, 4)

	runs, rowStart := extractRuns(mask, w, h)
	id, count := labelRuns(runs, rowStart)
	if count != 2 {
		t.Fatalf("期望 2 个分量，得到 %d", count)
	}
	// 紧凑 id 按首次出现顺序：行 0 的行程为 0，行 2 的行程为 1。
	if id[0] != 0 {
		t.Fatalf("期望首个行程 id=0，得到 %d", id[0])
	}
	// 行 0 的两个行程与行 1 的行程同属分量 0。
	if id[1] != 0 || id[2] != 0 {
		t.Fatalf("期望行 0/1 的行程同属分量 0，得到 %d/%d", id[1], id[2])
	}
	// 行 2 的行程为分量 1。
	if id[3] != 1 {
		t.Fatalf("期望行 2 的行程 id=1，得到 %d", id[3])
	}
	// 行 3、4 的行程同属分量 1（U 形合并）。
	for i := 4; i < len(id); i++ {
		if id[i] != 1 {
			t.Fatalf("期望 U 形行程 id=1，得到 %d", id[i])
		}
	}
	// id 必须紧凑覆盖 0..count-1。
	seen := make([]bool, count)
	for _, v := range id {
		if v < 0 || int(v) >= count {
			t.Fatalf("id %d 越界", v)
		}
		seen[v] = true
	}
	for i, s := range seen {
		if !s {
			t.Fatalf("id %d 未出现", i)
		}
	}
}

func TestBruteForceCrossCheck(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	for iter := 0; iter < 30; iter++ {
		w, h := 40, 30
		img := Image{Width: w, Height: h, Pix: make([]uint8, w*h)}
		reg := Regions{Width: w, Height: h, Material: make([]uint8, w*h)}
		for i := range img.Pix {
			img.Pix[i] = 100
		}
		// 混合 Metal/SR 区域。
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if x < w/2 {
					reg.Material[y*w+x] = uint8(Metal)
				} else {
					reg.Material[y*w+x] = uint8(SR)
				}
			}
		}
		// 稀疏随机亮点。
		density := 0.03 + rng.Float64()*0.05
		for i := range img.Pix {
			if rng.Float64() < density {
				img.Pix[i] = 200
			}
		}
		// 稀疏随机暗点。
		darkDensity := 0.02 + rng.Float64()*0.03
		for i := range img.Pix {
			if rng.Float64() < darkDensity {
				img.Pix[i] = 20
			}
		}
		size := 1 + rng.Intn(9)
		minArea := 1 + rng.Intn(3)
		p := DefaultParams()
		p.Materials[Metal].Low = 60
		p.Materials[Metal].High = 150
		p.Materials[Metal].Size = size
		p.Materials[Metal].MinArea = minArea
		p.Materials[SR].Low = 60
		p.Materials[SR].High = 150
		p.Materials[SR].Size = size
		p.Materials[SR].MinArea = minArea

		rep, err := Inspect(img, reg, p)
		if err != nil {
			t.Fatal(err)
		}
		naive := naiveInspect(img, reg, p)
		if !sameDefectMultiset(rep.Defects, naive) {
			t.Fatalf("iter %d size %d 不一致：\nInspect=%+v\nnaive=%+v", iter, size, rep.Defects, naive)
		}
	}
}

// naiveInspect 是朴素参考实现。
func naiveInspect(img Image, reg Regions, p Params) []Defect {
	w, h := img.Width, img.Height
	var out []Defect
	for mi := 0; mi < NumMaterials; mi++ {
		mp := p.Materials[mi]
		if !mp.Enabled {
			continue
		}
		m := Material(mi)
		for _, pol := range []Polarity{Bright, Dark} {
			mask := make([]uint8, w*h)
			for i := range mask {
				if reg.Material[i] != uint8(m) {
					continue
				}
				v := float64(img.Pix[i])
				if pol == Bright && v > mp.High {
					mask[i] = 1
				} else if pol == Dark && v < mp.Low {
					mask[i] = 1
				}
			}
			r := mp.Size / 2
			grown := make([]uint8, w*h)
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					if mask[y*w+x] == 0 {
						continue
					}
					for dy := -r; dy <= r; dy++ {
						for dx := -r; dx <= r; dx++ {
							ny, nx := y+dy, x+dx
							if ny < 0 || ny >= h || nx < 0 || nx >= w {
								continue
							}
							grown[ny*w+nx] = 1
						}
					}
				}
			}
			// 朴素 8 连通洪泛。
			label := make([]int, w*h)
			for i := range label {
				label[i] = -1
			}
			comp := 0
			for i := range grown {
				if grown[i] == 0 || label[i] >= 0 {
					continue
				}
				// BFS
				queue := []int{i}
				label[i] = comp
				for len(queue) > 0 {
					cur := queue[0]
					queue = queue[1:]
					cy, cx := cur/w, cur%w
					for dy := -1; dy <= 1; dy++ {
						for dx := -1; dx <= 1; dx++ {
							if dy == 0 && dx == 0 {
								continue
							}
							ny, nx := cy+dy, cx+dx
							if ny < 0 || ny >= h || nx < 0 || nx >= w {
								continue
							}
							ni := ny*w + nx
							if grown[ni] != 0 && label[ni] < 0 {
								label[ni] = comp
								queue = append(queue, ni)
							}
						}
					}
				}
				comp++
			}
			// 原始掩码的 8 连通分量标记，用于统计 Fragments。
			oLabel := make([]int, w*h)
			for i := range oLabel {
				oLabel[i] = -1
			}
			oComp := 0
			for i := range mask {
				if mask[i] == 0 || oLabel[i] >= 0 {
					continue
				}
				queue := []int{i}
				oLabel[i] = oComp
				for len(queue) > 0 {
					cur := queue[0]
					queue = queue[1:]
					cy, cx := cur/w, cur%w
					for dy := -1; dy <= 1; dy++ {
						for dx := -1; dx <= 1; dx++ {
							if dy == 0 && dx == 0 {
								continue
							}
							ny, nx := cy+dy, cx+dx
							if ny < 0 || ny >= h || nx < 0 || nx >= w {
								continue
							}
							ni := ny*w + nx
							if mask[ni] != 0 && oLabel[ni] < 0 {
								oLabel[ni] = oComp
								queue = append(queue, ni)
							}
						}
					}
				}
				oComp++
			}
			// 按分量聚合原始被标记像素。
			type acc struct {
				area                   int
				minX, minY, maxX, maxY int
				fragments              int
			}
			accs := make([]acc, comp)
			for i := range accs {
				accs[i].minX = 1 << 30
				accs[i].minY = 1 << 30
				accs[i].maxX = -1
				accs[i].maxY = -1
			}
			seenFrag := make([]bool, oComp)
			for i := range mask {
				if mask[i] == 0 {
					continue
				}
				c := label[i]
				if c < 0 {
					continue
				}
				x, y := i%w, i/w
				a := &accs[c]
				a.area++
				if x < a.minX {
					a.minX = x
				}
				if x > a.maxX {
					a.maxX = x
				}
				if y < a.minY {
					a.minY = y
				}
				if y > a.maxY {
					a.maxY = y
				}
				oc := oLabel[i]
				if oc >= 0 && !seenFrag[oc] {
					seenFrag[oc] = true
					a.fragments++
				}
			}
			for _, a := range accs {
				if a.area == 0 || a.area < mp.MinArea {
					continue
				}
				out = append(out, Defect{
					Material:  m,
					Polarity:  pol,
					Area:      a.area,
					BBox:      Rect{X: a.minX, Y: a.minY, W: a.maxX - a.minX + 1, H: a.maxY - a.minY + 1},
					Fragments: a.fragments,
				})
			}
		}
	}
	return out
}

// sameDefectMultiset 比较两个缺陷多重集的 (Material, Polarity, Area, BBox)。
func sameDefectMultiset(a, b []Defect) bool {
	if len(a) != len(b) {
		return false
	}
	type key struct {
		m    Material
		p    Polarity
		area int
		bb   Rect
		frag int
	}
	counts := map[key]int{}
	for _, d := range a {
		counts[key{d.Material, d.Polarity, d.Area, d.BBox, d.Fragments}]++
	}
	for _, d := range b {
		k := key{d.Material, d.Polarity, d.Area, d.BBox, d.Fragments}
		counts[k]--
		if counts[k] < 0 {
			return false
		}
	}
	for _, c := range counts {
		if c != 0 {
			return false
		}
	}
	return true
}

func TestLearn(t *testing.T) {
	rng := rand.New(rand.NewSource(999))
	w, h := 64, 48
	img := Image{Width: w, Height: h, Pix: make([]uint8, w*h)}
	reg := Regions{Width: w, Height: h, Material: make([]uint8, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			if x < w/2 {
				reg.Material[idx] = uint8(Metal)
				img.Pix[idx] = uint8(120 + rng.Intn(11) - 5)
			} else {
				reg.Material[idx] = uint8(SR)
				img.Pix[idx] = uint8(40 + rng.Intn(11) - 5)
			}
		}
	}

	base := DefaultParams()
	learned, err := Learn(img, reg, 3, base)
	if err != nil {
		t.Fatal(err)
	}
	// 用测量统计验证公式。
	rep, err := Inspect(img, reg, base)
	if err != nil {
		t.Fatal(err)
	}
	for _, mi := range []Material{Metal, SR} {
		ms := rep.PerMaterial[mi]
		wantLow := math.Floor(ms.Mean - 3*ms.Std)
		wantHigh := math.Ceil(ms.Mean + 3*ms.Std)
		if wantLow < 0 {
			wantLow = 0
		}
		if wantHigh > 255 {
			wantHigh = 255
		}
		if learned.Materials[mi].Low != wantLow {
			t.Fatalf("材质 %v Low 期望 %v，得到 %v", mi, wantLow, learned.Materials[mi].Low)
		}
		if learned.Materials[mi].High != wantHigh {
			t.Fatalf("材质 %v High 期望 %v，得到 %v", mi, wantHigh, learned.Materials[mi].High)
		}
	}
	// Open/Base 保持关闭。
	if learned.Materials[Open].Enabled || learned.Materials[Base].Enabled {
		t.Fatalf("Open/Base 应保持关闭")
	}

	// 用学习参数检查，标记像素应很少。
	rep2, err := Inspect(img, reg, learned)
	if err != nil {
		t.Fatal(err)
	}
	if rep2.FlaggedPixels > w*h/100 {
		t.Fatalf("k=3 标记像素过多：%d", rep2.FlaggedPixels)
	}

	// k=8 应无标记。
	learned8, err := Learn(img, reg, 8, base)
	if err != nil {
		t.Fatal(err)
	}
	rep3, err := Inspect(img, reg, learned8)
	if err != nil {
		t.Fatal(err)
	}
	if rep3.FlaggedPixels != 0 {
		t.Fatalf("k=8 期望 0 标记，得到 %d", rep3.FlaggedPixels)
	}

	// k<0 与 NaN 报错。
	if _, err := Learn(img, reg, -1, base); err == nil {
		t.Fatalf("k=-1 期望错误")
	}
	if _, err := Learn(img, reg, math.NaN(), base); err == nil {
		t.Fatalf("k=NaN 期望错误")
	}
}

func TestValidationErrors(t *testing.T) {
	img, reg := uniform(8, 8, 100, Metal)

	// Low > High
	p := DefaultParams()
	p.Materials[Metal].Low = 200
	p.Materials[Metal].High = 100
	if _, err := Inspect(img, reg, p); err == nil {
		t.Fatalf("Low>High 期望错误")
	}
	// Size 0
	p = DefaultParams()
	p.Materials[Metal].Size = 0
	if _, err := Inspect(img, reg, p); err == nil {
		t.Fatalf("Size=0 期望错误")
	}
	// MinArea 0
	p = DefaultParams()
	p.Materials[Metal].MinArea = 0
	if _, err := Inspect(img, reg, p); err == nil {
		t.Fatalf("MinArea=0 期望错误")
	}
	// 尺寸不匹配
	bad := Image{Width: 8, Height: 8, Pix: make([]uint8, 10)}
	if _, err := Inspect(bad, reg, DefaultParams()); err == nil {
		t.Fatalf("像素数不匹配期望错误")
	}
	badReg := Regions{Width: 4, Height: 4, Material: make([]uint8, 16)}
	if _, err := Inspect(img, badReg, DefaultParams()); err == nil {
		t.Fatalf("材质图尺寸不匹配期望错误")
	}
	// 材质值 4
	badReg2 := Regions{Width: 8, Height: 8, Material: make([]uint8, 64)}
	badReg2.Material[0] = 4
	if _, err := Inspect(img, badReg2, DefaultParams()); err == nil {
		t.Fatalf("材质值 4 期望错误")
	}
}

func TestProbeRect(t *testing.T) {
	w, h := 64, 48
	img, reg := buildImage(w, h, func(x, y int) (uint8, Material) {
		if x >= 10 && x < 20 && y >= 10 && y < 20 {
			return 200, Metal
		}
		return 100, Metal
	})

	p := DefaultParams()
	p.Materials[Metal].Low = 0
	p.Materials[Metal].High = 150
	p.Materials[Metal].Size = 1
	p.Materials[Metal].MinArea = 1
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("期望 1 个缺陷，得到 %d", len(rep.Defects))
	}

	// 相交矩形。
	pr, err := ProbeRect(img, reg, p, rep, Rect{X: 5, Y: 5, W: 20, H: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(pr.Defects) != 1 {
		t.Fatalf("期望 1 个相交缺陷，得到 %d", len(pr.Defects))
	}
	if pr.PerMaterial[Metal].Pixels != 20*20 {
		t.Fatalf("期望 400 像素，得到 %d", pr.PerMaterial[Metal].Pixels)
	}

	// 不相交矩形。
	pr, err = ProbeRect(img, reg, p, rep, Rect{X: 40, Y: 40, W: 10, H: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(pr.Defects) != 0 {
		t.Fatalf("期望 0 个相交缺陷，得到 %d", len(pr.Defects))
	}

	// 部分超出图像，应被裁剪。
	pr, err = ProbeRect(img, reg, p, rep, Rect{X: 60, Y: 40, W: 20, H: 20})
	if err != nil {
		t.Fatal(err)
	}
	if pr.Rect.W != 4 || pr.Rect.H != 8 {
		t.Fatalf("期望裁剪为 4x8，得到 %dx%d", pr.Rect.W, pr.Rect.H)
	}

	// 空矩形。
	pr, err = ProbeRect(img, reg, p, rep, Rect{X: 100, Y: 100, W: 10, H: 10})
	if err != nil {
		t.Fatal(err)
	}
	if pr.Rect.W != 0 || pr.Rect.H != 0 || len(pr.Defects) != 0 {
		t.Fatalf("期望空报告，得到 %+v", pr)
	}
}

func TestIndependence(t *testing.T) {
	w, h := 32, 24
	img, reg := buildImage(w, h, func(x, y int) (uint8, Material) {
		if x == 5 && y == 5 {
			return 200, Metal
		}
		return 100, Metal
	})
	p := DefaultParams()
	p.Materials[Metal].Low = 0
	p.Materials[Metal].High = 150
	p.Materials[Metal].MinArea = 1

	imgCopy := make([]uint8, len(img.Pix))
	copy(imgCopy, img.Pix)
	regCopy := make([]uint8, len(reg.Material))
	copy(regCopy, reg.Material)

	rep1, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	rep2, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep1.Defects) != len(rep2.Defects) {
		t.Fatalf("两次调用结果不同")
	}
	for i := range rep1.Defects {
		if rep1.Defects[i] != rep2.Defects[i] {
			t.Fatalf("两次调用结果不同：%+v vs %+v", rep1.Defects[i], rep2.Defects[i])
		}
	}
	for i := range img.Pix {
		if img.Pix[i] != imgCopy[i] {
			t.Fatalf("Inspect 修改了输入图像")
		}
	}
	for i := range reg.Material {
		if reg.Material[i] != regCopy[i] {
			t.Fatalf("Inspect 修改了输入材质图")
		}
	}
}
