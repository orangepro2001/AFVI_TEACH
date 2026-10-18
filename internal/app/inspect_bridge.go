package app

import (
	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/inspect"
)

// inspectImage 把仿真产生的测试图交给检出（共享像素切片，不复制）。
func inspectImage(g *core.GrayImage) inspect.Image {
	return inspect.Image{Width: g.Width, Height: g.Height, Pix: g.Data}
}

// inspectRegions 把产品的设计材质图交给检出（未经任何不良涂抹的原始材质层）。
func inspectRegions(s *Scene) inspect.Regions {
	return inspect.Regions{Width: s.W, Height: s.H, Material: s.Material}
}

// goodSampleNoise 为"学习良品"选择与测试图不同的噪声种子，使学到的区间不依赖于测试图自身的噪声实现。
func goodSampleNoise(n core.SensorNoise) core.SensorNoise {
	return core.GoodSampleNoise(n)
}
