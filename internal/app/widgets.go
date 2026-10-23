package app

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/core"
)

// Compact label-left controls for dense industrial panels. Every control takes a stable
// id separate from its (translated) label so switching language keeps widget state.

// labelWidth is the label column width, scaled from the 13 px design font.
func labelWidth() float32 { return sc(112) }

// groupInset is the extra right inset applied to label-left controls while they are
// drawn inside a GroupBox, so fields do not touch the etched frame.
var groupInset float32

// F formats a number with fixed digits; NaN/Inf render as an em dash.
func F(v float64, digits int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "—"
	}
	return strconv.FormatFloat(v, 'f', digits, 64)
}

func labelColumn(label string) {
	imgui.AlignTextToFramePadding()
	imgui.TextUnformatted(label)
	imgui.SameLineV(labelWidth(), -1)
	imgui.SetNextItemWidth(-1 - groupInset)
}

// Section is a collapsible group (open by default).
func Section(title, id string) bool {
	imgui.Dummy(imgui.NewVec2(0, sc(2)))
	face := PaletteVec4(CurrentPalette().Face)
	imgui.PushStyleColorVec4(imgui.ColHeader, face)
	imgui.PushStyleColorVec4(imgui.ColHeaderHovered, face)
	imgui.PushStyleColorVec4(imgui.ColHeaderActive, face)
	open := imgui.CollapsingHeaderTreeNodeFlagsV(title+"###sec-"+id, imgui.TreeNodeFlagsDefaultOpen)
	imgui.PopStyleColorV(3)
	ItemEdge(EdgeRaised)
	return open
}

// SectionClosed is a collapsible group that starts collapsed.
func SectionClosed(title, id string) bool {
	imgui.Dummy(imgui.NewVec2(0, sc(2)))
	face := PaletteVec4(CurrentPalette().Face)
	imgui.PushStyleColorVec4(imgui.ColHeader, face)
	imgui.PushStyleColorVec4(imgui.ColHeaderHovered, face)
	imgui.PushStyleColorVec4(imgui.ColHeaderActive, face)
	open := imgui.CollapsingHeaderTreeNodeFlagsV(title+"###sec-"+id, 0)
	imgui.PopStyleColorV(3)
	ItemEdge(EdgeRaised)
	return open
}

// trackbarEdit holds the ids of sliders whose value is being typed (Ctrl+click) instead of dragged.
var trackbarEdit = map[string]bool{}

// trackbar draws a Win32 trackbar (sunken groove, raised thumb) of the given size and returns
// the new position 0..1 and whether the user moved it. A click or drag sets the position under
// the cursor, like the page-jump of a classic trackbar thumb that follows the mouse.
func trackbar(id string, frac float32, size imgui.Vec2) (float32, bool) {
	imgui.InvisibleButton("##tb-"+id, size)
	min, max := imgui.ItemRectMin(), imgui.ItemRectMax()
	active := imgui.IsItemActive()
	thumbW, thumbH := sc(11), min2(size.Y, sc(22))
	span := max.X - min.X - thumbW
	if span < 1 {
		span = 1
	}
	changed := false
	if active {
		f := (imgui.MousePos().X - min.X - thumbW/2) / span
		if f < 0 {
			f = 0
		} else if f > 1 {
			f = 1
		}
		if f != frac {
			frac, changed = f, true
		}
	}
	dl := imgui.WindowDrawList()
	p := CurrentPalette()
	midY := (min.Y + max.Y) / 2
	grooveH := sc(4)
	DrawEdge(dl, imgui.NewVec2(min.X+thumbW/2-1, midY-grooveH/2), imgui.NewVec2(max.X-thumbW/2+1, midY+grooveH/2), EdgeSunken)
	tx := min.X + frac*span
	tmin := imgui.NewVec2(tx, midY-thumbH/2)
	tmax := imgui.NewVec2(tx+thumbW, midY+thumbH/2)
	FillRect(dl, tmin, tmax, p.Face)
	if active {
		DrawEdge(dl, tmin, tmax, EdgePressed)
	} else {
		DrawEdge(dl, tmin, tmax, EdgeRaised)
	}
	if imgui.IsItemFocused() {
		DottedRect(dl, imgui.NewVec2(min.X, min.Y), imgui.NewVec2(max.X, max.Y), TextColor())
	}
	return frac, changed
}

func min2(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// sliderValueW is the width reserved right of a trackbar for its value readout.
func sliderValueW() float32 { return sc(58) }

// sliderCore draws label, trackbar and value readout (or the typed field while Ctrl+click
// editing). It reports whether the value changed.
func sliderCore(label, id string, v *float64, min, max float64, format string, isInt bool, trailing float32) bool {
	labelColumn(label)
	avail := imgui.ContentRegionAvail().X
	readW := sliderValueW()
	trackW := avail - readW - imgui.CurrentStyle().ItemSpacing().X - trailing - groupInset
	if trackW < sc(40) {
		trackW = sc(40)
	}
	if trackbarEdit[id] {
		imgui.SetNextItemWidth(avail - groupInset - trailing)
		val := *v
		imgui.SetKeyboardFocusHere()
		done := imgui.InputDoubleV("##"+id+"-edit", &val, 0, 0, format, imgui.InputTextFlagsAutoSelectAll|imgui.InputTextFlagsEnterReturnsTrue)
		ItemEdge(EdgeSunken)
		if done || imgui.IsKeyPressedBool(imgui.KeyEscape) || (!imgui.IsItemActive() && !imgui.IsItemFocused() && imgui.IsMouseClickedBool(imgui.MouseButtonLeft)) {
			delete(trackbarEdit, id)
		}
		if done {
			val = math.Min(max, math.Max(min, val))
			if isInt {
				val = math.Round(val)
			}
			if val != *v {
				*v = val
				return true
			}
		}
		return false
	}
	frac := float32(0)
	if max > min {
		frac = float32((*v - min) / (max - min))
	}
	if frac < 0 {
		frac = 0
	} else if frac > 1 {
		frac = 1
	}
	nf, changed := trackbar(id, frac, imgui.NewVec2(trackW, imgui.FrameHeight()))
	if imgui.IsItemClicked() && imgui.CurrentIO().KeyCtrl() {
		trackbarEdit[id] = true
		changed = false
	}
	imgui.SameLine()
	imgui.AlignTextToFramePadding()
	text := F(*v, 2)
	if isInt {
		text = strconv.Itoa(int(math.Round(*v)))
	} else if format != "" {
		text = fmtSlider(*v, format)
	}
	tw := imgui.CalcTextSize(text).X
	if gap := readW - tw; gap > 0 {
		imgui.SetCursorPosX(imgui.CursorPosX() + gap)
	}
	imgui.TextUnformatted(text)
	if !changed {
		return false
	}
	nv := min + float64(nf)*(max-min)
	if isInt {
		nv = math.Round(nv)
	}
	if nv == *v {
		return false
	}
	*v = nv
	return true
}

// fmtSlider applies a printf-style float format such as "%.2f" or "%.0f".
func fmtSlider(v float64, format string) string {
	return fmt.Sprintf(format, v)
}

// SliderF64 edits a float64 with a trackbar (Ctrl+click to type a value).
func SliderF64(label, id string, v *float64, min, max float64, format string) bool {
	return sliderCore(label, id, v, min, max, format, false, 0)
}

// SliderInt edits an int with a trackbar.
func SliderInt(label, id string, v *int, min, max int) bool {
	f := float64(*v)
	if sliderCore(label, id, &f, float64(min), float64(max), "%d", true, 0) {
		*v = int(f)
		return true
	}
	return false
}

// SliderInputF64 edits a float64 with a trackbar and, to its right, a small numeric field in which the value can be typed.
func SliderInputF64(label, id string, v *float64, min, max float64, format string) bool {
	fieldW := sc(64)
	changed := sliderFieldCore(label, id, v, min, max, format, false, fieldW)
	imgui.SameLine()
	imgui.SetNextItemWidth(fieldW)
	val := *v
	if imgui.InputDoubleV("##"+id+"-in", &val, 0, 0, format, imgui.InputTextFlagsAutoSelectAll) {
		if val < min {
			val = min
		} else if val > max {
			val = max
		}
		if val != *v {
			*v = val
			changed = true
		}
	}
	ItemEdge(EdgeSunken)
	return changed
}

// SliderInputInt edits an int the same way.
func SliderInputInt(label, id string, v *int, min, max int) bool {
	f := float64(*v)
	fieldW := sc(64)
	changed := sliderFieldCore(label, id, &f, float64(min), float64(max), "%d", true, fieldW)
	if changed {
		*v = int(f)
	}
	imgui.SameLine()
	imgui.SetNextItemWidth(fieldW)
	val := int32(*v)
	if imgui.InputIntV("##"+id+"-in", &val, 0, 0, imgui.InputTextFlagsAutoSelectAll) {
		if val < int32(min) {
			val = int32(min)
		} else if val > int32(max) {
			val = int32(max)
		}
		if int(val) != *v {
			*v = int(val)
			changed = true
		}
	}
	ItemEdge(EdgeSunken)
	return changed
}

// sliderFieldCore is a trackbar without value readout (a typed field follows it).
func sliderFieldCore(label, id string, v *float64, min, max float64, format string, isInt bool, trailing float32) bool {
	labelColumn(label)
	avail := imgui.ContentRegionAvail().X
	trackW := avail - trailing - imgui.CurrentStyle().ItemSpacing().X - groupInset
	if trackW < sc(40) {
		trackW = sc(40)
	}
	frac := float32(0)
	if max > min {
		frac = float32((*v - min) / (max - min))
	}
	if frac < 0 {
		frac = 0
	} else if frac > 1 {
		frac = 1
	}
	nf, changed := trackbar(id, frac, imgui.NewVec2(trackW, imgui.FrameHeight()))
	if !changed {
		return false
	}
	nv := min + float64(nf)*(max-min)
	if isInt {
		nv = math.Round(nv)
	}
	if nv == *v {
		return false
	}
	*v = nv
	return true
}

// InputF64 edits a float64 in a numeric field with +/- step buttons.
func InputF64(label, id string, v *float64, step float64, format string) bool {
	labelColumn(label)
	changed := imgui.InputDoubleV("##"+id, v, step, step*10, format, imgui.InputTextFlagsNone)
	ItemEdge(EdgeSunken)
	return changed
}

// Check is a checkbox with a translated label.
func Check(label, id string, v *bool) bool {
	changed := imgui.Checkbox(label+"###chk-"+id, v)
	min := imgui.ItemRectMin()
	DrawEdge(imgui.WindowDrawList(), min, imgui.NewVec2(min.X+imgui.FrameHeight(), min.Y+imgui.FrameHeight()), EdgeSunken)
	return changed
}

// Combo selects one of items by index.
func Combo(label, id string, current *int, items []string) bool {
	labelColumn(label)
	preview := ""
	if *current >= 0 && *current < len(items) {
		preview = items[*current]
	}
	changed := false
	open := imgui.BeginCombo("##"+id, preview)
	min, max := imgui.ItemRectMin(), imgui.ItemRectMax()
	if open {
		for i, it := range items {
			if imgui.SelectableBoolV(it+"##"+strconv.Itoa(i), i == *current, 0, imgui.NewVec2(0, 0)) && i != *current {
				*current = i
				changed = true
			}
		}
		imgui.EndCombo()
	}
	DrawEdge(imgui.WindowDrawList(), min, max, EdgeSunken)
	return changed
}

// InputText is a label-left single-line text field with a sunken frame.
func InputText(label, id string, buf *string, hint string) bool {
	labelColumn(label)
	changed := imgui.InputTextWithHint("##"+id, hint, buf, imgui.InputTextFlagsNone, nil)
	ItemEdge(EdgeSunken)
	return changed
}

// WrapSameLine keeps the next button (with visible text nextLabel) on the current line when
// it fits, otherwise starts a new line, so button rows reflow for long translations.
func WrapSameLine(nextLabel string) {
	w := ButtonWidth(nextLabel) + imgui.CurrentStyle().ItemSpacing().X
	imgui.SameLine()
	if imgui.ContentRegionAvail().X < w {
		imgui.NewLine()
	}
}

// ButtonWidth is the shared push-button width: text plus padding, at least 75 px.
func ButtonWidth(label string) float32 {
	w := imgui.CalcTextSize(label).X + sc(24)
	if w < sc(75) {
		w = sc(75)
	}
	return w
}

// Button is a push button with a stable id.
func Button(label, id string) bool {
	size := imgui.NewVec2(ButtonWidth(label), imgui.FrameHeight()+2)
	return ButtonSized(label, id, size, true)
}

// ButtonSized is an owner-drawn Win32 push button. It never changes color on hover;
// the label is embossed when disabled and shifted by 1 px while pressed.
func ButtonSized(label, id string, size imgui.Vec2, enabled bool) bool {
	clicked := imgui.InvisibleButton("##btn-"+id, size)
	active := enabled && imgui.IsItemActive()
	min, max := imgui.ItemRectMin(), imgui.ItemRectMax()
	dl := imgui.WindowDrawList()
	p := CurrentPalette()
	FillRect(dl, min, max, p.Face)
	if active {
		DrawEdge(dl, min, max, EdgePressed)
	} else {
		DrawEdge(dl, min, max, EdgeRaised)
	}
	ts := imgui.CalcTextSize(label)
	pos := imgui.NewVec2(min.X+(max.X-min.X-ts.X)/2, min.Y+(max.Y-min.Y-ts.Y)/2)
	if active {
		pos = imgui.NewVec2(pos.X+1, pos.Y+1)
	}
	if !enabled {
		dl.AddTextVec2(imgui.NewVec2(pos.X+1, pos.Y+1), p.Highlight, label)
		dl.AddTextVec2(pos, p.Shadow, label)
	} else {
		dl.AddTextVec2(pos, TextColor(), label)
	}
	if enabled && imgui.IsItemFocused() {
		DottedRect(dl, imgui.NewVec2(min.X+4, min.Y+4), imgui.NewVec2(max.X-4, max.Y-4), TextColor())
	}
	return enabled && clicked
}

// ToolButton is a latching tool button: drawn pressed while active.
func ToolButton(label, id string, active bool) bool {
	size := imgui.NewVec2(ButtonWidth(label), imgui.FrameHeight()+2)
	clicked := imgui.InvisibleButton("##tool-"+id, size)
	min, max := imgui.ItemRectMin(), imgui.ItemRectMax()
	dl := imgui.WindowDrawList()
	p := CurrentPalette()
	if active {
		FillRect(dl, min, max, p.Light)
		DrawEdge(dl, min, max, EdgePressed)
	} else {
		FillRect(dl, min, max, p.Face)
		DrawEdge(dl, min, max, EdgeRaised)
	}
	ts := imgui.CalcTextSize(label)
	pos := imgui.NewVec2(min.X+(max.X-min.X-ts.X)/2, min.Y+(max.Y-min.Y-ts.Y)/2)
	if active {
		pos = imgui.NewVec2(pos.X+1, pos.Y+1)
	}
	dl.AddTextVec2(pos, TextColor(), label)
	return clicked
}

// GroupBox draws an etched frame with the title on the top border and runs body inside.
func GroupBox(title, id string, body func()) {
	avail := imgui.ContentRegionAvail().X
	start := imgui.CursorScreenPos()
	textH := imgui.TextLineHeight()
	// The 8 px inset is applied before BeginGroup so every line of the body (not only the
	// first) starts at the padded X.
	imgui.SetCursorScreenPos(imgui.NewVec2(start.X+8, start.Y+textH/2+8))
	imgui.PushIDStr("gb-" + id)
	imgui.BeginGroup()
	prev := groupInset
	groupInset = 10
	body()
	groupInset = prev
	imgui.EndGroup()
	maxY := imgui.ItemRectMax().Y + 8
	dl := imgui.WindowDrawList()
	p := CurrentPalette()
	top := start.Y + textH/2
	DrawEdge(dl, imgui.NewVec2(start.X, top), imgui.NewVec2(start.X+avail, maxY), EdgeEtched)
	ts := imgui.CalcTextSize(title)
	tx := start.X + 10
	FillRect(dl, imgui.NewVec2(tx-2, top-1), imgui.NewVec2(tx+ts.X+4, top+1), p.Face)
	dl.AddTextVec2(imgui.NewVec2(tx, start.Y), TextColor(), title)
	imgui.SetCursorScreenPos(imgui.NewVec2(start.X, maxY+6))
	imgui.Dummy(imgui.NewVec2(avail, 0))
	// Dummy resets X to the window indent; stay in the column the group box started in.
	imgui.SetCursorScreenPos(imgui.NewVec2(start.X, imgui.CursorScreenPos().Y))
	imgui.PopID()
}

// ToolbarSeparator draws a 2px etched vertical line at the current cursor.
func ToolbarSeparator() {
	imgui.SameLine()
	p := imgui.CursorScreenPos()
	h := imgui.FrameHeight()
	imgui.Dummy(imgui.NewVec2(6, h))
	dl := imgui.WindowDrawList()
	pal := CurrentPalette()
	x := p.X + 2
	strip(dl, x, p.Y, x+1, p.Y+h, pal.Shadow)
	strip(dl, x+1, p.Y, x+2, p.Y+h, pal.Highlight)
}

// TabStrip draws property-sheet tabs and returns the clicked index, or -1.
func TabStrip(id string, labels []string, active int) int {
	h := imgui.FrameHeight() + 4
	start := imgui.CursorScreenPos()
	clicked := -1
	dl := imgui.WindowDrawList()
	p := CurrentPalette()
	x := start.X
	for i, label := range labels {
		w := imgui.CalcTextSize(label).X + sc(24)
		imgui.SetCursorScreenPos(imgui.NewVec2(x, start.Y))
		if imgui.InvisibleButton("##"+id+strconv.Itoa(i), imgui.NewVec2(w, h)) {
			clicked = i
		}
		min, max := imgui.ItemRectMin(), imgui.ItemRectMax()
		if i == active {
			// Active tab merges with the page: no bottom edge, extends 2px sideways
			// and 2px down over the page border.
			FillRect(dl, imgui.NewVec2(min.X-2, min.Y-2), imgui.NewVec2(max.X+2, max.Y+2), p.Face)
			DrawEdge(dl, imgui.NewVec2(min.X-2, min.Y-2), imgui.NewVec2(max.X+2, max.Y+4), EdgeRaised)
			FillRect(dl, imgui.NewVec2(min.X, max.Y-1), imgui.NewVec2(max.X, max.Y+4), p.Face)
			TextCenteredIn(dl, imgui.NewVec2(min.X, min.Y-1), imgui.NewVec2(max.X, max.Y-1), label, TextColor())
		} else {
			top := imgui.NewVec2(min.X, min.Y+2)
			FillRect(dl, top, max, p.Face)
			DrawEdge(dl, top, imgui.NewVec2(max.X, max.Y+2), EdgeRaised)
			FillRect(dl, imgui.NewVec2(top.X+2, max.Y-1), imgui.NewVec2(max.X-2, max.Y+2), p.Face)
			TextCenteredIn(dl, top, max, label, TextColor())
		}
		x += w + 2
	}
	imgui.SetCursorScreenPos(imgui.NewVec2(start.X, start.Y+h+2))
	imgui.Dummy(imgui.NewVec2(0, 0))
	return clicked
}

func wrappedColored(text string, col imgui.Vec4) {
	imgui.PushStyleColorVec4(imgui.ColText, col)
	imgui.PushTextWrapPos()
	imgui.TextUnformatted(text)
	imgui.PopTextWrapPos()
	imgui.PopStyleColor()
}

// Note is dim wrapped helper text.
func Note(text string) { wrappedColored(text, CurrentPalette().TextDim) }

// Warn is emphasized wrapped warning text.
func Warn(text string) { wrappedColored(text, CurrentPalette().Warn) }

// Badge draws an OK / MANUAL / NG indicator lamp.
func Badge(level core.VerdictLevel) {
	text, bg, fg := "OK", rgba(0, 192, 0, 255), rgba(0, 0, 0, 255)
	switch level {
	case core.VerdictManual:
		text, bg, fg = "MANUAL", rgba(255, 216, 0, 255), rgba(0, 0, 0, 255)
	case core.VerdictNG:
		text, bg, fg = "NG", rgba(255, 0, 0, 255), rgba(255, 255, 255, 255)
	}
	ts := imgui.CalcTextSize(text)
	p := imgui.CursorScreenPos()
	size := imgui.NewVec2(ts.X+16, ts.Y+6)
	dl := imgui.WindowDrawList()
	dl.AddRectFilled(p, imgui.NewVec2(p.X+size.X, p.Y+size.Y), bg)
	DrawEdge(dl, p, imgui.NewVec2(p.X+size.X, p.Y+size.Y), EdgeSunken)
	dl.AddTextVec2(imgui.NewVec2(p.X+8, p.Y+3), fg, text)
	imgui.Dummy(size)
}

// KVTable shows label/value pairs with right-aligned values.
func KVTable(id string, rows [][2]string) {
	if !imgui.BeginTableV("##kv-"+id, 2, imgui.TableFlagsSizingStretchProp, imgui.NewVec2(0, 0), 0) {
		return
	}
	maxLabel := float32(0)
	for _, r := range rows {
		if w := imgui.CalcTextSize(r[0]).X; w > maxLabel {
			maxLabel = w
		}
	}
	imgui.TableSetupColumnV("", imgui.TableColumnFlagsWidthFixed, maxLabel+sc(12), 0)
	imgui.TableSetupColumnV("", imgui.TableColumnFlagsWidthStretch, 1, 0)
	dim := CurrentPalette().TextDim
	for _, r := range rows {
		imgui.TableNextRow()
		imgui.TableSetColumnIndex(0)
		imgui.PushStyleColorVec4(imgui.ColText, dim)
		imgui.TextUnformatted(r[0])
		imgui.PopStyleColor()
		imgui.TableSetColumnIndex(1)
		w := imgui.CalcTextSize(r[1]).X
		if gap := imgui.ContentRegionAvail().X - w; gap > 0 {
			imgui.SetCursorPosX(imgui.CursorPosX() + gap)
		}
		imgui.TextUnformatted(r[1])
	}
	imgui.EndTable()
}

// DataTable renders a scrollable grid with clipping (only visible rows are built), so
// tens of thousands of rows stay cheap. Column 0 is selectable; returns the clicked row or -1.
func DataTable(id string, headers []string, rows int, height float32, cell func(row, col int) string, selected int) int {
	return DataTableColored(id, headers, rows, height, func(row, col int) (string, *imgui.Vec4) {
		return cell(row, col), nil
	}, selected)
}

// tableWidthCache remembers the natural width of a table's content so columns can stretch
// across the available width when everything fits (and scroll when it does not).
var tableWidthCache = map[string]struct {
	w  float32
	at time.Time
}{}

// naturalTableWidth measures the headers and up to 120 rows (re-measured twice a second).
func naturalTableWidth(key string, headers []string, rows int, cell func(row, col int) (string, *imgui.Vec4)) float32 {
	if c, ok := tableWidthCache[key]; ok && time.Since(c.at) < 500*time.Millisecond {
		return c.w
	}
	pad := imgui.CurrentStyle().CellPadding().X*2 + sc(8)
	total := float32(0)
	for c, h := range headers {
		w := imgui.CalcTextSize(h).X
		n := rows
		if n > 120 {
			n = 120
		}
		for r := 0; r < n; r++ {
			t, _ := cell(r, c)
			if tw := imgui.CalcTextSize(t).X; tw > w {
				w = tw
			}
		}
		total += w + pad
	}
	tableWidthCache[key] = struct {
		w  float32
		at time.Time
	}{total, time.Now()}
	return total
}

// DataTableColored is DataTable whose cells may carry a text colour (nil keeps the default).
// The first column is the selectable row label and is never coloured. Classic look: raised
// column headers, navy selection with white text, columns that stretch across the width when
// the content fits and scroll (without cutting a column off) when it does not.
func DataTableColored(id string, headers []string, rows int, height float32, cell func(row, col int) (string, *imgui.Vec4), selected int) int {
	clicked := -1
	key := id + "-" + string(CurrentLang())
	need := naturalTableWidth(key, headers, rows, cell)
	fits := need+imgui.CurrentStyle().ScrollbarSize() <= imgui.ContentRegionAvail().X
	flags := imgui.TableFlagsRowBg | imgui.TableFlagsBorders | imgui.TableFlagsScrollY | imgui.TableFlagsResizable
	if fits {
		flags |= imgui.TableFlagsSizingStretchProp
	} else {
		flags |= imgui.TableFlagsScrollX | imgui.TableFlagsSizingFixedFit
	}
	// Language and mode in the id: fixed-fit widths are measured per script.
	mode := "f"
	if fits {
		mode = "s"
	}
	if !imgui.BeginTableV("##dt-"+key+mode, int32(len(headers)), flags, imgui.NewVec2(0, height), 0) {
		return clicked
	}
	imgui.TableSetupScrollFreeze(1, 1)
	for _, h := range headers {
		imgui.TableSetupColumnV(h, 0, 0, 0)
	}
	// Raised bevelled header cells, like a Win32 list-view header.
	imgui.TableNextRowV(imgui.TableRowFlagsHeaders, 0)
	for c, h := range headers {
		imgui.TableSetColumnIndex(int32(c))
		imgui.TableHeader(h)
		ItemEdge(EdgeRaised)
	}
	p := CurrentPalette()
	sel := PaletteVec4(p.Select)
	selText := PaletteVec4(p.SelectText)
	clipper := imgui.NewListClipper()
	clipper.Begin(int32(rows))
	for clipper.Step() {
		for r := int(clipper.DisplayStart()); r < int(clipper.DisplayEnd()); r++ {
			imgui.TableNextRow()
			isSel := r == selected
			if isSel {
				imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, p.Select)
				imgui.TableSetBgColor(imgui.TableBgTargetRowBg1, p.Select)
				imgui.PushStyleColorVec4(imgui.ColHeader, sel)
				imgui.PushStyleColorVec4(imgui.ColHeaderHovered, sel)
				imgui.PushStyleColorVec4(imgui.ColHeaderActive, sel)
				imgui.PushStyleColorVec4(imgui.ColText, selText)
			}
			for c := range headers {
				imgui.TableSetColumnIndex(int32(c))
				text, col := cell(r, c)
				if c == 0 {
					if imgui.SelectableBoolV(text+"##row"+strconv.Itoa(r), isSel, imgui.SelectableFlagsSpanAllColumns, imgui.NewVec2(0, 0)) {
						clicked = r
					}
					continue
				}
				if col != nil && !isSel {
					imgui.PushStyleColorVec4(imgui.ColText, *col)
				}
				imgui.TextUnformatted(text)
				if col != nil && !isSel {
					imgui.PopStyleColor()
				}
			}
			if isSel {
				imgui.PopStyleColorV(4)
			}
		}
	}
	clipper.End()
	clipper.Destroy()
	imgui.EndTable()
	return clicked
}

// Histogram draws a 256-bin log-scale GV histogram (classic scope look), optionally
// highlighting one bin (pass -1 for none).
func Histogram(id string, hist *[256]int, highlight int, height float32) {
	pal := CurrentPalette()
	w := imgui.ContentRegionAvail().X
	p := imgui.CursorScreenPos()
	dl := imgui.WindowDrawList()
	end := imgui.NewVec2(p.X+w, p.Y+height)
	dl.AddRectFilled(p, end, pal.ChartBg)
	for _, frac := range []float32{0.25, 0.5, 0.75} {
		y := p.Y + height*frac
		dl.AddLineArgs(imgui.NewVec2(p.X, y), imgui.NewVec2(end.X, y), pal.ChartGrid, 1)
	}
	max := 0
	for _, v := range hist {
		if v > max {
			max = v
		}
	}
	if max > 0 {
		logMax := math.Log(float64(max) + 1)
		barW := w / 256
		for i, v := range hist {
			if v == 0 {
				continue
			}
			bh := float32(math.Log(float64(v)+1)/logMax) * (height - 2)
			col := pal.ChartBar
			if i == highlight {
				col = pal.ChartMark
			}
			x0 := p.X + float32(i)*barW
			bw := barW
			if bw < 1 {
				bw = 1
			}
			dl.AddRectFilled(imgui.NewVec2(x0, end.Y-bh), imgui.NewVec2(x0+bw, end.Y), col)
		}
	}
	DrawEdge(dl, imgui.NewVec2(p.X-2, p.Y-2), imgui.NewVec2(end.X+2, end.Y+2), EdgeSunken)
	imgui.InvisibleButton("##hist-"+id, imgui.NewVec2(w, height))
}

// Heatmap draws an NCC surface (blue → yellow color ramp, same as the web app) in a
// square of the given size with the peak circled.
func Heatmap(id string, s *core.NccSurface, size float32) {
	p := imgui.CursorScreenPos()
	dl := imgui.WindowDrawList()
	end := imgui.NewVec2(p.X+size, p.Y+size)
	dl.AddRectFilled(p, end, CurrentPalette().ChartBg)
	if s != nil && s.Width > 0 && s.Height > 0 {
		cw := size / float32(s.Width)
		ch := size / float32(s.Height)
		peak, peakIdx := float32(-math.MaxFloat32), 0
		for i, v := range s.Values {
			if v > peak {
				peak, peakIdx = v, i
			}
		}
		for y := 0; y < s.Height; y++ {
			for x := 0; x < s.Width; x++ {
				t := (s.Values[y*s.Width+x] + 1) / 2
				if t < 0 {
					t = 0
				} else if t > 1 {
					t = 1
				}
				var r, g, b int
				if t < 0.5 {
					b = int(255 * t / 0.5)
				} else {
					k := (t - 0.5) / 0.5
					r, g, b = int(255*k), int(255*k), int(255*(1-k))
				}
				x0 := p.X + float32(x)*cw
				y0 := p.Y + float32(y)*ch
				dl.AddRectFilled(imgui.NewVec2(x0, y0), imgui.NewVec2(x0+cw+0.5, y0+ch+0.5), rgba(r, g, b, 255))
			}
		}
		px := p.X + float32(peakIdx%s.Width)*cw + cw/2
		py := p.Y + float32(peakIdx/s.Width)*ch + ch/2
		dl.AddCircleV(imgui.NewVec2(px, py), 5, OverlayOK, 0, 2)
	}
	DrawEdge(dl, imgui.NewVec2(p.X-2, p.Y-2), imgui.NewVec2(end.X+2, end.Y+2), EdgeSunken)
	imgui.InvisibleButton("##heat-"+id, imgui.NewVec2(size, size))
}
