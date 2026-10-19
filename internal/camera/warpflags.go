package camera

import (
	"flag"
	"fmt"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/snap"
)

// warpFlags are the command line flags that place and deform the product in the capture:
// misalignment (-dx -dy -rot -scale -shear-x -shear-y) and lift (-tilt-x -tilt-y -bend -bend-wave).
// Only the flags that were given on the command line change the recipe's warp, so they also
// work together with -in (re-render a description with another position).
type warpFlags struct {
	dx, dy, rot, scale, shearX, shearY *float64
	tiltX, tiltY, bend, bendWave       *float64
}

func newWarpFlags(fs *flag.FlagSet) *warpFlags {
	return &warpFlags{
		dx:       fs.Float64("dx", 0, "misalignment: shift of the product in x (pixels, positive = right)"),
		dy:       fs.Float64("dy", 0, "misalignment: shift of the product in y (pixels, positive = down)"),
		rot:      fs.Float64("rot", 0, "misalignment: rotation about the image centre (degrees, ±15)"),
		scale:    fs.Float64("scale", 1, "misalignment: scale about the image centre (0.8..1.2)"),
		shearX:   fs.Float64("shear-x", 0, "deformation: x shear (±0.2)"),
		shearY:   fs.Float64("shear-y", 0, "deformation: y shear (±0.2)"),
		tiltX:    fs.Float64("tilt-x", 0, "lift: perspective tilt across x (±0.5; positive shrinks the right side and stretches the left)"),
		tiltY:    fs.Float64("tilt-y", 0, "lift: perspective tilt across y (±0.5; positive shrinks the bottom and stretches the top)"),
		bend:     fs.Float64("bend", 0, "lift: bend (warpage) amplitude in pixels (±200)"),
		bendWave: fs.Float64("bend-wave", 2000, "lift: bend wavelength in pixels (>= 50)"),
	}
}

// apply returns recipe with the warp flags that were set on fs applied to its warp; the result is
// validated. With no warp flag the recipe is returned unchanged.
func (f *warpFlags) apply(fs *flag.FlagSet, recipe snap.Recipe) (snap.Recipe, error) {
	set := make(map[string]bool)
	fs.Visit(func(fl *flag.Flag) { set[fl.Name] = true })
	targets := []struct {
		name string
		src  *float64
		dst  func(w *core.WarpParams) *float64
	}{
		{"dx", f.dx, func(w *core.WarpParams) *float64 { return &w.Tx }},
		{"dy", f.dy, func(w *core.WarpParams) *float64 { return &w.Ty }},
		{"rot", f.rot, func(w *core.WarpParams) *float64 { return &w.ThetaDeg }},
		{"scale", f.scale, func(w *core.WarpParams) *float64 { return &w.Scale }},
		{"shear-x", f.shearX, func(w *core.WarpParams) *float64 { return &w.ShearX }},
		{"shear-y", f.shearY, func(w *core.WarpParams) *float64 { return &w.ShearY }},
		{"tilt-x", f.tiltX, func(w *core.WarpParams) *float64 { return &w.TiltX }},
		{"tilt-y", f.tiltY, func(w *core.WarpParams) *float64 { return &w.TiltY }},
		{"bend", f.bend, func(w *core.WarpParams) *float64 { return &w.BendAmplitude }},
		{"bend-wave", f.bendWave, func(w *core.WarpParams) *float64 { return &w.BendWavelength }},
	}
	warp := recipe.CoreWarp()
	changed := false
	for _, t := range targets {
		if set[t.name] {
			*t.dst(&warp) = *t.src
			changed = true
		}
	}
	if !changed {
		return recipe, nil
	}
	if err := core.ValidateCaptureWarp(warp); err != nil {
		return recipe, fmt.Errorf("%w (see -dx -dy -rot -scale -shear-x -shear-y -tilt-x -tilt-y -bend -bend-wave)", err)
	}
	return recipe.WithWarp(warp), nil
}
