package app

import "github.com/AllenDang/cimgui-go/imgui"

// Theme names accepted by ApplyTheme and the -theme flag.
const (
	ThemeDark  = "dark"
	ThemeLight = "light"
)

// Palette holds theme-dependent colors used by custom drawing (viewer, charts).
type Palette struct {
	Workspace  uint32
	ViewerEdge uint32
	ChartBg    uint32
	ChartGrid  uint32
	ChartBar   uint32
	ChartMark  uint32
	ChartFrame uint32
	TextDim    imgui.Vec4
	Warn       imgui.Vec4
	OK         imgui.Vec4
	NG         imgui.Vec4

	// Classic Win32 3D system colors (ABGR, packed with rgba).
	Face        uint32
	Highlight   uint32
	Light       uint32
	Shadow      uint32
	DkShadow    uint32
	Field       uint32
	Select      uint32
	SelectText  uint32
	CaptionL    uint32
	CaptionR    uint32
	CaptionText uint32
}

// Overlay colors are classic machine-vision primaries, identical in both themes because
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

var palette Palette

// CurrentPalette returns colors for the active theme.
func CurrentPalette() *Palette { return &palette }

// rgba packs IM_COL32 (ABGR) in pure Go: package-level colors are built before an ImGui
// context exists, and ImGui's GetColorU32 would dereference the missing context.
func rgba(r, g, b, a int) uint32 {
	return uint32(a)<<24 | uint32(b)<<16 | uint32(g)<<8 | uint32(r)
}

func rgb4(r, g, b int) imgui.Vec4 {
	return imgui.NewVec4(float32(r)/255, float32(g)/255, float32(b)/255, 1)
}

// ApplyTheme sets the ImGui style: flat industrial control panel, zero rounding,
// 1 px borders, dense spacing; dark (night) or light (classic Windows gray).
func ApplyTheme(name string) {
	style := imgui.CurrentStyle()
	if name == ThemeLight {
		imgui.StyleColorsLight()
	} else {
		imgui.StyleColorsDark()
	}
	style.SetWindowRounding(0)
	style.SetChildRounding(0)
	style.SetFrameRounding(0)
	style.SetPopupRounding(0)
	style.SetScrollbarRounding(0)
	style.SetGrabRounding(0)
	style.SetTabRounding(0)
	style.SetWindowBorderSize(1)
	style.SetChildBorderSize(1)
	style.SetFrameBorderSize(1)
	style.SetPopupBorderSize(1)
	style.SetWindowPadding(imgui.NewVec2(6, 6))
	style.SetFramePadding(imgui.NewVec2(4, 3))
	style.SetItemSpacing(imgui.NewVec2(6, 4))
	style.SetScrollbarSize(16)
	style.SetGrabMinSize(11)
	style.SetWindowMinSize(imgui.NewVec2(8, 8))
	c := style.Colors()
	if name == ThemeLight {
		palette = Palette{
			Workspace: rgba(128, 128, 128, 255), ViewerEdge: rgba(64, 64, 64, 255),
			ChartBg: rgba(255, 255, 255, 255), ChartGrid: rgba(192, 192, 192, 255),
			ChartBar: rgba(0, 0, 128, 255), ChartMark: rgba(255, 0, 0, 255), ChartFrame: rgba(128, 128, 128, 255),
			TextDim: rgb4(64, 64, 64), Warn: rgb4(128, 96, 0), OK: rgb4(0, 128, 0), NG: rgb4(192, 0, 0),

			Face: rgba(212, 208, 200, 255), Highlight: rgba(255, 255, 255, 255),
			Light: rgba(232, 229, 222, 255), Shadow: rgba(128, 128, 128, 255), DkShadow: rgba(64, 64, 64, 255),
			Field: rgba(255, 255, 255, 255), Select: rgba(10, 36, 106, 255), SelectText: rgba(255, 255, 255, 255),
			CaptionL: rgba(10, 36, 106, 255), CaptionR: rgba(166, 202, 240, 255), CaptionText: rgba(255, 255, 255, 255),
		}
		c[imgui.ColText] = rgb4(0, 0, 0)
		c[imgui.ColTextDisabled] = rgb4(128, 128, 128)
		c[imgui.ColWindowBg] = rgb4(212, 208, 200)
		c[imgui.ColChildBg] = rgb4(212, 208, 200)
		c[imgui.ColPopupBg] = rgb4(212, 208, 200)
		c[imgui.ColBorder] = rgb4(128, 128, 128)
		c[imgui.ColBorderShadow] = imgui.NewVec4(0, 0, 0, 0)
		c[imgui.ColFrameBg] = rgb4(255, 255, 255)
		c[imgui.ColFrameBgHovered] = rgb4(255, 255, 255)
		c[imgui.ColFrameBgActive] = rgb4(255, 255, 255)
		c[imgui.ColTitleBg] = rgb4(212, 208, 200)
		c[imgui.ColTitleBgActive] = rgb4(212, 208, 200)
		c[imgui.ColTitleBgCollapsed] = rgb4(212, 208, 200)
		c[imgui.ColButton] = rgb4(212, 208, 200)
		c[imgui.ColButtonHovered] = rgb4(212, 208, 200)
		c[imgui.ColButtonActive] = rgb4(212, 208, 200)
		// Selected/hovered list rows keep black text, so they use a soft selection blue;
		// menus and dialog lists repaint with the navy Select color + white text instead.
		c[imgui.ColHeader] = rgb4(166, 184, 214)
		c[imgui.ColHeaderHovered] = rgb4(186, 202, 228)
		c[imgui.ColHeaderActive] = rgb4(150, 170, 205)
		c[imgui.ColSliderGrab] = rgb4(212, 208, 200)
		c[imgui.ColSliderGrabActive] = rgb4(232, 229, 222)
		c[imgui.ColCheckMark] = rgb4(0, 0, 0)
		c[imgui.ColTableHeaderBg] = rgb4(212, 208, 200)
		c[imgui.ColTableBorderStrong] = rgb4(128, 128, 128)
		c[imgui.ColTableBorderLight] = rgb4(128, 128, 128)
		c[imgui.ColTableRowBgAlt] = imgui.NewVec4(0, 0, 0, 0.04)
		c[imgui.ColScrollbarBg] = rgb4(232, 229, 222)
		c[imgui.ColScrollbarGrab] = rgb4(212, 208, 200)
		c[imgui.ColScrollbarGrabHovered] = rgb4(212, 208, 200)
		c[imgui.ColScrollbarGrabActive] = rgb4(212, 208, 200)
		c[imgui.ColModalWindowDimBg] = imgui.NewVec4(0, 0, 0, 0.30)
		c[imgui.ColMenuBarBg] = rgb4(212, 208, 200)
		c[imgui.ColTextSelectedBg] = imgui.NewVec4(10.0/255, 36.0/255, 106.0/255, 0.60)
		c[imgui.ColSeparator] = rgb4(128, 128, 128)
	} else {
		palette = Palette{
			Workspace: rgba(43, 43, 43, 255), ViewerEdge: rgba(16, 16, 16, 255),
			ChartBg: rgba(0, 0, 0, 255), ChartGrid: rgba(30, 58, 30, 255),
			ChartBar: rgba(0, 192, 0, 255), ChartMark: rgba(255, 255, 0, 255), ChartFrame: rgba(64, 64, 64, 255),
			TextDim: rgb4(168, 168, 168), Warn: rgb4(224, 176, 48), OK: rgb4(60, 200, 60), NG: rgb4(255, 80, 80),

			Face: rgba(58, 58, 58, 255), Highlight: rgba(112, 112, 112, 255),
			Light: rgba(84, 84, 84, 255), Shadow: rgba(24, 24, 24, 255), DkShadow: rgba(8, 8, 8, 255),
			Field: rgba(28, 28, 28, 255), Select: rgba(29, 52, 97, 255), SelectText: rgba(255, 255, 255, 255),
			CaptionL: rgba(29, 52, 97, 255), CaptionR: rgba(70, 100, 150, 255), CaptionText: rgba(255, 255, 255, 255),
		}
		c[imgui.ColText] = rgb4(224, 224, 224)
		c[imgui.ColTextDisabled] = rgb4(130, 130, 130)
		c[imgui.ColWindowBg] = rgb4(58, 58, 58)
		c[imgui.ColChildBg] = rgb4(58, 58, 58)
		c[imgui.ColPopupBg] = rgb4(58, 58, 58)
		c[imgui.ColBorder] = rgb4(24, 24, 24)
		c[imgui.ColBorderShadow] = imgui.NewVec4(0, 0, 0, 0)
		c[imgui.ColFrameBg] = rgb4(28, 28, 28)
		c[imgui.ColFrameBgHovered] = rgb4(28, 28, 28)
		c[imgui.ColFrameBgActive] = rgb4(28, 28, 28)
		c[imgui.ColTitleBg] = rgb4(58, 58, 58)
		c[imgui.ColTitleBgActive] = rgb4(58, 58, 58)
		c[imgui.ColTitleBgCollapsed] = rgb4(58, 58, 58)
		c[imgui.ColButton] = rgb4(58, 58, 58)
		c[imgui.ColButtonHovered] = rgb4(58, 58, 58)
		c[imgui.ColButtonActive] = rgb4(58, 58, 58)
		c[imgui.ColHeader] = rgb4(29, 52, 97)
		c[imgui.ColHeaderHovered] = rgb4(29, 52, 97)
		c[imgui.ColHeaderActive] = rgb4(29, 52, 97)
		c[imgui.ColSliderGrab] = rgb4(58, 58, 58)
		c[imgui.ColSliderGrabActive] = rgb4(84, 84, 84)
		c[imgui.ColCheckMark] = rgb4(224, 224, 224)
		c[imgui.ColTableHeaderBg] = rgb4(58, 58, 58)
		c[imgui.ColTableBorderStrong] = rgb4(24, 24, 24)
		c[imgui.ColTableBorderLight] = rgb4(24, 24, 24)
		c[imgui.ColTableRowBgAlt] = imgui.NewVec4(1, 1, 1, 0.03)
		c[imgui.ColScrollbarBg] = rgb4(84, 84, 84)
		c[imgui.ColScrollbarGrab] = rgb4(58, 58, 58)
		c[imgui.ColScrollbarGrabHovered] = rgb4(58, 58, 58)
		c[imgui.ColScrollbarGrabActive] = rgb4(58, 58, 58)
		c[imgui.ColModalWindowDimBg] = imgui.NewVec4(0, 0, 0, 0.30)
		c[imgui.ColMenuBarBg] = rgb4(58, 58, 58)
		c[imgui.ColTextSelectedBg] = imgui.NewVec4(29.0/255, 52.0/255, 97.0/255, 0.60)
		c[imgui.ColSeparator] = rgb4(24, 24, 24)
	}
	style.SetColors(&c)
}
