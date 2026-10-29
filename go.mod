module afvi/teach-imgui

go 1.27.3

require (
	github.com/AllenDang/cimgui-go v1.6.1
	golang.org/x/image v0.47.0
)

// The browser build replaces the cgo/C++ Dear ImGui bindings with the pure Go shim in
// ../afvi_web/shim, so internal/app - the panels, unchanged - compiles to js/wasm.
replace github.com/AllenDang/cimgui-go => ../afvi_web/shim
