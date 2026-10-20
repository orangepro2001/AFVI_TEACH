package core

// DefaultCaptureWarp is the misalignment the simulated camera gives a capture when nothing else is
// asked for: a few pixels of shift and a fraction of a degree of rotation.
func DefaultCaptureWarp() WarpParams {
	return WarpParams{Tx: 14, Ty: -10, ThetaDeg: 0.3, Scale: 1, Gain: 1}
}

// DefaultCaptureLight is the illumination a capture is taken with: red, green and blue
// LEDs on at 512, white off. It differs from DefaultLightSetting (red only).
func DefaultCaptureLight() LightSetting {
	s := DefaultLightSetting()
	s.Channels = []LightChannel{
		{Color: LightRed, Value: 512, Enable: true},
		{Color: LightGreen, Value: 512, Enable: true},
		{Color: LightBlue, Value: 512, Enable: true},
		{Color: LightWhite, Value: 0, Enable: false},
	}
	return s
}
