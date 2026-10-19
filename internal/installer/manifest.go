// Package installer is the platform independent part of the setup program: the payload that is
// appended to the setup executable, extraction and removal of the installed files, the install
// log that the uninstaller works from, PATH string editing and the installer's texts. Everything
// that talks to Windows (window, registry, shortcuts) lives in cmd/setup. Standard library only.
package installer

import (
	"io/fs"
	"os"
	"sort"
	"strings"
)

// ManifestFormat is the value of Manifest.Format.
const ManifestFormat = "afvi-setup"

// manifestName is the zip entry that holds the manifest; it is never extracted.
const manifestName = "manifest.json"

// LogName is the install log, written to the install folder; the uninstaller works from it.
const LogName = "uninstall.json"

// UninstallerName is the copy of the setup program that is placed in the install folder.
const UninstallerName = "uninstall.exe"

// Manifest describes what a setup executable installs. It is stored as manifest.json at the root
// of the zip payload; every other entry of the payload is a file of the installation.
type Manifest struct {
	Format     string     `json:"format"`
	Product    string     `json:"product"`
	Version    string     `json:"version"`
	Publisher  string     `json:"publisher"`
	Folder     string     `json:"folder"`               // default install folder name and Start Menu folder
	Key        string     `json:"key"`                  // name of the key under ...\Uninstall
	Main       string     `json:"main"`                 // program started after the installation, icon of the entry
	StubSize   int64      `json:"stubSize"`             // size of the setup program in front of the payload (what uninstall.exe is made of)
	UserData   []string   `json:"userData"`             // created by the programs, removed on uninstall only on request
	Shortcuts  []Shortcut `json:"shortcuts"`            // Start Menu entries; the ones marked Desktop are offered on the desktop too
	Components []Text     `json:"components,omitempty"` // what is installed, one line each, shown on the first page
	Notes      []Text     `json:"notes,omitempty"`      // shown on the last page
}

// validRel reports whether name is a safe slash-separated path below the install folder: no
// "..", not absolute, no backslash (a separator on Windows) and no drive letter or stream colon.
func validRel(name string) bool {
	return fs.ValidPath(name) && !strings.ContainsAny(name, "\\:\x00")
}

// Text is a string by language code (zh, en, ko).
type Text map[string]string

// For returns the text in lang, falling back to English and then to any text.
func (t Text) For(lang string) string {
	if s := t[lang]; s != "" {
		return s
	}
	if s := t["en"]; s != "" {
		return s
	}
	keys := make([]string, 0, len(t))
	for k := range t {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		return t[k]
	}
	return ""
}

// Shortcut is one Start Menu (and optionally desktop) entry.
type Shortcut struct {
	Title   Text   `json:"title"`
	Target  string `json:"target"`            // relative to the install folder, or a placeholder such as {comspec}
	Args    string `json:"args,omitempty"`    // may use the placeholders
	WorkDir string `json:"workDir,omitempty"` // empty = the install folder
	Icon    string `json:"icon,omitempty"`    // relative to the install folder; empty = Target
	Desktop bool   `json:"desktop,omitempty"`
}

// Expand replaces the placeholders {dir} (the install folder), {comspec} (the command
// interpreter) and {home} (the user profile) in s.
func Expand(s, dir string) string {
	comspec := os.Getenv("ComSpec")
	if comspec == "" {
		comspec = "cmd.exe"
	}
	return strings.NewReplacer("{dir}", dir, "{comspec}", comspec, "{home}", os.Getenv("USERPROFILE")).Replace(s)
}
