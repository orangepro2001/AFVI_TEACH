//go:build windows

package app

import (
	"context"
	"io/fs"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"afvi/teach-imgui/internal/host"
)

// runWindow opens the frameless Wails window (WebView2). The front end loads from the
// embedded assets; /cmd, /poll and /img fall through to the host handler, so the same HTTP
// protocol serves the window and the browser test mode.
func runWindow(h *host.Host, a *App, assets fs.FS) error {
	var wctx context.Context
	go func() { // Exit menu / exit-after: close the window from the UI side
		for {
			time.Sleep(100 * time.Millisecond)
			if a.quit && wctx != nil {
				runtime.Quit(wctx)
				return
			}
		}
	}()
	return wails.Run(&options.App{
		Title:            a.profile.WindowTitle(),
		Width:            1600,
		Height:           1000,
		MinWidth:         960,
		MinHeight:        600,
		Frameless:        true,
		BackgroundColour: &options.RGBA{R: 212, G: 208, B: 200, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets, Handler: h.Handler(nil)},
		OnStartup:        func(ctx context.Context) { wctx = ctx },
	})
}
