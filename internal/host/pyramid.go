package host

import (
	"runtime"
	"sync"
)

// MaxLevel is the coarsest pyramid level (each level halves the size: 1/2^level).
const MaxLevel = 5

// pyramid serves one viewer image at several resolutions. The front end draws from the
// coarsest level that still has at least one pixel per screen pixel, so panning and
// zooming a 5860×4020 image never scales 23 million pixels, and a slider drag only ever
// moves the small levels over the wire. The source is the RGBA buffer the panel handed to
// the viewer (never modified); when every pixel is grey it is served as one byte per pixel.
type pyramid struct {
	src  []byte
	w, h int
	gray bool // src is already one byte per pixel (SetGray)

	mu     sync.Mutex
	ready  bool
	isGray bool
	lv     [MaxLevel + 1]level
}

type level struct {
	w, h int
	pix  []byte
	done bool
}

func newPyramid(src []byte, w, h int, gray bool) *pyramid {
	return &pyramid{src: src, w: w, h: h, gray: gray}
}

// get returns level k (0 = full size), building what it needs on first use.
func (p *pyramid) get(k int) (pix []byte, w, h int, gray bool) {
	if k < 0 {
		k = 0
	}
	if k > MaxLevel {
		k = MaxLevel
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.ready {
		p.prepare()
	}
	for i := 1; i <= k; i++ {
		if p.lv[i].done {
			continue
		}
		prev := p.lv[i-1]
		nw, nh := (prev.w+1)/2, (prev.h+1)/2
		p.lv[i] = level{w: nw, h: nh, pix: halve(prev.pix, prev.w, prev.h, p.isGray), done: true}
	}
	l := p.lv[k]
	return l.pix, l.w, l.h, p.isGray
}

// prepare builds level 0: grey sources and RGBA images whose pixels are all grey become one
// byte per pixel; everything else stays RGBA.
func (p *pyramid) prepare() {
	p.ready = true
	if p.gray {
		p.isGray = true
		p.lv[0] = level{w: p.w, h: p.h, pix: p.src, done: true}
		return
	}
	n := p.w * p.h
	if len(p.src) < 4*n {
		p.lv[0] = level{w: p.w, h: p.h, pix: p.src, done: true}
		return
	}
	g := make([]byte, n)
	isGray := true
	var mu sync.Mutex
	parallel(n, func(s, e int) {
		ok := true
		src := p.src
		for i := s; i < e; i++ {
			o := 4 * i
			v := src[o]
			if src[o+1] != v || src[o+2] != v {
				ok = false
				break
			}
			g[i] = v
		}
		if !ok {
			mu.Lock()
			isGray = false
			mu.Unlock()
		}
	})
	if isGray {
		p.isGray = true
		p.lv[0] = level{w: p.w, h: p.h, pix: g, done: true}
		return
	}
	p.lv[0] = level{w: p.w, h: p.h, pix: p.src[:4*n], done: true}
}

// halve averages 2×2 blocks (edges replicate) of a grey or RGBA image.
func halve(src []byte, w, h int, gray bool) []byte {
	nw, nh := (w+1)/2, (h+1)/2
	c := 4
	if gray {
		c = 1
	}
	dst := make([]byte, nw*nh*c)
	parallel(nh, func(y0, y1 int) {
		for y := y0; y < y1; y++ {
			ya, yb := 2*y, 2*y+1
			if yb >= h {
				yb = h - 1
			}
			for x := 0; x < nw; x++ {
				xa, xb := 2*x, 2*x+1
				if xb >= w {
					xb = w - 1
				}
				for ch := 0; ch < c; ch++ {
					s := int(src[(ya*w+xa)*c+ch]) + int(src[(ya*w+xb)*c+ch]) + int(src[(yb*w+xa)*c+ch]) + int(src[(yb*w+xb)*c+ch])
					dst[(y*nw+x)*c+ch] = byte((s + 2) / 4)
				}
			}
		}
	})
	return dst
}

// parallel splits [0,n) into one chunk per CPU.
func parallel(n int, fn func(start, end int)) {
	workers := runtime.NumCPU()
	if workers < 1 || n < 4096 {
		fn(0, n)
		return
	}
	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for s := 0; s < n; s += chunk {
		e := s + chunk
		if e > n {
			e = n
		}
		wg.Add(1)
		go func(a, b int) {
			defer wg.Done()
			fn(a, b)
		}(s, e)
	}
	wg.Wait()
}
