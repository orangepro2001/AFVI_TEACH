package core

// DesignBlurPx is the Gaussian sigma, in full-resolution pixels, applied to the imagined picture so
// its edges are as soft as the edges of a real picture.
const DesignBlurPx = 4.0

// DesignImage paints every pixel of the design with the mean grey value its material has in ref (the
// MEDIAN picture or a lighting render): a piecewise constant picture that carries the shapes of the
// product and nothing else (no texture, no noise). It is the full-resolution, unblurred picture.
// material has one entry (MatBase..MatMetal) per pixel of ref; a material without pixels gets 0.
// Levels are rounded with ClampByte(mean). Error when len(material) != ref.Width*ref.Height.
func DesignImage(material []uint8, ref *GrayImage) (*GrayImage, error) {
	stats, err := MaterialStats(ref, material)
	if err != nil {
		return nil, err
	}
	var level [4]uint8
	for m := 0; m < 4; m++ {
		level[m] = ClampByte(stats[m].Mean)
	}
	data := make([]uint8, len(material))
	for i, m := range material {
		if m < 4 {
			data[i] = level[m]
		}
	}
	return &GrayImage{Width: ref.Width, Height: ref.Height, Data: data}, nil
}

// DesignInfo says how DesignTemplate lined the imagined picture up with the reference.
type DesignInfo struct {
	Registered bool       // a fit was found and the picture was resampled with it
	Fit        Similarity // maps a point of ref to the matching point of the returned picture's source design, in pixels of the returned (working-resolution) image; {Scale: 1} when not Registered
	Inliers    int        // windows the fit agrees with
	Windows    int        // windows that were located (before RANSAC)
	Tried      int        // windows laid out on the reference
}

const (
	designCols = 6
	designRows = 5
)

// DesignTemplate builds the template for the Align tab at 1/scale resolution (scale < 1 counts as 1):
// the DesignImage of ref, area-downsampled by scale, blurred with sigma DesignBlurPx/scale (skip the blur
// when that sigma is below 0.3), and then registered to ref.
//
// Registration: the design layers and the real picture are never exactly on top of each other (on the
// reference model they differ by 2 to 6 full-resolution pixels of shift plus a little rotation), and the
// Align tab measures its truth against the MEDIAN, so the imagined picture is lined up with the
// downsampled ref once, here. Lay a grid of windows (6 columns x 5 rows) on the downsampled ref:
// side = min(w/16, h/12) integer, positions x0 = win + gx*(w-3*win)/(cols-1),
// y0 = win + gy*(h-3*win)/(rows-1); skip registration (Registered=false, no error) when side < 16.
// Match them with AlignWithRois(refSmall, designSmall, rois, cfg, Similarity{Scale: 1}) where cfg is a
// copy of DefaultAlignConfig with SearchRadius 32, MinNcc 0.4, MinUniqueness 0.05, MinOverlap 1,
// RansacThresholdPx 2, RansacIterations 300, and PyramidFactor 2 when side >= 64 else 1.
// (template = ref, input = design, so rep.Fit maps a ref point to the design point.)
// When rep.Fit != nil and len(rep.Inliers) >= 4: re := RealignInput(designSmall, *rep.Fit, nil) and the
// result is a copy of designSmall in which every pixel with re.Valid[i] is replaced by re.Image.Data[i]
// (pixels without data keep the unregistered design value, so there is no black border strip).
// Otherwise the (blurred) design is returned unregistered. Fill DesignInfo as documented.
// The returned image has the size of DownsampleArea(ref, scale). Error when len(material) != ref size
// or when DownsampleArea fails.
func DesignTemplate(material []uint8, ref *GrayImage, scale int) (*GrayImage, DesignInfo, error) {
	info := DesignInfo{Fit: Similarity{Scale: 1}}
	if scale < 1 {
		scale = 1
	}
	full, err := DesignImage(material, ref)
	if err != nil {
		return nil, info, err
	}
	refSmall := ref
	design := full
	if scale > 1 {
		design, err = DownsampleArea(full, scale)
		if err != nil {
			return nil, info, err
		}
		refSmall, err = DownsampleArea(ref, scale)
		if err != nil {
			return nil, info, err
		}
	}
	sigma := DesignBlurPx / float64(scale)
	if sigma >= 0.3 {
		design = GaussianBlur(design, sigma)
	}

	w, h := refSmall.Width, refSmall.Height
	side := w / 16
	if h/12 < side {
		side = h / 12
	}
	if side >= 16 {
		rois := make([]AlignRoi, 0, designCols*designRows)
		for gy := 0; gy < designRows; gy++ {
			for gx := 0; gx < designCols; gx++ {
				x0 := side + gx*(w-3*side)/(designCols-1)
				y0 := side + gy*(h-3*side)/(designRows-1)
				rois = append(rois, AlignRoi{
					Name: "design",
					Rect: Rect{X: float64(x0), Y: float64(y0), W: float64(side), H: float64(side)},
				})
			}
		}
		info.Tried = len(rois)
		cfg := DefaultAlignConfig
		cfg.SearchRadius = 32
		cfg.MinNcc = 0.4
		cfg.MinUniqueness = 0.05
		cfg.MinOverlap = 1
		cfg.RansacThresholdPx = 2
		cfg.RansacIterations = 300
		if side >= 64 {
			cfg.PyramidFactor = 2
		} else {
			cfg.PyramidFactor = 1
		}
		rep := AlignWithRois(refSmall, design, rois, cfg, Similarity{Scale: 1})
		for i := range rep.Matches {
			if rep.Matches[i].Located && rep.Matches[i].Found != nil {
				info.Windows++
			}
		}
		info.Inliers = len(rep.Inliers)
		if rep.Fit != nil && len(rep.Inliers) >= 4 {
			re := RealignInput(design, *rep.Fit, nil)
			out := design.Clone()
			for i := range out.Data {
				if i < len(re.Valid) && re.Valid[i] {
					out.Data[i] = re.Image.Data[i]
				}
			}
			info.Registered = true
			info.Fit = *rep.Fit
			return out, info, nil
		}
	}
	return design, info, nil
}
