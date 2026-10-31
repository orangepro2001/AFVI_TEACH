package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfigAndResolve(t *testing.T) {
	cfg, err := ParseConfig([]byte(`{
	  "default": "ds",
	  "providers": [
	    {"name": "ds", "base_url": "https://api.deepseek.com/v1", "model": "deepseek-chat",
	     "models": {"reasoner": "deepseek-reasoner", "fast": "deepseek-chat"}},
	    {"name": "local", "base_url": "http://127.0.0.1:11434/v1", "model": "qwen2.5-coder:7b"}
	  ]
	}`))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	cases := []struct {
		spec, provider, model string
	}{
		{"", "ds", "deepseek-chat"},
		{"ds", "ds", "deepseek-chat"},
		{"local", "local", "qwen2.5-coder:7b"},
		{"ds/reasoner", "ds", "deepseek-reasoner"},
		{"reasoner", "ds", "deepseek-reasoner"},
		{"fast", "ds", "deepseek-chat"},
		{"local/qwen2.5-coder:7b", "local", "qwen2.5-coder:7b"},
		{"deepseek-chat", "ds", "deepseek-chat"},
		{"brand-new-model", "ds", "brand-new-model"},
		{"nosuch/qwen", "ds", "nosuch/qwen"}, // unknown provider -> literal id on the default
	}
	for _, c := range cases {
		tgt, err := cfg.Resolve(c.spec)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", c.spec, err)
		}
		if tgt.Provider.Name != c.provider || tgt.Model != c.model {
			t.Errorf("Resolve(%q) = %s/%s, want %s/%s",
				c.spec, tgt.Provider.Name, tgt.Model, c.provider, c.model)
		}
	}
	if got := cfg.DefaultProvider().Name; got != "ds" {
		t.Errorf("DefaultProvider = %q", got)
	}
}

func TestParseConfigExpandsEnv(t *testing.T) {
	t.Setenv("AFVI_TEST_KEY", "sk-from-env")
	cfg, err := ParseConfig([]byte(`{"providers":[{"name":"p","base_url":"${AFVI_TEST_HOST}","api_key":"${AFVI_TEST_KEY}"}]}`))
	if err != nil {
		// AFVI_TEST_HOST is unset -> empty base_url must fail validation.
		if !strings.Contains(err.Error(), "base_url") {
			t.Fatalf("unexpected error: %v", err)
		}
	} else {
		t.Fatalf("expected base_url validation error, got %+v", cfg)
	}
	t.Setenv("AFVI_TEST_HOST", "https://example.test/v1")
	cfg, err = ParseConfig([]byte(`{"providers":[{"name":"p","base_url":"${AFVI_TEST_HOST}","api_key":"${AFVI_TEST_KEY}"}]}`))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if got := cfg.Providers[0].Key(); got != "sk-from-env" {
		t.Errorf("Key = %q", got)
	}
	if got := cfg.Providers[0].KeySource(); got != "inline api_key" {
		t.Errorf("KeySource = %q", got)
	}
}

func TestKeyFileAndPermissionWarning(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "api-key")
	if err := os.WriteFile(keyPath, []byte("  sk-test-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseConfig([]byte(`{"providers":[{"name":"p","base_url":"https://x.test/v1","api_key_file":"` + keyPath + `"}]}`))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if got := cfg.Providers[0].Key(); got != "sk-test-key" {
		t.Errorf("Key = %q, want trimmed file content", got)
	}
	if got := cfg.Providers[0].KeySource(); !strings.Contains(got, keyPath) {
		t.Errorf("KeySource = %q", got)
	}
	if warns := cfg.CheckSecrets(); len(warns) != 0 {
		t.Errorf("unexpected warnings for 0600 file: %v", warns)
	}

	if err := os.Chmod(keyPath, 0o644); err != nil {
		t.Fatal(err)
	}
	warns := cfg.CheckSecrets()
	if len(warns) != 1 || !strings.Contains(warns[0], "chmod 600") {
		t.Errorf("expected a permission warning, got %v", warns)
	}
}

func TestMissingKeyIsReported(t *testing.T) {
	cfg, err := ParseConfig([]byte(`{"providers":[{"name":"p","base_url":"https://x.test/v1","model":"m"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	warns := cfg.CheckSecrets()
	if len(warns) != 1 || !strings.Contains(warns[0], "no API key") {
		t.Fatalf("warnings = %v", warns)
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	if got, want := expandPath("~/.config/llm-mcp/api-key"), filepath.Join(home, ".config/llm-mcp/api-key"); got != want {
		t.Errorf("expandPath = %q, want %q", got, want)
	}
	if got := expandPath("  /tmp/x  "); got != "/tmp/x" {
		t.Errorf("expandPath = %q", got)
	}
}

func TestLoadConfigSearchOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"providers":[{"name":"p","base_url":"https://x.test/v1","model":"m"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, got, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got != path || cfg == nil {
		t.Fatalf("path = %q, cfg = %v", got, cfg)
	}
	if _, _, err := LoadConfig(filepath.Join(dir, "nope.json")); err == nil {
		t.Fatal("expected an error for an explicit missing path")
	}
}

func TestRedactedJSONHidesKey(t *testing.T) {
	cfg, err := ParseConfig([]byte(`{"token":"super-secret-token",
	  "providers":[{"name":"p","base_url":"https://x.test/v1","api_key":"sk-1234567890abcdef","model":"m"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	out := cfg.RedactedJSON()
	if strings.Contains(out, "sk-1234567890abcdef") || strings.Contains(out, "super-secret-token") {
		t.Fatalf("secret leaked into redacted config:\n%s", out)
	}
	if !strings.Contains(out, "sk-1****ef") {
		t.Errorf("expected a masked key, got:\n%s", out)
	}
}
