package app

import (
	"bytes"
	"fmt"
	"time"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/inspect"
)

// liveSelftestStep is the last step of a teaching-program self-test that was started with
// -link: it follows the camera studio that is running on that address and checks that the
// image received over the socket is what the published description renders to. The studio
// is expected to be idle on its default scenario (same product as the loaded model).
func liveSelftestStep(s *Selftest) selftestStep {
	return selftestStep{
		name: "live",
		enter: func(a *App) {
			dp := s.panel("defect").(*DefectPanel)
			for i, p := range a.panels {
				if p == Panel(dp) {
					a.switchTab(i)
				}
			}
			// Start from the default parameters: the earlier steps left their own intervals behind,
			// and the first live run learns the intervals from the studio's good sample.
			for i := range dp.inspectParams {
				dp.inspectParams[i] = newInspectParams()
			}
			dp.learned, dp.relearn = [3]bool{}, false
			dp.liveFollow = true
			// The step is blocked here on purpose: the connection takes up to a redial pause.
			deadline := time.Now().Add(20 * time.Second)
			for !dp.snapSrc.live && time.Now().Before(deadline) {
				dp.Tick()
				time.Sleep(20 * time.Millisecond)
			}
		},
		shot: "native-live.png",
		check: func(s *Selftest) (map[string]any, error) {
			dp := s.panel("defect").(*DefectPanel)
			if !dp.snapSrc.live || dp.snapSrc.meta == nil {
				return nil, fmt.Errorf("live: no camera studio followed on %s (start camera-gui.exe with the same -link address)", s.a.live.addr)
			}
			if dp.test == nil || dp.lastReport == nil {
				return nil, fmt.Errorf("live: no test image or report")
			}
			meta := dp.snapSrc.meta
			cr := meta.Recipe.Core()
			light := cr.Light
			_, want := core.RenderDefectPair(s.a.Scene.Light, &light, dp.channel, cr.Scene, cr.Defects, cr.Noise)
			if !bytes.Equal(dp.test.Data, want.Data) {
				return nil, fmt.Errorf("live: the image received from the studio differs from the render of its published description")
			}
			if dp.snapStale() {
				return nil, fmt.Errorf("live: a live source reported stale")
			}
			direct, err := inspect.Inspect(inspectImage(want), inspectRegions(s.a.Scene), dp.inspectParams[dp.channel])
			if err != nil {
				return nil, fmt.Errorf("live: %w", err)
			}
			if len(direct.Defects) != len(dp.lastReport.Defects) {
				return nil, fmt.Errorf("live: found %d defects, inspecting the published description directly finds %d", len(dp.lastReport.Defects), len(direct.Defects))
			}
			return map[string]any{
				"studio":   dp.liveSt.peer.App,
				"revision": dp.snapSrc.liveRev,
				"defects":  len(meta.Recipe.Defects),
				"found":    len(dp.lastReport.Defects),
			}, nil
		},
	}
}
