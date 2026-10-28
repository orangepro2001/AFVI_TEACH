//go:build windows

package app

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"syscall"
)

// launchInstaller runs the downloaded installer in the silent mode that upgrades in place and
// then starts the new program. The directory of the running executable is passed explicitly:
// the user may have installed somewhere other than the default, and an upgrade must land where
// this program lives.
//
// The child is detached (its own process group, no console) so it survives our exit: it has to
// replace the files of the program that started it.
func launchInstaller(path string) error {
	exe, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(exe)
	cmd := exec.Command(exe, "-silent", "-dir", dir, "-launch")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x00000008 | 0x00000200, // DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", filepath.Base(exe), err)
	}
	// the installer outlives us on purpose; do not wait for it
	go func() { _ = cmd.Wait() }()
	return nil
}
