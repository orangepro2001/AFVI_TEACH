package app

import (
	"os"
)

// "Side by side" opens the other face as a window of its own and lets Windows tile the two: the
// two faces are separate processes, exactly like the two old programs, but the user asks for it
// once from inside the window instead of hunting for the other exe. The live link between them
// keeps working, because one of them listens and the other dials, as before.
//
// Nothing here is required for the single-entry use: a user who never presses the button gets one
// window that switches faces in place.

// peerProfile is the face that a side-by-side window opens next to this one.
func (a *App) peerProfile() Profile {
	if a.profile == ProfileStudio {
		return ProfileTeach
	}
	return ProfileStudio
}

// SplitWithPeer opens the other face next to this window and tiles both. It returns an error
// string for the status bar ("" on success). Packages that cannot run two windows (a self-test,
// or a platform without the native tiling call) explain that instead of failing silently.
func (a *App) SplitWithPeer() {
	if a.selftest != nil {
		a.SetHint(T("profile.split.selftest"))
		return
	}
	if err := spawnPeer(a.peerProfile()); err != nil {
		a.SetHint(T("profile.split.failed", "msg", err.Error()))
		return
	}
	a.SetHint(T("profile.split.started", "profile", T("profile."+string(a.peerProfile()))))
}

// peerArgs is the command line of the window opened by SplitWithPeer: the same executable, the
// other face, told not to tile again (that would fight over the two windows).
func peerArgs(p Profile) []string {
	return []string{"-profile", string(p), "-split-peer"}
}

// SiblingExecutable reports the executable a side-by-side window runs: this one. A portable or
// installed copy is one program with two faces, so no second binary is needed.
func siblingExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return exe, nil
}
