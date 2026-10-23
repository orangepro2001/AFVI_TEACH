package ui

import (
	"fmt"
	"math"
)

// Pointer phases the front end reports for a tool drag on the viewer.
const (
	PtrDown       = "down"
	PtrMove       = "move"
	PtrUp         = "up"
	PtrRightClick = "rclick"
)

// Pointer is one viewer event in image coordinates (full-resolution pixels of the
// shown image).
type Pointer struct {
	Kind  string
	X, Y  float64
	Ctrl  bool
	Shift bool
}

// Shape is one overlay drawing instruction in image coordinates. Th is a thickness in
// screen pixels; the front end keeps it constant while zooming.
type Shape struct {
	T    string       `json:"t"` // rect | ellipse | poly | line | dot | text | fillrect | hatch
	X    float64      `json:"x"`
	Y    float64      `json:"y"`
	W    float64      `json:"w,omitempty"`
	H    float64      `json:"h,omitempty"`
	X2   float64      `json:"x2,omitempty"`
	Y2   float64      `json:"y2,omitempty"`
	R    float64      `json:"r,omitempty"`
	Pts  [][2]float64 `json:"pts,omitempty"`
	Col  string       `json:"col,omitempty"`
	Fill string       `json:"fill,omitempty"`
	Th   float64      `json:"th,omitempty"`
	Dash bool         `json:"dash,omitempty"`
	Text string       `json:"text,omitempty"`
}

// ViewState is what the front end reports about its canvas.
type ViewState struct {
	Zoom, PanX, PanY float64
	W, H             float64 // canvas size in CSS pixels
}

// Viewer is the Go side of the image canvas. The front end owns zoom and pan (so
// dragging never waits for a round trip); Go owns the image, the overlay shapes and
// the requests to fit or zoom on a point.
type Viewer struct {
	ImgW, ImgH int
	Rev        int // bumped by SetImage
	pix        []byte
	gray       bool
	Shapes     []Shape
	View       ViewState

	FitSeq   int // bumped to ask the front end to fit the image
	ZoomOnSq int
	zoomOn   *[3]float64 // x, y, zoom
	Claim    bool        // left drags go to the panel's tool instead of panning
	NoDblFit bool        // multi-click tools keep double click for themselves
	Cursor   string
	Note     string // status line shown by the canvas (hover readout)

	ptrs    []Pointer
	hoverFn func(x, y int) string
	hoverX  float64
	hoverY  float64
	hoverOn bool
}

// NewViewer returns an empty viewer.
func NewViewer() *Viewer { return &Viewer{View: ViewState{Zoom: 1}} }

// SetImage shows w×h RGBA pixels. A size change asks for a re-fit; same-size updates keep
// zoom and pan. The buffer is retained until the next image (the HTTP handler reads it).
func (v *Viewer) SetImage(w, h int, rgba []byte) {
	if w != v.ImgW || h != v.ImgH {
		v.FitSeq++
	}
	v.ImgW, v.ImgH = w, h
	v.pix = rgba
	v.gray = false
	v.Rev++
}

// SetGray shows an 8-bit image without expanding it to RGBA.
func (v *Viewer) SetGray(w, h int, gray []byte) {
	if w != v.ImgW || h != v.ImgH {
		v.FitSeq++
	}
	v.ImgW, v.ImgH = w, h
	v.pix = gray
	v.gray = true
	v.Rev++
}

// Clear removes the image.
func (v *Viewer) Clear() {
	v.ImgW, v.ImgH, v.pix = 0, 0, nil
	v.Rev++
}

// Pixels returns the current image bytes and layout (host image handler).
func (v *Viewer) Pixels() (pix []byte, w, h int, gray bool) { return v.pix, v.ImgW, v.ImgH, v.gray }

// Fit asks the front end to fit the image in the canvas.
func (v *Viewer) Fit() { v.FitSeq++ }

// ZoomOn sets the zoom (screen pixels per image pixel) and centres the view on image
// point (x, y).
func (v *Viewer) ZoomOn(x, y float64, zoom float32) {
	v.zoomOn = &[3]float64{x, y, float64(zoom)}
	v.ZoomOnSq++
}

// CenterOn centres the view on image point (x, y) keeping the zoom.
func (v *Viewer) CenterOn(x, y float64) {
	v.zoomOn = &[3]float64{x, y, 0}
	v.ZoomOnSq++
}

// Zoom returns screen pixels per image pixel as last reported by the front end.
func (v *Viewer) Zoom() float32 { return float32(v.View.Zoom) }

// ImageSize returns the size of the displayed image.
func (v *Viewer) ImageSize() (int, int) { return v.ImgW, v.ImgH }

// HasImage reports whether an image is shown.
func (v *Viewer) HasImage() bool { return v.ImgW > 0 && v.ImgH > 0 }

// Mouse returns the cursor position in image coordinates and whether it is over the image.
func (v *Viewer) Mouse() (x, y float64, inside bool) {
	in := v.hoverOn && v.hoverX >= 0 && v.hoverY >= 0 && v.hoverX < float64(v.ImgW) && v.hoverY < float64(v.ImgH)
	return v.hoverX, v.hoverY, in
}

// SetDoubleClickFit enables or disables "double click fits the view".
func (v *Viewer) SetDoubleClickFit(on bool) { v.NoDblFit = !on }

// ---- overlay recording -----------------------------------------------------------

// Overlay is handed to the active panel each frame to draw in image coordinates. Its
// methods keep the signatures of the Dear ImGui overlay so panel code did not change:
// colours are 0xAABBGGRR, thickness is in screen pixels.
type Overlay struct{ v *Viewer }

// Overlay starts a new frame of shapes.
func (v *Viewer) Overlay() *Overlay { v.Shapes = v.Shapes[:0]; return &Overlay{v: v} }

// Scale returns screen pixels per image pixel (as last reported by the front end).
func (o *Overlay) Scale() float32 { return float32(o.v.View.Zoom) }

func (o *Overlay) add(s Shape) { o.v.Shapes = append(o.v.Shapes, s) }

// Rect outlines an image-space rectangle; tiny rectangles stay at least 3 px on screen.
func (o *Overlay) Rect(x, y, w, h float64, col uint32, th float32) {
	o.add(Shape{T: "rect", X: x, Y: y, W: w, H: h, Col: Col(col), Th: float64(th)})
}

// FillRect fills an image-space rectangle.
func (o *Overlay) FillRect(x, y, w, h float64, col uint32) {
	o.add(Shape{T: "fillrect", X: x, Y: y, W: w, H: h, Fill: Col(col)})
}

// DashedRect outlines a rectangle with dashes.
func (o *Overlay) DashedRect(x, y, w, h float64, col uint32, th float32) {
	o.add(Shape{T: "rect", X: x, Y: y, W: w, H: h, Col: Col(col), Th: float64(th), Dash: true})
}

// DragBox draws the normalized rectangle spanned by two image points (rubber band).
func (o *Overlay) DragBox(x0, y0, x1, y1 float64, col uint32) {
	o.DashedRect(math.Min(x0, x1), math.Min(y0, y1), math.Abs(x1-x0), math.Abs(y1-y0), col, 1)
}

// Line draws an image-space segment.
func (o *Overlay) Line(x0, y0, x1, y1 float64, col uint32, th float32) {
	o.add(Shape{T: "line", X: x0, Y: y0, X2: x1, Y2: y1, Col: Col(col), Th: float64(th)})
}

// Circle outlines an image-space circle (radius in image pixels, at least 1.5 px on screen).
func (o *Overlay) Circle(cx, cy, r float64, col uint32, th float32) {
	o.add(Shape{T: "circle", X: cx, Y: cy, R: r, Col: Col(col), Th: float64(th)})
}

// Ellipse outlines the ellipse inscribed in the rectangle.
func (o *Overlay) Ellipse(x, y, w, h float64, col uint32, th float32) {
	o.add(Shape{T: "ellipse", X: x, Y: y, W: w, H: h, Col: Col(col), Th: float64(th)})
}

// Poly outlines (closed) or traces (open) a polygon; fill 0 = no fill.
func (o *Overlay) Poly(pts [][2]float64, closed bool, col, fill uint32, th float32) {
	t := "poly"
	if !closed {
		t = "polyline"
	}
	sh := Shape{T: t, Pts: pts, Col: Col(col), Th: float64(th)}
	if fill != 0 {
		sh.Fill = Col(fill)
	}
	o.add(sh)
}

// Text draws a label at an image-space anchor plus a screen-pixel offset.
func (o *Overlay) Text(x, y float64, dx, dy float32, col uint32, s string) {
	o.add(Shape{T: "text", X: x, Y: y, X2: float64(dx), Y2: float64(dy), Text: s, Col: Col(col)})
}

// Visible reports whether an image rectangle may intersect the canvas (always true until
// the front end has reported its view).
func (o *Overlay) Visible(x, y, w, h float64) bool {
	vs := o.v.View
	if vs.W == 0 || vs.Zoom == 0 {
		return true
	}
	x0, y0 := -vs.PanX/vs.Zoom, -vs.PanY/vs.Zoom
	x1, y1 := x0+vs.W/vs.Zoom, y0+vs.H/vs.Zoom
	return !(x+w < x0 || y+h < y0 || x > x1 || y > y1)
}

// ---- events -----------------------------------------------------------------------

// Handle consumes a pointer or view command from the front end.
func (v *Viewer) Handle(c Cmd) {
	switch c.Op {
	case "ptr":
		v.ptrs = append(v.ptrs, Pointer{Kind: c.K, X: c.X, Y: c.Y, Ctrl: c.Ctrl, Shift: c.Shift})
		v.hoverX, v.hoverY, v.hoverOn = c.X, c.Y, true
	case "hover":
		v.hoverX, v.hoverY, v.hoverOn = c.X, c.Y, c.K != "leave"
	case "view":
		if m, ok := c.V.(map[string]any); ok {
			get := func(k string) float64 { f, _ := m[k].(float64); return f }
			v.View = ViewState{Zoom: get("zoom"), PanX: get("px"), PanY: get("py"), W: get("w"), H: get("h")}
		}
	}
}

// TakePointers returns the pointer events received since the last call.
func (v *Viewer) TakePointers() []Pointer { p := v.ptrs; v.ptrs = nil; return p }

// SetHover installs the status readout for the pixel under the cursor.
func (v *Viewer) SetHover(fn func(x, y int) string) { v.hoverFn = fn }

// HoverText is the readout for the current cursor position ("" when outside).
func (v *Viewer) HoverText() string {
	if !v.hoverOn || v.hoverFn == nil || v.ImgW == 0 {
		return ""
	}
	x, y := int(math.Floor(v.hoverX)), int(math.Floor(v.hoverY))
	if x < 0 || y < 0 || x >= v.ImgW || y >= v.ImgH {
		return ""
	}
	return v.hoverFn(x, y)
}

// Out is the viewer part of a frame message.
type ViewerOut struct {
	Rev      int         `json:"rev"`
	W        int         `json:"w"`
	H        int         `json:"h"`
	Gray     bool        `json:"gray,omitempty"`
	Shapes   []Shape     `json:"shapes"`
	FitSeq   int         `json:"fitSeq"`
	ZoomOn   *[3]float64 `json:"zoomOn,omitempty"`
	ZoomOnSq int         `json:"zoomOnSeq"`
	Claim    bool        `json:"claim,omitempty"`
	NoDblFit bool        `json:"noDblFit,omitempty"`
	Cursor   string      `json:"cursor,omitempty"`
	Hover    string      `json:"hover,omitempty"`
}

// Out snapshots the viewer for the wire.
func (v *Viewer) Out() *ViewerOut {
	shapes := v.Shapes
	if shapes == nil {
		shapes = []Shape{}
	}
	return &ViewerOut{Rev: v.Rev, W: v.ImgW, H: v.ImgH, Gray: v.gray, Shapes: shapes, FitSeq: v.FitSeq,
		ZoomOn: v.zoomOn, ZoomOnSq: v.ZoomOnSq, Claim: v.Claim, NoDblFit: v.NoDblFit, Cursor: v.Cursor, Hover: v.HoverText()}
}

// Col converts a 0xAABBGGRR ImGui colour (the format of the old palettes) to CSS.
func Col(abgr uint32) string {
	r, g, b, a := abgr&0xff, (abgr>>8)&0xff, (abgr>>16)&0xff, (abgr>>24)&0xff
	if a == 255 {
		return fmt.Sprintf("#%02x%02x%02x", r, g, b)
	}
	return fmt.Sprintf("rgba(%d,%d,%d,%.3f)", r, g, b, float64(a)/255)
}

// RGBA builds a CSS colour from 0..255 channels and 0..1 alpha.
func RGBA(r, g, b uint8, a float64) string {
	return fmt.Sprintf("rgba(%d,%d,%d,%.3f)", r, g, b, a)
}
