package app

import (
	"math"
	"strconv"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/ui"
)

// The compact label-left controls of the old Dear ImGui UI, now recorded into the frame
// tree (internal/ui) and rendered by win32-vue components. Every control takes a stable id
// separate from its (translated) label so switching language keeps widget state.

// F formats a number with fixed digits; NaN/Inf render as an em dash.
func F(v float64, digits int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "—"
	}
	return strconv.FormatFloat(v, 'f', digits, 64)
}

// Section is a collapsible group (open by default); SectionClosed starts collapsed. The
// controls drawn until the next Section form its body.
func Section(title, id string) bool       { return ui.Section(title, id) }
func SectionClosed(title, id string) bool { return ui.SectionClosed(title, id) }

func SliderF64(label, id string, v *float64, min, max float64, format string) bool {
	return ui.SliderF64(label, id, v, min, max, format)
}
func SliderInt(label, id string, v *int, min, max int) bool {
	return ui.SliderInt(label, id, v, min, max)
}
func SliderInputF64(label, id string, v *float64, min, max float64, format string) bool {
	return ui.SliderInputF64(label, id, v, min, max, format)
}
func SliderInputInt(label, id string, v *int, min, max int) bool {
	return ui.SliderInputInt(label, id, v, min, max)
}
func InputF64(label, id string, v *float64, step float64, format string) bool {
	return ui.InputF64(label, id, v, step, format)
}
func Check(label, id string, v *bool) bool { return ui.Check(label, id, v) }
func Combo(label, id string, current *int, items []string) bool {
	return ui.Combo(label, id, current, items)
}
func InputText(label, id string, buf *string, hint string) bool {
	return ui.InputText(label, id, buf, hint)
}

// WrapSameLine keeps the next button on the current line; the browser wraps rows itself.
func WrapSameLine(string) { ui.SameLine() }

// Button is a push button with a stable id.
func Button(label, id string) bool { return ui.Button(label, id) }

// ButtonOn is a push button that can be disabled.
func ButtonOn(label, id string, enabled bool) bool { return ui.ButtonEx(label, id, enabled, false) }

// ToolButton is a latching tool button: drawn pressed while active.
func ToolButton(label, id string, active bool) bool { return ui.ToolButton(label, id, active) }

// GroupBox draws an etched frame with the title on the top border and runs body inside.
func GroupBox(title, id string, body func()) { ui.GroupBox(title, id, body) }

// TabStrip draws property-sheet tabs and returns the clicked index, or -1.
func TabStrip(id string, labels []string, active int) int { return ui.TabStrip(id, labels, active) }

// Note is dim wrapped helper text; Warn is emphasized wrapped warning text.
func Note(text string) { ui.Note(text) }
func Warn(text string) { ui.Warn(text) }

// Text appends plain text; SameLine puts the next control on the same line.
func Text(s string) { ui.Text(s) }
func SameLine()     { ui.SameLine() }

// Badge draws an OK / MANUAL / NG indicator lamp.
func Badge(level core.VerdictLevel) {
	switch level {
	case core.VerdictManual:
		ui.Badge("MANUAL", "warn")
	case core.VerdictNG:
		ui.Badge("NG", "ng")
	default:
		ui.Badge("OK", "ok")
	}
}

// KVTable shows label/value pairs.
func KVTable(id string, rows [][2]string) { ui.KVTable(id, rows) }

// DataTable renders a scrollable grid; height is in rows. It returns the clicked row or -1.
func DataTable(id string, headers []string, rows int, height int, cell func(row, col int) string, selected int) int {
	return ui.DataTable(id, headers, rows, height, func(r, c int) ui.Cell { return ui.Cell{T: cell(r, c)} }, selected)
}

// DataTableColored is DataTable whose cells may carry a tone (dim|warn|good|bad) and a
// colour chip.
func DataTableColored(id string, headers []string, rows int, height int, cell func(row, col int) ui.Cell, selected int) int {
	return ui.DataTable(id, headers, rows, height, cell, selected)
}

// Histogram draws a 256-bin log-scale GV histogram, optionally highlighting one bin.
func Histogram(id string, hist *[256]int, highlight int, height int) {
	ui.Histogram(id, hist, highlight, height)
}

// Heatmap draws an NCC surface (blue → yellow ramp) with the peak circled.
func Heatmap(id string, s *core.NccSurface, size int) {
	if s == nil || s.Width <= 0 || s.Height <= 0 {
		return
	}
	data := make([]byte, len(s.Values))
	peak, pi := float32(-math.MaxFloat32), 0
	for i, v := range s.Values {
		t := (v + 1) / 2
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
		data[i] = byte(t * 255)
		if v > peak {
			peak, pi = v, i
		}
	}
	ui.Add(&ui.Node{K: ui.KHeat, ID: id, P: map[string]any{"w": s.Width, "h": s.Height, "data": data, "peak": pi, "size": size}})
}

// Swatch draws a colour chip.
func Swatch(color uint32) { ui.Swatch(css(color)) }
