package inspect

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func sampleParamsFile() *ParamsFile {
	p := DefaultParams()
	p.EdgeMargin = 6
	p.BackgroundRadius = 2
	p.Materials[Metal].Low, p.Materials[Metal].High = 90, 170
	p.Materials[Metal].Size, p.Materials[Metal].MinArea = 5, 9
	p.Materials[SR].Low, p.Materials[SR].High = 20.5, 33.25
	p.Materials[SR].LocalBackground = false
	p.Materials[Open].Enabled = true
	p.Materials[Open].Low, p.Materials[Open].High = 0, 12
	other := p
	other.EdgeMargin = 3
	return &ParamsFile{
		Producer: "test",
		Model:    "FM1/abc",
		Width:    5860,
		Height:   4020,
		LearnK:   4.5,
		Channels: map[string]ChannelParams{
			"red":   {Learned: true, Params: p},
			"green": {Learned: false, Params: other},
		},
	}
}

func TestParamsFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", ParamsFileName)
	in := sampleParamsFile()
	if err := WriteParamsFile(path, in); err != nil {
		t.Fatalf("WriteParamsFile: %v", err)
	}
	out, err := ReadParamsFile(path)
	if err != nil {
		t.Fatalf("ReadParamsFile: %v", err)
	}
	if out.Format != ParamsFormat || out.Version != ParamsVersion || out.Saved == "" {
		t.Fatalf("header = %q v%d saved %q", out.Format, out.Version, out.Saved)
	}
	if out.Model != in.Model || out.Width != in.Width || out.Height != in.Height || out.LearnK != in.LearnK {
		t.Fatalf("meta fields differ: %+v", out)
	}
	if !reflect.DeepEqual(out.Channels, in.Channels) {
		t.Fatalf("channels differ:\n got %+v\nwant %+v", out.Channels, in.Channels)
	}
	// A directory path resolves to inspect.json for both reading and writing.
	dir := filepath.Dir(path)
	if got := ResolveParamsPath(dir); got != path {
		t.Fatalf("ResolveParamsPath(dir) = %q, want %q", got, path)
	}
	if _, err := ReadParamsFile(dir); err != nil {
		t.Fatalf("ReadParamsFile(dir): %v", err)
	}
	in.LearnK = 7
	if err := WriteParamsFile(dir, in); err != nil {
		t.Fatalf("WriteParamsFile(dir): %v", err)
	}
	again, err := ReadParamsFile(path)
	if err != nil || again.LearnK != 7 {
		t.Fatalf("rewrite: learnK %v err %v", again, err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("%d files after rewriting, want only %s", len(entries), ParamsFileName)
	}
}

func TestParamsFileNamedMaterials(t *testing.T) {
	path := filepath.Join(t.TempDir(), ParamsFileName)
	if err := WriteParamsFile(path, sampleParamsFile()); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	for _, want := range []string{`"base"`, `"sr"`, `"open"`, `"metal"`, `"edgeMargin"`, `"backgroundRadius"`, `"localBackground"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("file lacks %s", want)
		}
	}
}

func TestParamsFileTolerantReading(t *testing.T) {
	path := filepath.Join(t.TempDir(), ParamsFileName)
	// Only one material and one channel, plus an unknown field: the rest keeps the library defaults.
	text := `{"format":"afvi-inspect","version":1,"future":true,"channels":{"blue":{"learned":true,"params":{"materials":{"metal":{"enabled":true,"low":10,"high":20,"size":2,"minArea":3,"localBackground":false}},"edgeMargin":4}}}}`
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := ReadParamsFile(path)
	if err != nil {
		t.Fatalf("ReadParamsFile: %v", err)
	}
	p := f.Channels["blue"].Params
	def := DefaultParams()
	if p.EdgeMargin != 4 || p.BackgroundRadius != def.BackgroundRadius {
		t.Fatalf("edge %d background %d", p.EdgeMargin, p.BackgroundRadius)
	}
	if m := p.Materials[Metal]; m.Low != 10 || m.High != 20 || m.Size != 2 || m.MinArea != 3 || m.LocalBackground {
		t.Fatalf("metal = %+v", m)
	}
	if p.Materials[SR] != def.Materials[SR] {
		t.Fatalf("absent material changed: %+v", p.Materials[SR])
	}
}

func TestParamsFileTileRulesRoundTrip(t *testing.T) {
	p := DefaultParams()
	p.TileSize = 256
	p.TileSmooth = true
	p.Materials[Metal].IntervalOff = true
	p.Materials[Metal].Abs = TileRule{Enabled: true, Bright: 12.5, Dark: 7, BrightSize: 4, DarkSize: 5}
	p.Materials[Metal].Pct = TileRule{Enabled: true, Bright: 8, Dark: 9.5, BrightSize: 6, DarkSize: 7}
	in := &ParamsFile{
		Channels: map[string]ChannelParams{
			"red": {Learned: true, Params: p},
		},
	}
	path := filepath.Join(t.TempDir(), ParamsFileName)
	if err := WriteParamsFile(path, in); err != nil {
		t.Fatalf("WriteParamsFile: %v", err)
	}
	out, err := ReadParamsFile(path)
	if err != nil {
		t.Fatalf("ReadParamsFile: %v", err)
	}
	got := out.Channels["red"].Params
	if !reflect.DeepEqual(got, p) {
		t.Fatalf("params differ:\n got %+v\nwant %+v", got, p)
	}
	if got.Materials[SR] != DefaultParams().Materials[SR] {
		t.Fatalf("SR changed: %+v", got.Materials[SR])
	}
}

func TestParamsFileOldFileDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), ParamsFileName)
	text := `{"format":"afvi-inspect","version":1,"channels":{"red":{"learned":true,"params":{"materials":{"metal":{"enabled":true,"low":10,"high":20,"size":2,"minArea":3,"localBackground":false}},"edgeMargin":4}}}}`
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := ReadParamsFile(path)
	if err != nil {
		t.Fatalf("ReadParamsFile: %v", err)
	}
	p := f.Channels["red"].Params
	m := p.Materials[Metal]
	if m.Low != 10 || m.High != 20 || m.Size != 2 || m.MinArea != 3 || m.LocalBackground {
		t.Fatalf("metal old fields = %+v", m)
	}
	if m.IntervalOff {
		t.Error("IntervalOff should default to false")
	}
	if m.Abs != defaultAbsRule() {
		t.Errorf("Abs = %+v, want %+v", m.Abs, defaultAbsRule())
	}
	if m.Pct != defaultPctRule() {
		t.Errorf("Pct = %+v, want %+v", m.Pct, defaultPctRule())
	}
	if p.TileSize != DefaultTileSize {
		t.Errorf("TileSize = %d, want %d", p.TileSize, DefaultTileSize)
	}
	if p.TileSmooth {
		t.Error("TileSmooth should default to false")
	}
}

func TestParamsFilePartialTileRule(t *testing.T) {
	path := filepath.Join(t.TempDir(), ParamsFileName)
	text := `{"format":"afvi-inspect","version":1,"channels":{"red":{"params":{"materials":{"metal":{"enabled":true,"low":0,"high":255,"size":1,"minArea":1,"abs":{"enabled":true}}}}}}}`
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := ReadParamsFile(path)
	if err != nil {
		t.Fatalf("ReadParamsFile: %v", err)
	}
	abs := f.Channels["red"].Params.Materials[Metal].Abs
	def := defaultAbsRule()
	if !abs.Enabled {
		t.Fatal("Abs.Enabled should be true")
	}
	if abs.Bright != def.Bright || abs.Dark != def.Dark || abs.BrightSize != def.BrightSize || abs.DarkSize != def.DarkSize {
		t.Fatalf("Abs = %+v, want defaults %+v with Enabled=true", abs, def)
	}
}

func TestParamsFileTileRuleValidation(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	cases := map[string]string{
		"absBright": `{"format":"afvi-inspect","version":1,"channels":{"red":{"params":{"materials":{"metal":{"enabled":true,"low":0,"high":255,"size":1,"minArea":1,"abs":{"enabled":true,"bright":256,"dark":1,"brightSize":1,"darkSize":1}}}}}}}`,
		"pctDark":   `{"format":"afvi-inspect","version":1,"channels":{"red":{"params":{"materials":{"metal":{"enabled":true,"low":0,"high":255,"size":1,"minArea":1,"pct":{"enabled":true,"bright":1,"dark":1001,"brightSize":1,"darkSize":1}}}}}}}`,
		"absSize":   `{"format":"afvi-inspect","version":1,"channels":{"red":{"params":{"materials":{"metal":{"enabled":true,"low":0,"high":255,"size":1,"minArea":1,"abs":{"enabled":true,"bright":1,"dark":1,"brightSize":0,"darkSize":1}}}}}}}`,
		"tileSmall": `{"format":"afvi-inspect","version":1,"channels":{"red":{"params":{"tileSize":15}}}}`,
		"tileBig":   `{"format":"afvi-inspect","version":1,"channels":{"red":{"params":{"tileSize":4097}}}}`,
	}
	for name, text := range cases {
		if _, err := ReadParamsFile(write(name+".json", text)); err == nil {
			t.Errorf("%s: bad file accepted", name)
		}
	}
	// 关闭的规则不校验其字段。
	ok := `{"format":"afvi-inspect","version":1,"channels":{"red":{"params":{"materials":{"metal":{"enabled":true,"low":0,"high":255,"size":1,"minArea":1,"abs":{"enabled":false,"bright":-5,"dark":1,"brightSize":0,"darkSize":1}}}}}}}`
	if _, err := ReadParamsFile(write("disabled.json", ok)); err != nil {
		t.Errorf("disabled rule rejected: %v", err)
	}
	// WriteParamsFile 同样拒绝非法值。
	bad := DefaultParams()
	bad.Materials[Metal].Abs = TileRule{Enabled: true, Bright: 256, Dark: 1, BrightSize: 1, DarkSize: 1}
	if err := WriteParamsFile(filepath.Join(dir, "out.json"), &ParamsFile{Channels: map[string]ChannelParams{"red": {Params: bad}}}); err == nil {
		t.Error("WriteParamsFile wrote invalid tile rule")
	}
}

func TestParamsFileTileRuleKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), ParamsFileName)
	if err := WriteParamsFile(path, sampleParamsFile()); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	for _, want := range []string{`"intervalOff"`, `"abs"`, `"pct"`, `"tileSize"`, `"tileSmooth"`, `"brightSize"`, `"darkSize"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("file lacks %s", want)
		}
	}
}

func TestParamsFileRejectsBadFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	cases := map[string]string{
		"format":  `{"format":"other","version":1,"channels":{}}`,
		"version": `{"format":"afvi-inspect","version":2,"channels":{}}`,
		"range":   `{"format":"afvi-inspect","version":1,"channels":{"red":{"params":{"materials":{"metal":{"enabled":true,"low":200,"high":100,"size":1,"minArea":1}}}}}}`,
		"margin":  `{"format":"afvi-inspect","version":1,"channels":{"red":{"params":{"edgeMargin":999}}}}`,
		"json":    `{not json`,
	}
	for name, text := range cases {
		if _, err := ReadParamsFile(write(name+".json", text)); err == nil {
			t.Errorf("%s: bad file accepted", name)
		}
	}
	if _, err := ReadParamsFile(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("missing file accepted")
	}
	bad := sampleParamsFile()
	bad.Channels["red"] = ChannelParams{Params: Params{EdgeMargin: -1}}
	if err := WriteParamsFile(filepath.Join(dir, "out.json"), bad); err == nil {
		t.Error("WriteParamsFile wrote invalid parameters")
	}
}
