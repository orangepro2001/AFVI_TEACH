package core

import (
	"fmt"
	"math"
	"time"
)

// AlignBorderPx is the width in full-resolution pixels of the frame along the image border that
// CaptureAlignment.NoData always marks. The alignment (a few pixels of residual error, the
// interpolation) cannot vouch for the outermost pixels, and the picture a shifted product leaves
// at the border of the field of view is not the product, so the inspection skips them.
const AlignBorderPx = 8

// CaptureAlignment is the result of aligning a full-resolution capture onto the design grid.
type CaptureAlignment struct {
	Report    AlignReport // computed at working resolution (template and ROIs divided by Scale)
	Usable    bool        // Report.Fit != nil && Report.Level != VerdictNG
	Fit       Similarity  // FULL-resolution fit (template point -> capture point); {Scale: 1} when !Usable
	Image     *GrayImage  // the capture resampled onto the design grid, full resolution, no black border (see ReplicateNoData); nil when !Usable
	NoData    []bool      // NoData[i] is true where the capture had no data at design pixel i (the border the shift opened, which Image fills by replication) and in the AlignBorderPx frame; nil when !Usable
	Scale     int         // the working-resolution divisor actually used (a scale below 1 counts as 1)
	ElapsedMs float64     // wall time of the alignment
}

// AlignCapture aligns a capture (full resolution) to the imagined picture template (working
// resolution: the size of DownsampleArea(capture, scale)) with ROIs given in FULL-resolution
// design coordinates. A scale below 1 counts as 1. Errors: a nil image, fewer than 2 ROIs, a
// template whose size is not the working size, or a DownsampleArea failure. The working fit is
// converted to full resolution with FitToFullRes and the capture is resampled onto the design
// grid with RealignInput; ReplicateNoData then closes the black border the shift opened, so the
// inspection never sees one.
func AlignCapture(template, capture *GrayImage, rois []AlignRoi, scale int, cfg AlignConfig) (CaptureAlignment, error) {
	start := time.Now()
	if template == nil || capture == nil {
		return CaptureAlignment{}, fmt.Errorf("AlignCapture: template and capture must not be nil")
	}
	if len(rois) < 2 {
		return CaptureAlignment{}, fmt.Errorf("AlignCapture: need at least 2 ROIs, got %d", len(rois))
	}
	s := scale
	if s < 1 {
		s = 1
	}
	work, err := DownsampleArea(capture, s)
	if err != nil {
		return CaptureAlignment{}, fmt.Errorf("AlignCapture: %w", err)
	}
	if template.Width != work.Width || template.Height != work.Height {
		return CaptureAlignment{}, fmt.Errorf("AlignCapture: template is %dx%d, want the working size %dx%d",
			template.Width, template.Height, work.Width, work.Height)
	}
	div := float64(s)
	workRois := make([]AlignRoi, len(rois))
	for i, roi := range rois {
		workRois[i] = AlignRoi{Name: roi.Name, Rect: Rect{
			X: roi.X / div, Y: roi.Y / div, W: roi.W / div, H: roi.H / div,
		}}
	}
	rep := AlignWithRois(template, work, workRois, cfg, Similarity{Scale: 1})
	out := CaptureAlignment{Report: rep, Scale: s}
	out.Usable = rep.Fit != nil && rep.Level != VerdictNG
	if out.Usable {
		out.Fit = FitToFullRes(*rep.Fit, s)
		re := RealignInput(capture, out.Fit, nil)
		out.Image = ReplicateNoData(re)
		out.NoData = make([]bool, len(re.Valid))
		w, h := re.Image.Width, re.Image.Height
		for i, v := range re.Valid {
			x, y := i%w, i/w
			out.NoData[i] = !v || x < AlignBorderPx || y < AlignBorderPx || x >= w-AlignBorderPx || y >= h-AlignBorderPx
		}
	} else {
		out.Fit = Similarity{Scale: 1}
	}
	out.ElapsedMs = float64(time.Since(start).Nanoseconds()) / 1e6
	return out, nil
}

// FitToFullRes converts a working-resolution fit to full resolution. DownsampleArea puts working
// pixel i on full-resolution coordinate s*i + (s-1)/2 (s = scale), so with a = fit.Scale*cos(theta)
// and b = fit.Scale*sin(theta):
//
//	Tx_full = s*Tx + (s-1)/2 * (1 - (a - b)),  Ty_full = s*Ty + (s-1)/2 * (1 - (a + b));
//
// ThetaDeg and Scale stay. scale <= 1 returns fit unchanged.
func FitToFullRes(fit Similarity, scale int) Similarity {
	if scale <= 1 {
		return fit
	}
	s := float64(scale)
	c := (s - 1) / 2
	theta := degToRad(fit.ThetaDeg)
	a := fit.Scale * math.Cos(theta)
	b := fit.Scale * math.Sin(theta)
	return Similarity{
		Tx:       s*fit.Tx + c*(1-(a-b)),
		Ty:       s*fit.Ty + c*(1-(a+b)),
		ThetaDeg: fit.ThetaDeg,
		Scale:    fit.Scale,
	}
}

// ReplicateNoData returns a copy of re.Image in which every pixel with !re.Valid is replaced by
// the nearest valid pixel of the same row (the nearer side; equal distance takes the left one);
// rows without any valid pixel are copied from the nearest row that has one (the nearer side,
// the upper row on a tie). When nothing is valid the plain copy is returned. (The inspection
// must not see a black frame the shift opened.)
func ReplicateNoData(re *Realigned) *GrayImage {
	out := re.Image.Clone()
	w, h := out.Width, out.Height
	if w <= 0 || h <= 0 {
		return out
	}
	valid := func(i int) bool { return i >= 0 && i < len(re.Valid) && re.Valid[i] }
	leftDist := make([]int, w)
	leftVal := make([]uint8, w)
	rightDist := make([]int, w)
	rightVal := make([]uint8, w)
	hasValid := make([]bool, h)
	for y := 0; y < h; y++ {
		base := y * w
		last := -1
		for x := 0; x < w; x++ {
			if valid(base + x) {
				last = x
			} else if last >= 0 {
				leftDist[x] = x - last
				leftVal[x] = out.Data[base+last]
			} else {
				leftDist[x] = -1
			}
		}
		next := -1
		for x := w - 1; x >= 0; x-- {
			if valid(base + x) {
				next = x
			} else if next >= 0 {
				rightDist[x] = next - x
				rightVal[x] = out.Data[base+next]
			} else {
				rightDist[x] = -1
			}
		}
		for x := 0; x < w; x++ {
			i := base + x
			if valid(i) {
				hasValid[y] = true
				continue
			}
			ld, rd := leftDist[x], rightDist[x]
			if ld >= 0 && (rd < 0 || ld <= rd) {
				out.Data[i] = leftVal[x]
			} else if rd >= 0 {
				out.Data[i] = rightVal[x]
			}
		}
	}
	anyValid := false
	for _, v := range hasValid {
		if v {
			anyValid = true
			break
		}
	}
	if !anyValid {
		return out
	}
	for y := 0; y < h; y++ {
		if hasValid[y] {
			continue
		}
		src := -1
		for d := 1; d < h; d++ {
			if y-d >= 0 && hasValid[y-d] {
				src = y - d
				break
			}
			if y+d < h && hasValid[y+d] {
				src = y + d
				break
			}
		}
		if src < 0 {
			continue
		}
		copy(out.Data[y*w:(y+1)*w], out.Data[src*w:(src+1)*w])
	}
	return out
}
