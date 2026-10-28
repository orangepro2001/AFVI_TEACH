// Package config stores the machine configuration (afvi.toml): which machines exist,
// their network addresses and the three host data roots (FM1/FM2/BM). It is pure Go
// (no cgo, no third-party code) so it can be tested on any platform.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Machine is one inspection machine: a display name, an optional network address and
// the data root for each host role. Paths may be absolute, UNC or relative to BaseDir.
type Machine struct {
	Name    string
	Address string
	FM1     string
	FM2     string
	BM      string
}

// HostPath returns the configured path for host ("FM1"|"FM2"|"BM"), or "" for an
// unknown host.
func (m Machine) HostPath(host string) string {
	switch host {
	case "FM1":
		return m.FM1
	case "FM2":
		return m.FM2
	case "BM":
		return m.BM
	}
	return ""
}

// Config is the whole afvi.toml document.
type Config struct {
	Active   string // machine name; "" means the first machine
	Theme    string // "light" | "dark" | ""
	Lang     string // "zh" | "en" | "ko" | ""
	Machines []Machine

	// The window is one program with two faces (the teaching program and the camera
	// studio); these say which face it opens with and whether the last choice is kept.
	StartProfile string // "teach" | "studio" | "ask" ("" = ask)
	RememberLast bool   // remember the face the user switched to last
	SplitStart   bool   // also open the other face next to this one at startup

	BaseDir string // not serialized: directory of the loaded file
}

// Default builds the simulated REFERENCE machine pointing at refRoot.
func Default(refRoot string) *Config {
	return &Config{
		Active: "REFERENCE",
		Machines: []Machine{{
			Name:    "REFERENCE",
			Address: "localhost",
			FM1:     filepath.Join(refRoot, "FM1"),
			FM2:     filepath.Join(refRoot, "FM2"),
			BM:      filepath.Join(refRoot, "BM"),
		}},
	}
}

// ActiveIndex returns the index of Active, or 0 when Active is empty or unknown.
func (c *Config) ActiveIndex() int {
	if c == nil {
		return 0
	}
	for i, m := range c.Machines {
		if m.Name == c.Active {
			return i
		}
	}
	return 0
}

// Validate checks that there is at least one machine and that names are non-empty and
// unique (case-insensitive).
func (c *Config) Validate() error {
	if c == nil || len(c.Machines) == 0 {
		return errors.New("no machines configured")
	}
	seen := map[string]bool{}
	for i, m := range c.Machines {
		name := strings.TrimSpace(m.Name)
		if name == "" {
			return fmt.Errorf("machine %d has an empty name", i+1)
		}
		key := strings.ToLower(name)
		if seen[key] {
			return fmt.Errorf("duplicate machine name %q", name)
		}
		seen[key] = true
	}
	return nil
}

// UpsertReference sets or replaces the simulated REFERENCE machine from refRoot and
// makes it active.
func (c *Config) UpsertReference(refRoot string) {
	m := Machine{
		Name:    "REFERENCE",
		Address: "localhost",
		FM1:     filepath.Join(refRoot, "FM1"),
		FM2:     filepath.Join(refRoot, "FM2"),
		BM:      filepath.Join(refRoot, "BM"),
	}
	for i := range c.Machines {
		if strings.EqualFold(c.Machines[i].Name, "REFERENCE") {
			c.Machines[i] = m
			c.Active = m.Name
			return
		}
	}
	c.Machines = append(c.Machines, m)
	c.Active = m.Name
}

func isSlash(b byte) bool { return b == '\\' || b == '/' }

// IsUNC reports whether p is a network path (\\server..., also written //server...). It is
// decided on the text so it behaves the same on every platform: filepath.IsAbs on Windows
// rejects a bare server such as \\192.168.1.61 (no share part), and that path would then be
// joined onto the config directory.
func IsUNC(p string) bool {
	return len(p) > 2 && isSlash(p[0]) && isSlash(p[1]) && !isSlash(p[2])
}

// IsUNCServerOnly reports whether p names only a server (\\host or \\host\), with no share.
// Windows cannot stat such a path (it lists the server's shares, it is not a directory), but
// the shares below it, e.g. \\host\PxRepository, are ordinary directories.
func IsUNCServerOnly(p string) bool {
	if !IsUNC(p) {
		return false
	}
	for i := 2; i < len(p); i++ {
		if isSlash(p[i]) {
			return strings.Trim(p[i:], `\/`) == ""
		}
	}
	return true
}

// ResolvePath resolves a configured path: "" stays "", absolute paths and network paths
// (\\server\share, or just \\server) are unchanged and relative paths are joined with
// BaseDir (or left cwd-relative when BaseDir is empty).
func (c *Config) ResolvePath(p string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) || IsUNC(p) {
		return p
	}
	if c == nil || c.BaseDir == "" {
		return p
	}
	return filepath.Join(c.BaseDir, p)
}

// Resolve finds the configuration file: explicit path, AFVI_CONFIG, afvi.toml next to
// the executable, afvi.toml in the working directory. It returns the first existing
// regular file with found=true; otherwise the exe-adjacent default path (or afvi.toml
// in the working directory when the exe path is unavailable) with found=false. An
// explicit path that does not exist is returned as-is with found=false.
func Resolve(explicit string) (string, bool) {
	if explicit != "" {
		if isRegularFile(explicit) {
			return explicit, true
		}
		return explicit, false
	}
	if env := os.Getenv("AFVI_CONFIG"); env != "" {
		if isRegularFile(env) {
			return env, true
		}
	}
	exePath := ""
	if exe, err := os.Executable(); err == nil {
		exePath = filepath.Join(filepath.Dir(exe), "afvi.toml")
		if isRegularFile(exePath) {
			return exePath, true
		}
	}
	if wd, err := os.Getwd(); err == nil {
		cwdPath := filepath.Join(wd, "afvi.toml")
		if isRegularFile(cwdPath) {
			return cwdPath, true
		}
	}
	if exePath != "" {
		return exePath, false
	}
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, "afvi.toml"), false
	}
	return "afvi.toml", false
}

func isRegularFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}

// Load reads a configuration file, strips a UTF-8 BOM and records BaseDir.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	cfg.BaseDir = filepath.Dir(path)
	return cfg, nil
}

// Save writes the configuration to path: it creates the parent directory, writes a
// temporary file in the same directory and renames it over the target.
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp, err := os.CreateTemp(dir, ".afvi-*.toml")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(c.Marshal()); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// Marshal renders the configuration as deterministic TOML text with a comment header.
func (c *Config) Marshal() []byte {
	var b strings.Builder
	b.WriteString("# AFVI Teach machine configuration.\n")
	b.WriteString("# Paths may be absolute, UNC (\\\\server\\share\\FM1) or relative to this file.\n")
	b.WriteString("# Generated by the application; edit with care.\n\n")
	b.WriteString("active = " + tomlString(c.Active) + "\n")
	b.WriteString("theme = " + tomlString(c.Theme) + "\n")
	b.WriteString("lang = " + tomlString(c.Lang) + "\n")
	if c.StartProfile != "" {
		b.WriteString("start_profile = " + tomlString(c.StartProfile) + "\n")
	}
	if c.RememberLast {
		b.WriteString("remember_last = true\n")
	}
	if c.SplitStart {
		b.WriteString("split_start = true\n")
	}
	for _, m := range c.Machines {
		b.WriteString("\n[[machine]]\n")
		b.WriteString("name = " + tomlString(m.Name) + "\n")
		b.WriteString("address = " + tomlString(m.Address) + "\n")
		b.WriteString("fm1 = " + tomlString(m.FM1) + "\n")
		b.WriteString("fm2 = " + tomlString(m.FM2) + "\n")
		b.WriteString("bm = " + tomlString(m.BM) + "\n")
	}
	return []byte(b.String())
}

// tomlString renders a value as a literal string unless it contains a single quote or a
// control character, in which case a basic (escaped) string is used.
func tomlString(s string) string {
	if !strings.ContainsRune(s, '\'') && !hasControl(s) {
		return "'" + s + "'"
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04X`, r)
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func hasControl(s string) bool {
	for _, r := range s {
		if r < 0x20 {
			return true
		}
	}
	return false
}

// Parse reads the TOML subset used by afvi.toml. Errors carry a "line N: ..." prefix.
func Parse(data []byte) (*Config, error) {
	text := strings.TrimPrefix(string(data), "\ufeff")
	cfg := &Config{}
	lines := strings.Split(text, "\n")
	inMachine := false
	machineIdx := -1
	for i, raw := range lines {
		lineNo := i + 1
		line := strings.TrimSuffix(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[[") {
			if trimmed != "[[machine]]" {
				return nil, fmt.Errorf("line %d: unknown table %q", lineNo, trimmed)
			}
			cfg.Machines = append(cfg.Machines, Machine{})
			machineIdx = len(cfg.Machines) - 1
			inMachine = true
			continue
		}
		eq := indexOutsideString(trimmed)
		if eq < 0 {
			return nil, fmt.Errorf("line %d: expected key = value", lineNo)
		}
		key := strings.ToLower(strings.TrimSpace(trimmed[:eq]))
		valText := strings.TrimSpace(trimmed[eq+1:])
		val, err := parseString(valText)
		if err != nil {
			return nil, fmt.Errorf("line %d: %v", lineNo, err)
		}
		if inMachine {
			m := &cfg.Machines[machineIdx]
			switch key {
			case "name":
				m.Name = val
			case "address":
				m.Address = val
			case "fm1":
				m.FM1 = val
			case "fm2":
				m.FM2 = val
			case "bm":
				m.BM = val
			}
			continue
		}
		switch key {
		case "active":
			cfg.Active = val
		case "theme":
			cfg.Theme = val
		case "lang":
			cfg.Lang = val
		case "start_profile":
			cfg.StartProfile = val
		case "remember_last":
			cfg.RememberLast = val == "true" || val == "1"
		case "split_start":
			cfg.SplitStart = val == "true" || val == "1"
		}
	}
	return cfg, nil
}

// indexOutsideString returns the index of the first '=' that is not inside a quoted
// string, or -1.
func indexOutsideString(s string) int {
	inBasic, inLiteral := false, false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case inBasic:
			if ch == '\\' {
				i++
			} else if ch == '"' {
				inBasic = false
			}
		case inLiteral:
			if ch == '\'' {
				inLiteral = false
			}
		case ch == '"':
			inBasic = true
		case ch == '\'':
			inLiteral = true
		case ch == '=':
			return i
		}
	}
	return -1
}

// parseString parses a basic ("...") or literal ('...') string, ignoring a trailing
// comment outside the quotes.
func parseString(s string) (string, error) {
	if s == "" {
		return "", errors.New("missing value")
	}
	switch s[0] {
	case '\'':
		end := strings.IndexByte(s[1:], '\'')
		if end < 0 {
			return "", errors.New("unterminated literal string")
		}
		return s[1 : 1+end], nil
	case '"':
		var b strings.Builder
		for i := 1; i < len(s); i++ {
			ch := s[i]
			switch ch {
			case '"':
				return b.String(), nil
			case '\\':
				i++
				if i >= len(s) {
					return "", errors.New("unterminated basic string")
				}
				switch s[i] {
				case '\\':
					b.WriteByte('\\')
				case '"':
					b.WriteByte('"')
				case 'n':
					b.WriteByte('\n')
				case 't':
					b.WriteByte('\t')
				case 'r':
					b.WriteByte('\r')
				default:
					b.WriteByte(s[i])
				}
			default:
				b.WriteByte(ch)
			}
		}
		return "", errors.New("unterminated basic string")
	}
	return "", fmt.Errorf("unsupported value %q", s)
}
