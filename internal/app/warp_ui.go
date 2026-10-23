package app

import (
	"fmt"

	"afvi/teach-imgui/internal/core"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// warpPreset is a named misalignment / lift scenario. Only the geometry fields are used, so
// applying one never touches the photometric part (blur, noise, gain, offset, gradient,
// occlusions) of the warp it is applied to.
type warpPreset struct {
	id   string // widget id and translation suffix
	warp core.WarpParams
}

// warpPresets are the one-click scenarios of the camera studio and the align tab (full-resolution pixels).
func warpPresets() []warpPreset {
	return []warpPreset{
		{"none", core.IdentityWarp},
		{"slight", core.WarpParams{Scale: 1, Gain: 1, Tx: 8, Ty: -5, ThetaDeg: 0.3}},
		{"severe", core.WarpParams{Scale: 1.01, Gain: 1, Tx: 40, Ty: 25, ThetaDeg: 1.5}},
		{"lift", core.WarpParams{Scale: 1, Gain: 1, TiltX: 0.01, TiltY: 0.006, BendAmplitude: 6, BendWavelength: 1500}},
		{"camera", core.DefaultCaptureWarp()},
	}
}

// warpPresetByID returns the geometry of a preset (identity for an unknown id).
func warpPresetByID(id string) core.WarpParams {
	for _, pr := range warpPresets() {
		if pr.id == id {
			return pr.warp
		}
	}
	return core.IdentityWarp
}

// setWarpGeometry copies the geometry of src into dst and leaves the photometric fields alone.
func setWarpGeometry(dst *core.WarpParams, src core.WarpParams) {
	dst.Tx, dst.Ty, dst.ThetaDeg, dst.Scale = src.Tx, src.Ty, src.ThetaDeg, src.Scale
	dst.ShearX, dst.ShearY, dst.TiltX, dst.TiltY = src.ShearX, src.ShearY, src.TiltX, src.TiltY
	dst.BendAmplitude, dst.BendWavelength = src.BendAmplitude, src.BendWavelength
}

// warpSummary is the one-line form of a capture's warp, the same fields camera -info prints.
func warpSummary(w *snapfmt.Warp) string {
	return fmt.Sprintf("dx=%s dy=%s rot=%s° scale=%s shear=(%s,%s) tilt=(%s,%s) bend=%s@%s",
		F(w.Tx, 1), F(w.Ty, 1), F(w.ThetaDeg, 2), F(w.Scale, 3), F(w.ShearX, 3), F(w.ShearY, 3),
		F(w.TiltX, 3), F(w.TiltY, 3), F(w.BendAmplitude, 1), F(w.BendWavelength, 0))
}

// captureWarp is the warp the capture description will carry; nil at the nominal position.
func (p *DefectPanel) captureWarp() *snapfmt.Warp { return snapfmt.Recipe{}.WithWarp(p.warp).Warp }

// warped reports whether the capture is placed away from the nominal position.
func (p *DefectPanel) warped() bool { return !p.warp.GeometryIdentity() }

// drawWarpSection draws the camera studio's "Misalignment and lift" section: where the
// product sits in the capture. It returns true when the capture changed.
func (p *DefectPanel) drawWarpSection() bool {
	if !Section(T("defect.section.warp"), "defect-warp") {
		return false
	}
	if p.scrollToWarp {
		// The browser keeps the section where the user left it; the flag is just consumed.
		p.scrollToWarp = false
	}
	changed := false
	was := p.warped()
	w := &p.warp
	if w.Scale == 0 {
		w.Scale = 1
	}
	if w.BendWavelength < core.MinCaptureBendWave {
		w.BendWavelength = 1500
	}
	slider := func(label, id string, v *float64, lo, hi float64, format string) {
		if SliderF64(label, id, v, lo, hi, format) {
			changed = true
		}
	}
	Note(T("defect.warp.misalign"))
	slider(T("align.param.tx"), "warp-tx", &w.Tx, -200, 200, "%.0f")
	slider(T("align.param.ty"), "warp-ty", &w.Ty, -200, 200, "%.0f")
	slider(T("align.param.theta"), "warp-theta", &w.ThetaDeg, -core.MaxCaptureRotationDeg, core.MaxCaptureRotationDeg, "%.2f")
	slider(T("align.param.scale"), "warp-scale", &w.Scale, core.MinCaptureScale, core.MaxCaptureScale, "%.3f")
	Note(T("defect.warp.lift"))
	slider(T("align.param.tiltX"), "warp-tiltx", &w.TiltX, -0.1, 0.1, "%.4f")
	slider(T("align.param.tiltY"), "warp-tilty", &w.TiltY, -0.1, 0.1, "%.4f")
	slider(T("align.param.bendAmplitude"), "warp-benda", &w.BendAmplitude, -core.MaxCaptureBend, core.MaxCaptureBend, "%.1f")
	slider(T("align.param.bendWavelength"), "warp-bendw", &w.BendWavelength, core.MinCaptureBendWave, 6000, "%.0f")
	slider(T("align.param.shearX"), "warp-shearx", &w.ShearX, -core.MaxCaptureShear, core.MaxCaptureShear, "%.3f")
	slider(T("align.param.shearY"), "warp-sheary", &w.ShearY, -core.MaxCaptureShear, core.MaxCaptureShear, "%.3f")
	for i, pr := range warpPresets() {
		label := T("warp.preset." + pr.id)
		if i > 0 {
			WrapSameLine(label)
		}
		if Button(label, "warp-preset-"+pr.id) {
			setWarpGeometry(w, pr.warp)
			changed = true
		}
	}
	if p.warped() {
		Warn(T("defect.warp.editLocked"))
	}
	Note(T("defect.warp.note"))
	if changed {
		p.renderRev++
		if p.warped() && !was {
			// Painting and selecting work in the product's own coordinates; park the tools.
			p.tool = toolNone
			p.SetPaletteTool(PaletteNone)
		}
	}
	return changed
}

func init() {
	Messages(map[string]string{
		"defect.section.warp":    "错位与翘起",
		"defect.warp.misalign":   "错位：整块产品相对标准位置移动/旋转/缩放。",
		"defect.warp.lift":       "翘起与形变：一侧被压缩另一侧被拉伸（透视倾斜）、波浪弯曲、剪切。",
		"defect.warp.editLocked": "错位/翘起生效时，视图里的绘制与选择已停用：缺陷位置是产品自己的坐标，请先编辑缺陷，最后再设置错位。",
		"defect.warp.note":       "整块产品（含缺陷）一起移动，传感器噪声在其后叠加；良品样本保持标称位置。检出只看设计材质图，带错位的抓拍要先做对齐。",
		"defect.snap.warp":       "错位/翘起",
		"defect.snap.warpNote":   "此抓拍带有错位/翘起：检出仍按设计位置判断，边缘会被当成缺陷；请先在对齐页确认位置。",
		"studio.sum.warp":        "错位/翘起",
		"studio.sum.warpNone":    "无",
		"align.param.tiltX":      "翘起 X（透视）",
		"align.param.tiltY":      "翘起 Y（透视）",
		"warp.preset.none":       "无",
		"warp.preset.slight":     "轻微错位",
		"warp.preset.severe":     "严重错位",
		"warp.preset.lift":       "翘起",
		"warp.preset.camera":     "典型错位（相机默认）",
	}, map[string]string{
		"defect.section.warp":    "Misalignment & Lift",
		"defect.warp.misalign":   "Misalignment: the whole product is shifted, rotated or scaled against its nominal position.",
		"defect.warp.lift":       "Lift and deformation: one side compressed, the other stretched (perspective tilt), wave bend, shear.",
		"defect.warp.editLocked": "While misalignment or lift is active, drawing and selecting in the view are disabled: defects live in the product's own coordinates. Edit the defects first and set the misalignment last.",
		"defect.warp.note":       "The whole product (defects included) moves together and the sensor noise is added afterwards; the good sample stays at the nominal position. Inspection only sees the design material map, so a misaligned capture needs aligning first.",
		"defect.snap.warp":       "Misalignment / lift",
		"defect.snap.warpNote":   "This capture is misaligned or lifted: inspection still judges at the design position, so edges are flagged as defects. Check the position on the Align tab first.",
		"studio.sum.warp":        "Misalignment / lift",
		"studio.sum.warpNone":    "none",
		"align.param.tiltX":      "Lift X (perspective)",
		"align.param.tiltY":      "Lift Y (perspective)",
		"warp.preset.none":       "None",
		"warp.preset.slight":     "Slight shift",
		"warp.preset.severe":     "Severe shift",
		"warp.preset.lift":       "Lift",
		"warp.preset.camera":     "Camera default misalignment",
	}, map[string]string{
		"defect.section.warp":    "오정렬 및 들뜸",
		"defect.warp.misalign":   "오정렬: 제품 전체가 기준 위치에서 이동·회전·확대/축소됩니다.",
		"defect.warp.lift":       "들뜸과 변형: 한쪽은 압축, 반대쪽은 늘어남(원근 기울기), 물결 휨, 전단.",
		"defect.warp.editLocked": "오정렬/들뜸이 켜져 있는 동안 뷰에서의 그리기와 선택은 꺼집니다. 결함 위치는 제품 자체 좌표이므로 결함을 먼저 편집하고 오정렬은 마지막에 설정하세요.",
		"defect.warp.note":       "제품 전체(결함 포함)가 함께 움직이고 센서 노이즈는 그 뒤에 더해집니다. 양품 샘플은 기준 위치에 남습니다. 검사는 설계 재질 맵만 보므로 오정렬된 캡처는 먼저 정렬해야 합니다.",
		"defect.snap.warp":       "오정렬/들뜸",
		"defect.snap.warpNote":   "이 캡처는 오정렬 또는 들뜸이 있습니다. 검사는 여전히 설계 위치 기준이라 가장자리가 결함으로 잡힙니다. 먼저 정렬 탭에서 위치를 확인하세요.",
		"studio.sum.warp":        "오정렬/들뜸",
		"studio.sum.warpNone":    "없음",
		"align.param.tiltX":      "들뜸 X (원근)",
		"align.param.tiltY":      "들뜸 Y (원근)",
		"warp.preset.none":       "없음",
		"warp.preset.slight":     "약간 어긋남",
		"warp.preset.severe":     "크게 어긋남",
		"warp.preset.lift":       "들뜸",
		"warp.preset.camera":     "카메라 기본 어긋남",
	})
}
