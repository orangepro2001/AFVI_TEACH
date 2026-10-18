package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/inspect"
	"afvi/teach-imgui/internal/link"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// statusMsg is a translated message that follows language switches: an i18n key and its
// {msg} argument.
type statusMsg struct {
	key  string
	text string
	err  bool
}

func (m statusMsg) draw() {
	if m.key == "" {
		return
	}
	text := T(m.key, "msg", m.text)
	if m.err {
		Warn(text)
	} else {
		Note(text)
	}
}

// sessionDir is the folder "Save" writes and "Load" reads: the path field, which holds
// captures/<model id> next to the executable until the user edits it. A path that names a
// description file stands for the folder that contains it.
func (p *DefectPanel) sessionDir() string {
	dir := strings.TrimSpace(p.snapSrc.path)
	if dir == "" {
		dir = defaultCaptureDir(p.a.Scene)
	}
	if st, err := os.Stat(dir); err == nil && !st.IsDir() {
		dir = filepath.Dir(dir)
	}
	return dir
}

// paramsFile snapshots the inspection parameters of all three channels.
func (p *DefectPanel) paramsFile() *inspect.ParamsFile {
	scene := p.a.Scene
	pf := &inspect.ParamsFile{
		Producer: teachProducer,
		Model:    scene.Info.ID,
		Width:    scene.W,
		Height:   scene.H,
		LearnK:   p.learnK,
		Channels: make(map[string]inspect.ChannelParams, len(p.inspectParams)),
	}
	for i := range p.inspectParams {
		name := snapfmt.ChannelName(core.CameraChannel(i))
		pf.Channels[name] = inspect.ChannelParams{Learned: p.learned[i], Params: p.inspectParams[i]}
	}
	return pf
}

// applyParamsFile installs the parameters of a parameter file. Channels the file does not
// list keep their current parameters; the next run uses them (and re-learns what was not
// learned when the file was written).
func (p *DefectPanel) applyParamsFile(pf *inspect.ParamsFile) {
	for i := range p.inspectParams {
		cp, ok := pf.Channels[snapfmt.ChannelName(core.CameraChannel(i))]
		if !ok {
			continue
		}
		p.inspectParams[i] = cp.Params
		p.learned[i] = cp.Learned
	}
	if pf.LearnK >= 0.5 && pf.LearnK <= 12 {
		p.learnK = pf.LearnK
	}
	p.relearn = false
	p.renderRev++
}

// readSessionParams reads inspect.json from a capture folder. It returns a nil file when
// there is none, or when it belongs to another model or image size; the message says why
// (empty when there is simply no file).
func readSessionParams(dir string, scene *Scene) (*inspect.ParamsFile, statusMsg) {
	path := filepath.Join(dir, inspect.ParamsFileName)
	pf, err := inspect.ReadParamsFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, statusMsg{}
	}
	if err != nil {
		return nil, statusMsg{key: "defect.params.error", text: err.Error(), err: true}
	}
	if (pf.Model != "" && pf.Model != scene.Info.ID) || (pf.Width != 0 && pf.Width != scene.W) || (pf.Height != 0 && pf.Height != scene.H) {
		return nil, statusMsg{key: "defect.params.mismatch", text: pf.Model, err: true}
	}
	return pf, statusMsg{key: "defect.params.loaded", text: path}
}

// saveParamsFile writes inspect.json into the session folder.
func (p *DefectPanel) saveParamsFile() {
	if p.a.Scene == nil {
		return
	}
	path := filepath.Join(p.sessionDir(), inspect.ParamsFileName)
	if err := inspect.WriteParamsFile(path, p.paramsFile()); err != nil {
		p.inspMsg = statusMsg{key: "defect.params.error", text: err.Error(), err: true}
		return
	}
	p.inspMsg = statusMsg{key: "defect.params.saved", text: path}
}

// loadParamsFile reads inspect.json from the session folder; it returns true when the
// parameters were installed and the caller should re-run.
func (p *DefectPanel) loadParamsFile() bool {
	if p.a.Scene == nil {
		return false
	}
	dir := p.sessionDir()
	pf, msg := readSessionParams(dir, p.a.Scene)
	if pf == nil {
		if msg.key == "" {
			msg = statusMsg{key: "defect.params.missing", text: filepath.Join(dir, inspect.ParamsFileName), err: true}
		}
		p.inspMsg = msg
		return false
	}
	p.applyParamsFile(pf)
	p.inspMsg = msg
	return true
}

// saveLive writes the live revision as a capture folder: the test image and the good
// sample of every channel, taken from the frames already received where possible and
// pulled over the link for the rest, then the description file.
func saveLive(dir string, cl *link.Client, rev int64, meta *snapfmt.Meta, have map[snapKey]*core.GrayImage, w, h int) (*snapfmt.Meta, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	out := *meta
	out.Images = []snapfmt.Image{}
	for ch := core.ChRed; ch <= core.ChBlue; ch++ {
		for _, kind := range []string{snapfmt.KindTest, snapfmt.KindGood} {
			img := have[snapKey{ch: ch, kind: kind}]
			if img == nil {
				if cl == nil {
					return nil, errLiveDisconnected
				}
				var err error
				if img, err = fetchLiveImage(cl, rev, ch, kind, w, h); err != nil {
					return nil, err
				}
			}
			frame := snapfmt.Frame{Width: img.Width, Height: img.Height, Pix: img.Data}
			if err := snapfmt.WriteImage(dir, &out, snapfmt.ChannelName(ch), kind, frame); err != nil {
				return nil, err
			}
		}
	}
	// A recipe-only description of an earlier save would shadow nothing but confuse readers.
	_ = os.Remove(filepath.Join(dir, snapfmt.MetaFile+".gz"))
	if err := snapfmt.WriteMeta(filepath.Join(dir, snapfmt.MetaFile), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// requestSaveSession writes what the teaching program is looking at into the session
// folder so the next run can load it: snap.json plus the test and good PNG of every
// channel, and inspect.json with the inspection parameters. The live source writes the
// frames it received; the other sources have camera.exe render the same description.
func (p *DefectPanel) requestSaveSession() {
	a := p.a
	scene := a.Scene
	if scene == nil {
		return
	}
	dir := p.sessionDir()
	pf := p.paramsFile()
	var work func() (*snapfmt.Meta, error)
	if p.user.img != nil {
		work = func() (*snapfmt.Meta, error) { return nil, os.MkdirAll(dir, 0o755) }
	} else if p.snapSrc.live && p.snapSrc.meta != nil {
		meta := *p.snapSrc.meta
		cl, rev := p.liveSt.client, p.snapSrc.liveRev
		have := make(map[snapKey]*core.GrayImage, len(p.snapSrc.frames))
		for k, v := range p.snapSrc.frames {
			have[k] = v
		}
		w, h := scene.W, scene.H
		work = func() (*snapfmt.Meta, error) { return saveLive(dir, cl, rev, &meta, have, w, h) }
	} else {
		exe := findCameraExe()
		if exe == "" {
			p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.snap.noCamera", "", true
			return
		}
		var meta snapfmt.Meta
		if p.snapSrc.on && p.snapSrc.meta != nil {
			meta = *p.snapSrc.meta
		} else {
			meta = p.snapMeta()
		}
		channels := []core.CameraChannel{core.ChRed, core.ChGreen, core.ChBlue}
		work = func() (*snapfmt.Meta, error) { return cameraCapture(exe, &meta, dir, channels, true) }
	}
	p.snapSrc.msgKey = ""
	a.Jobs.Run("session-save", func() func() {
		_, err := work()
		if err == nil {
			err = inspect.WriteParamsFile(filepath.Join(dir, inspect.ParamsFileName), pf)
		}
		if err != nil {
			return func() {
				p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.save.error", err.Error(), true
				a.SetHint(T("common.error", "msg", err.Error()))
			}
		}
		return func() {
			p.snapSrc.msgKey, p.snapSrc.msgText, p.snapSrc.msgErr = "defect.save.ok", dir, false
			p.inspMsg = statusMsg{}
		}
	})
}

func init() {
	Messages(map[string]string{
		"defect.snap.save":       "保存",
		"defect.save.note":       "「保存」把三个通道的测试图和良品图连同检出参数（inspect.json）写进上面的文件夹，下次点「载入」即可读回。",
		"defect.save.ok":         "已保存到：{msg}",
		"defect.save.error":      "保存失败：{msg}",
		"defect.params.saved":    "已保存检出参数：{msg}",
		"defect.params.loaded":   "已载入检出参数：{msg}",
		"defect.params.missing":  "未找到检出参数文件：{msg}",
		"defect.params.error":    "检出参数文件有误：{msg}",
		"defect.params.mismatch": "检出参数文件属于另一个型号或尺寸（{msg}），已忽略。",
		"defect.insp.saveFile":   "保存检出参数",
		"defect.insp.loadFile":   "载入检出参数",
		"defect.insp.fileNote":   "检出参数文件：{path}",
	}, map[string]string{
		"defect.snap.save":       "Save",
		"defect.save.note":       "Save writes the test and good image of all three channels plus the inspection parameters (inspect.json) into the folder above; Load reads them back next time.",
		"defect.save.ok":         "Saved to: {msg}",
		"defect.save.error":      "Save failed: {msg}",
		"defect.params.saved":    "Inspection parameters saved: {msg}",
		"defect.params.loaded":   "Inspection parameters loaded: {msg}",
		"defect.params.missing":  "Inspection parameter file not found: {msg}",
		"defect.params.error":    "Bad inspection parameter file: {msg}",
		"defect.params.mismatch": "The inspection parameter file belongs to another model or size ({msg}) and was ignored.",
		"defect.insp.saveFile":   "Save parameters",
		"defect.insp.loadFile":   "Load parameters",
		"defect.insp.fileNote":   "Parameter file: {path}",
	}, map[string]string{
		"defect.snap.save":       "저장",
		"defect.save.note":       "저장은 세 채널의 테스트 이미지와 양품 이미지, 그리고 검출 파라미터(inspect.json)를 위 폴더에 기록하며 다음에 불러오기로 다시 읽을 수 있습니다.",
		"defect.save.ok":         "저장됨: {msg}",
		"defect.save.error":      "저장 실패: {msg}",
		"defect.params.saved":    "검출 파라미터 저장됨: {msg}",
		"defect.params.loaded":   "검출 파라미터 불러옴: {msg}",
		"defect.params.missing":  "검출 파라미터 파일을 찾을 수 없습니다: {msg}",
		"defect.params.error":    "검출 파라미터 파일 오류: {msg}",
		"defect.params.mismatch": "검출 파라미터 파일이 다른 모델 또는 크기({msg})의 것이어서 무시했습니다.",
		"defect.insp.saveFile":   "파라미터 저장",
		"defect.insp.loadFile":   "파라미터 불러오기",
		"defect.insp.fileNote":   "파라미터 파일: {path}",
	})
}
