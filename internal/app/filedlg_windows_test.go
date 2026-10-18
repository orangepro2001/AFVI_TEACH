//go:build windows

package app

import (
	"testing"
	"unsafe"
)

func TestOpenFileNameLayout(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("OPENFILENAMEW layout is checked on 64-bit only")
	}
	if got := unsafe.Sizeof(openFileName{}); got != 152 {
		t.Errorf("sizeof(openFileName) = %d, want 152", got)
	}
	if got := unsafe.Offsetof(openFileName{}.flags); got != 96 {
		t.Errorf("offsetof(openFileName.flags) = %d, want 96", got)
	}
}
