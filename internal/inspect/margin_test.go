package inspect

import (
	"math"
	"reflect"
	"testing"
)

// naiveEdgeExclusion 是边缘排除的朴素参考实现。
func naiveEdgeExclusion(reg Regions, r int) []uint8 {
	if r <= 0 {
		return nil
	}
	w, h := reg.Width, reg.Height
	out := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m := reg.Material[y*w+x]
			excluded := false
			for dy := -r; dy <= r && !excluded; dy++ {
				ny := y + dy
				if ny < 0 || ny >= h {
					continue
				}
				for dx := -r; dx <= r; dx++ {
					nx := x + dx
					if nx < 0 || nx >= w {
						continue
					}
					if reg.Material[ny*w+nx] != m {
						excluded = true
						break
					}
				}
			}
			if excluded {
				out[y*w+x] = 1
			}
		}
	}
	return out
}

func TestEdgeExclusionMatchesNaive(t *testing.T) {
	w, h := 40, 30
	reg := Regions{Width: w, Height: h, Material: make([]uint8, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			reg.Material[y*w+x] = uint8((x/7 + y/5*3) % 4)
		}
	}
	for _, r := range []int{0, 1, 2, 3, 7} {
		got := edgeExclusion(reg, r)
		want := naiveEdgeExclusion(reg, r)
		if r == 0 {
			if got != nil {
				t.Fatalf("r=0 期望 nil，得到非 nil")
			}
			continue
		}
		if len(got) != len(want) {
			t.Fatalf("r=%d 长度不一致：%d vs %d", r, len(got), len(want))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("r=%d 像素 %d 不一致：got=%d want=%d", r, i, got[i], want[i])
			}
		}
	}
}

// transitionImage 构建测试 2 使用的图像：列 0..31 为 SR（值 20），
// 其中列 30、31 为 100；列 32..63 为 Metal（值 200）。
func transitionImage() (Image, Regions) {
	w, h := 64, 32
	return buildImage(w, h, func(x, y int) (uint8, Material) {
		if x < 32 {
			if x >= 30 {
				return 100, SR
			}
			return 20, SR
		}
		return 200, Metal
	})
}

func TestEdgeMarginIgnoresTransition(t *testing.T) {
	img, reg := transitionImage()

	p := DefaultParams()
	p.Materials[SR].Low = 10
	p.Materials[SR].High = 30
	p.Materials[SR].Size = 1
	p.Materials[SR].MinArea = 1
	p.Materials[Metal].Enabled = false

	// EdgeMargin 0：列 30、31 为亮 SR，共 2*32=64 像素。
	p.EdgeMargin = 0
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("EdgeMargin=0 期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	if rep.Defects[0].Area != 64 {
		t.Fatalf("EdgeMargin=0 期望 Area=64，得到 %d", rep.Defects[0].Area)
	}
	if rep.EdgeIgnored != 0 {
		t.Fatalf("EdgeMargin=0 期望 EdgeIgnored=0，得到 %d", rep.EdgeIgnored)
	}

	// EdgeMargin 1：列 31 距 Metal 为 1 被忽略，列 30 距 Metal 为 2 仍被标记。
	p.EdgeMargin = 1
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("EdgeMargin=1 期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	if rep.Defects[0].Area != 32 {
		t.Fatalf("EdgeMargin=1 期望 Area=32，得到 %d", rep.Defects[0].Area)
	}

	// EdgeMargin 2：列 30、31 均被忽略，无缺陷。
	p.EdgeMargin = 2
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("EdgeMargin=2 期望 0 个缺陷，得到 %d", len(rep.Defects))
	}
	if rep.FlaggedPixels != 0 {
		t.Fatalf("EdgeMargin=2 期望 FlaggedPixels=0，得到 %d", rep.FlaggedPixels)
	}
	// 列 30..33 共 4 列 × 32 行 = 128。
	if rep.EdgeIgnored != 128 {
		t.Fatalf("EdgeMargin=2 期望 EdgeIgnored=128，得到 %d", rep.EdgeIgnored)
	}
}

func TestEdgeMarginStatsAndLearn(t *testing.T) {
	w, h := 64, 32
	img, reg := buildImage(w, h, func(x, y int) (uint8, Material) {
		if x < 32 {
			if x >= 29 {
				return 90, SR
			}
			return 20, SR
		}
		return 200, Metal
	})

	p := DefaultParams()
	p.Materials[SR].Low = 10
	p.Materials[SR].High = 30
	p.Materials[SR].Size = 1
	p.Materials[SR].MinArea = 1
	p.Materials[Metal].Low = 100
	p.Materials[Metal].High = 255
	p.Materials[Metal].Size = 1
	p.Materials[Metal].MinArea = 1

	// EdgeMargin 0：SR 全部 32 列 × 32 行 = 1024 像素。
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if rep.PerMaterial[SR].Pixels != 1024 {
		t.Fatalf("EdgeMargin=0 期望 SR Pixels=1024，得到 %d", rep.PerMaterial[SR].Pixels)
	}
	// SR mean 26.56, std 20.4: k=4 gives High 109 (> 90); k=3 would give 88 and is not enough.
	learned0, err := Learn(img, reg, 4, p)
	if err != nil {
		t.Fatal(err)
	}
	if learned0.Materials[SR].High <= 90 {
		t.Fatalf("EdgeMargin=0 期望 SR High > 90，得到 %v", learned0.Materials[SR].High)
	}

	// EdgeMargin 3：SR 列 0..28 共 29 列 × 32 行 = 928；Metal 列 35..63 共 29 列 × 32 行 = 928。
	p.EdgeMargin = 3
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if rep.PerMaterial[SR].Pixels != 928 {
		t.Fatalf("EdgeMargin=3 期望 SR Pixels=928，得到 %d", rep.PerMaterial[SR].Pixels)
	}
	if rep.PerMaterial[Metal].Pixels != 928 {
		t.Fatalf("EdgeMargin=3 期望 Metal Pixels=928，得到 %d", rep.PerMaterial[Metal].Pixels)
	}
	if math.Abs(rep.PerMaterial[SR].Mean-20) > 1e-9 {
		t.Fatalf("EdgeMargin=3 期望 SR Mean=20，得到 %v", rep.PerMaterial[SR].Mean)
	}
	learned3, err := Learn(img, reg, 3, p)
	if err != nil {
		t.Fatal(err)
	}
	if learned3.Materials[SR].Low != 20 || learned3.Materials[SR].High != 20 {
		t.Fatalf("EdgeMargin=3 期望 SR 区间 [20,20]，得到 [%v,%v]", learned3.Materials[SR].Low, learned3.Materials[SR].High)
	}
	if learned3.EdgeMargin != 3 {
		t.Fatalf("期望 learned EdgeMargin=3，得到 %d", learned3.EdgeMargin)
	}
}

func TestEdgeMarginValidation(t *testing.T) {
	img, reg := uniform(8, 8, 100, Metal)

	p := DefaultParams()
	p.EdgeMargin = -1
	if err := p.Validate(); err == nil {
		t.Fatalf("EdgeMargin=-1 期望 Validate 错误")
	}
	if _, err := Inspect(img, reg, p); err == nil {
		t.Fatalf("EdgeMargin=-1 期望 Inspect 错误")
	}
	if _, err := Learn(img, reg, 3, p); err == nil {
		t.Fatalf("EdgeMargin=-1 期望 Learn 错误")
	}

	p.EdgeMargin = MaxEdgeMargin + 1
	if err := p.Validate(); err == nil {
		t.Fatalf("EdgeMargin=MaxEdgeMargin+1 期望 Validate 错误")
	}
	if _, err := Inspect(img, reg, p); err == nil {
		t.Fatalf("EdgeMargin=MaxEdgeMargin+1 期望 Inspect 错误")
	}
	if _, err := Learn(img, reg, 3, p); err == nil {
		t.Fatalf("EdgeMargin=MaxEdgeMargin+1 期望 Learn 错误")
	}

	p.EdgeMargin = 0
	if err := p.Validate(); err != nil {
		t.Fatalf("EdgeMargin=0 期望通过，得到 %v", err)
	}
	if _, err := Inspect(img, reg, p); err != nil {
		t.Fatalf("EdgeMargin=0 期望 Inspect 通过，得到 %v", err)
	}

	p.EdgeMargin = MaxEdgeMargin
	if err := p.Validate(); err != nil {
		t.Fatalf("EdgeMargin=MaxEdgeMargin 期望通过，得到 %v", err)
	}
	if _, err := Inspect(img, reg, p); err != nil {
		t.Fatalf("EdgeMargin=MaxEdgeMargin 期望 Inspect 通过，得到 %v", err)
	}
}

func TestProbeRectEdgeMargin(t *testing.T) {
	img, reg := transitionImage()

	p := DefaultParams()
	p.Materials[SR].Low = 10
	p.Materials[SR].High = 30
	p.Materials[SR].Size = 1
	p.Materials[SR].MinArea = 1
	p.Materials[Metal].Low = 100
	p.Materials[Metal].High = 255
	p.Materials[Metal].Size = 1
	p.Materials[Metal].MinArea = 1
	p.EdgeMargin = 2

	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	pr, err := ProbeRect(img, reg, p, rep, Rect{X: 28, Y: 0, W: 8, H: 32})
	if err != nil {
		t.Fatal(err)
	}
	// 列 30..33 共 4 列 × 32 行 = 128。
	if pr.Ignored != 128 {
		t.Fatalf("期望 Ignored=128，得到 %d", pr.Ignored)
	}
	// 列 28、29 为 SR。
	if pr.PerMaterial[SR].Pixels != 64 {
		t.Fatalf("期望 SR Pixels=64，得到 %d", pr.PerMaterial[SR].Pixels)
	}
	// 列 34、35 为 Metal。
	if pr.PerMaterial[Metal].Pixels != 64 {
		t.Fatalf("期望 Metal Pixels=64，得到 %d", pr.PerMaterial[Metal].Pixels)
	}
	if pr.PerMaterial[SR].Bright != 0 || pr.PerMaterial[SR].Dark != 0 {
		t.Fatalf("期望 SR Bright/Dark=0/0，得到 %d/%d", pr.PerMaterial[SR].Bright, pr.PerMaterial[SR].Dark)
	}
	if pr.PerMaterial[Metal].Bright != 0 || pr.PerMaterial[Metal].Dark != 0 {
		t.Fatalf("期望 Metal Bright/Dark=0/0，得到 %d/%d", pr.PerMaterial[Metal].Bright, pr.PerMaterial[Metal].Dark)
	}
}

func TestEdgeMarginDeterministic(t *testing.T) {
	w, h := 80, 60
	img, reg := buildImage(w, h, func(x, y int) (uint8, Material) {
		if (x*7+y*13)%11 < 5 {
			return uint8(100 + (x*3+y*5)%40), Metal
		}
		return uint8(50 + (x*5+y*3)%30), SR
	})

	p := DefaultParams()
	p.Materials[Metal].Low = 60
	p.Materials[Metal].High = 150
	p.Materials[Metal].Size = 3
	p.Materials[Metal].MinArea = 2
	p.Materials[SR].Low = 40
	p.Materials[SR].High = 90
	p.Materials[SR].Size = 3
	p.Materials[SR].MinArea = 2
	p.EdgeMargin = 3

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
