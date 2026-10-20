package camera

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/model"
	"afvi/teach-imgui/internal/snap"
)

// Main runs the camera program. args excludes the program name. It returns the exit code:
// 0 success, 1 runtime failure, 2 usage error (messages go to stderr, results to stdout).
func Main(args []string, stdout, stderr io.Writer) int {
	start := time.Now()
	fs := flag.NewFlagSet("camera", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configFlag := fs.String("config", "", "machine configuration file")
	machineFlag := fs.String("machine", "", "machine name to activate")
	refFlag := fs.String("ref", "", "reference directory (contains FM1/FM2/BM)")
	modelFlag := fs.String("model", "", "model id such as FM1/25fbo015-15")
	inFlag := fs.String("in", "", "description file or capture folder to render")
	outFlag := fs.String("out", "snap-out", "output capture folder or description file")
	modeFlag := fs.String("mode", "image", "image or recipe")
	channelsFlag := fs.String("channels", "red", "comma list of red,green,blue or all")
	goodFlag := fs.Bool("good", false, "also render the defect-free good sample")
	countFlag := fs.Int("count", core.DefaultTinyCount, "number of default tiny defects")
	seedFlag := fs.Uint("seed", uint(core.DefaultTinySeed), "seed of the default tiny defects")
	strengthFlag := fs.Float64("strength", 1, "virtual good unit strength override")
	noProductFlag := fs.Bool("no-product", false, "disable the virtual good unit")
	noiseReadFlag := fs.Float64("noise-read", 0, "sensor noise sigma")
	noiseShotFlag := fs.Float64("noise-shot", 0, "sensor shot gain")
	noiseSeedFlag := fs.Uint("noise-seed", 1, "sensor noise seed")
	alignedFlag := fs.Bool("aligned", false, "render the built-in default scenario exactly in place, without the default camera misalignment (use together with -dx/-dy/-rot to place the product exactly; ignored with -in)")
	warpFlags := newWarpFlags(fs)
	listFlag := fs.Bool("list", false, "print the ids of the models of the active machine")
	infoFlag := fs.String("info", "", "print a summary of the capture at that path")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	if *infoFlag != "" {
		if err := printInfo(stdout, *infoFlag); err != nil {
			fmt.Fprintf(stderr, "camera: %v\n", err)
			return 1
		}
		return 0
	}

	if *listFlag {
		cfg, _, err := model.ResolveConfig(*configFlag, *refFlag, *machineFlag)
		if err != nil {
			fmt.Fprintf(stderr, "camera: %v\n", err)
			return 1
		}
		m := cfg.Machines[cfg.ActiveIndex()]
		for _, info := range model.ScanMachine(cfg, m) {
			fmt.Fprintln(stdout, info.ID)
		}
		return 0
	}

	if *modeFlag != "image" && *modeFlag != "recipe" {
		fmt.Fprintf(stderr, "camera: -mode %q is not image or recipe\n", *modeFlag)
		return 2
	}
	if *countFlag < 0 {
		fmt.Fprintln(stderr, "camera: -count must not be negative")
		return 2
	}

	var scene *model.Scene
	var recipe snap.Recipe
	if *inFlag != "" {
		meta, dir, err := snap.ReadMeta(*inFlag)
		if err != nil {
			fmt.Fprintf(stderr, "camera: %v\n", err)
			return 1
		}
		_ = dir
		recipe = meta.Recipe
		if *modelFlag != "" {
			scene, err = loadModelScene(*configFlag, *refFlag, *machineFlag, *modelFlag, stderr)
			if err != nil {
				fmt.Fprintf(stderr, "camera: %v\n", err)
				return 1
			}
		} else {
			scene, err = SceneFromMeta(meta)
			if err != nil {
				fmt.Fprintf(stderr, "camera: %v\n", err)
				return 1
			}
		}
	} else {
		var err error
		scene, err = loadModelScene(*configFlag, *refFlag, *machineFlag, *modelFlag, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "camera: %v\n", err)
			return 1
		}
		recipe = DefaultRecipe(scene, *countFlag, uint32(*seedFlag))
		if *alignedFlag {
			recipe = recipe.WithWarp(core.IdentityWarp)
		}
		warp := recipe.CoreWarp()
		cr := recipe.Core()
		cr.Product.Strength = *strengthFlag
		if *noProductFlag {
			cr.Product.Enabled = false
		}
		cr.Noise.ReadSigma = *noiseReadFlag
		cr.Noise.ShotGain = *noiseShotFlag
		cr.Noise.Seed = uint32(*noiseSeedFlag)
		recipe = snap.NewRecipe(cr.Product, cr.Light, cr.Noise, cr.Defects, cr.Scene).WithWarp(warp)
	}

	recipe, err := warpFlags.apply(fs, recipe)
	if err != nil {
		fmt.Fprintf(stderr, "camera: %v\n", err)
		return 2
	}

	if *modeFlag == "recipe" {
		meta := NewMeta(scene, recipe)
		target := *outFlag
		if !strings.HasSuffix(target, ".json") && !strings.HasSuffix(target, ".gz") {
			target = filepath.Join(target, snap.MetaFile)
		}
		if err := snap.WriteMeta(target, &meta); err != nil {
			fmt.Fprintf(stderr, "camera: %v\n", err)
			return 1
		}
		data, err := snap.Marshal(&meta)
		if err != nil {
			fmt.Fprintf(stderr, "camera: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "wrote %s (%d bytes)\n", target, len(data))
		return 0
	}

	if strings.HasSuffix(*outFlag, ".json") || strings.HasSuffix(*outFlag, ".gz") {
		fmt.Fprintln(stderr, "camera: -out must be a folder in image mode")
		return 2
	}
	chans, err := parseChannels(*channelsFlag)
	if err != nil {
		fmt.Fprintf(stderr, "camera: %v\n", err)
		return 2
	}
	res, err := Render(scene, recipe, Options{Channels: chans, Good: *goodFlag})
	if err != nil {
		fmt.Fprintf(stderr, "camera: %v\n", err)
		return 1
	}
	if err := Save(*outFlag, res); err != nil {
		fmt.Fprintf(stderr, "camera: %v\n", err)
		return 1
	}
	for _, img := range res.Meta.Images {
		fmt.Fprintf(stdout, "wrote %s (%dx%d)\n", filepath.Join(*outFlag, img.File), img.Width, img.Height)
	}
	fmt.Fprintf(stdout, "snap: %s %dx%d defects=%d time=%.3fs\n", res.Meta.Model.ID, res.Meta.Width, res.Meta.Height, len(res.Meta.Recipe.Defects), time.Since(start).Seconds())
	return 0
}

// loadModelScene loads the named model; an empty id picks the first model of the active
// machine (like the teach app) so a bare "camera" run still produces a capture.
func loadModelScene(configFlag, refFlag, machineFlag, id string, stderr io.Writer) (*model.Scene, error) {
	cfg, _, err := model.ResolveConfig(configFlag, refFlag, machineFlag)
	if err != nil {
		return nil, err
	}
	m := cfg.Machines[cfg.ActiveIndex()]
	infos := model.ScanMachine(cfg, m)
	if id == "" {
		if len(infos) == 0 {
			return nil, fmt.Errorf("machine %s has no models (use -ref, -config or -machine)", m.Name)
		}
		id = infos[0].ID
		fmt.Fprintf(stderr, "camera: no -model given, using %s (-list shows the others, -h shows all flags)\n", id)
	}
	info, ok := model.FindModel(infos, id)
	if !ok {
		return nil, fmt.Errorf("model %s not found", id)
	}
	return model.LoadScene(info)
}

func parseChannels(s string) ([]core.CameraChannel, error) {
	var out []core.CameraChannel
	seen := make(map[core.CameraChannel]bool)
	add := func(ch core.CameraChannel) {
		if !seen[ch] {
			seen[ch] = true
			out = append(out, ch)
		}
	}
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.EqualFold(item, "all") {
			add(core.ChRed)
			add(core.ChGreen)
			add(core.ChBlue)
			continue
		}
		ch, err := snap.ParseChannel(item)
		if err != nil {
			return nil, fmt.Errorf("-channels %q is not red, green, blue or all", item)
		}
		add(ch)
	}
	return out, nil
}

func printInfo(w io.Writer, path string) error {
	meta, _, err := snap.ReadMeta(path)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "path: %s\n", path)
	fmt.Fprintf(w, "producer: %s\n", meta.Producer)
	fmt.Fprintf(w, "created: %s\n", meta.Created)
	fmt.Fprintf(w, "model: %s\n", meta.Model.ID)
	fmt.Fprintf(w, "size: %dx%d\n", meta.Width, meta.Height)
	p := meta.Recipe.Product
	fmt.Fprintf(w, "product: enabled=%v seed=%d strength=%g\n", p.Enabled, p.Seed, p.Strength)
	for _, c := range meta.Recipe.Light.Channels {
		state := "off"
		if c.Enable {
			state = "on"
		}
		fmt.Fprintf(w, "light: color=%d value=%g %s\n", c.Color, c.Value, state)
	}
	n := meta.Recipe.Noise
	fmt.Fprintf(w, "noise: readSigma=%g shotGain=%g seed=%d\n", n.ReadSigma, n.ShotGain, n.Seed)
	if wp := meta.Recipe.Warp; wp != nil {
		fmt.Fprintf(w, "warp: dx=%g dy=%g rot=%g scale=%g shear=(%g,%g) tilt=(%g,%g) bend=%g wave=%g\n",
			wp.Tx, wp.Ty, wp.ThetaDeg, wp.Scale, wp.ShearX, wp.ShearY, wp.TiltX, wp.TiltY, wp.BendAmplitude, wp.BendWavelength)
	}
	bright, dark := 0, 0
	for _, d := range meta.Recipe.Defects {
		if d.Kind == 0 {
			bright++
		} else {
			dark++
		}
	}
	fmt.Fprintf(w, "defects: %d (bright=%d dark=%d)\n", len(meta.Recipe.Defects), bright, dark)
	fmt.Fprintf(w, "sceneStrokes: %d\n", len(meta.Recipe.SceneStrokes))
	for _, img := range meta.Images {
		prefix := img.SHA256
		if len(prefix) > 12 {
			prefix = prefix[:12]
		}
		fmt.Fprintf(w, "image: %s %s %s %dx%d %s\n", img.Channel, img.Kind, img.File, img.Width, img.Height, prefix)
	}
	return nil
}
