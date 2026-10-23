package app

import (
	"math"

	"github.com/AllenDang/cimgui-go/backend/glfwbackend"
	"github.com/AllenDang/cimgui-go/imgui"
)

// Custom window chrome: a classic Win32 caption bar (gradient, minimise / maximise / close)
// and resize edges drawn by the program itself, so the window looks the same everywhere
// and the caption can carry program state. Windows enables it by default (-nativeframe
// keeps the OS frame); elsewhere it is opt-in with -chrome. Dragging and resizing use only
// the GLFW position/size calls plus the absolute cursor position, so they work on any
// platform that lets a window move.

const (
	chromeEdge    = 5 // px of resize zone around the window
	chromeMinW    = 960
	chromeMinH    = 600
	chromeCapBase = 22 // caption height at the 13 px design font
)

type chromeState struct {
	on         bool
	drag       bool
	resize     int // bit set of edges while resizing (edgeL|edgeR|edgeT|edgeB)
	startWin   [2]int32
	startSize  [2]int32
	startMouse imgui.Vec2
	maxed      bool
	restore    [4]int32 // x, y, w, h before maximising
}

const (
	edgeL = 1 << iota
	edgeR
	edgeT
	edgeB
)

// captionHeight is the caption bar height (0 without custom chrome).
func (a *App) captionHeight() float32 {
	if !a.chrome.on {
		return 0
	}
	return sc(chromeCapBase)
}

// screenMouse is the cursor in screen coordinates (window origin plus local position).
func (a *App) screenMouse() imgui.Vec2 {
	x, y := a.be.GetWindowPos()
	m := imgui.MousePos()
	return imgui.NewVec2(float32(x)+m.X, float32(y)+m.Y)
}

// drawCaption draws the caption bar and handles dragging, double-click maximise and the
// three caption buttons.
func (a *App) drawCaption(x, y, w, h float32) {
	place(x, y, w, h)
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(0, 0))
	imgui.BeginV("##caption", nil, fixedWindowFlags()|imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoBackground)
	imgui.PopStyleVar()
	dl := imgui.WindowDrawList()
	min := imgui.NewVec2(x, y)
	max := imgui.NewVec2(x+w, y+h)
	Caption(dl, min, max, a.profile.WindowTitle(), imgui.IsWindowFocusedV(imgui.FocusedFlagsAnyWindow))

	btnW, btnH := sc(18), h-sc(6)
	pad := sc(3)
	// drag area: everything left of the buttons
	imgui.SetCursorScreenPos(imgui.NewVec2(x, y))
	imgui.InvisibleButton("##caption-drag", imgui.NewVec2(w-3*btnW-4*pad, h))
	if imgui.IsItemHovered() && imgui.IsMouseDoubleClicked(imgui.MouseButtonLeft) {
		a.toggleMaximise()
	}
	if imgui.IsItemActivated() && !a.chrome.maxed {
		a.chrome.drag = true
		a.chrome.startMouse = a.screenMouse()
		px, py := a.be.GetWindowPos()
		a.chrome.startWin = [2]int32{px, py}
	}
	if a.chrome.drag {
		if imgui.IsMouseDown(imgui.MouseButtonLeft) {
			m := a.screenMouse()
			a.be.SetWindowPos(int(float32(a.chrome.startWin[0])+m.X-a.chrome.startMouse.X), int(float32(a.chrome.startWin[1])+m.Y-a.chrome.startMouse.Y))
		} else {
			a.chrome.drag = false
		}
	}

	bx := x + w - 3*btnW - 3*pad
	by := y + (h-btnH)/2
	glyph := func(i int, label, id string, fn func()) {
		imgui.SetCursorScreenPos(imgui.NewVec2(bx+float32(i)*(btnW+pad/2), by))
		if ButtonSized(label, id, imgui.NewVec2(btnW, btnH), true) {
			fn()
		}
	}
	maxLabel := "□"
	if a.chrome.maxed {
		maxLabel = "❐"
	}
	glyph(0, "_", "caption-min", func() { chromeMinimise(a.profile.WindowTitle()) })
	glyph(1, maxLabel, "caption-max", a.toggleMaximise)
	glyph(2, "×", "caption-close", func() { a.be.SetShouldClose(true) })
	imgui.End()
}

// toggleMaximise maximises to the monitor work area or restores the previous rectangle.
func (a *App) toggleMaximise() {
	if a.chrome.maxed {
		r := a.chrome.restore
		a.be.SetWindowPos(int(r[0]), int(r[1]))
		a.be.SetWindowSize(int(r[2]), int(r[3]))
		a.chrome.maxed = false
		return
	}
	wx, wy, ww, wh, ok := chromeWorkArea()
	if !ok {
		return
	}
	px, py := a.be.GetWindowPos()
	sw, sh := a.be.DisplaySize()
	a.chrome.restore = [4]int32{px, py, sw, sh}
	a.be.SetWindowPos(wx, wy)
	a.be.SetWindowSize(ww, wh)
	a.chrome.maxed = true
}

// handleResizeEdges lets the user resize the borderless window from its 5 px edge zone.
func (a *App) handleResizeEdges(winX, winY, winW, winH float32) {
	if !a.chrome.on || a.chrome.maxed || a.chrome.drag {
		return
	}
	m := imgui.MousePos()
	if a.chrome.resize == 0 {
		var e int
		if m.X >= winX && m.X < winX+chromeEdge {
			e |= edgeL
		}
		if m.X <= winX+winW && m.X > winX+winW-chromeEdge {
			e |= edgeR
		}
		if m.Y >= winY && m.Y < winY+chromeEdge {
			e |= edgeT
		}
		if m.Y <= winY+winH && m.Y > winY+winH-chromeEdge {
			e |= edgeB
		}
		if e == 0 {
			return
		}
		switch {
		case e&(edgeL|edgeT) == edgeL|edgeT, e&(edgeR|edgeB) == edgeR|edgeB:
			imgui.SetMouseCursor(imgui.MouseCursorResizeNWSE)
		case e&(edgeR|edgeT) == edgeR|edgeT, e&(edgeL|edgeB) == edgeL|edgeB:
			imgui.SetMouseCursor(imgui.MouseCursorResizeNESW)
		case e&(edgeL|edgeR) != 0:
			imgui.SetMouseCursor(imgui.MouseCursorResizeEW)
		default:
			imgui.SetMouseCursor(imgui.MouseCursorResizeNS)
		}
		if imgui.IsMouseClickedBool(imgui.MouseButtonLeft) {
			a.chrome.resize = e
			a.chrome.startMouse = a.screenMouse()
			px, py := a.be.GetWindowPos()
			a.chrome.startWin = [2]int32{px, py}
			sw, sh := a.be.DisplaySize()
			a.chrome.startSize = [2]int32{sw, sh}
		}
		return
	}
	if !imgui.IsMouseDown(imgui.MouseButtonLeft) {
		a.chrome.resize = 0
		return
	}
	e := a.chrome.resize
	d := a.screenMouse()
	dx := int32(math.Round(float64(d.X - a.chrome.startMouse.X)))
	dy := int32(math.Round(float64(d.Y - a.chrome.startMouse.Y)))
	x, y, w, h := a.chrome.startWin[0], a.chrome.startWin[1], a.chrome.startSize[0], a.chrome.startSize[1]
	if e&edgeR != 0 {
		w += dx
	}
	if e&edgeB != 0 {
		h += dy
	}
	if e&edgeL != 0 {
		x += dx
		w -= dx
	}
	if e&edgeT != 0 {
		y += dy
		h -= dy
	}
	if w < chromeMinW {
		if e&edgeL != 0 {
			x -= chromeMinW - w
		}
		w = chromeMinW
	}
	if h < chromeMinH {
		if e&edgeT != 0 {
			y -= chromeMinH - h
		}
		h = chromeMinH
	}
	a.be.SetWindowPos(int(x), int(y))
	a.be.SetWindowSize(int(w), int(h))
}

// enableChrome removes the OS frame. The flag is a GLFW window hint, so it only takes
// effect for a window created afterwards: call it before CreateWindow.
func (a *App) enableChrome() {
	a.chrome.on = true
	a.be.SetWindowFlags(glfwbackend.GLFWWindowFlagsDecorated, 0)
}
