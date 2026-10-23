// Command studio-qt is the camera studio with the native Qt Widgets front end.
package main

import (
	"afvi/teach-imgui/internal/app"
	_ "afvi/teach-imgui/internal/qtui"
)

func main() { app.RunStudio(nil) }
