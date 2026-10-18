package main

import (
	"math"

	"github.com/AllenDang/cimgui-go/imgui"
)

// EdgeKind selects one of the classic Win32 3D bevel recipes.
type EdgeKind int

const (
	EdgeRaised  EdgeKind = iota // push button, toolbar button, tab, panel
	EdgeSunken                  // edit field, list box, viewer client, status pane (2px)
	EdgePressed                 // pushed button: 1px DkShadow outline + 1px Shadow inner outline
	EdgeStatus                  // 1px sunken: top/left Shadow, bottom/right Highlight
	EdgeEtched                  // 2px etched line: Shadow rect, then Highlight rect offset (+1,+1)
	EdgeWindow                  // raised window/dialog frame
)

// snap floors a coordinate so bevel strips land on whole pixels.
func snap(v float32) float32 { return float32(math.Floor(float64(v))) }

// strip fills the half-open rectangle [x0,x1) x [y0,y1) with col.
func strip(dl *imgui.DrawList, x0, y0, x1, y1 float32, col uint32) {
	if x1 <= x0 || y1 <= y0 {
		return
	}
	dl.AddRectFilled(imgui.NewVec2(x0, y0), imgui.NewVec2(x1, y1), col)
}

// ring draws a 1px bevel ring inside [min,max): top, left, bottom, right strips.
// Corners belong to the top and bottom strips so rings stay crisp.
func ring(dl *imgui.DrawList, min, max imgui.Vec2, topLeft, bottomRight uint32) {
	strip(dl, min.X, min.Y, max.X, min.Y+1, topLeft)
	strip(dl, min.X, min.Y+1, min.X+1, max.Y-1, topLeft)
	strip(dl, min.X, max.Y-1, max.X, max.Y, bottomRight)
	strip(dl, max.X-1, min.Y+1, max.X, max.Y-1, bottomRight)
}

// FillRect fills [min,max) with a solid color.
func FillRect(dl *imgui.DrawList, min, max imgui.Vec2, col uint32) {
	min = imgui.NewVec2(snap(min.X), snap(min.Y))
	max = imgui.NewVec2(snap(max.X), snap(max.Y))
	strip(dl, min.X, min.Y, max.X, max.Y, col)
}

// DrawEdge draws a bevel inside [min,max) without filling the interior.
func DrawEdge(dl *imgui.DrawList, min, max imgui.Vec2, kind EdgeKind) {
	min = imgui.NewVec2(snap(min.X), snap(min.Y))
	max = imgui.NewVec2(snap(max.X), snap(max.Y))
	if max.X-min.X < 1 || max.Y-min.Y < 1 {
		return
	}
	p := CurrentPalette()
	switch kind {
	case EdgeRaised:
		ring(dl, min, max, p.Highlight, p.DkShadow)
		ring(dl, imgui.NewVec2(min.X+1, min.Y+1), imgui.NewVec2(max.X-1, max.Y-1), p.Light, p.Shadow)
	case EdgeSunken:
		ring(dl, min, max, p.Shadow, p.Highlight)
		ring(dl, imgui.NewVec2(min.X+1, min.Y+1), imgui.NewVec2(max.X-1, max.Y-1), p.DkShadow, p.Light)
	case EdgePressed:
		ring(dl, min, max, p.DkShadow, p.DkShadow)
		ring(dl, imgui.NewVec2(min.X+1, min.Y+1), imgui.NewVec2(max.X-1, max.Y-1), p.Shadow, p.Shadow)
	case EdgeStatus:
		strip(dl, min.X, min.Y, max.X, min.Y+1, p.Shadow)
		strip(dl, min.X, min.Y+1, min.X+1, max.Y-1, p.Shadow)
		strip(dl, min.X, max.Y-1, max.X, max.Y, p.Highlight)
		strip(dl, max.X-1, min.Y+1, max.X, max.Y-1, p.Highlight)
	case EdgeEtched:
		// Outlines only: Highlight ring at (+1,+1), then Shadow ring at (0,0) so the
		// dark line wins on the top/left. Never fill the interior.
		ring(dl, imgui.NewVec2(min.X+1, min.Y+1), max, p.Highlight, p.Highlight)
		ring(dl, min, imgui.NewVec2(max.X-1, max.Y-1), p.Shadow, p.Shadow)
	case EdgeWindow:
		ring(dl, min, max, p.Light, p.DkShadow)
		ring(dl, imgui.NewVec2(min.X+1, min.Y+1), imgui.NewVec2(max.X-1, max.Y-1), p.Highlight, p.Shadow)
	}
}

// ItemEdge draws a bevel over the last item's rectangle.
func ItemEdge(kind EdgeKind) {
	DrawEdge(imgui.WindowDrawList(), imgui.ItemRectMin(), imgui.ItemRectMax(), kind)
}

// GradientBar fills [min,max) with a horizontal left→right gradient.
func GradientBar(dl *imgui.DrawList, min, max imgui.Vec2, left, right uint32) {
	min = imgui.NewVec2(snap(min.X), snap(min.Y))
	max = imgui.NewVec2(snap(max.X), snap(max.Y))
	dl.AddRectFilledMultiColor(min, max, left, right, right, left)
}

// Caption draws a Win32 title bar: gradient background plus white title text.
func Caption(dl *imgui.DrawList, min, max imgui.Vec2, title string, active bool) {
	p := CurrentPalette()
	l, r := p.CaptionL, p.CaptionR
	if !active {
		l, r = p.Shadow, p.Face
	}
	GradientBar(dl, min, max, l, r)
	ts := imgui.CalcTextSize(title)
	y := min.Y + (max.Y-min.Y-ts.Y)/2
	x := min.X + 3
	// Draw twice 1px apart to fake the bold Win32 caption font.
	dl.AddTextVec2(imgui.NewVec2(x, y), p.CaptionText, title)
	dl.AddTextVec2(imgui.NewVec2(x+1, y), p.CaptionText, title)
}

// HotText repaints a hovered menu/selectable row with the selection color and draws
// its label (and optional right-aligned shortcut) on top.
func HotText(label, shortcut string) {
	if !imgui.IsItemHovered() {
		return
	}
	p := CurrentPalette()
	min, max := imgui.ItemRectMin(), imgui.ItemRectMax()
	dl := imgui.WindowDrawList()
	FillRect(dl, min, max, p.Select)
	pad := imgui.CurrentStyle().FramePadding().X
	ts := imgui.CalcTextSize(label)
	y := min.Y + (max.Y-min.Y-ts.Y)/2
	dl.AddTextVec2(imgui.NewVec2(min.X+pad+2, y), p.SelectText, label)
	if shortcut != "" {
		ss := imgui.CalcTextSize(shortcut)
		dl.AddTextVec2(imgui.NewVec2(max.X-pad-ss.X, y), p.SelectText, shortcut)
	}
}

// DottedRect draws a 1px dotted focus rectangle along the four sides of [min,max).
func DottedRect(dl *imgui.DrawList, min, max imgui.Vec2, col uint32) {
	min = imgui.NewVec2(snap(min.X), snap(min.Y))
	max = imgui.NewVec2(snap(max.X), snap(max.Y))
	for x := min.X; x < max.X; x += 2 {
		strip(dl, x, min.Y, x+1, min.Y+1, col)
		strip(dl, x, max.Y-1, x+1, max.Y, col)
	}
	for y := min.Y; y < max.Y; y += 2 {
		strip(dl, min.X, y, min.X+1, y+1, col)
		strip(dl, max.X-1, y, max.X, y+1, col)
	}
}

// TextColor returns the current theme's text color as a packed ABGR uint32, for
// owner-drawn labels that must stay readable on both themes.
func TextColor() uint32 {
	c := imgui.CurrentStyle().Colors()
	return imgui.ColorU32Vec4(c[imgui.ColText])
}

// PaletteVec4 unpacks an ABGR uint32 palette color into an imgui.Vec4.
func PaletteVec4(col uint32) imgui.Vec4 {
	r := float32(col & 0xff)
	g := float32((col >> 8) & 0xff)
	b := float32((col >> 16) & 0xff)
	a := float32((col >> 24) & 0xff)
	return imgui.NewVec4(r/255, g/255, b/255, a/255)
}

// TextCenteredIn draws text centered inside [min,max).
func TextCenteredIn(dl *imgui.DrawList, min, max imgui.Vec2, text string, col uint32) {
	ts := imgui.CalcTextSize(text)
	pos := imgui.NewVec2(min.X+(max.X-min.X-ts.X)/2, min.Y+(max.Y-min.Y-ts.Y)/2)
	dl.AddTextVec2(pos, col, text)
}
