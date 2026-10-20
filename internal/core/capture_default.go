package core

// DefaultCaptureWarp is the misalignment the simulated camera gives a capture when nothing else is
// asked for: a few pixels of shift and a fraction of a degree of rotation.
func DefaultCaptureWarp() WarpParams {
	return WarpParams{Tx: 14, Ty: -10, ThetaDeg: 0.3, Scale: 1, Gain: 1}
}
