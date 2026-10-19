package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"afvi/teach-imgui/internal/core"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// scenarioAutosaveEvery is how often the studio checks whether its settings changed.
const scenarioAutosaveEvery = 500 * time.Millisecond

// scenarioState is the state of the studio's "Scenario file" section: the XML file that
// holds every setting of the studio (light, virtual product, noise, misalignment, all
// strokes, brush and output settings), written whenever something changes and read back
// when the model is loaded again. The teach program reads the same file without camera.exe.
type scenarioState struct {
	init      bool      // defaults were applied once
	path      string    // text field of the file path
	pathAuto  bool      // path still follows the default of the model (the user has not edited it)
	remember  bool      // autosave on change and restore when the model is loaded
	armed     bool      // autosave may write (the restore attempt for the current model has finished)
	created   string    // stable "created" stamp of this session's snapshots
	lastSaved []byte    // XML of the last snapshot that was written or adopted
	lastTick  time.Time // last time the snapshot was compared
	msgKey    string    // i18n key of the last message ("" = none)
	msgText   string    // {msg} argument of msgKey
	msgErr    bool
}

// defaultScenarioPath is the default scenario file: scenario.xml in the model's capture folder.
func defaultScenarioPath(scene *Scene) string {
	return filepath.Join(defaultCaptureDir(scene), snapfmt.ScenarioFile)
}

// ensureScenarioDefaults applies the defaults of the section once.
func (p *DefectPanel) ensureScenarioDefaults() {
	st := &p.studio.scen
	if !st.init {
		st.init = true
		st.remember = p.a.selftest == nil // the self-test starts from the defaults and never touches the remembered file
		st.pathAuto = true
	}
	if st.pathAuto && p.a.Scene != nil {
		st.path = defaultScenarioPath(p.a.Scene)
	}
}

func (st *scenarioState) message(key, text string, isErr bool) {
	st.msgKey, st.msgText, st.msgErr = key, text, isErr
}

// scenarioSnapshot describes everything the studio has set. The stroke list is complete
// (disabled and default strokes included) and keeps the panel order.
func (p *DefectPanel) scenarioSnapshot() *snapfmt.Scenario {
	scene := p.a.Scene
	st := &p.studio.scen
	if st.created == "" {
		st.created = time.Now().UTC().Format(time.RFC3339)
	}
	s := &snapfmt.Scenario{
		Producer: studioProducer,
		Created:  st.created,
		Model:    scene.Info.Ref(),
		Width:    scene.W,
		Height:   scene.H,
		Recipe:   snapfmt.NewRecipe(p.productOpts, p.a.Light.Clone(), p.noise, nil, nil).WithWarp(p.warp),
		Strokes:  make([]snapfmt.StrokeEntry, 0, len(p.strokes)),
	}
	for _, e := range p.strokes {
		switch {
		case e.gv != nil:
			d := snapfmt.WireDefect(*e.gv)
			s.Strokes = append(s.Strokes, snapfmt.StrokeEntry{Enabled: e.enabled, Auto: e.auto, GV: &d})
		case e.scene != nil:
			sc := snapfmt.WireScene(*e.scene)
			s.Strokes = append(s.Strokes, snapfmt.StrokeEntry{Enabled: e.enabled, Auto: e.auto, Scene: &sc})
		}
	}
	s.Studio = &snapfmt.Studio{
		View:      p.viewKind,
		Channel:   snapfmt.ChannelName(p.channel),
		ShowMarks: p.showMarks,
		Tool:      int(p.tool),
		Brush: snapfmt.StudioBrush{
			Radius:            p.radius,
			DeltaGv:           p.deltaGv,
			Softness:          p.softness,
			Amplitude:         int(p.amplitude),
			RelativePct:       p.relativePct,
			TargetBright:      p.targetBright,
			TargetDark:        p.targetDark,
			Profile:           int(p.profile),
			Grain:             p.grain,
			ForeignMaterial:   int(p.foreignMaterial),
			ContaminationFact: p.contaminationFact,
			UseColor:          p.useColor,
			Color:             p.color,
		},
		Random: snapfmt.StudioRandom{
			Count:     p.randomCount,
			Seed:      p.randomSeed,
			Kind:      int(p.randomKind),
			Shape:     int(p.randomShape),
			RadiusMin: p.randomRadiusMin,
			RadiusMax: p.randomRadiusMax,
			ValueMin:  p.randomValueMin,
			ValueMax:  p.randomValueMax,
			Material:  p.randomMaterial,
		},
		Auto: snapfmt.StudioAuto{
			Count:           p.autoCount,
			Seed:            p.autoSeed,
			StrokeSeed:      p.strokeSeed,
			ProductLenScale: p.productLenScale,
			ProductDraw:     p.productDraw,
		},
		Output: snapfmt.StudioOutput{
			Dir:        p.studio.dir,
			Channels:   p.studio.channels,
			Good:       p.studio.good,
			RecipeOnly: p.studio.recipe,
		},
	}
	return s
}

// readScenarioPath reads a scenario from an XML file (plain or gzip), a folder (its
// scenario.xml, else its snap.json) or a JSON capture description; a description has no
// stroke flags, so every stroke counts as enabled.
func readScenarioPath(path string) (*snapfmt.Scenario, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		if _, err := os.Stat(filepath.Join(path, snapfmt.ScenarioFile)); err == nil {
			return snapfmt.ReadScenario(path)
		}
		meta, _, err := snapfmt.ReadMeta(path)
		if err != nil {
			return nil, err
		}
		return snapfmt.ScenarioFromMeta(*meta), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s, xerr := snapfmt.UnmarshalScenario(data)
	if xerr == nil {
		return s, nil
	}
	meta, jerr := snapfmt.Unmarshal(data)
	if jerr != nil {
		return nil, xerr
	}
	return snapfmt.ScenarioFromMeta(*meta), nil
}

// applyScenario adopts s on the UI thread. It reports false (and why) when s belongs to a
// different model or image size.
func (p *DefectPanel) applyScenario(s *snapfmt.Scenario) (bool, string) {
	a := p.a
	scene := a.Scene
	if scene == nil {
		return false, ""
	}
	if s.Model.ID != scene.Info.ID || s.Width != scene.W || s.Height != scene.H {
		return false, s.Model.ID
	}
	meta := s.Meta()
	cr := meta.Recipe.Core()

	a.Light = cr.Light.Clone()
	p.noise = cr.Noise
	p.warp = cr.Warp
	if cr.Product != p.productOpts || cr.Product != scene.Product {
		p.productOpts = cr.Product
		a.ApplyProduct(cr.Product)
	}

	p.strokes = p.strokes[:0]
	for _, e := range s.Strokes {
		switch {
		case e.GV != nil:
			d := snapfmt.CoreDefect(*e.GV)
			p.strokes = append(p.strokes, strokeEntry{gv: &d, enabled: e.Enabled, auto: e.Auto})
		case e.Scene != nil:
			sc := snapfmt.CoreScene(*e.Scene)
			p.strokes = append(p.strokes, strokeEntry{scene: &sc, enabled: e.Enabled, auto: e.Auto})
		}
	}
	p.clearSelection()

	if st := s.Studio; st != nil {
		switch st.View {
		case "test", "golden", "diff":
			p.viewKind = st.View
		}
		if ch, err := snapfmt.ParseChannel(st.Channel); err == nil {
			p.channel = ch
		}
		p.showMarks = st.ShowMarks
		if t := defectTool(st.Tool); t >= toolNone && t <= toolContamination {
			p.tool = t
		}
		b := st.Brush
		if b.Radius > 0 {
			p.radius, p.deltaGv, p.softness = b.Radius, b.DeltaGv, b.Softness
			p.amplitude = core.DefectAmplitude(b.Amplitude)
			p.relativePct, p.targetBright, p.targetDark = b.RelativePct, b.TargetBright, b.TargetDark
			p.profile = core.DefectProfile(b.Profile)
			p.grain = b.Grain
			p.foreignMaterial = uint8(b.ForeignMaterial)
			p.contaminationFact = b.ContaminationFact
			p.useColor, p.color = b.UseColor, b.Color
		}
		if r := st.Random; r.Count > 0 || r.RadiusMax > 0 {
			p.randomCount, p.randomSeed = r.Count, r.Seed
			p.randomKind, p.randomShape = core.RandomKind(r.Kind), core.RandomShape(r.Shape)
			p.randomRadiusMin, p.randomRadiusMax = r.RadiusMin, r.RadiusMax
			p.randomValueMin, p.randomValueMax = r.ValueMin, r.ValueMax
			p.randomMaterial = r.Material
		}
		if au := st.Auto; au.ProductLenScale > 0 {
			p.autoCount, p.autoSeed, p.strokeSeed = au.Count, au.Seed, au.StrokeSeed
			p.productLenScale, p.productDraw = au.ProductLenScale, au.ProductDraw
		}
		if o := st.Output; o.Dir != "" {
			p.studio.dir = o.Dir
			p.studio.channels = o.Channels
			p.studio.good = o.Good
			p.studio.recipe = o.RecipeOnly
			p.studio.chInit = true
		}
	}
	if s.Created != "" {
		p.studio.scen.created = s.Created
	}
	p.renderRev++
	a.NotifyLightChanged()
	if p.active {
		p.requestRun()
	}
	return true, ""
}

// saveScenario writes the current settings to path.
func (p *DefectPanel) saveScenario(path string) error {
	if p.a.Scene == nil {
		return fmt.Errorf("no model loaded")
	}
	s := p.scenarioSnapshot()
	data, err := snapfmt.MarshalScenario(s)
	if err != nil {
		return err
	}
	if err := snapfmt.WriteScenario(path, s); err != nil {
		return err
	}
	p.studio.scen.lastSaved = data
	return nil
}

// loadScenario reads path and adopts it; the returned key/text describe the outcome.
func (p *DefectPanel) loadScenario(path string) (key, text string, isErr bool) {
	s, err := readScenarioPath(path)
	if err != nil {
		return "studio.scen.error", err.Error(), true
	}
	ok, model := p.applyScenario(s)
	if !ok {
		return "studio.scen.mismatch", model, true
	}
	return "studio.scen.loaded", path, false
}

// scenarioModelLoaded is called at the end of ModelLoaded (studio): it restores the model's
// saved scenario when "remember" is on and arms the autosave.
func (p *DefectPanel) scenarioModelLoaded() {
	if p.mode != modeAuthor || p.a == nil || p.a.Scene == nil {
		return
	}
	st := &p.studio.scen
	st.armed = false
	st.lastSaved = nil
	p.ensureScenarioDefaults()
	if st.remember {
		if _, err := os.Stat(st.path); err == nil {
			key, text, isErr := p.loadScenario(st.path)
			if key == "studio.scen.loaded" {
				key = "studio.scen.restored"
			}
			st.message(key, text, isErr)
		}
	}
	if data, err := snapfmt.MarshalScenario(p.scenarioSnapshot()); err == nil {
		st.lastSaved = data
	}
	st.armed = true
}

// tickScenario writes the settings when they changed (checked twice a second).
func (p *DefectPanel) tickScenario() {
	st := &p.studio.scen
	if p.mode != modeAuthor || !st.init || !st.remember || !st.armed || p.a.Scene == nil {
		return
	}
	now := time.Now()
	if now.Sub(st.lastTick) < scenarioAutosaveEvery {
		return
	}
	st.lastTick = now
	p.autosaveScenario()
}

// autosaveScenario writes the settings to the scenario file when they differ from the last
// snapshot that was written or adopted.
func (p *DefectPanel) autosaveScenario() {
	st := &p.studio.scen
	s := p.scenarioSnapshot()
	data, err := snapfmt.MarshalScenario(s)
	if err != nil {
		return
	}
	if bytes.Equal(data, st.lastSaved) {
		return
	}
	path := strings.TrimSpace(st.path)
	if path == "" {
		return
	}
	// lastSaved is only updated by a successful write, so a failed write is retried at the next tick.
	if err := snapfmt.WriteScenario(path, s); err != nil {
		st.message("studio.scen.error", err.Error(), true)
		return
	}
	st.lastSaved = data
	st.message("studio.scen.autosaved", path, false)
}

// scenarioClose is called when the panel is closed: a last autosave.
func (p *DefectPanel) scenarioClose() {
	st := &p.studio.scen
	if p.mode != modeAuthor || !st.init || !st.remember || !st.armed || p.a == nil || p.a.Scene == nil {
		return
	}
	p.autosaveScenario()
}

// drawScenarioSection draws the studio's "Scenario file" section of the Output column.
func (p *DefectPanel) drawScenarioSection() {
	p.ensureScenarioDefaults()
	st := &p.studio.scen
	if Section(T("studio.section.scenario"), "studio-scen") {
		Note(T("studio.scen.note"))
		if InputText(T("studio.scen.path"), "studio-scen-path", &st.path, T("studio.scen.pathHint")) {
			st.pathAuto = false
		}
		Check(T("studio.scen.remember"), "studio-scen-remember", &st.remember)
		if Button(T("studio.scen.save"), "studio-scen-save") {
			path := strings.TrimSpace(st.path)
			if err := p.saveScenario(path); err != nil {
				st.message("studio.scen.error", err.Error(), true)
			} else {
				st.message("studio.scen.saved", path, false)
			}
		}
		WrapSameLine(T("studio.scen.load"))
		if Button(T("studio.scen.load"), "studio-scen-load") {
			key, text, isErr := p.loadScenario(strings.TrimSpace(st.path))
			st.message(key, text, isErr)
		}
	}
	if st.msgKey != "" {
		text := T(st.msgKey, "msg", st.msgText)
		if st.msgErr {
			Warn(text)
		} else {
			Note(text)
		}
	}
}

func init() {
	Messages(map[string]string{
		"studio.section.scenario": "场景文件（XML）",
		"studio.scen.note":        "把光照混合、虚拟良品、噪声、错位与翘起、全部缺陷笔画（含禁用和默认的）以及笔刷、输出等界面设置高密度地存成一个 XML 文件，下次直接读取。教学程序不启动 camera.exe 也能读取它（「相机抓拍」→ 路径 → 载入）。",
		"studio.scen.path":        "场景文件",
		"studio.scen.pathHint":    "场景 XML 文件（.xml / .xml.gz）",
		"studio.scen.remember":    "自动保存，并在载入型号时自动恢复",
		"studio.scen.save":        "保存场景",
		"studio.scen.load":        "载入场景",
		"studio.scen.saved":       "已保存场景：{msg}",
		"studio.scen.autosaved":   "已自动保存：{msg}",
		"studio.scen.loaded":      "已载入场景：{msg}",
		"studio.scen.restored":    "已恢复上次的场景：{msg}",
		"studio.scen.error":       "场景文件出错：{msg}",
		"studio.scen.mismatch":    "该场景属于型号 {msg} 或图像尺寸不同，未载入。",
	}, map[string]string{
		"studio.section.scenario": "Scenario file (XML)",
		"studio.scen.note":        "Saves the light mix, virtual good unit, noise, misalignment and lift, every defect stroke (disabled and default ones included) and the brush and output settings as one dense XML file that is read back next time. The teach program reads it without camera.exe (\"Camera snap\" -> path -> Load).",
		"studio.scen.path":        "Scenario file",
		"studio.scen.pathHint":    "scenario XML file (.xml / .xml.gz)",
		"studio.scen.remember":    "Autosave, and restore when the model is loaded",
		"studio.scen.save":        "Save scenario",
		"studio.scen.load":        "Load scenario",
		"studio.scen.saved":       "Scenario saved: {msg}",
		"studio.scen.autosaved":   "Autosaved: {msg}",
		"studio.scen.loaded":      "Scenario loaded: {msg}",
		"studio.scen.restored":    "Restored the last scenario: {msg}",
		"studio.scen.error":       "Scenario file error: {msg}",
		"studio.scen.mismatch":    "The scenario belongs to model {msg} or a different image size and was not loaded.",
	}, map[string]string{
		"studio.section.scenario": "시나리오 파일 (XML)",
		"studio.scen.note":        "조명 혼합, 가상 양품, 노이즈, 오정렬 및 들뜸, 모든 결함 획(비활성 및 기본 포함)과 브러시·출력 설정을 하나의 조밀한 XML 파일로 저장하고 다음에 바로 읽어 옵니다. 교육 프로그램은 camera.exe 없이도 읽을 수 있습니다(\"카메라 캡처\" → 경로 → 불러오기).",
		"studio.scen.path":        "시나리오 파일",
		"studio.scen.pathHint":    "시나리오 XML 파일 (.xml / .xml.gz)",
		"studio.scen.remember":    "자동 저장, 모델을 불러올 때 자동 복원",
		"studio.scen.save":        "시나리오 저장",
		"studio.scen.load":        "시나리오 불러오기",
		"studio.scen.saved":       "시나리오 저장됨: {msg}",
		"studio.scen.autosaved":   "자동 저장됨: {msg}",
		"studio.scen.loaded":      "시나리오 불러옴: {msg}",
		"studio.scen.restored":    "마지막 시나리오를 복원했습니다: {msg}",
		"studio.scen.error":       "시나리오 파일 오류: {msg}",
		"studio.scen.mismatch":    "이 시나리오는 모델 {msg} 의 것이거나 이미지 크기가 달라 불러오지 않았습니다.",
	})
}
