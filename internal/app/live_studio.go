package app

import (
	"fmt"
	"sync"
	"time"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/link"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// livePublisher decides when the studio's scenario is published: a change of the scene or
// of the panel's render revision starts a burst; the scenario goes out once the burst has
// been quiet for liveQuiet (or has lasted liveMaxDelay, so a long drag still updates).
type livePublisher struct {
	scene     *Scene
	rev       uint64
	dirtyFrom time.Time // first change of the current burst (zero = nothing pending)
	changedAt time.Time // latest change
	key       string    // model + recipe of the scenario that was published last
}

// Tick publishes the studio's scenario (author mode) or adopts the studio's scenario
// (detect mode) when the live link is on.
func (p *DefectPanel) Tick() {
	if p.a.live == nil {
		return
	}
	if p.mode == modeAuthor {
		p.tickStudio()
	} else {
		p.tickTeach()
	}
}

func (p *DefectPanel) tickStudio() {
	l := p.a.live
	if l.srv == nil || p.a.Scene == nil {
		return
	}
	now := time.Now()
	pub := &p.pub
	if p.a.Scene != pub.scene || p.renderRev != pub.rev {
		pub.scene, pub.rev = p.a.Scene, p.renderRev
		if pub.dirtyFrom.IsZero() {
			pub.dirtyFrom = now
		}
		pub.changedAt = now
	}
	if pub.dirtyFrom.IsZero() {
		return
	}
	if now.Sub(pub.changedAt) < liveQuiet && now.Sub(pub.dirtyFrom) < liveMaxDelay {
		return
	}
	pub.dirtyFrom = time.Time{}
	p.publishLive()
}

// publishLive hands the current scenario to the server. The renderer works from a copy of
// the description, so later edits never change a revision that is already published.
func (p *DefectPanel) publishLive() {
	scene := p.a.Scene
	meta := p.snapMeta()
	meta.Producer = studioProducer
	key := fmt.Sprintf("%p|%s|%s", scene, scene.Info.ID, recipeKey(meta.Recipe))
	if key == p.pub.key {
		return
	}
	p.pub.key = key
	r := &liveRenderer{scene: scene, cr: meta.Recipe.Core()}
	p.a.live.srv.Publish(meta, r.render)
}

// liveRenderer renders the frames of one published revision exactly like the camera
// program does (same scene, same recipe, same functions), so a frame taken over the link
// is byte-identical to the PNG camera.exe writes for that description.
type liveRenderer struct {
	scene *Scene
	cr    snapfmt.CoreRecipe

	mu   sync.Mutex
	side map[snapKey]*core.GrayImage // the other half of the last test/golden pair
}

func (r *liveRenderer) render(channel, kind string) (snapfmt.Frame, error) {
	ch, err := snapfmt.ParseChannel(channel)
	if err != nil {
		return snapfmt.Frame{}, err
	}
	var img *core.GrayImage
	switch kind {
	case link.KindGood:
		light := r.cr.Light
		img = core.RenderGoodSample(r.scene.Light, &light, ch, goodSampleNoise(r.cr.Noise))
	case link.KindTest, link.KindGolden:
		img = r.pair(ch, kind)
	default:
		return snapfmt.Frame{}, fmt.Errorf("unknown kind %q", kind)
	}
	return snapfmt.Frame{Width: img.Width, Height: img.Height, Pix: img.Data}, nil
}

// pair returns the test or golden image of ch. One render produces both, so the half that
// was not asked for is kept for the request that usually follows.
func (r *liveRenderer) pair(ch core.CameraChannel, kind string) *core.GrayImage {
	r.mu.Lock()
	defer r.mu.Unlock()
	if img, ok := r.side[snapKey{ch: ch, kind: kind}]; ok {
		delete(r.side, snapKey{ch: ch, kind: kind})
		return img
	}
	light := r.cr.Light
	golden, test := core.RenderDefectPair(r.scene.Light, &light, ch, r.cr.Scene, r.cr.Defects, r.cr.Noise)
	if kind == link.KindGolden {
		r.side = map[snapKey]*core.GrayImage{{ch: ch, kind: link.KindTest}: test}
		return golden
	}
	r.side = map[snapKey]*core.GrayImage{{ch: ch, kind: link.KindGolden}: golden}
	return test
}

// drawLiveStudio draws the studio's "Live link" section of the Output column.
func (p *DefectPanel) drawLiveStudio() {
	if !Section(T("studio.section.live"), "studio-live") {
		return
	}
	l := p.a.live
	switch {
	case l == nil:
		Note(T("studio.live.off"))
	case l.srv == nil:
		Warn(T("studio.live.failed", "addr", l.addr, "msg", l.listenErr))
	default:
		Note(T("studio.live.note"))
		Note(T("studio.live.listening", "addr", l.srv.Addr().String()))
		peers := l.srv.Peers()
		if len(peers) == 0 {
			Note(T("studio.live.noPeers"))
		}
		for _, h := range peers {
			Note(T("studio.live.peer", "app", h.App, "pid", fmt.Sprint(h.PID)))
		}
		Note(T("studio.live.rev", "rev", fmt.Sprint(l.srv.Rev())))
	}
}

func init() {
	Messages(map[string]string{
		"studio.section.live":   "实时联机",
		"studio.live.note":      "教学程序通过本地端口实时取图：这里的修改约 0.1 秒后就会出现在教学程序的测试图里，不需要保存文件。",
		"studio.live.off":       "实时联机已关闭（-link off）。",
		"studio.live.failed":    "无法监听 {addr}：{msg}（端口被占用？用 -link 指定其他地址，教学程序也要用同一个地址）",
		"studio.live.listening": "正在监听 {addr}",
		"studio.live.noPeers":   "还没有教学程序连接。",
		"studio.live.peer":      "已连接：{app}（PID {pid}）",
		"studio.live.rev":       "当前修订：{rev}",
	}, map[string]string{
		"studio.section.live":   "Live link",
		"studio.live.note":      "The teach program pulls images over a local port: an edit here shows up in its test image about 0.1 s later, with no file saved.",
		"studio.live.off":       "The live link is off (-link off).",
		"studio.live.failed":    "Cannot listen on {addr}: {msg} (port in use? pass -link with another address, and the same one to the teach program)",
		"studio.live.listening": "Listening on {addr}",
		"studio.live.noPeers":   "No teach program is connected yet.",
		"studio.live.peer":      "Connected: {app} (PID {pid})",
		"studio.live.rev":       "Current revision: {rev}",
	}, map[string]string{
		"studio.section.live":   "실시간 연동",
		"studio.live.note":      "교육 프로그램이 로컬 포트로 이미지를 실시간으로 가져갑니다. 여기서 수정하면 약 0.1초 뒤 교육 프로그램의 테스트 이미지에 나타나며 파일 저장이 필요 없습니다.",
		"studio.live.off":       "실시간 연동이 꺼져 있습니다 (-link off).",
		"studio.live.failed":    "{addr} 에서 대기할 수 없습니다: {msg} (포트가 사용 중? -link 로 다른 주소를 지정하고 교육 프로그램에도 같은 주소를 지정하세요)",
		"studio.live.listening": "{addr} 에서 대기 중",
		"studio.live.noPeers":   "연결된 교육 프로그램이 아직 없습니다.",
		"studio.live.peer":      "연결됨: {app} (PID {pid})",
		"studio.live.rev":       "현재 리비전: {rev}",
	})
}
