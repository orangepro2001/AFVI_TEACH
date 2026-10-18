package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/core"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// snapKey identifies one decoded image of a capture.
type snapKey struct {
	ch   core.CameraChannel
	kind string // snapfmt.KindTest | snapfmt.KindGood
}

// snapSource is the optional camera snap used as the defect tab's test image.
type snapSource struct {
	on      bool
	srcDir  string                      // folder that holds the capture's images ("" until a capture is adopted)
	workDir string                      // app-owned temp folder for images rendered on demand ("" until needed); removed on reset
	owned   []string                    // every app-owned temp folder of this capture; all removed on reset
	meta    *snapfmt.Meta               // description of the adopted capture
	frames  map[snapKey]*core.GrayImage // decoded images cache, written only on the UI thread
	path    string                      // text field of the Load button
	msgKey  string                      // i18n key of the last message ("" = none)
	msgText string                      // {msg} argument of msgKey
	msgErr  bool                        // the last message is an error

	checkedRev uint64 // panel renderRev for which stale was computed
	stale      bool

	pathAuto bool // path still follows the default folder of the model (the user has not edited it)

	// Live source: the frames come from the camera studio over the live link, not from files.
	// on is true as well; meta is the studio's description of the revision.
	live      bool
	liveSess  string    // studio session of the adopted scene
	liveRev   int64     // revision of the adopted scene (0 = ask again after liveRetry)
	liveRetry time.Time // no new adoption before this time (after a failed fetch)
}

// snapFetchResult is what a background fetch returns to the UI thread.
type snapFetchResult struct {
	meta     *snapfmt.Meta
	workDir  string
	add      map[snapKey]*core.GrayImage
	test     *core.GrayImage
	good     *core.GrayImage // nil when wantGood was false
	golden   *core.GrayImage // live source only: nil unless the golden image was wanted
	rendered bool            // the camera program was run
}

// findCameraExe locates the camera program: AFVI_CAMERA_EXE, then next to this
// executable, then on PATH.
func findCameraExe() string {
	if env := os.Getenv("AFVI_CAMERA_EXE"); env != "" {
		if st, err := os.Stat(env); err == nil && st.Mode().IsRegular() {
			return env
		}
	}
	name := "camera"
	if runtime.GOOS == "windows" {
		name = "camera.exe"
	}
	if exe, err := os.Executable(); err == nil {
		cand := filepath.Join(filepath.Dir(exe), name)
		if st, err := os.Stat(cand); err == nil && st.Mode().IsRegular() {
			return cand
		}
	}
	if p, err := exec.LookPath("camera"); err == nil {
		return p
	}
	return ""
}

// runCamera renders a capture; the error carries the exit status and the tail of the output.
func runCamera(exe, in, out string, channels []core.CameraChannel, good bool) error {
	names := make([]string, 0, len(channels))
	for _, ch := range channels {
		names = append(names, snapfmt.ChannelName(ch))
	}
	args := []string{"-in", in, "-out", out, "-mode", "image", "-channels", strings.Join(names, ",")}
	if good {
		args = append(args, "-good")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, args...)
	hideConsole(cmd)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		tail := outBytes
		if len(tail) > 400 {
			tail = tail[len(tail)-400:]
		}
		return fmt.Errorf("camera: %w: %s", err, strings.TrimSpace(string(tail)))
	}
	return nil
}

// frameToGray wraps a decoded frame; the pixel slice is shared.
func frameToGray(f snapfmt.Frame) *core.GrayImage {
	return &core.GrayImage{Width: f.Width, Height: f.Height, Data: f.Pix}
}

// snapFetch decodes the wanted images, reading them from srcDir when the capture has
// them and otherwise rendering them with the camera program. It runs on a job
// goroutine and touches no panel state.
func snapFetch(exe, srcDir string, meta *snapfmt.Meta, have map[snapKey]*core.GrayImage, workDir string, ch core.CameraChannel, wantGood bool, w, h int) (snapFetchResult, error) {
	res := snapFetchResult{meta: meta, workDir: workDir, add: map[snapKey]*core.GrayImage{}}
	name := snapfmt.ChannelName(ch)
	kinds := []string{snapfmt.KindTest}
	if wantGood {
		kinds = append(kinds, snapfmt.KindGood)
	}
	var missing []string
	for _, kind := range kinds {
		key := snapKey{ch: ch, kind: kind}
		if img, ok := have[key]; ok && img != nil {
			if kind == snapfmt.KindTest {
				res.test = img
			} else {
				res.good = img
			}
			continue
		}
		if meta != nil {
			if _, ok := meta.FindImage(name, kind); ok {
				f, err := snapfmt.ReadImage(srcDir, meta, name, kind)
				if err != nil {
					return res, err
				}
				if f.Width != w || f.Height != h {
					return res, fmt.Errorf("snap: image %s is %dx%d, want %dx%d", name, f.Width, f.Height, w, h)
				}
				img := frameToGray(f)
				res.add[key] = img
				if kind == snapfmt.KindTest {
					res.test = img
				} else {
					res.good = img
				}
				continue
			}
		}
		missing = append(missing, kind)
	}
	if len(missing) == 0 {
		return res, nil
	}
	if exe == "" {
		return res, fmt.Errorf("snap: camera program not found")
	}
	if workDir == "" {
		dir, err := os.MkdirTemp("", "afvi-snap-")
		if err != nil {
			return res, err
		}
		workDir = dir
	}
	// Write the description unchanged: the camera drops entries whose files are missing
	// from the output folder and keeps the rest, so earlier channels stay listed.
	metaFile := filepath.Join(workDir, snapfmt.MetaFile)
	if err := snapfmt.WriteMeta(metaFile, meta); err != nil {
		return res, err
	}
	if err := runCamera(exe, metaFile, workDir, []core.CameraChannel{ch}, wantGood); err != nil {
		return res, err
	}
	rendered, _, err := snapfmt.ReadMeta(workDir)
	if err != nil {
		return res, err
	}
	for _, kind := range missing {
		f, err := snapfmt.ReadImage(workDir, rendered, name, kind)
		if err != nil {
			return res, err
		}
		if f.Width != w || f.Height != h {
			return res, fmt.Errorf("snap: image %s is %dx%d, want %dx%d", name, f.Width, f.Height, w, h)
		}
		img := frameToGray(f)
		res.add[snapKey{ch: ch, kind: kind}] = img
		if kind == snapfmt.KindTest {
			res.test = img
		} else {
			res.good = img
		}
	}
	res.meta = rendered
	res.workDir = workDir
	res.rendered = true
	return res, nil
}

// recipeKey is a comparable form of a recipe; "" when it cannot be marshalled.
func recipeKey(r snapfmt.Recipe) string {
	data, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	return string(data)
}

// snapMeta describes the current scenario for the camera program.
func (p *DefectPanel) snapMeta() snapfmt.Meta {
	scene := p.a.Scene
	return snapfmt.Meta{
		Format:   snapfmt.Format,
		Version:  snapfmt.Version,
		Producer: "afvi-teach",
		Created:  time.Now().UTC().Format(time.RFC3339),
		Model:    scene.Info.Ref(),
		Width:    scene.W,
		Height:   scene.H,
		Recipe:   snapfmt.NewRecipe(scene.Product, p.a.Light.Clone(), p.noise, p.gvStrokes(), p.sceneStrokes()),
		Images:   []snapfmt.Image{},
	}
}

// requestSnap renders a fresh capture of the current scenario.
func (p *DefectPanel) requestSnap() {
	a := p.a
	scene := a.Scene
	if scene == nil {
		return
	}
	exe := findCameraExe()
	if exe == "" {
		p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.snap.noCamera", "", true
		return
	}
	meta := p.snapMeta()
	ch := p.channel
	needGood := !p.learned[ch] || p.relearn
	w, h := scene.W, scene.H
	a.Jobs.Run("snap", func() func() {
		work, err := os.MkdirTemp("", "afvi-snap-")
		if err != nil {
			return func() {
				p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.snap.error", err.Error(), true
			}
		}
		res, err := snapFetch(exe, work, &meta, nil, work, ch, needGood, w, h)
		if err != nil {
			os.RemoveAll(work)
			return func() {
				p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.snap.error", err.Error(), true
			}
		}
		return func() {
			if a.Scene != scene {
				os.RemoveAll(work)
				return
			}
			p.leaveLive()
			p.clearUserImage()
			p.dropSnapDirs(work)
			p.ownDir(work)
			p.snapSrc.srcDir = work
			p.snapSrc.workDir = res.workDir
			p.snapSrc.meta = res.meta
			p.snapSrc.frames = res.add
			p.snapSrc.on = true
			p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.snap.ok", filepath.Base(work), false
			p.snapSrc.checkedRev = 0
			p.requestRun()
		}
	})
}

// requestSnapLoad adopts an existing capture folder or description file.
func (p *DefectPanel) requestSnapLoad(path string) {
	a := p.a
	scene := a.Scene
	if scene == nil {
		return
	}
	path = strings.TrimSpace(path)
	if path == "" {
		p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.snap.badPath", "", true
		return
	}
	exe := findCameraExe()
	ch := p.channel
	needGood := !p.learned[ch] || p.relearn
	w, h := scene.W, scene.H
	a.Jobs.Run("snap", func() func() {
		meta, dir, err := snapfmt.ReadMeta(path)
		if err != nil {
			return func() {
				p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.snap.error", err.Error(), true
			}
		}
		if meta.Model.ID != scene.Info.ID || meta.Width != w || meta.Height != h {
			return func() {
				p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.snap.mismatch", meta.Model.ID, true
			}
		}
		res, err := snapFetch(exe, dir, meta, nil, "", ch, needGood, w, h)
		if err != nil {
			return func() {
				p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.snap.error", err.Error(), true
			}
		}
		pf, pmsg := readSessionParams(dir, scene)
		return func() {
			if a.Scene != scene {
				if res.workDir != "" {
					os.RemoveAll(res.workDir)
				}
				return
			}
			p.leaveLive()
			p.clearUserImage()
			if pf != nil {
				p.applyParamsFile(pf)
			}
			if pmsg.key != "" {
				p.inspMsg = pmsg
			}
			wasOwned := p.isOwned(dir)
			p.dropSnapDirs(dir)
			if wasOwned {
				p.ownDir(dir)
			}
			p.snapSrc.srcDir = dir
			if res.workDir != "" {
				p.ownDir(res.workDir)
				p.snapSrc.workDir = res.workDir
			} else if wasOwned {
				p.snapSrc.workDir = dir
			} else {
				p.snapSrc.workDir = ""
			}
			p.snapSrc.meta = meta
			p.snapSrc.frames = res.add
			cr := meta.Recipe.Core()
			p.strokes = nil
			for i := range cr.Defects {
				s := cr.Defects[i]
				p.strokes = append(p.strokes, strokeEntry{gv: &s, enabled: true})
			}
			for i := range cr.Scene {
				s := cr.Scene[i]
				p.strokes = append(p.strokes, strokeEntry{scene: &s, enabled: true})
			}
			p.noise = cr.Noise
			p.clearSelection()
			p.renderRev++
			p.snapSrc.on = true
			p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.snap.loaded", filepath.Base(dir), false
			p.snapSrc.checkedRev = 0
			p.requestRun()
		}
	})
}

// snapStale reports whether the scenario changed since the capture was taken.
func (p *DefectPanel) snapStale() bool {
	if !p.snapSrc.on || p.snapSrc.live || p.snapSrc.meta == nil || p.a.Scene == nil {
		return false
	}
	if p.renderRev != p.snapSrc.checkedRev {
		scene := p.a.Scene
		cur := recipeKey(snapfmt.NewRecipe(scene.Product, p.a.Light.Clone(), p.noise, p.gvStrokes(), p.sceneStrokes()))
		p.snapSrc.stale = cur != recipeKey(p.snapSrc.meta.Recipe)
		p.snapSrc.checkedRev = p.renderRev
	}
	return p.snapSrc.stale
}

// setSnapOff returns to the in-process simulation.
func (p *DefectPanel) setSnapOff() {
	p.snapSrc.on = false
	p.renderRev++
	if p.active {
		p.requestRun()
	}
}

// resetSnap drops the adopted capture and its app-owned folders.
func (p *DefectPanel) resetSnap() {
	p.dropSnapDirs("")
	path, auto := p.snapSrc.path, p.snapSrc.pathAuto
	p.snapSrc = snapSource{path: path, pathAuto: auto}
}

// ownDir registers dir as an app-owned temporary folder of the adopted capture.
func (p *DefectPanel) ownDir(dir string) {
	if dir == "" {
		return
	}
	for _, d := range p.snapSrc.owned {
		if d == dir {
			return
		}
	}
	p.snapSrc.owned = append(p.snapSrc.owned, dir)
}

// isOwned reports whether dir is an app-owned temporary folder of the adopted capture.
func (p *DefectPanel) isOwned(dir string) bool {
	for _, d := range p.snapSrc.owned {
		if d == dir {
			return true
		}
	}
	return false
}

// dropSnapDirs removes the app-owned folders of the adopted capture except keep
// (a capture that is being loaded again must survive).
func (p *DefectPanel) dropSnapDirs(keep string) {
	for _, d := range p.snapSrc.owned {
		if d != "" && d != keep {
			os.RemoveAll(d)
		}
	}
	p.snapSrc.owned = nil
	p.snapSrc.workDir = ""
	p.snapSrc.srcDir = ""
}

// Close releases the capture's temporary folders when the program exits.
func (p *DefectPanel) Close() {
	p.resetSnap()
}

// drawSnapSection draws the Camera snap section; it returns true when the panel
// state changed and the caller should re-run.
func (p *DefectPanel) drawSnapSection() bool {
	changed := false
	if !Section(T("defect.section.snap"), "defect-snap") {
		return false
	}
	Note(T("defect.snap.note"))
	exe := findCameraExe()
	if exe == "" {
		Warn(T("defect.snap.noCamera"))
	} else {
		Note(T("defect.snap.camera", "path", exe))
	}
	on := p.snapSrc.on
	if Check(T("defect.snap.use"), "snap-on", &on) {
		switch {
		case p.snapSrc.live:
			p.setLiveFollow(false) // unchecking the live source returns to the simulation
		case on:
			p.clearUserImage()
			p.liveFollow = false // a capture chosen by hand is not replaced by the live image
			if p.snapSrc.meta == nil {
				p.requestSnap()
			} else {
				p.snapSrc.on = true
				p.renderRev++
				changed = true
			}
		default:
			p.setSnapOff()
		}
	}
	enabled := exe != "" && !p.a.Jobs.Busy("snap")
	if ButtonSized(T("defect.snap.button"), "snap-go", imgui.NewVec2(ButtonWidth(T("defect.snap.button")), imgui.FrameHeight()+2), enabled) {
		p.requestSnap()
	}
	if p.snapSrc.pathAuto || strings.TrimSpace(p.snapSrc.path) == "" {
		p.snapSrc.path, p.snapSrc.pathAuto = defaultCaptureDir(p.a.Scene), true
	}
	if InputText(T("defect.snap.path"), "snap-path", &p.snapSrc.path, T("defect.snap.pathHint")) {
		p.snapSrc.pathAuto = false
	}
	if Button(T("defect.snap.load"), "snap-load") {
		p.requestSnapLoad(p.snapSrc.path)
	}
	imgui.SameLine()
	saveLabel := T("defect.snap.save")
	if ButtonSized(saveLabel, "snap-save", imgui.NewVec2(ButtonWidth(saveLabel), imgui.FrameHeight()+2), !p.a.Jobs.Busy("session-save")) {
		p.requestSaveSession()
	}
	Note(T("defect.save.note"))
	if p.snapSrc.meta != nil {
		m := p.snapSrc.meta
		names := make([]string, 0, len(p.snapSrc.frames))
		for k := range p.snapSrc.frames {
			names = append(names, snapfmt.ChannelName(k.ch)+"-"+k.kind)
		}
		sort.Strings(names)
		folder := p.snapSrc.srcDir
		if p.snapSrc.live {
			folder = T("defect.live.srcLive", "rev", fmt.Sprint(p.snapSrc.liveRev))
		}
		rows := [][2]string{
			{T("defect.snap.folder"), folder},
			{T("defect.snap.producer"), m.Producer},
			{T("defect.snap.created"), m.Created},
			{T("defect.snap.model"), m.Model.ID},
			{T("defect.snap.size"), fmt.Sprintf("%dx%d", m.Width, m.Height)},
			{T("defect.snap.images"), strings.Join(names, ", ")},
			{T("defect.snap.defects"), fmt.Sprint(len(m.Recipe.Defects))},
		}
		KVTable("snap-meta", rows)
	}
	if p.snapStale() {
		Warn(T("defect.snap.stale"))
	}
	if p.snapSrc.msgKey != "" && !strings.HasPrefix(p.snapSrc.msgKey, "defect.live.") {
		text := T(p.snapSrc.msgKey, "msg", p.snapSrc.msgText)
		if p.snapSrc.msgErr {
			Warn(text)
		} else {
			Note(text)
		}
	}
	p.inspMsg.draw()
	return changed
}

func init() {
	Messages(map[string]string{
		"defect.section.snap":  "相机抓拍",
		"defect.snap.note":     "用相机程序渲染的抓拍图作为测试图；检出仍只看测试图与材质图。",
		"defect.snap.camera":   "相机程序：{path}",
		"defect.snap.noCamera": "未找到相机程序 camera.exe，请设置 AFVI_CAMERA_EXE 或把它放在本程序旁边。",
		"defect.snap.use":      "使用相机抓拍图",
		"defect.snap.button":   "抓拍",
		"defect.snap.path":     "抓拍路径",
		"defect.snap.pathHint": "抓拍文件夹或 snap.json",
		"defect.snap.load":     "载入",
		"defect.snap.folder":   "文件夹",
		"defect.snap.producer": "生成程序",
		"defect.snap.created":  "生成时间",
		"defect.snap.model":    "型号",
		"defect.snap.size":     "尺寸",
		"defect.snap.images":   "图像",
		"defect.snap.defects":  "缺陷数",
		"defect.snap.stale":    "抓拍后场景已修改，请重新抓拍。",
		"defect.snap.ok":       "抓拍完成：{msg}",
		"defect.snap.loaded":   "已载入抓拍：{msg}",
		"defect.snap.error":    "抓拍失败：{msg}",
		"defect.snap.badPath":  "请输入抓拍文件夹或描述文件路径。",
		"defect.snap.mismatch": "抓拍与当前型号或尺寸不符：{msg}",
	}, map[string]string{
		"defect.section.snap":  "Camera snap",
		"defect.snap.note":     "Use the image rendered by the camera program as the test image; inspection still only sees the test image and the material map.",
		"defect.snap.camera":   "Camera program: {path}",
		"defect.snap.noCamera": "camera.exe was not found; set AFVI_CAMERA_EXE or place it next to this program.",
		"defect.snap.use":      "Use camera snap image",
		"defect.snap.button":   "Snap",
		"defect.snap.path":     "Snap path",
		"defect.snap.pathHint": "capture folder or snap.json",
		"defect.snap.load":     "Load",
		"defect.snap.folder":   "Folder",
		"defect.snap.producer": "Producer",
		"defect.snap.created":  "Created",
		"defect.snap.model":    "Model",
		"defect.snap.size":     "Size",
		"defect.snap.images":   "Images",
		"defect.snap.defects":  "Defects",
		"defect.snap.stale":    "Edited after the snap - snap again.",
		"defect.snap.ok":       "Snap done: {msg}",
		"defect.snap.loaded":   "Capture loaded: {msg}",
		"defect.snap.error":    "Snap failed: {msg}",
		"defect.snap.badPath":  "Enter a capture folder or description file path.",
		"defect.snap.mismatch": "Capture does not match the current model or size: {msg}",
	}, map[string]string{
		"defect.section.snap":  "카메라 캡처",
		"defect.snap.note":     "카메라 프로그램이 렌더링한 캡처 이미지를 테스트 이미지로 사용합니다. 검출은 여전히 테스트 이미지와 재질 맵만 봅니다.",
		"defect.snap.camera":   "카메라 프로그램: {path}",
		"defect.snap.noCamera": "camera.exe 를 찾을 수 없습니다. AFVI_CAMERA_EXE 를 설정하거나 이 프로그램 옆에 두세요.",
		"defect.snap.use":      "카메라 캡처 이미지 사용",
		"defect.snap.button":   "캡처",
		"defect.snap.path":     "캡처 경로",
		"defect.snap.pathHint": "캡처 폴더 또는 snap.json",
		"defect.snap.load":     "불러오기",
		"defect.snap.folder":   "폴더",
		"defect.snap.producer": "생성 프로그램",
		"defect.snap.created":  "생성 시각",
		"defect.snap.model":    "모델",
		"defect.snap.size":     "크기",
		"defect.snap.images":   "이미지",
		"defect.snap.defects":  "결함 수",
		"defect.snap.stale":    "캡처 후 장면이 수정되었습니다. 다시 캡처하세요.",
		"defect.snap.ok":       "캡처 완료: {msg}",
		"defect.snap.loaded":   "캡처 불러옴: {msg}",
		"defect.snap.error":    "캡처 실패: {msg}",
		"defect.snap.badPath":  "캡처 폴더 또는 설명 파일 경로를 입력하세요.",
		"defect.snap.mismatch": "캡처가 현재 모델 또는 크기와 맞지 않습니다: {msg}",
	})
}
