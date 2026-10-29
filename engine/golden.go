package engine

import (
	"afvi/teach-imgui/internal/golden"
)

// The golden board: the inspection area of the plate. The desktop program keeps it in
// internal/golden (masks, morphology, Padding, the SR inset, UserMask); the web page draws the same
// layers in the same colours, so the operations are exposed here.

// GoldenMasks is what the golden page paints: the inspected metal area, the SR area, and the material
// map the inspection receives.
type GoldenMasks struct {
	MetalPixels int    `json:"metalPixels"`
	SRPixels    int    `json:"srPixels"`
	Mask        []byte `json:"mask"` // 1 = inspected metal
	SR          []byte `json:"sr"`   // 1 = inspected SR
	Material    []byte `json:"material"`
}

// GoldenAuto derives the inspection area the way the desktop page's "自动生成" does: the metal comes
// from the design material map, is eroded by padding (the machine's "Margine"), and the SR mask is
// the design SR eroded by the inset and kept away from the metal.
func GoldenAuto(material []uint8, w, h, padding, srInset int) *GoldenMasks {
	if w <= 0 || h <= 0 || len(material) != w*h {
		return nil
	}
	metal := golden.MaskFromMaterial(material, w, h, 3) // 3 = metal
	if metal == nil {
		return nil
	}
	metalInspect := golden.InspectMetal(metal, padding)
	srReal := golden.MaskFromMaterial(material, w, h, 1) // 1 = SR
	sr := golden.AutoSR(srReal, metal, srInset)
	return &GoldenMasks{
		MetalPixels: countMask(metalInspect),
		SRPixels:    countMask(sr),
		Mask:        maskBytes(metalInspect),
		SR:          maskBytes(sr),
		Material:    golden.Material(metalInspect, sr),
	}
}

// countMask returns the number of set pixels.
func countMask(m *golden.Mask) int {
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

// maskBytes flattens a mask for JSON (one byte per pixel, 0 or 1).
func maskBytes(m *golden.Mask) []byte {
	if m == nil {
		return nil
	}
	out := make([]byte, len(m.Pix))
	for i, v := range m.Pix {
		if v != 0 {
			out[i] = 1
		}
	}
	return out
}
