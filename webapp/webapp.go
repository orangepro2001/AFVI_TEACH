//go:build js && wasm

// Package webapp is the public door to the teaching program for the browser build.
//
// internal/app is internal to this module (Go forbids importing it from elsewhere), so the web
// host - a separate module - cannot call it directly. This package is the door: it starts the
// program with a browser-backed window and exposes the two things the host needs, the frame loop and
// the display list.
//
// There is no logic here: app.Run's desktop path parses flags and loads a model from disk, which a
// browser has neither of, so BootForWeb builds the same App around the demo plate instead. Everything
// the user sees and every number the inspection produces still comes from internal/app.
package webapp

import (
	"afvi/teach-imgui/internal/app"
	"github.com/AllenDang/cimgui-go/backend"
	"github.com/AllenDang/cimgui-go/backend/glfwbackend"
)

// Start boots the program on a browser-backed window; the host calls it once.
func Start() {
	be := glfwbackend.NewGLFWBackend()
	backend.SetTextureManager(be)
	app.BootForWeb(be)
}

// Loop returns the program's per-frame function (nil before Start).
func Loop() func() { return backend.Loop() }
