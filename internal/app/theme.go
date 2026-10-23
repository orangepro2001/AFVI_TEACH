package app

import "afvi/teach-imgui/internal/ui"

// Theme names accepted by the -theme flag and the View menu.
const (
	ThemeDark  = "dark"
	ThemeLight = "light"
)

// Palette holds the theme-dependent colours the Go side still needs (chart drawing that
// is sent as data, overlay colours). The widget skin itself is CSS (frontend/src/style.css).
type Palette struct {
	Workspace  uint32
	ViewerEdge uint32
	ChartBg    uint32
	ChartGrid  uint32
	ChartBar   uint32
	ChartMark  uint32
	ChartFrame uint32
	Face       uint32
	Highlight  uint32
	Light      uint32
	Shadow     uint32
	DkShadow   uint32
	Field      uint32
	Select     uint32
	SelectText uint32
}

// Overlay colours are classic machine-vision primaries, identical in both themes because
// they are drawn on top of the inspected image.
var (
	OverlayOK     = rgba(0, 255, 0, 255)
	OverlayNG     = rgba(255, 0, 0, 255)
	OverlayWarn   = rgba(255, 255, 0, 255)
	OverlayROI    = rgba(0, 255, 255, 255)
	OverlayMuted  = rgba(128, 128, 128, 255)
	OverlayProbe  = rgba(255, 255, 255, 255)
	OverlayNGFill = rgba(255, 0, 0, 64)
)

var palette = lightPalette

var lightPalette = Palette{
	Workspace: rgba(128, 128, 128, 255), ViewerEdge: rgba(64, 64, 64, 255),
	ChartBg: rgba(255, 255, 255, 255), ChartGrid: rgba(192, 192, 192, 255),
	ChartBar: rgba(0, 0, 128, 255), ChartMark: rgba(255, 0, 0, 255), ChartFrame: rgba(128, 128, 128, 255),
	Face: rgba(212, 208, 200, 255), Highlight: rgba(255, 255, 255, 255),
	Light: rgba(232, 229, 222, 255), Shadow: rgba(128, 128, 128, 255), DkShadow: rgba(64, 64, 64, 255),
	Field: rgba(255, 255, 255, 255), Select: rgba(10, 36, 106, 255), SelectText: rgba(255, 255, 255, 255),
}

var darkPalette = Palette{
	Workspace: rgba(43, 43, 43, 255), ViewerEdge: rgba(16, 16, 16, 255),
	ChartBg: rgba(0, 0, 0, 255), ChartGrid: rgba(30, 58, 30, 255),
	ChartBar: rgba(0, 192, 0, 255), ChartMark: rgba(255, 255, 0, 255), ChartFrame: rgba(64, 64, 64, 255),
	Face: rgba(58, 58, 58, 255), Highlight: rgba(112, 112, 112, 255),
	Light: rgba(84, 84, 84, 255), Shadow: rgba(24, 24, 24, 255), DkShadow: rgba(8, 8, 8, 255),
	Field: rgba(28, 28, 28, 255), Select: rgba(29, 52, 97, 255), SelectText: rgba(255, 255, 255, 255),
}

// CurrentPalette returns colours for the active theme.
func CurrentPalette() *Palette { return &palette }

// rgba packs IM_COL32 (ABGR); kept so the colour tables of the old UI read unchanged.
func rgba(r, g, b, a int) uint32 {
	return uint32(a)<<24 | uint32(b)<<16 | uint32(g)<<8 | uint32(r)
}

// ApplyTheme selects the palette for custom drawing.
func ApplyTheme(name string) {
	if name == ThemeLight {
		palette = lightPalette
	} else {
		palette = darkPalette
	}
}

// css converts a packed colour to a CSS string.
func css(c uint32) string { return ui.Col(c) }
