package app

import (
	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
)

// PackGrayTint is PackGray whose pixels inside mask (Pix[i] != 0) are blended with col at
// opacity op (0..100, clamped). A nil or invalid mask, or a mask of another size, yields
// exactly the bytes of PackGray(img). A nil img returns nil.
func PackGrayTint(img *core.GrayImage, mask *golden.Mask, col [3]uint8, op int) []byte {
	if img == nil {
		return nil
	}
	w, h := img.Width, img.Height
	n := w * h
	if !mask.Valid() || mask.W != w || mask.H != h {
		return PackGray(img)
	}
	if op < 0 {
		op = 0
	} else if op > 100 {
		op = 100
	}
	dst := GetRGBA(4 * n)
	src := img.Data
	parallelRows(n, func(s, e int) {
		for i := s; i < e; i++ {
			v := src[i]
			grey := [3]uint8{v, v, v}
			c := grey
			if mask.Pix[i] != 0 {
				c = goldenBlend(grey, col, op)
			}
			o := 4 * i
			dst[o], dst[o+1], dst[o+2], dst[o+3] = c[0], c[1], c[2], 255
		}
	})
	return dst
}
