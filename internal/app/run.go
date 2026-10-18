// Package app is the native AFVI teaching app: Dear ImGui (cimgui-go, GLFW +
// OpenGL3) front end over the Go port of teach-core. It replaces the WebView2 shell.
package app

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AllenDang/cimgui-go/backend"
	"github.com/AllenDang/cimgui-go/backend/glfwbackend"
	"github.com/AllenDang/cimgui-go/imgui"
	_ "github.com/AllenDang/cimgui-go/impl/glfw"
	_ "github.com/AllenDang/cimgui-go/impl/opengl3"

	"afvi/teach-imgui/internal/config"
)

// WindowTitle is also used by the self-test to find the window for screenshots.
const WindowTitle = "Color AFVI Master Teach"

// Run parses the command line, loads the machine configuration and runs the UI main loop.
// It must be called from the main goroutine on a locked OS thread (GLFW and OpenGL
// require every call to come from the main OS thread); cmd/teach-imgui does that.
func Run() {
	cfgFlag := flag.String("config", "", "machine configuration file (afvi.toml); default: AFVI_CONFIG, next to the exe, or the working directory")
	machine := flag.String("machine", "", "machine name to activate (case-insensitive)")
	ref := flag.String("ref", "", "reference directory (contains FM1/FM2/BM); upserts the simulated REFERENCE machine")
	theme := flag.String("theme", "", "ui theme: dark | light (default: config, else light)")
	lang := flag.String("lang", "", "ui language: zh | en | ko (default: config, else zh)")
	model := flag.String("model", "", "model id to open first, e.g. FM1/25fbo015-15")
	selftest := flag.Bool("selftest", false, "load the first model, visit every tab in both themes and three languages, save screenshots and metrics, then exit")
	out := flag.String("out", ".", "output directory for -selftest screenshots and metrics")
	exitAfter := flag.Duration("exitafter", 0, "close the window after this duration")
	flag.Parse()

	cfgPath, found := config.Resolve(*cfgFlag)
	var cfg *config.Config
	if found {
		loaded, err := config.Load(cfgPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
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
	if *ref != "" {
		abs, err := filepath.Abs(*ref)
		if err != nil {
			abs = *ref
		}
		cfg.UpsertReference(abs)
	}
	if *machine != "" {
		for _, m := range cfg.Machines {
			if strings.EqualFold(m.Name, *machine) {
				cfg.Active = m.Name
				break
			}
		}
	}
	themeName := *theme
	if themeName == "" {
		themeName = cfg.Theme
	}
	if themeName == "" {
		themeName = ThemeLight
	}
	if themeName != ThemeDark {
		themeName = ThemeLight
	}
	langName := *lang
	if langName == "" {
		langName = cfg.Lang
	}
	if langName == "" {
		langName = string(LangZh)
	}
	SetLang(ParseLang(langName))

	a := NewApp(cfg, cfgPath, themeName)
	a.exitAfter = *exitAfter
	if *selftest {
		a.selftest = NewSelftest(a, *out)
	}

	be, err := backend.CreateBackend(glfwbackend.NewGLFWBackend())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	a.be = be
	be.SetAfterCreateContextHook(setupFonts)
	be.SetAfterRenderHook(func() {
		if a.selftest != nil {
			a.selftest.AfterRender()
		}
	})
	be.SetBgColor(imgui.NewVec4(0.17, 0.17, 0.17, 1))
	be.CreateWindow(WindowTitle, 1600, 1000)
	if a.selftest != nil {
		be.SetTargetFPS(1000)
	} else {
		be.SetTargetFPS(60)
	}

	first := 0
	for i, m := range a.Models {
		if m.ID == *model {
			first = i
		}
	}
	a.LoadModel(first)
	be.Run(func() { loop(a) })
}

func loop(a *App) {
	if a.frame == 0 {
		// The GLFW backend force-enables multi-viewport after our hook; this fixed layout must
		// stay inside the main window, otherwise edge panels pop out as separate OS windows.
		io := imgui.CurrentIO()
		io.SetConfigFlags(io.ConfigFlags() &^ imgui.ConfigFlagsViewportsEnable)
	}
	if a.selftest != nil && a.frame == 1 {
		_ = a.be.SetSwapInterval(glfwbackend.GLFWSwapIntervalImmediate)
	}
	a.Frame()
	if a.selftest != nil {
		a.selftest.Step()
	}
	if a.exitAfter > 0 && time.Since(a.start) > a.exitAfter {
		a.be.SetShouldClose(true)
	}
}

// setupFonts merges Tahoma (Latin), Microsoft YaHei (CJK) and Malgun Gothic (Hangul);
// ImGui 1.92+ bakes glyphs on demand, so no glyph ranges are needed.
func setupFonts() {
	atlas := imgui.CurrentIO().Fonts()
	base := true
	for _, path := range []string{`C:\Windows\Fonts\tahoma.ttf`, `C:\Windows\Fonts\msyh.ttc`, `C:\Windows\Fonts\malgun.ttf`} {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		cfg := imgui.NewFontConfig()
		if !base {
			cfg.SetMergeMode(true)
		}
		atlas.AddFontFromFileTTFV(path, 13, cfg, nil)
		base = false
	}
}
