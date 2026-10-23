package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fullFile = `# AFVI machine configuration
active = "AFVI-01"
theme = "dark"
lang = "en"

[[machine]]
name = "REFERENCE"
address = "localhost"
fm1 = '../reference/FM1'
fm2 = '../reference/FM2'
bm = '../reference/BM'

[[machine]]
name = "AFVI-01"
address = "192.168.10.21"
fm1 = '\\192.168.10.21\AFVI\FM1'
fm2 = '\\192.168.10.21\AFVI\FM2'
bm = '\\192.168.10.21\AFVI\BM'
`

func TestParseFull(t *testing.T) {
	cfg, err := Parse([]byte(fullFile))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Active != "AFVI-01" || cfg.Theme != "dark" || cfg.Lang != "en" {
		t.Fatalf("top-level keys: %+v", cfg)
	}
	if len(cfg.Machines) != 2 {
		t.Fatalf("machines = %d, want 2", len(cfg.Machines))
	}
	if got := cfg.Machines[0].FM1; got != "../reference/FM1" {
		t.Errorf("literal string FM1 = %q", got)
	}
	if got := cfg.Machines[1].FM1; got != `\\192.168.10.21\AFVI\FM1` {
		t.Errorf("UNC FM1 = %q", got)
	}
	if cfg.ActiveIndex() != 1 {
		t.Errorf("ActiveIndex = %d, want 1", cfg.ActiveIndex())
	}
}

func TestParseDefaults(t *testing.T) {
	cfg, err := Parse([]byte("[[machine]]\nname = \"A\"\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Active != "" || cfg.Theme != "" || cfg.Lang != "" {
		t.Errorf("missing keys should stay empty: %+v", cfg)
	}
	if cfg.ActiveIndex() != 0 {
		t.Errorf("ActiveIndex = %d, want 0", cfg.ActiveIndex())
	}
	if cfg.Machines[0].Address != "" || cfg.Machines[0].FM1 != "" {
		t.Errorf("missing machine keys should stay empty: %+v", cfg.Machines[0])
	}
}

func TestParseBOMAndCRLF(t *testing.T) {
	data := "\ufeffactive = \"A\"\r\n[[machine]]\r\nname = \"A\"\r\n"
	cfg, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Active != "A" || len(cfg.Machines) != 1 || cfg.Machines[0].Name != "A" {
		t.Fatalf("BOM/CRLF parse: %+v", cfg)
	}
}

func TestParseHashInsideString(t *testing.T) {
	cfg, err := Parse([]byte("[[machine]]\nname = \"A#B\" # trailing comment\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Machines[0].Name != "A#B" {
		t.Errorf("name = %q, want A#B", cfg.Machines[0].Name)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"unterminated basic", "active = \"oops\n", "line 1"},
		{"unterminated literal", "active = 'oops\n", "line 1"},
		{"missing equals", "active \"x\"\n", "line 1"},
		{"unknown table", "[[bogus]]\n", "line 1"},
		{"error on later line", "active = \"a\"\n\nname = \"x\n", "line 3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.in))
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err, tc.want)
			}
		})
	}
}

func TestParseIgnoresUnknownKeys(t *testing.T) {
	cfg, err := Parse([]byte("bogus = \"x\"\n[[machine]]\nname = \"A\"\nother = \"y\"\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(cfg.Machines) != 1 || cfg.Machines[0].Name != "A" {
		t.Fatalf("unknown keys should be ignored: %+v", cfg)
	}
}

func TestRoundTripNastyStrings(t *testing.T) {
	cfg := &Config{
		Active: "A",
		Theme:  "light",
		Lang:   "zh",
		Machines: []Machine{
			{Name: "A", Address: "localhost", FM1: `D:\FM1`, FM2: `\\server\share\FM2`, BM: `C:\Program Files\AFVI\BM`},
			{Name: "中文机台", Address: "192.168.0.1", FM1: `D:\数据\FM1`, FM2: `it's here`, BM: `say "hi"`},
		},
	}
	data := cfg.Marshal()
	got, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse(Marshal): %v\n%s", err, data)
	}
	if len(got.Machines) != len(cfg.Machines) {
		t.Fatalf("machines = %d, want %d", len(got.Machines), len(cfg.Machines))
	}
	for i := range cfg.Machines {
		if got.Machines[i] != cfg.Machines[i] {
			t.Errorf("machine %d round trip: got %+v want %+v", i, got.Machines[i], cfg.Machines[i])
		}
	}
	if got.Active != cfg.Active || got.Theme != cfg.Theme || got.Lang != cfg.Lang {
		t.Errorf("top-level round trip: got %+v", got)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		cfg  *Config
		ok   bool
	}{
		{"zero machines", &Config{}, false},
		{"empty name", &Config{Machines: []Machine{{Name: "  "}}}, false},
		{"duplicate case", &Config{Machines: []Machine{{Name: "A"}, {Name: "a"}}}, false},
		{"valid", &Config{Machines: []Machine{{Name: "A"}, {Name: "B"}}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.ok && err != nil {
				t.Errorf("Validate = %v, want nil", err)
			}
			if !tc.ok && err == nil {
				t.Errorf("Validate = nil, want error")
			}
		})
	}
}

func TestActiveIndex(t *testing.T) {
	cfg := &Config{Active: "B", Machines: []Machine{{Name: "A"}, {Name: "B"}}}
	if cfg.ActiveIndex() != 1 {
		t.Errorf("ActiveIndex = %d, want 1", cfg.ActiveIndex())
	}
	cfg.Active = "missing"
	if cfg.ActiveIndex() != 0 {
		t.Errorf("unknown Active should fall back to 0")
	}
	cfg.Active = ""
	if cfg.ActiveIndex() != 0 {
		t.Errorf("empty Active should fall back to 0")
	}
}

func TestUpsertReference(t *testing.T) {
	cfg := &Config{Machines: []Machine{{Name: "AFVI-01"}}}
	cfg.UpsertReference("/data/reference")
	if len(cfg.Machines) != 2 {
		t.Fatalf("machines = %d, want 2", len(cfg.Machines))
	}
	if cfg.Active != "REFERENCE" {
		t.Errorf("Active = %q, want REFERENCE", cfg.Active)
	}
	ref := cfg.Machines[1]
	if ref.FM1 != filepath.Join("/data/reference", "FM1") {
		t.Errorf("FM1 = %q", ref.FM1)
	}
	cfg.UpsertReference("/other")
	if len(cfg.Machines) != 2 {
		t.Fatalf("upsert should replace, machines = %d", len(cfg.Machines))
	}
	if cfg.Machines[1].FM2 != filepath.Join("/other", "FM2") {
		t.Errorf("FM2 = %q", cfg.Machines[1].FM2)
	}
}

func TestResolvePath(t *testing.T) {
	cfg := &Config{BaseDir: "/base"}
	if got := cfg.ResolvePath(""); got != "" {
		t.Errorf("empty = %q, want empty", got)
	}
	if got := cfg.ResolvePath("/abs/FM1"); got != "/abs/FM1" {
		t.Errorf("absolute = %q", got)
	}
	if got := cfg.ResolvePath("rel/FM1"); got != filepath.Join("/base", "rel/FM1") {
		t.Errorf("relative = %q", got)
	}
	noBase := &Config{}
	if got := noBase.ResolvePath("rel/FM1"); got != "rel/FM1" {
		t.Errorf("no BaseDir = %q", got)
	}
}

func TestUNCPaths(t *testing.T) {
	cases := []struct {
		p         string
		unc, only bool
	}{
		{`\\192.168.1.61`, true, true},
		{`\\192.168.1.61\`, true, true},
		{`//192.168.1.61`, true, true},
		{`\\192.168.1.61\PxRepository`, true, false},
		{`\\srv\share\sub`, true, false},
		{`\\`, false, false},
		{`\\\x`, false, false},
		{`C:\data`, false, false},
		{`rel\x`, false, false},
		{``, false, false},
	}
	for _, c := range cases {
		if got := IsUNC(c.p); got != c.unc {
			t.Errorf("IsUNC(%q) = %v, want %v", c.p, got, c.unc)
		}
		if got := IsUNCServerOnly(c.p); got != c.only {
			t.Errorf("IsUNCServerOnly(%q) = %v, want %v", c.p, got, c.only)
		}
	}
	// a network path is never joined onto the config directory, even a bare server
	cfg := &Config{BaseDir: "/base"}
	for _, p := range []string{`\\192.168.1.61`, `\\192.168.1.61\PxRepository`, `//srv/share`} {
		if got := cfg.ResolvePath(p); got != p {
			t.Errorf("ResolvePath(%q) = %q, want unchanged", p, got)
		}
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "afvi.toml")
	cfg := &Config{
		Active:   "A",
		Theme:    "dark",
		Lang:     "ko",
		Machines: []Machine{{Name: "A", Address: "localhost", FM1: `D:\FM1`, FM2: `D:\FM2`, BM: `D:\BM`}},
	}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.BaseDir != filepath.Dir(path) {
		t.Errorf("BaseDir = %q, want %q", got.BaseDir, filepath.Dir(path))
	}
	if got.Active != "A" || got.Theme != "dark" || got.Lang != "ko" {
		t.Errorf("top-level: %+v", got)
	}
	if len(got.Machines) != 1 || got.Machines[0].FM1 != `D:\FM1` {
		t.Errorf("machines: %+v", got.Machines)
	}
}

func TestDefault(t *testing.T) {
	cfg := Default("/ref")
	if cfg.Active != "REFERENCE" || len(cfg.Machines) != 1 {
		t.Fatalf("Default: %+v", cfg)
	}
	m := cfg.Machines[0]
	if m.Name != "REFERENCE" || m.Address != "localhost" {
		t.Errorf("machine: %+v", m)
	}
	if m.FM1 != filepath.Join("/ref", "FM1") || m.FM2 != filepath.Join("/ref", "FM2") || m.BM != filepath.Join("/ref", "BM") {
		t.Errorf("paths: %+v", m)
	}
}

func TestResolveExplicitMissing(t *testing.T) {
	path, found := Resolve(filepath.Join(t.TempDir(), "nope.toml"))
	if found {
		t.Errorf("found = true for a missing explicit path")
	}
	if !strings.HasSuffix(path, "nope.toml") {
		t.Errorf("path = %q", path)
	}
}

func TestResolveExplicitExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "afvi.toml")
	if err := os.WriteFile(path, []byte("active = \"A\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, found := Resolve(path)
	if !found || got != path {
		t.Errorf("Resolve = (%q, %v), want (%q, true)", got, found, path)
	}
}

func TestExampleFileParses(t *testing.T) {
	cfg, err := Load("../../configs/afvi.example.toml")
	if err != nil {
		t.Fatalf("Load example: %v", err)
	}
	if len(cfg.Machines) != 3 {
		t.Fatalf("machines = %d, want 3", len(cfg.Machines))
	}
	if cfg.Machines[0].Name != "REFERENCE" {
		t.Errorf("first machine = %q, want REFERENCE", cfg.Machines[0].Name)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate example: %v", err)
	}
}
