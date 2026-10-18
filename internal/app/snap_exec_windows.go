package app

import (
	"os/exec"
	"syscall"
)

// hideConsole keeps the camera program from flashing a console window.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
