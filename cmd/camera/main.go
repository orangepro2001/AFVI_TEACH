// Command camera is the simulated camera of the AFVI teaching tools: it turns a perfect MEDIAN image into a flawed product capture (images plus a description file) that the teach app reads as a snap.
package main

import (
	"os"

	"afvi/teach-imgui/internal/camera"
)

func main() {
	os.Exit(camera.Main(os.Args[1:], os.Stdout, os.Stderr))
}
