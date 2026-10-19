package inspect

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

// ttMetalParams 返回只启用 Metal、区间规则关闭、并给定 Abs/Pct 的参数。
// 其余材质关闭，Size/MinArea 设为 1 以免被区间 Size 过滤（区间已关闭，Size 本就不参与）。
func ttMetalParams(abs, pct TileRule) Params {
	p := DefaultParams()
	for i := range p.Materials {
		p.Materials[i].Enabled = false
	}
	mp := &p.Materials[Metal]
	mp.Enabled = true
	mp.IntervalOff = true
	mp.Size = 1
	mp.MinArea = 1
	mp.Abs = abs
	mp.Pct = pct
	return p
}

// ttAbs 构造一条启用的绝对值规则（偏移 <= 0 表示该极性不检查）。
func ttAbs(bright, dark float64, bsize, dsize int) TileRule {
	return TileRule{Enabled: true, Bright: bright, Dark: dark, BrightSize: bsize, DarkSize: dsize}
}

// ttPct 构造一条启用的百分比规则。
func ttPct(bright, dark float64, bsize, dsize int) TileRule {
	return TileRule{Enabled: true, Bright: bright, Dark: dark, BrightSize: bsize, DarkSize: dsize}
}

// ttBlock 在 uniform 图像上把 [x,x+w) x [y,y+h) 改成 val（材质不变），返回新图。
func ttBlock(base Image, x, y, w, h int, val uint8) Image {
	img := Image{Width: base.Width, Height: base.Height, Pix: append([]uint8(nil), base.Pix...)}
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			img.Pix[yy*img.Width+xx] = val
		}
	}
	return img
}

// ttPutPixel 设置单像素，返回新图副本。
func ttPutPixel(base Image, x, y int, val uint8) Image {
	return ttBlock(base, x, y, 1, 1, val)
}

// ttOwnMean 按契约计算某小格内某材质有效像素的算术平均（本测试构造的全同材质全有效格）。
// 用于第 4 条独立推导参考值，而不是猜实现。
func ttOwnMean(sum float64, count int) float64 {
	return sum / float64(count)
}

func TestTileAbsBright(t *testing.T) {
	// 64x64 全 Metal GV100，TileSize32 → 2x2 格，每格 32x32=1024 像素（>=64，用本格平均）。
	// 左上格内放 3x3 GV140 块：本格平均 = (1015*100 + 9*140)/1024 = 100 + 360/1024 = 100.3515625。
	// Abs.Bright=30：判定线 130.3515625，140 > 130.35 → 9 个像素被标记；100 不被标记。
	img, reg := uniform(64, 64, 100, Metal)
	img = ttBlock(img, 0, 0, 3, 3, 140)

	p := ttMetalParams(ttAbs(30, 0, 1, 1), TileRule{})
	p.TileSize = 32
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	d := rep.Defects[0]
	if d.Area != 9 {
		t.Fatalf("期望 Area=9，得到 %d", d.Area)
	}
	if d.Polarity != Bright {
		t.Fatalf("期望 Bright，得到 %v", d.Polarity)
	}
	if d.Rules != RuleAbsolute {
		t.Fatalf("期望 Rules=RuleAbsolute，得到 %v", d.Rules)
	}
	if d.BBox != (Rect{X: 0, Y: 0, W: 3, H: 3}) {
		t.Fatalf("期望 BBox={0,0,3,3}，得到 %v", d.BBox)
	}

	// Bright=50：判定线 150.35，140 不 > → 无缺陷。
	p = ttMetalParams(ttAbs(50, 0, 1, 1), TileRule{})
	p.TileSize = 32
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("Bright=50 期望 0 缺陷，得到 %d", len(rep.Defects))
	}
}

func TestTileAbsDark(t *testing.T) {
	// 同第 1 条但块为 GV60：本格平均 = 100 - 360/1024 = 99.6484375。
	// Dark=30：判定线 ref-30=69.6484375，60 < 69.65 → 9 像素暗缺陷。
	img, reg := uniform(64, 64, 100, Metal)
	img = ttBlock(img, 0, 0, 3, 3, 60)

	p := ttMetalParams(ttAbs(0, 30, 1, 1), TileRule{})
	p.TileSize = 32
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	d := rep.Defects[0]
	if d.Polarity != Dark || d.Area != 9 || d.Rules != RuleAbsolute {
		t.Fatalf("期望 Dark/9/RuleAbsolute，得到 %v/%d/%v", d.Polarity, d.Area, d.Rules)
	}

	// Dark=0 表示该极性不检查 → 无缺陷。
	p = ttMetalParams(ttAbs(0, 0, 1, 1), TileRule{})
	p.TileSize = 32
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("Dark=0 期望 0 缺陷，得到 %d", len(rep.Defects))
	}
}

func TestTilePercent(t *testing.T) {
	// 32x32 单格全 Metal（T=32：n=1），1024 像素。一个像素 GV120。
	// 本格平均 = (1023*100 + 120)/1024 = 100 + 20/1024 = 100.01953125。
	// Pct.Bright=15 → 乘数 1.15，判定线 = 115.0224609375；120 > 115.02 → 该像素亮缺陷。
	img, reg := uniform(32, 32, 100, Metal)
	img120 := ttPutPixel(img, 5, 5, 120)

	p := ttMetalParams(TileRule{}, ttPct(15, 0, 1, 1))
	p.TileSize = 32
	rep, err := Inspect(img120, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("GV120 期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	d := rep.Defects[0]
	if d.Polarity != Bright || d.Rules != RulePercent || d.Area != 1 {
		t.Fatalf("期望 Bright/RulePercent/1，得到 %v/%v/%d", d.Polarity, d.Rules, d.Area)
	}

	// GV112：平均 = (1023*100 + 112)/1024 = 100 + 12/1024 = 100.01171875，判定线 115.013；112 不 > → 不检出。
	img112 := ttPutPixel(img, 5, 5, 112)
	rep, err = Inspect(img112, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("GV112 期望 0 缺陷，得到 %d", len(rep.Defects))
	}

	// Pct.Dark=15，某像素 80：平均 = (1023*100 + 80)/1024 = 100 - 20/1024 = 99.98046875。
	// 乘数 0.85，判定线 = 84.9833984375；80 < 84.98 → 暗缺陷。
	img80 := ttPutPixel(img, 5, 5, 80)
	pDark := ttMetalParams(TileRule{}, ttPct(0, 15, 1, 1))
	rep, err = Inspect(img80, reg, pDark)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("GV80 暗 期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	if rep.Defects[0].Polarity != Dark || rep.Defects[0].Rules != RulePercent {
		t.Fatalf("期望 Dark/RulePercent，得到 %v/%v", rep.Defects[0].Polarity, rep.Defects[0].Rules)
	}
}

func TestTileStrictInequality(t *testing.T) {
	// 16x16 单格（T=16：n=1），256 像素，255 个 100 + 1 个 156。cnt>=64 用本格平均。
	// 平均 = (255*100 + 156)/256 = 25656/256 = 100.21875（精确可表示）。
	img, reg := uniform(16, 16, 100, Metal)
	img = ttPutPixel(img, 3, 3, 156)

	ref := ttOwnMean(255*100+156, 256) // 100.21875
	brightEq := 156.0 - ref            // 55.78125（精确可表示）
	if ref != 100.21875 {
		t.Fatalf("参考值推导异常：ref=%v", ref)
	}

	// Bright 恰等于 v-ref → 判定线 ref+Bright = 156.0；严格不等号 156 > 156 为假 → 不检出。
	p := ttMetalParams(ttAbs(brightEq, 0, 1, 1), TileRule{})
	p.TileSize = 16
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("Bright=ref差 期望 0 缺陷（严格大于），得到 %d", len(rep.Defects))
	}

	// Bright 略小 → 判定线 < 156 → 检出。
	p = ttMetalParams(ttAbs(brightEq-1e-6, 0, 1, 1), TileRule{})
	p.TileSize = 16
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("Bright 略小 期望 1 缺陷，得到 %d", len(rep.Defects))
	}
	if rep.Defects[0].Area != 1 || rep.Defects[0].Rules != RuleAbsolute {
		t.Fatalf("期望 Area=1/RuleAbsolute，得到 %d/%v", rep.Defects[0].Area, rep.Defects[0].Rules)
	}
}

func TestTileSizeFilter(t *testing.T) {
	// 16x16 单格全 100，2 个相邻亮像素 200。平均 = (254*100 + 2*200)/256 = 25800/256 = 100.78125。
	// Bright=30 → 判定线 130.78，200 被标记（2 像素相连）。
	img, reg := uniform(16, 16, 100, Metal)
	imgPair := ttBlock(img, 2, 2, 2, 1, 200)

	// BrightSize=3：连通块只有 2 像素 < 3 → 丢弃，原始并集连通块无被接受像素 → Filtered=1。
	p := ttMetalParams(ttAbs(30, 0, 3, 1), TileRule{})
	p.TileSize = 16
	rep, err := Inspect(imgPair, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 || rep.Filtered != 1 {
		t.Fatalf("BrightSize=3 期望 0 缺陷/Filtered=1，得到 %d/%d", len(rep.Defects), rep.Filtered)
	}

	// BrightSize=2 → 2 像素块被接受，1 缺陷 Area=2。
	p = ttMetalParams(ttAbs(30, 0, 2, 1), TileRule{})
	p.TileSize = 16
	rep, err = Inspect(imgPair, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 || rep.Defects[0].Area != 2 {
		t.Fatalf("BrightSize=2 期望 1 缺陷 Area2，得到 %d", len(rep.Defects))
	}

	// 亮/暗极性互不影响：再加一个远离亮块的暗点 60。
	// 平均 = (253*100 + 2*200 + 1*60)/256 = 25760/256 = 100.625。
	// Dark=30 → 判定线 ref-30=70.625，60 < 70.63 → 暗标记（1 像素，DarkSize=1 接受）。
	// BrightSize=3 仍把亮对过滤掉。
	imgBoth := ttPutPixel(imgPair, 10, 10, 60)
	p = ttMetalParams(ttAbs(30, 30, 3, 1), TileRule{})
	p.TileSize = 16
	rep, err = Inspect(imgBoth, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("期望仅 1 个缺陷（暗点），得到 %d", len(rep.Defects))
	}
	if rep.Defects[0].Polarity != Dark || rep.Defects[0].Area != 1 {
		t.Fatalf("期望暗/1，得到 %v/%d", rep.Defects[0].Polarity, rep.Defects[0].Area)
	}
	if rep.Filtered != 1 {
		t.Fatalf("期望亮对被过滤 Filtered=1，得到 %d", rep.Filtered)
	}

	// MinArea=3 对 Abs 同样生效：BrightSize=1 但 MinArea=3 → 2 像素块被过滤。
	// need = max(BrightSize=1, MinArea=3) = 3 > 2。
	p = ttMetalParams(ttAbs(30, 0, 1, 1), TileRule{})
	p.Materials[Metal].MinArea = 3
	p.TileSize = 16
	rep, err = Inspect(imgPair, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 || rep.Filtered != 1 {
		t.Fatalf("MinArea=3 期望 0 缺陷/Filtered=1，得到 %d/%d", len(rep.Defects), rep.Filtered)
	}
}

func TestTileIndependentTiles(t *testing.T) {
	// 64x32，T=32 → x 方向 n=2、y 方向 n=1，共 2 格。左格全 200，右格全 100。
	// 左格平均 = (1023*200 + 130)/1024 = 200 - 70/1024 = 199.931640625。
	// 右格平均 = (1023*100 + 130)/1024 = 100 + 30/1024 = 100.029296875。
	// Abs{20,20}：右格 130 > 100.029+20=120.03 → 亮；左格 130 < 199.93-20=179.93 → 暗。
	img, reg := buildImage(64, 32, func(x, y int) (uint8, Material) {
		if x < 32 {
			return 200, Metal
		}
		return 100, Metal
	})
	img = ttPutPixel(img, 5, 5, 130)   // 左格（全 200 中）
	img = ttPutPixel(img, 40, 10, 130) // 右格（全 100 中）

	p := ttMetalParams(ttAbs(20, 20, 1, 1), TileRule{})
	p.TileSize = 32
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 2 {
		t.Fatalf("期望 2 个缺陷，得到 %d", len(rep.Defects))
	}
	var sawBright, sawDark bool
	for _, d := range rep.Defects {
		switch d.Polarity {
		case Bright:
			sawBright = true
			// 右格的 130 在 x=40。
			if d.BBox.X != 40 || d.BBox.Y != 10 || d.Area != 1 {
				t.Fatalf("亮缺陷位置/面积错误：%v area=%d", d.BBox, d.Area)
			}
		case Dark:
			sawDark = true
			// 左格的 130 在 x=5。
			if d.BBox.X != 5 || d.BBox.Y != 5 || d.Area != 1 {
				t.Fatalf("暗缺陷位置/面积错误：%v area=%d", d.BBox, d.Area)
			}
		}
	}
	if !sawBright || !sawDark {
		t.Fatalf("期望同时有亮与暗缺陷，got bright=%v dark=%v", sawBright, sawDark)
	}
}

func TestTilePerMaterialMean(t *testing.T) {
	// 32x32 单格，左半 Metal 全 200、右半 SR 全 100；每材质本格平均按自身有效像素分别算。
	// Metal 平均=200，SR 平均=100，Abs{20,20} → 200/100 都不偏离各自平均 → 无缺陷。
	base, reg := buildImage(32, 32, func(x, y int) (uint8, Material) {
		if x < 16 {
			return 200, Metal
		}
		return 100, SR
	})

	p := ttMetalParams(ttAbs(20, 20, 1, 1), TileRule{})
	p.TileSize = 32
	// 需要同时启用 SR，用独立的参数构造。
	p.Materials[SR].Enabled = true
	p.Materials[SR].IntervalOff = true
	p.Materials[SR].Size = 1
	p.Materials[SR].MinArea = 1
	p.Materials[SR].Abs = ttAbs(20, 20, 1, 1)

	rep, err := Inspect(base, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("两材质各按自身平均，期望 0 缺陷，得到 %d", len(rep.Defects))
	}

	// 在 SR 区域放一个 135：SR 平均 = (511*100 + 135)/512 = 100 + 35/512 = 100.068359375。
	// 判定线 120.07 → 135 > 120.07 → 恰 1 个 SR 亮缺陷；Metal 不受影响。
	img := ttPutPixel(base, 20, 8, 135)
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("期望 1 个 SR 缺陷，得到 %d", len(rep.Defects))
	}
	d := rep.Defects[0]
	if d.Material != SR || d.Polarity != Bright || d.Rules != RuleAbsolute || d.Area != 1 {
		t.Fatalf("期望 SR/Bright/Abs/1，得到 %v/%v/%v/%d", d.Material, d.Polarity, d.Rules, d.Area)
	}
}

func TestTileEdgeMarginExcluded(t *testing.T) {
	// 材质图左半 Metal(全 200) 右半 SR(全 100)，64x32，EdgeMargin=2。
	// 交界列：Metal 侧 x=30,31、SR 侧 x=32,33 距异材质 <=2 → 被排除，既不参与小格平均也不被标记。
	img, reg := buildImage(64, 32, func(x, y int) (uint8, Material) {
		if x < 32 {
			return 200, Metal
		}
		return 100, SR
	})

	p := ttMetalParams(ttAbs(20, 20, 1, 1), TileRule{})
	p.EdgeMargin = 2
	p.TileSize = 32
	p.Materials[SR].Enabled = true
	p.Materials[SR].IntervalOff = true
	p.Materials[SR].Size = 1
	p.Materials[SR].MinArea = 1
	p.Materials[SR].Abs = ttAbs(20, 20, 1, 1)

	// 极亮像素落在被排除的交界列 x=31（Metal）→ 不参与、不标记 → 无缺陷。
	onEdge := ttPutPixel(img, 31, 10, 255)
	rep, err := Inspect(onEdge, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("交界极亮应被排除，期望 0 缺陷，得到 %d", len(rep.Defects))
	}

	// 同样的 255 远离交界（x=5）：Metal 有效像素不含被排除列，本格平均约 200，
	// 255 > 200+20=220 → 被检出。
	far := ttPutPixel(img, 5, 10, 255)
	rep, err = Inspect(far, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 || rep.Defects[0].Material != Metal {
		t.Fatalf("远离交界的 255 应被检出为 Metal 缺陷，得到 %d 个", len(rep.Defects))
	}
}

func TestTileRuleUnionAndRules(t *testing.T) {
	// 32x32 单格全 100，3x3 块 GV200。
	// 区间 High=110（亮）标记该块；Abs Bright=20 也标记（本格平均≈100.88，+20<200）。
	// 两条规则在同一极性同一连通块上取并集 → 1 个缺陷，Rules=RuleInterval|RuleAbsolute，Area=9。
	img, reg := uniform(32, 32, 100, Metal)
	img = ttBlock(img, 4, 4, 3, 3, 200)

	p := ttMetalParams(ttAbs(20, 0, 1, 1), TileRule{})
	p.TileSize = 32
	p.Materials[Metal].IntervalOff = false
	p.Materials[Metal].High = 110
	p.Materials[Metal].Low = 0
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 {
		t.Fatalf("期望 1 个缺陷，得到 %d", len(rep.Defects))
	}
	d := rep.Defects[0]
	if d.Rules != (RuleInterval | RuleAbsolute) {
		t.Fatalf("期望 Rules=Interval|Absolute，得到 %v", d.Rules)
	}
	if d.Area != 9 {
		t.Fatalf("期望 Area=9，得到 %d", d.Area)
	}

	// 只开区间规则（Abs 关闭）→ Rules=RuleInterval。
	p = ttMetalParams(TileRule{}, TileRule{})
	p.TileSize = 32
	p.Materials[Metal].IntervalOff = false
	p.Materials[Metal].High = 110
	rep, err = Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 1 || rep.Defects[0].Rules != RuleInterval {
		t.Fatalf("只开区间期望 1 缺陷/Rules=Interval，得到 %d", len(rep.Defects))
	}

	// RuleNames 短名顺序固定。
	names := RuleNames(RuleInterval | RulePercent)
	if len(names) != 2 || names[0] != "interval" || names[1] != "pct" {
		t.Fatalf("RuleNames 期望 [interval pct]，得到 %v", names)
	}
}

func TestTileIntervalOff(t *testing.T) {
	// 8x8 全 Metal，区间 [90,110] 关闭（IntervalOff），无小格规则。
	// 一个像素 200 超出区间，但区间不起作用 → 0 缺陷、FlaggedPixels=0。
	img, reg := uniform(8, 8, 100, Metal)
	img = ttPutPixel(img, 3, 3, 200)

	p := ttMetalParams(TileRule{}, TileRule{})
	p.Materials[Metal].Low = 90
	p.Materials[Metal].High = 110
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Defects) != 0 || rep.FlaggedPixels != 0 {
		t.Fatalf("IntervalOff 期望 0 缺陷/0 标记，得到 %d/%d", len(rep.Defects), rep.FlaggedPixels)
	}
}

func TestTileCountsAndStats(t *testing.T) {
	// 100x100 全 Metal，T=64：宽 n=100/64=1、rem=36>=32 → n=2；高同理 → 2x2 → Tiles=4。
	// 需要起作用的小格规则才报告 Tiles，用 Bright=200（全 100，不偏离，无缺陷）。
	img100, reg100 := uniform(100, 100, 100, Metal)
	p := ttMetalParams(ttAbs(200, 0, 1, 1), TileRule{})
	p.TileSize = 64
	rep, err := Inspect(img100, reg100, p)
	if err != nil {
		t.Fatal(err)
	}
	if rep.PerMaterial[Metal].Tiles != 4 {
		t.Fatalf("100x100/T64 期望 Tiles=4，得到 %d", rep.PerMaterial[Metal].Tiles)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("均匀图期望 0 缺陷，得到 %d", len(rep.Defects))
	}

	// 90x90/T64：宽 n=1、rem=26 < 32 → n=1；高同理 → 1x1 → Tiles=1。
	img90, reg90 := uniform(90, 90, 100, Metal)
	rep, err = Inspect(img90, reg90, p)
	if err != nil {
		t.Fatal(err)
	}
	if rep.PerMaterial[Metal].Tiles != 1 {
		t.Fatalf("90x90/T64 期望 Tiles=1，得到 %d", rep.PerMaterial[Metal].Tiles)
	}

	// TileMeanMin/Max = 各小格自身平均的最小/最大。
	// 128x64，T=64 → x n=2、y n=1 → 2 格：左格全 100（平均 100）、右格全 150（平均 150）。
	imgRamp, regRamp := buildImage(128, 64, func(x, y int) (uint8, Material) {
		if x < 64 {
			return 100, Metal
		}
		return 150, Metal
	})
	pStats := ttMetalParams(ttAbs(200, 200, 1, 1), TileRule{})
	pStats.TileSize = 64
	rep, err = Inspect(imgRamp, regRamp, pStats)
	if err != nil {
		t.Fatal(err)
	}
	if rep.PerMaterial[Metal].Tiles != 2 {
		t.Fatalf("128x64/T64 期望 Tiles=2，得到 %d", rep.PerMaterial[Metal].Tiles)
	}
	if rep.PerMaterial[Metal].TileMeanMin != 100 || rep.PerMaterial[Metal].TileMeanMax != 150 {
		t.Fatalf("期望 TileMeanMin/Max=100/150，得到 %v/%v",
			rep.PerMaterial[Metal].TileMeanMin, rep.PerMaterial[Metal].TileMeanMax)
	}

	// 没有起作用的小格规则（Abs/Pct 关闭）→ Tiles/Min/Max 全 0。
	pNone := ttMetalParams(TileRule{}, TileRule{})
	pNone.TileSize = 64
	rep, err = Inspect(imgRamp, regRamp, pNone)
	if err != nil {
		t.Fatal(err)
	}
	if rep.PerMaterial[Metal].Tiles != 0 || rep.PerMaterial[Metal].TileMeanMin != 0 || rep.PerMaterial[Metal].TileMeanMax != 0 {
		t.Fatalf("无小格规则期望 Tiles/Min/Max=0，得到 %d/%v/%v",
			rep.PerMaterial[Metal].Tiles, rep.PerMaterial[Metal].TileMeanMin, rep.PerMaterial[Metal].TileMeanMax)
	}
}

func TestTileSmoothRamp(t *testing.T) {
	// 128x32 水平斜坡 v=50+x，T=32 → 4 格（每格宽 32、高 32，cnt=1024>=64 用本格平均）。
	// 每格平均 = 50 + 格中心 x；格中心 = 32*tx + 15.5。
	img, reg := buildImage(128, 32, func(x, y int) (uint8, Material) {
		return uint8(50 + x), Metal
	})

	// TileSmooth=false：ref = 本格平均；|v-ref| 最大约 15.5，Abs{0.5,0.5} → 大量标记。
	pNS := ttMetalParams(ttAbs(0.5, 0.5, 1, 1), TileRule{})
	pNS.TileSize = 32
	pNS.TileSmooth = false
	repNS, err := Inspect(img, reg, pNS)
	if err != nil {
		t.Fatal(err)
	}

	// TileSmooth=true：格中心之间双线性插值，斜率与斜坡一致 → 内部像素 ref==v，不标记；
	// 仅第一个中心之前 / 最后一个中心之后被钳位到端点平均，产生标记。
	pS := ttMetalParams(ttAbs(0.5, 0.5, 1, 1), TileRule{})
	pS.TileSize = 32
	pS.TileSmooth = true
	repS, err := Inspect(img, reg, pS)
	if err != nil {
		t.Fatal(err)
	}

	if !(repS.FlaggedPixels < repNS.FlaggedPixels) {
		t.Fatalf("期望 smooth 标记像素严格少于非 smooth：%d >= %d", repS.FlaggedPixels, repNS.FlaggedPixels)
	}

	// 第一个中心 x=15.5，最后一个中心 = 3*32+15.5=111.5。smooth 的缺陷只可能落在两端。
	for _, d := range repS.Defects {
		right := d.BBox.X + d.BBox.W - 1
		if !(d.BBox.X >= 112 || right <= 15) {
			t.Fatalf("smooth 缺陷跨在中间区域：BBox=%v", d.BBox)
		}
	}
}

func TestTileSmallRegionFallback(t *testing.T) {
	// 64x64 全 SR(1)，材质图里一小块 2x5=10 个 Metal(3) 像素（Metal 值 50，其中 1 个 200）。
	// T=64 → 单格；Metal 有效像素仅 10 个 < 64，3x3 邻域合并仍 = 10 < 64 → 退化到整幅 Metal 平均
	// = (9*50 + 200)/10 = 65。Abs.Bright=30 → 判定线 95，200 > 95 → 被检出；50 不被标记。
	img, reg := buildImage(64, 64, func(x, y int) (uint8, Material) {
		if x >= 10 && x < 15 && y >= 10 && y < 12 {
			if x == 10 && y == 10 {
				return 200, Metal
			}
			return 50, Metal
		}
		return 100, SR
	})

	p := ttMetalParams(ttAbs(30, 30, 1, 1), TileRule{})
	p.TileSize = 64
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if rep.PerMaterial[Metal].Tiles < 1 {
		t.Fatalf("Metal 至少含 1 个有小格像素的格，期望 Tiles>=1，得到 %d", rep.PerMaterial[Metal].Tiles)
	}
	// Metal 缺陷恰 1（200 那个像素）。
	metalDefs := 0
	for _, d := range rep.Defects {
		if d.Material == Metal {
			metalDefs++
			if d.Polarity != Bright || d.Area != 1 {
				t.Fatalf("Metal 缺陷期望 Bright/1，得到 %v/%d", d.Polarity, d.Area)
			}
		}
	}
	if metalDefs != 1 {
		t.Fatalf("期望恰 1 个 Metal 缺陷，得到 %d", metalDefs)
	}
	// SR 全一致 → 无 SR 缺陷。
	for _, d := range rep.Defects {
		if d.Material == SR {
			t.Fatalf("SR 均匀不应有缺陷，得到 BBox=%v", d.BBox)
		}
	}
}

func TestTileNoMaterialPixels(t *testing.T) {
	// 材质图全是 SR，没有启用的 Metal 像素；Metal 开启 Abs 规则也不能报错、无缺陷。
	img, reg := uniform(32, 32, 100, SR)

	p := ttMetalParams(ttAbs(30, 30, 1, 1), TileRule{})
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatalf("无启用材质像素时不应报错：%v", err)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("期望 0 缺陷，得到 %d", len(rep.Defects))
	}
	if rep.PerMaterial[Metal].Tiles != 0 {
		t.Fatalf("Metal 无有效像素期望 Tiles=0，得到 %d", rep.PerMaterial[Metal].Tiles)
	}
}

func TestTileValidate(t *testing.T) {
	// TileSize：15/4097/-1 非法；0/16/4096 合法。
	for _, bad := range []int{15, 4097, -1} {
		p := DefaultParams()
		p.TileSize = bad
		if err := p.Validate(); err == nil {
			t.Fatalf("TileSize=%d 期望报错", bad)
		}
	}
	for _, ok := range []int{0, 16, 4096} {
		p := DefaultParams()
		p.TileSize = ok
		if err := p.Validate(); err != nil {
			t.Fatalf("TileSize=%d 期望通过，得到 %v", ok, err)
		}
	}

	// 启用材质 + 启用 Abs 规则：Bright 越界 / NaN / +Inf 报错。
	badAbs := []TileRule{
		{Enabled: true, Bright: -1, Dark: 0, BrightSize: 1, DarkSize: 1},
		{Enabled: true, Bright: 256, Dark: 0, BrightSize: 1, DarkSize: 1},
		{Enabled: true, Bright: math.NaN(), Dark: 0, BrightSize: 1, DarkSize: 1},
		{Enabled: true, Bright: math.Inf(1), Dark: 0, BrightSize: 1, DarkSize: 1},
	}
	for _, r := range badAbs {
		p := DefaultParams()
		p.Materials[Metal].Abs = r
		p.Materials[Metal].Pct = TileRule{}
		err := p.Validate()
		if err == nil {
			t.Fatalf("启用 Abs Bright=%v 期望报错", r.Bright)
		}
		if !strings.Contains(err.Error(), "metal") {
			t.Fatalf("错误文本应含材质名 metal，得到 %q", err.Error())
		}
	}

	// 启用 Pct 规则 Dark=1001 报错。
	p := DefaultParams()
	p.Materials[Metal].Pct = TileRule{Enabled: true, Bright: 0, Dark: 1001, BrightSize: 1, DarkSize: 1}
	p.Materials[Metal].Abs = TileRule{}
	if err := p.Validate(); err == nil {
		t.Fatal("Pct Dark=1001 期望报错")
	} else if !strings.Contains(err.Error(), "metal") {
		t.Fatalf("错误文本应含 metal，得到 %q", err.Error())
	}

	// BrightSize=0 / DarkSize=0 报错。
	p = DefaultParams()
	p.Materials[Metal].Abs = TileRule{Enabled: true, Bright: 30, Dark: 30, BrightSize: 0, DarkSize: 1}
	if err := p.Validate(); err == nil {
		t.Fatal("BrightSize=0 期望报错")
	}
	p = DefaultParams()
	p.Materials[Metal].Abs = TileRule{Enabled: true, Bright: 30, Dark: 30, BrightSize: 1, DarkSize: 0}
	if err := p.Validate(); err == nil {
		t.Fatal("DarkSize=0 期望报错")
	}

	// 关闭的规则里的非法值不报错。
	p = DefaultParams()
	p.Materials[Metal].Abs = TileRule{Enabled: false, Bright: -5, Dark: 999, BrightSize: 0, DarkSize: 0}
	p.Materials[Metal].Pct = TileRule{}
	if err := p.Validate(); err != nil {
		t.Fatalf("关闭规则的非法值期望通过，得到 %v", err)
	}

	// 关闭的材质里的非法值不报错。
	p = DefaultParams()
	p.Materials[Metal].Enabled = false
	p.Materials[Metal].Abs = TileRule{Enabled: true, Bright: -5, Dark: 999, BrightSize: 0, DarkSize: 0}
	if err := p.Validate(); err != nil {
		t.Fatalf("关闭材质的非法值期望通过，得到 %v", err)
	}
}

func TestTileLearnKeepsRules(t *testing.T) {
	// base 带非默认 Abs/Pct/IntervalOff/TileSize/TileSmooth。Learn 只改 Low/High。
	img, reg := uniform(64, 64, 100, Metal)
	base := DefaultParams()
	base.TileSize = 32
	base.TileSmooth = true
	base.Materials[Metal].IntervalOff = true
	base.Materials[Metal].Abs = ttAbs(30, 20, 3, 3)
	base.Materials[Metal].Pct = ttPct(15, 25, 2, 4)
	base.Materials[SR].IntervalOff = true
	base.Materials[SR].Abs = ttAbs(11, 12, 5, 6)

	out, err := Learn(img, reg, 3, base)
	if err != nil {
		t.Fatal(err)
	}
	for i := range out.Materials {
		if !reflect.DeepEqual(out.Materials[i].Abs, base.Materials[i].Abs) {
			t.Fatalf("材质 %d Abs 被 Learn 改动：%v vs %v", i, out.Materials[i].Abs, base.Materials[i].Abs)
		}
		if !reflect.DeepEqual(out.Materials[i].Pct, base.Materials[i].Pct) {
			t.Fatalf("材质 %d Pct 被 Learn 改动：%v vs %v", i, out.Materials[i].Pct, base.Materials[i].Pct)
		}
		if out.Materials[i].IntervalOff != base.Materials[i].IntervalOff {
			t.Fatalf("材质 %d IntervalOff 被改动", i)
		}
	}
	if out.TileSize != base.TileSize {
		t.Fatalf("TileSize 被改动：%d vs %d", out.TileSize, base.TileSize)
	}
	if out.TileSmooth != base.TileSmooth {
		t.Fatalf("TileSmooth 被改动")
	}
	// 均匀图 std=0 → Metal 学到 Low=High=100（base 是 0/255，确实变了）。
	if out.Materials[Metal].Low != 100 || out.Materials[Metal].High != 100 {
		t.Fatalf("期望 Metal 区间 [100,100]，得到 [%v,%v]", out.Materials[Metal].Low, out.Materials[Metal].High)
	}
}

func TestTileProbeCounts(t *testing.T) {
	// 16x16 单格全 100，3x3 块 GV200，Abs.Bright=30，区间关闭。
	// 本格平均 = (253*100 + 9*200)/256 = 27100/256 = 105.859375；判定线 135.86 → 200 被标记 9 像素。
	// ProbeRect 的 Bright 不做 Size 过滤 → 等于 9。
	img, reg := uniform(16, 16, 100, Metal)
	img = ttBlock(img, 0, 0, 3, 3, 200)

	p := ttMetalParams(ttAbs(30, 0, 5, 5), TileRule{})
	p.TileSize = 16
	rep, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	// BrightSize=5 会把 9 像素块（9>=5）接受，这里只为构造，不依赖缺陷数。
	pr, err := ProbeRect(img, reg, p, rep, Rect{X: 0, Y: 0, W: 16, H: 16})
	if err != nil {
		t.Fatal(err)
	}
	if pr.PerMaterial[Metal].Bright != 9 {
		t.Fatalf("期望 Metal Bright=9，得到 %d", pr.PerMaterial[Metal].Bright)
	}
	if pr.PerMaterial[Metal].Dark != 0 {
		t.Fatalf("期望 Metal Dark=0，得到 %d", pr.PerMaterial[Metal].Dark)
	}
}

func TestTileDeterministic(t *testing.T) {
	// 用一条含多格、多缺陷的图，连续两次 Inspect 结果应完全相等。
	img, reg := buildImage(96, 64, func(x, y int) (uint8, Material) {
		if (x*5+y*3)%13 == 0 {
			return 200, Metal
		}
		return 100, Metal
	})
	p := ttMetalParams(ttAbs(30, 30, 2, 2), ttPct(20, 20, 2, 2))
	p.TileSize = 32
	p.TileSmooth = true

	rep1, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	rep2, err := Inspect(img, reg, p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep1, rep2) {
		t.Fatal("两次 Inspect 结果不一致")
	}
}

func TestTileTallImageTiles(t *testing.T) {
	// 规格里 T=40 与 T=200 的“界线两侧各偏离整幅平均超过 20”对同一图无法同时成立：
	// 40 对齐的边界 y=120 使整幅平均恰为 120，暗侧 100 落在判定线 100（严格不等号）不被标记。
	// 因此这里分别用两张各自自洽的图。

	// (a) T=40，无缺陷：边界落在小格行边界 y=120，每格内亮度一致，本格平均即本格值。
	// 行格：y0..39=100, 40..79=100, 80..119=100, 120..159=150, 160..199=150。每格 ref=本格值 → 不偏离。
	imgA, regA := buildImage(40, 200, func(x, y int) (uint8, Material) {
		if y < 120 {
			return 100, Metal
		}
		return 150, Metal
	})
	pA := ttMetalParams(ttAbs(20, 20, 1, 1), TileRule{})
	pA.TileSize = 40
	rep, err := Inspect(imgA, regA, pA)
	if err != nil {
		t.Fatal(err)
	}
	if rep.PerMaterial[Metal].Tiles != 5 {
		t.Fatalf("T=40 期望 5 行格 Tiles=5，得到 %d", rep.PerMaterial[Metal].Tiles)
	}
	if len(rep.Defects) != 0 {
		t.Fatalf("T=40 期望 0 缺陷，得到 %d", len(rep.Defects))
	}

	// (b) T=200，两个大块：单格（整幅），边界 y=100 使平均落在两侧判定线之间。
	// 整幅平均 = (100*40*100 + 100*40*150)/8000 = (400000+600000)/8000 = 125。
	// 暗侧上半 100 < 125-20=105 → 检出（一个暗块）；亮侧下半 150 > 125+20=145 → 检出（一个亮块）。
	imgB, regB := buildImage(40, 200, func(x, y int) (uint8, Material) {
		if y < 100 {
			return 100, Metal
		}
		return 150, Metal
	})
	pB := ttMetalParams(ttAbs(20, 20, 1, 1), TileRule{})
	pB.TileSize = 200
	rep, err = Inspect(imgB, regB, pB)
	if err != nil {
		t.Fatal(err)
	}
	if rep.PerMaterial[Metal].Tiles != 1 {
		t.Fatalf("T=200 期望 Tiles=1，得到 %d", rep.PerMaterial[Metal].Tiles)
	}
	if len(rep.Defects) != 2 {
		t.Fatalf("T=200 期望 2 个缺陷，得到 %d", len(rep.Defects))
	}
	var brightDefects, darkDefects int
	for _, d := range rep.Defects {
		switch d.Polarity {
		case Bright:
			brightDefects++
			// 下半部分：BBox 起点应在 y>=100。
			if d.BBox.Y < 100 {
				t.Fatalf("亮缺陷应在下半：BBox=%v", d.BBox)
			}
		case Dark:
			darkDefects++
			// 上半部分：BBox 应完全在 y<100。
			if d.BBox.Y+d.BBox.H-1 >= 100 {
				t.Fatalf("暗缺陷应在上半：BBox=%v", d.BBox)
			}
		}
	}
	if brightDefects != 1 || darkDefects != 1 {
		t.Fatalf("期望 1 亮 1 暗缺陷，得到 %d/%d", brightDefects, darkDefects)
	}
}
