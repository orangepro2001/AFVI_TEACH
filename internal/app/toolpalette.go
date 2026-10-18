package app

import (
	"math"

	"github.com/AllenDang/cimgui-go/imgui"
)

// PaletteTool is one tool of the floating viewer palette.
type PaletteTool int

const (
	PaletteNone PaletteTool = iota // no tool: left drags pan the view
	PalettePick                    // point select
	PaletteRect
	PaletteEllipse
	PalettePolygon
)

// PaletteHost is implemented by panels that use frame tools. The palette is only a view over
// the panel's own tool state, so a panel keeps full control of what each tool does.
type PaletteHost interface {
	PaletteSupports(t PaletteTool) bool // false => the button is greyed out and ignores clicks
	PaletteTool() PaletteTool           // currently active tool (PaletteNone if none)
	SetPaletteTool(t PaletteTool)       // user clicked an enabled button; PaletteNone = back to pan
}

// Palette geometry: five 34x34 buttons, 4 px padding, 2 px gaps, a 6 px separator.
const (
	paletteBtn     = 34
	palettePad     = 4
	paletteGap     = 2
	paletteSep     = 6
	paletteInset   = 8
	paletteButtons = 5
)

// paletteSize returns the palette panel size in pixels.
func paletteSize() imgui.Vec2 {
	h := float32(palettePad*2 + paletteBtn*paletteButtons + paletteGap*(paletteButtons-1) + paletteSep)
	return imgui.NewVec2(palettePad*2+paletteBtn, h)
}

// toggleFocus flips focus mode and refits the image to the new canvas.
func (a *App) toggleFocus() {
	a.focusView = !a.focusView
	a.Viewer.Fit()
}

// drawPalette draws the floating tool palette at the right edge of the viewer canvas.
// It must be called inside the ##viewer window right after Viewer.Draw so the canvas
// rectangle is current.
func (a *App) drawPalette(panel Panel) {
	origin, size := a.Viewer.Canvas()
	ps := paletteSize()
	if size.X < ps.X+16 || size.Y < ps.Y+16 {
		return
	}
	x := origin.X + size.X - ps.X - paletteInset
	y := origin.Y + paletteInset
	imgui.SetCursorScreenPos(imgui.NewVec2(x, y))

	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(palettePad, palettePad))
	imgui.BeginChildStrV("##palette", ps, 0, imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoScrollWithMouse)
	imgui.PopStyleVar()
	// Item spacing is read per item, so it must stay pushed until the buttons are laid out.
	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.NewVec2(0, paletteGap))

	// Fill the child rectangle and draw the raised window frame.
	wp, ws := imgui.WindowPos(), imgui.WindowSize()
	dl := imgui.WindowDrawList()
	FillRect(dl, wp, imgui.NewVec2(wp.X+ws.X, wp.Y+ws.Y), CurrentPalette().Face)
	DrawEdge(dl, wp, imgui.NewVec2(wp.X+ws.X, wp.Y+ws.Y), EdgeWindow)

	host, _ := panel.(PaletteHost)

	tools := []struct {
		t    PaletteTool
		id   string
		name string
	}{
		{PalettePick, "pick", "pick"},
		{PaletteRect, "rect", "rect"},
		{PaletteEllipse, "ellipse", "ellipse"},
		{PalettePolygon, "polygon", "polygon"},
	}
	for _, entry := range tools {
		supported := host != nil && host.PaletteSupports(entry.t)
		active := supported && host.PaletteTool() == entry.t
		if paletteButton("pal-"+entry.id, entry.t, active, supported) {
			if active {
				host.SetPaletteTool(PaletteNone)
			} else {
				host.SetPaletteTool(entry.t)
			}
		}
		if imgui.IsItemHovered() {
			tip := T("palette.tool." + entry.name)
			if !supported {
				tip += "\n" + T("palette.unsupported")
			}
			imgui.SetTooltip(tip)
		}
	}

	// Separator line between the tools and the focus toggle.
	sepPos := imgui.CursorScreenPos()
	imgui.Dummy(imgui.NewVec2(paletteBtn, paletteSep))
	sepY := sepPos.Y + paletteSep/2
	strip(dl, sepPos.X, sepY, sepPos.X+paletteBtn, sepY+1, CurrentPalette().Shadow)
	strip(dl, sepPos.X, sepY+1, sepPos.X+paletteBtn, sepY+2, CurrentPalette().Highlight)

	if paletteButton("pal-focus", PaletteNone, a.focusView, true) {
		a.toggleFocus()
	}
	if imgui.IsItemHovered() {
		imgui.SetTooltip(T("palette.tool.focus"))
	}

	imgui.PopStyleVar()
	imgui.EndChild()
}

// paletteButton draws one owner-drawn 34x34 palette button and returns whether it was
// clicked. Disabled buttons are embossed and ignore clicks.
func paletteButton(id string, icon PaletteTool, active, enabled bool) bool {
	clicked := imgui.InvisibleButton("##"+id, imgui.NewVec2(paletteBtn, paletteBtn))
	min, max := imgui.ItemRectMin(), imgui.ItemRectMax()
	dl := imgui.WindowDrawList()
	p := CurrentPalette()
	if active {
		FillRect(dl, min, max, p.Light)
		DrawEdge(dl, min, max, EdgePressed)
	} else {
		FillRect(dl, min, max, p.Face)
		DrawEdge(dl, min, max, EdgeRaised)
	}
	off := float32(0)
	if active {
		off = 1
	}
	if !enabled {
		drawPaletteIcon(dl, min, max, icon, off+1, p.Highlight)
		drawPaletteIcon(dl, min, max, icon, off, p.Shadow)
	} else {
		drawPaletteIcon(dl, min, max, icon, off, TextColor())
	}
	return enabled && clicked
}

// drawPaletteIcon draws the icon of one palette button inside [min,max) with a pixel offset.
func drawPaletteIcon(dl *imgui.DrawList, min, max imgui.Vec2, icon PaletteTool, off float32, col uint32) {
	const margin = 8
	const thickness = 1.5
	x0 := min.X + margin + off
	y0 := min.Y + margin + off
	x1 := max.X - margin + off
	y1 := max.Y - margin + off
	cx := (x0 + x1) / 2
	cy := (y0 + y1) / 2
	switch icon {
	case PalettePick:
		// Mouse arrow: a filled triangle plus a short tail line.
		dl.AddTriangleFilled(
			imgui.NewVec2(x0, y0),
			imgui.NewVec2(x0, y1),
			imgui.NewVec2(x1, cy),
			col,
		)
		dl.AddLineArgs(imgui.NewVec2(cx, cy), imgui.NewVec2(x1, y1), col, thickness)
	case PaletteRect:
		// Dashed rectangle outline: 4 short segments per side.
		dashLine(dl, x0, y0, x1, y0, col, thickness)
		dashLine(dl, x0, y1, x1, y1, col, thickness)
		dashLine(dl, x0, y0, x0, y1, col, thickness)
		dashLine(dl, x1, y0, x1, y1, col, thickness)
	case PaletteEllipse:
		// 24-segment polyline ellipse.
		const segments = 24
		rx := (x1 - x0) / 2
		ry := (y1 - y0) / 2
		prevX := cx + rx
		prevY := cy
		for i := 1; i <= segments; i++ {
			a := 2 * math.Pi * float64(i) / segments
			x := cx + rx*float32(math.Cos(a))
			y := cy + ry*float32(math.Sin(a))
			dl.AddLineArgs(imgui.NewVec2(prevX, prevY), imgui.NewVec2(x, y), col, thickness)
			prevX, prevY = x, y
		}
	case PalettePolygon:
		// Closed pentagon outline with a 3x3 filled square on every vertex.
		const sides = 5
		pts := make([]imgui.Vec2, sides)
		rx := (x1 - x0) / 2
		ry := (y1 - y0) / 2
		for i := 0; i < sides; i++ {
			a := -math.Pi/2 + 2*math.Pi*float64(i)/sides
			pts[i] = imgui.NewVec2(cx+rx*float32(math.Cos(a)), cy+ry*float32(math.Sin(a)))
		}
		for i := 0; i < sides; i++ {
			dl.AddLineArgs(pts[i], pts[(i+1)%sides], col, thickness)
		}
		for _, pt := range pts {
			dl.AddRectFilled(imgui.NewVec2(pt.X-1.5, pt.Y-1.5), imgui.NewVec2(pt.X+1.5, pt.Y+1.5), col)
		}
	default:
		// Focus: four short corner brackets (fullscreen glyph).
		const arm = 5
		dl.AddLineArgs(imgui.NewVec2(x0, y0), imgui.NewVec2(x0+arm, y0), col, thickness)
		dl.AddLineArgs(imgui.NewVec2(x0, y0), imgui.NewVec2(x0, y0+arm), col, thickness)
		dl.AddLineArgs(imgui.NewVec2(x1, y0), imgui.NewVec2(x1-arm, y0), col, thickness)
		dl.AddLineArgs(imgui.NewVec2(x1, y0), imgui.NewVec2(x1, y0+arm), col, thickness)
		dl.AddLineArgs(imgui.NewVec2(x0, y1), imgui.NewVec2(x0+arm, y1), col, thickness)
		dl.AddLineArgs(imgui.NewVec2(x0, y1), imgui.NewVec2(x0, y1-arm), col, thickness)
		dl.AddLineArgs(imgui.NewVec2(x1, y1), imgui.NewVec2(x1-arm, y1), col, thickness)
		dl.AddLineArgs(imgui.NewVec2(x1, y1), imgui.NewVec2(x1, y1-arm), col, thickness)
	}
}

// dashLine draws a 6/4 px dashed segment between two points.
func dashLine(dl *imgui.DrawList, x0, y0, x1, y1 float32, col uint32, thickness float32) {
	dx := x1 - x0
	dy := y1 - y0
	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length <= 0 {
		return
	}
	ux := dx / length
	uy := dy / length
	for s := float32(0); s < length; s += 10 {
		e := s + 6
		if e > length {
			e = length
		}
		dl.AddLineArgs(
			imgui.NewVec2(x0+ux*s, y0+uy*s),
			imgui.NewVec2(x0+ux*e, y0+uy*e),
			col, thickness,
		)
	}
}

func init() {
	Messages(map[string]string{
		"palette.tool.pick":    "点选",
		"palette.tool.rect":    "矩形框",
		"palette.tool.ellipse": "椭圆框",
		"palette.tool.polygon": "多边形框",
		"palette.tool.focus":   "专注模式 (F11)：隐藏两侧面板，放大中央图像",
		"palette.unsupported":  "当前页面不支持该工具",
		"menu.focus":           "专注模式",
	}, map[string]string{
		"palette.tool.pick":    "Pick",
		"palette.tool.rect":    "Rectangle frame",
		"palette.tool.ellipse": "Ellipse frame",
		"palette.tool.polygon": "Polygon frame",
		"palette.tool.focus":   "Focus mode (F11): hide the side panels to enlarge the centre image",
		"palette.unsupported":  "Not supported on this page",
		"menu.focus":           "Focus mode",
	}, map[string]string{
		"palette.tool.pick":    "선택",
		"palette.tool.rect":    "사각형 프레임",
		"palette.tool.ellipse": "타원 프레임",
		"palette.tool.polygon": "다각형 프레임",
		"palette.tool.focus":   "집중 모드 (F11): 양쪽 패널을 숨기고 중앙 이미지를 확대",
		"palette.unsupported":  "이 페이지에서는 지원되지 않습니다",
		"menu.focus":           "집중 모드",
	})
}
