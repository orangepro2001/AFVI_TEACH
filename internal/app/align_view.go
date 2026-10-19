package app

import (
	"fmt"
	"math"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/core"
)

// The align tab's views: the overlay shows the template in red and the warped input in cyan,
// so a misalignment is visible as colour fringes without running anything.
var alignViews = []string{"overlay", "template", "input", "diff"}

func alignViewIndex(kind string) int {
	for i, v := range alignViews {
		if v == kind {
			return i
		}
	}
	return 0
}

// packAlignView renders the texture of a view kind.
func packAlignView(kind string, template, input *core.GrayImage) []byte {
	switch kind {
	case "overlay":
		return PackOverlay(template, input)
	case "input":
		return PackGray(input)
	case "diff":
		return PackAbsDiff(template, input)
	}
	return PackGray(template)
}

// levelColor is the text colour of a verdict level.
func levelColor(l core.VerdictLevel) imgui.Vec4 {
	pal := CurrentPalette()
	switch l {
	case core.VerdictOK:
		return pal.OK
	case core.VerdictManual:
		return pal.Warn
	}
	return pal.NG
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
	bg, fg := rgba(0, 192, 0, 255), rgba(0, 0, 0, 255)
	switch level {
	case core.VerdictManual:
		bg = rgba(255, 216, 0, 255)
	case core.VerdictNG:
		bg, fg = rgba(255, 0, 0, 255), rgba(255, 255, 255, 255)
	}
	avail := imgui.ContentRegionAvail().X
	imgui.PushFont(imgui.CurrentFont(), imgui.CurrentStyle().FontSizeBase()*1.5)
	ts := imgui.CalcTextSize(title)
	h := ts.Y + sc(14)
	pos := imgui.CursorScreenPos()
	end := imgui.NewVec2(pos.X+avail, pos.Y+h)
	dl := imgui.WindowDrawList()
	dl.AddRectFilled(pos, end, bg)
	DrawEdge(dl, pos, end, EdgeSunken)
	dl.AddTextVec2(imgui.NewVec2(pos.X+(avail-ts.X)/2, pos.Y+(h-ts.Y)/2), fg, title)
	imgui.PopFont()
	imgui.Dummy(imgui.NewVec2(avail, h))
}

// tableCell is one cell of ResultTable; a nil colour keeps the default text colour.
type tableCell struct {
	text string
	col  *imgui.Vec4
}

// ResultTable shows a small read-only grid with optionally coloured cells.
func ResultTable(id string, headers []string, rows [][]tableCell) {
	flags := imgui.TableFlagsRowBg | imgui.TableFlagsBorders | imgui.TableFlagsSizingStretchProp
	if !imgui.BeginTableV("##rt-"+id+"-"+string(CurrentLang()), int32(len(headers)), flags, imgui.NewVec2(0, 0), 0) {
		return
	}
	for _, h := range headers {
		imgui.TableSetupColumnV(h, 0, 0, 0)
	}
	imgui.TableHeadersRow()
	for _, row := range rows {
		imgui.TableNextRow()
		for c, cell := range row {
			imgui.TableSetColumnIndex(int32(c))
			if cell.col != nil {
				imgui.PushStyleColorVec4(imgui.ColText, *cell.col)
			}
			imgui.TextUnformatted(cell.text)
			if cell.col != nil {
				imgui.PopStyleColor()
			}
		}
	}
	imgui.EndTable()
}

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
	if p.dirty {
		Warn(T("align.result.stale"))
	}
	for _, reason := range r.Reasons {
		Warn(p.reasonText(reason))
	}

	imgui.Spacing()
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

	imgui.Spacing()
	if r.Fit != nil && p.reportW > 0 {
		sc := float64(p.lastScale)
		truthPose := core.TruthPose(&p.reportParams, p.reportW, p.reportH)
		fitPose := core.FitPose(*r.Fit, p.reportW, p.reportH)
		shiftCell := func(set, got float64) []tableCell {
			d := got - set
			col := levelColor(errLevel(math.Abs(d), cfg))
			return []tableCell{{text: F(set*sc, 2)}, {text: F(got*sc, 2)}, {text: F(d*sc, 2), col: &col}}
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
	imgui.AlignTextToFramePadding()
	imgui.TextUnformatted(T("align.result.residualRms") + ": ")
	imgui.SameLine()
	imgui.PushStyleColorVec4(imgui.ColText, rmsCol)
	imgui.TextUnformatted(F(p.fullPx(r.ResidualRms), 3) + " px")
	imgui.PopStyleColor()
	Note(T("align.result.thresholds", "ok", F(p.fullPx(cfg.OkResidualPx), 2), "ng", F(p.fullPx(cfg.NgResidualPx), 2)))
	KVTable("align-kv", [][2]string{
		{T("align.result.meanNcc"), F(r.MeanNcc, 4)},
		{T("align.result.elapsed"), F(p.elapsedMs, 1) + " ms"},
	})
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
		T("align.col.residual"), T("align.col.truth"), T("align.col.status"),
	}
	matches := r.Matches
	cell := func(row, col int) (string, *imgui.Vec4) {
		m := matches[row]
		switch col {
		case 0:
			return m.Name, nil
		case 1:
			return F(m.Ncc, 4), nil
		case 2:
			return F(m.Uniqueness, 4), nil
		case 3:
			if row < len(r.Residuals) && r.Residuals[row] >= 0 {
				c := levelColor(errLevel(r.Residuals[row], cfg))
				return F(p.fullPx(r.Residuals[row]), 3), &c
			}
			return "—", nil
		case 4:
			if p.truth != nil && row < len(p.truth.PerRoiErrorPx) && !math.IsNaN(p.truth.PerRoiErrorPx[row]) {
				e := p.truth.PerRoiErrorPx[row]
				c := levelColor(errLevel(e, cfg))
				return F(p.fullPx(e), 3), &c
			}
			return "—", nil
		}
		if !m.Located {
			c := levelColor(core.VerdictNG)
			return p.matchReasonText(m.Reason), &c
		}
		if row < len(r.Residuals) && r.Residuals[row] >= 0 {
			c := levelColor(core.VerdictOK)
			return T("align.status.inlier"), &c
		}
		c := levelColor(core.VerdictManual)
		return T("align.status.outlier"), &c
	}
	clicked := DataTableColored("align-matches", headers, len(matches), 8*imgui.FrameHeightWithSpacing()+8, cell, p.selected)
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
	p.drawAlignLegend(o)
}

// drawAlignLegend explains the overlay's colours in the corner of the canvas.
func (p *AlignPanel) drawAlignLegend(o *Overlay) {
	type entry struct {
		col  uint32
		text string
	}
	entries := []entry{
		{OverlayMuted, T("align.legend.design")},
		{OverlayROI, T("align.legend.truth")},
		{OverlayOK, T("align.legend.found")},
		{OverlayNG, T("align.legend.match")},
		{OverlayWarn, T("align.legend.shift")},
	}
	var width float32
	for _, e := range entries {
		if tw := imgui.CalcTextSize(e.text).X; tw > width {
			width = tw
		}
	}
	line := imgui.TextLineHeightWithSpacing()
	pad := sc(6)
	boxW := width + sc(16) + 2*pad
	boxH := float32(len(entries))*line + 2*pad
	// Bottom-right corner: the top-right holds the tool palette and the ROIs sit mostly on the left.
	origin := imgui.NewVec2(o.Max.X-boxW-sc(8), o.Max.Y-boxH-sc(8))
	end := imgui.NewVec2(origin.X+boxW, origin.Y+boxH)
	o.DL.AddRectFilled(origin, end, rgba(0, 0, 0, 150))
	for i, e := range entries {
		at := imgui.NewVec2(origin.X+pad, origin.Y+pad+float32(i)*line+2)
		legendSwatch(o.DL, at, e.col, e.text, rgba(255, 255, 255, 255))
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
		"align.col.truth":         "真值误差",
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
		"align.col.truth":         "Truth error",
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
		"align.col.truth":         "정답 오차",
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
