package core

import (
	"math"
)

// DefectKind 是缺陷类型。
type DefectKind uint8

// 缺陷类型常量。
const (
	Bright DefectKind = iota
	Dark
)

// DefectShape 是缺陷形状。
type DefectShape uint8

// 缺陷形状常量。
const (
	ShapeSpot DefectShape = iota
	ShapeLine
	ShapeRect
	ShapeEllipse
	ShapePolygon
)

// DefectAmplitude 是幅度模式。
type DefectAmplitude uint8

// 幅度模式常量。
const (
	AmpAbsolute DefectAmplitude = iota
	AmpRelative
	AmpTarget
)

// DefectProfile 是轮廓模式。
type DefectProfile uint8

// 轮廓模式常量。
const (
	ProfileLinear DefectProfile = iota
	ProfileGaussian
)

// DefaultRelativePct 是相对幅度默认百分比。
const DefaultRelativePct = 30.0

// DefectStroke 是灰度缺陷笔画。
type DefectStroke struct {
	Kind         DefectKind
	Shape        DefectShape
	X, Y, X2, Y2 float64
	Radius       float64
	DeltaGv      float64
	Softness     float64
	Amplitude    DefectAmplitude
	RelativePct  float64
	TargetGv     float64
	HasTarget    bool
	Profile      DefectProfile
	Grain        float64
	Seed         uint32
	// Points 是多边形顶点（图像坐标，隐式闭合），仅 ShapePolygon 使用。
	Points []Point
	// HasColor 为 true 时按 Color 对相机通道加权。
	HasColor bool
	// Color 是 [R,G,B] 通道权重，索引为 CameraChannel，取值 0..1。
	Color [3]float64
}

// ChannelFactor 返回该笔画在指定相机通道上的权重因子。
// 未设置颜色时返回 1；否则返回 Color[ch] 并钳到 [0,1]（NaN→0，越界 ch→1）。
func (s DefectStroke) ChannelFactor(ch CameraChannel) float64 {
	if !s.HasColor {
		return 1
	}
	if ch < 0 || int(ch) >= len(s.Color) {
		return 1
	}
	v := s.Color[ch]
	if math.IsNaN(v) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// SceneMode 是场景笔画模式。
type SceneMode uint8

// 场景笔画模式常量。
const (
	SceneMaterial SceneMode = iota
	SceneTexture
)

// SceneStroke 是场景笔画。
type SceneStroke struct {
	Mode                 SceneMode
	Shape                DefectShape
	X, Y, X2, Y2, Radius float64
	Material             uint8
	Factor               float64
}

// SensorNoise 是传感器噪声参数。
type SensorNoise struct {
	ReadSigma, ShotGain float64
	Seed                uint32
}

// NoNoise 是无噪声设置。
var NoNoise = SensorNoise{Seed: 1}

// RandomShape 是随机形状模式。
type RandomShape uint8

// 随机形状常量。
const (
	RandomSpot RandomShape = iota
	RandomLine
	RandomMixedShape
)

// RandomKind 是随机类型模式。
type RandomKind uint8

// 随机类型常量。
const (
	RandomBright RandomKind = iota
	RandomDark
	RandomMixedKind
)

// RandomDefectOptions 是随机缺陷生成选项。
type RandomDefectOptions struct {
	Count                                    int
	Seed                                     uint32
	Width, Height                            int
	Kind                                     RandomKind
	Shape                                    RandomShape
	RadiusMin, RadiusMax, ValueMin, ValueMax float64
	Amplitude                                DefectAmplitude
	Profile                                  DefectProfile
	Softness, Grain, LineLengthMax           float64
	Material                                 []uint8
	Materials                                []uint8
	// HasColor 为 true 时把 Color 复制到每个生成的笔画。
	HasColor bool
	// Color 是 [R,G,B] 通道权重。
	Color [3]float64
}

func distanceToShape(x, y float64, shape DefectShape, sx, sy, x2, y2 float64) float64 {
	if shape == ShapeSpot {
		return math.Hypot(x-sx, y-sy)
	}
	vx := x2 - sx
	vy := y2 - sy
	lenSq := vx*vx + vy*vy
	if lenSq == 0 {
		return math.Hypot(x-sx, y-sy)
	}
	t := ((x-sx)*vx + (y-sy)*vy) / lenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	px := sx + t*vx
	py := sy + t*vy
	return math.Hypot(x-px, y-py)
}

func shapeWeight(d, radius, softness float64) float64 {
	if radius <= 0 {
		return 0
	}
	inner := radius * (1 - softness)
	if d <= inner {
		return 1
	}
	if d >= radius {
		return 0
	}
	return (radius - d) / (radius - inner)
}

func rectWeight(x, y, minX, maxX, minY, maxY, softness float64) float64 {
	depth := math.Min(math.Min(x-minX, maxX-x), math.Min(y-minY, maxY-y))
	if depth < 0 {
		return 0
	}
	feather := softness * math.Max(0.5, math.Min(maxX-minX, maxY-minY)/2)
	if feather <= 0 || depth >= feather {
		return 1
	}
	return depth / feather
}

func shapeBounds(shape DefectShape, x, y, x2, y2, radius float64) Rect {
	if shape == ShapeRect {
		minX := math.Min(x, x2)
		maxX := math.Max(x, x2)
		minY := math.Min(y, y2)
		maxY := math.Max(y, y2)
		return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
	}
	ex := x
	if shape == ShapeLine {
		ex = x2
	}
	ey := y
	if shape == ShapeLine {
		ey = y2
	}
	minX := math.Min(x, ex) - radius
	maxX := math.Max(x, ex) + radius
	minY := math.Min(y, ey) - radius
	maxY := math.Max(y, ey) + radius
	return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
}

// strokeEnd 返回笔画用于扫描的第二个端点；Spot 与 Polygon 使用 (X,Y)。
func strokeEnd(s DefectStroke) (x2, y2 float64) {
	if s.Shape == ShapeLine || s.Shape == ShapeRect || s.Shape == ShapeEllipse {
		return s.X2, s.Y2
	}
	return s.X, s.Y
}

// strokeBounds 返回笔画的整数扫描包围盒来源。
// 椭圆使用角点盒（同矩形）；多边形使用顶点 min/max（空则零矩形）；其余委托 shapeBounds。
func strokeBounds(s DefectStroke) Rect {
	if s.Shape == ShapeEllipse {
		minX := math.Min(s.X, s.X2)
		maxX := math.Max(s.X, s.X2)
		minY := math.Min(s.Y, s.Y2)
		maxY := math.Max(s.Y, s.Y2)
		return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
	}
	if s.Shape == ShapePolygon {
		if len(s.Points) == 0 {
			return Rect{}
		}
		minX := s.Points[0].X
		maxX := s.Points[0].X
		minY := s.Points[0].Y
		maxY := s.Points[0].Y
		for _, p := range s.Points[1:] {
			if p.X < minX {
				minX = p.X
			}
			if p.X > maxX {
				maxX = p.X
			}
			if p.Y < minY {
				minY = p.Y
			}
			if p.Y > maxY {
				maxY = p.Y
			}
		}
		return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
	}
	x2, y2 := strokeEnd(s)
	return shapeBounds(s.Shape, s.X, s.Y, x2, y2, s.Radius)
}

// ellipseWeight 计算椭圆权重；X,Y,X2,Y2 为包围盒对角点，Radius 忽略。
func ellipseWeight(s DefectStroke, x, y float64) float64 {
	cx := (s.X + s.X2) / 2
	cy := (s.Y + s.Y2) / 2
	rx := math.Abs(s.X2-s.X) / 2
	ry := math.Abs(s.Y2-s.Y) / 2
	if rx == 0 || ry == 0 {
		return 0
	}
	dx := (x - cx) / rx
	dy := (y - cy) / ry
	q := math.Sqrt(dx*dx + dy*dy)
	return shapeWeight(q, 1, s.Softness)
}

// pointInPolygon 使用奇偶射线法判断点是否在多边形内。
func pointInPolygon(px, py float64, pts []Point) bool {
	inside := false
	n := len(pts)
	j := n - 1
	for i := 0; i < n; i++ {
		xi := pts[i].X
		yi := pts[i].Y
		xj := pts[j].X
		yj := pts[j].Y
		if (yi > py) != (yj > py) {
			xCross := (xj-xi)*(py-yi)/(yj-yi) + xi
			if px < xCross {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}

// distanceToSegment 返回点到线段的最短距离。
func distanceToSegment(px, py, ax, ay, bx, by float64) float64 {
	vx := bx - ax
	vy := by - ay
	lenSq := vx*vx + vy*vy
	if lenSq == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	t := ((px-ax)*vx + (py-ay)*vy) / lenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return math.Hypot(px-(ax+t*vx), py-(ay+t*vy))
}

// polygonWeight 计算多边形权重；X,Y,X2,Y2,Radius 忽略。
func polygonWeight(s DefectStroke, x, y float64) float64 {
	pts := s.Points
	if len(pts) < 3 {
		return 0
	}
	if !pointInPolygon(x, y, pts) {
		return 0
	}
	if s.Softness <= 0 {
		return 1
	}
	b := strokeBounds(s)
	feather := s.Softness * math.Max(0.5, math.Min(b.W, b.H)/2)
	if feather <= 0 {
		return 1
	}
	depth := math.Inf(1)
	n := len(pts)
	j := n - 1
	for i := 0; i < n; i++ {
		d := distanceToSegment(x, y, pts[j].X, pts[j].Y, pts[i].X, pts[i].Y)
		if d < depth {
			depth = d
		}
		j = i
	}
	if depth >= feather {
		return 1
	}
	return depth / feather
}

func hashGrain(x, y int, seed uint32) float64 {
	h := uint32(int32(x))*0x27d4eb2d ^ uint32(int32(y))*0x165667b1 ^ seed*0x9e3779b9
	h = (h ^ (h >> 15)) * 0x85ebca6b
	h = (h ^ (h >> 13)) * 0xc2b2ae35
	h = h ^ (h >> 16)
	return (float64(h)/4294967296)*2 - 1
}

func applyProfile(lw float64, profile DefectProfile) float64 {
	if profile == ProfileGaussian {
		if lw <= 0 {
			return 0
		}
		if lw >= 1 {
			return 1
		}
		k := 1 - lw
		return math.Exp(-4.5 * k * k)
	}
	return lw
}

func strokeWeight(stroke DefectStroke, x, y, x2, y2 float64) float64 {
	var lw float64
	switch stroke.Shape {
	case ShapeRect:
		minX := math.Min(stroke.X, x2)
		maxX := math.Max(stroke.X, x2)
		minY := math.Min(stroke.Y, y2)
		maxY := math.Max(stroke.Y, y2)
		lw = rectWeight(x, y, minX, maxX, minY, maxY, stroke.Softness)
	case ShapeEllipse:
		lw = ellipseWeight(stroke, x, y)
	case ShapePolygon:
		lw = polygonWeight(stroke, x, y)
	default:
		d := distanceToShape(x, y, stroke.Shape, stroke.X, stroke.Y, x2, y2)
		lw = shapeWeight(d, stroke.Radius, stroke.Softness)
	}
	w := applyProfile(lw, stroke.Profile)
	grain := stroke.Grain
	if grain > 0 && w > 0 {
		seed := stroke.Seed
		if seed == 0 {
			seed = 1
		}
		h := hashGrain(int(x), int(y), seed)
		w = math.Max(0, w*(1+grain*h))
	}
	return w
}

func applyAmplitude(stroke DefectStroke, v, w float64) float64 {
	sign := 1.0
	if stroke.Kind == Dark {
		sign = -1
	}
	switch stroke.Amplitude {
	case AmpRelative:
		pct := stroke.RelativePct
		return v + sign*math.Max(v, 0)*(pct/100)*w
	case AmpTarget:
		tg := stroke.TargetGv
		if !stroke.HasTarget {
			if stroke.Kind == Bright {
				tg = 255
			} else {
				tg = 0
			}
		}
		nv := v + (tg-v)*w
		if stroke.Kind == Bright {
			return math.Max(v, nv)
		}
		return math.Min(v, nv)
	}
	return v + sign*stroke.DeltaGv*w
}

// PaintDefects 在灰度图上绘制缺陷笔画。
func PaintDefects(img *GrayImage, strokes []DefectStroke) *GrayImage {
	width := img.Width
	height := img.Height
	touched := make(map[int]float64)
	for _, stroke := range strokes {
		x2, y2 := strokeEnd(stroke)
		bounds := strokeBounds(stroke)
		x0 := int(math.Max(0, math.Floor(bounds.X)))
		y0 := int(math.Max(0, math.Floor(bounds.Y)))
		x1 := int(math.Min(float64(width), math.Floor(bounds.X+bounds.W)+1))
		y1 := int(math.Min(float64(height), math.Floor(bounds.Y+bounds.H)+1))
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				w := strokeWeight(stroke, float64(x), float64(y), x2, y2)
				if w <= 0 {
					continue
				}
				idx := y*width + x
				base, ok := touched[idx]
				if !ok {
					base = float64(img.Data[idx])
				}
				touched[idx] = applyAmplitude(stroke, base, w)
			}
		}
	}
	out := make([]uint8, len(img.Data))
	copy(out, img.Data)
	for idx, v := range touched {
		out[idx] = ClampByte(v)
	}
	return &GrayImage{Width: width, Height: height, Data: out}
}

// PaintScene 在场景上绘制笔画。
func PaintScene(scene *LightScene, strokes []SceneStroke) *LightScene {
	material := make([]uint8, len(scene.Material))
	copy(material, scene.Material)
	texture := make([]float32, len(scene.Texture))
	copy(texture, scene.Texture)
	width := scene.Width
	height := scene.Height
	for _, stroke := range strokes {
		x2 := stroke.X2
		if stroke.Shape != ShapeLine && stroke.Shape != ShapeRect {
			x2 = stroke.X
		}
		y2 := stroke.Y2
		if stroke.Shape != ShapeLine && stroke.Shape != ShapeRect {
			y2 = stroke.Y
		}
		bounds := shapeBounds(stroke.Shape, stroke.X, stroke.Y, x2, y2, stroke.Radius)
		x0 := int(math.Max(0, math.Floor(bounds.X)))
		y0 := int(math.Max(0, math.Floor(bounds.Y)))
		x1 := int(math.Min(float64(width), math.Floor(bounds.X+bounds.W)+1))
		y1 := int(math.Min(float64(height), math.Floor(bounds.Y+bounds.H)+1))
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				var w float64
				if stroke.Shape == ShapeRect {
					minX := math.Min(stroke.X, x2)
					maxX := math.Max(stroke.X, x2)
					minY := math.Min(stroke.Y, y2)
					maxY := math.Max(stroke.Y, y2)
					w = rectWeight(float64(x), float64(y), minX, maxX, minY, maxY, 0)
				} else {
					d := distanceToShape(float64(x), float64(y), stroke.Shape, stroke.X, stroke.Y, x2, y2)
					w = shapeWeight(d, stroke.Radius, 0)
				}
				if w <= 0 {
					continue
				}
				idx := y*width + x
				if stroke.Mode == SceneMaterial {
					if w > 0.5 {
						material[idx] = stroke.Material
					}
				} else {
					factor := stroke.Factor
					t := float64(texture[idx]) * (1 + (factor-1)*w)
					if t < 0 {
						t = 0
					} else if t > 4 {
						t = 4
					}
					texture[idx] = float32(t)
				}
			}
		}
	}
	return &LightScene{Width: width, Height: height, Material: material, Texture: texture}
}

// RenderDefectPair 渲染 golden/test 图像对。
func RenderDefectPair(scene *LightScene, s *LightSetting, ch CameraChannel, sceneStrokes []SceneStroke, gvStrokes []DefectStroke, noise SensorNoise) (golden, test *GrayImage) {
	golden = RenderChannel(scene, s, ch)
	painted := scene
	if len(sceneStrokes) > 0 {
		painted = PaintScene(scene, sceneStrokes)
	}
	test = RenderChannel(painted, s, ch)
	transfer := NewChannelTransfer(s, ch)
	n := len(painted.Material)
	touched := make(map[int]float64)
	for _, stroke := range gvStrokes {
		x2, y2 := strokeEnd(stroke)
		bounds := strokeBounds(stroke)
		x0 := int(math.Max(0, math.Floor(bounds.X)))
		y0 := int(math.Max(0, math.Floor(bounds.Y)))
		x1 := int(math.Min(float64(painted.Width), math.Floor(bounds.X+bounds.W)+1))
		y1 := int(math.Min(float64(painted.Height), math.Floor(bounds.Y+bounds.H)+1))
		cf := stroke.ChannelFactor(ch)
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				w := strokeWeight(stroke, float64(x), float64(y), x2, y2)
				if w <= 0 {
					continue
				}
				w *= cf
				if w <= 0 {
					continue
				}
				idx := y*painted.Width + x
				base, ok := touched[idx]
				if !ok {
					base = LinearValue(painted, transfer, idx)
				}
				touched[idx] = applyAmplitude(stroke, base, w)
			}
		}
	}
	active := noise.ReadSigma > 0 || noise.ShotGain > 0
	if active {
		rng := NewRng(noise.Seed)
		readSigma := noise.ReadSigma
		shotGain := noise.ShotGain
		for idx := 0; idx < n; idx++ {
			v, ok := touched[idx]
			if !ok {
				v = LinearValue(painted, transfer, idx)
			}
			sigma := math.Sqrt(readSigma*readSigma + shotGain*math.Max(v, 0))
			test.Data[idx] = ClampByte(v + sigma*rng.NextGaussian())
		}
	} else {
		for idx, v := range touched {
			test.Data[idx] = ClampByte(v)
		}
	}
	return golden, test
}

// RenderGoodSample 渲染一张良品图像（不带任何不良笔画），可叠加传感器噪声。
// 它供教学（学习检出区间）使用，只产生测试图，不产生 golden。
func RenderGoodSample(scene *LightScene, s *LightSetting, ch CameraChannel, noise SensorNoise) *GrayImage {
	_, test := RenderDefectPair(scene, s, ch, nil, nil, noise)
	return test
}

// GenerateRandomDefects 生成随机缺陷笔画。
func GenerateRandomDefects(o RandomDefectOptions) []DefectStroke {
	rng := NewRng(o.Seed)
	var out []DefectStroke
	wanted := make(map[uint8]bool)
	for _, m := range o.Materials {
		wanted[m] = true
	}
	maxAttempts := o.Count * 50
	attempts := 0
	for len(out) < o.Count && attempts < maxAttempts {
		attempts++
		radius := o.RadiusMin + rng.Next()*(o.RadiusMax-o.RadiusMin)
		lo := math.Ceil(radius)
		hiX := float64(o.Width-1) - math.Ceil(radius)
		hiY := float64(o.Height-1) - math.Ceil(radius)
		if hiX < lo || hiY < lo {
			break
		}
		cx := lo + rng.Next()*(hiX-lo)
		cy := lo + rng.Next()*(hiY-lo)
		if len(wanted) > 0 && o.Material != nil {
			ix := int(math.Min(float64(o.Width-1), math.Max(0, math.Round(cx))))
			iy := int(math.Min(float64(o.Height-1), math.Max(0, math.Round(cy))))
			if !wanted[o.Material[iy*o.Width+ix]] {
				continue
			}
		}
		kind := Bright
		if o.Kind == RandomMixedKind {
			if rng.Next() < 0.5 {
				kind = Bright
			} else {
				kind = Dark
			}
		} else if o.Kind == RandomDark {
			kind = Dark
		}
		shape := ShapeSpot
		if o.Shape == RandomMixedShape {
			if rng.Next() < 0.5 {
				shape = ShapeSpot
			} else {
				shape = ShapeLine
			}
		} else if o.Shape == RandomLine {
			shape = ShapeLine
		}
		value := o.ValueMin + rng.Next()*(o.ValueMax-o.ValueMin)
		stroke := DefectStroke{
			Kind:      kind,
			Shape:     shape,
			X:         cx,
			Y:         cy,
			Radius:    radius,
			DeltaGv:   0,
			Softness:  o.Softness,
			Amplitude: o.Amplitude,
			Profile:   o.Profile,
			Grain:     o.Grain,
			Seed:      o.Seed*1000 + uint32(len(out)),
			HasColor:  o.HasColor,
			Color:     o.Color,
		}
		if shape == ShapeLine {
			minLen := 2 * radius
			maxLen := math.Max(minLen, o.LineLengthMax)
			length := minLen + rng.Next()*(maxLen-minLen)
			angle := rng.Next() * math.Pi * 2
			stroke.X2 = cx + math.Cos(angle)*length
			stroke.Y2 = cy + math.Sin(angle)*length
		}
		switch o.Amplitude {
		case AmpAbsolute:
			stroke.DeltaGv = value
		case AmpRelative:
			stroke.RelativePct = value
		case AmpTarget:
			stroke.TargetGv = math.Min(255, math.Max(0, value))
			stroke.HasTarget = true
		}
		out = append(out, stroke)
	}
	return out
}
