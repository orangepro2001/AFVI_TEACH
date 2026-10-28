package app

// Profile selects which of the two programs built on this UI framework is running:
// the teaching program (align + defect detection) or the camera studio (light + defect
// synthesis, a GUI front end of the camera program).
type Profile string

const (
	// ProfileTeach is the teaching program: align tab and defect-detection tab.
	ProfileTeach Profile = "teach"
	// ProfileStudio is the camera studio: light tab and defect-synthesis tab; it writes
	// captures (snap.json plus PNGs) by running the camera program.
	ProfileStudio Profile = "studio"
	// ProfileNone is "no face": the zero value, used for "no switch is pending".
	ProfileNone Profile = ""
)

// Window titles are also used by the self-test to find the window for screenshots.
const (
	TeachWindowTitle  = "Color AFVI Master Teach"
	StudioWindowTitle = "Color AFVI Camera Studio"
)

// WindowTitle returns the OS window title of the profile.
func (p Profile) WindowTitle() string {
	if p == ProfileStudio {
		return StudioWindowTitle
	}
	return TeachWindowTitle
}

// defectMode says what the shared DefectPanel does in a profile.
type defectMode int

const (
	// modeDetect adjusts the inspection parameters and inspects the test image (a camera
	// capture or the built-in default scenario). It has no painting tools and never draws
	// the simulated stroke outlines, which are the answer.
	modeDetect defectMode = iota
	// modeAuthor paints the synthetic defects, product and noise and writes captures. It
	// never runs the inspection.
	modeAuthor
)

// defectMode returns the DefectPanel mode of the profile.
func (p Profile) defectMode() defectMode {
	if p == ProfileStudio {
		return modeAuthor
	}
	return modeDetect
}
