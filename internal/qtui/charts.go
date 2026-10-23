package qtui

import (
	"math"
	"strconv"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"afvi/teach-imgui/internal/ui"
)

// parseColor reads "#rrggbb" or "rgba(r,g,b,a)" as produced by ui.Col.
func parseColor(s string) *qt.QColor {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") && len(s) == 7 {
		v, _ := strconv.ParseUint(s[1:], 16, 32)
		return qt.NewQColor3(int(v>>16&255), int(v>>8&255), int(v&255))
	}
	if strings.HasPrefix(s, "rgba(") {
		parts := strings.Split(strings.TrimSuffix(s[5:], ")"), ",")
		if len(parts) == 4 {
			n := func(i int) int { f, _ := strconv.ParseFloat(strings.TrimSpace(parts[i]), 64); return int(f) }
			a, _ := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
			return qt.NewQColor11(n(0), n(1), n(2), int(a*255))
		}
	}
	return qt.NewQColor3(0, 255, 0)
}

// newHist draws the 256-bin log-scale GV histogram.
func newHist(n *ui.Node) *qt.QWidget {
	w := qt.NewQWidget2()
	h := int(pf(n, "height", 80))
	w.SetFixedHeight(h)
	bins, _ := p(n, "bins").([]any)
	mark := int(pf(n, "mark", -1))
	w.OnPaintEvent(func(super func(*qt.QPaintEvent), ev *qt.QPaintEvent) {
		pt := qt.NewQPainter2(w.QPaintDevice)
		defer pt.Delete()
		pt.FillRect5(0, 0, w.Width(), w.Height(), qt.NewQColor3(255, 255, 255))
		maxv := 0.0
		for _, b := range bins {
			if f, _ := b.(float64); f > maxv {
				maxv = f
			}
		}
		if maxv > 0 {
			lm := math.Log(maxv + 1)
			bw := float64(w.Width()) / 256
			for i, b := range bins {
				f, _ := b.(float64)
				if f == 0 {
					continue
				}
				bh := int(math.Log(f+1) / lm * float64(w.Height()-2))
				col := qt.NewQColor3(0, 0, 128)
				if i == mark {
					col = qt.NewQColor3(255, 0, 0)
				}
				x := int(float64(i) * bw)
				pt.FillRect5(x, w.Height()-bh, int(math.Max(1, bw)), bh, col)
			}
		}
		pt.End()
	})
	return w
}

// newHeat draws the NCC surface with the peak circled.
func newHeat(n *ui.Node) *qt.QWidget {
	size := int(pf(n, "size", 120))
	gw, gh := int(pf(n, "w", 1)), int(pf(n, "h", 1))
	peak := int(pf(n, "peak", 0))
	data := decodeB64(ps(n, "data"))
	w := qt.NewQWidget2()
	w.SetFixedSize2(size, size)
	w.OnPaintEvent(func(super func(*qt.QPaintEvent), ev *qt.QPaintEvent) {
		pt := qt.NewQPainter2(w.QPaintDevice)
		defer pt.Delete()
		cw, ch := float64(size)/float64(gw), float64(size)/float64(gh)
		for y := 0; y < gh; y++ {
			for x := 0; x < gw; x++ {
				i := y*gw + x
				if i >= len(data) {
					continue
				}
				t := float64(data[i]) / 255
				var r, g, b int
				if t < 0.5 {
					b = int(255 * t / 0.5)
				} else {
					k := (t - 0.5) / 0.5
					r, g, b = int(255*k), int(255*k), int(255*(1-k))
				}
				pt.FillRect5(int(float64(x)*cw), int(float64(y)*ch), int(cw)+1, int(ch)+1, qt.NewQColor3(r, g, b))
			}
		}
		pen := qt.NewQPen3(qt.NewQColor3(0, 255, 0))
		pen.SetWidth(2)
		pt.SetPenWithPen(pen)
		pt.DrawEllipse2(int((float64(peak%gw)+0.5)*cw)-5, int((float64(peak/gw)+0.5)*ch)-5, 10, 10)
		pt.End()
	})
	return w
}
