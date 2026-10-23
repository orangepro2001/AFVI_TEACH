package app

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/ui"
)

// The align tab's views: the overlay shows the template in red and the warped input in cyan,
// so a misalignment is visible as colour fringes without running anything.
var alignViews = []string{"overlay", "template", "input", "diff", "aligned", "alignedOverlay", "alignedDiff"}

// isAlignedView reports whether a view kind shows the input after the algorithm's own alignment.
func isAlignedView(kind string) bool { return strings.HasPrefix(kind, "aligned") }

func alignViewIndex(kind string) int {
	for i, v := range alignViews {
		if v == kind {
			return i
		}
	}
	return 0
}

// packAlignView renders the texture of a view kind.
func packAlignView(kind string, template, input *core.GrayImage, al *alignedResult) []byte {
	switch kind {
	case "overlay":
		return PackOverlay(template, input)
	case "input":
		return PackGray(input)
	case "diff":
		return PackAbsDiff(template, input)
	case "aligned", "alignedOverlay", "alignedDiff":
		if al == nil {
			// No alignment result for this image: a flat grey texture, the panel says why.
			return PackFlat(template.Width, template.Height, 48)
		}
		switch kind {
		case "alignedOverlay":
			return PackOverlay(template, al.re.Image)
		case "alignedDiff":
			return PackAbsDiffMasked(template, al.re.Image, al.re.Valid)
		}
		return PackGray(al.re.Image)
	}
	return PackGray(template)
}

// alignedResult is the input resampled onto the template grid with the transform the algorithm
// found (never the true warp), and how much closer to the template that brought it.
type alignedResult struct {
	input         *core.GrayImage // the input image it was made from
	reference     *core.GrayImage // the picture the grey difference and the check windows were measured against
	fit           core.Similarity
	re            *core.Realigned
	before, after float64 // mean |reference - image| over the pixels that have data, before and after aligning
	used          int     // number of pixels that have data (the occlusions are left out)
	pyramid       int     // search pyramid factor of the check
	check         core.ShiftCheck
}

// makeAligned realigns the input with the fit; the occlusions (working-resolution input
// coordinates) are left out of the comparison because they hide the image on purpose. reference
// is the real un-warped picture the input was warped from (the template itself with the "image"
// template): the grey difference and the check windows measure against it, never against the
// shape-only design picture.
func makeAligned(reference, input *core.GrayImage, fit core.Similarity, occs []core.Occlusion, pyramid int) *alignedResult {
	rects := make([]core.Rect, len(occs))
	for i, o := range occs {
		rects[i] = o.Rect
	}
	re := core.RealignInput(input, fit, rects)
	before, used := core.MeanAbsDiff(reference, input, re.Valid)
	after, _ := core.MeanAbsDiff(reference, re.Image, re.Valid)
	return &alignedResult{
		input: input, reference: reference, fit: fit, re: re, before: before, after: after, used: used,
		pyramid: pyramid, check: core.CheckRealigned(reference, re, pyramid),
	}
}

// alignedFor returns the realigned image for input and fit, reusing cached when it was made from
// the same input and fit; nil when there is no fit. reference is the real un-warped picture the
// input was warped from (the template itself with the "image" template).
func alignedFor(cached *alignedResult, reference, input *core.GrayImage, fit *core.Similarity, occs []core.Occlusion, pyramid int) *alignedResult {
	if fit == nil {
		return nil
	}
	if cached != nil && cached.input == input && cached.reference == reference && cached.fit == *fit && cached.pyramid == pyramid {
		return cached
	}
	return makeAligned(reference, input, *fit, occs, pyramid)
}

// alignFit is the transform the last run found for the image that is shown now; nil when there is
// none (no run yet, no fit, or the input was generated again after the run).
func (p *AlignPanel) alignFit() *core.Similarity {
	if p.report == nil || p.report.Fit == nil || p.fitInput == nil || p.fitInput != p.input {
		return nil
	}
	f := *p.report.Fit
	return &f
}

// alignReference is the real un-warped picture the whole-picture checks measure against: the
// reference when there is one, else the template (the "image" template is its own reference).
func (p *AlignPanel) alignReference() *core.GrayImage {
	if p.reference != nil {
		return p.reference
	}
	return p.template
}

// alignedNow is the realigned image that belongs to the image and the fit shown now, or nil.
func (p *AlignPanel) alignedNow() *alignedResult {
	fit := p.alignFit()
	if fit == nil || p.aligned == nil || p.aligned.input != p.input || p.aligned.fit != *fit ||
		p.aligned.reference != p.alignReference() {
		return nil
	}
	return p.aligned
}

// levelColor is the text tone of a verdict level (good / warn / bad).
func levelColor(l core.VerdictLevel) string {
	switch l {
	case core.VerdictOK:
		return "good"
	case core.VerdictManual:
		return "warn"
	}
	return "bad"
}

// levelOverlayColor is the overlay colour of a verdict level.
func levelOverlayColor(l core.VerdictLevel) uint32 {
	switch l {
	case core.VerdictOK:
		return OverlayOK
	case core.VerdictManual:
		return OverlayWarn
	}
	return OverlayNG
}

// errLevel classifies a position error (in the working resolution, like the thresholds in cfg).
func errLevel(errPx float64, cfg core.AlignConfig) core.VerdictLevel {
	return core.TruthLevel(errPx, cfg)
}

// levelText is the translated short name of a verdict level.
func levelText(l core.VerdictLevel) string { return T("align.level." + string(l)) }

// VerdictBanner is the big full-width verdict lamp: OK green, MANUAL yellow, NG red.
func VerdictBanner(level core.VerdictLevel, title string) {
	lv := "ok"
	switch level {
	case core.VerdictManual:
		lv = "manual"
	case core.VerdictNG:
		lv = "ng"
	}
	ui.AlignBanner(title, lv)
}

// tableCell is one cell of ResultTable; an empty tone keeps the default text colour.
type tableCell struct {
	text string
	tone string
}

// ResultTable shows a small read-only grid with optionally coloured cells.
func ResultTable(id string, headers []string, rows [][]tableCell) {
	DataTableColored(id+"-"+string(CurrentLang()), headers, len(rows), len(rows)+1, func(r, c int) ui.Cell {
		cell := rows[r][c]
		return ui.Cell{T: cell.text, Tone: cell.tone}
	}, -1)
}

// wrappedColored is wrapped text in a tone colour (good / warn / bad / dim).
func wrappedColored(text, tone string) { ui.AlignNoteTone(text, tone) }

// fullPx converts a working-resolution distance to full-resolution pixels, the unit of every
// number the align tab shows (the thresholds are converted the same way).
func (p *AlignPanel) fullPx(v float64) float64 { return v * float64(p.lastScale) }

// truthChecked reports whether the last report has a usable truth comparison.
func (p *AlignPanel) truthChecked() bool {
	return p.report != nil && p.truth != nil && !math.IsNaN(p.truth.MaxErrorPx)
}

// drawVerdict draws the verdict block of the right panel: the lamp, the reasons, the check of
// the verdict against the ground truth, the set/measured/difference table and the key numbers.
func (p *AlignPanel) drawVerdict() {
	if p.report == nil {
		Note(T("align.result.empty"))
		return
	}
	r := p.report
	cfg := p.reportConfig
	VerdictBanner(r.Level, T("align.banner."+string(r.Level)))
	if p.reportStale() {
		if p.a.Jobs.Busy("align") {
			Warn(T("align.result.running"))
		} else {
			Warn(T("align.result.stale"))
		}
	}
	for _, reason := range r.Reasons {
		Warn(p.reasonText(reason))
	}

	ui.Spacing()
	Note(T("align.truth.title"))
	if p.truthChecked() {
		maxErr := p.truth.MaxErrorPx
		truthLevel := core.TruthLevel(maxErr, cfg)
		judge := core.JudgeAgainstTruth(r.Level, truthLevel)
		args := []any{
			"err", F(p.fullPx(maxErr), 2),
			"ok", F(p.fullPx(cfg.OkResidualPx), 2),
			"ng", F(p.fullPx(cfg.NgResidualPx), 2),
			"verdict", levelText(r.Level),
			"truth", levelText(truthLevel),
		}
		col := levelColor(core.VerdictOK)
		switch judge {
		case core.JudgeMissed, core.JudgeFalseAlarm:
			col = levelColor(core.VerdictNG)
		case core.JudgeStrict, core.JudgeCautious:
			col = levelColor(core.VerdictManual)
		}
		wrappedColored(T("align.truth."+string(judge), args...), col)
	} else {
		Note(T("align.truth.none"))
	}

	ui.Spacing()
	if r.Fit != nil && p.reportW > 0 {
		sc := float64(p.lastScale)
		truthPose := core.TruthPose(&p.reportParams, p.reportW, p.reportH)
		fitPose := core.FitPose(*r.Fit, p.reportW, p.reportH)
		shiftCell := func(set, got float64) []tableCell {
			d := got - set
			col := levelColor(errLevel(math.Abs(d), cfg))
			return []tableCell{{text: F(set*sc, 2)}, {text: F(got*sc, 2)}, {text: F(d*sc, 2), tone: col}}
		}
		rows := [][]tableCell{
			append([]tableCell{{text: T("align.pose.shiftX")}}, shiftCell(truthPose.ShiftX, fitPose.ShiftX)...),
			append([]tableCell{{text: T("align.pose.shiftY")}}, shiftCell(truthPose.ShiftY, fitPose.ShiftY)...),
			{{text: T("align.pose.theta")}, {text: F(truthPose.ThetaDeg, 3)}, {text: F(fitPose.ThetaDeg, 3)}, {text: F(fitPose.ThetaDeg-truthPose.ThetaDeg, 3)}},
			{{text: T("align.pose.scale")}, {text: F(truthPose.Scale, 5)}, {text: F(fitPose.Scale, 5)}, {text: F(fitPose.Scale-truthPose.Scale, 5)}},
		}
		ResultTable("pose", []string{T("align.pose.item"), T("align.pose.set"), T("align.pose.measured"), T("align.pose.delta")}, rows)
		if p.reportParams.ShearX != 0 || p.reportParams.ShearY != 0 || p.reportParams.TiltX != 0 || p.reportParams.TiltY != 0 ||
			(p.reportParams.BendAmplitude != 0 && p.reportParams.BendWavelength > 0) {
			Note(T("align.result.modelNote"))
		}
	} else {
		Warn(T("align.result.fitFailed"))
	}

	rmsCol := levelColor(errLevel(r.ResidualRms, cfg))
	Text(T("align.result.residualRms") + ": ")
	SameLine()
	ui.TextTone(F(p.fullPx(r.ResidualRms), 3)+" px", rmsCol)
	Note(T("align.result.thresholds", "ok", F(p.fullPx(cfg.OkResidualPx), 2), "ng", F(p.fullPx(cfg.NgResidualPx), 2)))
	rows := [][2]string{
		{T("align.result.meanNcc"), F(r.MeanNcc, 4)},
		{T("align.result.elapsed"), F(p.elapsedMs, 1) + " ms"},
	}
	al := p.alignedNow()
	if al != nil && al.used > 0 && !math.IsNaN(al.before) {
		rows = append(rows, [2]string{T("align.gain.label"), F(al.before, 2) + " → " + F(al.after, 2)})
	}
	KVTable("align-kv", rows)
	if al == nil {
		return
	}
	if c := al.check; c.Windows > 0 {
		col := levelColor(errLevel(c.MaxPx, cfg))
		Text(T("align.check.label") + ": ")
		SameLine()
		ui.TextTone(F(p.fullPx(c.MaxPx), 2)+" / "+F(p.fullPx(c.RmsPx), 2)+" px", col)
		Note(T("align.check.note", "n", strconv.Itoa(c.Windows), "total", strconv.Itoa(c.Total),
			"ok", F(p.fullPx(cfg.OkResidualPx), 2)))
	} else {
		Note(T("align.check.none"))
	}
	Note(T("align.gain.note"))
}

// drawRoiResults draws the per-ROI table: correlation, residual of the fit, error against the
// ground truth (both coloured by the OK/NG thresholds) and the status.
func (p *AlignPanel) drawRoiResults() {
	if p.report == nil {
		Note(T("align.result.empty"))
		return
	}
	a := p.a
	r := p.report
	cfg := p.reportConfig
	headers := []string{
		T("align.col.roi"), T("align.col.ncc"), T("align.col.uniqueness"),
		T("align.col.residual"), T("align.col.match"), T("align.col.truth"), T("align.col.status"),
	}
	matches := r.Matches
	cell := func(row, col int) ui.Cell {
		m := matches[row]
		switch col {
		case 0:
			return ui.Cell{T: m.Name}
		case 1:
			return ui.Cell{T: F(m.Ncc, 4)}
		case 2:
			return ui.Cell{T: F(m.Uniqueness, 4)}
		case 3:
			if row < len(r.Residuals) && r.Residuals[row] >= 0 {
				return ui.Cell{T: F(p.fullPx(r.Residuals[row]), 3), Tone: levelColor(errLevel(r.Residuals[row], cfg))}
			}
			return ui.Cell{T: "—"}
		case 4:
			if row < len(p.diag) && !math.IsNaN(p.diag[row].MatchErrPx) {
				e := p.diag[row].MatchErrPx
				return ui.Cell{T: F(p.fullPx(e), 3), Tone: levelColor(errLevel(e, cfg))}
			}
			return ui.Cell{T: "—"}
		case 5:
			if p.truth != nil && row < len(p.truth.PerRoiErrorPx) && !math.IsNaN(p.truth.PerRoiErrorPx[row]) {
				e := p.truth.PerRoiErrorPx[row]
				return ui.Cell{T: F(p.fullPx(e), 3), Tone: levelColor(errLevel(e, cfg))}
			}
			return ui.Cell{T: "—"}
		}
		if !m.Located {
			return ui.Cell{T: p.matchReasonText(m.Reason), Tone: levelColor(core.VerdictNG)}
		}
		if row < len(r.Residuals) && r.Residuals[row] >= 0 {
			return ui.Cell{T: T("align.status.inlier"), Tone: levelColor(core.VerdictOK)}
		}
		return ui.Cell{T: T("align.status.outlier"), Tone: levelColor(core.VerdictManual)}
	}
	clicked := DataTableColored("align-matches", headers, len(matches), 8, cell, p.selected)
	if clicked >= 0 && clicked < len(matches) {
		p.selected = clicked
		m := matches[clicked]
		p.heat = m.Surface
		if m.Found != nil {
			a.Viewer.CenterOn(m.Found.X, m.Found.Y)
		} else {
			a.Viewer.CenterOn(m.Expected.X, m.Expected.Y)
		}
	}
	Note(T("align.col.explain"))
	if notes := p.roiNotes(); len(notes) > 0 {
		ui.Spacing()
		Note(T("align.notes.title"))
		for _, n := range notes {
			switch n.level {
			case core.VerdictOK:
				Note(n.text)
			default:
				wrappedColored(n.text, levelColor(n.level))
			}
		}
	}
}

// roiNote is one line of the per-ROI diagnosis under the results table: NG for a ROI that was not
// located, MANUAL for an outlier, OK for a remark about a ROI that is fine.
type roiNote struct {
	level core.VerdictLevel
	text  string
}

// effectiveOverlap is the smallest share of a matching window that must lie inside the image
// (1 means all of it, the behaviour of MinOverlap 0).
func effectiveOverlap(cfg core.AlignConfig) float64 {
	if cfg.MinOverlap <= 0 || cfg.MinOverlap >= 1 {
		return 1
	}
	return cfg.MinOverlap
}

// pctText formats a share 0..1 as a percentage; a sliver below 1% or short of 100% keeps one
// decimal so that it never reads "0%" or "100%".
func pctText(v float64) string {
	p := v * 100
	switch {
	case p > 0 && p < 1:
		return F(p, 1)
	case p > 99 && p < 100:
		return F(math.Floor(p*10)/10, 1)
	}
	return F(p, 0)
}

// roiNotes explains, with the help of the ground truth, why a ROI was not located or was thrown out
// as an outlier (window outside the image, covered by an occlusion) and remarks on ROIs that only
// worked because the matcher accepted a partly outside window.
func (p *AlignPanel) roiNotes() []roiNote {
	r := p.report
	if r == nil || len(p.diag) != len(r.Matches) {
		return nil
	}
	cfg := p.reportConfig
	pct := pctText
	var notes []roiNote
	for i, m := range r.Matches {
		d := p.diag[i]
		inlier := i < len(r.Residuals) && r.Residuals[i] >= 0
		var causes []string
		if d.InsideShare < 0.9995 {
			eff := effectiveOverlap(cfg)
			px := F(p.fullPx(d.OverhangPx), 1)
			if d.InsideShare < eff {
				causes = append(causes, T("align.cause.tooFarOut", "in", pct(d.InsideShare), "min", pct(eff), "px", px))
			} else {
				causes = append(causes, T("align.cause.overhang", "px", px, "out", pct(1-d.InsideShare), "min", pct(eff)))
			}
		}
		if d.Occluded >= 0.05 {
			causes = append(causes, T("align.cause.occluded", "pct", pct(d.Occluded)))
		}
		note := roiNote{level: core.VerdictOK}
		head := ""
		switch {
		case !m.Located:
			note.level = core.VerdictNG
			head = T("align.note.missing", "name", m.Name, "why", p.matchReasonText(m.Reason), "ncc", F(m.Ncc, 3))
		case !inlier:
			note.level = core.VerdictManual
			head = T("align.note.outlier", "name", m.Name, "dev", F(p.fullPx(d.FitDevPx), 1), "th", F(p.fullPx(cfg.RansacThresholdPx), 1))
		default:
			// A sliver of overhang does no harm; only remark on windows that lost a noticeable share.
			if d.InsideShare >= 0.99 && d.Occluded < 0.05 {
				continue
			}
			head = m.Name + T("align.colon")
		}
		note.text = head
		if len(causes) > 0 {
			note.text += " " + strings.Join(causes, T("align.sep")) + T("align.end")
		}
		notes = append(notes, note)
	}
	return notes
}

// drawAlignOverlay draws, for every ROI, where the design says it is (grey dashes), where the
// warp really put it (cyan dashes, the ground truth) and where the algorithm found it (solid
// green or red), joined by the displacement vector and labelled with the error.
func (p *AlignPanel) drawAlignOverlay(o *Overlay) {
	if p.template == nil {
		return
	}
	s := float64(p.scale)
	w, h := p.template.Width, p.template.Height
	wp := p.scaledParams()
	cfg := p.config
	if p.report != nil {
		cfg = p.reportConfig
	}
	for i, r := range p.rois {
		rx, ry, rw, rh := r.X/s, r.Y/s, r.W/s, r.H/s
		th := float32(1.5)
		if i == p.selected {
			th = 3
		}
		o.DashedRect(rx, ry, rw, rh, OverlayMuted, th)
		o.Text(rx, ry, 2, -14, OverlayMuted, r.Name)
		centre := core.Point{X: rx + rw/2, Y: ry + rh/2}
		truth := core.WarpPoint(&wp, centre, w, h)
		o.DashedRect(truth.X-rw/2, truth.Y-rh/2, rw, rh, OverlayROI, th)
		o.Line(centre.X, centre.Y, truth.X, truth.Y, OverlayWarn, 1.5)
		o.Circle(truth.X, truth.Y, 2/float64(o.Scale()), OverlayROI, 2)
		if p.report == nil || i >= len(p.report.Matches) {
			continue
		}
		m := p.report.Matches[i]
		if m.Found != nil {
			// The match is the evidence the fit is built from: a dot, green when located.
			col := OverlayNG
			if m.Located {
				col = OverlayOK
			}
			o.Circle(m.Found.X, m.Found.Y, 3/float64(o.Scale()), col, 2)
		}
		if p.report.Fit == nil {
			continue
		}
		// The alignment result is what the machine will use: the fitted transform applied to the ROI.
		pred := core.ApplySimilarity(*p.report.Fit, centre)
		col := OverlayWarn
		label := m.Name
		if p.truth != nil && i < len(p.truth.PerRoiErrorPx) && !math.IsNaN(p.truth.PerRoiErrorPx[i]) {
			e := p.truth.PerRoiErrorPx[i]
			col = levelOverlayColor(errLevel(e, cfg))
			label = fmt.Sprintf("%s  Δ%s px", m.Name, F(p.fullPx(e), 2))
		}
		o.Rect(pred.X-rw/2, pred.Y-rh/2, rw, rh, col, 2)
		o.Line(truth.X, truth.Y, pred.X, pred.Y, col, 2)
		o.Text(pred.X-rw/2, pred.Y-rh/2, 0, -16, col, label)
	}
}

// drawAlignedOverlay marks, on the realigned image, where every ROI ought to be (grey dashes, the
// design), where the ground truth puts it once the input is brought back with the fit the
// algorithm found (cyan dashes, joined to the design position by a line coloured by the error) and
// where the ROI's own match lands after the same transform (dot). With a perfect alignment the
// three coincide; the distance between them is what is still left over.
func (p *AlignPanel) drawAlignedOverlay(o *Overlay) {
	if p.template == nil {
		return
	}
	s := float64(p.scale)
	fit := p.alignFit()
	var back core.Similarity
	if fit != nil {
		back = core.InvertSimilarity(*fit)
	}
	cfg := p.reportConfig
	for i, r := range p.rois {
		rx, ry, rw, rh := r.X/s, r.Y/s, r.W/s, r.H/s
		th := float32(1.5)
		if i == p.selected {
			th = 3
		}
		o.DashedRect(rx, ry, rw, rh, OverlayMuted, th)
		o.Text(rx, ry, 2, -14, OverlayMuted, r.Name)
		if fit == nil || i >= len(p.report.Matches) {
			continue
		}
		m := p.report.Matches[i]
		centre := core.Point{X: rx + rw/2, Y: ry + rh/2}
		truth := core.ApplySimilarity(back, core.WarpPoint(&p.reportParams, centre, p.reportW, p.reportH))
		errPx := math.Hypot(truth.X-centre.X, truth.Y-centre.Y)
		if p.truth != nil && i < len(p.truth.PerRoiErrorPx) && !math.IsNaN(p.truth.PerRoiErrorPx[i]) {
			errPx = p.truth.PerRoiErrorPx[i]
		}
		col := levelOverlayColor(errLevel(errPx, cfg))
		o.DashedRect(truth.X-rw/2, truth.Y-rh/2, rw, rh, OverlayROI, th)
		o.Line(centre.X, centre.Y, truth.X, truth.Y, col, 2)
		o.Text(rx, ry+rh, 2, 2, col, fmt.Sprintf("%s  Δ%s px", m.Name, F(p.fullPx(errPx), 2)))
		if m.Found != nil {
			f := core.ApplySimilarity(back, *m.Found)
			dot := OverlayNG
			if m.Located {
				dot = OverlayOK
			}
			o.Circle(f.X, f.Y, 3/float64(o.Scale()), dot, 2)
		}
	}
	if al := p.alignedNow(); al != nil && al.check.Windows > 0 {
		// The window of the local check that is still furthest off, to zoom in on.
		c := al.check
		col := levelOverlayColor(errLevel(c.MaxPx, cfg))
		half := float64(c.Win) / 2
		o.Rect(c.Worst.X-half, c.Worst.Y-half, float64(c.Win), float64(c.Win), col, 2)
		o.Text(c.Worst.X-half, c.Worst.Y-half, 2, -16, col, T("align.check.worst", "px", F(p.fullPx(c.MaxPx), 2)))
	}
}

// hoverAligned is the status-bar text of the aligned views.
func (p *AlignPanel) hoverAligned(x, y int) string {
	al := p.alignedNow()
	if al == nil || p.template == nil || x < 0 || y < 0 || x >= p.template.Width || y >= p.template.Height {
		return ""
	}
	i := y*p.template.Width + x
	if !al.re.Valid[i] {
		return T("align.view.noData")
	}
	t, a := int(p.template.Data[i]), int(al.re.Image.Data[i])
	switch p.viewKind {
	case "alignedOverlay":
		return fmt.Sprintf("T %d  A %d", t, a)
	case "alignedDiff":
		d := t - a
		if d < 0 {
			d = -d
		}
		return fmt.Sprintf("GV %d", d)
	}
	return fmt.Sprintf("GV %d", a)
}

// legendEntry is one colour swatch with its explanation in the canvas legend.
type legendEntry struct {
	col  uint32
	text string
}

// legendEntries are the colours the overlay of the current view uses. The canvas has no
// screen-space drawing, so the legend is listed in the right panel (drawLegend).
func (p *AlignPanel) legendEntries() []legendEntry {
	if p.template == nil || p.viewKind == "template" {
		return nil
	}
	if isAlignedView(p.viewKind) {
		entries := []legendEntry{{OverlayMuted, T("align.legend.design")}}
		if p.alignFit() != nil {
			entries = append(entries,
				legendEntry{OverlayROI, T("align.legend.alignedTruth")},
				legendEntry{OverlayNG, T("align.legend.match")},
				legendEntry{OverlayWarn, T("align.legend.rest")},
			)
		}
		return entries
	}
	return []legendEntry{
		{OverlayMuted, T("align.legend.design")},
		{OverlayROI, T("align.legend.truth")},
		{OverlayOK, T("align.legend.found")},
		{OverlayNG, T("align.legend.match")},
		{OverlayWarn, T("align.legend.shift")},
	}
}

// drawLegend lists the legend entries as colour chips with their explanation.
func drawLegend(entries []legendEntry) {
	for _, e := range entries {
		Swatch(e.col)
		SameLine()
		Text(e.text)
	}
}

func init() {
	Messages(map[string]string{
		"align.view.overlay":      "叠加（模板红 / 输入青）",
		"align.section.misalign":  "错位（位置）",
		"align.section.lift":      "翘起与形变",
		"align.section.photo":     "成像（光度）",
		"align.presets":           "预设",
		"align.banner.ok":         "OK  对齐通过",
		"align.banner.manual":     "MANUAL  需人工确认",
		"align.banner.ng":         "NG  对齐失败",
		"align.level.ok":          "OK",
		"align.level.manual":      "人工确认",
		"align.level.ng":          "NG",
		"align.result.stale":      "参数已改动，下方结果还是上一次运行的，请重新运行。",
		"align.truth.title":       "与真值核对",
		"align.truth.none":        "没有拟合结果，无法与真值比较。",
		"align.truth.correct":     "判定正确：真值最大误差 {err} px，应判 {truth}，算法判 {verdict}。",
		"align.truth.missed":      "漏判：真值最大误差 {err} px，已超过 OK 阈值 {ok} px，算法却判 {verdict}。",
		"align.truth.false-alarm": "误判：真值最大误差只有 {err} px（不超过 OK 阈值 {ok} px），算法却判 {verdict}。",
		"align.truth.strict":      "偏严：真值最大误差 {err} px 只需要人工确认，算法直接判 {verdict}。",
		"align.truth.cautious":    "保守：真值最大误差 {err} px 应判 {truth}，算法转人工确认。",
		"align.pose.item":         "项目",
		"align.pose.set":          "设定",
		"align.pose.measured":     "测得",
		"align.pose.delta":        "差",
		"align.pose.shiftX":       "中心位移 X (px)",
		"align.pose.shiftY":       "中心位移 Y (px)",
		"align.pose.theta":        "旋转 (°)",
		"align.pose.scale":        "缩放",
		"align.result.modelNote":  "含剪切、翘起或弯曲时，相似变换无法完全描述，真值误差里包含这部分模型误差。",
		"align.result.thresholds": "阈值（全分辨率像素）：OK ≤ {ok}，NG > {ng}",
		"align.section.roiResult": "各 ROI 结果",
		"align.col.truth":         "对齐误差",
		"align.col.match":         "匹配误差",
		"align.col.explain":       "匹配误差：该 ROI 自己匹配到的位置离真实位置多远；对齐误差：整体拟合把该 ROI 放到的位置离真实位置多远（机器实际使用的）。外点就是匹配误差大而对齐误差小的 ROI。",
		"align.notes.title":       "逐个 ROI 诊断",
		"align.note.missing":      "{name}：未定位（{why}，NCC {ncc}）。",
		"align.note.outlier":      "{name}：外点，匹配位置偏离整体拟合 {dev} px（阈值 {th} px）。",
		"align.cause.overhang":    "真实位置的窗口伸出图像 {px} px（占面积 {out}%），只用图内部分匹配（最小重叠 {min}%）",
		"align.cause.tooFarOut":   "真实位置的窗口只有 {in}% 在图内，低于最小重叠 {min}%，匹配不到真实位置，只能停在图像边缘（伸出 {px} px）",
		"align.cause.occluded":    "窗口有 {pct}% 被遮挡盖住",
		"align.colon":             "：",
		"align.sep":               "；",
		"align.end":               "。",
		"align.detail.outlier":    "{name}（偏离拟合 {dev} px，阈值 {th} px）",
		"align.detail.ambiguous":  "{name}（唯一性 {u}，下限 {min}）",
		"align.detail.missing":    "{name}（{why}）",
		"align.result.running":    "正在重新计算，下方还是上一次的结果。",
		"align.col.status":        "状态",
		"align.status.inlier":     "内点",
		"align.status.outlier":    "外点",
		"align.legend.design":     "设计位置",
		"align.legend.truth":      "真值位置（扭曲后）",
		"align.legend.found":      "对齐结果（拟合，颜色=真值误差）",
		"align.legend.match":      "匹配点（绿=定位 红=失败）",
		"align.legend.shift":      "实际位移",
	}, map[string]string{
		"align.view.overlay":      "Overlay (template red / input cyan)",
		"align.section.misalign":  "Misalignment (position)",
		"align.section.lift":      "Lift & deformation",
		"align.section.photo":     "Imaging (photometric)",
		"align.presets":           "Presets",
		"align.banner.ok":         "OK  aligned",
		"align.banner.manual":     "MANUAL  check by hand",
		"align.banner.ng":         "NG  alignment failed",
		"align.level.ok":          "OK",
		"align.level.manual":      "MANUAL",
		"align.level.ng":          "NG",
		"align.result.stale":      "Parameters changed; the results below are from the previous run. Run again.",
		"align.truth.title":       "Check against ground truth",
		"align.truth.none":        "No fit, so there is nothing to compare with the truth.",
		"align.truth.correct":     "Correct verdict: max truth error {err} px means {truth}; the algorithm said {verdict}.",
		"align.truth.missed":      "Missed: max truth error {err} px exceeds the OK threshold {ok} px, yet the algorithm said {verdict}.",
		"align.truth.false-alarm": "False alarm: max truth error is only {err} px (within the OK threshold {ok} px), yet the algorithm said {verdict}.",
		"align.truth.strict":      "Too strict: max truth error {err} px only needs a manual check, but the algorithm said {verdict}.",
		"align.truth.cautious":    "Cautious: max truth error {err} px means {truth}; the algorithm asked for a manual check.",
		"align.pose.item":         "Item",
		"align.pose.set":          "Set",
		"align.pose.measured":     "Measured",
		"align.pose.delta":        "Diff",
		"align.pose.shiftX":       "Centre shift X (px)",
		"align.pose.shiftY":       "Centre shift Y (px)",
		"align.pose.theta":        "Rotation (°)",
		"align.pose.scale":        "Scale",
		"align.result.modelNote":  "With shear, lift or bend a similarity transform cannot describe the warp completely; the truth error includes that model error.",
		"align.result.thresholds": "Thresholds (full-resolution pixels): OK ≤ {ok}, NG > {ng}",
		"align.section.roiResult": "Per-ROI results",
		"align.col.truth":         "Align error",
		"align.col.match":         "Match error",
		"align.col.explain":       "Match error: how far the position this ROI matched on its own is from the true position. Align error: how far the overall fit puts this ROI from the true position (what the machine uses). An outlier has a large match error but a small align error.",
		"align.notes.title":       "Per-ROI diagnosis",
		"align.note.missing":      "{name}: not located ({why}, NCC {ncc}).",
		"align.note.outlier":      "{name}: outlier, its match is {dev} px away from the overall fit (limit {th} px).",
		"align.cause.overhang":    "the true window sticks out of the image by {px} px ({out}% of its area); only the part inside is matched (minimum overlap {min}%)",
		"align.cause.tooFarOut":   "only {in}% of the true window is inside the image, below the minimum overlap {min}%, so the true position cannot be matched and the match stops at the image edge (sticks out {px} px)",
		"align.cause.occluded":    "{pct}% of the window is covered by an occlusion",
		"align.colon":             ": ",
		"align.sep":               "; ",
		"align.end":               ".",
		"align.detail.outlier":    "{name} ({dev} px off the fit, limit {th} px)",
		"align.detail.ambiguous":  "{name} (uniqueness {u}, minimum {min})",
		"align.detail.missing":    "{name} ({why})",
		"align.result.running":    "Recalculating; the results below are still the previous ones.",
		"align.col.status":        "Status",
		"align.status.inlier":     "inlier",
		"align.status.outlier":    "outlier",
		"align.legend.design":     "Design position",
		"align.legend.truth":      "True position (warped)",
		"align.legend.found":      "Alignment result (fit; colour = truth error)",
		"align.legend.match":      "Match point (green located, red failed)",
		"align.legend.shift":      "Actual shift",
	}, map[string]string{
		"align.view.overlay":      "오버레이 (템플릿 빨강 / 입력 청록)",
		"align.section.misalign":  "오정렬 (위치)",
		"align.section.lift":      "들뜸과 변형",
		"align.section.photo":     "촬상 (광학)",
		"align.presets":           "프리셋",
		"align.banner.ok":         "OK  정렬 통과",
		"align.banner.manual":     "MANUAL  수동 확인 필요",
		"align.banner.ng":         "NG  정렬 실패",
		"align.level.ok":          "OK",
		"align.level.manual":      "수동 확인",
		"align.level.ng":          "NG",
		"align.result.stale":      "파라미터가 바뀌었습니다. 아래 결과는 이전 실행의 것이니 다시 실행하세요.",
		"align.truth.title":       "정답과 비교",
		"align.truth.none":        "피팅 결과가 없어 정답과 비교할 수 없습니다.",
		"align.truth.correct":     "판정 정확: 정답 최대 오차 {err} px 는 {truth} 에 해당하며 알고리즘은 {verdict} 로 판정했습니다.",
		"align.truth.missed":      "미검출: 정답 최대 오차 {err} px 가 OK 임계값 {ok} px 를 넘는데 알고리즘은 {verdict} 로 판정했습니다.",
		"align.truth.false-alarm": "오검출: 정답 최대 오차는 {err} px 뿐이며(OK 임계값 {ok} px 이내) 알고리즘은 {verdict} 로 판정했습니다.",
		"align.truth.strict":      "과도하게 엄격: 정답 최대 오차 {err} px 는 수동 확인이면 충분한데 알고리즘은 {verdict} 로 판정했습니다.",
		"align.truth.cautious":    "보수적: 정답 최대 오차 {err} px 는 {truth} 에 해당하며 알고리즘은 수동 확인을 요청했습니다.",
		"align.pose.item":         "항목",
		"align.pose.set":          "설정",
		"align.pose.measured":     "측정",
		"align.pose.delta":        "차이",
		"align.pose.shiftX":       "중심 이동 X (px)",
		"align.pose.shiftY":       "중심 이동 Y (px)",
		"align.pose.theta":        "회전 (°)",
		"align.pose.scale":        "배율",
		"align.result.modelNote":  "전단, 들뜸 또는 휨이 있으면 유사 변환으로 완전히 표현할 수 없어 정답 오차에 그 모델 오차가 포함됩니다.",
		"align.result.thresholds": "임계값 (전체 해상도 픽셀): OK ≤ {ok}, NG > {ng}",
		"align.section.roiResult": "ROI 별 결과",
		"align.col.truth":         "정렬 오차",
		"align.col.match":         "매칭 오차",
		"align.col.explain":       "매칭 오차: 이 ROI가 단독으로 찾은 위치가 실제 위치에서 얼마나 떨어졌는지. 정렬 오차: 전체 피팅이 이 ROI를 놓은 위치가 실제 위치에서 얼마나 떨어졌는지(기계가 실제로 쓰는 값). 아웃라이어는 매칭 오차는 크고 정렬 오차는 작은 ROI입니다.",
		"align.notes.title":       "ROI별 진단",
		"align.note.missing":      "{name}: 위치 확인 실패 ({why}, NCC {ncc}).",
		"align.note.outlier":      "{name}: 아웃라이어, 매칭 위치가 전체 피팅에서 {dev} px 벗어났습니다 (한계 {th} px).",
		"align.cause.overhang":    "실제 위치의 창이 이미지 밖으로 {px} px({out}%) 벗어나 이미지 안쪽 부분만으로 매칭합니다 (최소 겹침 {min}%)",
		"align.cause.tooFarOut":   "실제 위치의 창은 {in}%만 이미지 안에 있어 최소 겹침 {min}% 미만이므로 실제 위치를 찾지 못하고 이미지 가장자리에서 멈춥니다 ({px} px 벗어남)",
		"align.cause.occluded":    "창의 {pct}%가 가림으로 덮여 있습니다",
		"align.colon":             ": ",
		"align.sep":               "; ",
		"align.end":               ".",
		"align.detail.outlier":    "{name} (피팅에서 {dev} px, 한계 {th} px)",
		"align.detail.ambiguous":  "{name} (고유성 {u}, 최소 {min})",
		"align.detail.missing":    "{name} ({why})",
		"align.result.running":    "다시 계산 중입니다. 아래는 이전 결과입니다.",
		"align.col.status":        "상태",
		"align.status.inlier":     "인라이어",
		"align.status.outlier":    "아웃라이어",
		"align.legend.design":     "설계 위치",
		"align.legend.truth":      "실제 위치 (왜곡 후)",
		"align.legend.found":      "정렬 결과 (피팅, 색 = 정답 오차)",
		"align.legend.match":      "매칭 점 (초록 = 위치 확인, 빨강 = 실패)",
		"align.legend.shift":      "실제 이동",
	})
}

func init() {
	Messages(map[string]string{
		"align.match.minOverlap":     "最小重叠",
		"align.match.minOverlapNote": "匹配窗口落在图像内的最小比例。贴边的 ROI 在图像移动后真实位置会伸出图像：不低于该比例时只用图内部分匹配；1 = 必须完整在图内（旧行为）。",
		"align.match.slow":           "按当前金字塔和搜索半径，每次运行约需 {sec} 秒（粗估）。金字塔 1 在全分辨率上逐点搜索，建议用金字塔 4。",
	}, map[string]string{
		"align.match.minOverlap":     "Min overlap",
		"align.match.minOverlapNote": "Smallest share of the matching window that must lie inside the image. After the image moves, the true window of a ROI at the border sticks out: down to this share only the part inside is matched; 1 = the window must be completely inside (the old behaviour).",
		"align.match.slow":           "With this pyramid and search radius one run takes about {sec} s (rough estimate). Pyramid 1 searches every pixel at full resolution; pyramid 4 is recommended.",
	}, map[string]string{
		"align.match.minOverlap":     "최소 겹침",
		"align.match.minOverlapNote": "매칭 창이 이미지 안에 있어야 하는 최소 비율입니다. 이미지가 움직이면 가장자리 ROI의 실제 위치가 이미지 밖으로 나가는데, 이 비율 이상이면 안쪽 부분만으로 매칭합니다. 1 = 창이 완전히 이미지 안에 있어야 함(예전 동작).",
		"align.match.slow":           "현재 피라미드와 검색 반경에서는 한 번 실행에 약 {sec}초가 걸립니다(대략적 추정). 피라미드 1은 전체 해상도에서 모든 픽셀을 검색하므로 피라미드 4를 권장합니다.",
	})
}

func init() {
	Messages(map[string]string{
		"align.view.aligned":        "对齐后（算法结果）",
		"align.view.alignedOverlay": "对齐后叠加（模板红 / 对齐后青）",
		"align.view.alignedDiff":    "对齐后差异",
		"align.view.noFit":          "没有可用的对齐结果：先运行对齐（改了参数或遮挡后要重新运行）；定位成功的 ROI 少于 2 个时没有拟合结果。",
		"align.view.alignedNote":    "对齐后的图：把输入图按算法自己拟合出的相似变换重采样到模板网格，没有用到设定的错位参数。没有数据的边缘是黑色（差异图中是蓝色），遮挡区不参与比较。重合越好，叠加图越接近灰色，差异图越暗。",
		"align.view.noData":         "无数据",
		"align.legend.alignedTruth": "真实位置（按算法的变换回正后）",
		"align.legend.rest":         "剩余偏移（颜色=真值误差）",
		"align.gain.label":          "全图灰度差 对齐前→后",
		"align.check.label":         "对齐后局部残余位移 最大 / 均方根",
		"align.check.worst":         "最大残余 {px} px",
		"align.view.gotoWorst":      "放大到最大残余处",
		"align.check.note":          "把未扭曲的原图（不是假想图）切成 {total} 个窗口，在对齐后的图里重新找一遍（{n} 个找到），量出每个窗口还差多少。颜色按 OK/NG 阈值（OK ≤ {ok} px）。ROI 残差只看 ROI 处，这一项看整幅图。",
		"align.check.none":          "对齐后的图里没有可用于复核的窗口（图太小、没有数据或纹理不足）。",
		"align.gain.note":           "平均 |原图 − 图像| 灰度差（原图 = 扭曲前的真实图；用图像模板时就是模板），只计有数据的像素，遮挡区不计；越小越好。噪声、光照和翘起等相似变换描述不了的部分会让它降不到 0。",
	}, map[string]string{
		"align.view.aligned":        "Aligned (algorithm result)",
		"align.view.alignedOverlay": "Aligned overlay (template red / aligned cyan)",
		"align.view.alignedDiff":    "Aligned difference",
		"align.view.noFit":          "No alignment result to show: run the alignment first (again after changing parameters or occlusions); there is no fit when fewer than 2 ROIs were located.",
		"align.view.alignedNote":    "The aligned image is the input resampled onto the template grid with the similarity transform the algorithm fitted itself; the misalignment you set is not used. Edges without data are black (blue in the difference view) and occlusions are left out of the comparison. The better the match, the greyer the overlay and the darker the difference.",
		"align.view.noData":         "no data",
		"align.legend.alignedTruth": "True position (after undoing the fitted transform)",
		"align.legend.rest":         "Remaining offset (colour = truth error)",
		"align.gain.label":          "Whole-image grey difference, before → after",
		"align.check.label":         "Remaining local shift after aligning, max / rms",
		"align.check.worst":         "Largest remaining shift {px} px",
		"align.view.gotoWorst":      "Zoom to the largest remaining shift",
		"align.check.note":          "The un-warped source picture (not the design picture) is cut into {total} windows and each is searched again in the aligned image ({n} found); this is how far each one is still off. Colour by the OK/NG thresholds (OK ≤ {ok} px). The ROI residual only looks at the ROIs, this looks at the whole picture.",
		"align.check.none":          "The aligned image has no window to check with (too small, no data or too little texture).",
		"align.gain.note":           "Mean |source − image| grey difference (source = the un-warped real picture, the template itself with the image template) over the pixels that have data, occlusions left out; smaller is better. Noise, lighting and lift that a similarity cannot describe keep it above 0.",
	}, map[string]string{
		"align.view.aligned":        "정렬 후 (알고리즘 결과)",
		"align.view.alignedOverlay": "정렬 후 오버레이 (템플릿 빨강 / 정렬 후 청록)",
		"align.view.alignedDiff":    "정렬 후 차이",
		"align.view.noFit":          "표시할 정렬 결과가 없습니다. 먼저 정렬을 실행하세요(파라미터나 가림을 바꾼 뒤에는 다시 실행). 위치를 확인한 ROI가 2개 미만이면 피팅이 없습니다.",
		"align.view.alignedNote":    "정렬 후 이미지는 알고리즘이 스스로 피팅한 유사 변환으로 입력을 템플릿 격자에 리샘플링한 것이며, 설정한 어긋남 값은 쓰지 않습니다. 데이터가 없는 가장자리는 검정(차이 보기에서는 파랑)이고 가림 영역은 비교에서 제외됩니다. 잘 맞을수록 오버레이는 회색에 가깝고 차이는 어둡습니다.",
		"align.view.noData":         "데이터 없음",
		"align.legend.alignedTruth": "실제 위치 (피팅한 변환으로 되돌린 후)",
		"align.legend.rest":         "남은 어긋남 (색 = 정답 오차)",
		"align.gain.label":          "전체 이미지 밝기 차이, 정렬 전 → 후",
		"align.check.label":         "정렬 후 국소 잔여 이동량 최대 / RMS",
		"align.check.worst":         "가장 큰 잔여 이동량 {px} px",
		"align.view.gotoWorst":      "가장 큰 잔여 위치로 확대",
		"align.check.note":          "왜곡되지 않은 원본 이미지(가상 이미지가 아님)를 {total}개 창으로 나눠 정렬된 이미지에서 다시 찾고({n}개 찾음) 각 창이 아직 얼마나 벗어났는지 측정합니다. 색은 OK/NG 임계값 기준(OK ≤ {ok} px). ROI 잔차는 ROI 위치만 보지만 이 값은 이미지 전체를 봅니다.",
		"align.check.none":          "정렬된 이미지에 확인할 수 있는 창이 없습니다(이미지가 작거나 데이터·질감이 부족).",
		"align.gain.note":           "데이터가 있는 픽셀에 대한 평균 |원본 − 이미지| 밝기 차이(원본 = 왜곡 전 실제 이미지, 이미지 템플릿이면 템플릿 자체)이며 가림 영역은 제외합니다. 작을수록 좋고, 노이즈·조명·유사 변환으로 표현할 수 없는 들뜸 때문에 0까지 내려가지는 않습니다.",
	})
}

func init() {
	Messages(map[string]string{
		"align.legend.title": "图例",
	}, map[string]string{
		"align.legend.title": "Legend",
	}, map[string]string{
		"align.legend.title": "범례",
	})
}
