package app

// Layout constants in this package were designed for a 13 px font. The UI font is larger
// now (easier on the eyes), so every hard-coded pixel length is passed through sc.
const (
	designFontPx  = 13
	defaultFontPx = 16
	minFontPx     = 10
	maxFontPx     = 28
)

// fontPx is the UI font size in pixels; set it once with SetFontPx before the first frame.
var fontPx float32 = defaultFontPx

// SetFontPx sets the UI font size, clamped to [minFontPx, maxFontPx].
func SetFontPx(px float32) {
	if px < minFontPx {
		px = minFontPx
	}
	if px > maxFontPx {
		px = maxFontPx
	}
	fontPx = px
}

// sc scales a pixel length that was designed for the 13 px font.
func sc(v float32) float32 { return v * fontPx / designFontPx }
