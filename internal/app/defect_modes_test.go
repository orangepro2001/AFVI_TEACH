package app

import "testing"

// Tests of the defect panel in its two modes (ported from profile_test.go; the other tests of
// that file belong to the owners of the files they exercise).

func TestPaletteSupportsByMode(t *testing.T) {
	teach := newTestApp(t, ProfileTeach)
	tdp := defectPanelOf(teach)
	if tdp == nil {
		t.Fatal("teach: no defect panel")
	}
	for _, tool := range []PaletteTool{PalettePick, PaletteRect, PaletteEllipse, PalettePolygon} {
		if tdp.PaletteSupports(tool) {
			t.Errorf("detect mode supports tool %v, want false", tool)
		}
	}

	studio := newTestApp(t, ProfileStudio)
	sdp := defectPanelOf(studio)
	if sdp == nil {
		t.Fatal("studio: no defect panel")
	}
	for _, tool := range []PaletteTool{PalettePick, PaletteRect, PaletteEllipse, PalettePolygon} {
		if !sdp.PaletteSupports(tool) {
			t.Errorf("author mode does not support tool %v, want true", tool)
		}
	}
}

func TestDetectModeInspects(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	legacyDetect(a)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	dp.requestRun()
	pollJobs(t, a)
	if dp.lastReport == nil {
		t.Fatal("lastReport is nil")
	}
	if dp.test == nil {
		t.Fatal("test image is nil")
	}
	if dp.golden == nil {
		t.Fatal("golden image is nil")
	}
	if !dp.learned[dp.channel] {
		t.Errorf("learned[%v] = false, want true", dp.channel)
	}
	if dp.showMarks {
		t.Error("showMarks = true in detect mode, want false")
	}
}

func TestAuthorModeRendersWithoutInspection(t *testing.T) {
	a := newTestApp(t, ProfileStudio)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	dp.requestRun()
	pollJobs(t, a)
	if dp.lastReport != nil {
		t.Error("lastReport is not nil in author mode")
	}
	if dp.test == nil {
		t.Fatal("test image is nil")
	}
	if dp.golden == nil {
		t.Fatal("golden image is nil")
	}
	if dp.learned[dp.channel] {
		t.Errorf("learned[%v] = true in author mode, want false", dp.channel)
	}
	if !dp.showMarks {
		t.Error("showMarks = false in author mode, want true")
	}
	if dp.renderedRev != dp.renderRev {
		t.Errorf("renderedRev = %d, renderRev = %d, want equal", dp.renderedRev, dp.renderRev)
	}
}
