package app

import (
	"fmt"
	"strconv"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
)

// The two column drawings of the golden page. All board methods only queue work, so the
// page reads current values and submits requests; the board repaints the picture.

// DrawLeft draws the editing column of the golden page.
func (p *GoldenPanel) DrawLeft() {
	imgui.AlignTextToFramePadding()
	imgui.TextUnformatted(T("golden.title"))
	imgui.Separator()
	b := p.board()
	if b == nil {
		Note(T("golden.title"))
		return
	}
	Note(T("golden.intro"))

	p.goldenStatus(b)
	p.goldenSource(b)
	p.goldenRange(b)
	p.goldenAction(b)
	p.goldenArea()
	p.goldenMorph()
	p.goldenPadding(b)
	p.goldenDisplay(b)
	p.goldenFileRow(b)

	if b.Ready() {
		avail := imgui.ContentRegionAvail().X
		if avail < sc(120) {
			avail = sc(120)
		}
		if ButtonSized(T("golden.next"), "golden-next", imgui.NewVec2(avail, sc(44)), true) {
			for _, id := range []string{"op-inspect", "defect"} {
				if p.a.ShowPage(id) {
					break
				}
			}
		}
	}

	if p.a.Viewer.Hovered() && imgui.CurrentIO().KeyCtrl() && imgui.IsKeyPressedBool(imgui.KeyZ) {
		b.Undo()
	}
}

// goldenStatus shows the board state and its last message.
func (p *GoldenPanel) goldenStatus(b *goldenBoard) {
	if b.Busy() {
		Note(T("golden.status.busy"))
	} else if b.Confirmed() {
		Note(T("golden.status.saved"))
	} else {
		Warn(T("golden.status.unsaved"))
	}
	key, text, isErr := b.Message()
	if key != "" {
		shown := T(key, "dir", b.Dir(), "msg", text)
		if isErr {
			Warn(shown)
		} else {
			Note(shown)
		}
	}
}

// goldenSource selects the channel of the golden sample picture.
func (p *GoldenPanel) goldenSource(b *goldenBoard) {
	if !Section(T("golden.section.source"), "golden-source") {
		return
	}
	imgui.AlignTextToFramePadding()
	imgui.TextUnformatted(T("golden.channel"))
	cur := b.Channel()
	for i, ch := range []struct {
		label, id string
		ch        core.CameraChannel
	}{{"R", "golden-ch-r", core.ChRed}, {"G", "golden-ch-g", core.ChGreen}, {"B", "golden-ch-b", core.ChBlue}} {
		if i > 0 {
			WrapSameLine(ch.label)
		} else {
			imgui.SameLine()
		}
		if ToolButton(ch.label, ch.id, cur == ch.ch) {
			b.SetChannel(ch.ch)
		}
	}
}

// goldenRange edits the binary threshold of the Metal mask.
func (p *GoldenPanel) goldenRange(b *goldenBoard) {
	if !Section(T("golden.section.range"), "golden-range") {
		return
	}
	lo, hi, out := b.Range()
	changed := false
	if ToolButton(T("golden.range.in"), "golden-range-in", !out) {
		out, changed = false, true
	}
	WrapSameLine(T("golden.range.out"))
	if ToolButton(T("golden.range.out"), "golden-range-out", out) {
		out, changed = true, true
	}
	if SliderInputInt(T("golden.range.low"), "golden-lo", &lo, 0, 255) {
		if lo > hi {
			hi = lo
		}
		changed = true
	}
	if SliderInputInt(T("golden.range.high"), "golden-hi", &hi, 0, 255) {
		if hi < lo {
			lo = hi
		}
		changed = true
	}
	if changed {
		b.SetRange(lo, hi, out)
	}
}

// goldenAction selects the edit tool, the shape and the Add threshold.
func (p *GoldenPanel) goldenAction(b *goldenBoard) {
	if !Section(T("golden.section.action"), "golden-action") {
		return
	}
	for i, t := range []struct {
		label, id string
		tool      goldenTool
	}{
		{T("golden.tool.point"), "golden-tool-point", goldenToolPoint},
		{T("golden.tool.add"), "golden-tool-add", goldenToolAdd},
		{T("golden.tool.erase"), "golden-tool-erase", goldenToolErase},
	} {
		if i > 0 {
			WrapSameLine(t.label)
		}
		if ToolButton(t.label, t.id, p.tool == t.tool) {
			p.setTool(t.tool)
		}
	}
	if p.tool != goldenToolPoint {
		for i, s := range []struct {
			label, id string
			kind      FrameKind
		}{
			{T("golden.shape.rect"), "golden-shape-rect", FrameRect},
			{T("golden.shape.ellipse"), "golden-shape-ellipse", FrameEllipse},
			{T("golden.shape.polygon"), "golden-shape-pen", FramePen},
		} {
			if i > 0 {
				WrapSameLine(s.label)
			}
			if ToolButton(s.label, s.id, p.frame.Kind == s.kind) {
				p.setShape(s.kind)
			}
		}
	}
	if p.tool == goldenToolAdd {
		SliderInputInt(T("golden.add.minGV"), "golden-mingv", &p.minGV, 0, 255)
		Note(T("golden.add.minGV.hint"))
	}
	if p.frame.Kind == FramePen && p.tool != goldenToolPoint {
		Note(T("golden.polygon.hint"))
	}
}

// goldenArea edits the area threshold.
func (p *GoldenPanel) goldenArea() {
	if !Section(T("golden.section.area"), "golden-area") {
		return
	}
	SliderInputInt(T("golden.area.min"), "golden-area-min", &p.areaMin, 2, 5000)
	Check(T("golden.area.fill"), "golden-area-fill", &p.areaFill)
	if Button(T("golden.apply"), "golden-area-apply") {
		p.applyArea()
	}
}

// goldenMorph edits and applies the morphological operations.
func (p *GoldenPanel) goldenMorph() {
	if !Section(T("golden.section.morph"), "golden-morph") {
		return
	}
	for i, key := range []string{"open", "close", "erode", "dilate"} {
		imgui.TextUnformatted(T("golden.morph." + key))
		SliderInputInt(T("golden.morph.n"), "golden-morph-n-"+key, &p.morphN[i], 1, 64)
		if Button(T("golden.apply"), "golden-morph-apply-"+key) {
			p.applyMorph(i)
		}
	}
}

// goldenPadding edits the padding and the SR inset.
func (p *GoldenPanel) goldenPadding(b *goldenBoard) {
	if !Section(T("golden.section.padding"), "golden-padding") {
		return
	}
	pad := b.Padding()
	if SliderInputInt(T("golden.padding"), "golden-padding", &pad, 0, golden.MaxPad) {
		b.SetPadding(pad)
	}
	Note(T("golden.padding.hint"))
	inset := b.SRInset()
	if SliderInputInt(T("golden.srInset"), "golden-srinset", &inset, 0, golden.MaxPad) {
		b.SetSRInset(inset)
	}
	Note(T("golden.srInset.hint"))
}

// goldenDisplay edits the colour and the layers of the mask overlay.
func (p *GoldenPanel) goldenDisplay(b *goldenBoard) {
	if !Section(T("golden.section.display"), "golden-display") {
		return
	}
	v := b.View()
	changed := false
	imgui.AlignTextToFramePadding()
	imgui.TextUnformatted(T("golden.display.color"))
	for i, c := range goldenColors {
		if i > 0 {
			WrapSameLine(T("golden.color." + c.key))
		} else {
			imgui.SameLine()
		}
		if ToolButton(T("golden.color."+c.key), "golden-color-"+c.key, v.Color == c.rgb) {
			v.Color, changed = c.rgb, true
		}
	}
	if SliderInputInt(T("golden.display.opacity"), "golden-opacity", &v.Opacity, 0, 100) {
		changed = true
	}
	if Check(T("golden.display.mask"), "golden-show-mask", &v.ShowMask) {
		changed = true
	}
	if Check(T("golden.display.band"), "golden-show-band", &v.ShowBand) {
		changed = true
	}
	if Check(T("golden.display.sr"), "golden-show-sr", &v.ShowSR) {
		changed = true
	}
	if changed {
		b.SetView(v)
	}
	Note(T("golden.display.note"))
}

// goldenFileRow draws Undo / Auto / Save / Load.
func (p *GoldenPanel) goldenFileRow(b *goldenBoard) {
	undo := T("golden.undo")
	undoSize := imgui.NewVec2(ButtonWidth(undo), imgui.FrameHeight()+2)
	if ButtonSized(undo, "golden-undo", undoSize, b.CanUndo()) {
		b.Undo()
	}
	auto := T("golden.auto")
	WrapSameLine(auto)
	if Button(auto, "golden-auto") {
		b.Auto()
	}
	save := T("golden.save")
	WrapSameLine(save)
	if Button(save, "golden-save") {
		_ = b.Save()
	}
	load := T("golden.load")
	WrapSameLine(load)
	if Button(load, "golden-load") {
		_ = b.Load()
	}
}

// DrawRight draws the help and the statistics column of the golden page.
func (p *GoldenPanel) DrawRight() {
	Note(T("golden.help.1"))
	Note(T("golden.help.2"))
	Note(T("golden.help.3"))
	Note(T("golden.help.4"))
	imgui.Separator()
	b := p.board()
	if b == nil {
		return
	}
	var rows [][2]string
	if d := b.Derived(); d != nil {
		rows = append(rows,
			[2]string{T("golden.stat.metal"), strconv.Itoa(d.metalPx)},
			[2]string{T("golden.stat.band"), strconv.Itoa(d.bandPx)},
			[2]string{T("golden.stat.sr"), strconv.Itoa(d.srPx)},
		)
	}
	if g := b.Geometry(); g != nil && g.registered {
		full := core.FitToFullRes(g.fit, goldenRegScale)
		rows = append(rows, [2]string{T("golden.stat.shift"), fmt.Sprintf("%.1f, %.1f px  %.2f°", full.Tx, full.Ty, full.ThetaDeg)})
	}
	KVTable("golden-stats", rows)
	if g := b.Geometry(); g != nil && g.registered {
		Note(T("golden.stat.registered"))
	} else if g != nil {
		Note(T("golden.stat.notRegistered"))
	}
}
