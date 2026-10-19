package inspect

import "testing"

// contiguityParams 返回只启用 SR 的参数：区间 [0,100]，关闭 Metal，
// 关闭边缘排除与局部背景，便于精确控制被标记像素。
func contiguityParams() Params {
	p := DefaultParams()
	p.Materials[SR].Low = 0
	p.Materials[SR].High = 100
	p.Materials[Metal].Enabled = false
	p.EdgeMargin = 0
	p.BackgroundRadius = 0
	return p
}

// contiguityImage 构建 16x3 的 SR 图像，背景值 50，在中间行 y=1 的
// 指定 x 位置写入给定值。
func contiguityImage(vals map[int]uint8) (Image, Regions) {
	return buildImage(16, 3, func(x, y int) (uint8, Material) {
		if y == 1 {
			if v, ok := vals[x]; ok {
				return v, SR
			}
		}
		return 50, SR
	})
}

func TestContiguousPixelsDetected(t *testing.T) {
	// 105、106 相邻，Size=2 时构成一个 2 像素的连通块。
	img, reg := contiguityImage(map[int]uint8{4: 105, 5: 106})
	p := contiguityParams()
	p.Materials[SR].Size = 2
	p.Materials[SR].MinArea = 1
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	d := rep.Defects[0]
	if d.Area != 2 {
		t.Fatalf("期望 Area=2，得到 %d", d.Area)
	}
	if d.Polarity != Bright {
		t.Fatalf("期望 Bright，得到 %v", d.Polarity)
	}
	if d.BBox.W != 2 || d.BBox.H != 1 {
		t.Fatalf("期望 BBox 2x1，得到 %dx%d", d.BBox.W, d.BBox.H)
	}

	// 105、106、107 相邻，Size=3 时构成一个 3 像素的连通块。
	img, reg = contiguityImage(map[int]uint8{4: 105, 5: 106, 6: 107})
	p.Materials[SR].Size = 3
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("Size=3 期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	if rep.Defects[0].Area != 3 {
		t.Fatalf("Size=3 期望 Area=3，得到 %d", rep.Defects[0].Area)
	}

	// 105、106 只有 2 个像素，Size=3 时被过滤。
	img, reg = contiguityImage(map[int]uint8{4: 105, 5: 106})
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("Size=3 期望 0 个缺陷，得到 %d", len(rep.Defects))
	}
	if rep.Filtered != 1 {
		t.Fatalf("Size=3 期望 Filtered=1，得到 %d", rep.Filtered)
	}
}

func TestGapBreaksContiguity(t *testing.T) {
	// 105、99、102：99 未被标记，因此 105 与 102 不连通。
	// 三个值的平均为 102，高于 High=100，但判定逐像素进行，绝不使用平均值。
	img, reg := contiguityImage(map[int]uint8{4: 105, 5: 99, 6: 102})
	p := contiguityParams()
	p.Materials[SR].Size = 2
	p.Materials[SR].MinArea = 1
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("Size=2 期望 0 个缺陷，得到 %d", len(rep.Defects))
	}
	if rep.FlaggedPixels != 2 {
		t.Fatalf("Size=2 期望 FlaggedPixels=2，得到 %d", rep.FlaggedPixels)
	}
	if rep.Filtered != 2 {
		t.Fatalf("Size=2 期望 Filtered=2，得到 %d", rep.Filtered)
	}

	// Size=1：两个孤立的被标记像素各自成为一个缺陷。
	p.Materials[SR].Size = 1
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 2 {
		t.Fatalf("Size=1 期望 2 个缺陷，得到 %d", len(rep.Defects))
	}
	for _, d := range rep.Defects {
		if d.Area != 1 {
			t.Fatalf("Size=1 期望每个缺陷 Area=1，得到 %d", d.Area)
		}
	}
}

func TestNoAveraging(t *testing.T) {
	// 105、100、106：100 恰好等于 High，区间为闭区间，因此不被标记。
	// 105 与 106 被 100 隔开，Size=2 时无缺陷。
	img, reg := contiguityImage(map[int]uint8{4: 105, 5: 100, 6: 106})
	p := contiguityParams()
	p.Materials[SR].Size = 2
	p.Materials[SR].MinArea = 1
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("期望 0 个缺陷，得到 %d", len(rep.Defects))
	}

	// 130、60、130：60 未被标记，两个 130 不连通；平均值 106.7 高于 High 也不起作用。
	img, reg = contiguityImage(map[int]uint8{4: 130, 5: 60, 6: 130})
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("期望 0 个缺陷，得到 %d", len(rep.Defects))
	}
}

func TestContiguityGeometry(t *testing.T) {
	p := contiguityParams()
	p.Materials[SR].Size = 2
	p.Materials[SR].MinArea = 1

	// 竖直相邻：(4,0) 与 (4,1)。
	img, reg := buildImage(16, 3, func(x, y int) (uint8, Material) {
		if x == 4 && (y == 0 || y == 1) {
			return 105, SR
		}
		return 50, SR
	})
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("竖直相邻期望 1 个缺陷，得到 %d", len(rep.Defects))
	}

	// 对角相邻：(4,0) 与 (5,1)，8 连通。
	img, reg = buildImage(16, 3, func(x, y int) (uint8, Material) {
		if (x == 4 && y == 0) || (x == 5 && y == 1) {
			return 105, SR
		}
		return 50, SR
	})
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("对角相邻期望 1 个缺陷，得到 %d", len(rep.Defects))
	}

	// 对角相隔两像素：(4,0) 与 (6,2)，不连通。
	img, reg = buildImage(16, 3, func(x, y int) (uint8, Material) {
		if (x == 4 && y == 0) || (x == 6 && y == 2) {
			return 105, SR
		}
		return 50, SR
	})
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("对角相隔两像素期望 0 个缺陷，得到 %d", len(rep.Defects))
	}
}

func TestBrightDarkNeverJoin(t *testing.T) {
	p := contiguityParams()
	p.Materials[SR].Low = 40
	p.Materials[SR].High = 100
	p.Materials[SR].Size = 2
	p.Materials[SR].MinArea = 1

	// 亮像素 105 紧邻暗像素 30：极性不同，永不合并。
	img, reg := contiguityImage(map[int]uint8{4: 105, 5: 30})
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("亮暗相邻期望 0 个缺陷，得到 %d", len(rep.Defects))
	}

	// 105、106、30、29：两个亮像素与两个暗像素各自成块。
	img, reg = contiguityImage(map[int]uint8{4: 105, 5: 106, 6: 30, 7: 29})
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 2 {
		t.Fatalf("期望 2 个缺陷，得到 %d", len(rep.Defects))
	}
	bright, dark := 0, 0
	for _, d := range rep.Defects {
		switch d.Polarity {
		case Bright:
			bright++
		case Dark:
			dark++
		}
	}
	if bright != 1 || dark != 1 {
		t.Fatalf("期望 1 亮 1 暗，得到 %d 亮 %d 暗", bright, dark)
	}
}

func TestMaterialsNeverJoin(t *testing.T) {
	// 一个 SR 像素与一个 Metal 像素相邻，两者都启用且 High=100，Size=2。
	img, reg := buildImage(16, 3, func(x, y int) (uint8, Material) {
		if y == 1 {
			if x == 4 {
				return 105, SR
			}
			if x == 5 {
				return 105, Metal
			}
		}
		return 50, SR
	})
	p := DefaultParams()
	p.Materials[SR].Low = 0
	p.Materials[SR].High = 100
	p.Materials[Metal].Low = 0
	p.Materials[Metal].High = 100
	p.Materials[SR].Size = 2
	p.Materials[SR].MinArea = 1
	p.Materials[Metal].Size = 2
	p.Materials[Metal].MinArea = 1
	p.EdgeMargin = 0
	p.BackgroundRadius = 0
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("不同材质期望 0 个缺陷，得到 %d", len(rep.Defects))
	}
}
