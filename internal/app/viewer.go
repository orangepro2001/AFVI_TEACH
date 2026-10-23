package app

import "afvi/teach-imgui/internal/ui"

// Viewer is the shared image canvas (see internal/ui.Viewer): the front end owns zoom and
// pan, Go owns the image and the overlay shapes.
type Viewer = ui.Viewer

// Overlay is what the active panel draws on, in image coordinates.
type Overlay = ui.Overlay

// NewViewer returns an empty viewer.
func NewViewer() *Viewer { return ui.NewViewer() }

// PointerKind is the phase of a tool drag on the viewer.
type PointerKind int

const (
	PointerDown PointerKind = iota
	PointerMove
	PointerUp
	// PointerRightClick is reported once on the release of a right click (under 4 px of
	// movement, no Shift).
	PointerRightClick
)

// PointerEvent carries image coordinates (full-resolution image pixels of the shown image).
type PointerEvent struct {
	Kind PointerKind
	X, Y float64
	// Ctrl is true while the Ctrl key is held; used for add-to-selection.
	Ctrl bool
}

func pointerEvent(p ui.Pointer) (PointerEvent, bool) {
	var k PointerKind
	switch p.Kind {
	case ui.PtrDown:
		k = PointerDown
	case ui.PtrMove:
		k = PointerMove
	case ui.PtrUp:
		k = PointerUp
	case ui.PtrRightClick:
		k = PointerRightClick
	default:
		return PointerEvent{}, false
	}
	return PointerEvent{Kind: k, X: p.X, Y: p.Y, Ctrl: p.Ctrl}, true
}
