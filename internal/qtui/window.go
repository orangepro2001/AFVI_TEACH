package qtui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"afvi/teach-imgui/internal/app"
	"afvi/teach-imgui/internal/host"
	"afvi/teach-imgui/internal/ui"
)

func init() { app.WindowRunner = run }

type frameMsg struct {
	Seq    int                 `json:"seq"`
	Title  string              `json:"title"`
	Slots  map[string]*ui.Node `json:"slots"`
	Viewer *ui.ViewerOut       `json:"viewer"`
	Meta   map[string]any      `json:"meta"`
}

// slotView is one region of the window (toolbar, a column, ...): the current widget tree,
// its structural signature and the updaters for in-place value refresh.
type slotView struct {
	holder *qt.QVBoxLayout // the layout the slot's widget lives in
	scroll *qt.QScrollArea // set for the scrolling columns
	cur    *qt.QWidget
	sig    string
	reg    *registry
}

type window struct {
	h       *host.Host
	mw      *qt.QMainWindow
	slots   map[string]*slotView
	buttons map[string]*qt.QAbstractButton
	viewer  *viewerWidget
	menuBar *qt.QMenuBar
	menuSig string
	palette *qt.QWidget
	palSig  string
	status  []*qt.QLabel
	hover   *qt.QLabel
	dialogs map[string]*dialogView
	seq     int
	theme   string
	fontPx  float64
	focus   bool
	left    *qt.QWidget
	right   *qt.QWidget
	lastTit string
}

type dialogView struct {
	d   *qt.QDialog
	sig string
	reg *registry
	box *qt.QVBoxLayout
}

// run opens the Qt window for the program driven by h and returns when it is closed.
func run(h *host.Host, a *app.App) error {
	args := os.Args
	qt.NewQApplication(args)
	w := &window{h: h, slots: map[string]*slotView{}, dialogs: map[string]*dialogView{}, buttons: map[string]*qt.QAbstractButton{}, seq: -1}
	w.build()
	w.mw.Show()
	timer := qt.NewQTimer()
	timer.OnTimeout(func() {
		if w.pump() || a.Quitting() {
			qt.QCoreApplication_Quit()
		}
	})
	timer.Start(16)
	if sp := os.Getenv("AFVI_QT_SCRIPT"); sp != "" {
		w.runScript(sp)
	}
	qt.QApplication_Exec()
	return nil
}

func (w *window) build() {
	mw := qt.NewQMainWindow2()
	w.mw = mw
	mw.Resize(1600, 1000)
	central := qt.NewQWidget2()
	root := qt.NewQVBoxLayout(central)
	root.SetContentsMargins(2, 2, 2, 2)
	root.SetSpacing(2)
	for _, name := range []string{"toolbar", "tabs", "sub"} {
		holder := qt.NewQWidget2()
		hl := qt.NewQVBoxLayout(holder)
		hl.SetContentsMargins(0, 0, 0, 0)
		root.AddWidget(holder)
		w.slots[name] = &slotView{holder: hl}
	}
	cols := qt.NewQWidget2()
	cl := qt.NewQHBoxLayout(cols)
	cl.SetContentsMargins(0, 0, 0, 0)
	w.left = w.column("left", 340)
	w.viewer = newViewer(w.h)
	w.right = w.column("right", 360)
	cl.AddWidget(w.left)
	cl.AddWidget2(w.viewer.w, 1)
	cl.AddWidget(w.right)
	root.AddWidget2(cols, 1)
	mw.SetCentralWidget(central)

	sb := mw.StatusBar()
	w.hover = qt.NewQLabel3("")
	w.hover.SetMinimumWidth(260)
	sb.AddPermanentWidget(w.hover.QWidget)
	for i := 0; i < 3; i++ {
		l := qt.NewQLabel3("")
		l.SetMinimumWidth(110)
		w.status = append(w.status, l)
		sb.AddPermanentWidget(l.QWidget)
	}
	w.menuBar = qt.NewQMenuBar2()
	mw.SetMenuBar(w.menuBar)
	mw.OnKeyPressEvent(func(super func(*qt.QKeyEvent), ev *qt.QKeyEvent) { w.key(ev) })
}

func (w *window) column(name string, width int) *qt.QWidget {
	sa := qt.NewQScrollArea2()
	sa.SetWidgetResizable(true)
	sa.SetFixedWidth(width)
	sa.SetHorizontalScrollBarPolicy(qt.ScrollBarAlwaysOff)
	w.slots[name] = &slotView{scroll: sa}
	return sa.QWidget
}

var keyNames = map[qt.Key]string{
	qt.Key_F1: "F1", qt.Key_F5: "F5", qt.Key_F11: "F11", qt.Key_Escape: "Escape",
	qt.Key_Backspace: "Backspace", qt.Key_Delete: "Delete", qt.Key_Return: "Enter", qt.Key_Enter: "Enter",
}

func (w *window) key(ev *qt.QKeyEvent) {
	k := qt.Key(ev.Key())
	name, ok := keyNames[k]
	if ev.Modifiers()&qt.ControlModifier != 0 && k == qt.Key_Z {
		name, ok = "Ctrl+Z", true
	}
	if !ok {
		return
	}
	if fw := qt.QApplication_FocusWidget(); fw != nil {
		cn := fw.MetaObject().ClassName()
		if strings.Contains(cn, "LineEdit") || strings.Contains(cn, "SpinBox") {
			if name == "Backspace" || name == "Delete" || name == "Enter" || name == "Ctrl+Z" {
				return
			}
		}
	}
	w.h.Do(ui.Cmd{ID: "key:" + name, Op: "key"})
}

// pump applies the newest frame (if any). It reports whether the window should close.
func (w *window) pump() bool {
	seq, body := w.h.Latest()
	if seq == w.seq || body == nil {
		return false
	}
	w.seq = seq
	var f frameMsg
	if err := json.Unmarshal(body, &f); err != nil {
		fmt.Fprintln(os.Stderr, "qtui: bad frame:", err)
		return false
	}
	w.apply(&f)
	q, _ := f.Meta["quit"].(bool)
	return q
}

func (w *window) apply(f *frameMsg) {
	if f.Title != w.lastTit {
		w.lastTit = f.Title
		w.mw.SetWindowTitle(f.Title)
	}
	th, _ := f.Meta["theme"].(string)
	px, _ := f.Meta["fontPx"].(float64)
	if th != w.theme || px != w.fontPx {
		w.theme, w.fontPx = th, px
		themeOf = th
		applyStyle(th, px)
	}
	focus, _ := f.Meta["focusView"].(bool)
	if focus != w.focus {
		w.focus = focus
		w.left.SetVisible(!focus)
		w.right.SetVisible(!focus)
	}
	for _, name := range []string{"toolbar", "tabs", "sub", "left", "right"} {
		w.slot(name, f.Slots[name])
	}
	w.applyMenu(f.Slots["menu"])
	w.applyPalette(f.Slots["palette"])
	w.applyStatus(f.Slots["status"], f.Viewer)
	w.applyDialogs(f.Slots["dialogs"])
	if m := f.Slots["message"]; m != nil && len(m.C) > 0 {
		w.viewer.message = m.C[0].L
	} else {
		w.viewer.message = ""
	}
	w.viewer.update(f.Viewer)
}

// slot rebuilds a region when its structure changed and refreshes values otherwise.
func (w *window) slot(name string, n *ui.Node) {
	sv := w.slots[name]
	if n == nil || len(n.C) == 0 {
		if sv.cur != nil {
			w.drop(sv)
		}
		return
	}
	var sb strings.Builder
	signature(n, &sb)
	if sig := sb.String(); sig != sv.sig || sv.cur == nil {
		sv.sig = sig
		sv.reg = &registry{win: w, upd: map[string]func(*ui.Node){}}
		syncing = true
		nw := sv.reg.build(n)
		syncing = false
		if sv.scroll != nil {
			pos := sv.scroll.VerticalScrollBar().Value()
			sv.scroll.SetWidget(nw)
			sv.scroll.VerticalScrollBar().SetValue(pos)
		} else {
			if sv.cur != nil {
				sv.holder.RemoveWidget(sv.cur)
				sv.cur.DeleteLater()
			}
			sv.holder.AddWidget(nw)
		}
		sv.cur = nw
	}
	sv.reg.refresh(n)
}

func (w *window) drop(sv *slotView) {
	if sv.scroll != nil {
		sv.scroll.SetWidget(qt.NewQWidget2())
	} else if sv.cur != nil {
		sv.holder.RemoveWidget(sv.cur)
		sv.cur.DeleteLater()
	}
	sv.cur, sv.sig = nil, ""
}

func (w *window) applyMenu(n *ui.Node) {
	if n == nil {
		return
	}
	var sb strings.Builder
	signature(n, &sb)
	fmt.Fprintf(&sb, "%v", menuState(n))
	if sig := sb.String(); sig == w.menuSig {
		return
	} else {
		w.menuSig = sig
	}
	w.menuBar.Clear()
	for _, m := range n.C {
		menu := qt.NewQMenu3(m.L)
		w.menuBar.AddMenu(menu)
		for _, it := range m.C {
			if it.K == ui.KMenuSep {
				menu.AddSeparator()
				continue
			}
			act := qt.NewQAction2(it.L)
			menu.AddAction(act)
			if sc := ps(it, "shortcut"); sc != "" {
				act.SetShortcut(qt.NewQKeySequence2(sc))
			}
			act.SetEnabled(p(it, "enabled") != false)
			if _, isBool := it.V.(bool); isBool {
				act.SetCheckable(true)
				act.SetChecked(vb(it))
			}
			id := it.ID
			act.OnTriggered(func() { w.h.Do(ui.Cmd{ID: id, Op: "click"}) })
		}
	}
}

func menuState(n *ui.Node) []any {
	var out []any
	for _, m := range n.C {
		for _, it := range m.C {
			out = append(out, it.V, it.P["enabled"])
		}
	}
	return out
}

func (w *window) applyPalette(n *ui.Node) {
	if n == nil || len(n.C) == 0 {
		if w.palette != nil {
			w.palette.Hide()
		}
		return
	}
	items, _ := p(n.C[0], "items").([]any)
	sig := fmt.Sprintf("%v", items)
	if sig == w.palSig && w.palette != nil {
		w.palette.Show()
		return
	}
	w.palSig = sig
	if w.palette != nil {
		w.palette.DeleteLater()
	}
	pw := qt.NewQWidget(w.viewer.w)
	l := qt.NewQVBoxLayout(pw)
	l.SetContentsMargins(3, 3, 3, 3)
	l.SetSpacing(2)
	glyph := map[string]string{"pick": "➤", "rect": "▭", "ellipse": "◯", "polygon": "⬠", "focus": "⛶"}
	for _, it := range items {
		m, _ := it.(map[string]any)
		id, _ := m["id"].(string)
		icon, _ := m["icon"].(string)
		tip, _ := m["tip"].(string)
		active, _ := m["active"].(bool)
		enabled, _ := m["enabled"].(bool)
		b := qt.NewQToolButton2()
		b.SetText(glyph[icon])
		b.SetFixedSize2(34, 34)
		b.SetCheckable(true)
		b.SetChecked(active)
		b.SetEnabled(enabled)
		b.SetToolTip(tip)
		b.OnClicked(func() { w.h.Do(ui.Cmd{ID: id, Op: "click"}) })
		l.AddWidget(b.QWidget)
	}
	pw.AdjustSize()
	w.palette = pw
	w.viewer.paletteHolder = pw
	w.viewer.positionPalette()
	pw.Show()
}

func (w *window) applyStatus(n *ui.Node, v *ui.ViewerOut) {
	hint := ""
	if n != nil && len(n.C) > 0 {
		st := n.C[0]
		hint = st.L
		for i, c := range st.C {
			if i < len(w.status) {
				w.status[i].SetText(c.L)
			}
		}
	}
	w.mw.StatusBar().ShowMessage(hint)
	if v != nil {
		w.hover.SetText(v.Hover)
	}
}

func (w *window) applyDialogs(n *ui.Node) {
	seen := map[string]bool{}
	if n != nil {
		for _, d := range n.C {
			seen[d.ID] = true
			dv := w.dialogs[d.ID]
			var sb strings.Builder
			signature(d, &sb)
			sig := sb.String()
			if dv == nil {
				dlg := qt.NewQDialog(w.mw.QWidget)
				id := d.ID
				dlg.OnRejected(func() {
					if !syncing {
						w.h.Do(ui.Cmd{ID: "close:" + id, Op: "click"})
					}
				})
				box := qt.NewQVBoxLayout(dlg.QWidget)
				dv = &dialogView{d: dlg, box: box}
				w.dialogs[d.ID] = dv
				dlg.SetWindowTitle(d.L)
				dlg.SetMinimumWidth(int(pf(d, "w", 420)))
				dlg.SetModal(true)
				dlg.Show()
			}
			if sig != dv.sig {
				dv.sig = sig
				dv.reg = &registry{win: w, upd: map[string]func(*ui.Node){}}
				for dv.box.Count() > 0 {
					it := dv.box.TakeAt(0)
					if wd := it.Widget(); wd != nil {
						wd.DeleteLater()
					}
				}
				syncing = true
				content := dv.reg.build(&ui.Node{K: ui.KRoot, C: d.C})
				syncing = false
				dv.box.AddWidget(content)
			}
			dv.reg.refresh(d)
		}
	}
	for id, dv := range w.dialogs {
		if !seen[id] {
			syncing = true
			dv.d.Close()
			dv.d.DeleteLater()
			syncing = false
			delete(w.dialogs, id)
		}
	}
}

// applyStyle gives the application the classic grey Win32 palette (or the dark one) on the
// Fusion style; Qt draws the controls natively.
func applyStyle(theme string, px float64) {
	qt.QApplication_SetStyle(qt.QStyleFactory_Create("Fusion"))
	face, text, base, sel := "#d4d0c8", "#000000", "#ffffff", "#0a246a"
	if theme == "dark" {
		face, text, base, sel = "#3a3a3a", "#e0e0e0", "#1c1c1c", "#1d3461"
	}
	pal := qt.NewQPalette()
	set := func(role qt.QPalette__ColorRole, c string) { pal.SetColor2(role, parseColor(c)) }
	set(qt.QPalette__Window, face)
	set(qt.QPalette__Button, face)
	set(qt.QPalette__WindowText, text)
	set(qt.QPalette__ButtonText, text)
	set(qt.QPalette__Text, text)
	set(qt.QPalette__Base, base)
	set(qt.QPalette__AlternateBase, face)
	set(qt.QPalette__Highlight, sel)
	set(qt.QPalette__HighlightedText, "#ffffff")
	qt.QApplication_SetPalette(pal)
	if px >= 10 {
		f := qt.QApplication_Font()
		f.SetPixelSize(int(px * 0.85))
		qt.QApplication_SetFont(f)
	}
}
