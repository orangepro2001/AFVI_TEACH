package core

// PaintedTemplate builds an alignment template from a class map that already sits on the geometry of ref (for example the masks of a golden board):
// DesignImage(classes, ref) paints every pixel with the mean grey value its class has in ref, the result is area-downsampled by scale (scale < 1 counts as 1; scale 1 skips
// DownsampleArea) and blurred with sigma DesignBlurPx/scale (the blur is skipped when that sigma is below 0.3), exactly like DesignTemplate does, but WITHOUT the registration
// step: the picture keeps the geometry of classes. Error when len(classes) != ref.Width*ref.Height (DesignImage reports it) or when DownsampleArea fails.
func PaintedTemplate(classes []uint8, ref *GrayImage, scale int) (*GrayImage, error) {
	if scale < 1 {
		scale = 1
	}
	full, err := DesignImage(classes, ref)
	if err != nil {
		return nil, err
	}
	out := full
	if scale > 1 {
		out, err = DownsampleArea(full, scale)
		if err != nil {
			return nil, err
		}
	}
	sigma := DesignBlurPx / float64(scale)
	if sigma >= 0.3 {
		out = GaussianBlur(out, sigma)
	}
	return out, nil
}
