package app

import (
	"math"
	"strings"
	"testing"

	"afvi/teach-imgui/internal/core"
)

// alignPanelOf returns the teach app's align panel with two small ROIs on the synthetic model.
func alignPanelOf(t *testing.T) (*App, *AlignPanel) {
	t.Helper()
	SetLang(LangZh)
	a := newTestApp(t, ProfileTeach)
	var ap *AlignPanel
	for _, p := range a.panels {
		if x, ok := p.(*AlignPanel); ok {
			ap = x
		}
	}
	if ap == nil {
		t.Fatal("no align panel")
	}
	ap.ModelLoaded()
	// These tests exercise the diagnostics and staleness logic on the 96x64 synthetic model with the
	// same-picture template, where every ROI trivially matches; the design-picture tests live in
	// align_design_test.go.
	ap.tplSource = tplImage
	ap.scale = 1
	ap.rois = []core.AlignRoi{
		{Name: "A", Rect: core.Rect{X: 2, Y: 2, W: 30, H: 30}},
		{Name: "B", Rect: core.Rect{X: 60, Y: 20, W: 30, H: 30}},
	}
	return a, ap
}

func TestAlignReportIsStaleWhenTheInputsChange(t *testing.T) {
	a, ap := alignPanelOf(t)
	ap.requestRun(true)
	pollJobs(t, a)
	if ap.report == nil {
		t.Fatal("no report")
	}
	if ap.dirty || ap.reportStale() {
		t.Fatalf("a fresh report is stale: dirty=%v stale=%v", ap.dirty, ap.reportStale())
	}

	ap.occs = append(ap.occs, core.Occlusion{Rect: core.Rect{X: 10, Y: 10, W: 20, H: 20}})
	if !ap.reportStale() {
		t.Error("adding an occlusion must make the report stale")
	}
	ap.occs = nil
	if ap.reportStale() {
		t.Error("restoring the inputs must make the report current again")
	}
	ap.params.Tx = 3
	if !ap.reportStale() {
		t.Error("changing the warp must make the report stale")
	}
	ap.params.Tx = 0
	ap.config.MinOverlap = 0.7
	if !ap.reportStale() {
		t.Error("changing a matching parameter must make the report stale")
	}
	ap.config.MinOverlap = core.DefaultAlignConfig.MinOverlap
	ap.rois[0].X += 1
	if !ap.reportStale() {
		t.Error("moving a ROI must make the report stale")
	}
	ap.rois[0].X -= 1
	if ap.reportStale() {
		t.Error("moving the ROI back must make the report current again")
	}
}

// A slow run (pyramid 1 takes 30 s) can finish after the user has changed something; it must not
// mark the newer edit clean, or the verdict panel would present the old result as current.
func TestFinishedJobDoesNotCleanNewerEdits(t *testing.T) {
	a, ap := alignPanelOf(t)
	ap.params.Tx = 2
	ap.dirty = true
	ap.requestRun(false)
	// The user edits again before the first run is applied.
	ap.params.Ty = 3
	ap.dirty = true
	pollJobs(t, a)
	if ap.report == nil {
		t.Fatal("no report")
	}
	if !ap.dirty {
		t.Error("the older job cleared dirty although the warp changed after it started")
	}
	if !ap.reportStale() {
		t.Error("the report of the older job is shown as current")
	}
	ap.requestRun(false)
	pollJobs(t, a)
	if ap.dirty || ap.reportStale() {
		t.Errorf("after the rerun: dirty=%v stale=%v", ap.dirty, ap.reportStale())
	}
	if got := ap.reportParams.Ty; got != 3 {
		t.Errorf("the rerun used ty %v, want 3", got)
	}
}

func TestRunComputesAPerRoiDiagnosis(t *testing.T) {
	a, ap := alignPanelOf(t)
	ap.requestRun(true)
	pollJobs(t, a)
	if len(ap.diag) != len(ap.report.Matches) || len(ap.diag) != 2 {
		t.Fatalf("%d diagnoses for %d matches", len(ap.diag), len(ap.report.Matches))
	}
}

// reportWithNotes fills the panel with a hand-made report: P4 fine but partly outside the image,
// P5 fine, P7 an outlier whose true window is mostly outside, P8 not located and covered.
func reportWithNotes(ap *AlignPanel, cfg core.AlignConfig) {
	ap.lastScale = 2
	ap.reportConfig = cfg
	found := func(x, y float64) *core.Point { return &core.Point{X: x, Y: y} }
	ap.report = &core.AlignReport{
		Matches: []core.RoiMatch{
			{Name: "P4", Located: true, Found: found(1, 1), Ncc: 0.99, Uniqueness: 0.8},
			{Name: "P5", Located: true, Found: found(1, 1), Ncc: 0.99, Uniqueness: 0.8},
			{Name: "P7", Located: true, Found: found(1, 1), Ncc: 0.76, Uniqueness: 0.6},
			{Name: "P8", Located: false, Found: found(1, 1), Ncc: 0.04, Uniqueness: 0.02, Reason: "low-ncc"},
		},
		Residuals: []float64{0.3, 0.2, -1, -1},
		Level:     core.VerdictManual,
	}
	ap.diag = []core.RoiDiagnosis{
		{MatchErrPx: 0.4, FitDevPx: 0.3, OverhangPx: 6, InsideShare: 0.95},
		{MatchErrPx: 0.3, FitDevPx: 0.2, InsideShare: 1},
		{MatchErrPx: 4.5, FitDevPx: 5.1, OverhangPx: 30, InsideShare: 0.8},
		{MatchErrPx: 250, FitDevPx: 249, InsideShare: 1, Occluded: 0.5},
	}
}

func TestRoiNotesExplainWhyAnRoiFailed(t *testing.T) {
	SetLang(LangZh)
	_, ap := alignPanelOf(t)
	reportWithNotes(ap, core.DefaultAlignConfig)
	notes := ap.roiNotes()
	if len(notes) != 3 {
		t.Fatalf("%d notes, want 3 (P5 has nothing to say): %+v", len(notes), notes)
	}
	byName := map[string]roiNote{}
	for _, n := range notes {
		for _, name := range []string{"P4", "P5", "P7", "P8"} {
			if strings.HasPrefix(n.text, name) {
				byName[name] = n
			}
		}
	}
	if _, ok := byName["P5"]; ok {
		t.Error("P5 is fine and must not have a note")
	}
	p4 := byName["P4"]
	if p4.level != core.VerdictOK || !strings.Contains(p4.text, "12.0") || !strings.Contains(p4.text, "5%") || !strings.Contains(p4.text, "90%") {
		t.Errorf("P4 note %+v: want an information line with 12.0 px, 5%% and 90%%", p4)
	}
	p7 := byName["P7"]
	if p7.level != core.VerdictManual || !strings.Contains(p7.text, "外点") || !strings.Contains(p7.text, "10.2") ||
		!strings.Contains(p7.text, "只有 80% 在图内") || !strings.Contains(p7.text, "60.0") {
		t.Errorf("P7 note %+v: want outlier, 10.2 px off the fit, 80%% inside, 60.0 px out", p7)
	}
	p8 := byName["P8"]
	if p8.level != core.VerdictNG || !strings.Contains(p8.text, "未定位") || !strings.Contains(p8.text, "50%") {
		t.Errorf("P8 note %+v: want not located and 50%% covered", p8)
	}

	// With MinOverlap 0 (the old behaviour) any window that leaves the image cannot be matched.
	strict := core.DefaultAlignConfig
	strict.MinOverlap = 0
	reportWithNotes(ap, strict)
	for _, n := range ap.roiNotes() {
		if strings.HasPrefix(n.text, "P4") && !strings.Contains(n.text, "低于最小重叠 100%") {
			t.Errorf("strict P4 note %q must say the overlap is below 100%%", n.text)
		}
	}
}

func TestVerdictReasonsNameTheRois(t *testing.T) {
	_, ap := alignPanelOf(t)
	reportWithNotes(ap, core.DefaultAlignConfig)
	cfg := ap.reportConfig
	out := ap.reasonText(core.AlignVerdictReason{Code: "outlier"})
	if !strings.Contains(out, "P7") || !strings.Contains(out, "10.2") || !strings.Contains(out, F(ap.fullPx(cfg.RansacThresholdPx), 1)) {
		t.Errorf("outlier reason %q does not name P7 with its numbers", out)
	}
	if strings.Contains(out, "P4") || strings.Contains(out, "P8") {
		t.Errorf("outlier reason %q names an ROI that is not an outlier", out)
	}
	out = ap.reasonText(core.AlignVerdictReason{Code: "roi-missing"})
	if !strings.Contains(out, "P8") || !strings.Contains(out, "NCC 过低") {
		t.Errorf("roi-missing reason %q does not name P8 and its cause", out)
	}
	ap.report.Matches[1].Uniqueness = 0.05
	out = ap.reasonText(core.AlignVerdictReason{Code: "ambiguous"})
	if !strings.Contains(out, "P5") || !strings.Contains(out, "0.050") {
		t.Errorf("ambiguous reason %q does not name P5 with its uniqueness", out)
	}
	ap.report = nil
	if out := ap.reasonText(core.AlignVerdictReason{Code: "outlier"}); out != T("align.reason.outlier") {
		t.Errorf("without a report the reason is the plain text, got %q", out)
	}
}

func TestSlowRunWarningThreshold(t *testing.T) {
	rois := make([]core.AlignRoi, 6)
	for i := range rois {
		rois[i] = core.AlignRoi{Name: "r", Rect: core.Rect{W: 280, H: 250}}
	}
	cfg := core.DefaultAlignConfig
	if est := core.MatchCost(rois, cfg) / matchOpsPerSecond; est >= slowRunSeconds {
		t.Errorf("the default configuration is estimated at %.1f s and would warn", est)
	}
	cfg.PyramidFactor = 1
	cfg.SearchRadius = 230
	if est := core.MatchCost(rois, cfg) / matchOpsPerSecond; est < slowRunSeconds || math.IsNaN(est) {
		t.Errorf("pyramid 1 with radius 230 is estimated at %.1f s and would not warn", est)
	}
}

func TestAlignDiagnosisTextsHaveAllLanguages(t *testing.T) {
	keys := []string{
		"align.col.match", "align.col.truth", "align.col.explain", "align.notes.title",
		"align.note.missing", "align.note.outlier", "align.cause.overhang", "align.cause.tooFarOut", "align.cause.occluded",
		"align.colon", "align.sep", "align.end", "align.detail.outlier", "align.detail.ambiguous", "align.detail.missing",
		"align.result.running", "align.match.minOverlap", "align.match.minOverlapNote", "align.match.slow",
	}
	for _, lang := range []Lang{LangZh, LangEn, LangKo} {
		for _, k := range keys {
			if dictionaries[lang][k] == "" {
				t.Errorf("dictionaries[%s][%q] is missing", lang, k)
			}
		}
	}
}

func TestPctText(t *testing.T) {
	cases := map[float64]string{0: "0", 0.004: "0.4", 0.05: "5", 0.5: "50", 0.9: "90", 0.992: "99.2", 0.9998: "99.9", 1: "100"}
	for in, want := range cases {
		if got := pctText(in); got != want {
			t.Errorf("pctText(%v) = %q, want %q", in, got, want)
		}
	}
}
