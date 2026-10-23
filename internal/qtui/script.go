package qtui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"afvi/teach-imgui/internal/ui"
)

// A test script (AFVI_QT_SCRIPT=file.json) drives the running window without a person:
//
//	[{"wait":3000},{"cmd":{"id":"tabs","op":"set","v":1}},{"shot":"/tmp/a.png"},
//	 {"click":"widget-id"}, {"text":"substring that must appear in some label"}]
//
// "cmd" goes through the same host command path as a user action, "click" presses the real
// Qt button whose id the front end registered, "shot" saves the window as PNG, "text" checks
// that a label/button anywhere in the window contains the text (failures are reported on
// stderr and make the process exit with status 3 at the end).
type step struct {
	Wait  int     `json:"wait"`
	Cmd   *ui.Cmd `json:"cmd"`
	Shot  string  `json:"shot"`
	Text  string  `json:"text"`
	Click string  `json:"click"`
	Quit  bool    `json:"quit"`
}

func (w *window) runScript(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "script:", err)
		return
	}
	var steps []step
	if err := json.Unmarshal(b, &steps); err != nil {
		fmt.Fprintln(os.Stderr, "script:", err)
		return
	}
	failed := false
	var next func(i int)
	next = func(i int) {
		if i >= len(steps) {
			if failed {
				os.Exit(3)
			}
			qt.QCoreApplication_Quit()
			return
		}
		s := steps[i]
		delay := s.Wait
		if delay == 0 {
			delay = 150
		}
		t := qt.NewQTimer()
		t.SetSingleShot(true)
		t.OnTimeout(func() {
			switch {
			case s.Cmd != nil:
				w.h.Do(*s.Cmd)
			case s.Click != "":
				if b := w.buttons[s.Click]; b != nil {
					b.Click()
				} else {
					fmt.Fprintf(os.Stderr, "script: no button %q\n", s.Click)
					failed = true
				}
			case s.Shot != "":
				path := strings.ReplaceAll(s.Shot, "@OUT", os.Getenv("AFVI_QT_OUT"))
				if !w.mw.Grab().Save(path) {
					fmt.Fprintf(os.Stderr, "script: cannot save %s\n", s.Shot)
					failed = true
				}
			case s.Text != "":
				if !w.hasText(s.Text) {
					fmt.Fprintf(os.Stderr, "script: text %q not found\n", s.Text)
					failed = true
				}
			case s.Quit:
				qt.QCoreApplication_Quit()
				return
			}
			next(i + 1)
		})
		t.Start(delay)
	}
	next(0)
}

// hasText reports whether any control label recorded while building the window contains s.
func (w *window) hasText(s string) bool {
	for _, sv := range w.slots {
		if sv.reg != nil {
			for _, t := range sv.reg.texts {
				if contains(t, s) {
					return true
				}
			}
		}
	}
	for _, dv := range w.dialogs {
		if dv.reg != nil {
			for _, t := range dv.reg.texts {
				if contains(t, s) {
					return true
				}
			}
		}
	}
	return false
}

func contains(a, b string) bool { return strings.Contains(a, b) }
