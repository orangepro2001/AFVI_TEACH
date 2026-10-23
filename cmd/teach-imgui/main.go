// Command teach-imgui is the AFVI teaching program (operator pages and the admin
// interface). The window is a Wails/WebView2 front end (frontend/, win32-vue components);
// all logic lives in internal/app.
package main

import (
	"afvi/teach-imgui/frontend"
	"afvi/teach-imgui/internal/app"
)

func main() {
	app.Run(frontend.Dist())
}
