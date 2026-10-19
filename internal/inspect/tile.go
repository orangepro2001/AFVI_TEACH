package inspect

import (
	"runtime"
	"sync"
)

// 小格平均（见 docs/INSPECT_RULES.md）：图像切成 TileSize x TileSize 的小格，
// 每个小格、每个材质各有一个"本格平均 GV"，作为 Offset 1 / Offset 2 规则的参考值。

// tileMinPixels 是小格有自己可靠平均所需的最少有效像素数；不足时退化到 3x3 邻域合并，再退化到整幅平均。
const tileMinPixels = 64

// tileAxis 返回某方向上的小格起点：长度 n 的轴按 t 切分，起点数组长度为小格数 + 1，最后一项是 n。
// 尾巴不足半格（rem < t/2）并入最后一格，否则自成一格。
func tileAxis(n, t int) []int {
	cnt := n / t
	rem := n - cnt*t
	if cnt == 0 || rem >= t/2 {
		cnt++
	}
	starts := make([]int, cnt+1)
	for i := 0; i < cnt; i++ {
		starts[i] = i * t
	}
	starts[cnt] = n
	return starts
}

// tileIndexTable 返回每个坐标所属的小格号。
func tileIndexTable(starts []int) []int32 {
	n := starts[len(starts)-1]
	out := make([]int32, n)
	for i := 0; i+1 < len(starts); i++ {
		for x := starts[i]; x < starts[i+1]; x++ {
			out[x] = int32(i)
		}
	}
	return out
}

// tileInterp 是一个方向上的双线性插值表：坐标 c 的参考值由小格 i0、i1 按权重 w 混合。
type tileInterp struct {
	i0, i1 []int32
	w      []float64
}

// newTileInterp 按小格中心 (start+end-1)/2 建插值表；在第一个中心之前/最后一个中心之后取端点，不外推。
func newTileInterp(starts []int) tileInterp {
	n := starts[len(starts)-1]
	cnt := len(starts) - 1
	centres := make([]float64, cnt)
	for i := 0; i < cnt; i++ {
		centres[i] = float64(starts[i]+starts[i+1]-1) / 2
	}
	ti := tileInterp{i0: make([]int32, n), i1: make([]int32, n), w: make([]float64, n)}
	seg := 0
	for c := 0; c < n; c++ {
		x := float64(c)
		switch {
		case cnt == 1 || x <= centres[0]:
			ti.i0[c], ti.i1[c], ti.w[c] = 0, 0, 0
		case x >= centres[cnt-1]:
			ti.i0[c], ti.i1[c], ti.w[c] = int32(cnt-1), int32(cnt-1), 0
		default:
			for seg+2 < cnt && x >= centres[seg+1] {
				seg++
			}
			ti.i0[c] = int32(seg)
			ti.i1[c] = int32(seg + 1)
			ti.w[c] = (x - centres[seg]) / (centres[seg+1] - centres[seg])
		}
	}
	return ti
}

// tileRef 保存每个材质、每个小格的参考值（小格平均，已含退化）。
type tileRef struct {
	nx, ny int
	xTile  []int32
	yTile  []int32
	smooth bool
	xi, yi tileInterp
	ref    [NumMaterials][]float64
	has    [NumMaterials]bool // 该材质在整幅图里有有效像素
	// 报告用：含该材质有效像素的小格数，及这些小格自身平均的最小/最大值。
	tiles            [NumMaterials]int
	minMean, maxMean [NumMaterials]float64
}

// buildTileRef 统计每个小格每个材质的有效像素数与灰度和（excl 非 nil 时 excl[idx] != 0 的像素被跳过），
// 求出参考值。need 为 false 的材质不会被用到，但统计成本相同，所以一并算出。
// 用整数累加，结果与并行切分无关。
func buildTileRef(img Image, reg Regions, excl []uint8, size int, smooth bool) *tileRef {
	w, h := img.Width, img.Height
	xs := tileAxis(w, size)
	ys := tileAxis(h, size)
	nx, ny := len(xs)-1, len(ys)-1
	nt := nx * ny
	tr := &tileRef{
		nx: nx, ny: ny,
		xTile:  tileIndexTable(xs),
		yTile:  tileIndexTable(ys),
		smooth: smooth,
	}
	if smooth {
		tr.xi = newTileInterp(xs)
		tr.yi = newTileInterp(ys)
	}

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > h {
		workers = h
	}
	type partial struct {
		sum [NumMaterials][]int64
		cnt [NumMaterials][]int32
	}
	parts := make([]partial, workers)
	for i := range parts {
		for m := 0; m < NumMaterials; m++ {
			parts[i].sum[m] = make([]int64, nt)
			parts[i].cnt[m] = make([]int32, nt)
		}
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
		go func(part *partial, y0, y1 int) {
			defer wg.Done()
			for y := y0; y < y1; y++ {
				base := y * w
				rowTiles := int(tr.yTile[y]) * nx
				for tx := 0; tx < nx; tx++ {
					t := rowTiles + tx
					for x := xs[tx]; x < xs[tx+1]; x++ {
						idx := base + x
						if excl != nil && excl[idx] != 0 {
							continue
						}
						m := reg.Material[idx]
						part.sum[m][t] += int64(img.Pix[idx])
						part.cnt[m][t]++
					}
				}
			}
		}(&parts[wi], y0, y1)
	}
	wg.Wait()

	for m := 0; m < NumMaterials; m++ {
		sum := make([]int64, nt)
		cnt := make([]int64, nt)
		for wi := range parts {
			for t := 0; t < nt; t++ {
				sum[t] += parts[wi].sum[m][t]
				cnt[t] += int64(parts[wi].cnt[m][t])
			}
		}
		var totalSum, totalCnt int64
		for t := 0; t < nt; t++ {
			totalSum += sum[t]
			totalCnt += cnt[t]
		}
		if totalCnt == 0 {
			continue
		}
		tr.has[m] = true
		globalMean := float64(totalSum) / float64(totalCnt)
		ref := make([]float64, nt)
		first := true
		for ty := 0; ty < ny; ty++ {
			for tx := 0; tx < nx; tx++ {
				t := ty*nx + tx
				if cnt[t] > 0 {
					own := float64(sum[t]) / float64(cnt[t])
					tr.tiles[m]++
					if first || own < tr.minMean[m] {
						tr.minMean[m] = own
					}
					if first || own > tr.maxMean[m] {
						tr.maxMean[m] = own
					}
					first = false
				}
				if cnt[t] >= tileMinPixels {
					ref[t] = float64(sum[t]) / float64(cnt[t])
					continue
				}
				// 本格像素太少：合并 3x3 邻域。
				var ps, pc int64
				for dy := -1; dy <= 1; dy++ {
					yy := ty + dy
					if yy < 0 || yy >= ny {
						continue
					}
					for dx := -1; dx <= 1; dx++ {
						xx := tx + dx
						if xx < 0 || xx >= nx {
							continue
						}
						ps += sum[yy*nx+xx]
						pc += cnt[yy*nx+xx]
					}
				}
				if pc >= tileMinPixels {
					ref[t] = float64(ps) / float64(pc)
				} else {
					ref[t] = globalMean
				}
			}
		}
		tr.ref[m] = ref
	}
	return tr
}

// at 返回材质 m 在 (x, y) 处的参考值；调用前须确认 tr.has[m]。
func (tr *tileRef) at(x, y int, m uint8) float64 {
	r := tr.ref[m]
	if !tr.smooth {
		return r[int(tr.yTile[y])*tr.nx+int(tr.xTile[x])]
	}
	nx := tr.nx
	ya, yb := int(tr.yi.i0[y])*nx, int(tr.yi.i1[y])*nx
	xa, xb := int(tr.xi.i0[x]), int(tr.xi.i1[x])
	wx, wy := tr.xi.w[x], tr.yi.w[y]
	top := r[ya+xa] + (r[ya+xb]-r[ya+xa])*wx
	bot := r[yb+xa] + (r[yb+xb]-r[yb+xa])*wx
	return top + (bot-top)*wy
}

// tileRulesActive 报告材质参数里是否有任何起作用的小格规则（Abs 或 Pct 的任一极性）。
func tileRulesActive(mp MaterialParams) bool {
	return mp.Enabled && (mp.Abs.active(Bright) || mp.Abs.active(Dark) || mp.Pct.active(Bright) || mp.Pct.active(Dark))
}

// anyTileRules 报告参数里是否有任何起作用的小格规则。
func anyTileRules(p Params) bool {
	for i := 0; i < NumMaterials; i++ {
		if tileRulesActive(p.Materials[i]) {
			return true
		}
	}
	return false
}

// polRules 是某材质某极性起作用的规则及其判定常数。
type polRules struct {
	interval bool
	abs      bool
	pct      bool
	absOff   float64 // Abs 的偏移量（GV）
	pctMul   float64 // Pct 的乘数：亮 1+Bright/100，暗 1-Dark/100
	need     [3]int  // 每条规则（按位号）的最小连续像素数 max(Size, MinArea)，规则不起作用时为 0
}

// rulesFor 求某材质某极性起作用的规则。
func rulesFor(mp MaterialParams, pol Polarity) polRules {
	var r polRules
	minArea := mp.MinArea
	if minArea < 1 {
		minArea = 1
	}
	size := func(n int) int {
		if minArea > n {
			return minArea
		}
		return n
	}
	if !mp.IntervalOff {
		r.interval = true
		r.need[0] = size(mp.Size)
	}
	if mp.Abs.active(pol) {
		r.abs = true
		r.absOff = mp.Abs.offset(pol)
		r.need[1] = size(mp.Abs.size(pol))
	}
	if mp.Pct.active(pol) {
		r.pct = true
		off := mp.Pct.offset(pol)
		if pol == Bright {
			r.pctMul = 1 + off/100
		} else {
			r.pctMul = 1 - off/100
		}
		r.need[2] = size(mp.Pct.size(pol))
	}
	return r
}

// any 报告是否至少有一条规则起作用。
func (r polRules) any() bool { return r.interval || r.abs || r.pct }

// buildFlags 为材质 m 的某极性构建标记位图（并行）：每个像素一个字节，位为 RuleInterval/RuleAbsolute/RulePercent。
// excl 非 nil 时，excl[idx] != 0 的像素被完全跳过；lb 非 nil 时区间规则用局部背景归一化值；tr 供小格规则使用。
func buildFlags(img Image, reg Regions, m Material, pol Polarity, mp MaterialParams, rl polRules, excl []uint8, lb *localBackground, tr *tileRef) []uint8 {
	w, h := img.Width, img.Height
	flags := make([]uint8, w*h)
	useTiles := tr != nil && tr.has[m] && (rl.abs || rl.pct)
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > h {
		workers = h
	}
	if workers < 1 {
		return flags
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
					var bits uint8
					if rl.interval {
						f := v
						if lb != nil {
							f = lb.value(idx, reg.Material[idx])
						}
						if pol == Bright {
							if f > mp.High {
								bits |= RuleInterval
							}
						} else if f < mp.Low {
							bits |= RuleInterval
						}
					}
					if useTiles {
						ref := tr.at(x, y, uint8(m))
						if rl.abs {
							if pol == Bright {
								if v > ref+rl.absOff {
									bits |= RuleAbsolute
								}
							} else if v < ref-rl.absOff {
								bits |= RuleAbsolute
							}
						}
						if rl.pct {
							if pol == Bright {
								if v > ref*rl.pctMul {
									bits |= RulePercent
								}
							} else if v < ref*rl.pctMul {
								bits |= RulePercent
							}
						}
					}
					flags[idx] = bits
				}
			}
		}(y0, y1)
	}
	wg.Wait()
	return flags
}

// extractRunsBit 与 extractRuns 相同，但只看 mask 中含 bit 的像素。
func extractRunsBit(mask []uint8, bit uint8, w, h int) (runs []run, rowStart []int32) {
	rowStart = make([]int32, h+1)
	for y := 0; y < h; y++ {
		rowStart[y] = int32(len(runs))
		base := y * w
		x := 0
		for x < w {
			if mask[base+x]&bit == 0 {
				x++
				continue
			}
			x0 := x
			for x < w && mask[base+x]&bit != 0 {
				x++
			}
			runs = append(runs, run{y: int32(y), x0: int32(x0), x1: int32(x - 1)})
		}
	}
	rowStart[h] = int32(len(runs))
	return runs, rowStart
}

// acceptRules 对每条规则的标记各自做 8 连通分组，丢弃被标记像素数 < need 的连通块，
// 返回被接受像素的位图（每个像素是接受它的规则位的或），以及被丢弃的"原始标记并集"连通块数
// （并集连通块里没有任何被接受像素才计数）。
func acceptRules(flags []uint8, w, h int, need [3]int) (acc []uint8, filtered int) {
	acc = make([]uint8, len(flags))
	for bi := 0; bi < 3; bi++ {
		if need[bi] == 0 {
			continue
		}
		bit := uint8(1) << uint(bi)
		runs, rowStart := extractRunsBit(flags, bit, w, h)
		if len(runs) == 0 {
			continue
		}
		id, count := labelRuns(runs, rowStart)
		area := make([]int, count)
		for ri, r := range runs {
			area[id[ri]] += int(r.x1-r.x0) + 1
		}
		for ri, r := range runs {
			if area[id[ri]] < need[bi] {
				continue
			}
			base := int(r.y) * w
			for x := int(r.x0); x <= int(r.x1); x++ {
				acc[base+x] |= bit
			}
		}
	}

	// 被丢弃的原始并集连通块。
	runs, rowStart := extractRunsBit(flags, 0xFF, w, h)
	if len(runs) == 0 {
		return acc, 0
	}
	id, count := labelRuns(runs, rowStart)
	kept := make([]bool, count)
	for ri, r := range runs {
		if kept[id[ri]] {
			continue
		}
		base := int(r.y) * w
		for x := int(r.x0); x <= int(r.x1); x++ {
			if acc[base+x] != 0 {
				kept[id[ri]] = true
				break
			}
		}
	}
	for _, k := range kept {
		if !k {
			filtered++
		}
	}
	return acc, filtered
}

// analyzeAccepted 对被接受像素的并集做 8 连通分组，并逐组统计，生成缺陷。
// 每个像素的 Excess 取所有标记它的规则里超出各自判定线最远者。
func analyzeAccepted(img Image, acc []uint8, m Material, pol Polarity, mp MaterialParams, rl polRules, lb *localBackground, tr *tileRef) []Defect {
	w, h := img.Width, img.Height
	runs, rowStart := extractRunsBit(acc, 0xFF, w, h)
	id, count := labelRuns(runs, rowStart)
	if count == 0 {
		return nil
	}
	type groupAcc struct {
		area                   int
		minX, minY, maxX, maxY int
		sumX, sumY, sumGV      float64
		peakGV                 uint8
		peakExcess             float64
		hasPeak                bool
		rules                  uint8
	}
	groups := make([]groupAcc, count)
	for i := range groups {
		groups[i].minX = w
		groups[i].minY = h
		groups[i].maxX = -1
		groups[i].maxY = -1
	}
	useTiles := tr != nil && tr.has[m]
	for y := 0; y < h; y++ {
		rowBase := y * w
		r0, r1 := int(rowStart[y]), int(rowStart[y+1])
		for ri := r0; ri < r1; ri++ {
			r := runs[ri]
			ga := &groups[id[ri]]
			for x := int(r.x0); x <= int(r.x1); x++ {
				idx := rowBase + x
				bits := acc[idx]
				v := float64(img.Pix[idx])
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
				ga.rules |= bits

				var excess float64
				first := true
				take := func(e float64) {
					if first || e > excess {
						excess = e
						first = false
					}
				}
				if bits&RuleInterval != 0 {
					fv := v
					if lb != nil {
						fv = lb.value(idx, uint8(m))
					}
					if pol == Bright {
						take(fv - mp.High)
					} else {
						take(mp.Low - fv)
					}
				}
				if bits&(RuleAbsolute|RulePercent) != 0 && useTiles {
					ref := tr.at(x, y, uint8(m))
					if bits&RuleAbsolute != 0 {
						if pol == Bright {
							take(v - (ref + rl.absOff))
						} else {
							take((ref - rl.absOff) - v)
						}
					}
					if bits&RulePercent != 0 {
						if pol == Bright {
							take(v - ref*rl.pctMul)
						} else {
							take(ref*rl.pctMul - v)
						}
					}
				}
				if !ga.hasPeak || excess > ga.peakExcess {
					ga.hasPeak = true
					ga.peakExcess = excess
					ga.peakGV = img.Pix[idx]
				}
			}
		}
	}

	defects := make([]Defect, 0, count)
	for gi := range groups {
		ga := &groups[gi]
		if ga.area == 0 {
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
			Rules:     ga.rules,
		})
	}
	return defects
}

// pixelFlagged 判断单个像素在某材质某极性下是否被任一规则标记（ProbeRect 使用，不做 Size 过滤）。
func pixelFlagged(v float64, f float64, ref float64, hasRef bool, pol Polarity, mp MaterialParams, rl polRules) bool {
	if rl.interval {
		if pol == Bright {
			if f > mp.High {
				return true
			}
		} else if f < mp.Low {
			return true
		}
	}
	if hasRef {
		if rl.abs {
			if pol == Bright {
				if v > ref+rl.absOff {
					return true
				}
			} else if v < ref-rl.absOff {
				return true
			}
		}
		if rl.pct {
			if pol == Bright {
				if v > ref*rl.pctMul {
					return true
				}
			} else if v < ref*rl.pctMul {
				return true
			}
		}
	}
	return false
}
