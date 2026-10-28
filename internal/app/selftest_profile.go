package app

import "fmt"

// profileSteps proves the single-entry promise: one window, two faces, switching in place with
// no restart and nothing thrown away. It only runs in the teaching program (it needs the
// teaching face to return to).
func (s *Selftest) profileSteps() []selftestStep {
	var teachPages, studioPages int
	var goldenPtr *goldenBoard
	var alignFlow *AlignFlow
	return []selftestStep{
		{
			name: "profile-switch",
			enter: func(a *App) {
				teachPages = len(a.allPanels)
				goldenPtr, alignFlow = a.golden, a.align
				if !a.SwitchProfile(ProfileStudio) {
					return
				}
				studioPages = len(a.allPanels)
			},
			shot: "native-profile-studio.png",
			check: func(s *Selftest) (map[string]any, error) {
				a := s.a
				if a.profile != ProfileStudio {
					return nil, fmt.Errorf("switching to the studio failed (still %q)", a.profile)
				}
				if studioPages == 0 || studioPages == teachPages {
					return nil, fmt.Errorf("studio page set looks wrong: %d teach, %d studio", teachPages, studioPages)
				}
				if a.panelByID("light") == nil {
					return nil, fmt.Errorf("the studio's light page is missing")
				}
				if a.panelByID("defect") == nil {
					return nil, fmt.Errorf("the studio's defect page is missing")
				}
				// and back: same page objects, so the user's work is still there
				defect := a.panelByID("defect")
				if !a.SwitchProfile(ProfileTeach) {
					return nil, fmt.Errorf("switching back to the teaching program failed")
				}
				if a.profile != ProfileTeach {
					return nil, fmt.Errorf("still %q after switching back", a.profile)
				}
				if len(a.allPanels) != teachPages {
					return nil, fmt.Errorf("%d pages after switching back, want %d", len(a.allPanels), teachPages)
				}
				if a.panelByID("align") == nil || a.panelByID("golden") == nil {
					return nil, fmt.Errorf("the teaching pages did not come back")
				}
				if a.golden != goldenPtr || a.align != alignFlow {
					return nil, fmt.Errorf("the golden board / alignment flow were rebuilt, the user's work is gone")
				}
				// the studio's defect page is the same object it was, not a fresh one
				if again := a.sets[ProfileStudio]; again == nil {
					return nil, fmt.Errorf("the studio page set was dropped")
				} else if again.allPanels[0] == nil || panelIndex(again.allPanels, "defect") != indexOfPanel(a.sets[ProfileStudio].allPanels, defect) {
					return nil, fmt.Errorf("the studio page objects changed")
				}
				return map[string]any{
					"teachPages":  teachPages,
					"studioPages": studioPages,
					"keptGolden":  a.golden == goldenPtr,
					"keptAlign":   a.align == alignFlow,
				}, nil
			},
		},
	}
}

// panelIndex returns the position of the page with id in pages, or -1.
func panelIndex(pages []Panel, id string) int {
	for i, p := range pages {
		if p.ID() == id {
			return i
		}
	}
	return -1
}

// indexOfPanel returns the position of p in pages, or -1.
func indexOfPanel(pages []Panel, p Panel) int {
	for i, q := range pages {
		if q == p {
			return i
		}
	}
	return -1
}
