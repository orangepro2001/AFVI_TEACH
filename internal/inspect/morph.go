package inspect

// grow 用 (2r+1)x(2r+1) 的方形结构元膨胀 mask，返回新切片，不修改输入。
// 可分离 O(N)：
//
//	水平：逐行前向扫描记住最近被标记的 x（当 x-last <= r 时标记 x），后向扫描同理；
//	垂直：维护 up[x] / down[x] 数组（到上方/下方最近水平标记像素的行距离，int32，封顶）；
//	      当 up<=r 或 down<=r 时该像素被置位。
func grow(mask []uint8, w, h, r int) []uint8 {
	out := make([]uint8, len(mask))
	if r <= 0 {
		copy(out, mask)
		return out
	}
	if w <= 0 || h <= 0 {
		return out
	}
	// 水平膨胀。
	horiz := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		base := y * w
		last := -1 << 30
		for x := 0; x < w; x++ {
			if mask[base+x] != 0 {
				last = x
			}
			if x-last <= r {
				horiz[base+x] = 1
			}
		}
		last = 1 << 30
		for x := w - 1; x >= 0; x-- {
			if mask[base+x] != 0 {
				last = x
			}
			if last-x <= r {
				horiz[base+x] = 1
			}
		}
	}
	// 垂直膨胀：两趟扫描，分别维护到上方/下方最近水平标记像素的行距离。
	const far = 1 << 30
	up := make([]int32, w)
	down := make([]int32, w)
	for x := 0; x < w; x++ {
		up[x] = far
		down[x] = far
	}
	// 从上往下：up[x] 为到上方最近水平标记像素的行距离。
	for y := 0; y < h; y++ {
		base := y * w
		for x := 0; x < w; x++ {
			if horiz[base+x] != 0 {
				up[x] = 0
			} else if up[x] < far {
				up[x]++
			}
			if up[x] <= int32(r) {
				out[base+x] = 1
			}
		}
	}
	// 从下往上：down[x] 为到下方最近水平标记像素的行距离。
	for y := h - 1; y >= 0; y-- {
		base := y * w
		for x := 0; x < w; x++ {
			if horiz[base+x] != 0 {
				down[x] = 0
			} else if down[x] < far {
				down[x]++
			}
			if down[x] <= int32(r) {
				out[base+x] = 1
			}
		}
	}
	return out
}

// run 表示一行内的一段连续标记像素，x1 为闭区间端点。
type run struct {
	y, x0, x1 int32
}

// extractRuns 提取掩码中的所有行程，返回行程列表与每行起始索引（长度 h+1）。
// 行程按 (y, x0) 排序。
func extractRuns(mask []uint8, w, h int) (runs []run, rowStart []int32) {
	rowStart = make([]int32, h+1)
	for y := 0; y < h; y++ {
		rowStart[y] = int32(len(runs))
		base := y * w
		x := 0
		for x < w {
			if mask[base+x] == 0 {
				x++
				continue
			}
			x0 := x
			for x < w && mask[base+x] != 0 {
				x++
			}
			runs = append(runs, run{y: int32(y), x0: int32(x0), x1: int32(x - 1)})
		}
	}
	rowStart[h] = int32(len(runs))
	return runs, rowStart
}

// labelRuns 对行程做 8 连通标记，返回每个行程的紧凑 id（0..count-1，按首次出现顺序）。
// 相邻两行中的两个行程 a、b 连通当且仅当 a.x0 <= b.x1+1 && b.x0 <= a.x1+1。
// 使用并查集（路径压缩）与相邻行双指针扫描。
func labelRuns(runs []run, rowStart []int32) (id []int32, count int) {
	n := len(runs)
	id = make([]int32, n)
	if n == 0 {
		return id, 0
	}
	parent := make([]int32, n)
	for i := range parent {
		parent[i] = int32(i)
	}
	var find func(x int32) int32
	find = func(x int32) int32 {
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}
		return x
	}
	union := func(a, b int32) {
		ra, rb := find(a), find(b)
		if ra != rb {
			if ra < rb {
				parent[rb] = ra
			} else {
				parent[ra] = rb
			}
		}
	}
	h := len(rowStart) - 1
	for y := 0; y+1 < h; y++ {
		a0, a1 := int(rowStart[y]), int(rowStart[y+1])
		b0, b1 := int(rowStart[y+1]), int(rowStart[y+2])
		i, j := a0, b0
		for i < a1 && j < b1 {
			a := runs[i]
			b := runs[j]
			if a.x0 <= b.x1+1 && b.x0 <= a.x1+1 {
				union(int32(i), int32(j))
			}
			if a.x1 < b.x1 {
				i++
			} else {
				j++
			}
		}
	}
	// 紧凑化 id，按首次出现顺序。
	remap := make([]int32, n)
	for i := range remap {
		remap[i] = -1
	}
	next := int32(0)
	for i := 0; i < n; i++ {
		root := find(int32(i))
		if remap[root] < 0 {
			remap[root] = next
			next++
		}
		id[i] = remap[root]
	}
	return id, int(next)
}
