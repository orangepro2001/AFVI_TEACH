//go:build !windows

package app

// chromeMinimise has no portable implementation; the caption button does nothing here.
func chromeMinimise(title string) {}

// chromeWorkArea is unknown off Windows; maximise is a no-op.
func chromeWorkArea() (x, y, w, h int, ok bool) { return 0, 0, 0, 0, false }
