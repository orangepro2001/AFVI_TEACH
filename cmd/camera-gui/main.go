// Command camera-gui is the camera studio: the GUI front end of the camera program. It
// processes the original MEDIAN images into synthetic captures (product, light, noise,
// defects) and writes them with camera.exe. All logic lives in internal/app.
package main

import (
	"runtime"

	"afvi/teach-imgui/internal/app"
)

func init() {
	// GLFW and OpenGL require all calls from the main OS thread.
	runtime.LockOSThread()
}

func main() {
	app.RunStudio()
}
