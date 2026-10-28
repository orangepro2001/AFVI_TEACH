//go:build !windows

package app

import "errors"

// portableInstall is Windows-only: the green distribution is a folder of Windows programs, and the
// swap is done by a batch file after the program closes.
func portableInstall(zipPath string) error {
	return errors.New("updating a green distribution is only supported on Windows")
}
