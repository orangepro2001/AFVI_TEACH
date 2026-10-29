//go:build js && wasm

package app

import "math"

// mathSin / mathCos are thin wrappers so boot_web.go stays readable.
func mathSin(v float64) float64 { return math.Sin(v) }
func mathCos(v float64) float64 { return math.Cos(v) }
