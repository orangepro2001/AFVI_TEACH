// Command teach-imgui is the native AFVI teaching app (Dear ImGui front end over the
// Go port of teach-core). All logic lives in internal/app.
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
	app.Run()
}
