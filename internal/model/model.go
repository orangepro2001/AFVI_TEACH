// Package model scans the reference data roots of a machine and loads one model (MEDIAN plus the Metal/SR/Open design layers). It is pure Go and shared by the teach app and the camera simulator.
package model

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"afvi/teach-imgui/internal/config"
	"afvi/teach-imgui/internal/core"
)

// Sentinel errors returned by CheckHostPath.
var (
	ErrPathNotSet   = errors.New("path not set")
	ErrPathNotFound = errors.New("path not found")
	ErrNoRepository = errors.New("PxRepository not found")
)

// ModelInfo describes one reference model; paths are absolute. Pattern/AlignSpec may be "".
type ModelInfo struct {
	ID, Host, Model, Side, Version string
	Metal, SR, Open, Median        string
	Pattern, AlignSpec             string
}

var hosts = []string{"FM1", "FM2", "BM"}

// isDir reports whether p is a directory. A bare network server (\\host) cannot be stat'ed,
// so it counts as one; whether it really serves PxRepository is checked by the caller on
// the share below it (\\host\PxRepository).
func isDir(p string) bool {
	if config.IsUNCServerOnly(p) {
		return true
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func looksLikeReference(dir string) bool {
	for _, h := range hosts {
		if isDir(filepath.Join(dir, h)) {
			return true
		}
	}
	return false
}

// FindReferenceRoot resolves the reference directory: -ref flag, AFVI_REFERENCE_DIR,
// reference/ next to the exe (or up to 5 parents), then the working directory (and parents).
func FindReferenceRoot(override string) (string, error) {
	var candidates []string
	if override != "" {
		candidates = append(candidates, override)
	}
	if env := os.Getenv("AFVI_REFERENCE_DIR"); env != "" {
		candidates = append(candidates, env)
	}
	walk := func(start string) {
		dir := start
		for i := 0; i <= 5 && dir != ""; i++ {
			candidates = append(candidates, filepath.Join(dir, "reference"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if exe, err := os.Executable(); err == nil {
		walk(filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		walk(wd)
	}
	for _, c := range candidates {
		if abs, err := filepath.Abs(c); err == nil && looksLikeReference(abs) {
			return abs, nil
		}
	}
	return "", errors.New("reference directory not found")
}

func listDirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

func normalizeModelKey(name string) string {
	s := strings.ToUpper(name)
	return strings.TrimSuffix(s, "-00")
}

func medianPath(modelDir, version, side string) string {
	return filepath.Join(modelDir, version, "Line", side, "Layer", "MEDIAN", "Median_0_1_0.tif")
}

// ScanHost mirrors teach-app/server.ts scanModels for one host root: every PxRepository
// model with Metal/SR/Open and a version holding MEDIAN (preferring 2.0 for FM, 3.5 for
// BM). root is the host data root (the directory that contains PxRepository).
func ScanHost(host, root string) []ModelInfo {
	var out []ModelInfo
	if root == "" {
		return out
	}
	repo := filepath.Join(root, "PxRepository")
	for _, model := range listDirs(repo) {
		modelDir := filepath.Join(repo, model)
		metal, sr, open := filepath.Join(modelDir, "Metal.tif"), filepath.Join(modelDir, "SR.tif"), filepath.Join(modelDir, "Open.tif")
		if !isFile(metal) || !isFile(sr) || !isFile(open) {
			continue
		}
		side, preferred := "TOP", "2.0"
		if host == "BM" {
			side, preferred = "BOTTOM", "3.5"
		}
		sideDir := map[string]string{"TOP": "Top", "BOTTOM": "Bottom"}[side]
		version := ""
		if isFile(medianPath(modelDir, preferred, sideDir)) {
			version = preferred
		} else {
			for _, v := range listDirs(modelDir) {
				if isFile(medianPath(modelDir, v, sideDir)) {
					version = v
					break
				}
			}
		}
		if version == "" {
			continue
		}
		info := ModelInfo{
			ID: host + "/" + model, Host: host, Model: model, Side: side, Version: version,
			Metal: metal, SR: sr, Open: open, Median: medianPath(modelDir, version, sideDir),
		}
		if p := filepath.Join(modelDir, version, "Line", sideDir, "Layer", "GB", "Pattern.tif"); isFile(p) {
			info.Pattern = p
		}
		specRoot := filepath.Join(root, "PxInventory", "INSPECT_SPEC")
		for _, dir := range listDirs(specRoot) {
			if normalizeModelKey(dir) != normalizeModelKey(model) {
				continue
			}
			if p := filepath.Join(specRoot, dir, side, "AlignSpec.xml"); isFile(p) {
				info.AlignSpec = p
				break
			}
		}
		out = append(out, info)
	}
	return out
}

// ScanMachine scans every host of one machine, resolving relative paths against the
// configuration file directory and skipping hosts with an empty or missing root.
func ScanMachine(cfg *config.Config, m config.Machine) []ModelInfo {
	var out []ModelInfo
	for _, host := range hosts {
		root := cfg.ResolvePath(m.HostPath(host))
		if root == "" || !isDir(root) {
			continue
		}
		out = append(out, ScanHost(host, root)...)
	}
	return out
}

// CheckHostPath reports how many models a configured host path yields, or one of the
// sentinel errors describing why it is unusable.
func CheckHostPath(cfg *config.Config, host, path string) (int, error) {
	if strings.TrimSpace(path) == "" {
		return 0, ErrPathNotSet
	}
	root := cfg.ResolvePath(path)
	if !isDir(root) {
		return 0, ErrPathNotFound
	}
	if !isDir(filepath.Join(root, "PxRepository")) {
		return 0, ErrNoRepository
	}
	return len(ScanHost(host, root)), nil
}

var (
	reUnit   = regexp.MustCompile(`(?is)<Unit\b[^>]*>(.*?)</Unit>`)
	reParam  = regexp.MustCompile(`(?i)<Param\b([^>]*)/?>`)
	reType   = regexp.MustCompile(`(?i)Type="(\d+)"`)
	reID     = regexp.MustCompile(`(?i)ID="(\d+)"`)
	reLeft   = regexp.MustCompile(`(?i)Left="(-?\d+)"`)
	reTop    = regexp.MustCompile(`(?i)Top="(-?\d+)"`)
	reRight  = regexp.MustCompile(`(?i)Right="(-?\d+)"`)
	reBottom = regexp.MustCompile(`(?i)Bottom="(-?\d+)"`)
)

// ParseAlignSpec mirrors teach-app/src/worker.ts parseAlignSpec: Type="0" Params of the
// first <Unit> become machine ROIs named P<ID>.
func ParseAlignSpec(xml string) []core.AlignRoi {
	var rois []core.AlignRoi
	unit := reUnit.FindStringSubmatch(xml)
	if unit == nil {
		return rois
	}
	for _, m := range reParam.FindAllStringSubmatch(unit[1], -1) {
		attrs := m[1]
		typ := reType.FindStringSubmatch(attrs)
		if typ == nil || typ[1] != "0" {
			continue
		}
		id, l, t, r, b := reID.FindStringSubmatch(attrs), reLeft.FindStringSubmatch(attrs), reTop.FindStringSubmatch(attrs), reRight.FindStringSubmatch(attrs), reBottom.FindStringSubmatch(attrs)
		if id == nil || l == nil || t == nil || r == nil || b == nil {
			continue
		}
		x, _ := strconv.Atoi(l[1])
		y, _ := strconv.Atoi(t[1])
		right, _ := strconv.Atoi(r[1])
		bottom, _ := strconv.Atoi(b[1])
		w, h := right-x, bottom-y
		if w <= 0 || h <= 0 {
			continue
		}
		rois = append(rois, core.AlignRoi{Name: "P" + id[1], Rect: core.Rect{X: float64(x), Y: float64(y), W: float64(w), H: float64(h)}})
	}
	return rois
}

// Scene is the loaded model shared by all panels; treat it as read-only.
type Scene struct {
	Info        ModelInfo
	W, H        int
	Median      *core.GrayImage
	Material    []uint8
	Light       *core.LightScene
	Stats       [4]core.MaterialStat // MEDIAN statistics per material
	MachineRois []core.AlignRoi
	LoadMs      float64
	// Product is the virtual good unit baked into Light.Texture; MEDIAN statistics in
	// Stats stay those of the original image.
	Product core.ProductOptions
}

// LoadScene decodes the four layers in parallel and builds the light scene.
func LoadScene(info ModelInfo) (*Scene, error) {
	start := time.Now()
	paths := []string{info.Metal, info.SR, info.Open, info.Median}
	imgs := make([]*core.GrayImage, len(paths))
	errs := make([]error, len(paths))
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			imgs[i], errs[i] = core.ReadTiffGray(p)
		}(i, p)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(paths[i]), err)
		}
	}
	material, err := core.BuildMaterialMap(imgs[0], imgs[1], imgs[2], 128)
	if err != nil {
		return nil, err
	}
	median := imgs[3]
	if median.Width*median.Height != len(material) {
		return nil, fmt.Errorf("MEDIAN %dx%d does not match layers", median.Width, median.Height)
	}
	light, err := core.NewLightScene(median, material)
	if err != nil {
		return nil, err
	}
	product := core.DefaultProductOptions()
	core.ApplyProductVariation(light, product)
	stats, err := core.MaterialStats(median, material)
	if err != nil {
		return nil, err
	}
	s := &Scene{Info: info, W: median.Width, H: median.Height, Median: median, Material: material, Light: light, Stats: stats, Product: product}
	if info.AlignSpec != "" {
		if data, err := os.ReadFile(info.AlignSpec); err == nil {
			s.MachineRois = ParseAlignSpec(strings.TrimPrefix(string(data), "\xef\xbb\xbf"))
		}
	}
	s.LoadMs = float64(time.Since(start).Microseconds()) / 1000
	return s, nil
}

// FindModel returns the model whose ID equals id (case-insensitive).
func FindModel(models []ModelInfo, id string) (ModelInfo, bool) {
	for _, m := range models {
		if strings.EqualFold(m.ID, id) {
			return m, true
		}
	}
	return ModelInfo{}, false
}

// ResolveConfig loads the machine configuration like the teach app does: the explicit
// path cfgFlag, then AFVI_CONFIG, then afvi.toml next to the executable or in the working
// directory (config.Resolve). When no file exists a REFERENCE machine is synthesised from
// FindReferenceRoot("") (falling back to a "reference" folder next to the executable) and
// cfg.BaseDir is set to the directory of the resolved config path. A non-empty refFlag is
// made absolute and upserted as the REFERENCE machine (cfg.UpsertReference). A non-empty
// machineFlag activates the machine of that name (case-insensitive) when it exists.
// It returns the configuration and the resolved configuration path. A configuration file
// that exists but cannot be loaded is returned as an error.
func ResolveConfig(cfgFlag, refFlag, machineFlag string) (*config.Config, string, error) {
	cfgPath, found := config.Resolve(cfgFlag)
	var cfg *config.Config
	if found {
		loaded, err := config.Load(cfgPath)
		if err != nil {
			return nil, cfgPath, err
		}
		cfg = loaded
	} else {
		refRoot, err := FindReferenceRoot("")
		if err != nil {
			exe, exeErr := os.Executable()
			if exeErr != nil {
				exe = "."
			}
			refRoot = filepath.Join(filepath.Dir(exe), "reference")
		}
		cfg = config.Default(refRoot)
		cfg.BaseDir = filepath.Dir(cfgPath)
	}
	if refFlag != "" {
		abs, err := filepath.Abs(refFlag)
		if err != nil {
			abs = refFlag
		}
		cfg.UpsertReference(abs)
	}
	if machineFlag != "" {
		for _, m := range cfg.Machines {
			if strings.EqualFold(m.Name, machineFlag) {
				cfg.Active = m.Name
				break
			}
		}
	}
	return cfg, cfgPath, nil
}
