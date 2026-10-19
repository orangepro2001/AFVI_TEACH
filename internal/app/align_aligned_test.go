package app

import (
	"strings"
	"testing"

	"afvi/teach-imgui/internal/core"
)

// texturedImage is a random block pattern (8 px blocks, seeded) that the matcher can lock onto.
func texturedImage(w, h int) *core.GrayImage {
	img := core.NewGray(w, h, 0)
	rng := core.NewRng(77)
	const block = 8
	for by := 0; by < h; by += block {
		for bx := 0; bx < w; bx += block {
			v := uint8(int(rng.Next() * 256))
			for y := by; y < by+block && y < h; y++ {
				for x := bx; x < bx+block && x < w; x++ {
					img.Data[y*w+x] = v
				}
			}
		}
	}
	return img
}

// alignedPanel is an align panel holding a textured template, an input shifted and rotated by a
// known warp and the report the algorithm made from them, as a finished run leaves it.
func alignedPanel(t *testing.T) (*App, *AlignPanel, core.WarpParams) {
	t.Helper()
	a, ap := alignPanelOf(t)
	const w, h = 360, 260
	template := texturedImage(w, h)
	warp := core.WarpParams{Tx: 6, Ty: -4, ThetaDeg: 0.4, Scale: 1, Gain: 1}
	input := core.ApplyWarp(template, &warp, 1)
	ap.rois = []core.AlignRoi{
		{Name: "A", Rect: core.Rect{X: 30, Y: 30, W: 60, H: 60}},
		{Name: "B", Rect: core.Rect{X: w - 90, Y: 30, W: 60, H: 60}},
		{Name: "C", Rect: core.Rect{X: 30, Y: h - 90, W: 60, H: 60}},
		{Name: "D", Rect: core.Rect{X: w - 90, Y: h - 90, W: 60, H: 60}},
	}
	rep := core.AlignWithRois(template, input, ap.rois, core.DefaultAlignConfig, core.Similarity{Scale: 1})
	if rep.Fit == nil || rep.Level != core.VerdictOK {
		t.Fatalf("no usable fit for the test pattern: %s %+v", rep.Level, rep.Reasons)
	}
	ap.template, ap.input = template, input
	ap.report = &rep
	ap.fitInput = input
	ap.reportW, ap.reportH = w, h
	ap.reportParams = warp
	ap.reportConfig = core.DefaultAlignConfig
	ap.lastScale = 1
	return a, ap, warp
}

func TestAlignedViewIsMadeFromTheFoundFit(t *testing.T) {
	a, ap, warp := alignedPanel(t)
	if ap.alignedNow() != nil {
		t.Fatal("an aligned image exists before it was made")
	}
	ap.viewKind = "alignedOverlay"
	ap.requestView()
	pollJobs(t, a)
	al := ap.alignedNow()
	if al == nil {
		t.Fatal("no aligned image after requestView")
	}
	if al.fit != *ap.report.Fit {
		t.Errorf("aligned with %+v, want the fit the report found %+v", al.fit, *ap.report.Fit)
	}
	if truth := core.WarpToSimilarity(&warp, ap.template.Width, ap.template.Height); al.fit == truth {
		t.Error("the aligned image used the true warp")
	}
	if !(al.after < al.before/4) {
		t.Errorf("mean difference %.2f -> %.2f, want at least a four-fold drop", al.before, al.after)
	}
	if al.used < ap.template.Width*ap.template.Height/2 {
		t.Errorf("only %d pixels compared", al.used)
	}
	if al.check.Windows < 3 || al.check.MaxPx > 2 {
		t.Errorf("local check: %+v, want at least 3 windows found and a leftover below 2 px", al.check)
	}

	// The cache is reused for the same input and fit and dropped for another input.
	if again := alignedFor(al, ap.template, ap.input, ap.report.Fit, nil, ap.config.PyramidFactor); again != al {
		t.Error("the aligned image was made again for the same input and fit")
	}
	ap.input = ap.input.Clone()
	if ap.alignFit() != nil || ap.alignedNow() != nil {
		t.Error("a fit is offered for an input it was not computed for")
	}
}

// Changing the warp without running the alignment generates a new input; the old fit must not be
// applied to it.
func TestAlignedViewIsNotOfferedForARegeneratedInput(t *testing.T) {
	a, ap, _ := alignedPanel(t)
	ap.viewKind = "aligned"
	ap.requestView()
	pollJobs(t, a)
	if ap.alignedNow() == nil {
		t.Fatal("no aligned image")
	}
	ap.autoRun = false
	ap.params.Tx = 9
	ap.dirty = true
	ap.requestRun(false)
	pollJobs(t, a)
	if ap.alignFit() != nil || ap.alignedNow() != nil {
		t.Error("the fit of the previous input is still applied to the new input")
	}
	pix := packAlignView(ap.viewKind, ap.template, ap.input, ap.alignedNow())
	defer PutRGBA(pix)
	if pix[0] != 48 || pix[1] != 48 || pix[2] != 48 || pix[3] != 255 {
		t.Errorf("without a result the aligned view is %v, want flat grey 48", pix[:4])
	}
	// Running again brings it back.
	ap.requestRun(true)
	pollJobs(t, a)
	if ap.report == nil {
		t.Fatal("no report after the run")
	}
	if ap.report.Fit != nil && ap.alignFit() == nil {
		t.Error("a run that found a fit offers no aligned image")
	}
}

func TestPackAlignedViews(t *testing.T) {
	_, ap, _ := alignedPanel(t)
	al := alignedFor(nil, ap.template, ap.input, ap.report.Fit, nil, ap.config.PyramidFactor)
	if al == nil {
		t.Fatal("no aligned image")
	}
	w := ap.template.Width
	last := len(ap.template.Data) - 1 // the bottom-right corner has no data for this shift
	if al.re.Valid[last] {
		t.Fatal("the test needs a pixel without data")
	}
	cases := map[string]func(i int) [3]uint8{
		"aligned":        func(i int) [3]uint8 { v := al.re.Image.Data[i]; return [3]uint8{v, v, v} },
		"alignedOverlay": func(i int) [3]uint8 { v := al.re.Image.Data[i]; return [3]uint8{ap.template.Data[i], v, v} },
		"alignedDiff": func(i int) [3]uint8 {
			if !al.re.Valid[i] {
				return [3]uint8{0, 0, 96}
			}
			d := int(ap.template.Data[i]) - int(al.re.Image.Data[i])
			if d < 0 {
				d = -d
			}
			return [3]uint8{uint8(d), uint8(d), uint8(d)}
		},
	}
	for kind, want := range cases {
		pix := packAlignView(kind, ap.template, ap.input, al)
		for _, i := range []int{0, w + 5, len(ap.template.Data) / 2, last} {
			got := [3]uint8{pix[4*i], pix[4*i+1], pix[4*i+2]}
			if got != want(i) || pix[4*i+3] != 255 {
				t.Errorf("%s pixel %d = %v, want %v", kind, i, pix[4*i:4*i+4], want(i))
			}
		}
		PutRGBA(pix)
	}
}

func TestHoverAligned(t *testing.T) {
	a, ap, _ := alignedPanel(t)
	ap.viewKind = "alignedOverlay"
	if got := ap.Hover(10, 10); got != "" {
		t.Errorf("hover without an aligned image: %q", got)
	}
	ap.requestView()
	pollJobs(t, a)
	i := 40*ap.template.Width + 50
	al := ap.alignedNow()
	if got, want := ap.Hover(50, 40), "T "+itoa(int(ap.template.Data[i]))+"  A "+itoa(int(al.re.Image.Data[i])); got != want {
		t.Errorf("overlay hover %q, want %q", got, want)
	}
	ap.viewKind = "aligned"
	if got, want := ap.Hover(50, 40), "GV "+itoa(int(al.re.Image.Data[i])); got != want {
		t.Errorf("aligned hover %q, want %q", got, want)
	}
	SetLang(LangZh)
	if got := ap.Hover(ap.template.Width-1, ap.template.Height-1); got != T("align.view.noData") {
		t.Errorf("hover over a pixel without data: %q", got)
	}
	if got := ap.Hover(-1, 0); got != "" {
		t.Errorf("hover outside the image: %q", got)
	}
}

func itoa(v int) string { return strings.TrimSpace(F(float64(v), 0)) }

func TestAlignedTextsHaveAllLanguages(t *testing.T) {
	keys := []string{
		"align.view.aligned", "align.view.alignedOverlay", "align.view.alignedDiff", "align.view.noFit",
		"align.view.alignedNote", "align.view.noData", "align.legend.alignedTruth", "align.legend.rest",
		"align.gain.label", "align.gain.note", "align.check.label", "align.check.note", "align.check.none", "align.check.worst", "align.view.gotoWorst",
	}
	for _, lang := range []Lang{LangZh, LangEn, LangKo} {
		for _, k := range keys {
			if dictionaries[lang][k] == "" {
				t.Errorf("dictionaries[%s][%q] is missing", lang, k)
			}
		}
	}
	for _, v := range alignViews {
		if !isAlignedView(v) {
			continue
		}
		if dictionaries[LangZh]["align.view."+v] == "" {
			t.Errorf("aligned view %q has no name", v)
		}
	}
}
