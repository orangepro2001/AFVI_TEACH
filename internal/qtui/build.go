// Package qtui renders the frame tree (internal/ui) with native Qt Widgets (miqt, Qt 6).
// It is the third front end next to the Wails/Vue window and the browser test mode: the
// panels, the host loop and the protocol are shared, only the drawing differs. Everything
// here runs on the Qt main thread; the host loop runs on its own goroutine and the two meet
// through host.Latest / host.Do / host.Image.
package qtui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"afvi/teach-imgui/internal/ui"
)

// syncing is true while frame values are pushed into widgets, so their change signals do
// not echo back to the host as user edits.
var syncing bool

// registry collects, per slot, the functions that refresh widget values from a new frame
// without rebuilding the widget tree.
type registry struct {
	win   *window
	upd   map[string]func(*ui.Node)
	texts []string // labels seen while building (script assertions)
}

func (r *registry) on(id string, f func(*ui.Node)) {
	if id != "" {
		r.upd[id] = f
	}
}

func (r *registry) send(id, op string, v any) {
	if syncing {
		return
	}
	r.win.h.Do(ui.Cmd{ID: id, Op: op, V: v})
}

func p(n *ui.Node, k string) any {
	if n.P == nil {
		return nil
	}
	return n.P[k]
}
func pf(n *ui.Node, k string, def float64) float64 {
	if f, ok := p(n, k).(float64); ok {
		return f
	}
	return def
}
func pb(n *ui.Node, k string) bool   { b, _ := p(n, k).(bool); return b }
func ps(n *ui.Node, k string) string { s, _ := p(n, k).(string); return s }
func vf(n *ui.Node) float64 {
	switch v := n.V.(type) {
	case float64:
		return v
	case bool:
		if v {
			return 1
		}
	}
	return 0
}
func vb(n *ui.Node) bool { b, _ := n.V.(bool); return b }

// signature identifies the structure of a tree: kinds, ids, labels of controls and static
// properties. Values that change every frame (sliders, texts, selections) are left out, so
// those only refresh in place.
func signature(n *ui.Node, sb *strings.Builder) {
	sb.WriteString(n.K)
	sb.WriteByte('|')
	sb.WriteString(n.ID)
	switch n.K {
	case "alignbanner", ui.KBadge:
	default:
		sb.WriteByte('|')
		sb.WriteString(n.L)
	}
	switch n.K {
	case ui.KCombo, ui.KTabs, ui.KSubTabs, ui.KTable, ui.KKV, ui.KHist, ui.KHeat, "palette":
		fmt.Fprintf(sb, "|%v", n.P)
	case ui.KSection:
		fmt.Fprintf(sb, "|%v", n.V)
	case ui.KButton:
		fmt.Fprintf(sb, "|%v", n.P)
	}
	sb.WriteByte('(')
	for _, c := range n.C {
		signature(c, sb)
		sb.WriteByte(',')
	}
	sb.WriteByte(')')
}

// refresh runs every registered updater over the new tree.
func (r *registry) refresh(n *ui.Node) {
	syncing = true
	defer func() { syncing = false }()
	var walk func(*ui.Node)
	walk = func(n *ui.Node) {
		if f, ok := r.upd[n.ID]; ok && n.ID != "" {
			f(n)
		}
		for _, c := range n.C {
			walk(c)
		}
	}
	walk(n)
}

func vbox(margin int) (*qt.QWidget, *qt.QVBoxLayout) {
	w := qt.NewQWidget2()
	l := qt.NewQVBoxLayout(w)
	l.SetContentsMargins(margin, margin, margin, margin)
	l.SetSpacing(3)
	return w, l
}

func hbox() (*qt.QWidget, *qt.QHBoxLayout) {
	w := qt.NewQWidget2()
	l := qt.NewQHBoxLayout(w)
	l.SetContentsMargins(0, 0, 0, 0)
	l.SetSpacing(4)
	return w, l
}

func toneStyle(tone string) string {
	switch tone {
	case "dim":
		return "color:#606060;"
	case "warn":
		return "color:#a07000;"
	case "good":
		return "color:#008000;"
	case "bad":
		return "color:#c00000;"
	}
	return ""
}

func decimals(format string) int {
	if i := strings.Index(format, "."); i >= 0 {
		j := i + 1
		k := j
		for k < len(format) && format[k] >= '0' && format[k] <= '9' {
			k++
		}
		if n, err := strconv.Atoi(format[j:k]); err == nil {
			return n
		}
	}
	return 0
}

func fmtNum(v float64, format string) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "—"
	}
	if format == "" {
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	if strings.Contains(format, "%d") {
		return strings.Replace(format, "%d", strconv.Itoa(int(math.Round(v))), 1)
	}
	return strings.Replace(format, "%."+strconv.Itoa(decimals(format))+"f", strconv.FormatFloat(v, 'f', decimals(format), 64), 1)
}

// labelRow lays out "label | control" like the old label-left controls.
func labelRow(label string, ctl *qt.QWidget) *qt.QWidget {
	w, l := hbox()
	if label != "" {
		lb := qt.NewQLabel3(label)
		lb.SetMinimumWidth(110)
		lb.SetMaximumWidth(110)
		l.AddWidget(lb.QWidget)
	}
	l.AddWidget2(ctl, 1)
	return w
}

// build creates the widget for node n and registers its updater.
func (r *registry) build(n *ui.Node) *qt.QWidget {
	if n.L != "" {
		r.texts = append(r.texts, n.L)
	}
	switch n.K {
	case ui.KRoot:
		w, l := vbox(0)
		for _, c := range n.C {
			l.AddWidget(r.build(c))
		}
		l.AddStretch()
		return w
	case ui.KRow:
		w, l := hbox()
		for _, c := range n.C {
			l.AddWidget(r.build(c))
		}
		l.AddStretch()
		return w
	case ui.KGroup:
		g := qt.NewQGroupBox3(n.L)
		l := qt.NewQVBoxLayout(g.QWidget)
		l.SetSpacing(3)
		for _, c := range n.C {
			l.AddWidget(r.build(c))
		}
		return g.QWidget
	case ui.KSection:
		return r.section(n)
	case ui.KText, ui.KNote, ui.KWarn:
		lb := qt.NewQLabel3(n.L)
		style := toneStyle(ps(n, "tone"))
		if n.K == ui.KNote {
			style = toneStyle("dim")
			lb.SetWordWrap(true)
		} else if n.K == ui.KWarn {
			style = toneStyle("warn")
			lb.SetWordWrap(true)
		}
		if style != "" {
			lb.SetStyleSheet(style)
		}
		return lb.QWidget
	case ui.KSep:
		f := qt.NewQFrame2()
		f.SetFrameShape(qt.QFrame__HLine)
		f.SetFrameShadow(qt.QFrame__Sunken)
		return f.QWidget
	case ui.KSpacer:
		w := qt.NewQWidget2()
		w.SetFixedHeight(int(pf(n, "h", 6)))
		return w
	case ui.KButton, ui.KTool:
		b := qt.NewQPushButton3(n.L)
		id := n.ID
		r.win.buttons[id] = b.QAbstractButton
		if n.K == ui.KTool {
			b.SetCheckable(true)
			b.SetChecked(vb(n))
			r.on(id, func(m *ui.Node) { b.SetChecked(vb(m)) })
		} else {
			b.SetEnabled(p(n, "enabled") != false)
			b.SetDefault(pb(n, "def"))
		}
		b.OnClicked(func() {
			if n.K == ui.KTool {
				b.SetChecked(vb(n)) // the host decides the state
			}
			r.send(id, "click", nil)
		})
		return b.QWidget
	case ui.KCheck:
		c := qt.NewQCheckBox3(n.L)
		c.SetChecked(vb(n))
		id := n.ID
		c.OnToggled(func(on bool) { r.send(id, "set", on) })
		r.on(id, func(m *ui.Node) { c.SetChecked(vb(m)) })
		return c.QWidget
	case ui.KRadio:
		b := qt.NewQRadioButton3(n.L)
		b.SetChecked(vb(n))
		id := n.ID
		b.OnClicked(func() { r.send(id, "click", nil) })
		r.on(id, func(m *ui.Node) { b.SetChecked(vb(m)) })
		return b.QWidget
	case ui.KSlider:
		return r.slider(n)
	case ui.KInput:
		sp := qt.NewQDoubleSpinBox2()
		sp.SetRange(-1e9, 1e9)
		sp.SetDecimals(decimals(ps(n, "fmt")))
		sp.SetSingleStep(pf(n, "step", 1))
		sp.SetValue(vf(n))
		id := n.ID
		sp.OnValueChanged(func(v float64) { r.send(id, "set", v) })
		r.on(id, func(m *ui.Node) {
			if !sp.HasFocus() {
				sp.SetValue(vf(m))
			}
		})
		return labelRow(n.L, sp.QWidget)
	case "num":
		sp := qt.NewQDoubleSpinBox2()
		sp.SetRange(pf(n, "min", -1e9), pf(n, "max", 1e9))
		sp.SetDecimals(decimals(ps(n, "fmt")))
		sp.SetSingleStep(pf(n, "step", 1))
		sp.SetValue(vf(n))
		sp.SetMinimumWidth(80)
		id := n.ID
		sp.OnValueChanged(func(v float64) { r.send(id, "set", v) })
		r.on(id, func(m *ui.Node) {
			if !sp.HasFocus() {
				sp.SetValue(vf(m))
			}
		})
		return sp.QWidget
	case ui.KText1:
		le := qt.NewQLineEdit2()
		s, _ := n.V.(string)
		le.SetText(s)
		le.SetPlaceholderText(ps(n, "hint"))
		if pb(n, "password") {
			le.SetEchoMode(qt.QLineEdit__Password)
		}
		id := n.ID
		le.OnTextEdited(func(t string) { r.send(id, "set", t) })
		r.on(id, func(m *ui.Node) {
			if !le.HasFocus() {
				t, _ := m.V.(string)
				le.SetText(t)
			}
		})
		return labelRow(n.L, le.QWidget)
	case ui.KCombo:
		cb := qt.NewQComboBox2()
		items, _ := p(n, "items").([]any)
		for _, it := range items {
			s, _ := it.(string)
			cb.AddItem(s)
		}
		cb.SetCurrentIndex(int(vf(n)))
		id := n.ID
		cb.OnCurrentIndexChanged(func(i int) {
			if i >= 0 {
				r.send(id, "set", float64(i))
			}
		})
		r.on(id, func(m *ui.Node) { cb.SetCurrentIndex(int(vf(m))) })
		return labelRow(n.L, cb.QWidget)
	case ui.KTabs:
		tb := qt.NewQTabBar2()
		items, _ := p(n, "items").([]any)
		for _, it := range items {
			s, _ := it.(string)
			r.texts = append(r.texts, s)
			tb.AddTab(s)
		}
		tb.SetExpanding(false)
		tb.SetCurrentIndex(int(vf(n)))
		id := n.ID
		tb.OnCurrentChanged(func(i int) {
			if i >= 0 {
				r.send(id, "set", float64(i))
			}
		})
		r.on(id, func(m *ui.Node) { tb.SetCurrentIndex(int(vf(m))) })
		tw, tl := hbox()
		tl.AddWidget(tb.QWidget)
		tl.AddStretch()
		return tw
	case ui.KSubTabs:
		return r.subtabs(n)
	case ui.KTable:
		return r.table(n)
	case ui.KKV:
		w := qt.NewQWidget2()
		g := qt.NewQGridLayout(w)
		g.SetContentsMargins(0, 0, 0, 0)
		rows, _ := p(n, "rows").([]any)
		for i, row := range rows {
			cells, _ := row.([]any)
			if len(cells) < 2 {
				continue
			}
			k, _ := cells[0].(string)
			v, _ := cells[1].(string)
			kl := qt.NewQLabel3(k)
			kl.SetStyleSheet(toneStyle("dim"))
			vl := qt.NewQLabel3(v)
			g.AddWidget2(kl.QWidget, i, 0)
			g.AddWidget2(vl.QWidget, i, 1)
		}
		return w
	case ui.KBadge, "alignbanner":
		lb := qt.NewQLabel3(n.L)
		lb.SetAlignment(qt.AlignCenter)
		level := ps(n, "level")
		bg, fg := "#d4d0c8", "#000"
		switch level {
		case "ok":
			bg, fg = "#00a000", "#fff"
		case "ng":
			bg, fg = "#c00000", "#fff"
		case "warn", "manual":
			bg, fg = "#ffd800", "#000"
		}
		size := "12px"
		if pb(n, "big") || n.K == "alignbanner" {
			size = "22px"
		}
		lb.SetStyleSheet("background:" + bg + ";color:" + fg + ";font-weight:bold;font-size:" + size + ";padding:4px 12px;border:1px solid #404040;")
		r.on(n.ID, func(m *ui.Node) { lb.SetText(m.L) })
		return lb.QWidget
	case "tonetext":
		lb := qt.NewQLabel3(n.L)
		lb.SetWordWrap(true)
		lb.SetStyleSheet(toneStyle(ps(n, "tone")))
		return lb.QWidget
	case ui.KSwatch:
		lb := qt.NewQLabel2()
		lb.SetFixedSize2(56, 18)
		lb.SetStyleSheet("background:" + ps(n, "color") + ";border:1px solid #404040;")
		return lb.QWidget
	case ui.KHist:
		return newHist(n)
	case ui.KHeat:
		return newHeat(n)
	case ui.KProgress:
		pr := qt.NewQProgressBar2()
		pr.SetRange(0, 1000)
		pr.SetValue(int(vf(n) * 1000))
		return pr.QWidget
	}
	lb := qt.NewQLabel3("⟨" + n.K + "⟩")
	lb.SetStyleSheet(toneStyle("warn"))
	return lb.QWidget
}

func (r *registry) section(n *ui.Node) *qt.QWidget {
	w, l := vbox(0)
	open := vb(n)
	arrow := "▸ "
	if open {
		arrow = "▾ "
	}
	h := qt.NewQPushButton3(arrow + n.L)
	h.SetStyleSheet("text-align:left;padding:3px 6px;")
	id := n.ID
	h.OnClicked(func() { r.send(id, "toggle", nil) })
	l.AddWidget(h.QWidget)
	if open {
		body, bl := vbox(4)
		for _, c := range n.C {
			bl.AddWidget(r.build(c))
		}
		l.AddWidget(body)
	}
	return w
}

func (r *registry) slider(n *ui.Node) *qt.QWidget {
	min, max := pf(n, "min", 0), pf(n, "max", 1)
	isInt := pb(n, "int")
	const steps = 1000
	toPos := func(v float64) int {
		if max <= min {
			return 0
		}
		if isInt {
			return int(math.Round(v - min))
		}
		return int(math.Round((v - min) / (max - min) * steps))
	}
	fromPos := func(pos int) float64 {
		if isInt {
			return min + float64(pos)
		}
		return min + float64(pos)/steps*(max-min)
	}
	sl := qt.NewQSlider3(qt.Horizontal)
	if isInt {
		sl.SetRange(0, int(max-min))
	} else {
		sl.SetRange(0, steps)
	}
	sl.SetValue(toPos(vf(n)))
	format := ps(n, "fmt")
	val := qt.NewQLabel3(fmtNum(vf(n), format))
	val.SetMinimumWidth(52)
	val.SetAlignment(qt.AlignRight | qt.AlignVCenter)
	var sp *qt.QDoubleSpinBox
	id := n.ID
	w, l := hbox()
	if n.L != "" {
		lb := qt.NewQLabel3(n.L)
		lb.SetMinimumWidth(110)
		lb.SetMaximumWidth(110)
		l.AddWidget(lb.QWidget)
	}
	l.AddWidget2(sl.QWidget, 1)
	if pb(n, "field") {
		sp = qt.NewQDoubleSpinBox2()
		sp.SetRange(min, max)
		sp.SetDecimals(decimals(format))
		if isInt {
			sp.SetDecimals(0)
		}
		sp.SetValue(vf(n))
		sp.SetMinimumWidth(72)
		sp.OnValueChanged(func(v float64) { r.send(id, "set", v) })
		l.AddWidget(sp.QWidget)
	} else {
		l.AddWidget(val.QWidget)
	}
	sl.OnValueChanged(func(pos int) {
		v := fromPos(pos)
		val.SetText(fmtNum(v, format))
		r.send(id, "set", v)
	})
	r.on(id, func(m *ui.Node) {
		if sl.IsSliderDown() {
			return
		}
		sl.SetValue(toPos(vf(m)))
		val.SetText(fmtNum(vf(m), format))
		if sp != nil && !sp.HasFocus() {
			sp.SetValue(vf(m))
		}
	})
	return w
}

func (r *registry) subtabs(n *ui.Node) *qt.QWidget {
	w, l := hbox()
	items, _ := p(n, "items").([]any)
	var btns []*qt.QPushButton
	id := n.ID
	for i, it := range items {
		m, _ := it.(map[string]any)
		label, _ := m["l"].(string)
		if c, _ := m["count"].(string); c != "" {
			label += " " + c
		}
		b := qt.NewQPushButton3(label)
		b.SetCheckable(true)
		if col, _ := m["color"].(string); col != "" {
			pm := qt.NewQPixmap2(12, 12)
			pm.FillWithFillColor(parseColor(col))
			b.SetIcon(qt.NewQIcon2(pm))
		}
		idx := i
		b.OnClicked(func() {
			for j, o := range btns {
				o.SetChecked(j == int(vf(n)))
			}
			r.send(id, "set", float64(idx))
		})
		btns = append(btns, b)
		l.AddWidget(b.QWidget)
	}
	set := func(m *ui.Node) {
		for j, b := range btns {
			b.SetChecked(j == int(vf(m)))
		}
	}
	set(n)
	r.on(id, set)
	if n.L != "" {
		nl := qt.NewQLabel3(n.L)
		nl.SetStyleSheet(toneStyle("dim"))
		l.AddWidget(nl.QWidget)
	}
	l.AddStretch()
	return w
}

func (r *registry) table(n *ui.Node) *qt.QWidget {
	headers, _ := p(n, "headers").([]any)
	rows, _ := p(n, "rows").([]any)
	t := qt.NewQTableWidget3(len(rows), len(headers))
	var hs []string
	for _, h := range headers {
		s, _ := h.(string)
		hs = append(hs, s)
	}
	t.SetHorizontalHeaderLabels(hs)
	t.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	t.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	t.SetSelectionMode(qt.QAbstractItemView__SingleSelection)
	t.VerticalHeader().SetVisible(false)
	t.HorizontalHeader().SetStretchLastSection(true)
	for ri, row := range rows {
		cells, _ := row.([]any)
		for ci, c := range cells {
			m, _ := c.(map[string]any)
			txt, _ := m["t"].(string)
			it := qt.NewQTableWidgetItem2(txt)
			if tone, _ := m["tone"].(string); tone != "" {
				it.SetForeground(qt.NewQBrush3(parseColor(map[string]string{"dim": "#606060", "warn": "#a07000", "good": "#008000", "bad": "#c00000"}[tone])))
			}
			t.SetItem(ri, ci, it)
		}
	}
	t.ResizeColumnsToContents()
	h := int(pf(n, "height", 8))
	if h > len(rows)+1 {
		h = len(rows) + 1
	}
	t.SetFixedHeight(h*22 + 30)
	id := n.ID
	t.OnCellClicked(func(row, col int) { r.send(id, "set", float64(row)) })
	sel := func(m *ui.Node) {
		if s := int(vf(m)); s >= 0 && s < len(rows) {
			t.SelectRow(s)
		} else {
			t.ClearSelection()
		}
	}
	sel(n)
	r.on(id, sel)
	return t.QWidget
}
