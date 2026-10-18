package core

import (
	"math"
	"runtime"
	"sync"
)

// ProductMaterial is the surface non-uniformity of one material of a virtual good unit.
type ProductMaterial struct {
	Sigma  float64 // relative std (fraction of the local brightness) of the spatially correlated part
	Length float64 // correlation length in pixels of the coarsest octave (>= 2)
	Grain  float64 // relative std of the uncorrelated per-pixel part
}

// ProductOptions describes one virtual good unit.
type ProductOptions struct {
	Enabled   bool
	Seed      uint32
	Strength  float64            // global multiplier of every Sigma and Grain (1 = nominal, 0 = ideal uniform unit)
	Materials [4]ProductMaterial // indexed by MatBase, MatSR, MatOpen, MatMetal
}

// DefaultProductOptions returns the nominal virtual good unit.
func DefaultProductOptions() ProductOptions {
	return ProductOptions{
		Enabled:  true,
		Seed:     1,
		Strength: 1,
		Materials: [4]ProductMaterial{
			{Sigma: 0.03, Length: 32, Grain: 0.02},
			{Sigma: 0.10, Length: 24, Grain: 0.06},
			{Sigma: 0.04, Length: 32, Grain: 0.03},
			{Sigma: 0.04, Length: 20, Grain: 0.03},
		},
	}
}

// valueNoise is lattice value noise with smoothstep interpolation.
func valueNoise(x, y float64, cell float64, seed uint32) float64 {
	fx := x / cell
	fy := y / cell
	ix := int(math.Floor(fx))
	iy := int(math.Floor(fy))
	tx := fx - float64(ix)
	ty := fy - float64(iy)
	sx := tx * tx * (3 - 2*tx)
	sy := ty * ty * (3 - 2*ty)
	v00 := hashGrain(ix, iy, seed)
	v10 := hashGrain(ix+1, iy, seed)
	v01 := hashGrain(ix, iy+1, seed)
	v11 := hashGrain(ix+1, iy+1, seed)
	a := v00 + (v10-v00)*sx
	b := v01 + (v11-v01)*sx
	return a + (b-a)*sy
}

// productMaterialConsts holds the precomputed per-material constants.
type productMaterialConsts struct {
	sigma float64
	grain float64
	cell0 float64
	cell1 float64
	cell2 float64
	seed0 uint32
	seed1 uint32
	seed2 uint32
}

// ApplyProductVariation multiplies, in place, every s.Texture[i] by a deterministic
// spatially correlated variation of the virtual good unit described by o, then clamps
// the result to [0, 4] (same clamp as NewLightScene). It does nothing when s == nil,
// !o.Enabled or o.Strength <= 0.
func ApplyProductVariation(s *LightScene, o ProductOptions) {
	if s == nil || !o.Enabled || o.Strength <= 0 {
		return
	}
	var consts [4]productMaterialConsts
	for m := 0; m < 4; m++ {
		mat := o.Materials[m]
		length := mat.Length
		if length < 2 {
			length = 2
		}
		consts[m] = productMaterialConsts{
			sigma: mat.Sigma * o.Strength,
			grain: mat.Grain * o.Strength,
			cell0: math.Max(2, length/1),
			cell1: math.Max(2, length/2),
			cell2: math.Max(2, length/4),
			seed0: o.Seed*7919 + 0*104729 + 17,
			seed1: o.Seed*7919 + 1*104729 + 17,
			seed2: o.Seed*7919 + 2*104729 + 17,
		}
	}
	grainSeed := o.Seed*6151 + 99991
	width := s.Width
	height := s.Height
	if width <= 0 || height <= 0 {
		return
	}
	norm := 0.429 * math.Sqrt(1+0.25+0.0625)
	applyRows := func(y0, y1 int) {
		for y := y0; y < y1; y++ {
			row := y * width
			fy := float64(y)
			for x := 0; x < width; x++ {
				idx := row + x
				m := s.Material[idx] & 3
				c := consts[m]
				fx := float64(x)
				n0 := valueNoise(fx, fy, c.cell0, c.seed0)
				n1 := valueNoise(fx, fy, c.cell1, c.seed1)
				n2 := valueNoise(fx, fy, c.cell2, c.seed2)
				corr := (n0 + 0.5*n1 + 0.25*n2) / norm
				g := math.Sqrt(3) * hashGrain(x, y, grainSeed)
				v := float64(s.Texture[idx]) * (1 + c.sigma*corr + c.grain*g)
				if v < 0 {
					v = 0
				} else if v > 4 {
					v = 4
				}
				s.Texture[idx] = float32(v)
			}
		}
	}
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > height {
		workers = height
	}
	if workers <= 1 {
		applyRows(0, height)
		return
	}
	rowsPer := (height + workers - 1) / workers
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		y0 := w * rowsPer
		y1 := y0 + rowsPer
		if y1 > height {
			y1 = height
		}
		if y0 >= y1 {
			break
		}
		wg.Add(1)
		go func(y0, y1 int) {
			defer wg.Done()
			applyRows(y0, y1)
		}(y0, y1)
	}
	wg.Wait()
}

// NewProductLightScene builds a light scene and bakes the virtual good unit into it.
func NewProductLightScene(median *GrayImage, material []uint8, o ProductOptions) (*LightScene, error) {
	scene, err := NewLightScene(median, material)
	if err != nil {
		return nil, err
	}
	ApplyProductVariation(scene, o)
	return scene, nil
}
