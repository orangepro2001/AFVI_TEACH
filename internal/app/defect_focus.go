package app

import (
	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
	"afvi/teach-imgui/internal/inspect"
)

// focusPolarity is the polarity filter of the defect page sub-tabs.
type focusPolarity int

// focusPolarity values.
const (
	focusAll    focusPolarity = iota // both polarities
	focusBright                      // inspect.Bright only
	focusDark                        // inspect.Dark only
)

// defectFocus is the sub-tab state of the detect page (zero value: Metal, all polarities).
type defectFocus struct {
	target   golden.Target
	polarity focusPolarity
}

// material returns the inspected material of the selected target.
func (f defectFocus) material() inspect.Material {
	if f.target == golden.TargetSR {
		return inspect.SR
	}
	return inspect.Metal
}

// match reports whether d is of the focused material and passes the polarity filter.
func (f defectFocus) match(d *inspect.Defect) bool {
	if d == nil || d.Material != f.material() {
		return false
	}
	switch f.polarity {
	case focusBright:
		return d.Polarity == inspect.Bright
	case focusDark:
		return d.Polarity == inspect.Dark
	}
	return true
}

// indices returns the indices into r.Defects of the matching defects, in report order. The
// result is a fresh slice (nil for a nil report).
func (f defectFocus) indices(r *inspect.Report) []int {
	if r == nil {
		return nil
	}
	var out []int
	for i := range r.Defects {
		if f.match(&r.Defects[i]) {
			out = append(out, i)
		}
	}
	return out
}

// focusCounts counts the defects of material m by polarity; all == bright + dark.
func focusCounts(r *inspect.Report, m inspect.Material) (all, bright, dark int) {
	if r == nil {
		return 0, 0, 0
	}
	for i := range r.Defects {
		d := &r.Defects[i]
		if d.Material != m {
			continue
		}
		all++
		if d.Polarity == inspect.Bright {
			bright++
		} else if d.Polarity == inspect.Dark {
			dark++
		}
	}
	return all, bright, dark
}

// focusColor returns the tint colour of the target: Metal magenta, SR green.
func focusColor(t golden.Target) [3]uint8 {
	if t == golden.TargetSR {
		return goldenSRColor
	}
	return [3]uint8{255, 0, 255}
}

// focusOpacity is the tint opacity in percent.
const focusOpacity = 35

// tintSpec is what the viewer picture is tinted with; the zero value means "no tint".
type tintSpec struct {
	mask   *golden.Mask
	col    [3]uint8
	target golden.Target
}

// on reports whether the picture has to be tinted.
func (t tintSpec) on() bool { return t.mask != nil }

// pack renders img, tinted when on().
func (t tintSpec) pack(img *core.GrayImage) []byte {
	if !t.on() {
		return PackGray(img)
	}
	return PackGrayTint(img, t.mask, t.col, focusOpacity)
}

// focusMask returns the final inspected mask of the target (nil for a nil receiver).
func (d *goldenDerived) focusMask(t golden.Target) *golden.Mask {
	if d == nil {
		return nil
	}
	if t == golden.TargetSR {
		return d.sr
	}
	return d.metalInspect
}

// newTintSpec returns the tint of the target mask in d; zero when d is nil or the mask is
// nil or invalid.
func newTintSpec(d *goldenDerived, t golden.Target) tintSpec {
	m := d.focusMask(t)
	if !m.Valid() {
		return tintSpec{}
	}
	return tintSpec{mask: m, col: focusColor(t), target: t}
}

// tintKey identifies what a packed picture was tinted with.
type tintKey struct {
	on     bool
	target golden.Target
	rev    int // goldenBoard.Rev() at the time
}
