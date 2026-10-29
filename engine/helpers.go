package engine

import (
	"bytes"
	"time"
)

// bytesReader keeps the decode call readable and lets the wasm build use the same code.
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }

// nowMillis is the clock the timing fields use.
func nowMillis() float64 { return float64(time.Since(start).Microseconds()) / 1000 }

var start = time.Now()
