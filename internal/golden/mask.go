// Package golden is the golden board of the teach program: the mask on the
// design grid that decides what is inspected, plus the morphology used to
// build it. It uses the standard library only.
package golden

import (
	"runtime"
	"sync"
	// math is not needed; the separable passes only use integer counts.
)

// MaxMorph is the largest morphology radius accepted by Erode and Dilate.
const MaxMorph = 256

// Mask is a binary mask on the design grid. Pix[y*W+x] is 0 = outside and any
// nonzero value = inside. Every function of this package returns 0/1 values.
type Mask struct {
	W, H int
	Pix  []uint8
}

// NewMask returns a zeroed w*h mask. Negative sizes count as 0.
func NewMask(w, h int) *Mask {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return &Mask{W: w, H: h, Pix: make([]uint8, w*h)}
}

// Clone returns a deep copy. A nil mask yields nil.
func (m *Mask) Clone() *Mask {
	if m == nil {
		return nil
	}
	c := &Mask{W: m.W, H: m.H, Pix: make([]uint8, len(m.Pix))}
	copy(c.Pix, m.Pix)
	return c
}

// Count returns the number of set pixels. A nil mask yields 0.
func (m *Mask) Count() int {
	if m == nil {
		return 0
	}
	n := 0
	for _, v := range m.Pix {
		if v != 0 {
			n++
		}
	}
	return n
}

// At reports whether the pixel is inside. It is false outside the mask and
// for a nil mask.
func (m *Mask) At(x, y int) bool {
	if m == nil || x < 0 || y < 0 || x >= m.W || y >= m.H {
		return false
	}
	return m.Pix[y*m.W+x] != 0
}

// Equal reports same size and same inside/outside pixels. Nil equals nil.
func (m *Mask) Equal(o *Mask) bool {
	if m == nil || o == nil {
		return m == nil && o == nil
	}
	if m.W != o.W || m.H != o.H || len(m.Pix) != len(o.Pix) {
		return false
	}
	for i := range m.Pix {
		if (m.Pix[i] != 0) != (o.Pix[i] != 0) {
			return false
		}
	}
	return true
}

// Valid reports a non-nil mask with positive size and a matching pixel slice.
func (m *Mask) Valid() bool {
	return m != nil && m.W > 0 && m.H > 0 && len(m.Pix) == m.W*m.H
}

// Threshold builds a mask where inside <=> lo <= v <= hi; with outRange the
// inside test is v < lo || v > hi. A short pix slice yields an empty w*h mask.
func Threshold(pix []uint8, w, h int, lo, hi uint8, outRange bool) *Mask {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	m := NewMask(w, h)
	if len(pix) < w*h {
		return m
	}
	for i := 0; i < w*h; i++ {
		v := pix[i]
		in := lo <= v && v <= hi
		if outRange {
			in = v < lo || v > hi
		}
		if in {
			m.Pix[i] = 1
		}
	}
	return m
}

// clampMorph clamps a morphology radius into 0..MaxMorph.
func clampMorph(n int) int {
	if n < 0 {
		return 0
	}
	if n > MaxMorph {
		return MaxMorph
	}
	return n
}

// Erode returns the erosion with a square window of radius n. A pixel stays
// inside only when every pixel of its window that lies inside the image is
// inside; pixels outside the image count as inside. n <= 0 clones the mask.
func Erode(m *Mask, n int) *Mask {
	n = clampMorph(n)
	if !m.Valid() {
		return m.Clone()
	}
	if n == 0 {
		return m.Clone()
	}
	return morph(m, n, false)
}

// Dilate returns the dilation with a square window of radius n. A pixel
// becomes inside when any pixel of its window that lies inside the image is
// inside; pixels outside the image count as outside. n <= 0 clones the mask.
func Dilate(m *Mask, n int) *Mask {
	n = clampMorph(n)
	if !m.Valid() {
		return m.Clone()
	}
	if n == 0 {
		return m.Clone()
	}
	return morph(m, n, true)
}

// morph runs the separable morphology. A horizontal pass stores, per pixel, the number of inside
// pixels (dilate) or outside pixels (erode) of its clipped row window; a vertical pass sums those
// counts over the clipped column window, so dilate keeps pixels with a nonzero sum and erode
// pixels with a zero sum. Pixels outside the image never enter a window, which gives erode
// "outside counts as inside" and dilate "outside counts as outside".
func morph(m *Mask, n int, dilate bool) *Mask {
	w, h := m.W, m.H
	hor := make([]int32, w*h)
	parallelRows(h, func(y int) {
		base := y * w
		cnt := int32(0) // inside pixels in the clipped window x-n..x+n
		for x := 0; x <= n && x < w; x++ {
			if insideByte(m.Pix[base+x]) {
				cnt++
			}
		}
		for x := 0; x < w; x++ {
			if dilate {
				hor[base+x] = cnt
			} else {
				hor[base+x] = int32(windowLen(x, w, n)) - cnt // outside pixels of the clipped window
			}
			if x+n+1 < w && insideByte(m.Pix[base+x+n+1]) {
				cnt++
			}
			if x-n >= 0 && insideByte(m.Pix[base+x-n]) {
				cnt--
			}
		}
	})
	out := NewMask(w, h)
	// The vertical pass streams over rows for a strip of columns, so memory access stays contiguous.
	const strip = 256
	strips := (w + strip - 1) / strip
	parallelRows(strips, func(s int) {
		x0, x1 := s*strip, (s+1)*strip
		if x1 > w {
			x1 = w
		}
		acc := make([]int32, x1-x0)
		for y := 0; y <= n && y < h; y++ {
			row := hor[y*w+x0 : y*w+x1]
			for i, v := range row {
				acc[i] += v
			}
		}
		for y := 0; y < h; y++ {
			dst := out.Pix[y*w+x0 : y*w+x1]
			for i, v := range acc {
				if (dilate && v > 0) || (!dilate && v == 0) {
					dst[i] = 1
				}
			}
			if y+n+1 < h {
				row := hor[(y+n+1)*w+x0 : (y+n+1)*w+x1]
				for i, v := range row {
					acc[i] += v
				}
			}
			if y-n >= 0 {
				row := hor[(y-n)*w+x0 : (y-n)*w+x1]
				for i, v := range row {
					acc[i] -= v
				}
			}
		}
	})
	return out
}

// windowLen returns the number of in-range indices of i-n..i+n clipped to
// 0..size-1.
func windowLen(i, size, n int) int {
	lo := i - n
	if lo < 0 {
		lo = 0
	}
	hi := i + n
	if hi > size-1 {
		hi = size - 1
	}
	return hi - lo + 1
}

// insideByte reports a nonzero pixel.
func insideByte(v uint8) bool { return v != 0 }

// parallelRows runs fn(0..n-1) on at most runtime.NumCPU goroutines.
func parallelRows(n int, fn func(i int)) {
	if n <= 0 {
		return
	}
	workers := runtime.NumCPU()
	if workers > n {
		workers = n
	}
	if workers <= 1 {
		for i := 0; i < n; i++ {
			fn(i)
		}
		return
	}
	var wg sync.WaitGroup
	chunk := (n + workers - 1) / workers
	for w := 0; w < workers; w++ {
		lo := w * chunk
		hi := lo + chunk
		if hi > n {
			hi = n
		}
		if lo >= hi {
			break
		}
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			for i := lo; i < hi; i++ {
				fn(i)
			}
		}(lo, hi)
	}
	wg.Wait()
}

// Open is Dilate(Erode(m, n), n).
func Open(m *Mask, n int) *Mask {
	return Dilate(Erode(m, n), n)
}

// Close is Erode(Dilate(m, n), n).
func Close(m *Mask, n int) *Mask {
	return Erode(Dilate(m, n), n)
}

// And returns a AND b. Different sizes or a nil b yield an empty mask.
func And(a, b *Mask) *Mask {
	if !a.Valid() {
		return nil
	}
	if !b.Valid() || a.W != b.W || a.H != b.H {
		return NewMask(a.W, a.H)
	}
	out := NewMask(a.W, a.H)
	for i := range out.Pix {
		if a.Pix[i] != 0 && b.Pix[i] != 0 {
			out.Pix[i] = 1
		}
	}
	return out
}

// AndNot returns a AND NOT b. A nil b is a no-op (a clone); different sizes
// yield a clone of a.
func AndNot(a, b *Mask) *Mask {
	if !a.Valid() {
		return a.Clone()
	}
	if !b.Valid() || a.W != b.W || a.H != b.H {
		return a.Clone()
	}
	out := NewMask(a.W, a.H)
	for i := range out.Pix {
		if a.Pix[i] != 0 && b.Pix[i] == 0 {
			out.Pix[i] = 1
		}
	}
	return out
}
