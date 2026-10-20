package app

import (
	"bytes"
	"strings"
	"testing"

	"afvi/teach-imgui/internal/core"
)

// TestAlignTemplateDefaultsToTheDesignPicture checks that a freshly built align panel (the way
// alignPanelOf builds one, but without forcing the same-picture template) starts on the design
// picture: the whole point of the redesign is that a new panel no longer trivially self-matches.
func TestAlignTemplateDefaultsToTheDesignPicture(t *testing.T) {
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
	if ap.tplSource != tplDesign {
		t.Errorf("a new align panel uses template source %q, want %q", ap.tplSource, tplDesign)
	}
}

// TestAlignImageTemplateIsTheSourcePicture pins the old behaviour: with the "image" template at
// scale 1 the template is a copy of the MEDIAN, and the input is that picture warped by the panel's
// parameters (so the template and the un-warped input can never be the same image once shifted).
func TestAlignImageTemplateIsTheSourcePicture(t *testing.T) {
	a, ap := alignPanelOf(t)
	ap.params.Tx = 3
	ap.requestRun(true)
	pollJobs(t, a)
	if ap.template == nil || ap.input == nil {
		t.Fatal("no template or input after the run")
	}
	if !bytes.Equal(ap.template.Data, a.Scene.Median.Data) {
		t.Error("the image template is not a byte-for-byte copy of the MEDIAN picture")
	}
	if bytes.Equal(ap.input.Data, a.Scene.Median.Data) {
		t.Error("with tx=3 the warped input still equals the un-warped MEDIAN")
	}
}

// TestAlignDesignTemplateHasNoRealTexture checks that with the design template the picture the ROI
// patches are cut from is not the real texture: it has the MEDIAN's size but not its pixels, it is
// the cached design image, and the input is still the warped real picture.
func TestAlignDesignTemplateHasNoRealTexture(t *testing.T) {
	a, ap := alignPanelOf(t)
	ap.tplSource = tplDesign
	ap.params.Tx = 3
	ap.requestRun(true)
	pollJobs(t, a)
	if ap.design == nil {
		t.Fatal("no design cache after a design-template run")
	}
	if ap.template != ap.design.img {
		t.Fatal("the run did not use the cached design image as its template")
	}
	if ap.template.Width != a.Scene.Median.Width || ap.template.Height != a.Scene.Median.Height {
		t.Errorf("design template %dx%d, want the MEDIAN size %dx%d",
			ap.template.Width, ap.template.Height, a.Scene.Median.Width, a.Scene.Median.Height)
	}
	if bytes.Equal(ap.template.Data, a.Scene.Median.Data) {
		t.Error("the design template still holds the real texture of the MEDIAN")
	}
	// The input is always the warped real picture, never the design shape.
	sp := ap.scaledParams()
	want := core.ApplyWarp(a.Scene.Median, &sp, 1)
	if !bytes.Equal(ap.input.Data, want.Data) {
		t.Error("the input is not the warped MEDIAN picture")
	}
	if ap.designInfo != ap.design.info {
		t.Errorf("designInfo %+v, want the info cached in design.info %+v", ap.designInfo, ap.design.info)
	}
}

// TestAlignDesignTemplateIsCachedAcrossWarpChanges: the design picture only depends on the model,
// the source and the working scale, so changing the warp must reuse it; changing the scale rebuilds
// it, and loading a new model drops it.
func TestAlignDesignTemplateIsCachedAcrossWarpChanges(t *testing.T) {
	a, ap := alignPanelOf(t)
	ap.tplSource = tplDesign
	ap.requestRun(true)
	pollJobs(t, a)
	if ap.design == nil || ap.template != ap.design.img {
		t.Fatal("no cached design template after the first run")
	}
	d0 := ap.design

	ap.params.Tx = 3
	ap.dirty = true
	ap.requestRun(true)
	pollJobs(t, a)
	if ap.design != d0 {
		t.Error("changing the warp rebuilt the design cache")
	}
	if ap.template != ap.design.img {
		t.Error("after the warp change the template is not the cached design image")
	}

	ap.scale = 2
	ap.dirty = true
	ap.requestRun(true)
	pollJobs(t, a)
	if ap.design == d0 {
		t.Error("changing the scale did not rebuild the design cache")
	}
	if ap.template != ap.design.img {
		t.Error("after the scale change the template is not the rebuilt design image")
	}
	if ap.template.Width != d0.img.Width/2 || ap.template.Height != d0.img.Height/2 {
		t.Errorf("rebuilding at scale 2 gave a %dx%d template, want half of %dx%d",
			ap.template.Width, ap.template.Height, d0.img.Width, d0.img.Height)
	}

	ap.ModelLoaded()
	if ap.design != nil {
		t.Error("loading a model did not drop the design cache")
	}
}

// TestAlignDesignCacheFollowsTheCameraChannel: with the light source the design picture is built
// from the means of the displayed channel, so the cache and the report must follow a channel switch.
func TestAlignDesignCacheFollowsTheCameraChannel(t *testing.T) {
	a, ap := alignPanelOf(t)
	ap.tplSource = tplDesign
	ap.source = "light"
	a.Display = DisplayRed
	ap.dirty = true
	ap.requestRun(true)
	pollJobs(t, a)
	if ap.design == nil || ap.design.channel != core.ChRed {
		t.Fatalf("design cache %+v, want one built from the red channel", ap.design)
	}
	d0 := ap.design

	a.Display = DisplayGreen
	ap.dirty = true
	if !ap.reportStale() {
		t.Error("switching the camera channel must make the report stale")
	}
	ap.requestRun(true)
	pollJobs(t, a)
	if ap.design == d0 {
		t.Error("switching the camera channel reused the design cache of the other channel")
	}
	if ap.design == nil || ap.design.channel != core.ChGreen {
		t.Fatalf("design cache %+v, want one built from the green channel", ap.design)
	}
	d1 := ap.design

	ap.requestRun(true)
	pollJobs(t, a)
	if ap.design != d1 {
		t.Error("running again with the same channel rebuilt the design cache")
	}
}

// TestAlignTemplateSourceMakesTheReportStale: alignKey carries the template source, so flipping it
// invalidates the report without touching anything else.
func TestAlignTemplateSourceMakesTheReportStale(t *testing.T) {
	a, ap := alignPanelOf(t)
	ap.requestRun(true)
	pollJobs(t, a)
	if ap.report == nil {
		t.Fatal("no report")
	}
	if ap.reportStale() {
		t.Fatal("a fresh report is stale")
	}
	ap.tplSource = tplDesign
	if !ap.reportStale() {
		t.Error("switching to the design template must make the report stale")
	}
	ap.tplSource = tplImage
	if ap.reportStale() {
		t.Error("switching back to the image template must make the report current again")
	}
}

// TestAlignReferenceIsTheUnwarpedRealPicture pins what the whole-picture checks measure against:
// the real un-warped picture the input was warped from, never the shape-only design template.
func TestAlignReferenceIsTheUnwarpedRealPicture(t *testing.T) {
	a, ap := alignPanelOf(t)
	ap.tplSource = tplImage
	ap.dirty = true
	ap.requestRun(true)
	pollJobs(t, a)
	if ap.reference == nil {
		t.Fatal("no reference after an image-template run")
	}
	if !bytes.Equal(ap.reference.Data, ap.template.Data) {
		t.Error("with the image template the reference is not the template")
	}
	if !bytes.Equal(ap.reference.Data, a.Scene.Median.Data) {
		t.Error("with the image template the reference is not the MEDIAN picture")
	}

	ap.tplSource = tplDesign
	ap.dirty = true
	ap.requestRun(true)
	pollJobs(t, a)
	if ap.reference == nil {
		t.Fatal("no reference after a design-template run")
	}
	if !bytes.Equal(ap.reference.Data, a.Scene.Median.Data) {
		t.Error("with the design template the reference is not the un-warped MEDIAN picture")
	}
	if bytes.Equal(ap.template.Data, a.Scene.Median.Data) {
		t.Error("the design template still holds the real texture of the MEDIAN")
	}
	if ap.reference == ap.template {
		t.Error("with the design template the reference is the template")
	}

	ap.ModelLoaded()
	if ap.reference != nil {
		t.Error("loading a model did not drop the reference")
	}
}

// TestAlignTemplateTextsHaveAllLanguages keeps the template combo and its notes translated in every
// language, and pins the placeholders of the registration line.
func TestAlignTemplateTextsHaveAllLanguages(t *testing.T) {
	keys := []string{
		"align.template", "align.template.design", "align.template.image",
		"align.template.designNote", "align.template.imageNote", "align.template.registered",
	}
	for _, lang := range []Lang{LangZh, LangEn, LangKo} {
		for _, k := range keys {
			if dictionaries[lang][k] == "" {
				t.Errorf("dictionaries[%s][%q] is missing", lang, k)
			}
		}
		reg := dictionaries[lang]["align.template.registered"]
		for _, ph := range []string{"{dx}", "{dy}", "{deg}"} {
			if !strings.Contains(reg, ph) {
				t.Errorf("dictionaries[%s][align.template.registered] = %q lacks the placeholder %s", lang, reg, ph)
			}
		}
	}
}
