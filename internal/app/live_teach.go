package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/link"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// errLiveDisconnected is reported when an image is needed while no studio is connected.
var errLiveDisconnected = errors.New("not connected to the camera studio")

// liveClient keeps the teaching program connected to the studio: a goroutine dials, waits
// for the connection to end and dials again. The UI thread only reads State.
type liveClient struct {
	addr string
	app  string
	stop chan struct{}
	wg   sync.WaitGroup

	mu      sync.Mutex
	cl      *link.Client
	lastErr error
}

// liveState is a snapshot of the connection for one frame.
type liveState struct {
	connected bool
	client    *link.Client
	peer      link.Hello
	scene     link.Scene
	hasScene  bool
	err       error // why the last attempt or connection failed (nil = no problem to show)
}

func newLiveClient(addr, app string) *liveClient {
	c := &liveClient{addr: addr, app: app, stop: make(chan struct{})}
	c.wg.Add(1)
	go c.run()
	return c
}

func (c *liveClient) run() {
	defer c.wg.Done()
	for {
		cl, err := link.Dial(c.addr, c.app, liveDialTimeout)
		if err != nil {
			c.set(nil, err)
		} else {
			c.set(cl, nil)
			select {
			case <-cl.Done():
				c.set(nil, cl.Err())
				_ = cl.Close()
			case <-c.stop:
				_ = cl.Close()
				c.set(nil, nil)
				return
			}
		}
		select {
		case <-time.After(liveRedial):
		case <-c.stop:
			return
		}
	}
}

func (c *liveClient) set(cl *link.Client, err error) {
	c.mu.Lock()
	c.cl, c.lastErr = cl, err
	c.mu.Unlock()
}

// State returns the connection state and the newest scene the studio published.
func (c *liveClient) State() liveState {
	c.mu.Lock()
	cl, err := c.cl, c.lastErr
	c.mu.Unlock()
	st := liveState{err: err}
	if cl == nil {
		return st
	}
	st.connected, st.client, st.peer = true, cl, cl.Peer()
	st.scene, st.hasScene = cl.Scene()
	return st
}

// Close ends the connection and stops redialling.
func (c *liveClient) Close() {
	select {
	case <-c.stop:
		return
	default:
		close(c.stop)
	}
	c.mu.Lock()
	cl := c.cl
	c.mu.Unlock()
	if cl != nil {
		_ = cl.Close()
	}
	c.wg.Wait()
}

// waitingOnly reports whether err just says that nobody is listening (the normal state
// while the studio is not running), which is not worth a warning.
func waitingOnly(err error) bool {
	var op *net.OpError
	return err == nil || errors.Is(err, io.EOF) || (errors.As(err, &op) && op.Op == "dial")
}

// tickTeach follows the studio: a new revision (or a first connection) replaces the test
// image source with the live frames of that revision.
func (p *DefectPanel) tickTeach() {
	lc := p.a.live.cli
	if lc == nil {
		return
	}
	p.liveSt = lc.State()
	p.liveMismatch = ""
	scene := p.a.Scene
	st := p.liveSt
	if scene == nil || !p.liveFollow || !st.connected || !st.hasScene {
		return
	}
	s := st.scene
	if p.snapSrc.live && p.snapSrc.liveSess == s.Session && p.snapSrc.liveRev == s.Rev {
		return
	}
	if time.Now().Before(p.snapSrc.liveRetry) {
		return
	}
	if s.Meta.Model.ID != scene.Info.ID || s.Meta.Width != scene.W || s.Meta.Height != scene.H {
		p.liveMismatch = s.Meta.Model.ID
		return
	}
	p.adoptLive(s)
}

// adoptLive makes the studio's scene revision the test image source. Nothing of the
// studio's light, product or strokes is copied: the teaching program only gets pixels.
func (p *DefectPanel) adoptLive(s link.Scene) {
	meta := s.Meta
	meta.Images = nil
	src := &p.snapSrc
	p.dropSnapDirs("")
	p.clearUserImage()
	src.on, src.live = true, true
	src.liveSess, src.liveRev = s.Session, s.Rev
	src.liveRetry = time.Time{}
	src.meta = &meta
	src.frames = map[snapKey]*core.GrayImage{}
	src.msgKey, src.msgText, src.msgErr = "", "", false
	src.checkedRev = 0
	src.stale = false
	p.golden = nil
	p.renderRev++
	if p.active {
		p.requestRun()
	}
}

// leaveLive stops following the studio and returns to the in-process simulation (the
// capture actions call it before they adopt a capture of their own).
func (p *DefectPanel) leaveLive() {
	p.liveFollow = false
	src := &p.snapSrc
	if !src.live {
		return
	}
	src.on, src.live = false, false
	src.liveSess, src.liveRev = "", 0
	src.meta = nil
	src.frames = nil
	p.golden, p.test = nil, nil
	p.renderRev++
}

// setLiveFollow switches the "follow the studio" checkbox.
func (p *DefectPanel) setLiveFollow(on bool) {
	if on {
		p.liveFollow = true
		return
	}
	wasLive := p.snapSrc.live
	p.leaveLive()
	if wasLive && p.active {
		p.requestRun()
	}
}

// liveFetchFailed records a failed fetch of the live revision (sess, rev).
func (p *DefectPanel) liveFetchFailed(sess string, rev int64, err error) {
	if errors.Is(err, link.ErrStale) {
		return // the studio moved on; tickTeach adopts the newer revision
	}
	src := &p.snapSrc
	if !src.live || src.liveSess != sess || src.liveRev != rev {
		return
	}
	src.msgKey, src.msgText, src.msgErr = "defect.live.error", err.Error(), true
	// Ask again after a pause: forget the revision so tickTeach adopts it once more.
	src.liveRetry = time.Now().Add(liveRetryDelay)
	src.liveRev = 0
}

// liveFetch returns the wanted frames of one channel for revision rev: the test image,
// the good sample (for learning) and the golden image (for the golden and diff views).
// Frames already in have are reused; the rest are pulled over the link in parallel.
func liveFetch(cl *link.Client, rev int64, have map[snapKey]*core.GrayImage, ch core.CameraChannel, wantGood, wantGolden bool, w, h int) (snapFetchResult, error) {
	res := snapFetchResult{add: map[snapKey]*core.GrayImage{}}
	kinds := []string{link.KindTest}
	if wantGood {
		kinds = append(kinds, link.KindGood)
	}
	if wantGolden {
		kinds = append(kinds, link.KindGolden)
	}
	imgs := make([]*core.GrayImage, len(kinds))
	errs := make([]error, len(kinds))
	var wg sync.WaitGroup
	for i, kind := range kinds {
		if img := have[snapKey{ch: ch, kind: kind}]; img != nil {
			imgs[i] = img
			continue
		}
		if cl == nil {
			return res, errLiveDisconnected
		}
		wg.Add(1)
		go func(i int, kind string) {
			defer wg.Done()
			imgs[i], errs[i] = fetchLiveImage(cl, rev, ch, kind, w, h)
		}(i, kind)
	}
	wg.Wait()
	for i, kind := range kinds {
		if errs[i] != nil {
			return res, errs[i]
		}
		key := snapKey{ch: ch, kind: kind}
		if have[key] == nil {
			res.add[key] = imgs[i]
		}
		switch kind {
		case link.KindTest:
			res.test = imgs[i]
		case link.KindGood:
			res.good = imgs[i]
		default:
			res.golden = imgs[i]
		}
	}
	return res, nil
}

// fetchLiveImage pulls one frame and checks its size against the loaded model.
func fetchLiveImage(cl *link.Client, rev int64, ch core.CameraChannel, kind string, w, h int) (*core.GrayImage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), liveFetchLimit)
	defer cancel()
	f, _, err := cl.Frame(ctx, rev, snapfmt.ChannelName(ch), kind)
	if err != nil {
		return nil, err
	}
	if f.Width != w || f.Height != h {
		return nil, fmt.Errorf("live: %s %s image is %dx%d, want %dx%d", snapfmt.ChannelName(ch), kind, f.Width, f.Height, w, h)
	}
	return frameToGray(f), nil
}

// drawLiveSection draws the teaching program's "Live link" section.
func (p *DefectPanel) drawLiveSection() {
	if !Section(T("defect.section.live"), "defect-live") {
		return
	}
	l := p.a.live
	if l == nil || l.cli == nil {
		Note(T("defect.live.off"))
		return
	}
	Note(T("defect.live.note"))
	st := p.liveSt
	if st.connected {
		Note(T("defect.live.connected", "app", st.peer.App, "pid", fmt.Sprint(st.peer.PID), "addr", l.addr))
	} else {
		Note(T("defect.live.waiting", "addr", l.addr))
		if !waitingOnly(st.err) {
			Warn(st.err.Error())
		}
	}
	follow := p.liveFollow
	if Check(T("defect.live.follow"), "live-follow", &follow) {
		p.setLiveFollow(follow)
	}
	var source string
	switch {
	case p.user.img != nil:
		source = T("defect.manual.srcImage", "name", filepath.Base(p.user.path))
	case p.snapSrc.live:
		source = T("defect.live.srcLive", "rev", fmt.Sprint(p.snapSrc.liveRev))
	case p.snapSrc.on && p.snapSrc.meta != nil:
		source = T("defect.live.srcSnap", "dir", p.snapSrc.srcDir)
	default:
		source = T("defect.live.srcLocal")
	}
	rows := [][2]string{{T("defect.live.source"), source}}
	if st.connected && st.hasScene {
		m := st.scene.Meta
		rows = append(rows,
			[2]string{T("defect.live.studioRev"), fmt.Sprint(st.scene.Rev)},
			[2]string{T("defect.snap.model"), m.Model.ID},
			[2]string{T("defect.snap.defects"), fmt.Sprint(len(m.Recipe.Defects))},
		)
		if m.Recipe.Warp != nil {
			rows = append(rows, [2]string{T("defect.snap.warp"), warpSummary(m.Recipe.Warp)})
		}
	}
	KVTable("live-kv", rows)
	if p.liveMismatch != "" && p.liveFollow {
		Warn(T("defect.live.mismatch", "model", p.liveMismatch))
		idx := -1
		for i, m := range p.a.Models {
			if m.ID == p.liveMismatch {
				idx = i
			}
		}
		if idx < 0 {
			Warn(T("defect.live.noModel", "model", p.liveMismatch))
		} else if Button(T("defect.live.switchModel"), "live-switch") {
			p.a.LoadModel(idx)
		}
	}
	if src := &p.snapSrc; src.live && strings.HasPrefix(src.msgKey, "defect.live.") {
		Warn(T(src.msgKey, "msg", src.msgText))
	}
}

func init() {
	Messages(map[string]string{
		"defect.section.live":     "实时联机",
		"defect.live.note":        "通过本地端口实时取相机工作室里的图：工作室里画的缺陷马上出现在测试图里，不经过磁盘；检出仍只看测试图与材质图。",
		"defect.live.off":         "实时联机已关闭（-link off）。",
		"defect.live.waiting":     "等待相机工作室（{addr}）……",
		"defect.live.connected":   "已连接：{app}（PID {pid}），{addr}",
		"defect.live.follow":      "跟随相机工作室",
		"defect.live.source":      "测试图来源",
		"defect.live.srcLive":     "相机工作室（实时，修订 {rev}）",
		"defect.live.srcSnap":     "抓拍文件夹 {dir}",
		"defect.live.srcLocal":    "本程序的本地仿真",
		"defect.live.studioRev":   "工作室修订",
		"defect.live.mismatch":    "相机工作室打开的是型号 {model}，与当前型号不同，所以没有跟随。",
		"defect.live.noModel":     "本机没有型号 {model}。",
		"defect.live.switchModel": "切换到该型号",
		"defect.live.error":       "实时取图失败：{msg}",
	}, map[string]string{
		"defect.section.live":     "Live link",
		"defect.live.note":        "Pulls the image from the camera studio over a local port: a defect drawn there shows up in the test image at once, without a file; inspection still only sees the test image and the material map.",
		"defect.live.off":         "The live link is off (-link off).",
		"defect.live.waiting":     "Waiting for the camera studio ({addr}) ...",
		"defect.live.connected":   "Connected: {app} (PID {pid}), {addr}",
		"defect.live.follow":      "Follow the camera studio",
		"defect.live.source":      "Test image source",
		"defect.live.srcLive":     "Camera studio (live, revision {rev})",
		"defect.live.srcSnap":     "Capture folder {dir}",
		"defect.live.srcLocal":    "This program's own simulation",
		"defect.live.studioRev":   "Studio revision",
		"defect.live.mismatch":    "The camera studio has model {model} open, which differs from the current one, so it is not followed.",
		"defect.live.noModel":     "Model {model} is not available on this machine.",
		"defect.live.switchModel": "Switch to that model",
		"defect.live.error":       "Live fetch failed: {msg}",
	}, map[string]string{
		"defect.section.live":     "실시간 연동",
		"defect.live.note":        "로컬 포트로 카메라 스튜디오의 이미지를 실시간으로 가져옵니다. 스튜디오에서 그린 결함이 파일 없이 바로 테스트 이미지에 나타나며, 검출은 여전히 테스트 이미지와 재질 맵만 봅니다.",
		"defect.live.off":         "실시간 연동이 꺼져 있습니다 (-link off).",
		"defect.live.waiting":     "카메라 스튜디오({addr}) 대기 중 ...",
		"defect.live.connected":   "연결됨: {app} (PID {pid}), {addr}",
		"defect.live.follow":      "카메라 스튜디오 따라가기",
		"defect.live.source":      "테스트 이미지 출처",
		"defect.live.srcLive":     "카메라 스튜디오 (실시간, 리비전 {rev})",
		"defect.live.srcSnap":     "캡처 폴더 {dir}",
		"defect.live.srcLocal":    "이 프로그램의 자체 시뮬레이션",
		"defect.live.studioRev":   "스튜디오 리비전",
		"defect.live.mismatch":    "카메라 스튜디오에 열린 모델은 {model} 이며 현재 모델과 달라 따라가지 않습니다.",
		"defect.live.noModel":     "이 장비에는 모델 {model} 이 없습니다.",
		"defect.live.switchModel": "해당 모델로 전환",
		"defect.live.error":       "실시간 가져오기 실패: {msg}",
	})
}
