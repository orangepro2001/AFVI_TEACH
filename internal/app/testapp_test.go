package app

import (
	"testing"
	"time"

	"afvi/teach-imgui/internal/model"
	"afvi/teach-imgui/internal/model/modeltest"
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
	a.align.modelLoaded(scene)
	a.golden.modelLoaded(scene)
	if dp := defectPanelOf(a); dp != nil {
		dp.ModelLoaded()
	}
	if a.golden != nil {
		pollJobs(t, a) // the golden board builds its regions
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
