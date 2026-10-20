package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"afvi/teach-imgui/internal/model"
	"afvi/teach-imgui/internal/model/modeltest"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// newTestApp builds an App for profile p on the synthetic reference model (96x64) and
// installs the loaded scene like LoadModel does, but without touching the viewer.
func newTestApp(t *testing.T, p Profile) *App {
	t.Helper()
	t.Setenv("AFVI_CONFIG", "")
	t.Setenv("AFVI_CAPTURES", t.TempDir()) // scenario autosave and captures stay out of the test binary's folder
	root := t.TempDir()
	if _, err := modeltest.WriteReference(root); err != nil {
		t.Fatalf("WriteReference: %v", err)
	}
	cfg, cfgPath, err := model.ResolveConfig("", root, "")
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	a := NewApp(cfg, cfgPath, ThemeLight, p)
	if len(a.Models) == 0 {
		t.Fatal("no models")
	}
	scene, err := LoadScene(a.Models[0])
	if err != nil {
		t.Fatalf("LoadScene: %v", err)
	}
	a.modelIdx = 0
	a.Scene = scene
	a.Viewer.noGL = true // no OpenGL context in the headless tests
	a.align.modelLoaded(scene)
	if dp := defectPanelOf(a); dp != nil {
		dp.ModelLoaded()
	}
	return a
}

// pollJobs applies finished jobs until no job is running (15 s limit).
func pollJobs(t *testing.T, a *App) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		a.Jobs.Poll()
		if !a.Jobs.AnyBusy() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("pollJobs: timeout waiting for background jobs")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestProfilePanels(t *testing.T) {
	SetLang(LangZh)

	teach := newTestApp(t, ProfileTeach)
	var teachIDs []string
	for _, p := range teach.panels {
		teachIDs = append(teachIDs, p.ID())
	}
	if strings.Join(teachIDs, ",") != "roi,align,defect" {
		t.Errorf("teach panel ids = %v, want [roi align defect]", teachIDs)
	}
	tdp := defectPanelOf(teach)
	if tdp == nil {
		t.Fatal("teach: no defect panel")
	}
	if tdp.mode != modeDetect {
		t.Errorf("teach defect mode = %v, want modeDetect", tdp.mode)
	}
	if got, want := tdp.Title(), T("tab.defect"); got != want {
		t.Errorf("teach defect title = %q, want %q", got, want)
	}

	studio := newTestApp(t, ProfileStudio)
	var studioIDs []string
	for _, p := range studio.panels {
		studioIDs = append(studioIDs, p.ID())
	}
	if strings.Join(studioIDs, ",") != "light,defect" {
		t.Errorf("studio panel ids = %v, want [light defect]", studioIDs)
	}
	sdp := defectPanelOf(studio)
	if sdp == nil {
		t.Fatal("studio: no defect panel")
	}
	if sdp.mode != modeAuthor {
		t.Errorf("studio defect mode = %v, want modeAuthor", sdp.mode)
	}
	if got, want := sdp.Title(), T("tab.synth"); got != want {
		t.Errorf("studio defect title = %q, want %q", got, want)
	}

	if got := ProfileTeach.WindowTitle(); got != TeachWindowTitle {
		t.Errorf("teach window title = %q, want %q", got, TeachWindowTitle)
	}
	if got := ProfileStudio.WindowTitle(); got != StudioWindowTitle {
		t.Errorf("studio window title = %q, want %q", got, StudioWindowTitle)
	}
	if TeachWindowTitle == StudioWindowTitle {
		t.Error("teach and studio window titles are equal")
	}
}

func TestSelftestStepsPerProfile(t *testing.T) {
	for _, p := range []Profile{ProfileTeach, ProfileStudio} {
		s := NewSelftest(newTestApp(t, p), t.TempDir())
		names := map[string]bool{}
		for _, st := range s.steps {
			names[st.name] = true
		}
		if !names["load"] {
			t.Errorf("%s: missing step \"load\"", p)
		}
		if p == ProfileStudio {
			for name := range studioOnlySteps {
				if !names[name] {
					t.Errorf("%s: missing studio step %q", p, name)
				}
			}
			for name := range teachOnlySteps {
				if names[name] {
					t.Errorf("%s: contains teach step %q", p, name)
				}
			}
		} else {
			for name := range teachOnlySteps {
				if !names[name] {
					t.Errorf("%s: missing teach step %q", p, name)
				}
			}
			for name := range studioOnlySteps {
				if names[name] {
					t.Errorf("%s: contains studio step %q", p, name)
				}
			}
		}
	}
	for name := range studioOnlySteps {
		if teachOnlySteps[name] {
			t.Errorf("step %q is in both studioOnlySteps and teachOnlySteps", name)
		}
	}
}

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

func TestStudioSaveCapture(t *testing.T) {
	if findCameraExe() == "" {
		t.Skip("camera program not found; set AFVI_CAMERA_EXE to run this test")
	}
	a := newTestApp(t, ProfileStudio)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	dp.studio.chInit = true
	dp.studio.channels = [3]bool{true, false, false}
	dp.studio.good = true
	dp.studio.recipe = false
	dp.studio.dir = filepath.Join(t.TempDir(), "cap")
	dp.requestSaveCapture()
	pollJobs(t, a)
	if dp.studio.msgErr {
		t.Fatalf("save capture failed: %s %s", dp.studio.msgKey, dp.studio.msgText)
	}
	if dp.studio.last == nil {
		t.Fatal("studio.last is nil after a successful save")
	}
	meta, dir, err := snapfmt.ReadMeta(dp.studio.dir)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	// camera.exe renders the images, so it stamps its own producer on an image capture.
	if !strings.HasPrefix(meta.Producer, "afvi-camera") {
		t.Errorf("producer = %q, want the afvi-camera prefix", meta.Producer)
	}
	if meta.Width != a.Scene.W || meta.Height != a.Scene.H {
		t.Errorf("size = %dx%d, want %dx%d", meta.Width, meta.Height, a.Scene.W, a.Scene.H)
	}
	if len(meta.Recipe.Defects) != len(dp.gvStrokes()) {
		t.Errorf("%d recipe defects, want %d", len(meta.Recipe.Defects), len(dp.gvStrokes()))
	}
	for _, kind := range []string{snapfmt.KindTest, snapfmt.KindGood} {
		f, err := snapfmt.ReadImage(dir, meta, "red", kind)
		if err != nil {
			t.Fatalf("ReadImage red %s: %v", kind, err)
		}
		if f.Width != a.Scene.W || f.Height != a.Scene.H {
			t.Errorf("red %s image %dx%d, want %dx%d", kind, f.Width, f.Height, a.Scene.W, a.Scene.H)
		}
	}
	if _, ok := meta.FindImage("green", snapfmt.KindTest); ok {
		t.Error("green test image was written, want none")
	}

	dp.studio.recipe = true
	dp.studio.dir = filepath.Join(t.TempDir(), "rec")
	dp.requestSaveCapture()
	pollJobs(t, a)
	if dp.studio.msgErr {
		t.Fatalf("save recipe failed: %s %s", dp.studio.msgKey, dp.studio.msgText)
	}
	recPath := filepath.Join(dp.studio.dir, snapfmt.MetaFile+".gz")
	if _, err := os.Stat(recPath); err != nil {
		t.Fatalf("recipe file missing: %v", err)
	}
	recMeta, _, err := snapfmt.ReadMeta(recPath)
	if err != nil {
		t.Fatalf("ReadMeta recipe: %v", err)
	}
	if recMeta.Producer != "afvi-camera-studio 1" {
		t.Errorf("recipe producer = %q, want afvi-camera-studio 1", recMeta.Producer)
	}
	entries, err := os.ReadDir(dp.studio.dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	// The description and the scenario file that carries every studio setting.
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if want := []string{snapfmt.ScenarioFile, snapfmt.MetaFile + ".gz"}; !reflect.DeepEqual(names, want) {
		t.Errorf("recipe folder holds %v, want %v", names, want)
	}
}

func TestDefaultCaptureDir(t *testing.T) {
	if got := defaultCaptureDir(nil); !strings.HasSuffix(got, filepath.Join("captures", "capture")) {
		t.Errorf("defaultCaptureDir(nil) = %q, want suffix %q", got, filepath.Join("captures", "capture"))
	}
	scene := &Scene{}
	scene.Info.ID = "FM1/abc"
	if got := defaultCaptureDir(scene); !strings.HasSuffix(got, filepath.Join("captures", "FM1-abc")) {
		t.Errorf("defaultCaptureDir(scene) = %q, want suffix %q", got, filepath.Join("captures", "FM1-abc"))
	}
}

func TestScale(t *testing.T) {
	saved := fontPx
	t.Cleanup(func() { SetFontPx(saved) })

	SetFontPx(26)
	if got := sc(13); got != 26 {
		t.Errorf("sc(13) with fontPx 26 = %v, want 26", got)
	}
	SetFontPx(100)
	if fontPx != maxFontPx {
		t.Errorf("fontPx after SetFontPx(100) = %v, want %v", fontPx, maxFontPx)
	}
	SetFontPx(1)
	if fontPx != minFontPx {
		t.Errorf("fontPx after SetFontPx(1) = %v, want %v", fontPx, minFontPx)
	}
}
