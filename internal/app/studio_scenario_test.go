package app

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"afvi/teach-imgui/internal/core"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// scenMutate changes the studio state so that it differs clearly from the defaults:
// the green light is enabled and bright, the sensor noise and the misalignment are set,
// one disabled GV stroke and one scene stroke are added, and the brush radius and the
// random count are changed.
func scenMutate(t *testing.T, a *App, p *DefectPanel) {
	t.Helper()
	if len(a.Light.Channels) < 2 {
		t.Fatalf("light has %d channels, want at least 2", len(a.Light.Channels))
	}
	a.Light.Channels[1].Enable = true
	a.Light.Channels[1].Value = 300
	p.noise = core.SensorNoise{ReadSigma: 3, ShotGain: 0.02, Seed: 9}
	p.warp.Tx = 2.5
	p.warp.ThetaDeg = 0.4
	gv := core.DefectStroke{
		Kind:      core.Bright,
		Shape:     core.ShapeSpot,
		X:         30,
		Y:         20,
		Radius:    5,
		DeltaGv:   40,
		Softness:  0.3,
		Amplitude: core.AmpAbsolute,
		Profile:   core.ProfileLinear,
		Seed:      1,
	}
	sc := core.SceneStroke{
		Mode:     core.SceneMaterial,
		Shape:    core.ShapeSpot,
		X:        60,
		Y:        40,
		Radius:   4,
		Material: core.MatMetal,
	}
	p.strokes = append(p.strokes,
		strokeEntry{gv: &gv, enabled: false},
		strokeEntry{scene: &sc, enabled: true},
	)
	p.radius = 17
	p.randomCount = 5
	p.renderRev++
}

// TestScenarioSnapshotCoversState checks that the snapshot holds every stroke (disabled
// ones included) and the studio settings, while the recipe only holds the enabled strokes.
func TestScenarioSnapshotCoversState(t *testing.T) {
	a := newTestApp(t, ProfileStudio)
	p := defectPanelOf(a)
	if p == nil {
		t.Fatal("no defect panel")
	}
	scenMutate(t, a, p)
	s := p.scenarioSnapshot()
	if len(s.Strokes) != len(p.strokes) {
		t.Fatalf("%d snapshot strokes, want %d", len(s.Strokes), len(p.strokes))
	}
	for i, e := range p.strokes {
		if s.Strokes[i].Enabled != e.enabled {
			t.Errorf("stroke %d: Enabled = %v, want %v", i, s.Strokes[i].Enabled, e.enabled)
		}
		if s.Strokes[i].Auto != e.auto {
			t.Errorf("stroke %d: Auto = %v, want %v", i, s.Strokes[i].Auto, e.auto)
		}
	}
	if s.Studio == nil {
		t.Fatal("Studio is nil")
	}
	if s.Studio.Brush.Radius != 17 {
		t.Errorf("brush radius = %v, want 17", s.Studio.Brush.Radius)
	}
	if len(s.Recipe.Light.Channels) < 2 || !s.Recipe.Light.Channels[1].Enable {
		t.Errorf("green channel not enabled in the recipe: %+v", s.Recipe.Light.Channels)
	}
	meta := s.Meta()
	if got, want := len(meta.Recipe.Defects), len(p.gvStrokes()); got != want {
		t.Errorf("%d recipe defects, want %d (disabled strokes must not enter the recipe)", got, want)
	}
}

// TestScenarioSaveLoadRoundTrip saves the mutated state and loads it into a fresh app;
// the snapshots must be byte-identical (plain XML and gzip).
func TestScenarioSaveLoadRoundTrip(t *testing.T) {
	for _, name := range []string{"s.xml", "s.xml.gz"} {
		t.Run(name, func(t *testing.T) {
			app1 := newTestApp(t, ProfileStudio)
			p1 := defectPanelOf(app1)
			if p1 == nil {
				t.Fatal("app1: no defect panel")
			}
			scenMutate(t, app1, p1)
			path := filepath.Join(t.TempDir(), name)
			if err := p1.saveScenario(path); err != nil {
				t.Fatalf("saveScenario: %v", err)
			}

			app2 := newTestApp(t, ProfileStudio)
			p2 := defectPanelOf(app2)
			if p2 == nil {
				t.Fatal("app2: no defect panel")
			}
			key, text, isErr := p2.loadScenario(path)
			if key != "studio.scen.loaded" || isErr {
				t.Fatalf("loadScenario = %q, %q, %v; want studio.scen.loaded, false", key, text, isErr)
			}
			pollJobs(t, app2)

			if !reflect.DeepEqual(app2.Light, app1.Light) {
				t.Errorf("light differs after the round trip:\n got %+v\nwant %+v", app2.Light, app1.Light)
			}
			if p2.noise != p1.noise {
				t.Errorf("noise = %+v, want %+v", p2.noise, p1.noise)
			}
			if !reflect.DeepEqual(p2.warp, p1.warp) {
				t.Errorf("warp = %+v, want %+v", p2.warp, p1.warp)
			}
			if len(p2.strokes) != len(p1.strokes) {
				t.Fatalf("%d strokes after the round trip, want %d", len(p2.strokes), len(p1.strokes))
			}
			for i := range p1.strokes {
				if p2.strokes[i].enabled != p1.strokes[i].enabled {
					t.Errorf("stroke %d: enabled = %v, want %v", i, p2.strokes[i].enabled, p1.strokes[i].enabled)
				}
			}

			s1 := p1.scenarioSnapshot()
			s2 := p2.scenarioSnapshot()
			// The created stamp is adopted from the file, but pin it anyway so the
			// comparison only covers the settings.
			s1.Created = "2026-01-01T00:00:00Z"
			s2.Created = "2026-01-01T00:00:00Z"
			// Each test app has its own temporary reference tree, so the layer paths differ.
			s2.Model = s1.Model
			d1, err := snapfmt.MarshalScenario(s1)
			if err != nil {
				t.Fatalf("MarshalScenario app1: %v", err)
			}
			d2, err := snapfmt.MarshalScenario(s2)
			if err != nil {
				t.Fatalf("MarshalScenario app2: %v", err)
			}
			if !bytes.Equal(d1, d2) {
				t.Errorf("snapshots differ after the round trip:\n--- app1 ---\n%s\n--- app2 ---\n%s", d1, d2)
			}
		})
	}
}

// TestScenarioMismatchRejected checks that a scenario of another model or of another
// image size is rejected and leaves the panel untouched.
func TestScenarioMismatchRejected(t *testing.T) {
	a := newTestApp(t, ProfileStudio)
	p := defectPanelOf(a)
	if p == nil {
		t.Fatal("no defect panel")
	}
	scenMutate(t, a, p)
	strokes := len(p.strokes)
	light := a.Light.Clone()

	cases := []struct {
		name   string
		mutate func(s *snapfmt.Scenario)
	}{
		{"model", func(s *snapfmt.Scenario) { s.Model.ID = "FM1/other" }},
		{"size", func(s *snapfmt.Scenario) { s.Width++ }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := p.scenarioSnapshot()
			c.mutate(s)
			path := filepath.Join(t.TempDir(), "s.xml")
			if err := snapfmt.WriteScenario(path, s); err != nil {
				t.Fatalf("WriteScenario: %v", err)
			}
			key, _, isErr := p.loadScenario(path)
			if key != "studio.scen.mismatch" || !isErr {
				t.Fatalf("loadScenario = %q, %v; want studio.scen.mismatch, true", key, isErr)
			}
			if len(p.strokes) != strokes {
				t.Errorf("%d strokes after the rejected load, want %d", len(p.strokes), strokes)
			}
			if !reflect.DeepEqual(a.Light, light) {
				t.Error("light changed after the rejected load")
			}
		})
	}
}

// TestScenarioRestoreOnModelLoaded checks that ModelLoaded restores the remembered
// scenario when "remember" is on and leaves the defaults alone when it is off.
func TestScenarioRestoreOnModelLoaded(t *testing.T) {
	dir := t.TempDir()

	app1 := newTestApp(t, ProfileStudio)
	p1 := defectPanelOf(app1)
	if p1 == nil {
		t.Fatal("app1: no defect panel")
	}
	t.Setenv("AFVI_CAPTURES", dir)
	p1.studio.scen = scenarioState{}
	p1.ensureScenarioDefaults()
	if !p1.studio.scen.remember {
		t.Fatal("remember = false, want true")
	}
	if want := defaultCaptureDir(app1.Scene); filepath.Dir(p1.studio.scen.path) != want {
		t.Fatalf("scenario path = %q, want it under %q", p1.studio.scen.path, want)
	}
	scenMutate(t, app1, p1)
	if err := p1.saveScenario(p1.studio.scen.path); err != nil {
		t.Fatalf("saveScenario: %v", err)
	}
	wantLight := app1.Light.Clone()
	wantNoise := p1.noise

	t.Run("remember", func(t *testing.T) {
		app2 := newTestApp(t, ProfileStudio)
		p2 := defectPanelOf(app2)
		if p2 == nil {
			t.Fatal("app2: no defect panel")
		}
		t.Setenv("AFVI_CAPTURES", dir)
		p2.studio.scen = scenarioState{}
		p2.ModelLoaded()
		pollJobs(t, app2)
		if p2.studio.scen.msgKey != "studio.scen.restored" {
			t.Fatalf("msgKey = %q, want studio.scen.restored", p2.studio.scen.msgKey)
		}
		if !p2.studio.scen.armed {
			t.Error("armed = false after ModelLoaded")
		}
		if !reflect.DeepEqual(app2.Light, wantLight) {
			t.Errorf("light was not restored:\n got %+v\nwant %+v", app2.Light, wantLight)
		}
		if p2.noise != wantNoise {
			t.Errorf("noise = %+v, want %+v", p2.noise, wantNoise)
		}
		found := false
		for _, e := range p2.strokes {
			if e.gv != nil && !e.enabled && e.gv.X == 30 && e.gv.Y == 20 {
				found = true
			}
		}
		if !found {
			t.Error("the disabled GV stroke was not restored")
		}
	})

	t.Run("noRemember", func(t *testing.T) {
		app3 := newTestApp(t, ProfileStudio)
		p3 := defectPanelOf(app3)
		if p3 == nil {
			t.Fatal("app3: no defect panel")
		}
		t.Setenv("AFVI_CAPTURES", dir)
		p3.studio.scen = scenarioState{init: true, remember: false, pathAuto: true}
		p3.ModelLoaded()
		pollJobs(t, app3)
		if p3.studio.scen.msgKey == "studio.scen.restored" {
			t.Error("the scenario was restored although remember is off")
		}
		if p3.noise != (core.SensorNoise{Seed: 1}) {
			t.Errorf("noise = %+v, want the default", p3.noise)
		}
	})
}

// TestScenarioAutosave checks that tickScenario writes only when the settings changed,
// that remember=false stops the autosave, and that scenarioClose writes a last time.
func TestScenarioAutosave(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, ProfileStudio)
	p := defectPanelOf(a)
	if p == nil {
		t.Fatal("no defect panel")
	}
	t.Setenv("AFVI_CAPTURES", dir)
	p.studio.scen = scenarioState{}
	p.ModelLoaded()
	st := &p.studio.scen
	if !st.armed || !st.remember {
		t.Fatalf("armed = %v, remember = %v; want both true", st.armed, st.remember)
	}
	if _, err := os.Stat(st.path); err == nil {
		t.Fatalf("scenario file %q exists before the first autosave", st.path)
	}

	st.lastTick = time.Time{}
	p.tickScenario()
	if _, err := os.Stat(st.path); err == nil {
		t.Fatalf("scenario file %q was written although nothing changed", st.path)
	}

	p.noise.ReadSigma = 4
	st.lastTick = time.Time{}
	p.tickScenario()
	if _, err := os.Stat(st.path); err != nil {
		t.Fatalf("scenario file was not written: %v", err)
	}
	saved, err := snapfmt.ReadScenario(st.path)
	if err != nil {
		t.Fatalf("ReadScenario: %v", err)
	}
	if saved.Recipe.Noise.ReadSigma != 4 {
		t.Errorf("saved read sigma = %v, want 4", saved.Recipe.Noise.ReadSigma)
	}
	if st.msgKey != "studio.scen.autosaved" {
		t.Errorf("msgKey = %q, want studio.scen.autosaved", st.msgKey)
	}

	p.noise.ReadSigma = 5
	st.remember = false
	st.lastTick = time.Time{}
	p.tickScenario()
	saved, err = snapfmt.ReadScenario(st.path)
	if err != nil {
		t.Fatalf("ReadScenario: %v", err)
	}
	if saved.Recipe.Noise.ReadSigma != 4 {
		t.Errorf("saved read sigma = %v after remember=false, want 4", saved.Recipe.Noise.ReadSigma)
	}

	st.remember = true
	p.scenarioClose()
	saved, err = snapfmt.ReadScenario(st.path)
	if err != nil {
		t.Fatalf("ReadScenario: %v", err)
	}
	if saved.Recipe.Noise.ReadSigma != 5 {
		t.Errorf("saved read sigma = %v after scenarioClose, want 5", saved.Recipe.Noise.ReadSigma)
	}
}

// TestScenarioTeachReadsXMLWithoutCamera checks that the teach program renders the
// studio's scenario.xml in-process and gets the same pixels as the studio itself.
func TestScenarioTeachReadsXMLWithoutCamera(t *testing.T) {
	dir := t.TempDir()
	app1 := newTestApp(t, ProfileStudio)
	p1 := defectPanelOf(app1)
	if p1 == nil {
		t.Fatal("studio: no defect panel")
	}
	scenMutate(t, app1, p1)
	p1.studio.scen.path = filepath.Join(dir, snapfmt.ScenarioFile)
	if err := p1.saveScenario(p1.studio.scen.path); err != nil {
		t.Fatalf("saveScenario: %v", err)
	}

	teach := newTestApp(t, ProfileTeach)
	meta, d, err := snapfmt.ReadMeta(dir)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	res, err := snapFetch("", teach.Scene, d, meta, nil, "", core.ChRed, true, teach.Scene.W, teach.Scene.H)
	if err != nil {
		t.Fatalf("snapFetch: %v", err)
	}
	if res.test == nil || res.good == nil {
		t.Fatal("test or good image is nil")
	}

	light := app1.Light.Clone()
	_, wantTest := core.RenderDefectPairWarped(app1.Scene.Light, &light, core.ChRed, p1.sceneStrokes(), p1.gvStrokes(), p1.noise, p1.warp)
	wantGood := core.RenderGoodSample(app1.Scene.Light, &light, core.ChRed, goodSampleNoise(p1.noise))
	if !bytes.Equal(res.test.Data, wantTest.Data) {
		t.Error("test image from the XML scenario differs from the studio's render")
	}
	if !bytes.Equal(res.good.Data, wantGood.Data) {
		t.Error("good image from the XML scenario differs from the studio's render")
	}
}

// TestScenarioAutosaveRetriesAfterAFailedWrite checks that a write error does not make the
// autosave forget the change: the next tick writes it once the path works again.
func TestScenarioAutosaveRetriesAfterAFailedWrite(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, ProfileStudio)
	t.Setenv("AFVI_CAPTURES", dir)
	p := defectPanelOf(a)
	if p == nil {
		t.Fatal("no defect panel")
	}
	p.studio.scen = scenarioState{}
	p.ModelLoaded()
	st := &p.studio.scen
	if !st.armed || !st.remember {
		t.Fatalf("armed=%v remember=%v, want both true", st.armed, st.remember)
	}
	good := st.path

	// A regular file where the folder should be makes every write fail.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	st.path = filepath.Join(blocker, snapfmt.ScenarioFile)
	p.noise.ReadSigma = 6
	st.lastTick = time.Time{}
	p.tickScenario()
	if st.msgKey != "studio.scen.error" || !st.msgErr {
		t.Fatalf("message after the failed write = %q (err %v), want studio.scen.error", st.msgKey, st.msgErr)
	}

	st.path = good
	st.lastTick = time.Time{}
	p.tickScenario()
	if st.msgKey != "studio.scen.autosaved" {
		t.Fatalf("message after the retry = %q, want studio.scen.autosaved", st.msgKey)
	}
	s, err := snapfmt.ReadScenario(good)
	if err != nil {
		t.Fatalf("ReadScenario: %v", err)
	}
	if s.Recipe.Noise.ReadSigma != 6 {
		t.Errorf("saved read sigma = %v, want 6", s.Recipe.Noise.ReadSigma)
	}
}
