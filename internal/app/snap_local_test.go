package app

import (
	"bytes"
	"path/filepath"
	"testing"

	"afvi/teach-imgui/internal/core"
	snapfmt "afvi/teach-imgui/internal/snap"
)

func TestLocalFramesMatchSimulation(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	meta := dp.snapMeta()
	got, err := localFrames(a.Scene, &meta, core.ChRed, []string{snapfmt.KindTest, snapfmt.KindGood})
	if err != nil {
		t.Fatalf("localFrames: %v", err)
	}
	light := a.Light.Clone()
	_, wantTest := core.RenderDefectPairWarped(a.Scene.Light, &light, core.ChRed, dp.sceneStrokes(), dp.gvStrokes(), dp.noise, dp.warp)
	wantGood := core.RenderGoodSample(a.Scene.Light, &light, core.ChRed, goodSampleNoise(dp.noise))
	test := got[snapKey{ch: core.ChRed, kind: snapfmt.KindTest}]
	good := got[snapKey{ch: core.ChRed, kind: snapfmt.KindGood}]
	if test == nil || good == nil {
		t.Fatal("missing frames")
	}
	if !bytes.Equal(test.Data, wantTest.Data) {
		t.Error("test image differs from the simulation")
	}
	if !bytes.Equal(good.Data, wantGood.Data) {
		t.Error("good image differs from the simulation")
	}
}

func TestLocalFramesRebakeProduct(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	meta := dp.snapMeta()
	baseMeta := dp.snapMeta()
	sceneLight := a.Scene.Light
	po := a.Scene.Product
	po.Seed += 7
	po.Strength *= 4
	light := a.Light.Clone()
	meta.Recipe = snapfmt.NewRecipe(po, light, dp.noise, dp.gvStrokes(), dp.sceneStrokes()).WithWarp(dp.warp)
	got, err := localFrames(a.Scene, &meta, core.ChRed, []string{snapfmt.KindTest})
	if err != nil {
		t.Fatalf("localFrames: %v", err)
	}
	ls, err := core.NewLightScene(a.Scene.Median, a.Scene.Material)
	if err != nil {
		t.Fatalf("NewLightScene: %v", err)
	}
	core.ApplyProductVariation(ls, po)
	_, want := core.RenderDefectPairWarped(ls, &light, core.ChRed, dp.sceneStrokes(), dp.gvStrokes(), dp.noise, dp.warp)
	test := got[snapKey{ch: core.ChRed, kind: snapfmt.KindTest}]
	if test == nil {
		t.Fatal("missing test frame")
	}
	if !bytes.Equal(test.Data, want.Data) {
		t.Error("rebaked test image differs from the simulation")
	}
	base, err := localFrames(a.Scene, &baseMeta, core.ChRed, []string{snapfmt.KindTest})
	if err != nil {
		t.Fatalf("localFrames base: %v", err)
	}
	if bytes.Equal(test.Data, base[snapKey{ch: core.ChRed, kind: snapfmt.KindTest}].Data) {
		t.Error("rebaked image equals the original product's image")
	}
	if a.Scene.Light != sceneLight {
		t.Error("scene light changed")
	}
}

func TestSnapFetchWithoutCamera(t *testing.T) {
	t.Setenv("AFVI_CAMERA_EXE", "")
	if findCameraExe() != "" {
		t.Skip("camera program found; set AFVI_CAMERA_EXE to an empty value to run this test")
	}
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	meta := dp.snapMeta()
	res, err := snapFetch("", a.Scene, "", &meta, nil, "", core.ChRed, true, a.Scene.W, a.Scene.H)
	if err != nil {
		t.Fatalf("snapFetch: %v", err)
	}
	if res.test == nil || res.good == nil {
		t.Fatal("test or good image is nil")
	}
	if len(res.add) != 2 {
		t.Errorf("add has %d entries, want 2", len(res.add))
	}
	if res.rendered {
		t.Error("rendered = true, want false")
	}
}

func TestSaveLocalRoundTrip(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	dir := t.TempDir()
	meta := dp.snapMeta()
	out, err := saveLocal(dir, a.Scene, &meta)
	if err != nil {
		t.Fatalf("saveLocal: %v", err)
	}
	got, d, err := snapfmt.ReadMeta(dir)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if len(got.Images) != 6 {
		t.Errorf("%d images, want 6", len(got.Images))
	}
	if len(out.Images) != 6 {
		t.Errorf("%d images in the result, want 6", len(out.Images))
	}
	f, err := snapfmt.ReadImage(d, got, "red", snapfmt.KindTest)
	if err != nil {
		t.Fatalf("ReadImage: %v", err)
	}
	frames, err := localFrames(a.Scene, &meta, core.ChRed, []string{snapfmt.KindTest})
	if err != nil {
		t.Fatalf("localFrames: %v", err)
	}
	want := frames[snapKey{ch: core.ChRed, kind: snapfmt.KindTest}]
	if !bytes.Equal(f.Pix, want.Data) {
		t.Error("blue test image differs from the in-process render")
	}
}

func TestLoadXMLScenarioWithoutCamera(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	dir := t.TempDir()
	meta := dp.snapMeta()
	sc := snapfmt.ScenarioFromMeta(meta)
	if err := snapfmt.WriteScenario(filepath.Join(dir, snapfmt.ScenarioFile), sc); err != nil {
		t.Fatalf("WriteScenario: %v", err)
	}
	meta2, d, err := snapfmt.ReadMeta(dir)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	res, err := snapFetch("", a.Scene, d, meta2, nil, "", core.ChRed, false, a.Scene.W, a.Scene.H)
	if err != nil {
		t.Fatalf("snapFetch: %v", err)
	}
	frames, err := localFrames(a.Scene, &meta, core.ChRed, []string{snapfmt.KindTest})
	if err != nil {
		t.Fatalf("localFrames: %v", err)
	}
	want := frames[snapKey{ch: core.ChRed, kind: snapfmt.KindTest}]
	if res.test == nil || !bytes.Equal(res.test.Data, want.Data) {
		t.Error("test image from the XML scenario differs from the in-process render")
	}
}
