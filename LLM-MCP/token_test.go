package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTokenFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")
	if err := os.WriteFile(path, []byte("  tok-abc123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseConfig([]byte(`{"token_file":"` + path + `","providers":[{"name":"p","base_url":"https://x.test/v1","model":"m"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	tok, err := cfg.ResolveToken()
	if err != nil || tok != "tok-abc123" {
		t.Fatalf("ResolveToken = %q, %v", tok, err)
	}
	if warns := cfg.CheckSecrets(); len(warns) != 1 || !strings.Contains(warns[0], "no API key") {
		// the provider has no key configured; the token file itself is fine
		t.Errorf("warnings = %v", warns)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	warns := cfg.CheckSecrets()
	loose := false
	for _, w := range warns {
		if strings.Contains(w, "token_file") && strings.Contains(w, "chmod 600") {
			loose = true
		}
	}
	if !loose {
		t.Errorf("expected a token_file permission warning, got %v", warns)
	}
	if out := cfg.RedactedJSON(); strings.Contains(out, "tok-abc123") {
		t.Errorf("token leaked into the redacted config:\n%s", out)
	}
}

func TestResolveTokenInlineWins(t *testing.T) {
	cfg, err := ParseConfig([]byte(`{"token":"inline","token_file":"/nonexistent/x","providers":[{"name":"p","base_url":"https://x.test/v1","model":"m"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if tok, err := cfg.ResolveToken(); err != nil || tok != "inline" {
		t.Fatalf("ResolveToken = %q, %v", tok, err)
	}
}

func TestResolveTokenMissingFile(t *testing.T) {
	cfg, err := ParseConfig([]byte(`{"token_file":"/definitely/not/here","providers":[{"name":"p","base_url":"https://x.test/v1","model":"m"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.ResolveToken(); err == nil {
		t.Fatal("expected an error for a missing token file")
	}
}
