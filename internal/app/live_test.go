package app

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/inspect"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// startLinkedApps builds a studio and a teaching program on the synthetic model and joins
// them over a loopback socket (the studio listens on a free port, the teaching program
// dials it).
func startLinkedApps(t *testing.T) (studio, teach *App, sdp, tdp *DefectPanel) {
	t.Helper()
	studio = newTestApp(t, ProfileStudio)
	teach = newTestApp(t, ProfileTeach)
	legacyDetect(teach)
	studio.StartLive("127.0.0.1:0")
	if studio.live == nil || studio.live.srv == nil {
		t.Fatalf("studio link did not start: %+v", studio.live)
	}
	teach.StartLive(studio.live.srv.Addr().String())
	t.Cleanup(func() {
		teach.closeLive()
		studio.closeLive()
	})
	sdp, tdp = defectPanelOf(studio), defectPanelOf(teach)
	if sdp == nil || tdp == nil {
		t.Fatal("missing defect panel")
	}
	return studio, teach, sdp, tdp
}

// tickUntil calls tick every 10 ms until cond holds (10 s limit).
func tickUntil(t *testing.T, what string, tick func(), cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		tick()
		if cond() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestLiveLinkStudioToTeach(t *testing.T) {
	SetLang(LangZh)
	studio, teach, sdp, tdp := startLinkedApps(t)
	srv := studio.live.srv
	strokesBefore := len(tdp.strokes)

	tickUntil(t, "the studio to publish", sdp.Tick, func() bool { return srv.Rev() >= 1 })
	tickUntil(t, "the teach program to adopt the scene", tdp.Tick, func() bool {
		return tdp.snapSrc.live && tdp.snapSrc.liveRev == srv.Rev()
	})
	if !tdp.snapSrc.on || tdp.snapSrc.meta == nil {
		t.Fatal("live source is not on")
	}
	if tdp.snapSrc.meta.Producer != studioProducer {
		t.Errorf("live meta producer = %q, want %q", tdp.snapSrc.meta.Producer, studioProducer)
	}
	if tdp.snapStale() {
		t.Error("a live source is never stale")
	}

	// The teaching program shows what the studio renders, and inspects it.
	sdp.requestRun()
	pollJobs(t, studio)
	tdp.requestRun()
	pollJobs(t, teach)
	if tdp.test == nil || tdp.lastReport == nil {
		t.Fatal("teach has no live test image or report")
	}
	if !bytes.Equal(tdp.test.Data, sdp.test.Data) {
		t.Fatal("live test image differs from the studio's own render")
	}
	if !tdp.learned[tdp.channel] {
		t.Error("the first live run did not learn the intervals from the studio's good sample")
	}
	if tdp.snapSrc.frames[snapKey{ch: core.ChRed, kind: snapfmt.KindGood}] == nil {
		t.Error("the good sample was not fetched for learning")
	}
	if len(tdp.strokes) != strokesBefore {
		t.Errorf("teach strokes changed from %d to %d: only pixels may cross the link", strokesBefore, len(tdp.strokes))
	}

	// Draw a defect in the studio; the teaching program follows without any file.
	x, y := sdp.findSrPoint()
	w := teach.Scene.W
	was := tdp.test.Data[y*w+x]
	rev := srv.Rev()
	sdp.strokes = append(sdp.strokes, strokeEntry{gv: &core.DefectStroke{
		Kind: core.Bright, Shape: core.ShapeSpot, X: float64(x), Y: float64(y), Radius: 3,
		DeltaGv: 60, Softness: 0.3, Amplitude: core.AmpAbsolute, Profile: core.ProfileLinear, Seed: 99,
	}, enabled: true})
	sdp.renderRev++
	tickUntil(t, "the studio to publish the stroke", sdp.Tick, func() bool { return srv.Rev() > rev })
	tickUntil(t, "the teach program to adopt the new revision", tdp.Tick, func() bool { return tdp.snapSrc.liveRev == srv.Rev() })
	tdp.requestRun()
	pollJobs(t, teach)
	if now := tdp.test.Data[y*w+x]; now < was+20 && now != 255 {
		t.Errorf("pixel (%d,%d) is %d after the stroke, was %d", x, y, now, was)
	}
	sdp.requestRun()
	pollJobs(t, studio)
	if !bytes.Equal(tdp.test.Data, sdp.test.Data) {
		t.Fatal("live test image differs from the studio's render after the stroke")
	}
	want, err := inspect.Inspect(inspectImage(sdp.test), inspectRegions(teach.Scene), tdp.inspectParams[tdp.channel])
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if len(want.Defects) != len(tdp.lastReport.Defects) {
		t.Errorf("teach found %d defects, inspecting the studio image directly finds %d", len(tdp.lastReport.Defects), len(want.Defects))
	}

	// The golden image is only pulled for the golden and diff views.
	if tdp.golden != nil {
		t.Error("golden image fetched although the test view was shown")
	}
	tdp.viewKind = "golden"
	tdp.requestRun()
	pollJobs(t, teach)
	if tdp.golden == nil || !bytes.Equal(tdp.golden.Data, sdp.golden.Data) {
		t.Fatal("live golden image differs from the studio's clean render")
	}
	tdp.viewKind = "test"

	// A frame over the link is byte-identical to what the camera program writes.
	if exe := findCameraExe(); exe != "" {
		meta := tdp.liveSt.scene.Meta
		dir := filepath.Join(t.TempDir(), "camera")
		saved, err := cameraCapture(exe, &meta, dir, []core.CameraChannel{core.ChRed}, true)
		if err != nil {
			t.Fatalf("cameraCapture: %v", err)
		}
		f, err := snapfmt.ReadImage(dir, saved, "red", snapfmt.KindTest)
		if err != nil {
			t.Fatalf("ReadImage: %v", err)
		}
		if !bytes.Equal(f.Pix, tdp.test.Data) {
			t.Error("camera.exe's red image differs from the frame received over the link")
		}
		g, err := snapfmt.ReadImage(dir, saved, "red", snapfmt.KindGood)
		if err != nil {
			t.Fatalf("ReadImage good: %v", err)
		}
		live, err := fetchLiveImage(tdp.liveSt.client, tdp.snapSrc.liveRev, core.ChRed, snapfmt.KindGood, teach.Scene.W, teach.Scene.H)
		if err != nil {
			t.Fatalf("fetch good sample: %v", err)
		}
		if !bytes.Equal(g.Pix, live.Data) {
			t.Error("camera.exe's good sample differs from the one received over the link")
		}
	}

	// Save what the teaching program sees, then load it into a teaching program that has no link.
	dir := filepath.Join(t.TempDir(), "session")
	tdp.snapSrc.path, tdp.snapSrc.pathAuto = dir, false
	tdp.requestSaveSession()
	pollJobs(t, teach)
	if tdp.snapSrc.msgKey != "defect.save.ok" {
		t.Fatalf("save: %s %s", tdp.snapSrc.msgKey, tdp.snapSrc.msgText)
	}
	for _, name := range []string{"snap.json", "inspect.json", "red.png", "green.png", "blue.png", "red-good.png", "green-good.png", "blue-good.png"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("save left no %s: %v", name, err)
		}
	}
	meta, mdir, err := snapfmt.ReadMeta(dir)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	for _, ch := range []string{"red", "green", "blue"} {
		for _, kind := range []string{snapfmt.KindTest, snapfmt.KindGood} {
			if _, err := snapfmt.ReadImage(mdir, meta, ch, kind); err != nil {
				t.Errorf("saved %s %s: %v", ch, kind, err)
			}
		}
	}
	pf, err := inspect.ReadParamsFile(dir)
	if err != nil {
		t.Fatalf("ReadParamsFile: %v", err)
	}
	if !pf.Channels["red"].Learned || !reflect.DeepEqual(pf.Channels["red"].Params, tdp.inspectParams[core.ChRed]) {
		t.Error("saved red parameters differ from the panel's")
	}

	off := newTestApp(t, ProfileTeach)
	legacyDetect(off)
	odp := defectPanelOf(off)
	odp.requestSnapLoad(dir)
	pollJobs(t, off)
	if !odp.snapSrc.on || odp.snapSrc.live || odp.test == nil || odp.lastReport == nil {
		t.Fatalf("load: on=%v live=%v msg=%s %s", odp.snapSrc.on, odp.snapSrc.live, odp.snapSrc.msgKey, odp.snapSrc.msgText)
	}
	if !bytes.Equal(odp.test.Data, tdp.test.Data) {
		t.Error("loaded test image differs from the live one")
	}
	if !odp.learned[core.ChRed] || !reflect.DeepEqual(odp.inspectParams[core.ChRed], tdp.inspectParams[core.ChRed]) {
		t.Error("loaded parameters differ from the saved ones")
	}
	if len(odp.lastReport.Defects) != len(tdp.lastReport.Defects) {
		t.Errorf("loaded session finds %d defects, the live one %d", len(odp.lastReport.Defects), len(tdp.lastReport.Defects))
	}
}

func TestLiveFollowAndCaptureExclusive(t *testing.T) {
	studio, teach, sdp, tdp := startLinkedApps(t)
	srv := studio.live.srv
	tickUntil(t, "the studio to publish", sdp.Tick, func() bool { return srv.Rev() >= 1 })
	tickUntil(t, "adoption", tdp.Tick, func() bool { return tdp.snapSrc.live })
	tdp.requestRun()
	pollJobs(t, teach)

	// Unchecking "follow" returns to the in-process simulation and stays there.
	tdp.setLiveFollow(false)
	if tdp.snapSrc.live || tdp.snapSrc.on || tdp.liveFollow {
		t.Fatal("still on the live source after unchecking follow")
	}
	tdp.requestRun() // the panel is not active in this test, so it does not re-run by itself
	pollJobs(t, teach)
	for i := 0; i < 5; i++ {
		tdp.Tick()
		time.Sleep(10 * time.Millisecond)
	}
	if tdp.snapSrc.live {
		t.Fatal("tick adopted the studio although follow is off")
	}
	if tdp.test == nil || tdp.lastReport == nil {
		t.Fatal("no local simulation after leaving the live source")
	}

	// Following again adopts the current revision.
	tdp.setLiveFollow(true)
	tickUntil(t, "re-adoption", tdp.Tick, func() bool { return tdp.snapSrc.live && tdp.snapSrc.liveRev == srv.Rev() })

	// Loading a capture stops following.
	dir := filepath.Join(t.TempDir(), "session")
	tdp.snapSrc.path, tdp.snapSrc.pathAuto = dir, false
	tdp.requestRun()
	pollJobs(t, teach)
	tdp.requestSaveSession()
	pollJobs(t, teach)
	tdp.requestSnapLoad(dir)
	pollJobs(t, teach)
	if tdp.snapSrc.live || tdp.liveFollow || !tdp.snapSrc.on {
		t.Fatalf("after load: live=%v follow=%v on=%v", tdp.snapSrc.live, tdp.liveFollow, tdp.snapSrc.on)
	}
	for i := 0; i < 5; i++ {
		tdp.Tick()
		time.Sleep(10 * time.Millisecond)
	}
	if tdp.snapSrc.live {
		t.Fatal("a loaded capture was replaced by the live image")
	}
}

func TestLiveModelMismatch(t *testing.T) {
	studio, teach, sdp, tdp := startLinkedApps(t)
	srv := studio.live.srv
	tickUntil(t, "the studio to publish", sdp.Tick, func() bool { return srv.Rev() >= 1 })

	other := *teach.Scene
	other.Info.ID = "FM1/other"
	real := teach.Scene
	teach.Scene = &other
	tickUntil(t, "the mismatch to be noticed", tdp.Tick, func() bool { return tdp.liveMismatch != "" })
	if tdp.liveMismatch != real.Info.ID {
		t.Errorf("liveMismatch = %q, want %q", tdp.liveMismatch, real.Info.ID)
	}
	if tdp.snapSrc.live {
		t.Error("followed a scene of another model")
	}
	teach.Scene = real
	tickUntil(t, "adoption once the models match", tdp.Tick, func() bool { return tdp.snapSrc.live })
	if tdp.liveMismatch != "" {
		t.Errorf("liveMismatch = %q after the models match", tdp.liveMismatch)
	}
}

func TestLiveClientRedials(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	teach := newTestApp(t, ProfileTeach)
	legacyDetect(teach)
	tdp := defectPanelOf(teach)
	teach.StartLive(addr)
	defer teach.closeLive()
	time.Sleep(100 * time.Millisecond)
	tdp.Tick()
	if tdp.liveSt.connected {
		t.Fatal("connected before any studio listens")
	}

	studio := newTestApp(t, ProfileStudio)
	studio.StartLive(addr)
	defer studio.closeLive()
	if studio.live.srv == nil {
		t.Fatalf("studio could not listen on %s: %s", addr, studio.live.listenErr)
	}
	sdp := defectPanelOf(studio)
	tickUntil(t, "the studio to publish", sdp.Tick, func() bool { return studio.live.srv.Rev() >= 1 })
	tickUntil(t, "the teach program to connect and adopt", tdp.Tick, func() bool { return tdp.snapSrc.live })
	if peers := studio.live.srv.Peers(); len(peers) != 1 || peers[0].App != teachProducer {
		t.Errorf("studio peers = %+v", peers)
	}

	// The studio going away keeps the last frames and the connection state follows.
	tdp.requestRun()
	pollJobs(t, teach)
	studio.closeLive()
	tickUntil(t, "the disconnect to show", tdp.Tick, func() bool { return !tdp.liveSt.connected })
	if tdp.test == nil || !tdp.snapSrc.live {
		t.Error("the last live image was dropped on disconnect")
	}
}

func TestStudioPublishDedupesAndDebounces(t *testing.T) {
	studio := newTestApp(t, ProfileStudio)
	studio.StartLive("127.0.0.1:0")
	defer studio.closeLive()
	srv := studio.live.srv
	sdp := defectPanelOf(studio)

	sdp.Tick() // the first sight of the scene starts a burst: nothing goes out yet
	if srv.Rev() != 0 {
		t.Fatalf("published immediately (rev %d)", srv.Rev())
	}
	time.Sleep(liveQuiet + 30*time.Millisecond)
	sdp.Tick()
	if srv.Rev() != 1 {
		t.Fatalf("rev = %d after the quiet period, want 1", srv.Rev())
	}

	sdp.publishLive()
	if srv.Rev() != 1 {
		t.Errorf("an unchanged scenario was published again (rev %d)", srv.Rev())
	}
	sdp.noise.ReadSigma = 2
	sdp.renderRev++
	sdp.Tick()
	time.Sleep(liveQuiet + 30*time.Millisecond)
	sdp.Tick()
	if srv.Rev() != 2 {
		t.Errorf("rev = %d after a noise change, want 2", srv.Rev())
	}

	// A continuous burst still goes out after liveMaxDelay.
	start := time.Now()
	for srv.Rev() == 2 && time.Since(start) < 3*time.Second {
		sdp.noise.ReadSigma += 0.5
		sdp.renderRev++
		sdp.Tick()
		time.Sleep(20 * time.Millisecond)
	}
	if srv.Rev() != 3 {
		t.Fatalf("a continuous burst was never published (rev %d)", srv.Rev())
	}
	if took := time.Since(start); took > liveMaxDelay+300*time.Millisecond {
		t.Errorf("burst published after %v, want about %v", took, liveMaxDelay)
	}
}

func TestResolveLinkAddr(t *testing.T) {
	t.Setenv("AFVI_LINK", "")
	cases := []struct {
		value   string
		profile Profile
		addr    string
		on      bool
		bad     bool
	}{
		{"", ProfileStudio, "127.0.0.1:47631", true, false},
		{"", ProfileTeach, "127.0.0.1:47631", true, false},
		{"off", ProfileStudio, "", false, false},
		{"None", ProfileTeach, "", false, false},
		{"0", ProfileTeach, "", false, false},
		{"5000", ProfileStudio, "127.0.0.1:5000", true, false},
		{"0.0.0.0:6000", ProfileTeach, "127.0.0.1:6000", true, false},
		{":6000", ProfileTeach, "127.0.0.1:6000", true, false},
		{"0.0.0.0:6000", ProfileStudio, "0.0.0.0:6000", true, false},
		{"10.1.2.3:7000", ProfileTeach, "10.1.2.3:7000", true, false},
		{"localhost", ProfileTeach, "", false, true},
		{"127.0.0.1:99999", ProfileStudio, "", false, true},
	}
	for _, c := range cases {
		addr, on, err := resolveLinkAddr(c.value, c.profile)
		if (err != nil) != c.bad || addr != c.addr || on != c.on {
			t.Errorf("resolveLinkAddr(%q, %v) = %q, %v, %v; want %q, %v, bad=%v", c.value, c.profile, addr, on, err, c.addr, c.on, c.bad)
		}
	}
	t.Setenv("AFVI_LINK", "7777")
	if addr, on, err := resolveLinkAddr("", ProfileStudio); err != nil || !on || addr != "127.0.0.1:7777" {
		t.Errorf("AFVI_LINK: %q, %v, %v", addr, on, err)
	}
	if addr, on, err := resolveLinkAddr("off", ProfileStudio); err != nil || on || addr != "" {
		t.Errorf("flag must win over AFVI_LINK: %q, %v, %v", addr, on, err)
	}
}

func TestSnapPathFollowsModelUntilEdited(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	dp.snapSrc.path, dp.snapSrc.pathAuto = "", false
	if got, want := dp.sessionDir(), defaultCaptureDir(a.Scene); got != want {
		t.Errorf("empty path: sessionDir = %q, want the default %q", got, want)
	}
	dp.snapSrc.path, dp.snapSrc.pathAuto = "custom", false
	dp.resetSnap()
	if dp.snapSrc.path != "custom" || dp.snapSrc.pathAuto {
		t.Errorf("resetSnap changed the path to %q (auto %v)", dp.snapSrc.path, dp.snapSrc.pathAuto)
	}
	dp.snapSrc.path, dp.snapSrc.pathAuto = defaultCaptureDir(a.Scene), true
	dp.resetSnap()
	if !dp.snapSrc.pathAuto {
		t.Error("resetSnap dropped the automatic path")
	}
}

func TestParamsFileRoundTripThroughPanel(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	legacyDetect(a)
	dp := defectPanelOf(a)
	dp.requestRun()
	pollJobs(t, a)
	dir := filepath.Join(t.TempDir(), "p")
	dp.snapSrc.path, dp.snapSrc.pathAuto = dir, false
	dp.learnK = 3.5
	dp.inspectParams[core.ChGreen].Materials[inspect.Metal].Low = 11
	dp.saveParamsFile()
	if dp.inspMsg.key != "defect.params.saved" {
		t.Fatalf("save: %+v", dp.inspMsg)
	}
	dp.learnK = 5
	dp.learned = [3]bool{}
	dp.inspectParams[core.ChGreen] = newInspectParams()
	if !dp.loadParamsFile() {
		t.Fatalf("load: %+v", dp.inspMsg)
	}
	if dp.learnK != 3.5 || dp.inspectParams[core.ChGreen].Materials[inspect.Metal].Low != 11 || !dp.learned[core.ChRed] {
		t.Errorf("loaded learnK=%v green metal low=%v red learned=%v", dp.learnK, dp.inspectParams[core.ChGreen].Materials[inspect.Metal].Low, dp.learned[core.ChRed])
	}

	// A file of another model is ignored with a message.
	pf, err := inspect.ReadParamsFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	pf.Model = "FM1/other"
	if err := inspect.WriteParamsFile(filepath.Join(dir, inspect.ParamsFileName), pf); err != nil {
		t.Fatal(err)
	}
	if dp.loadParamsFile() || dp.inspMsg.key != "defect.params.mismatch" {
		t.Errorf("mismatching file: %+v", dp.inspMsg)
	}
	if err := os.Remove(filepath.Join(dir, inspect.ParamsFileName)); err != nil {
		t.Fatal(err)
	}
	if dp.loadParamsFile() || dp.inspMsg.key != "defect.params.missing" {
		t.Errorf("missing file: %+v", dp.inspMsg)
	}
}
