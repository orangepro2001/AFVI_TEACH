//go:build !windows

package app

import "os/exec"

// hideConsole is a no-op on platforms without a console window.
func hideConsole(cmd *exec.Cmd) {}
