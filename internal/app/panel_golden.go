package app

import (
	"strconv"

	"afvi/teach-imgui/internal/access"
	"afvi/teach-imgui/internal/golden"
)

func init() {
	registerPanelFor([]access.Role{access.RoleOperator, access.RoleAdmin}, 3,
		func(a *App) Panel { return newGoldenPanel(a) }, ProfileTeach)
}

// goldenTool selects what a drag on the picture does.
type goldenTool int

const (
	goldenToolPoint goldenTool = iota // pixel readout only (no edit; left drag pans)
	goldenToolAdd                     // add a shape to the Metal mask
	goldenToolErase                   // erase a shape from the Metal mask
)

// goldenColors are the mask colours the page offers.
var goldenColors = [...]struct {
	key string
	rgb [3]uint8
}{{"magenta", [3]uint8{255, 0, 255}}, {"red", [3]uint8{255, 0, 0}}, {"yellow", [3]uint8{255, 255, 0}}, {"cyan", [3]uint8{0, 255, 255}}, {"blue", [3]uint8{0, 128, 255}}}

// GoldenPanel is the golden board page: it edits the Metal mask and the settings of the
// golden board. The column widgets live in panel_golden_ui.go.
type GoldenPanel struct {
	a        *App
	active   bool
	tool     goldenTool
	frame    *FrameTool // shape of Add / Erase
	minGV    int        // Add: only pixels with GV >= minGV (0 = every pixel of the shape)
	areaMin  int        // area threshold in pixels
	areaFill bool       // the area threshold also fills small holes
	morphN   [4]int     // N of Open, Close, Erode, Dilate (index = int(goldenMorph))
	hx, hy   float64    // last hover position in image coordinates
}

var _ Panel = (*GoldenPanel)(nil)
var _ interface{ HidePalette() bool } = (*GoldenPanel)(nil)

// newGoldenPanel returns the page with the default settings.
func newGoldenPanel(a *App) *GoldenPanel {
	p := &GoldenPanel{a: a, tool: goldenToolPoint, frame: NewFrameTool()}
	p.minGV = 0
	p.areaMin = 20
	p.areaFill = true
	for i := range p.morphN {
		p.morphN[i] = 1
	}
	return p
}

// ID returns the page id.
func (p *GoldenPanel) ID() string { return "golden" }

// Title returns the translated tab title.
func (p *GoldenPanel) Title() string { return T("tab.golden") }

// HidePalette: the golden page has no floating tool palette.
func (p *GoldenPanel) HidePalette() bool { return true }

// Activate opens the page and asks the board to repaint the picture.
func (p *GoldenPanel) Activate() {
	p.active = true
	p.frame.Reset()
	p.a.Viewer.SetDoubleClickFit(true)
	p.board().Activate()
}

// Deactivate closes the page.
func (p *GoldenPanel) Deactivate() {
	p.active = false
	p.frame.Reset()
	p.a.Viewer.SetDoubleClickFit(true)
}

// ModelLoaded resets the page for a new model.
func (p *GoldenPanel) ModelLoaded() {
	p.frame.Reset()
	p.tool = goldenToolPoint
}

// LightChanged: the golden sample picture does not depend on the display light.
func (p *GoldenPanel) LightChanged() {}

// Selftest has nothing to seed: the board derives everything itself.
func (p *GoldenPanel) Selftest() {}

// board returns the golden board of the app (nil in the studio).
func (p *GoldenPanel) board() *goldenBoard { return p.a.goldenBoard() }

// Pointer forwards a frame tool event; the point tool leaves the drag to the viewer.
func (p *GoldenPanel) Pointer(ev PointerEvent) bool {
	b := p.board()
	if b == nil || p.tool == goldenToolPoint {
		return false
	}
	switch ev.Kind {
	case PointerDown, PointerMove, PointerUp, PointerRightClick:
	default:
		return false
	}
	owned, done := p.frame.Pointer(ev)
	if done != nil {
		p.finishFrame(done)
	}
	p.a.Viewer.SetDoubleClickFit(!p.frame.Busy())
	return owned
}

// finishFrame hands a finished shape to the board.
func (p *GoldenPanel) finishFrame(res *FrameResult) {
	b := p.board()
	if b == nil || res == nil {
		return
	}
	sh := goldenShapeFromFrame(res)
	switch p.tool {
	case goldenToolAdd:
		b.AddShape(sh, p.minGV)
	case goldenToolErase:
		b.EraseShape(sh)
	}
}

// goldenShapeFromFrame converts a frame tool result to a golden shape.
func goldenShapeFromFrame(res *FrameResult) golden.Shape {
	if res == nil {
		return golden.Shape{}
	}
	switch res.Kind {
	case FrameEllipse:
		return golden.Shape{
			Kind: golden.ShapeEllipse,
			P0:   golden.Pt{X: res.P0.X, Y: res.P0.Y},
			P1:   golden.Pt{X: res.P1.X, Y: res.P1.Y},
		}
	case FramePen:
		pts := make([]golden.Pt, len(res.Points))
		for i, pt := range res.Points {
			pts[i] = golden.Pt{X: pt.X, Y: pt.Y}
		}
		return golden.Shape{Kind: golden.ShapePolygon, Points: pts}
	}
	return golden.Shape{
		Kind: golden.ShapeRect,
		P0:   golden.Pt{X: res.P0.X, Y: res.P0.Y},
		P1:   golden.Pt{X: res.P1.X, Y: res.P1.Y},
	}
}

// setTool selects the edit tool and drops an in-progress frame.
func (p *GoldenPanel) setTool(t goldenTool) {
	p.tool = t
	p.frame.Reset()
}

// setShape selects the shape of the frame tool.
func (p *GoldenPanel) setShape(k FrameKind) {
	p.frame.Kind = k
	p.frame.Reset()
}

// applyMorph applies Open / Close / Erode / Dilate with its N.
func (p *GoldenPanel) applyMorph(i int) {
	if i < 0 || i > 3 {
		return
	}
	p.board().Morph(goldenMorph(i), p.morphN[i])
}

// applyArea applies the area threshold.
func (p *GoldenPanel) applyArea() {
	p.board().AreaFilter(p.areaMin, p.areaFill)
}

// Overlay draws the rubber band of the frame tool.
func (p *GoldenPanel) Overlay(o *Overlay) {
	if p.tool == goldenToolPoint || o == nil {
		return
	}
	p.frame.Draw(o, p.hx, p.hy, rgba(255, 255, 255, 255))
}

// Hover returns the grey value and the mask membership of the pixel.
func (p *GoldenPanel) Hover(x, y int) string {
	p.hx, p.hy = float64(x)+0.5, float64(y)+0.5
	gv, inMetal, inInspect, inSR, ok := p.board().PixelInfo(x, y)
	if !ok {
		return ""
	}
	var where string
	switch {
	case inInspect:
		where = T("golden.hover.inspect")
	case inMetal:
		where = T("golden.hover.band")
	case inSR:
		where = T("golden.hover.sr")
	default:
		where = T("golden.hover.none")
	}
	return T("golden.hover", "gv", strconv.Itoa(gv), "where", where)
}

// ClaimsDrag: the Add / Erase tools take left drags; the point tool leaves them to the viewer.
func (p *GoldenPanel) ClaimsDrag() bool { return p.board() != nil && p.tool != goldenToolPoint }
