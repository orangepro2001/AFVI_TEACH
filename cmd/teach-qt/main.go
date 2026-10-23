// Command teach-qt is the teaching program with the native Qt Widgets front end
// (internal/qtui); the logic, panels and protocol are the same as teach-imgui.
package main

import (
	"afvi/teach-imgui/internal/app"
	_ "afvi/teach-imgui/internal/qtui"
)

func main() { app.Run(nil) }
