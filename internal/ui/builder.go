package ui

import (
	"encoding/json"
	"math"
	"strconv"
)

// Builder records one frame. Create one per host, call Begin before the panels draw and
// Finish afterwards. The package-level functions (Button, SliderF64, ...) write to the
// current builder, which Use installs; this keeps panel code close to its ImGui form.
type Builder struct {
	stack   []*frame
	cmds    map[string]Cmd // last set/click per control id, consumed by the widget
	keys    map[string]bool
	events  []Cmd // pointer/view commands, in order
	open    map[string]bool
	openSet map[string]bool
	sameReq bool
}

type frame struct {
	n       *Node
	section *Node // open Section container, closed by the next Section or End
}

var cur *Builder

// Use makes b the builder the package-level functions write to.
func Use(b *Builder) { cur = b }

// Current returns the builder installed by Use (nil before any).
func Current() *Builder { return cur }

// NewBuilder returns an empty builder.
func NewBuilder() *Builder {
	return &Builder{cmds: map[string]Cmd{}, keys: map[string]bool{}, open: map[string]bool{}, openSet: map[string]bool{}}
}

// Feed queues the commands received since the last frame. Widgets consume the value
// commands while the next frame is built; unconsumed ones are dropped by Begin.
func (b *Builder) Feed(cmds []Cmd) {
	for _, c := range cmds {
		switch c.Op {
		case "set", "click":
			b.cmds[c.ID] = c
		case "toggle":
			b.open[c.ID] = !b.isOpen(c.ID, true)
			b.openSet[c.ID] = true
		case "key":
			b.keys[c.ID] = true
		default:
			b.events = append(b.events, c)
		}
	}
}

func (b *Builder) isOpen(id string, def bool) bool {
	if b.openSet[id] {
		return b.open[id]
	}
	return def
}

// SetOpen overrides the open state of a Section (self-test, state restore).
func (b *Builder) SetOpen(id string, open bool) { b.open[id] = open; b.openSet[id] = true }

// Begin starts a frame and returns its root node.
func (b *Builder) Begin() *Node {
	root := &Node{K: KRoot}
	b.stack = []*frame{{n: root}}
	b.sameReq = false
	return root
}

// Finish ends one root: open containers are closed and same-line runs become rows. A
// program may build several roots (menu, left column, ...) in one frame; Drop ends the
// frame and forgets the commands no widget consumed.
func (b *Builder) Finish() *Node {
	for len(b.stack) > 1 {
		b.End()
	}
	root := b.stack[0].n
	b.stack = nil
	rowify(root)
	return root
}

// Drop ends the frame: unconsumed value commands and keys are discarded.
func (b *Builder) Drop() {
	b.cmds = map[string]Cmd{}
	b.keys = map[string]bool{}
}

// TakeEvents returns and clears the pointer/view events received since the last call.
func (b *Builder) TakeEvents() []Cmd { e := b.events; b.events = nil; return e }

func (b *Builder) top() *frame { return b.stack[len(b.stack)-1] }

func (b *Builder) add(n *Node) *Node {
	n.same = b.sameReq
	b.sameReq = false
	f := b.top()
	parent := f.n
	if f.section != nil && n.K != KSection {
		parent = f.section
	}
	parent.C = append(parent.C, n)
	return n
}

// Push adds n and makes it the container for what follows, until End.
func (b *Builder) Push(n *Node) *Node {
	b.add(n)
	b.stack = append(b.stack, &frame{n: n})
	return n
}

// End closes the container opened by Push.
func (b *Builder) End() {
	if len(b.stack) > 1 {
		b.stack = b.stack[:len(b.stack)-1]
	}
}

// rowify groups runs of same-line nodes into row nodes, recursively.
func rowify(n *Node) {
	for _, c := range n.C {
		rowify(c)
	}
	if len(n.C) < 2 {
		return
	}
	out := make([]*Node, 0, len(n.C))
	for i := 0; i < len(n.C); {
		j := i + 1
		for j < len(n.C) && n.C[j].same {
			j++
		}
		if j-i > 1 {
			row := &Node{K: KRow, C: append([]*Node(nil), n.C[i:j]...)}
			for _, c := range row.C {
				c.same = false
			}
			out = append(out, row)
		} else {
			out = append(out, n.C[i])
		}
		i = j
	}
	n.C = out
}

// ---- package-level recording API ------------------------------------------------

func id2(id string) string { return id }

// SameLine puts the next node on the line of the previous one.
func SameLine() { cur.sameReq = true }

// Text appends plain text.
func Text(s string) *Node { return cur.add(&Node{K: KText, L: s}) }

// TextTone appends text with a tone: dim, warn, good or bad.
func TextTone(s, tone string) *Node {
	return cur.add(&Node{K: KText, L: s, P: map[string]any{"tone": tone}})
}

// Note is dim wrapped text; Warn is wrapped warning text.
func Note(s string) *Node { return cur.add(&Node{K: KNote, L: s}) }
func Warn(s string) *Node { return cur.add(&Node{K: KWarn, L: s}) }

// Separator appends an etched line; Spacing a small gap.
func Separator() *Node { return cur.add(&Node{K: KSep}) }
func Spacing() *Node   { return cur.add(&Node{K: KSpacer, P: map[string]any{"h": 6}}) }

// Section is a collapsible group (open by default); the nodes appended until the next
// Section or the end of the enclosing container become its body.
func Section(title, id string) bool { return section(title, id, true) }

// SectionClosed starts collapsed.
func SectionClosed(title, id string) bool { return section(title, id, false) }

func section(title, id string, def bool) bool {
	b := cur
	f := b.top()
	f.section = nil
	open := b.isOpen("sec-"+id, def)
	n := &Node{K: KSection, ID: "sec-" + id, L: title, V: open}
	b.add(n)
	f.section = n
	return open
}

// GroupBox is an etched frame with a title; body runs inside it.
func GroupBox(title, id string, body func()) {
	n := cur.Push(&Node{K: KGroup, ID: "gb-" + id, L: title})
	body()
	_ = n
	cur.End()
}

// Container pushes an anonymous or kinded container; pair with EndContainer.
func Container(kind, id string) *Node { return cur.Push(&Node{K: kind, ID: id}) }
func EndContainer()                   { cur.End() }

// Button is a push button; it reports whether it was clicked since the last frame.
func Button(label, id string) bool { return ButtonEx(label, id, true, false) }

// ButtonEx is Button with an enabled flag and the default-button look.
func ButtonEx(label, id string, enabled, def bool) bool {
	cur.add(&Node{K: KButton, ID: id, L: label, P: map[string]any{"enabled": enabled, "def": def}})
	c, ok := cur.cmds[id]
	if ok && c.Op == "click" {
		delete(cur.cmds, id)
		return enabled
	}
	return false
}

// ToolButton is a latching tool button, drawn pressed while active.
func ToolButton(label, id string, active bool) bool {
	cur.add(&Node{K: KTool, ID: id, L: label, V: active})
	c, ok := cur.cmds[id]
	if ok && c.Op == "click" {
		delete(cur.cmds, id)
		return true
	}
	return false
}

// Check is a checkbox.
func Check(label, id string, v *bool) bool {
	cur.add(&Node{K: KCheck, ID: id, L: label, V: *v})
	if x, ok := takeBool(id); ok && x != *v {
		*v = x
		return true
	}
	return false
}

// Radio draws a radio button and reports a click on it.
func Radio(label, id string, selected bool) bool {
	cur.add(&Node{K: KRadio, ID: id, L: label, V: selected})
	c, ok := cur.cmds[id]
	if ok && c.Op == "click" {
		delete(cur.cmds, id)
		return true
	}
	return false
}

// SliderF64 edits a float64 with a slider.
func SliderF64(label, id string, v *float64, min, max float64, format string) bool {
	return slider(label, id, v, min, max, format, false, false)
}

// SliderInputF64 adds a typed field next to the slider.
func SliderInputF64(label, id string, v *float64, min, max float64, format string) bool {
	return slider(label, id, v, min, max, format, false, true)
}

// SliderInt and SliderInputInt are the integer forms.
func SliderInt(label, id string, v *int, min, max int) bool {
	f := float64(*v)
	if slider(label, id, &f, float64(min), float64(max), "%d", true, false) {
		*v = int(math.Round(f))
		return true
	}
	return false
}
func SliderInputInt(label, id string, v *int, min, max int) bool {
	f := float64(*v)
	if slider(label, id, &f, float64(min), float64(max), "%d", true, true) {
		*v = int(math.Round(f))
		return true
	}
	return false
}

func slider(label, id string, v *float64, min, max float64, format string, isInt, field bool) bool {
	cur.add(&Node{K: KSlider, ID: id, L: label, V: num(*v), P: map[string]any{
		"min": min, "max": max, "fmt": format, "int": isInt, "field": field}})
	if x, ok := takeNum(id); ok {
		x = math.Max(min, math.Min(max, x))
		if isInt {
			x = math.Round(x)
		}
		if x != *v {
			*v = x
			return true
		}
	}
	return false
}

// InputF64 is a numeric field with step buttons.
func InputF64(label, id string, v *float64, step float64, format string) bool {
	cur.add(&Node{K: KInput, ID: id, L: label, V: num(*v), P: map[string]any{"step": step, "fmt": format}})
	if x, ok := takeNum(id); ok && x != *v {
		*v = x
		return true
	}
	return false
}

// InputText is a single-line text field.
func InputText(label, id string, buf *string, hint string) bool {
	return InputTextEx(label, id, buf, hint, false)
}

// InputTextEx can mask the content (password).
func InputTextEx(label, id string, buf *string, hint string, password bool) bool {
	cur.add(&Node{K: KText1, ID: id, L: label, V: *buf, P: map[string]any{"hint": hint, "password": password}})
	if c, ok := cur.cmds[id]; ok && c.Op == "set" {
		delete(cur.cmds, id)
		if s, ok := c.V.(string); ok && s != *buf {
			*buf = s
			return true
		}
	}
	return false
}

// Combo selects one of items by index.
func Combo(label, id string, current *int, items []string) bool {
	cur.add(&Node{K: KCombo, ID: id, L: label, V: *current, P: map[string]any{"items": items}})
	if x, ok := takeNum(id); ok {
		i := int(x)
		if i != *current && i >= 0 && i < len(items) {
			*current = i
			return true
		}
	}
	return false
}

// TabStrip draws property-sheet tabs and returns the clicked index, or -1.
func TabStrip(id string, labels []string, active int) int {
	cur.add(&Node{K: KTabs, ID: id, V: active, P: map[string]any{"items": labels}})
	if x, ok := takeNum(id); ok {
		return int(x)
	}
	return -1
}

// SubTabItem is one entry of a SubTabStrip.
type SubTabItem struct {
	Label string `json:"l"`
	Color string `json:"color,omitempty"`
	Count string `json:"count,omitempty"`
}

// SubTabStrip is the segmented second-level strip; returns the clicked index or -1.
func SubTabStrip(id string, items []SubTabItem, active int, note string) int {
	cur.add(&Node{K: KSubTabs, ID: id, V: active, L: note, P: map[string]any{"items": items}})
	if x, ok := takeNum(id); ok {
		return int(x)
	}
	return -1
}

// Cell is one table cell; Tone colours the text (dim|warn|good|bad|"") and Color is an
// optional "#rrggbb" chip drawn before the text.
type Cell struct {
	T     string `json:"t"`
	Tone  string `json:"tone,omitempty"`
	Color string `json:"color,omitempty"`
}

// DataTable draws a table and returns the clicked row, or -1. height is in rows.
func DataTable(id string, headers []string, rows int, height int, cell func(row, col int) Cell, selected int) int {
	data := make([][]Cell, rows)
	for r := range data {
		data[r] = make([]Cell, len(headers))
		for c := range headers {
			data[r][c] = cell(r, c)
		}
	}
	cur.add(&Node{K: KTable, ID: id, V: selected, P: map[string]any{"headers": headers, "rows": data, "height": height}})
	if x, ok := takeNum(id); ok {
		return int(x)
	}
	return -1
}

// KVTable draws label/value rows.
func KVTable(id string, rows [][2]string) *Node {
	return cur.add(&Node{K: KKV, ID: id, P: map[string]any{"rows": rows}})
}

// Badge draws a verdict badge; level is ok|ng|warn|none.
func Badge(text, level string) *Node {
	return cur.add(&Node{K: KBadge, L: text, P: map[string]any{"level": level}})
}

// Swatch draws a colour chip ("#rrggbb").
func Swatch(color string) *Node { return cur.add(&Node{K: KSwatch, P: map[string]any{"color": color}}) }

// Histogram draws a 256-bin histogram with an optional highlighted bin (-1 = none).
func Histogram(id string, hist *[256]int, highlight int, height int) *Node {
	bins := make([]int, 256)
	copy(bins, hist[:])
	return cur.add(&Node{K: KHist, ID: id, P: map[string]any{"bins": bins, "mark": highlight, "height": height}})
}

// Dialog opens a modal window; body runs inside it. closed is true when the user pressed
// the close button (command "close:<id>").
func Dialog(title, id string, width int, body func()) (closed bool) {
	cur.Push(&Node{K: KDialog, ID: id, L: title, P: map[string]any{"w": width}})
	body()
	cur.End()
	if _, ok := cur.cmds["close:"+id]; ok {
		delete(cur.cmds, "close:"+id)
		return true
	}
	return false
}

// Menu starts a menu bar entry; MenuItem adds an item and reports a click.
func Menu(title, id string, body func()) {
	cur.Push(&Node{K: KMenu, ID: id, L: title})
	body()
	cur.End()
}

// MenuItem is one menu entry. checked < 0 means no check mark.
func MenuItem(label, id, shortcut string, checked int, enabled bool) bool {
	n := &Node{K: KMenuItem, ID: id, L: label, P: map[string]any{"shortcut": shortcut, "enabled": enabled}}
	if checked >= 0 {
		n.V = checked == 1
	}
	cur.add(n)
	if c, ok := cur.cmds[id]; ok && c.Op == "click" {
		delete(cur.cmds, id)
		return enabled
	}
	return false
}

// MenuSeparator draws a menu divider.
func MenuSeparator() { cur.add(&Node{K: KMenuSep}) }

// KeyPressed reports (and consumes) a key command such as "F5" or "Escape".
func KeyPressed(name string) bool {
	if cur.keys["key:"+name] {
		delete(cur.keys, "key:"+name)
		return true
	}
	return false
}

// Clicked reports (and consumes) a click on a global control such as a toolbar button
// whose node the caller added with Add.
func Clicked(id string) bool {
	if c, ok := cur.cmds[id]; ok && c.Op == "click" {
		delete(cur.cmds, id)
		return true
	}
	return false
}

// Add appends a ready-made node (for composite widgets owned by a panel).
func Add(n *Node) *Node { return cur.add(n) }

// TakeSet returns the pending "set" value of a control id, consuming it.
func TakeSet(id string) (any, bool) {
	c, ok := cur.cmds[id]
	if !ok || c.Op != "set" {
		return nil, false
	}
	delete(cur.cmds, id)
	return c.V, true
}

func takeNum(id string) (float64, bool) {
	c, ok := cur.cmds[id]
	if !ok {
		return 0, false
	}
	delete(cur.cmds, id)
	switch v := c.V.(type) {
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	}
	if c.Op == "click" {
		return c.X, true
	}
	return 0, false
}

func takeBool(id string) (bool, bool) {
	c, ok := cur.cmds[id]
	if !ok {
		return false, false
	}
	delete(cur.cmds, id)
	b, ok := c.V.(bool)
	return b, ok
}

// num keeps NaN/Inf out of the JSON.
func num(f float64) float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return f
}

// InputNum is a compact label-less numeric field (tables of values); the value is
// clamped to [min, max].
func InputNum(id string, v *float64, min, max, step float64, format string) bool {
	cur.add(&Node{K: "num", ID: id, V: num(*v), P: map[string]any{"min": min, "max": max, "step": step, "fmt": format}})
	if x, ok := takeNum(id); ok {
		x = math.Max(min, math.Min(max, x))
		if x != *v {
			*v = x
			return true
		}
	}
	return false
}
