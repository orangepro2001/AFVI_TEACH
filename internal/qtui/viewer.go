package qtui

import (
	"math"
	"time"
	"unsafe"

	qt "github.com/mappu/miqt/qt6"

	"afvi/teach-imgui/internal/host"
	"afvi/teach-imgui/internal/ui"
)

const maxLevel = 5

// viewerWidget is the image canvas: it draws the host's image pyramid level that fits the
// zoom, then the overlay shapes in image coordinates, and turns mouse input into the same
// pointer/view commands the browser front end sends (wheel 1.2x, Shift+click 1.5x zoom,
// middle/right drag pans, `claim` routes left drags to the panel).
type viewerWidget struct {
	w *qt.QWidget
	h *host.Host

	out      *ui.ViewerOut
	rev      int
	imgW     int
	imgH     int
	levels   map[int]*qt.QImage
	zoom     float64
	panX     float64
	panY     float64
	lastFit  int
	fitSeq   int
	lastZoom int
	message  string

	mode           string // "", "pan", "tool"
	button         qt.MouseButton
	startX, startY float64
	lastX, lastY   float64
	lastView       time.Time
	lastSent       [5]float64
	paletteHolder  *qt.QWidget
}

func newViewer(h *host.Host) *viewerWidget {
	v := &viewerWidget{h: h, levels: map[int]*qt.QImage{}, zoom: 1, lastFit: -1, fitSeq: -1}
	v.w = qt.NewQWidget2()
	v.w.SetMouseTracking(true)
	v.w.SetMinimumSize2(200, 200)
	v.w.OnPaintEvent(func(super func(*qt.QPaintEvent), ev *qt.QPaintEvent) { v.paint() })
	v.w.OnResizeEvent(func(super func(*qt.QResizeEvent), ev *qt.QResizeEvent) {
		super(ev)
		if v.out != nil && v.imgW > 0 && v.lastFit < 0 {
			v.fit()
		}
		v.sendView(true)
		v.positionPalette()
	})
	v.w.OnMousePressEvent(func(super func(*qt.QMouseEvent), ev *qt.QMouseEvent) { v.press(ev) })
	v.w.OnMouseMoveEvent(func(super func(*qt.QMouseEvent), ev *qt.QMouseEvent) { v.move(ev) })
	v.w.OnMouseReleaseEvent(func(super func(*qt.QMouseEvent), ev *qt.QMouseEvent) { v.release(ev) })
	v.w.OnMouseDoubleClickEvent(func(super func(*qt.QMouseEvent), ev *qt.QMouseEvent) {
		if v.mode != "tool" && (v.out == nil || !v.out.NoDblFit) {
			v.fit()
		}
	})
	v.w.OnWheelEvent(func(super func(*qt.QWheelEvent), ev *qt.QWheelEvent) {
		pos := ev.Position()
		f := 1.2
		if ev.AngleDelta().Y() < 0 {
			f = 1 / 1.2
		}
		v.zoomAt(pos.X(), pos.Y(), f)
	})
	v.w.OnLeaveEvent(func(super func(*qt.QEvent), ev *qt.QEvent) {
		v.h.Do(ui.Cmd{ID: "viewer", Op: "hover", K: "leave"})
	})
	return v
}

func (v *viewerWidget) width() float64  { return float64(v.w.Width()) }
func (v *viewerWidget) height() float64 { return float64(v.w.Height()) }

func (v *viewerWidget) fit() {
	if v.imgW == 0 {
		return
	}
	v.zoom = math.Min(v.width()/float64(v.imgW), v.height()/float64(v.imgH)) * 0.98
	v.panX = (v.width() - float64(v.imgW)*v.zoom) / 2
	v.panY = (v.height() - float64(v.imgH)*v.zoom) / 2
	v.lastFit = 0
	v.sendView(true)
	v.w.Update()
}

func (v *viewerWidget) zoomAt(sx, sy, f float64) {
	nz := math.Min(64, math.Max(0.02, v.zoom*f))
	ix, iy := (sx-v.panX)/v.zoom, (sy-v.panY)/v.zoom
	v.zoom = nz
	v.panX, v.panY = sx-ix*nz, sy-iy*nz
	v.sendView(false)
	v.w.Update()
}

func (v *viewerWidget) sendView(force bool) {
	if !force && time.Since(v.lastView) < 40*time.Millisecond {
		return
	}
	cur := [5]float64{v.zoom, v.panX, v.panY, v.width(), v.height()}
	if cur == v.lastSent && !force {
		return
	}
	v.lastView, v.lastSent = time.Now(), cur
	v.h.Do(ui.Cmd{ID: "viewer", Op: "view", V: map[string]any{"zoom": cur[0], "px": cur[1], "py": cur[2], "w": cur[3], "h": cur[4]}})
}

func (v *viewerWidget) toImage(sx, sy float64) (float64, float64) {
	return (sx - v.panX) / v.zoom, (sy - v.panY) / v.zoom
}

func (v *viewerWidget) press(ev *qt.QMouseEvent) {
	pos := ev.Position()
	sx, sy := pos.X(), pos.Y()
	v.button = ev.Button()
	v.startX, v.startY, v.lastX, v.lastY = sx, sy, sx, sy
	shift := ev.Modifiers()&qt.ShiftModifier != 0
	ctrl := ev.Modifiers()&qt.ControlModifier != 0
	if shift && (v.button == qt.LeftButton || v.button == qt.RightButton) {
		f := 1.5
		if v.button == qt.RightButton {
			f = 1 / 1.5
		}
		v.zoomAt(sx, sy, f)
		v.mode = ""
		return
	}
	ix, iy := v.toImage(sx, sy)
	if v.button == qt.LeftButton && v.out != nil && v.out.Claim {
		v.mode = "tool"
		v.h.Do(ui.Cmd{ID: "viewer", Op: "ptr", K: "down", X: ix, Y: iy, Ctrl: ctrl, Shift: shift})
		return
	}
	v.mode = "pan"
	if v.button == qt.LeftButton {
		v.h.Do(ui.Cmd{ID: "viewer", Op: "ptr", K: "down", X: ix, Y: iy, Ctrl: ctrl})
	}
}

func (v *viewerWidget) move(ev *qt.QMouseEvent) {
	pos := ev.Position()
	sx, sy := pos.X(), pos.Y()
	ix, iy := v.toImage(sx, sy)
	switch v.mode {
	case "pan":
		v.panX += sx - v.lastX
		v.panY += sy - v.lastY
		v.sendView(false)
		v.w.Update()
	case "tool":
		v.h.Do(ui.Cmd{ID: "viewer", Op: "ptr", K: "move", X: ix, Y: iy, Ctrl: ev.Modifiers()&qt.ControlModifier != 0})
	default:
		v.h.Do(ui.Cmd{ID: "viewer", Op: "hover", X: ix, Y: iy})
	}
	v.lastX, v.lastY = sx, sy
}

func (v *viewerWidget) release(ev *qt.QMouseEvent) {
	pos := ev.Position()
	sx, sy := pos.X(), pos.Y()
	ix, iy := v.toImage(sx, sy)
	if v.mode == "tool" {
		v.h.Do(ui.Cmd{ID: "viewer", Op: "ptr", K: "up", X: ix, Y: iy})
	}
	if v.button == qt.RightButton && math.Hypot(sx-v.startX, sy-v.startY) < 4 && ev.Modifiers()&qt.ShiftModifier == 0 {
		v.h.Do(ui.Cmd{ID: "viewer", Op: "ptr", K: "rclick", X: ix, Y: iy})
	}
	v.sendView(true)
	v.mode = ""
}

// update applies the viewer part of a new frame.
func (v *viewerWidget) update(o *ui.ViewerOut) {
	v.out = o
	if o == nil {
		return
	}
	if o.W == 0 {
		v.imgW, v.imgH, v.levels, v.rev = 0, 0, map[int]*qt.QImage{}, -1
		v.w.Update()
		return
	}
	v.imgW, v.imgH = o.W, o.H
	if o.Rev != v.rev {
		v.rev = o.Rev
		v.levels = map[int]*qt.QImage{}
		if v.lastFit < 0 || o.FitSeq != v.lastFitSeq() {
			v.setFitSeq(o.FitSeq)
			v.fit()
		}
	} else if o.FitSeq != v.lastFitSeq() {
		v.setFitSeq(o.FitSeq)
		v.fit()
	}
	if o.ZoomOn != nil && o.ZoomOnSq != v.lastZoom {
		v.lastZoom = o.ZoomOnSq
		if o.ZoomOn[2] > 0 {
			v.zoom = o.ZoomOn[2]
		}
		v.panX = v.width()/2 - o.ZoomOn[0]*v.zoom
		v.panY = v.height()/2 - o.ZoomOn[1]*v.zoom
		v.sendView(true)
	}
	v.w.Update()
}

func (v *viewerWidget) lastFitSeq() int { return v.fitSeq }
func (v *viewerWidget) setFitSeq(s int) { v.fitSeq = s }

func levelFor(z float64) int {
	if z >= 1 {
		return 0
	}
	return int(math.Min(maxLevel, math.Max(0, math.Floor(math.Log2(1/z)))))
}

// level returns the QImage of pyramid level k (copied from the host once per revision).
func (v *viewerWidget) level(k int) *qt.QImage {
	if im, ok := v.levels[k]; ok {
		return im
	}
	pix, w, h, gray, _, _, _ := v.h.Image(k)
	if pix == nil || w == 0 {
		return nil
	}
	format := qt.QImage__Format_RGBA8888
	bpp := 4
	if gray {
		format, bpp = qt.QImage__Format_Grayscale8, 1
	}
	im := qt.NewQImage3(w, h, format)
	dst := unsafe.Slice(im.Bits(), im.SizeInBytes())
	stride := im.BytesPerLine()
	if int(stride) == w*bpp {
		copy(dst, pix)
	} else {
		for y := 0; y < h; y++ {
			copy(dst[y*int(stride):y*int(stride)+w*bpp], pix[y*w*bpp:(y+1)*w*bpp])
		}
	}
	v.levels[k] = im
	return im
}

func (v *viewerWidget) paint() {
	pt := qt.NewQPainter2(v.w.QPaintDevice)
	defer pt.Delete()
	bg := qt.NewQColor3(128, 128, 128)
	if v.out != nil && v.theme() == "dark" {
		bg = qt.NewQColor3(43, 43, 43)
	}
	pt.FillRect5(0, 0, v.w.Width(), v.w.Height(), bg)
	if v.imgW > 0 {
		k := levelFor(v.zoom)
		if im := v.level(k); im != nil {
			f := float64(im.Width()) / float64(v.imgW)
			vx0, vy0 := math.Max(0, -v.panX/v.zoom), math.Max(0, -v.panY/v.zoom)
			vx1, vy1 := math.Min(float64(v.imgW), (v.width()-v.panX)/v.zoom), math.Min(float64(v.imgH), (v.height()-v.panY)/v.zoom)
			if vx1 > vx0 && vy1 > vy0 {
				sx, sy := math.Floor(vx0*f), math.Floor(vy0*f)
				sx1, sy1 := math.Min(float64(im.Width()), math.Ceil(vx1*f)), math.Min(float64(im.Height()), math.Ceil(vy1*f))
				pt.SetRenderHint2(qt.QPainter__SmoothPixmapTransform, v.zoom*float64(v.imgW)/float64(im.Width()) < 1)
				dst := qt.NewQRectF4(v.panX+sx/f*v.zoom, v.panY+sy/f*v.zoom, (sx1-sx)/f*v.zoom, (sy1-sy)/f*v.zoom)
				src := qt.NewQRectF4(sx, sy, sx1-sx, sy1-sy)
				pt.DrawImage(dst, im, src)
			}
		}
		pt.SetPenWithStyle(qt.SolidLine)
		pt.SetPen(qt.NewQColor3(64, 64, 64))
		pt.DrawRect2(int(v.panX)-1, int(v.panY)-1, int(float64(v.imgW)*v.zoom)+2, int(float64(v.imgH)*v.zoom)+2)
	}
	if v.out != nil {
		v.drawShapes(pt)
	}
	if v.message != "" && v.imgW == 0 {
		pt.SetPen(qt.NewQColor3(255, 255, 255))
		pt.DrawText3(24, 40, v.message)
	}
	pt.End()
}

func (v *viewerWidget) theme() string { return themeOf }

var themeOf = "light"

func (v *viewerWidget) drawShapes(pt *qt.QPainter) {
	X := func(x float64) float64 { return v.panX + x*v.zoom }
	Y := func(y float64) float64 { return v.panY + y*v.zoom }
	pen := func(s ui.Shape) {
		col := s.Col
		if col == "" {
			col = "#00ff00"
		}
		pn := qt.NewQPen3(parseColor(col))
		th := s.Th
		if th == 0 {
			th = 1.5
		}
		pn.SetWidthF(th)
		if s.Dash {
			pn.SetStyle(qt.DashLine)
		}
		pt.SetPenWithPen(pn)
		pt.SetBrush(qt.NewQBrush())
	}
	for _, s := range v.out.Shapes {
		switch s.T {
		case "rect":
			pen(s)
			w, h := s.W*v.zoom, s.H*v.zoom
			x, y := X(s.X), Y(s.Y)
			if w < 3 {
				x -= (3 - w) / 2
				w = 3
			}
			if h < 3 {
				y -= (3 - h) / 2
				h = 3
			}
			pt.DrawRect(qt.NewQRectF4(x, y, w, h))
		case "fillrect":
			pt.FillRect4(qt.NewQRectF4(X(s.X), Y(s.Y), s.W*v.zoom, s.H*v.zoom), parseColor(s.Fill))
		case "ellipse":
			pen(s)
			pt.DrawEllipse(qt.NewQRectF4(X(s.X), Y(s.Y), s.W*v.zoom, s.H*v.zoom))
		case "circle":
			pen(s)
			r := math.Max(1.5, s.R*v.zoom)
			pt.DrawEllipse(qt.NewQRectF4(X(s.X)-r, Y(s.Y)-r, 2*r, 2*r))
		case "line":
			pen(s)
			pt.DrawLine4(qt.NewQPointF3(X(s.X), Y(s.Y)), qt.NewQPointF3(X(s.X2), Y(s.Y2)))
		case "poly", "polyline":
			if len(s.Pts) < 2 {
				continue
			}
			pen(s)
			path := qt.NewQPainterPath()
			path.MoveTo(qt.NewQPointF3(X(s.Pts[0][0]), Y(s.Pts[0][1])))
			for _, q := range s.Pts[1:] {
				path.LineTo(qt.NewQPointF3(X(q[0]), Y(q[1])))
			}
			if s.T == "poly" {
				path.CloseSubpath()
				if s.Fill != "" {
					pt.FillPath(path, qt.NewQBrush3(parseColor(s.Fill)))
				}
			}
			pt.DrawPath(path)
		case "text":
			x, y := X(s.X)+s.X2, Y(s.Y)+s.Y2+12
			pt.SetPen(qt.NewQColor3(0, 0, 0))
			pt.DrawText3(int(x)+1, int(y)+1, s.Text)
			col := s.Col
			if col == "" {
				col = "#ffff00"
			}
			pt.SetPen(parseColor(col))
			pt.DrawText3(int(x), int(y), s.Text)
		}
	}
}

func (v *viewerWidget) positionPalette() {
	if v.paletteHolder != nil {
		v.paletteHolder.Move(v.w.Width()-v.paletteHolder.Width()-8, 8)
	}
}
