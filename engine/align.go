package engine

import (
	"afvi/teach-imgui/internal/core"
)

// The alignment page: the desktop program warps the design plate by the parameters the user edits
// (core.ApplyWarp) and then finds the machine's ROIs in the warped picture with core.AlignCapture.
// The web copy needs exactly those two steps.

// WarpRequest is the warp the alignment page edits (core.WarpParams).
type WarpRequest struct {
	Dx, Dy   float64 `json:"dx"`
	Rot      float64 `json:"rot"`
	Scale    float64 `json:"scale"`
	ShearX   float64 `json:"shearX"`
	ShearY   float64 `json:"shearY"`
	TiltX    float64 `json:"tiltX"`
	TiltY    float64 `json:"tiltY"`
	Bend     float64 `json:"bend"`
	BendWave float64 `json:"bendWave"`
}

// toCore maps the request to core's parameters.
func (w WarpRequest) toCore() core.WarpParams {
	scale := w.Scale
	if scale == 0 {
		scale = 1
	}
	return core.WarpParams{
		Tx: w.Dx, Ty: w.Dy, ThetaDeg: w.Rot, Scale: scale,
		ShearX: w.ShearX, ShearY: w.ShearY,
		TiltX: w.TiltX, TiltY: w.TiltY,
		BendAmplitude: w.Bend, BendWavelength: w.BendWave,
	}
}

// ROIRequest is one machine ROI in design coordinates.
type ROIRequest struct {
	Name     string  `json:"name"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	W        float64 `json:"w"`
	H        float64 `json:"h"`
	Occluded bool    `json:"occluded"`
}

// ROIResult is one ROI's outcome.
type ROIResult struct {
	Name  string  `json:"name"`
	X, Y  float64 `json:"x"`
	W, H  float64 `json:"w"`
	NCC   float64 `json:"ncc"`
	Error float64 `json:"error"`
	OK    bool    `json:"ok"`
}

// AlignResult is what the alignment page shows.
type AlignResult struct {
	OK      bool        `json:"ok"`
	Matched int         `json:"matched"`
	Total   int         `json:"total"`
	Dx      float64     `json:"dx"`
	Dy      float64     `json:"dy"`
	Rot     float64     `json:"rot"`
	Scale   float64     `json:"scale"`
	ROIs    []ROIResult `json:"rois"`
	Elapsed float64     `json:"elapsed"`
}

// AlignPlate warps the plate by w, aligns the ROIs on the warped picture, and reports both.
//
// This is the desktop page's flow with one difference: the "capture" is the warped plate itself,
// because the web copy has no camera image. Everything the numbers come from is the same code.
func AlignPlate(median *Gray, material []uint8, ch Channel, w WarpRequest, rois []ROIRequest, minOverlap float64) (*AlignResult, error) {
	scene, err := NewLightScene(median, material)
	if err != nil {
		return nil, err
	}
	tpl := scene.Render(DefaultLight(), ch) // the design picture ("imagined" template)
	template := &core.GrayImage{Width: tpl.W, Height: tpl.H, Data: tpl.Data}
	wp := w.toCore()
	capture := core.ApplyWarp(template, &wp, 1) // the warped input

	alignRois := make([]core.AlignRoi, 0, len(rois))
	for _, r := range rois {
		if r.W <= 0 || r.H <= 0 {
			continue
		}
		alignRois = append(alignRois, core.AlignRoi{
			Name: r.Name,
			Rect: core.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H},
		})
	}
	res := &AlignResult{Total: len(rois), Dx: wp.Tx, Dy: wp.Ty, Rot: wp.ThetaDeg, Scale: wp.Scale}
	if len(alignRois) < 2 {
		res.ROIs = nominalROIs(rois)
		return res, nil
	}
	cfg := core.DefaultAlignConfig
	if minOverlap > 0 && minOverlap < 1 {
		cfg.MinOverlap = minOverlap
	}
	out, err := core.AlignCapture(template, capture, alignRois, 1, cfg)
	if err != nil {
		return nil, err
	}
	res.Elapsed = out.ElapsedMs
	res.OK = out.Usable
	res.Matched = len(out.Report.Inliers)
	if out.Report.Fit != nil {
		res.Dx = out.Report.Fit.Tx
		res.Dy = out.Report.Fit.Ty
		res.Rot = out.Report.Fit.ThetaDeg
		res.Scale = out.Report.Fit.Scale
	}
	for _, m := range out.Report.Matches {
		rr := ROIResult{Name: m.Name, NCC: m.Ncc, OK: m.Located, W: 0, H: 0}
		if m.Found != nil {
			rr.X, rr.Y = m.Found.X, m.Found.Y
		}
		res.ROIs = append(res.ROIs, rr)
	}
	return res, nil
}

// nominalROIs reports the ROIs as given (used when there are too few to align).
func nominalROIs(rois []ROIRequest) []ROIResult {
	out := make([]ROIResult, 0, len(rois))
	for _, r := range rois {
		out = append(out, ROIResult{Name: r.Name, X: r.X, Y: r.Y, W: r.W, H: r.H})
	}
	return out
}
