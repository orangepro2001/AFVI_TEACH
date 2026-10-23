package app

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"

	"afvi/teach-imgui/internal/core"
)

// userImage is a test image loaded from a file (Load Image). It replaces every other
// source (live studio, camera snap, local simulation) until the user leaves it.
type userImage struct {
	img  *core.GrayImage
	path string
	dir  string // folder of the last loaded file, start folder of the dialog
}

// errNoFileDialog is returned by pickImageFile on platforms without a native dialog.
var errNoFileDialog = errors.New("no file dialog on this platform")

// readGrayImage decodes path (PNG, JPEG, BMP or TIFF) into an 8-bit gray image.
func readGrayImage(path string) (*core.GrayImage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	defer f.Close()
	src, _, err := image.Decode(bufio.NewReaderSize(f, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("%s: empty image", filepath.Base(path))
	}
	out := &core.GrayImage{Width: w, Height: h, Data: make([]uint8, w*h)}
	switch g := src.(type) {
	case *image.Gray:
		for y := 0; y < h; y++ {
			src := g.PixOffset(b.Min.X, b.Min.Y+y)
			copy(out.Data[y*w:(y+1)*w], g.Pix[src:src+w])
		}
	case *image.Gray16:
		for y := 0; y < h; y++ {
			row := y * w
			base := g.PixOffset(b.Min.X, b.Min.Y+y)
			for x := 0; x < w; x++ {
				out.Data[row+x] = g.Pix[base+2*x]
			}
		}
	default:
		for y := 0; y < h; y++ {
			row := y * w
			for x := 0; x < w; x++ {
				c := color.GrayModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.Gray)
				out.Data[row+x] = c.Y
			}
		}
	}
	return out, nil
}

// clearUserImage drops the loaded image but keeps the dialog's start folder.
func (p *DefectPanel) clearUserImage() {
	p.user = userImage{dir: p.user.dir}
}

// requestInspect learns (when needed) and inspects the current test image now.
func (p *DefectPanel) requestInspect() {
	if p.mode != modeDetect || p.a.Scene == nil {
		return
	}
	p.inspectNow = true
	p.requestRun()
}

// requestLoadImage loads path as the test image of the defect tab.
func (p *DefectPanel) requestLoadImage(path string) {
	if p.mode != modeDetect || p.a.Scene == nil {
		return
	}
	path = strings.TrimSpace(path)
	if path == "" {
		p.imgMsg = statusMsg{key: "defect.manual.loadError", text: "empty path", err: true}
		return
	}
	a := p.a
	scene := a.Scene
	w, h := scene.W, scene.H
	a.Jobs.Run("load-image", func() func() {
		img, err := readGrayImage(path)
		if err != nil {
			return func() {
				p.imgMsg = statusMsg{key: "defect.manual.loadError", text: err.Error(), err: true}
			}
		}
		if img.Width != w || img.Height != h {
			return func() {
				p.imgMsg = statusMsg{
					key:  "defect.manual.sizeMismatch",
					text: fmt.Sprintf("%dx%d (%dx%d)", img.Width, img.Height, w, h),
					err:  true,
				}
			}
		}
		return func() {
			if a.Scene != scene {
				return
			}
			p.leaveLive()
			p.snapSrc.on = false
			p.user = userImage{img: img, path: path, dir: filepath.Dir(path)}
			p.raw, p.aligned, p.alignOut = nil, nil, nil
			p.golden, p.test = nil, nil
			p.lastReport, p.lastProbe, p.probeRect = nil, nil, nil
			p.renderRev++
			p.imgMsg = statusMsg{key: "defect.manual.loaded", text: filepath.Base(path)}
			p.requestRun()
		}
	})
}

// pickAndLoadImage opens the native open dialog and loads the chosen image.
func (p *DefectPanel) pickAndLoadImage() {
	path, err := pickImageFile(p.user.dir)
	switch {
	case errors.Is(err, errNoFileDialog):
		p.imgMsg = statusMsg{key: "defect.manual.noDialog", err: true}
		return
	case err != nil:
		p.imgMsg = statusMsg{key: "defect.manual.dialogError", text: err.Error(), err: true}
		return
	}
	if strings.TrimSpace(path) == "" {
		return
	}
	p.requestLoadImage(path)
}

// drawManualSection draws the teaching program's Manual control section.
func (p *DefectPanel) drawManualSection() {
	if !Section(T("defect.section.manual"), "defect-manual") {
		return
	}
	manual := p.manual
	if Check(T("defect.manual.mode"), "manual-mode", &manual) {
		p.manual = manual
		if !manual {
			p.requestRun()
		}
	}
	Note(T("defect.manual.note"))
	loadLabel := T("defect.manual.loadImage")
	if ButtonOn(loadLabel, "manual-load", !p.a.Jobs.Busy("load-image")) {
		p.pickAndLoadImage()
	}
	SameLine()
	inspectLabel := T("defect.manual.inspect")
	if ButtonOn(inspectLabel, "manual-inspect", p.a.Scene != nil && !p.a.Jobs.Busy("defect")) {
		p.requestInspect()
	}
	if p.user.img != nil {
		Note(T("defect.manual.srcImage", "name", fmt.Sprintf("%s (%dx%d)", filepath.Base(p.user.path), p.user.img.Width, p.user.img.Height)))
		if Button(T("defect.manual.clear"), "manual-clear") {
			p.clearUserImage()
			p.renderRev++
			p.golden, p.test = nil, nil
			p.imgMsg = statusMsg{}
			p.requestRun()
		}
		Note(T("defect.manual.learnNote"))
		if p.viewKind != "test" {
			Note(T("defect.manual.noGolden"))
		}
	}
	if p.lastReport == nil {
		Note(T("defect.manual.pending"))
	} else {
		Note(T("defect.manual.done", "msg", fmt.Sprint(len(p.lastReport.Defects))))
	}
	p.imgMsg.draw()
}

func init() {
	Messages(map[string]string{
		"defect.section.manual":      "手动控制",
		"defect.manual.mode":         "手动模式（改动后不自动检出）",
		"defect.manual.note":         "手动模式下，换通道、改光照、改参数或换图只刷新测试图并清除旧结果；按「检出」(F5) 才学习并检出。",
		"defect.manual.loadImage":    "载入图像…",
		"defect.manual.inspect":      "检出",
		"defect.manual.clear":        "改用仿真",
		"defect.manual.srcImage":     "载入的图像：{name}",
		"defect.manual.learnNote":    "载入的图像没有良品样本：首次检出（或重新学习区间）用它自己学习区间。先载入良品图检出一次，再载入待测图。",
		"defect.manual.noGolden":     "载入的图像没有标准图，标准图/差异图视图显示测试图。",
		"defect.manual.pending":      "尚未检出，按「检出」(F5)。",
		"defect.manual.done":         "已检出：{msg} 个缺陷",
		"defect.manual.loaded":       "已载入图像：{msg}",
		"defect.manual.loadError":    "载入图像失败：{msg}",
		"defect.manual.sizeMismatch": "图像尺寸与当前型号不符：{msg}",
		"defect.manual.noDialog":     "此平台没有文件对话框。",
		"defect.manual.dialogError":  "文件对话框出错：{msg}",
	}, map[string]string{
		"defect.section.manual":      "Manual control",
		"defect.manual.mode":         "Manual mode (no automatic inspection)",
		"defect.manual.note":         "In manual mode, changing the channel, light, parameters or image only refreshes the test image and clears the old result; press Inspect (F5) to learn and inspect.",
		"defect.manual.loadImage":    "Load Image...",
		"defect.manual.inspect":      "Inspect",
		"defect.manual.clear":        "Use simulation",
		"defect.manual.srcImage":     "Loaded image: {name}",
		"defect.manual.learnNote":    "A loaded image has no good sample: the first inspection (or Relearn) learns the intervals from the image itself. Load a good image and inspect it once, then load the image to test.",
		"defect.manual.noGolden":     "A loaded image has no golden image; the golden and diff views show the test image.",
		"defect.manual.pending":      "Not inspected yet - press Inspect (F5).",
		"defect.manual.done":         "Inspected: {msg} defect(s)",
		"defect.manual.loaded":       "Image loaded: {msg}",
		"defect.manual.loadError":    "Loading the image failed: {msg}",
		"defect.manual.sizeMismatch": "Image size does not match the model: {msg}",
		"defect.manual.noDialog":     "No file dialog on this platform.",
		"defect.manual.dialogError":  "The file dialog failed: {msg}",
	}, map[string]string{
		"defect.section.manual":      "수동 제어",
		"defect.manual.mode":         "수동 모드 (변경 시 자동 검사 안 함)",
		"defect.manual.note":         "수동 모드에서는 채널·조명·파라미터·이미지를 바꿔도 테스트 이미지만 갱신하고 이전 결과를 지웁니다. 검사(F5)를 눌러야 학습하고 검사합니다.",
		"defect.manual.loadImage":    "이미지 불러오기...",
		"defect.manual.inspect":      "검사",
		"defect.manual.clear":        "시뮬레이션 사용",
		"defect.manual.srcImage":     "불러온 이미지: {name}",
		"defect.manual.learnNote":    "불러온 이미지에는 양품 샘플이 없으므로 첫 검사(또는 구간 재학습) 때 이미지 자체로 구간을 학습합니다. 양품 이미지를 먼저 불러와 한 번 검사한 뒤 테스트할 이미지를 불러오세요.",
		"defect.manual.noGolden":     "불러온 이미지에는 표준 이미지가 없어 표준/차이 보기에도 테스트 이미지를 표시합니다.",
		"defect.manual.pending":      "아직 검사하지 않았습니다. 검사(F5)를 누르세요.",
		"defect.manual.done":         "검사 완료: 결함 {msg}개",
		"defect.manual.loaded":       "이미지 불러옴: {msg}",
		"defect.manual.loadError":    "이미지 불러오기 실패: {msg}",
		"defect.manual.sizeMismatch": "이미지 크기가 현재 모델과 맞지 않습니다: {msg}",
		"defect.manual.noDialog":     "이 플랫폼에는 파일 대화상자가 없습니다.",
		"defect.manual.dialogError":  "파일 대화상자 오류: {msg}",
	})
}
