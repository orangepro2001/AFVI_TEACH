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

func TestDefaultCaptureLight(t *testing.T) {
	s := DefaultCaptureLight()
	want := []LightChannel{
		{Color: LightRed, Value: 512, Enable: true},
		{Color: LightGreen, Value: 512, Enable: true},
		{Color: LightBlue, Value: 512, Enable: true},
		{Color: LightWhite, Value: 0, Enable: false},
	}
	if !reflect.DeepEqual(s.Channels, want) {
		t.Fatalf("channels = %+v, want %+v", s.Channels, want)
	}
	d := DefaultLightSetting()
	if !reflect.DeepEqual(s.Reflectance, d.Reflectance) {
		t.Errorf("reflectance differs from DefaultLightSetting")
	}
	if s.Gain != d.Gain || s.Offset != d.Offset || s.TextureStrength != d.TextureStrength {
		t.Errorf("gain/offset/texture = %v/%v/%v, want %v/%v/%v",
			s.Gain, s.Offset, s.TextureStrength, d.Gain, d.Offset, d.TextureStrength)
	}
	if s.Crosstalk != nil || d.Crosstalk != nil {
		t.Errorf("crosstalk = %v, want nil", s.Crosstalk)
	}
}

func TestDefaultCaptureLightIsIndependent(t *testing.T) {
	first := DefaultCaptureLight()
	first.Channels[0].Enable = false
	first.Channels[0].Value = 0
	second := DefaultCaptureLight()
	if !second.Channels[0].Enable || second.Channels[0].Value != 512 {
		t.Fatalf("second call shares Channels: %+v", second.Channels)
	}
}
