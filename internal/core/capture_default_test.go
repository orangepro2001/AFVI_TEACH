package core

import (
	"reflect"
	"testing"
)

func TestDefaultCaptureWarp(t *testing.T) {
	p := DefaultCaptureWarp()
	if err := ValidateCaptureWarp(p); err != nil {
		t.Fatalf("ValidateCaptureWarp: %v", err)
	}
	if p.GeometryIdentity() {
		t.Fatalf("the default capture warp must not be the identity")
	}
	want := WarpParams{Tx: 14, Ty: -10, ThetaDeg: 0.3, Scale: 1, Gain: 1}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("DefaultCaptureWarp() = %+v, want %+v", p, want)
	}
}
