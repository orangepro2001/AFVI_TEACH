// Package golden: derivation of the inspection masks from the golden board.
// Standard library only.
package golden

// DefaultPadding is the default erosion applied to the Metal mask.
const DefaultPadding = 4

// DefaultSRInset is the default erosion/inset of the SR mask.
const DefaultSRInset = 4

// MaxPad is the largest padding / inset accepted.
const MaxPad = 64

// ClampPad clamps n into 0..MaxPad.
func ClampPad(n int) int {
	if n < 0 {
		return 0
	}
	if n > MaxPad {
		return MaxPad
	}
	return n
}

// InspectMetal returns the inspected Metal region: the Metal mask eroded by
// the clamped padding. A nil mask yields nil.
func InspectMetal(metal *Mask, padding int) *Mask {
	if metal == nil {
		return nil
	}
	return Erode(metal, ClampPad(padding))
}

// AutoSR returns the automatic SR mask: srReal eroded by the clamped inset and
// kept at least inset away from metal. A nil metal yields only the erosion.
func AutoSR(srReal, metal *Mask, inset int) *Mask {
	inset = ClampPad(inset)
	if srReal == nil {
		return nil
	}
	er := Erode(srReal, inset)
	if metal == nil || !metal.Valid() {
		return er
	}
	return AndNot(er, Dilate(metal, inset))
}

// Material builds the inspection material map over the design grid: 3 inside
// metalInspect, else 1 inside sr, else 0. Nil or invalid masks count as empty;
// two non-nil valid masks of different sizes yield nil; both empty yield nil.
func Material(metalInspect, sr *Mask) []uint8 {
	hasMetal := metalInspect != nil && metalInspect.Valid()
	hasSR := sr != nil && sr.Valid()
	if hasMetal && hasSR && (metalInspect.W != sr.W || metalInspect.H != sr.H) {
		return nil
	}
	if !hasMetal && !hasSR {
		return nil
	}
	w, h := 0, 0
	if hasMetal {
		w, h = metalInspect.W, metalInspect.H
	} else {
		w, h = sr.W, sr.H
	}
	out := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		row := y * w
		for x := 0; x < w; x++ {
			if hasMetal && metalInspect.Pix[row+x] != 0 {
				out[row+x] = 3
			} else if hasSR && sr.Pix[row+x] != 0 {
				out[row+x] = 1
			}
		}
	}
	return out
}

// MaskFromMaterial returns a mask that is inside where material[i] == mat. A
// wrong length yields an empty w*h mask.
func MaskFromMaterial(material []uint8, w, h int, mat uint8) *Mask {
	m := NewMask(w, h)
	if len(material) != w*h {
		return m
	}
	for i, v := range material {
		if v == mat {
			m.Pix[i] = 1
		}
	}
	return m
}

// AutoRange computes the threshold between the metal and the other class. The
// class interiors are the masks eroded by 3 (falling back to the mask itself
// when the interior is empty). lo is the rounded midpoint of the mean greys of
// the interiors, clamped to 0..255. ok is false (and lo = 128) when a class is
// empty or the means differ by less than 4 grey values.
func AutoRange(pix []uint8, w, h int, metal, other *Mask) (lo uint8, ok bool) {
	if w <= 0 || h <= 0 || len(pix) < w*h {
		return 128, false
	}
	meanMetal, okMetal := classMean(pix, w, h, metal)
	meanOther, okOther := classMean(pix, w, h, other)
	if !okMetal || !okOther {
		return 128, false
	}
	diff := meanMetal - meanOther
	if diff < 0 {
		diff = -diff
	}
	if diff < 4 {
		return 128, false
	}
	mid := int((meanMetal+meanOther)/2 + 0.5)
	if mid < 0 {
		mid = 0
	}
	if mid > 255 {
		mid = 255
	}
	return uint8(mid), true
}

// classMean returns the mean grey of the interior of a class mask, falling back
// to the mask itself when the interior is empty.
func classMean(pix []uint8, w, h int, m *Mask) (float64, bool) {
	if m == nil || !m.Valid() || m.W != w || m.H != h {
		return 0, false
	}
	if mean, ok := meanOf(pix, w, h, Erode(m, 3)); ok {
		return mean, true
	}
	return meanOf(pix, w, h, m)
}

// meanOf returns the mean grey of the set pixels of m.
func meanOf(pix []uint8, w, h int, m *Mask) (float64, bool) {
	if m == nil || !m.Valid() || m.W != w || m.H != h {
		return 0, false
	}
	sum := 0
	n := 0
	for i := 0; i < w*h; i++ {
		if m.Pix[i] != 0 {
			sum += int(pix[i])
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return float64(sum) / float64(n), true
}
