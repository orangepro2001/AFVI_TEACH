//go:build !windows

package app

import (
	"errors"
	"io/fs"

	"afvi/teach-imgui/internal/host"
)

// runWindow: the native window is WebView2 and exists on Windows only; elsewhere use -web.
func runWindow(h *host.Host, a *App, assets fs.FS) error {
	return errors.New("the native window is Windows-only; run with -web 127.0.0.1:47700 and open it in a browser")
}
