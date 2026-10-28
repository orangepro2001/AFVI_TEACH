package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPortableAt: the marker file is the whole difference between the installed and the green
// distribution, so it must be read from the program's own folder and nowhere else.
func TestPortableAt(t *testing.T) {
	dir := t.TempDir()
	if portableAt(dir) {
		t.Fatal("a folder without portable.txt must not look portable")
	}
	if err := os.WriteFile(filepath.Join(dir, portableMarker), []byte("green\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !portableAt(dir) {
		t.Fatalf("a folder holding %s must be portable", portableMarker)
	}
	if portableAt(filepath.Join(dir, "nope")) {
		t.Fatal("a folder that does not exist must not be portable")
	}
}

// TestPortablePlatform: the green build asks the update service under its own platform name, so
// the two distributions never offer each other's package.
func TestPortablePlatform(t *testing.T) {
	if portablePlatform != "win-x64-portable" {
		t.Fatalf("portablePlatform = %q, want win-x64-portable", portablePlatform)
	}
	if portablePlatform == updatePlat {
		t.Fatal("the green build must not share the installer's platform name")
	}
}

// TestSwapScript: the batch file is what actually replaces the files while the program is closed,
// so its commands are pinned here: copy (not mirror) with retries, restart, delete itself.
func TestSwapScript(t *testing.T) {
	stage := `C:\AFVI\updates\staged`
	dir := `C:\Program Files\AFVI Teach` // a space on purpose: the quoting has to hold
	exe := dir + `\teach-imgui.exe`
	s := swapScript(stage, dir, exe)

	for _, want := range []string{
		`robocopy "` + stage + `" "` + dir + `" /e`,
		`if %ERRORLEVEL% LSS 8 goto done`,
		`if %tries% GEQ 60 goto done`,
		`rmdir /s /q "` + stage + `"`,
		`start "" "` + exe + `"`,
		`del "%~f0"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("swap script is missing %q\n%s", want, s)
		}
	}
	if strings.Contains(s, "/mir") {
		t.Error("a mirror would delete the user's afvi.toml and captures; copy instead")
	}
	if !strings.HasSuffix(s, "\r\n") || strings.Count(s, "\r\n") != strings.Count(s, "\n") {
		t.Error("a batch file needs CRLF line endings throughout")
	}
}
