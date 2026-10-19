// Package inspect 实现一个自包含的缺陷检查算法。
//
// 该包只依赖标准库，只根据测试图像与产品的设计材质图进行判断，
// 绝不接收或计算任何来自仿真的数据（金标准图、缺陷笔画、绘制场景等）。
package inspect

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
)

// Material 表示设计材质图中的材质索引，数值与应用的材质图一致。
type Material uint8

// 支持的材质常量。
const (
	Base  Material = 0 // 基底
	SR    Material = 1 // 感光胶
	Open  Material = 2 // 开口
	Metal Material = 3 // 金属
)

// NumMaterials 是材质总数。
const NumMaterials = 4

// MaxEdgeMargin 是 EdgeMargin 允许的最大值。
const MaxEdgeMargin = 64

// String 返回材质的中文/英文短名。
func (m Material) String() string {
	switch m {
	case Base:
		return "base"
	case SR:
		return "sr"
	case Open:
		return "open"
	case Metal:
		return "metal"
	default:
		return fmt.Sprintf("material(%d)", uint8(m))
	}
}

// Image 是 8 位灰度图，行优先存储。
type Image struct {
	Width, Height int
	Pix           []uint8
}

// Regions 是逐像素的设计材质索引图，取值 0..3。
type Regions struct {
	Width, Height int
	Material      []uint8
}

// Rect 是矩形区域。
type Rect struct {
	X, Y, W, H int
}

// MaterialParams 描述单个材质的检查参数。
type MaterialParams struct {
	Enabled   bool    // 关闭的材质像素永不被标记
	Low, High float64 // 可接受的 GV 区间，闭区间；0 <= Low <= High <= 255
	Size      int     // 连续（8 连通）被标记像素数下限：一个连通块至少有 Size 个被标记像素才算缺陷；不膨胀、不合并、不取平均
	MinArea   int     // 连通块被标记像素数的额外下限，实际阈值为 max(Size, MinArea)；默认 1，即只由 Size 决定
	// LocalBackground 为 true 时该材质按局部背景判断，仅当 Params.BackgroundRadius > 0 时生效；
	// DefaultParams 为 true。
	LocalBackground bool
}

// Params 是全部材质的检查参数。
type Params struct {
	Materials [NumMaterials]MaterialParams
	// EdgeMargin 为 0..MaxEdgeMargin：与不同材质的像素的切比雪夫距离
	// 不超过该值的像素会被忽略（统计、Learn、标记、探测均不涉及）；0 表示关闭。
	EdgeMargin int
	// BackgroundRadius 为 0..MaxBackgroundRadius：0 关闭，即全局区间；
	// > 0 时 LocalBackground 为 true 的材质，其每个有效像素与"同材质有效邻域中位数"
	// （半径 r 的 (2r+1)^2 窗口）比较，区间作用于扣除局部背景后的值；
	// LocalBackground 为 false 的材质仍使用原始全局区间。
	BackgroundRadius int
}

// Polarity 表示缺陷极性。
type Polarity uint8

// 极性常量。
const (
	Bright Polarity = iota // 过亮
	Dark                   // 过暗
)

// Defect 描述一个被报告的缺陷分组。
type Defect struct {
	Material             Material
	Polarity             Polarity
	Area                 int // 连通块中被标记的像素数
	BBox                 Rect
	CentroidX, CentroidY float64
	PeakGV               uint8   // 区间外最远的被标记像素的 GV
	Excess               float64 // 该像素超出区间的距离（亮：v-High，暗：Low-v）
	MeanGV               float64 // 被标记像素的平均 GV（仅用于显示）
}

// MaterialStats 是单个材质的统计信息。
// 在该材质启用局部背景（BackgroundRadius > 0 且 LocalBackground 为 true）时，
// Std/Min/Max 描述的是局部归一化值 f，而 Mean 仍是原始灰度均值。
type MaterialStats struct {
	Pixels       int
	Mean, Std    float64
	Min, Max     uint8
	Bright, Dark int // MinArea 过滤前的被标记像素数（材质关闭时为 0）
}

// Report 是检查结果。
type Report struct {
	Defects       []Defect // 被报告的连通块（Area >= max(Size, MinArea)），按 Area 降序，同面积按 BBox.Y 再按 BBox.X
	Filtered      int      // 因 Area < max(Size, MinArea) 被丢弃的连通块数
	FlaggedPixels int      // 所有启用材质的被标记像素总数
	EdgeIgnored   int      // 因 EdgeMargin 被忽略的像素数（所有材质合计）
	PerMaterial   [NumMaterials]MaterialStats
}

// DefaultParams 返回默认参数：Metal 与 SR 启用，Open 与 Base 关闭；
// 每个材质 Low=0、High=255、Size=3、MinArea=1。
func DefaultParams() Params {
	var p Params
	for i := 0; i < NumMaterials; i++ {
		p.Materials[i] = MaterialParams{
			Enabled:         false,
			Low:             0,
			High:            255,
			Size:            3,
			MinArea:         1,
			LocalBackground: true,
		}
	}
	p.Materials[Metal].Enabled = true
	p.Materials[SR].Enabled = true
	return p
}

// Validate 校验参数，返回描述性错误（包含材质名）。
func (p Params) Validate() error {
	if p.EdgeMargin < 0 || p.EdgeMargin > MaxEdgeMargin {
		return fmt.Errorf("EdgeMargin=%d 非法：必须在 0..%d", p.EdgeMargin, MaxEdgeMargin)
	}
	if p.BackgroundRadius < 0 || p.BackgroundRadius > MaxBackgroundRadius {
		return fmt.Errorf("BackgroundRadius=%d 非法：必须在 0..%d", p.BackgroundRadius, MaxBackgroundRadius)
	}
	for i := 0; i < NumMaterials; i++ {
		mp := p.Materials[i]
		if !mp.Enabled {
			continue
		}
		m := Material(i)
		if math.IsNaN(mp.Low) || math.IsNaN(mp.High) {
			return fmt.Errorf("材质 %s：Low/High 不能为 NaN", m)
		}
		if mp.Low < 0 || mp.Low > 255 {
			return fmt.Errorf("材质 %s：Low=%v 超出 [0,255]", m, mp.Low)
		}
		if mp.High < 0 || mp.High > 255 {
			return fmt.Errorf("材质 %s：High=%v 超出 [0,255]", m, mp.High)
		}
		if mp.Low > mp.High {
			return fmt.Errorf("材质 %s：Low=%v 大于 High=%v", m, mp.Low, mp.High)
		}
		if mp.Size < 1 {
			return fmt.Errorf("材质 %s：Size=%d 必须 >= 1", m, mp.Size)
		}
		if mp.MinArea < 1 {
			return fmt.Errorf("材质 %s：MinArea=%d 必须 >= 1", m, mp.MinArea)
		}
	}
	return nil
}

// backgroundUse 返回每个材质是否使用局部背景，以及是否有任何材质使用。
// BackgroundRadius <= 0 时全部为 false。
func backgroundUse(p Params) (use [NumMaterials]bool, any bool) {
	if p.BackgroundRadius <= 0 {
		return use, false
	}
	for i := 0; i < NumMaterials; i++ {
		use[i] = p.Materials[i].LocalBackground
		if use[i] {
			any = true
		}
	}
	return use, any
}

// validateInputs 校验图像与材质图尺寸，以及材质取值。
func validateInputs(img Image, reg Regions) error {
	if img.Width <= 0 || img.Height <= 0 {
		return fmt.Errorf("图像尺寸非法：%dx%d", img.Width, img.Height)
	}
	if len(img.Pix) != img.Width*img.Height {
		return fmt.Errorf("图像像素数 %d 与尺寸 %dx%d 不匹配", len(img.Pix), img.Width, img.Height)
	}
	if reg.Width != img.Width || reg.Height != img.Height {
		return fmt.Errorf("材质图尺寸 %dx%d 与图像 %dx%d 不匹配", reg.Width, reg.Height, img.Width, img.Height)
	}
	if len(reg.Material) != reg.Width*reg.Height {
		return fmt.Errorf("材质图元素数 %d 与尺寸 %dx%d 不匹配", len(reg.Material), reg.Width, reg.Height)
	}
	for i, v := range reg.Material {
		if v > uint8(Metal) {
			return fmt.Errorf("像素 %d 的材质值 %d 非法（必须 <= %d）", i, v, uint8(Metal))
		}
	}
	return nil
}

// statsAcc 是统计累加器。
type statsAcc struct {
	count int
	sum   float64
	sumSq float64
	min   uint8
	max   uint8
}

// Inspect 根据图像与设计材质图执行检查。
func Inspect(img Image, reg Regions, p Params) (Report, error) {
	var rep Report
	if err := validateInputs(img, reg); err != nil {
		return rep, err
	}
	if err := p.Validate(); err != nil {
		return rep, err
	}

	// 边缘排除掩码。
	excl := edgeExclusion(reg, p.EdgeMargin)
	rep.EdgeIgnored = countOnes(excl)

	// 局部背景模式（仅对启用的材质）。
	var lb *localBackground
	if use, any := backgroundUse(p); any {
		lb = computeBackground(img, reg, excl, p.BackgroundRadius, use)
	}

	// 1. 统计遍历（并行）。局部背景只覆盖启用了它的材质，其余材质始终用全局统计。
	accs := parallelStats(img, reg, excl)
	for i := 0; i < NumMaterials; i++ {
		ms := &rep.PerMaterial[i]
		if lb != nil && lb.active(i) {
			ms.Pixels = lb.pixels(i)
			ms.Mean = lb.mean(i)
			ms.Std = lb.std(i)
			ms.Min, ms.Max = lb.minmax(i)
			continue
		}
		a := accs[i]
		ms.Pixels = a.count
		if a.count > 0 {
			mean := a.sum / float64(a.count)
			variance := a.sumSq/float64(a.count) - mean*mean
			if variance < 0 {
				variance = 0
			}
			ms.Mean = mean
			ms.Std = math.Sqrt(variance)
			ms.Min = a.min
			ms.Max = a.max
		}
	}

	// 2. 逐材质、逐极性处理。
	for mi := 0; mi < NumMaterials; mi++ {
		mp := p.Materials[mi]
		if !mp.Enabled {
			continue
		}
		m := Material(mi)
		var mlb *localBackground
		if lb != nil && lb.active(mi) {
			mlb = lb
		}
		for _, pol := range []Polarity{Bright, Dark} {
			mask := buildMask(img, reg, m, pol, mp.Low, mp.High, excl, mlb)
			flagged := countOnes(mask)
			if flagged == 0 {
				continue
			}
			if pol == Bright {
				rep.PerMaterial[mi].Bright = flagged
			} else {
				rep.PerMaterial[mi].Dark = flagged
			}
			rep.FlaggedPixels += flagged

			defects, filtered := analyzeMask(img, mask, m, pol, mp, mlb)
			rep.Defects = append(rep.Defects, defects...)
			rep.Filtered += filtered
		}
	}

	sort.SliceStable(rep.Defects, func(i, j int) bool {
		a, b := rep.Defects[i], rep.Defects[j]
		if a.Area != b.Area {
			return a.Area > b.Area
		}
		if a.BBox.Y != b.BBox.Y {
			return a.BBox.Y < b.BBox.Y
		}
		return a.BBox.X < b.BBox.X
	})

	return rep, nil
}

// analyzeMask 对一个材质、一种极性的标记掩码做 8 连通分组与统计。
// 不膨胀、不合并：每个连通块独立判断，阈值 need = max(Size, MinArea)。
func analyzeMask(img Image, mask []uint8, m Material, pol Polarity, mp MaterialParams, lb *localBackground) (defects []Defect, filtered int) {
	w, h := img.Width, img.Height

	need := mp.Size
	if mp.MinArea > need {
		need = mp.MinArea
	}

	runs, rowStart := extractRuns(mask, w, h)
	id, count := labelRuns(runs, rowStart)

	// 每个连通块的累加器。
	type groupAcc struct {
		area                   int
		minX, minY, maxX, maxY int
		sumX, sumY, sumGV      float64
		peakGV                 uint8
		peakExcess             float64
		hasPeak                bool
	}
	groups := make([]groupAcc, count)
	for i := range groups {
		groups[i].minX = math.MaxInt32
		groups[i].minY = math.MaxInt32
		groups[i].maxX = -1
		groups[i].maxY = -1
	}

	// 单趟遍历行程，累加到所属连通块。
	for y := 0; y < h; y++ {
		rowBase := y * w
		r0, r1 := int(rowStart[y]), int(rowStart[y+1])
		for ri := r0; ri < r1; ri++ {
			r := runs[ri]
			ga := &groups[id[ri]]
			for x := int(r.x0); x <= int(r.x1); x++ {
				v := float64(img.Pix[rowBase+x])
				fv := v
				if lb != nil {
					fv = lb.value(rowBase+x, uint8(m))
				}
				ga.area++
				if x < ga.minX {
					ga.minX = x
				}
				if x > ga.maxX {
					ga.maxX = x
				}
				if y < ga.minY {
					ga.minY = y
				}
				if y > ga.maxY {
					ga.maxY = y
				}
				ga.sumX += float64(x)
				ga.sumY += float64(y)
				ga.sumGV += v
				var excess float64
				if pol == Bright {
					excess = fv - mp.High
				} else {
					excess = mp.Low - fv
				}
				if !ga.hasPeak || excess > ga.peakExcess {
					ga.hasPeak = true
					ga.peakExcess = excess
					ga.peakGV = img.Pix[rowBase+x]
				}
			}
		}
	}

	// 生成缺陷。
	for gi := range groups {
		ga := &groups[gi]
		if ga.area == 0 {
			continue
		}
		if ga.area < need {
			filtered++
			continue
		}
		defects = append(defects, Defect{
			Material:  m,
			Polarity:  pol,
			Area:      ga.area,
			BBox:      Rect{X: ga.minX, Y: ga.minY, W: ga.maxX - ga.minX + 1, H: ga.maxY - ga.minY + 1},
			CentroidX: ga.sumX / float64(ga.area),
			CentroidY: ga.sumY / float64(ga.area),
			PeakGV:    ga.peakGV,
			Excess:    ga.peakExcess,
			MeanGV:    ga.sumGV / float64(ga.area),
		})
	}
	return defects, filtered
}

// parallelStats 并行统计每个材质的像素数、和、平方和、最小/最大值。
// excl 非 nil 时，excl[idx] != 0 的像素被完全跳过。
func parallelStats(img Image, reg Regions, excl []uint8) [NumMaterials]statsAcc {
	var accs [NumMaterials]statsAcc
	for i := range accs {
		accs[i].min = 255
		accs[i].max = 0
	}
	w, h := img.Width, img.Height
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > h {
		workers = h
	}
	if workers < 1 {
		// 高度为 0 已在 validateInputs 中拒绝，这里兜底。
		return accs
	}
	partials := make([][NumMaterials]statsAcc, workers)
	for i := range partials {
		for j := range partials[i] {
			partials[i][j].min = 255
			partials[i][j].max = 0
		}
	}
	var wg sync.WaitGroup
	rowsPer := (h + workers - 1) / workers
	for wi := 0; wi < workers; wi++ {
		y0 := wi * rowsPer
		y1 := y0 + rowsPer
		if y1 > h {
			y1 = h
		}
		if y0 >= y1 {
			continue
		}
		wg.Add(1)
		go func(wi, y0, y1 int) {
			defer wg.Done()
			local := &partials[wi]
			for y := y0; y < y1; y++ {
				base := y * w
				for x := 0; x < w; x++ {
					idx := base + x
					if excl != nil && excl[idx] != 0 {
						continue
					}
					m := reg.Material[idx]
					v := img.Pix[idx]
					a := &local[m]
					a.count++
					fv := float64(v)
					a.sum += fv
					a.sumSq += fv * fv
					if v < a.min {
						a.min = v
					}
					if v > a.max {
						a.max = v
					}
				}
			}
		}(wi, y0, y1)
	}
	wg.Wait()
	// 按 worker 顺序合并，保证确定性。
	for wi := 0; wi < workers; wi++ {
		for mi := 0; mi < NumMaterials; mi++ {
			src := partials[wi][mi]
			if src.count == 0 {
				continue
			}
			dst := &accs[mi]
			if dst.count == 0 {
				dst.min = src.min
				dst.max = src.max
			} else {
				if src.min < dst.min {
					dst.min = src.min
				}
				if src.max > dst.max {
					dst.max = src.max
				}
			}
			dst.count += src.count
			dst.sum += src.sum
			dst.sumSq += src.sumSq
		}
	}
	return accs
}

// buildMask 构建某材质某极性的标记掩码（并行）。
// excl 非 nil 时，excl[idx] != 0 的像素被完全跳过。
func buildMask(img Image, reg Regions, m Material, pol Polarity, low, high float64, excl []uint8, lb *localBackground) []uint8 {
	w, h := img.Width, img.Height
	mask := make([]uint8, w*h)
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > h {
		workers = h
	}
	if workers < 1 {
		return mask
	}
	rowsPer := (h + workers - 1) / workers
	var wg sync.WaitGroup
	for wi := 0; wi < workers; wi++ {
		y0 := wi * rowsPer
		y1 := y0 + rowsPer
		if y1 > h {
			y1 = h
		}
		if y0 >= y1 {
			continue
		}
		wg.Add(1)
		go func(y0, y1 int) {
			defer wg.Done()
			for y := y0; y < y1; y++ {
				base := y * w
				for x := 0; x < w; x++ {
					idx := base + x
					if excl != nil && excl[idx] != 0 {
						continue
					}
					if reg.Material[idx] != uint8(m) {
						continue
					}
					v := float64(img.Pix[idx])
					if lb != nil {
						v = lb.value(idx, reg.Material[idx])
					}
					if pol == Bright {
						if v > high {
							mask[idx] = 1
						}
					} else {
						if v < low {
							mask[idx] = 1
						}
					}
				}
			}
		}(y0, y1)
	}
	wg.Wait()
	return mask
}

// countOnes 统计掩码中 1 的个数。
func countOnes(mask []uint8) int {
	c := 0
	for _, v := range mask {
		if v != 0 {
			c++
		}
	}
	return c
}

// edgeExclusion 返回边缘排除掩码：1 表示该像素与不同材质的像素的切比雪夫距离
// 不超过 r（在图像边界处裁剪；边界本身不会排除任何像素）。
// r <= 0 时返回 nil。
func edgeExclusion(reg Regions, r int) []uint8 {
	if r <= 0 {
		return nil
	}
	w, h := reg.Width, reg.Height
	if w <= 0 || h <= 0 {
		return nil
	}
	// 边界掩码：像素在图像内存在一个 8 邻域的不同材质像素。
	boundary := make([]uint8, w*h)
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > h {
		workers = h
	}
	if workers < 1 {
		return nil
	}
	rowsPer := (h + workers - 1) / workers
	var wg sync.WaitGroup
	for wi := 0; wi < workers; wi++ {
		y0 := wi * rowsPer
		y1 := y0 + rowsPer
		if y1 > h {
			y1 = h
		}
		if y0 >= y1 {
			continue
		}
		wg.Add(1)
		go func(y0, y1 int) {
			defer wg.Done()
			for y := y0; y < y1; y++ {
				base := y * w
				for x := 0; x < w; x++ {
					idx := base + x
					m := reg.Material[idx]
					diff := false
					for dy := -1; dy <= 1 && !diff; dy++ {
						ny := y + dy
						if ny < 0 || ny >= h {
							continue
						}
						nbase := ny * w
						for dx := -1; dx <= 1; dx++ {
							if dy == 0 && dx == 0 {
								continue
							}
							nx := x + dx
							if nx < 0 || nx >= w {
								continue
							}
							if reg.Material[nbase+nx] != m {
								diff = true
								break
							}
						}
					}
					if diff {
						boundary[idx] = 1
					}
				}
			}
		}(y0, y1)
	}
	wg.Wait()
	if r == 1 {
		return boundary
	}
	return grow(boundary, w, h, r-1)
}

// Learn 根据良好样本图像学习每个启用材质的区间。
func Learn(img Image, reg Regions, k float64, base Params) (Params, error) {
	if math.IsNaN(k) || math.IsInf(k, 0) || k < 0 {
		return Params{}, fmt.Errorf("k=%v 非法：必须为有限且 >= 0 的数", k)
	}
	if err := validateInputs(img, reg); err != nil {
		return Params{}, err
	}
	if base.EdgeMargin < 0 || base.EdgeMargin > MaxEdgeMargin {
		return Params{}, fmt.Errorf("EdgeMargin=%d 非法：必须在 0..%d", base.EdgeMargin, MaxEdgeMargin)
	}
	if base.BackgroundRadius < 0 || base.BackgroundRadius > MaxBackgroundRadius {
		return Params{}, fmt.Errorf("BackgroundRadius=%d 非法：必须在 0..%d", base.BackgroundRadius, MaxBackgroundRadius)
	}
	out := base
	excl := edgeExclusion(reg, base.EdgeMargin)
	var lb *localBackground
	if use, any := backgroundUse(base); any {
		lb = computeBackground(img, reg, excl, base.BackgroundRadius, use)
	}
	accs := parallelStats(img, reg, excl)
	for i := 0; i < NumMaterials; i++ {
		if !base.Materials[i].Enabled {
			continue
		}
		var mean, std float64
		if lb != nil && lb.active(i) {
			mean = lb.mean(i)
			std = lb.std(i)
		} else {
			a := accs[i]
			if a.count == 0 {
				continue
			}
			mean = a.sum / float64(a.count)
			variance := a.sumSq/float64(a.count) - mean*mean
			if variance < 0 {
				variance = 0
			}
			std = math.Sqrt(variance)
		}
		low := math.Floor(mean - k*std)
		high := math.Ceil(mean + k*std)
		out.Materials[i].Low = clampFloat(low, 0, 255)
		out.Materials[i].High = clampFloat(high, 0, 255)
	}
	return out, nil
}

// clampFloat 将 v 限制在 [lo, hi]。
func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ProbeStats 是矩形区域内的统计信息。
type ProbeStats struct {
	Pixels       int
	Mean         float64
	Min, Max     uint8
	Bright, Dark int // 区间外的像素数（仅启用材质）
}

// ProbeReport 是矩形探测结果。
type ProbeReport struct {
	Rect        Rect
	PerMaterial [NumMaterials]ProbeStats
	Defects     []Defect // BBox 与（裁剪后的）矩形相交的 rep.Defects
	Ignored     int      // 矩形内因 EdgeMargin 被忽略的像素数
}

// ProbeRect 在给定矩形内统计，并返回与该矩形相交的缺陷。
func ProbeRect(img Image, reg Regions, p Params, rep Report, r Rect) (ProbeReport, error) {
	var out ProbeReport
	if err := validateInputs(img, reg); err != nil {
		return out, err
	}
	if err := p.Validate(); err != nil {
		return out, err
	}
	w, h := img.Width, img.Height
	// 裁剪矩形。
	x0 := r.X
	y0 := r.Y
	x1 := r.X + r.W
	y1 := r.Y + r.H
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > w {
		x1 = w
	}
	if y1 > h {
		y1 = h
	}
	if x0 >= x1 || y0 >= y1 {
		return out, nil
	}
	out.Rect = Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}

	excl := edgeExclusion(reg, p.EdgeMargin)
	var lb *localBackground
	if use, any := backgroundUse(p); any {
		lb = computeBackground(img, reg, excl, p.BackgroundRadius, use)
	}

	for i := range out.PerMaterial {
		out.PerMaterial[i].Min = 255
		out.PerMaterial[i].Max = 0
	}
	for y := y0; y < y1; y++ {
		base := y * w
		for x := x0; x < x1; x++ {
			idx := base + x
			if excl != nil && excl[idx] != 0 {
				out.Ignored++
				continue
			}
			m := reg.Material[idx]
			v := img.Pix[idx]
			ps := &out.PerMaterial[m]
			ps.Pixels++
			ps.Mean += float64(v)
			if v < ps.Min {
				ps.Min = v
			}
			if v > ps.Max {
				ps.Max = v
			}
			mp := p.Materials[m]
			if mp.Enabled {
				fv := float64(v)
				if lb != nil && lb.active(int(m)) {
					fv = lb.value(idx, m)
				}
				if fv > mp.High {
					ps.Bright++
				} else if fv < mp.Low {
					ps.Dark++
				}
			}
		}
	}
	for i := range out.PerMaterial {
		ps := &out.PerMaterial[i]
		if ps.Pixels > 0 {
			ps.Mean /= float64(ps.Pixels)
		} else {
			ps.Min = 0
			ps.Max = 0
		}
	}

	// 与矩形相交的缺陷。
	for _, d := range rep.Defects {
		if rectsIntersect(d.BBox, out.Rect) {
			out.Defects = append(out.Defects, d)
		}
	}
	return out, nil
}

// rectsIntersect 判断两个矩形是否相交（半开区间）。
func rectsIntersect(a, b Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}
