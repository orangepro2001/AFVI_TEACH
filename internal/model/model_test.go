package model_test

import (
	"os"
	"path/filepath"
	"testing"

	"afvi/teach-imgui/internal/config"
	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/model"
	"afvi/teach-imgui/internal/model/modeltest"
)

func TestScanHostFindsSyntheticModel(t *testing.T) {
	root := t.TempDir()
	ref, err := modeltest.WriteReference(root)
	if err != nil {
		t.Fatalf("WriteReference: %v", err)
	}
	models := model.ScanHost("FM1", filepath.Join(root, "FM1"))
	if len(models) != 1 {
		t.Fatalf("ScanHost returned %d models, want 1", len(models))
	}
	m := models[0]
	if m.ID != ref.ID {
		t.Errorf("ID = %q, want %q", m.ID, ref.ID)
	}
	if m.Side != "TOP" {
		t.Errorf("Side = %q, want TOP", m.Side)
	}
	if m.Version != "2.0" {
		t.Errorf("Version = %q, want 2.0", m.Version)
	}
	for _, p := range []string{m.Metal, m.SR, m.Open, m.Median} {
		if st, err := os.Stat(p); err != nil || st.IsDir() {
			t.Errorf("path %q not a regular file: %v", p, err)
		}
	}
}

func TestLoadSceneSynthetic(t *testing.T) {
	root := t.TempDir()
	ref, err := modeltest.WriteReference(root)
	if err != nil {
		t.Fatalf("WriteReference: %v", err)
	}
	models := model.ScanHost("FM1", filepath.Join(root, "FM1"))
	if len(models) != 1 {
		t.Fatalf("ScanHost returned %d models, want 1", len(models))
	}
	scene, err := model.LoadScene(models[0])
	if err != nil {
		t.Fatalf("LoadScene: %v", err)
	}
	if scene.W != ref.Width || scene.H != ref.Height {
		t.Errorf("size = %dx%d, want %dx%d", scene.W, scene.H, ref.Width, ref.Height)
	}
	if len(scene.Material) != ref.Width*ref.Height {
		t.Errorf("len(Material) = %d, want %d", len(scene.Material), ref.Width*ref.Height)
	}
	var counts [4]int
	for _, m := range scene.Material {
		counts[m]++
	}
	for _, m := range []uint8{core.MatBase, core.MatSR, core.MatOpen, core.MatMetal} {
		if counts[m] == 0 {
			t.Errorf("material %d has no pixels", m)
		}
	}
	if scene.Product != core.DefaultProductOptions() {
		t.Errorf("Product = %+v, want default", scene.Product)
	}
	if scene.Light == nil {
		t.Fatal("Light is nil")
	}
	if scene.Light.Width != ref.Width {
		t.Errorf("Light.Width = %d, want %d", scene.Light.Width, ref.Width)
	}
	mean := scene.Stats[core.MatSR].Mean
	if mean < 85 || mean > 95 {
		t.Errorf("Stats[MatSR].Mean = %v, want between 85 and 95", mean)
	}
}

func TestLoadSceneMissingFile(t *testing.T) {
	root := t.TempDir()
	if _, err := modeltest.WriteReference(root); err != nil {
		t.Fatalf("WriteReference: %v", err)
	}
	models := model.ScanHost("FM1", filepath.Join(root, "FM1"))
	if len(models) != 1 {
		t.Fatalf("ScanHost returned %d models, want 1", len(models))
	}
	if err := os.Remove(models[0].SR); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	_, err := model.LoadScene(models[0])
	if err == nil {
		t.Fatal("LoadScene succeeded, want error")
	}
	if !contains(err.Error(), "SR.tif") {
		t.Errorf("error %q does not name SR.tif", err.Error())
	}
}

func TestFindModel(t *testing.T) {
	models := []model.ModelInfo{{ID: "FM1/synthetic-1"}}
	if m, ok := model.FindModel(models, "fm1/SYNTHETIC-1"); !ok || m.ID != "FM1/synthetic-1" {
		t.Errorf("FindModel hit = (%+v, %v), want the model", m, ok)
	}
	if _, ok := model.FindModel(models, "FM1/other"); ok {
		t.Error("FindModel miss returned ok=true")
	}
}

func TestResolveConfigWithRef(t *testing.T) {
	t.Setenv("AFVI_CONFIG", "")
	root := t.TempDir()
	if _, err := modeltest.WriteReference(root); err != nil {
		t.Fatalf("WriteReference: %v", err)
	}
	cfg, _, err := model.ResolveConfig("", root, "")
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if cfg.Active != "REFERENCE" {
		t.Errorf("Active = %q, want REFERENCE", cfg.Active)
	}
	m := cfg.Machines[cfg.ActiveIndex()]
	found := model.ScanMachine(cfg, m)
	if len(found) != 1 || found[0].ID != "FM1/synthetic-1" {
		t.Errorf("ScanMachine = %+v, want the synthetic model", found)
	}
}

func TestResolveConfigMachineFlag(t *testing.T) {
	t.Setenv("AFVI_CONFIG", "")
	root := t.TempDir()
	if _, err := modeltest.WriteReference(root); err != nil {
		t.Fatalf("WriteReference: %v", err)
	}
	fm1 := filepath.Join(root, "FM1")
	text := "active = 'ALPHA'\n" +
		"\n[[machine]]\n" +
		"name = 'ALPHA'\n" +
		"address = 'localhost'\n" +
		"fm1 = '" + fm1 + "'\n" +
		"fm2 = ''\n" +
		"bm = ''\n" +
		"\n[[machine]]\n" +
		"name = 'BETA'\n" +
		"address = 'localhost'\n" +
		"fm1 = '" + fm1 + "'\n" +
		"fm2 = ''\n" +
		"bm = ''\n"
	if _, err := config.Parse([]byte(text)); err != nil {
		t.Fatalf("config.Parse: %v", err)
	}
	path := filepath.Join(t.TempDir(), "afvi.toml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfg, _, err := model.ResolveConfig(path, "", "beta")
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if cfg.Active != "BETA" {
		t.Errorf("Active = %q, want BETA", cfg.Active)
	}
}

func TestParseAlignSpec(t *testing.T) {
	xml := `<Root><Unit>` +
		`<Param Type="0" ID="3" Left="10" Top="20" Right="50" Bottom="60"/>` +
		`<Param Type="1" ID="4" Left="0" Top="0" Right="10" Bottom="10"/>` +
		`</Unit></Root>`
	rois := model.ParseAlignSpec(xml)
	if len(rois) != 1 {
		t.Fatalf("ParseAlignSpec returned %d ROIs, want 1", len(rois))
	}
	r := rois[0]
	if r.Name != "P3" {
		t.Errorf("Name = %q, want P3", r.Name)
	}
	if r.Rect.X != 10 || r.Rect.Y != 20 || r.Rect.W != 40 || r.Rect.H != 40 {
		t.Errorf("Rect = %+v, want {10 20 40 40}", r.Rect)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestCheckHostPathBareServer(t *testing.T) {
	// \\host has no share part, Windows cannot stat it; it must not be reported as a
	// missing path (only the PxRepository share below it can be missing)
	_, err := model.CheckHostPath(&config.Config{BaseDir: t.TempDir()}, "FM1", `\\192.0.2.1`)
	if err != model.ErrNoRepository {
		t.Errorf("err = %v, want ErrNoRepository", err)
	}
}
