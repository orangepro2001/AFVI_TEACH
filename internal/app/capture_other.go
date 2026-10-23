//go:build !windows

package app

import (
	"fmt"
	"os/exec"
	"strings"
)

// captureWindow saves the window with the given title as PNG on X11 (Xvfb in CI): xdotool
// finds the window id, ImageMagick's import grabs it. Both are external tools so the Go
// side needs no X11 bindings.
func captureWindow(title, path string) error {
	out, err := exec.Command("xdotool", "search", "--name", title).Output()
	if err != nil {
		return fmt.Errorf("xdotool search %q: %w", title, err)
	}
	ids := strings.Fields(string(out))
	if len(ids) == 0 {
		return fmt.Errorf("window %q not found", title)
	}
	if b, err := exec.Command("import", "-window", ids[len(ids)-1], path).CombinedOutput(); err != nil {
		return fmt.Errorf("import: %v: %s", err, b)
	}
	return nil
}
