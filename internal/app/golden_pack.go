package app

import (
	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
)

// goldenView is how the golden page paints the masks over the golden sample
// picture.
type goldenView struct {
	Color    [3]uint8 // colour of the Metal mask
	Opacity  int      // 0..100
	ShowMask bool     // paint the Metal mask (the inspected part, i.e. after the padding)
	ShowSR   bool     // paint the automatic SR mask in green
	ShowBand bool     // paint the padding band (drawn Metal minus inspected Metal) in azure
}

// defaultGoldenView returns the initial view: magenta (the final inspection area),
// half opacity, SR on, padding band off.
func defaultGoldenView() goldenView {
	return goldenView{Color: [3]uint8{255, 0, 255}, Opacity: 50, ShowMask: true, ShowSR: true, ShowBand: false}
}

// goldenSRColor and goldenBandColor are the fixed colours of the SR shade and of
// the padding band.
var goldenSRColor = [3]uint8{0, 200, 0}
var goldenBandColor = [3]uint8{70, 140, 255}

// PackGolden returns an opaque RGBA picture (GetRGBA buffer, the caller
// PutRGBA's it) of src painted with the masks.
func PackGolden(src *core.GrayImage, metal, metalInspect, sr *golden.Mask, view goldenView) []byte {
	if src == nil {
		return nil
	}
	w, h := src.Width, src.Height
	n := w * h
	if n <= 0 {
		dst := GetRGBA(0)
		return dst
	}
	op := view.Opacity
	if op < 0 {
		op = 0
	} else if op > 100 {
		op = 100
	}
	useInspect := metalInspect != nil && metalInspect.Valid() && metalInspect.W == w && metalInspect.H == h
	useMetal := metal != nil && metal.Valid() && metal.W == w && metal.H == h
	useSR := sr != nil && sr.Valid() && sr.W == w && sr.H == h

	paintMetal := view.ShowMask
	if paintMetal && !useInspect && !useMetal {
		paintMetal = false
	}
	paintBand := false
	var bandMask *golden.Mask
	if useInspect && useMetal {
		bandMask = metal
		paintBand = view.ShowBand
	} else if !useInspect && useMetal && !view.ShowMask {
		bandMask = metal
		paintBand = view.ShowBand
	}
	paintSR := view.ShowSR && useSR

	dst := GetRGBA(4 * n)
	srcPix := src.Data
	parallelRows(n, func(start, end int) {
		for i := start; i < end; i++ {
			o := 4 * i
			gv := srcPix[i]
			var c [3]uint8
			c[0], c[1], c[2] = gv, gv, gv
			if paintMetal {
				if useInspect && metalInspect.Pix[i] != 0 {
					c = goldenBlend(c, view.Color, op)
				} else if !useInspect && useMetal && metal.Pix[i] != 0 {
					c = goldenBlend(c, view.Color, op)
				}
			}
			if paintBand && bandMask.Pix[i] != 0 {
				if useInspect {
					if metalInspect.Pix[i] == 0 {
						c = goldenBlend(c, goldenBandColor, op)
					}
				} else {
					c = goldenBlend(c, goldenBandColor, op)
				}
			}
			if paintSR && sr.Pix[i] != 0 {
				c = goldenBlend(c, goldenSRColor, op)
			}
			dst[o], dst[o+1], dst[o+2], dst[o+3] = c[0], c[1], c[2], 255
		}
	})
	return dst
}

// goldenBlend mixes a grey value with col at opacity op (0..100) per channel:
// out = (grey*(100-op) + col*op + 50) / 100.
func goldenBlend(grey [3]uint8, col [3]uint8, op int) [3]uint8 {
	var out [3]uint8
	for k := 0; k < 3; k++ {
		v := (int(grey[k])*(100-op) + int(col[k])*op + 50) / 100
		if v < 0 {
			v = 0
		} else if v > 255 {
			v = 255
		}
		out[k] = uint8(v)
	}
	return out
}
