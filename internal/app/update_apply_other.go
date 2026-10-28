//go:build !windows

package app

import "errors"

// launchInstaller is Windows-only: the installer is a Windows program (per-user install, silent
// upgrade). On other platforms the update dialog explains that it cannot install and the menu
// entry stays a status report.
func launchInstaller(path string) error {
	return errors.New("installing an update is only supported on Windows")
}
