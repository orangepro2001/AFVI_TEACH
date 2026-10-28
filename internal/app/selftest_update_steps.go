package app

import (
	"fmt"
	"net/http/httptest"

	"afvi/teach-imgui/internal/updater"
)

// updateSteps exercises the update dialog end to end against a stand-in server: the check, the
// signed manifest, the download and its checksum, and the "ready to install" state. It never
// presses install (that would start a real installer) and it puts the update state back to off
// afterwards, so the steps that follow see the same dialog-less screen as before.
func (s *Selftest) updateSteps() []selftestStep {
	var srv *httptest.Server
	var want *updater.Update
	var setupErr error
	return []selftestStep{
		{
			name: "update-check",
			enter: func(a *App) {
				srv, want, setupErr = updateSelftestInstall(a)
				if setupErr == nil {
					a.startUpdateCheck(false) // serves the offer, then the download starts
				}
			},
			shot: "native-update.png",
			check: func(s *Selftest) (map[string]any, error) {
				a := s.a
				defer func() {
					a.update = updateState{off: true, offWhy: "update.off.flag"}
					if srv != nil {
						srv.Close()
					}
					srv, want = nil, nil
				}()
				if setupErr != nil {
					return nil, fmt.Errorf("stand-in update server: %w", setupErr)
				}
				detail, err := updateSelftestCheck(a, want)
				if err != nil {
					return detail, err
				}
				// the dialog is open at this point; close it so the next step can click again
				a.update.closeReq = true
				return detail, nil
			},
		},
	}
}
