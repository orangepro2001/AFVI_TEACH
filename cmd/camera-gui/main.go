// Command camera-gui is the camera studio: the GUI front end of the camera program. It
// processes the original MEDIAN images into synthetic captures (product, light, noise,
// defects) and writes them with camera.exe. The window is a Wails/WebView2 front end
// (frontend/); all logic lives in internal/app.
package main

import (
	"afvi/teach-imgui/frontend"
	"afvi/teach-imgui/internal/app"
)

func main() {
	app.RunStudio(frontend.Dist())
}
