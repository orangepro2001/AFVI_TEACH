package core

import (
	"math"
	"runtime"
	"sync"
)

// WarpMaterialToReference moves a per-pixel material map (one byte per pixel of a w*h image, e.g.
// MatBase..MatMetal) from the grid of the imagined design picture onto the grid of the real
// reference picture. fit is DesignInfo.Fit of DesignTemplate: it maps a point of the reference
// picture to the matching design point, in pixels of the image downsampled by scale (scale < 1
// counts as 1). Always returns a new slice of len w*h; a nil result means invalid input.
func WarpMaterialToReference(material []uint8, w, h int, fit Similarity, scale int) []uint8 {
	if w <= 0 || h <= 0 || len(material) != w*h {
		return nil
	}
	full := FitToFullRes(fit, scale)
	out := make([]uint8, w*h)
	theta := degToRad(full.ThetaDeg)
	cos := math.Cos(theta)
	sin := math.Sin(theta)
	rows := runtime.NumCPU()
	if rows < 1 {
		rows = 1
	}
	if rows > h {
		rows = h
	}
	if rows <= 1 {
		warpRows(material, out, w, h, full, cos, sin, 0, h)
		return out
	}
	var wg sync.WaitGroup
	rowsPer := (h + rows - 1) / rows
	for r := 0; r < rows; r++ {
		y0 := r * rowsPer
		y1 := y0 + rowsPer
		if y1 > h {
			y1 = h
		}
		if y0 >= y1 {
			break
		}
		wg.Add(1)
		go func(y0, y1 int) {
			defer wg.Done()
			warpRows(material, out, w, h, full, cos, sin, y0, y1)
		}(y0, y1)
	}
	wg.Wait()
	return out
}

// warpRows fills out[y*w+x] for y in [y0, y1) by sampling the nearest input pixel of
// ApplySimilarity(full, Point{x, y}), clamped to the image.
func warpRows(material, out []uint8, w, h int, full Similarity, cos, sin float64, y0, y1 int) {
	for y := y0; y < y1; y++ {
		fy := float64(y)
		base := y * w
		for x := 0; x < w; x++ {
			fx := float64(x)
			px := full.Scale*(cos*fx-sin*fy) + full.Tx
			py := full.Scale*(sin*fx+cos*fy) + full.Ty
			sx := int(math.Floor(px + 0.5))
			sy := int(math.Floor(py + 0.5))
			if sx < 0 {
				sx = 0
			} else if sx > w-1 {
				sx = w - 1
			}
			if sy < 0 {
				sy = 0
			} else if sy > h-1 {
				sy = h - 1
			}
			out[base+x] = material[sy*w+sx]
		}
	}
}
