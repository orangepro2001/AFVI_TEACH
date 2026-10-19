package core

import (
	"fmt"
	"math"
)

// 拍摄时产品相对相机的位置与形变（错位、翘起）的取值范围。
const (
	MaxCaptureShift       = 2000.0 // Tx、Ty 的绝对值上限（像素）
	MaxCaptureRotationDeg = 15.0   // ThetaDeg 的绝对值上限（度）
	MinCaptureScale       = 0.8
	MaxCaptureScale       = 1.2
	MaxCaptureShear       = 0.2 // ShearX、ShearY 的绝对值上限
	MaxCaptureTilt        = 0.5 // TiltX、TiltY 的绝对值上限，且二者绝对值之和须小于 MaxCaptureTiltSum
	MaxCaptureTiltSum     = 0.9
	MaxCaptureBend        = 200.0 // BendAmplitude 的绝对值上限（像素）
	MinCaptureBendWave    = 50.0  // BendWavelength 的下限（像素），弯曲振幅非 0 时有效
)

// GeometryIdentity 报告 p 的几何部分（错位、旋转、缩放、切变、翘起、弯曲）是否什么都不做。
// 光度与噪声部分（模糊、噪声、增益、偏移、渐变、遮挡）不属于拍摄几何，这里不看。
// Scale 为 0 视为 1（零值的 WarpParams 即恒等）。
func (p WarpParams) GeometryIdentity() bool {
	return p.Tx == 0 && p.Ty == 0 && p.ThetaDeg == 0 && (p.Scale == 1 || p.Scale == 0) &&
		p.ShearX == 0 && p.ShearY == 0 && p.TiltX == 0 && p.TiltY == 0 &&
		(p.BendAmplitude == 0 || p.BendWavelength <= 0)
}

// Geometry 只保留 p 的几何部分，其余清零（Gain 为 1、Scale 为 0 时取 1）；
// 弯曲振幅为 0 或波长无效时弯曲两项都清零，所以相等的几何得到相等的结果。
func (p WarpParams) Geometry() WarpParams {
	g := WarpParams{
		Tx: p.Tx, Ty: p.Ty, ThetaDeg: p.ThetaDeg, Scale: p.Scale,
		ShearX: p.ShearX, ShearY: p.ShearY, TiltX: p.TiltX, TiltY: p.TiltY,
		Gain: 1,
	}
	if g.Scale == 0 {
		g.Scale = 1
	}
	if p.BendAmplitude != 0 && p.BendWavelength > 0 {
		g.BendAmplitude = p.BendAmplitude
		g.BendWavelength = p.BendWavelength
	}
	return g
}

// ValidateCaptureWarp 检查 p 的几何部分是否在拍摄范围内，错误信息带参数名。
func ValidateCaptureWarp(p WarpParams) error {
	for _, f := range []struct {
		name string
		v    float64
	}{
		{"tx", p.Tx}, {"ty", p.Ty}, {"theta", p.ThetaDeg}, {"scale", p.Scale},
		{"shearX", p.ShearX}, {"shearY", p.ShearY}, {"tiltX", p.TiltX}, {"tiltY", p.TiltY},
		{"bendAmplitude", p.BendAmplitude}, {"bendWavelength", p.BendWavelength},
	} {
		if math.IsNaN(f.v) || math.IsInf(f.v, 0) {
			return fmt.Errorf("warp %s is not a finite number", f.name)
		}
	}
	if math.Abs(p.Tx) > MaxCaptureShift || math.Abs(p.Ty) > MaxCaptureShift {
		return fmt.Errorf("warp offset (%v, %v) is outside ±%v px", p.Tx, p.Ty, MaxCaptureShift)
	}
	if math.Abs(p.ThetaDeg) > MaxCaptureRotationDeg {
		return fmt.Errorf("warp rotation %v is outside ±%v degrees", p.ThetaDeg, MaxCaptureRotationDeg)
	}
	if p.Scale != 0 && (p.Scale < MinCaptureScale || p.Scale > MaxCaptureScale) {
		return fmt.Errorf("warp scale %v is outside %v..%v", p.Scale, MinCaptureScale, MaxCaptureScale)
	}
	if math.Abs(p.ShearX) > MaxCaptureShear || math.Abs(p.ShearY) > MaxCaptureShear {
		return fmt.Errorf("warp shear (%v, %v) is outside ±%v", p.ShearX, p.ShearY, MaxCaptureShear)
	}
	if math.Abs(p.TiltX) > MaxCaptureTilt || math.Abs(p.TiltY) > MaxCaptureTilt ||
		math.Abs(p.TiltX)+math.Abs(p.TiltY) >= MaxCaptureTiltSum {
		return fmt.Errorf("warp tilt (%v, %v) is outside ±%v each or its absolute sum reaches %v", p.TiltX, p.TiltY, MaxCaptureTilt, MaxCaptureTiltSum)
	}
	if math.Abs(p.BendAmplitude) > MaxCaptureBend {
		return fmt.Errorf("warp bend amplitude %v is outside ±%v px", p.BendAmplitude, MaxCaptureBend)
	}
	if p.BendAmplitude != 0 && p.BendWavelength < MinCaptureBendWave {
		return fmt.Errorf("warp bend wavelength %v is below %v px", p.BendWavelength, MinCaptureBendWave)
	}
	return nil
}

// AddSensorNoise 返回叠加了传感器噪声的新图：标准差 sqrt(ReadSigma² + ShotGain·v)，v 为该像素的灰度，
// 随机数种子为 noise.Seed，按行优先顺序消耗；噪声不起作用时返回 img 的副本。
// 它用于已经做过几何变换的图：噪声在变换之后加，才不会被重采样抹平。
func AddSensorNoise(img *GrayImage, noise SensorNoise) *GrayImage {
	out := img.Clone()
	if noise.ReadSigma <= 0 && noise.ShotGain <= 0 {
		return out
	}
	rng := NewRng(noise.Seed)
	for i, v := range img.Data {
		fv := float64(v)
		sigma := math.Sqrt(noise.ReadSigma*noise.ReadSigma + noise.ShotGain*math.Max(fv, 0))
		out.Data[i] = ClampByte(fv + sigma*rng.NextGaussian())
	}
	return out
}

// RenderDefectPairWarped 与 RenderDefectPair 相同，但整个产品（golden 与 test，连同其上的不良）
// 再按 w 的几何部分做一次错位/翘起变换，传感器噪声在变换之后叠加到 test 上。
// w 的几何部分是恒等时，返回的图与 RenderDefectPair 逐字节相同。变换后移出视野的区域为黑色。
func RenderDefectPairWarped(scene *LightScene, s *LightSetting, ch CameraChannel, sceneStrokes []SceneStroke, gvStrokes []DefectStroke, noise SensorNoise, w WarpParams) (golden, test *GrayImage) {
	if w.GeometryIdentity() {
		return RenderDefectPair(scene, s, ch, sceneStrokes, gvStrokes, noise)
	}
	golden0, test0 := RenderDefectPair(scene, s, ch, sceneStrokes, gvStrokes, SensorNoise{Seed: noise.Seed})
	geo := w.Geometry()
	golden = ApplyWarp(golden0, &geo, 0)
	test = AddSensorNoise(ApplyWarp(test0, &geo, 0), noise)
	return golden, test
}
