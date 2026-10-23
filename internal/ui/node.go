// Package ui records one frame of a panel's controls as a tree and takes the user's
// actions back as commands. It is the Go half of the Wails front end: panels keep the
// immediate-mode shape they had with Dear ImGui (`if ui.Button(...) {...}`,
// `ui.SliderF64(..., &v, ...)` returns "changed"), but instead of drawing they append
// nodes that the Vue renderer (frontend/) turns into win32-vue components.
//
// Everything here runs on the single UI goroutine of the host; there is no locking.
// The package is standard-library only so it tests natively on Linux.
package ui

// Node kinds understood by the front end (frontend/src/render/Node.vue).
const (
	KRoot     = "root"
	KRow      = "row"      // children laid out on one line (built from SameLine)
	KText     = "text"     // L = text; P.tone = dim|warn|good|bad|""
	KNote     = "note"     // dim wrapped text
	KWarn     = "warn"     // warning wrapped text
	KSep      = "sep"      // horizontal etched line
	KSpacer   = "spacer"   // vertical gap, P.h px
	KSection  = "section"  // collapsible group; V = open, children = body
	KGroup    = "group"    // etched group box, L = title
	KButton   = "button"   // L = label; P.enabled, P.def
	KTool     = "tool"     // latching tool button; V = active
	KCheck    = "check"    // V = bool
	KRadio    = "radio"    // V = bool (selected)
	KSlider   = "slider"   // V = number, P.min P.max P.fmt P.int P.field
	KInput    = "input"    // numeric field; V = number, P.step P.fmt
	KText1    = "edit"     // text field; V = string, P.hint P.password
	KCombo    = "combo"    // V = index, P.items
	KTabs     = "tabs"     // V = active, P.items
	KSubTabs  = "subtabs"  // segmented strip with colour chips; P.items = [{l,color,count}]
	KTable    = "table"    // P.headers, P.rows ([][]cell), V = selected row, P.height
	KKV       = "kv"       // P.rows = [][2]string
	KBadge    = "badge"    // L = text, P.level
	KSwatch   = "swatch"   // colour chip, P.color = "#rrggbb"
	KHist     = "hist"     // P.bins (256), P.mark
	KHeat     = "heat"     // P.w P.h P.data (base64 gray), P.size
	KProgress = "progress" // V = 0..1
	KDialog   = "dialog"   // modal window, L = title, P.w; P.closeable
	KMenu     = "menu"     // menu bar entry, L = title, children = items
	KMenuItem = "mitem"    // L, P.shortcut, V = checked, P.enabled
	KMenuSep  = "msep"
	KGrid     = "grid" // label/value pairs in P.cols columns
)

// Node is one control. Field names are short because whole trees are serialised each
// frame the UI changed.
type Node struct {
	K  string         `json:"k"`
	ID string         `json:"id,omitempty"`
	L  string         `json:"l,omitempty"`
	V  any            `json:"v,omitempty"`
	P  map[string]any `json:"p,omitempty"`
	C  []*Node        `json:"c,omitempty"`

	same bool // continues the previous node's line (SameLine)
}

// Prop sets one property and returns the node.
func (n *Node) Prop(k string, v any) *Node {
	if n.P == nil {
		n.P = map[string]any{}
	}
	n.P[k] = v
	return n
}

// Cmd is one user action coming from the front end.
type Cmd struct {
	ID    string  `json:"id"`          // control id, or a global name such as "key:F5", "menu:file.exit"
	Op    string  `json:"op"`          // set | click | toggle | key | ptr | view
	V     any     `json:"v,omitempty"` // new value for set
	X     float64 `json:"x,omitempty"`
	Y     float64 `json:"y,omitempty"`
	K     string  `json:"k,omitempty"` // pointer phase: down|move|up|rclick
	Ctrl  bool    `json:"ctrl,omitempty"`
	Shift bool    `json:"shift,omitempty"`
}
