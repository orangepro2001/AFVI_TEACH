package inspect

import (
	"math"
	"runtime"
	"sync"
)

// MaxBackgroundRadius 是 BackgroundRadius 允许的最大值。
const MaxBackgroundRadius = 16

// localBackground 保存局部背景模式下的逐像素残差与逐材质统计。
type localBackground struct {
	resid  []int16               // d；被忽略像素为 0
	use    [NumMaterials]bool    // 该材质是否启用局部背景
	anchor [NumMaterials]float64 // anchor_m
	count  [NumMaterials]int
	sumV   [NumMaterials]float64 // sum of v over valid pixels
	sumD   [NumMaterials]float64
	sumDD  [NumMaterials]float64
	minD   [NumMaterials]int16
	maxD   [NumMaterials]int16
}

// computeBackground 计算局部背景：对 use[m] 为 true 的材质分别做 (2r+1)^2 窗口的
// 中位数滤波，得到残差 d = v - b 以及逐材质的累加量。r 必须已在
// 1..MaxBackgroundRadius 内，且输入已通过校验。
func computeBackground(img Image, reg Regions, excl []uint8, r int, use [NumMaterials]bool) *localBackground {
	w, h := img.Width, img.Height
	lb := &localBackground{resid: make([]int16, w*h), use: use}
	for m := 0; m < NumMaterials; m++ {
		lb.minD[m] = 32767
		lb.maxD[m] = -32768
	}

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > h {
		workers = h
	}
	if workers < 1 {
		return lb
	}

	for m := 0; m < NumMaterials; m++ {
		if !use[m] {
			continue
		}
		// 该材质没有任何有效像素时跳过整个 pass。
		hasMember := false
		for i := 0; i < w*h; i++ {
			if reg.Material[i] != uint8(m) {
				continue
			}
			if excl != nil && excl[i] != 0 {
				continue
			}
			hasMember = true
			break
		}
		if !hasMember {
			continue
		}

		type partial struct {
			count int
			sumV  float64
			sumD  float64
			sumDD float64
			minD  int16
			maxD  int16
		}
		partials := make([]partial, workers)
		for i := range partials {
			partials[i].minD = 32767
			partials[i].maxD = -32768
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
			go func(wi, y0, y1 int) {
				defer wg.Done()
				p := &partials[wi]
				var hist [256]int32
				for y := y0; y < y1; y++ {
					// 每行重新开始滑动窗口。
					for i := range hist {
						hist[i] = 0
					}
					n := 0
					med := 0
					lt := 0

					addCol := func(x int) {
						if x < 0 || x >= w {
							return
						}
						cy0 := y - r
						if cy0 < 0 {
							cy0 = 0
						}
						cy1 := y + r
						if cy1 > h-1 {
							cy1 = h - 1
						}
						for cy := cy0; cy <= cy1; cy++ {
							idx := cy*w + x
							if reg.Material[idx] != uint8(m) {
								continue
							}
							if excl != nil && excl[idx] != 0 {
								continue
							}
							v := int(img.Pix[idx])
							hist[v]++
							n++
							if v < med {
								lt++
							}
						}
					}
					removeCol := func(x int) {
						if x < 0 || x >= w {
							return
						}
						cy0 := y - r
						if cy0 < 0 {
							cy0 = 0
						}
						cy1 := y + r
						if cy1 > h-1 {
							cy1 = h - 1
						}
						for cy := cy0; cy <= cy1; cy++ {
							idx := cy*w + x
							if reg.Material[idx] != uint8(m) {
								continue
							}
							if excl != nil && excl[idx] != 0 {
								continue
							}
							v := int(img.Pix[idx])
							hist[v]--
							n--
							if v < med {
								lt--
							}
						}
					}

					// 初始窗口：列 0..min(r, w-1)。
					last := r
					if last > w-1 {
						last = w - 1
					}
					for x := 0; x <= last; x++ {
						addCol(x)
					}

					for x := 0; x < w; x++ {
						if x > 0 {
							removeCol(x - r - 1)
							addCol(x + r)
						}
						idx := y*w + x
						if reg.Material[idx] != uint8(m) {
							continue
						}
						if excl != nil && excl[idx] != 0 {
							continue
						}
						if n == 0 {
							continue
						}
						k := n / 2
						for lt > k {
							med--
							lt -= int(hist[med])
						}
						for lt+int(hist[med]) <= k {
							lt += int(hist[med])
							med++
						}
						b := med
						v := int(img.Pix[idx])
						d := int16(v) - int16(b)
						lb.resid[idx] = d
						p.count++
						p.sumV += float64(v)
						p.sumD += float64(d)
						p.sumDD += float64(d) * float64(d)
						if d < p.minD {
							p.minD = d
						}
						if d > p.maxD {
							p.maxD = d
						}
					}
				}
			}(wi, y0, y1)
		}
		wg.Wait()

		// 按 worker 顺序合并，保证确定性。
		for wi := 0; wi < workers; wi++ {
			p := partials[wi]
			if p.count == 0 {
				continue
			}
			if lb.count[m] == 0 {
				lb.minD[m] = p.minD
				lb.maxD[m] = p.maxD
			} else {
				if p.minD < lb.minD[m] {
					lb.minD[m] = p.minD
				}
				if p.maxD > lb.maxD[m] {
					lb.maxD[m] = p.maxD
				}
			}
			lb.count[m] += p.count
			lb.sumV[m] += p.sumV
			lb.sumD[m] += p.sumD
			lb.sumDD[m] += p.sumDD
		}
	}

	for m := 0; m < NumMaterials; m++ {
		if lb.count[m] > 0 {
			lb.anchor[m] = (lb.sumV[m] - lb.sumD[m]) / float64(lb.count[m])
		} else {
			lb.minD[m] = 0
			lb.maxD[m] = 0
		}
	}
	return lb
}

// active 报告材质 m 是否使用局部背景且存在有效像素。
func (lb *localBackground) active(m int) bool {
	return lb.use[m] && lb.count[m] > 0
}

// value 返回像素 idx（材质 m）的局部归一化值 f = d + anchor_m，限制在 [0,255]，
// 因此区间 [0,255] 与全局模式一样永不标记任何像素。
func (lb *localBackground) value(idx int, m uint8) float64 {
	return clampFloat(float64(lb.resid[idx])+lb.anchor[m], 0, 255)
}

// pixels 返回材质 m 的有效像素数。
func (lb *localBackground) pixels(m int) int {
	return lb.count[m]
}

// mean 返回材质 m 的原始灰度均值（无像素时为 0）。
func (lb *localBackground) mean(m int) float64 {
	if lb.count[m] == 0 {
		return 0
	}
	return lb.sumV[m] / float64(lb.count[m])
}

// std 返回材质 m 的残差总体标准差（无像素时为 0）。
func (lb *localBackground) std(m int) float64 {
	if lb.count[m] == 0 {
		return 0
	}
	n := float64(lb.count[m])
	meanD := lb.sumD[m] / n
	variance := lb.sumDD[m]/n - meanD*meanD
	if variance < 0 {
		variance = 0
	}
	return math.Sqrt(variance)
}

// minmax 返回材质 m 的局部归一化值 f 的最小/最大值（无像素时为 0/0）。
func (lb *localBackground) minmax(m int) (uint8, uint8) {
	if lb.count[m] == 0 {
		return 0, 0
	}
	lo := clampU8(math.Round(lb.anchor[m] + float64(lb.minD[m])))
	hi := clampU8(math.Round(lb.anchor[m] + float64(lb.maxD[m])))
	return lo, hi
}

// clampU8 将 v 限制在 [0,255]（调用方需先取整）。
func clampU8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
