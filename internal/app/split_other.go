//go:build !windows

package app

import "errors"

// spawnPeer has no native tiling call outside Windows; the two windows still open, the user
// arranges them. This keeps the button meaningful for a Linux build (the Xvfb self-test) instead
// of making the feature Windows-only in the UI.
func spawnPeer(p Profile) error {
	return errors.New("side-by-side tiling is only implemented on Windows; start the other program yourself")
}
