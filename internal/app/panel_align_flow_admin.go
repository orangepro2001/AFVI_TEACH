package app

import (
	"fmt"
	"math"
	"strconv"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/ui"
)

// drawAdminLeft is the engineering column of the roi page.
func (p *AlignFlowPanel) drawAdminLeft() {
	Text(T("tab.roi"))
	ui.Separator()

	if Section(T("roi.section.view"), "roi-view") {
		sel := 0
		for i, v := range flowViews {
			if v == p.view {
				sel = i
			}
		}
		labels := make([]string, len(flowViews))
		for i, v := range flowViews {
			labels[i] = T("flow.view." + v.key())
		}
		if Combo(T("roi.section.view"), "roi-viewkind", &sel, labels) {
			if sel >= 0 && sel < len(flowViews) {
				p.view = flowViews[sel]
				p.refreshImage(true)
			}
		}
	}

	if Section(T("roi.section.rois"), "roi-rois") {
		rois := p.currentRois()
		if len(rois) == 0 {
			Note(T("roi.empty"))
		} else {
			for i, r := range rois {
				// A latching tool button stands in for the ImGui selectable row.
				label := fmt.Sprintf("%s  x%.0f y%.0f w%.0f h%.0f", r.Name, r.X, r.Y, r.W, r.H)
				if ToolButton(label, fmt.Sprintf("roi-sel-%d", i), i == p.editor.sel) {
					p.editor.sel = i
				}
			}
		}
		if Button(T("roi.restore"), "roi-restore") {
			p.resetBoxes()
		}
		WrapSameLine(T("roi.clear"))
		if Button(T("roi.clear"), "roi-clear") {
			if f := p.flow(); f != nil {
				f.SetRois(nil)
				p.editor.Reset()
				if dp := p.dp(); dp != nil {
					dp.RequestCapture()
				}
			}
		}
		WrapSameLine(T("roi.delete"))
		if Button(T("roi.delete"), "roi-delete") {
			p.deleteSelected()
		}
	}

	if Section(T("roi.section.params"), "roi-params") {
		flow := p.flow()
		if flow == nil {
			Note(T("roi.none"))
		} else {
			scale := flow.Scale
			if SliderInputInt(T("roi.param.scale"), "roi-scale", &scale, 1, 4) {
				flow.SetScale(scale)
				if dp := p.dp(); dp != nil {
					dp.RequestCapture()
				}
			}
			cfg := flow.Config
			changed := false
			changed = SliderInputF64(T("roi.param.search"), "roi-search", &cfg.SearchRadius, 10, 400, "%.0f") || changed
			changed = SliderInputF64(T("roi.param.minNcc"), "roi-minncc", &cfg.MinNcc, 0, 1, "%.2f") || changed
			changed = SliderInputF64(T("roi.param.minUniq"), "roi-minuniq", &cfg.MinUniqueness, 0, 1, "%.2f") || changed
			changed = SliderInputF64(T("roi.param.minOverlap"), "roi-minoverlap", &cfg.MinOverlap, 0, 1, "%.2f") || changed
			if changed {
				flow.SetConfig(cfg)
				if dp := p.dp(); dp != nil {
					dp.RequestCapture()
				}
			}
		}
	}

	if Section(T("roi.section.taught"), "roi-taught") {
		dp := p.dp()
		st := alignOff
		if dp != nil {
			st = dp.AlignState()
		}
		Text(alignStateText(st))
		if Button(T("roi.confirm"), "roi-confirm") {
			p.confirm()
		}
		WrapSameLine(T("roi.cancel"))
		if Button(T("roi.cancel"), "roi-cancel") {
			if f := p.flow(); f != nil {
				f.Disarm()
				if d := p.dp(); d != nil {
					d.RequestCapture()
				}
			}
		}
		if p.msg == "needTwo" {
			Warn(T("opAlign.needTwo"))
		}
	}

	if p.mouseOverPicture() && ui.KeyPressed("Delete") {
		p.deleteSelected()
	}
}

// drawAdminRight is the result column of the roi page.
func (p *AlignFlowPanel) drawAdminRight() {
	dp := p.dp()
	if dp == nil {
		Note(T("roi.noReport"))
		return
	}
	out := dp.AlignOutcome()
	if out == nil {
		Note(T("roi.noReport"))
		return
	}
	flow := p.flow()
	scale := 1
	if flow != nil {
		scale = flow.Scale
	}

	Badge(out.Report.Level)

	rows := [][2]string{
		{T("roi.info.elapsed"), F(out.ElapsedMs, 1)},
	}
	if out.Report.Fit != nil {
		full := core.FitToFullRes(*out.Report.Fit, scale)
		rows = append(rows,
			[2]string{T("roi.info.fitTx"), F(full.Tx, 2)},
			[2]string{T("roi.info.fitTy"), F(full.Ty, 2)},
			[2]string{T("roi.info.rot"), F(full.ThetaDeg, 3)},
			[2]string{T("roi.info.scale"), F(full.Scale, 4)},
		)
	} else {
		rows = append(rows,
			[2]string{T("roi.info.fitTx"), T("roi.none")},
			[2]string{T("roi.info.fitTy"), T("roi.none")},
			[2]string{T("roi.info.rot"), T("roi.none")},
			[2]string{T("roi.info.scale"), T("roi.none")},
		)
	}
	rows = append(rows,
		[2]string{T("roi.info.meanNcc"), F(out.Report.MeanNcc, 3)},
		[2]string{T("roi.info.residual"), F(out.Report.ResidualRms*float64(scale), 3)},
	)
	KVTable("roi-info", rows)

	drawn := 0
	if flow != nil {
		drawn = len(flow.Rois)
	}
	if text := explainAlign(out, drawn); text != "" {
		Warn(text)
	}

	headers := []string{
		T("roi.col.name"), T("roi.col.ncc"), T("roi.col.uniq"), T("roi.col.located"), T("roi.col.error"),
	}
	matches := out.Report.Matches
	cell := func(row, col int) string {
		if row < 0 || row >= len(matches) {
			return ""
		}
		m := matches[row]
		switch col {
		case 0:
			return m.Name
		case 1:
			return F(m.Ncc, 2)
		case 2:
			return F(m.Uniqueness, 2)
		case 3:
			if m.Located {
				return T("roi.yes")
			}
			return T("roi.no")
		case 4:
			if m.Found == nil {
				return T("roi.none")
			}
			dx := m.Found.X - m.Expected.X
			dy := m.Found.Y - m.Expected.Y
			err := dx*dx + dy*dy
			if err < 0 {
				err = 0
			}
			return F(math.Sqrt(err)*float64(scale), 2)
		}
		return strconv.Itoa(col)
	}
	DataTable("roi-matches", headers, len(matches), 8, cell, -1)
}
