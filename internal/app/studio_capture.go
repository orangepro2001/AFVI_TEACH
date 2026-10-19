package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/core"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// studioOutput is the state of the studio's Output section.
type studioOutput struct {
	dir      string        // output folder text field; "" until first drawn
	channels [3]bool       // red, green, blue
	chInit   bool          // channels were defaulted once
	good     bool          // also write the defect-free good samples
	recipe   bool          // write only the description file (snap.json.gz)
	last     *snapfmt.Meta // description of the last saved capture
	lastDir  string        // folder (or file) of the last saved capture
	msgKey   string        // i18n key of the last message ("" = none)
	msgText  string        // {msg} argument of msgKey
	msgErr   bool
}

// defaultCaptureDir is the default output folder: captures/<model id> next to the
// running executable, or a relative path when the executable cannot be located.
func defaultCaptureDir(scene *Scene) string {
	id := "capture"
	if scene != nil && scene.Info.ID != "" {
		id = scene.Info.ID
	}
	id = strings.ReplaceAll(id, "/", "-")
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "captures", id)
	}
	return filepath.Join("captures", id)
}

// drawOutputPanel draws the studio's right-hand Output column: where to write the
// capture, which channels to render, and a summary of the current scenario.
func (p *DefectPanel) drawOutputPanel() {
	scene := p.a.Scene
	if scene == nil {
		return
	}
	if !p.studio.chInit {
		p.studio.channels = [3]bool{true, true, true}
		p.studio.good = true
		p.studio.chInit = true
	}
	if p.studio.dir == "" {
		p.studio.dir = defaultCaptureDir(scene)
	}

	if Section(T("studio.section.output"), "studio-out") {
		Note(T("studio.out.note"))
		exe := findCameraExe()
		if exe == "" {
			Warn(T("studio.out.noCamera"))
		} else {
			Note(T("defect.snap.camera", "path", exe))
		}
		InputText(T("studio.out.dir"), "studio-dir", &p.studio.dir, T("studio.out.dirHint"))
		Check("Red", "studio-ch-r", &p.studio.channels[0])
		Check("Green", "studio-ch-g", &p.studio.channels[1])
		Check("Blue", "studio-ch-b", &p.studio.channels[2])
		Check(T("studio.out.good"), "studio-good", &p.studio.good)
		Check(T("studio.out.recipe"), "studio-recipe", &p.studio.recipe)
		enabled := (exe != "" || p.studio.recipe) && !p.a.Jobs.Busy("studio-save")
		if ButtonSized(T("studio.out.save"), "studio-save", imgui.NewVec2(ButtonWidth(T("studio.out.save")), imgui.FrameHeight()+2), enabled) {
			if !p.studio.recipe && !p.studio.channels[0] && !p.studio.channels[1] && !p.studio.channels[2] {
				p.studio.msgKey, p.studio.msgText, p.studio.msgErr = "studio.out.noChannel", "", true
			} else {
				p.requestSaveCapture()
			}
		}
	}
	if p.studio.msgKey != "" {
		text := T(p.studio.msgKey, "msg", p.studio.msgText)
		if p.studio.msgErr {
			Warn(text)
		} else {
			Note(text)
		}
	}

	p.drawLiveStudio()

	if Section(T("studio.section.summary"), "studio-sum") {
		product := T("studio.sum.productOff")
		if scene.Product.Enabled {
			product = T("studio.sum.productOn", "strength", F(scene.Product.Strength, 2), "seed", fmt.Sprint(scene.Product.Seed))
		}
		var lightParts []string
		for i := 0; i < 3 && i < len(p.a.Light.Channels); i++ {
			name := []string{"R", "G", "B"}[i]
			ch := p.a.Light.Channels[i]
			if !ch.Enable {
				lightParts = append(lightParts, name+" off")
				continue
			}
			lightParts = append(lightParts, name+" "+F(ch.Value, 0))
		}
		warp := T("studio.sum.warpNone")
		if w := p.captureWarp(); w != nil {
			warp = warpSummary(w)
		}
		rows := [][2]string{
			{T("studio.sum.model"), scene.Info.ID},
			{T("studio.sum.size"), fmt.Sprintf("%dx%d", scene.W, scene.H)},
			{T("studio.sum.product"), product},
			{T("studio.sum.light"), strings.Join(lightParts, " / ")},
			{T("studio.sum.noise"), T("studio.sum.noiseValue", "read", F(p.noise.ReadSigma, 1), "shot", F(p.noise.ShotGain, 2), "seed", fmt.Sprint(p.noise.Seed))},
			{T("studio.sum.warp"), warp},
			{T("studio.sum.defects"), fmt.Sprint(len(p.gvStrokes())) + " / " + fmt.Sprint(len(p.strokes))},
		}
		KVTable("studio-kv", rows)
	}

	if p.studio.last != nil && Section(T("studio.section.last"), "studio-last-sec") {
		m := p.studio.last
		names := make([]string, 0, len(m.Images))
		for _, im := range m.Images {
			names = append(names, im.Channel+"-"+im.Kind)
		}
		sort.Strings(names)
		rows := [][2]string{
			{T("defect.snap.folder"), p.studio.lastDir},
			{T("defect.snap.producer"), m.Producer},
			{T("defect.snap.created"), m.Created},
			{T("defect.snap.images"), strings.Join(names, ", ")},
			{T("defect.snap.defects"), fmt.Sprint(len(m.Recipe.Defects))},
		}
		if m.Recipe.Warp != nil {
			rows = append(rows, [2]string{T("defect.snap.warp"), warpSummary(m.Recipe.Warp)})
		}
		KVTable("studio-last", rows)
	}
}

// cameraCapture has the camera program render the description meta into dir (images of the
// given channels, plus the good samples when good is set) and returns the description it
// wrote there. An output folder that already holds the same recipe keeps its earlier images.
func cameraCapture(exe string, meta *snapfmt.Meta, dir string, channels []core.CameraChannel, good bool) (*snapfmt.Meta, error) {
	tmp, err := os.MkdirTemp("", "afvi-studio-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	in := filepath.Join(tmp, snapfmt.MetaFile)
	if err := snapfmt.WriteMeta(in, meta); err != nil {
		return nil, err
	}
	if err := runCamera(exe, in, dir, channels, good); err != nil {
		return nil, err
	}
	saved, _, err := snapfmt.ReadMeta(dir)
	return saved, err
}

// requestSaveCapture writes the current scenario as a capture folder (or only the
// description file) by running the camera program.
func (p *DefectPanel) requestSaveCapture() {
	a := p.a
	scene := a.Scene
	if scene == nil {
		return
	}
	meta := p.snapMeta()
	meta.Producer = studioProducer
	dir := strings.TrimSpace(p.studio.dir)
	if dir == "" {
		dir = defaultCaptureDir(scene)
	}
	var channels []core.CameraChannel
	if p.studio.channels[0] {
		channels = append(channels, core.ChRed)
	}
	if p.studio.channels[1] {
		channels = append(channels, core.ChGreen)
	}
	if p.studio.channels[2] {
		channels = append(channels, core.ChBlue)
	}
	good := p.studio.good
	recipeOnly := p.studio.recipe
	exe := findCameraExe()
	p.studio.msgKey = ""
	a.Jobs.Run("studio-save", func() func() {
		if recipeOnly {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return func() {
					p.studio.msgKey, p.studio.msgText, p.studio.msgErr = "studio.out.error", err.Error(), true
					a.SetHint(T("common.error", "msg", err.Error()))
				}
			}
			target := filepath.Join(dir, snapfmt.MetaFile+".gz")
			if err := snapfmt.WriteMeta(target, &meta); err != nil {
				return func() {
					p.studio.msgKey, p.studio.msgText, p.studio.msgErr = "studio.out.error", err.Error(), true
					a.SetHint(T("common.error", "msg", err.Error()))
				}
			}
			saved := meta
			return func() {
				p.studio.last = &saved
				p.studio.lastDir = target
				p.studio.msgKey, p.studio.msgText, p.studio.msgErr = "studio.out.saved", target, false
			}
		}
		if exe == "" {
			return func() {
				p.studio.msgKey, p.studio.msgText, p.studio.msgErr = "studio.out.noCamera", "", true
			}
		}
		saved, err := cameraCapture(exe, &meta, dir, channels, good)
		if err != nil {
			return func() {
				p.studio.msgKey, p.studio.msgText, p.studio.msgErr = "studio.out.error", err.Error(), true
				a.SetHint(T("common.error", "msg", err.Error()))
			}
		}
		return func() {
			p.studio.last = saved
			p.studio.lastDir = dir
			p.studio.msgKey, p.studio.msgText, p.studio.msgErr = "studio.out.saved", dir, false
		}
	})
}

func init() {
	Messages(map[string]string{
		"studio.section.output":  "输出",
		"studio.section.summary": "场景摘要",
		"studio.section.last":    "最近一次保存",
		"studio.out.note":        "把当前合成场景交给 camera.exe 渲染，写出抓拍文件夹（snap.json 与各通道 PNG），教学程序用「相机抓拍」载入它。",
		"studio.out.dir":         "输出文件夹",
		"studio.out.dirHint":     "抓拍输出文件夹",
		"studio.out.good":        "同时输出良品样本（学习区间用）",
		"studio.out.recipe":      "仅输出描述文件（snap.json.gz，几 KB）",
		"studio.out.save":        "保存抓拍",
		"studio.out.noCamera":    "未找到相机程序 camera.exe，请设置 AFVI_CAMERA_EXE 或把它放在本程序旁边。",
		"studio.out.noChannel":   "请至少选择一个通道。",
		"studio.out.saved":       "已保存：{msg}",
		"studio.out.error":       "保存失败：{msg}",
		"studio.sum.model":       "型号",
		"studio.sum.size":        "尺寸",
		"studio.sum.product":     "虚拟良品",
		"studio.sum.light":       "光源",
		"studio.sum.noise":       "传感器噪声",
		"studio.sum.defects":     "缺陷 启用/总数",
		"studio.sum.productOn":   "强度 {strength}，种子 {seed}",
		"studio.sum.productOff":  "关闭",
		"studio.sum.noiseValue":  "读出 {read}，散粒 {shot}，种子 {seed}",
	}, map[string]string{
		"studio.section.output":  "Output",
		"studio.section.summary": "Scene summary",
		"studio.section.last":    "Last capture",
		"studio.out.note":        "Hands the current synthetic scene to camera.exe and writes a capture folder (snap.json plus one PNG per channel); load it in the teach program with \"Camera snap\".",
		"studio.out.dir":         "Output folder",
		"studio.out.dirHint":     "capture output folder",
		"studio.out.good":        "Also write the defect-free good samples (for learning)",
		"studio.out.recipe":      "Description file only (snap.json.gz, a few KB)",
		"studio.out.save":        "Save capture",
		"studio.out.noCamera":    "camera.exe was not found; set AFVI_CAMERA_EXE or place it next to this program.",
		"studio.out.noChannel":   "Select at least one channel.",
		"studio.out.saved":       "Saved: {msg}",
		"studio.out.error":       "Save failed: {msg}",
		"studio.sum.model":       "Model",
		"studio.sum.size":        "Size",
		"studio.sum.product":     "Virtual good unit",
		"studio.sum.light":       "Light",
		"studio.sum.noise":       "Sensor noise",
		"studio.sum.defects":     "Defects enabled/total",
		"studio.sum.productOn":   "strength {strength}, seed {seed}",
		"studio.sum.productOff":  "off",
		"studio.sum.noiseValue":  "read {read}, shot {shot}, seed {seed}",
	}, map[string]string{
		"studio.section.output":  "출력",
		"studio.section.summary": "장면 요약",
		"studio.section.last":    "최근 저장",
		"studio.out.note":        "현재 합성 장면을 camera.exe 로 렌더링하여 캡처 폴더(snap.json 및 채널별 PNG)를 만듭니다. 교육 프로그램의 \"카메라 캡처\"로 불러오세요.",
		"studio.out.dir":         "출력 폴더",
		"studio.out.dirHint":     "캡처 출력 폴더",
		"studio.out.good":        "양품 샘플도 저장 (학습용)",
		"studio.out.recipe":      "설명 파일만 저장 (snap.json.gz, 수 KB)",
		"studio.out.save":        "캡처 저장",
		"studio.out.noCamera":    "camera.exe 를 찾을 수 없습니다. AFVI_CAMERA_EXE 를 설정하거나 이 프로그램 옆에 두세요.",
		"studio.out.noChannel":   "채널을 하나 이상 선택하세요.",
		"studio.out.saved":       "저장됨: {msg}",
		"studio.out.error":       "저장 실패: {msg}",
		"studio.sum.model":       "모델",
		"studio.sum.size":        "크기",
		"studio.sum.product":     "가상 양품",
		"studio.sum.light":       "조명",
		"studio.sum.noise":       "센서 노이즈",
		"studio.sum.defects":     "결함 사용/전체",
		"studio.sum.productOn":   "강도 {strength}, 시드 {seed}",
		"studio.sum.productOff":  "끄기",
		"studio.sum.noiseValue":  "읽기 {read}, 샷 {shot}, 시드 {seed}",
	})
}
