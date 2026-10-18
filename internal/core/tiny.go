package core

import "math"

// DefaultTinyCount is the number of tiny bright specks the default scenario adds.
const DefaultTinyCount = 10

// DefaultTinySeed is the seed of the default tiny specks.
const DefaultTinySeed uint32 = 7

// DefaultTinyEdgeMargin is the number of pixels kept free around each speck.
const DefaultTinyEdgeMargin = 6

// GoodSampleNoise is the noise of the defect-free good sample used for learning: the same
// parameters with Seed+1000, so learned intervals do not depend on the test image's own noise.
func GoodSampleNoise(n SensorNoise) SensorNoise {
	n.Seed += 1000
	return n
}

// InteriorMaterial reports whether the whole (2r+1)x(2r+1) window around (x,y) lies inside the
// width x height image and every material value in it equals mat (r < 0 or an undersized
// material slice returns false).
func InteriorMaterial(material []uint8, width, height, x, y, r int, mat uint8) bool {
	if r < 0 || width <= 0 || height <= 0 {
		return false
	}
	if len(material) < width*height {
		return false
	}
	if x-r < 0 || y-r < 0 || x+r >= width || y+r >= height {
		return false
	}
	for dy := -r; dy <= r; dy++ {
		row := (y + dy) * width
		for dx := -r; dx <= r; dx++ {
			if material[row+x+dx] != mat {
				return false
			}
		}
	}
	return true
}

// DefaultTinyDefects returns up to count tiny bright specks placed deep inside the SR material.
func DefaultTinyDefects(material []uint8, width, height, count int, seed uint32, edgeMargin int) []DefectStroke {
	out := make([]DefectStroke, 0)
	if count <= 0 {
		return out
	}
	candidates := GenerateRandomDefects(RandomDefectOptions{
		Count:     count * 12,
		Seed:      seed,
		Width:     width,
		Height:    height,
		Kind:      RandomBright,
		Shape:     RandomSpot,
		RadiusMin: 1.5,
		RadiusMax: 3,
		ValueMin:  45,
		ValueMax:  90,
		Amplitude: AmpAbsolute,
		Profile:   ProfileGaussian,
		Softness:  0.3,
		Grain:     0,
		Material:  material,
		Materials: []uint8{MatSR},
	})
	for i := range candidates {
		if len(out) >= count {
			break
		}
		s := candidates[i]
		r := edgeMargin + int(math.Ceil(s.Radius)) + 3
		if !InteriorMaterial(material, width, height, int(math.Round(s.X)), int(math.Round(s.Y)), r, MatSR) {
			continue
		}
		out = append(out, s)
	}
	return out
}
