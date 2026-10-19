package core

import (
	"math"
	"runtime"
	"sync"
)

// Realigned is an input image resampled onto the template grid.
type Realigned struct {
	Image *GrayImage
	// Valid[i] is false where the pixel has no data: its source point lies outside the input image
	// or inside an excluded rectangle. Such pixels are 0 in Image.
	Valid []bool
}

// RealignInput resamples the input onto the template grid through the fitted similarity, the way
// the machine would use the alignment result: the pixel (x, y) of the result is the input at
// fit(x, y), the position the fit says the template point (x, y) moved to. Nothing but the input and
// the fit goes in; the true warp of a simulated capture is never used, so the result shows what the
// alignment really achieved. exclude lists rectangles in input coordinates (occlusions) whose
// pixels, with one pixel of margin for the interpolation, are reported as having no data.
func RealignInput(input *GrayImage, fit Similarity, exclude []Rect) *Realigned {
	width, height := input.Width, input.Height
	out := &Realigned{
		Image: &GrayImage{Width: width, Height: height, Data: make([]uint8, width*height)},
		Valid: make([]bool, width*height),
	}
	theta := degToRad(fit.ThetaDeg)
	cos := math.Cos(theta)
	sin := math.Sin(theta)
	type box struct{ x0, y0, x1, y1 float64 }
	boxes := make([]box, len(exclude))
	for i, r := range exclude {
		boxes[i] = box{math.Floor(r.X) - 1, math.Floor(r.Y) - 1, math.Ceil(r.X+r.W) + 1, math.Ceil(r.Y+r.H) + 1}
	}
	rows := func(y0, y1 int) {
		for y := y0; y < y1; y++ {
			for x := 0; x < width; x++ {
				sx := fit.Scale*(cos*float64(x)-sin*float64(y)) + fit.Tx
				sy := fit.Scale*(sin*float64(x)+cos*float64(y)) + fit.Ty
				covered := false
				for _, b := range boxes {
					if sx >= b.x0 && sx < b.x1 && sy >= b.y0 && sy < b.y1 {
						covered = true
						break
					}
				}
				if covered {
					continue
				}
				if v, ok := SampleBilinear(input, sx, sy); ok {
					out.Image.Data[y*width+x] = ClampByte(v)
					out.Valid[y*width+x] = true
				}
			}
		}
	}
	workers := runtime.NumCPU()
	if workers > height {
		workers = height
	}
	if workers <= 1 {
		rows(0, height)
		return out
	}
	var wg sync.WaitGroup
	per := (height + workers - 1) / workers
	for y0 := 0; y0 < height; y0 += per {
		y1 := y0 + per
		if y1 > height {
			y1 = height
		}
		wg.Add(1)
		go func(y0, y1 int) {
			defer wg.Done()
			rows(y0, y1)
		}(y0, y1)
	}
	wg.Wait()
	return out
}

// MeanAbsDiff is the mean of |a-b| over the pixels where mask is true (every pixel when mask is
// nil) and the number of pixels used; the mean is NaN when no pixel is used.
func MeanAbsDiff(a, b *GrayImage, mask []bool) (float64, int) {
	n := len(a.Data)
	if len(b.Data) < n {
		n = len(b.Data)
	}
	if mask != nil && len(mask) < n {
		n = len(mask)
	}
	var sum uint64
	used := 0
	for i := 0; i < n; i++ {
		if mask != nil && !mask[i] {
			continue
		}
		d := int(a.Data[i]) - int(b.Data[i])
		if d < 0 {
			d = -d
		}
		sum += uint64(d)
		used++
	}
	if used == 0 {
		return math.NaN(), 0
	}
	return float64(sum) / float64(used), used
}

// ShiftCheck is the outcome of CheckRealigned.
type ShiftCheck struct {
	Total   int     // windows tried: those that lie completely on pixels that have data
	Windows int     // windows that were found again in the realigned image
	MaxPx   float64 // largest remaining shift of a window, in pixels of the image
	RmsPx   float64
	Worst   Point // centre of the window with the largest shift, template coordinates
	Win     int   // side of the windows in pixels
}

// CheckRealigned measures how well a realigned image sits on the template without the ground
// truth and without the ROIs: it cuts a grid of windows (5 x 4, about 1/20 of the image width)
// out of the template, looks for each of them again in the realigned image and reports how far
// the matches are from where they would be if the alignment were perfect. A ROI fit can look
// excellent at the ROIs and still leave the rest of the image off, because a similarity fitted
// to a few points extrapolates badly; this is the check on the whole picture. pyramid is the
// coarse-to-fine factor of the search (limited to 2..4); windows that are flat, ambiguous (a
// periodic pattern) or not found are left out of the statistics.
func CheckRealigned(template *GrayImage, re *Realigned, pyramid int) ShiftCheck {
	const cols, rows = 5, 4
	w, h := template.Width, template.Height
	win := w / 20
	if h/14 < win {
		win = h / 14
	}
	var out ShiftCheck
	if win < 16 || len(re.Valid) < w*h {
		return out
	}
	covered := func(x0, y0 int) bool {
		for y := y0; y < y0+win; y++ {
			for x := x0; x < x0+win; x++ {
				if !re.Valid[y*w+x] {
					return false
				}
			}
		}
		return true
	}
	var rois []AlignRoi
	for gy := 0; gy < rows; gy++ {
		for gx := 0; gx < cols; gx++ {
			x0 := win + gx*(w-2*win-win)/(cols-1)
			y0 := win + gy*(h-2*win-win)/(rows-1)
			if !covered(x0, y0) {
				continue
			}
			rois = append(rois, AlignRoi{Name: "check", Rect: Rect{X: float64(x0), Y: float64(y0), W: float64(win), H: float64(win)}})
		}
	}
	out.Total = len(rois)
	out.Win = win
	if len(rois) == 0 {
		return out
	}
	if pyramid < 2 {
		pyramid = 2
	}
	if pyramid > 4 {
		pyramid = 4
	}
	cfg := DefaultAlignConfig
	cfg.PyramidFactor = pyramid
	cfg.SearchRadius = 48
	cfg.MinNcc = 0.5
	cfg.MinOverlap = 1
	rep := AlignWithRois(template, re.Image, rois, cfg, Similarity{Scale: 1})
	var sumSq float64
	for _, m := range rep.Matches {
		if !m.Located || m.Found == nil || m.Uniqueness < 0.1 {
			continue
		}
		d := math.Hypot(m.Found.X-m.Template.X, m.Found.Y-m.Template.Y)
		out.Windows++
		sumSq += d * d
		if d > out.MaxPx {
			out.MaxPx = d
			out.Worst = m.Template
		}
	}
	if out.Windows > 0 {
		out.RmsPx = math.Sqrt(sumSq / float64(out.Windows))
	}
	return out
}
