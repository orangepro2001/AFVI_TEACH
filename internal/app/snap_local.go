package app

import (
	"fmt"
	"os"
	"path/filepath"

	"afvi/teach-imgui/internal/core"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// localScene returns scene itself when its baked virtual good unit already equals the
// recipe's product; otherwise a shallow copy whose Light is rebuilt for that product.
func localScene(scene *Scene, product core.ProductOptions) (*Scene, error) {
	if scene == nil {
		return nil, fmt.Errorf("snap: no scene")
	}
	if product == scene.Product {
		return scene, nil
	}
	light, err := core.NewLightScene(scene.Median, scene.Material)
	if err != nil {
		return nil, err
	}
	core.ApplyProductVariation(light, product)
	next := *scene
	next.Light = light
	next.Product = product
	return &next, nil
}

// localFrames renders the wanted kinds (snapfmt.KindTest / snapfmt.KindGood) of channel
// ch in-process, exactly like camera.exe renders the same description.
func localFrames(scene *Scene, meta *snapfmt.Meta, ch core.CameraChannel, kinds []string) (map[snapKey]*core.GrayImage, error) {
	if meta == nil {
		return nil, fmt.Errorf("snap: no description")
	}
	cr := meta.Recipe.Core()
	ls, err := localScene(scene, cr.Product)
	if err != nil {
		return nil, err
	}
	r := &liveRenderer{scene: ls, cr: cr}
	out := make(map[snapKey]*core.GrayImage, len(kinds))
	for _, kind := range kinds {
		f, err := r.render(snapfmt.ChannelName(ch), kind)
		if err != nil {
			return nil, err
		}
		if f.Width != scene.W || f.Height != scene.H {
			return nil, fmt.Errorf("snap: image %s is %dx%d, want %dx%d", kind, f.Width, f.Height, scene.W, scene.H)
		}
		out[snapKey{ch: ch, kind: kind}] = frameToGray(f)
	}
	return out, nil
}

// saveLocal writes a full capture (test and good image of every channel plus the
// description) rendered in-process.
func saveLocal(dir string, scene *Scene, meta *snapfmt.Meta) (*snapfmt.Meta, error) {
	if meta == nil {
		return nil, fmt.Errorf("snap: no description")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	out := *meta
	out.Images = []snapfmt.Image{}
	for ch := core.ChRed; ch <= core.ChBlue; ch++ {
		imgs, err := localFrames(scene, meta, ch, []string{snapfmt.KindTest, snapfmt.KindGood})
		if err != nil {
			return nil, err
		}
		for _, kind := range []string{snapfmt.KindTest, snapfmt.KindGood} {
			img := imgs[snapKey{ch: ch, kind: kind}]
			if img == nil {
				return nil, fmt.Errorf("snap: image %s-%s missing", snapfmt.ChannelName(ch), kind)
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
