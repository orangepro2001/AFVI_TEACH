//go:build !windows

package app

// pickImageFile has no native dialog on this platform.
func pickImageFile(string) (string, error) {
	return "", errNoFileDialog
}
