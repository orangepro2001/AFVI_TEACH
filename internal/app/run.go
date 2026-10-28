// Package app is the native AFVI teaching app: Dear ImGui (cimgui-go, GLFW +
// OpenGL3) front end over the Go port of teach-core. It replaces the WebView2 shell.
package app

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/AllenDang/cimgui-go/backend"
	"github.com/AllenDang/cimgui-go/backend/glfwbackend"
	"github.com/AllenDang/cimgui-go/imgui"
	_ "github.com/AllenDang/cimgui-go/impl/glfw"
	_ "github.com/AllenDang/cimgui-go/impl/opengl3"

	"afvi/teach-imgui/internal/access"
	"afvi/teach-imgui/internal/model"
)

// Run runs the teaching program (align bench + guided operator pages); cmd/teach-imgui
// calls it. It starts on the operator pages unless -login is given. It must be called
// from the main goroutine on a locked OS thread (GLFW and OpenGL require every call to
// come from the main OS thread).
func Run() { runProfile(ProfileTeach) }

// RunStudio runs the camera studio (light + defect synthesis); cmd/camera-gui calls it
// from a locked OS thread. It starts locked (login dialog) unless -login is given and
// only accepts an admin login.
func RunStudio() { runProfile(ProfileStudio) }

// runProfile parses the command line, loads the machine configuration and runs the UI
// main loop for the given profile.
func runProfile(profile Profile) {
	cfgFlag := flag.String("config", "", "machine configuration file (afvi.toml); default: AFVI_CONFIG, next to the exe, or the working directory")
	machine := flag.String("machine", "", "machine name to activate (case-insensitive)")
	ref := flag.String("ref", "", "reference directory (contains FM1/FM2/BM); upserts the simulated REFERENCE machine")
	theme := flag.String("theme", "", "ui theme: dark | light (default: config, else light)")
	lang := flag.String("lang", "", "ui language: zh | en | ko (default: config, else zh)")
	modelID := flag.String("model", "", "model id to open first, e.g. FM1/25fbo015-15")
	selftest := flag.Bool("selftest", false, "load the first model, visit every tab in both themes and three languages, save screenshots and metrics, then exit")
	out := flag.String("out", ".", "output directory for -selftest screenshots and metrics")
	exitAfter := flag.Duration("exitafter", 0, "close the window after this duration")
	fontsize := flag.Float64("fontsize", defaultFontPx, "UI font size in pixels (10-28)")
	linkAddr := flag.String("link", "", "live link address: the studio listens on it, the teach program dials it (default 127.0.0.1:47631, or AFVI_LINK); off disables the link")
	chromeFlag := flag.Bool("chrome", runtime.GOOS == "windows", "draw the classic Win32 caption bar and resize edges instead of the OS window frame (default on Windows)")
	nativeFrame := flag.Bool("nativeframe", false, "keep the OS window frame (overrides -chrome)")
	login := flag.String("login", "", "start with a login, user:password (admin or operator; the studio accepts admin only)")
	updateFlag := flag.String("update", "", "update service base URL, or off (default "+updateURL+", or AFVI_UPDATE_URL)")
	flag.Parse()
	SetFontPx(float32(*fontsize))

	cfg, cfgPath, err := model.ResolveConfig(*cfgFlag, *ref, *machine)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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

	a := NewApp(cfg, cfgPath, themeName, profile)
	a.exitAfter = *exitAfter
	// The self-test drives the update dialog through its own step (a private server and key);
	// a real check would pop a modal in the middle of the other steps.
	updURL := *updateFlag
	if *selftest && updURL == "" {
		updURL = "off"
	}
	a.setupUpdate(updURL)
	startRole(flag.Visit, *login, *selftest, a, profile)
	// The self-test walks the programs alone: it only opens the link when -link is given.
	linkGiven := false
	flag.Visit(func(f *flag.Flag) { linkGiven = linkGiven || f.Name == "link" })
	if !*selftest || linkGiven {
		addr, on, err := resolveLinkAddr(*linkAddr, profile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if on {
			a.StartLive(addr)
		}
	}
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
	if *chromeFlag && !*nativeFrame {
		a.enableChrome() // a window hint: it must be set before the window is created
	}
	be.CreateWindow(profile.WindowTitle(), 1600, 1000)
	if a.selftest != nil {
		be.SetTargetFPS(1000)
	} else {
		be.SetTargetFPS(60)
	}

	first := 0
	for i, m := range a.Models {
		if m.ID == *modelID {
			first = i
		}
	}
	a.LoadModel(first)
	be.Run(func() { loop(a) })
	a.closeLive()
	a.closePanels()
}

// startRole applies the access level the program starts with (see JOB_CONTRACT D3):
// the self-test authenticates the admin itself, -login is checked before the window
// exists, without it the teaching program starts as the operator and the studio locked.
func startRole(visited func(func(*flag.Flag)), login string, selftest bool, a *App, profile Profile) {
	given := false
	visited(func(f *flag.Flag) { given = given || f.Name == "login" })
	if selftest {
		if role, ok := access.Authenticate("admin", "1"); ok {
			a.SetRole(role)
		}
		return
	}
	if given {
		if login == "" || !strings.Contains(login, ":") {
			fmt.Fprintln(os.Stderr, "-login must be user:password")
			os.Exit(2)
		}
		i := strings.Index(login, ":")
		user, password := login[:i], login[i+1:]
		role, ok := access.Authenticate(user, password)
		if ok && profile == ProfileStudio && role != access.RoleAdmin {
			ok = false
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "login failed")
			os.Exit(2)
		}
		a.SetRole(role)
		return
	}
	if profile == ProfileStudio {
		a.Lock()
		return
	}
	a.SetRole(access.RoleOperator)
}

// closePanels lets panels release resources (temporary files) when the window has closed.
func (a *App) closePanels() {
	for _, p := range a.allPanels {
		if c, ok := p.(interface{ Close() }); ok {
			c.Close()
		}
	}
}

func loop(a *App) {
	if a.frame == 0 {
		// The GLFW backend force-enables multi-viewport after our hook; this fixed layout must
		// stay inside the main window, otherwise edge panels pop out as separate OS windows.
		io := imgui.CurrentIO()
		io.SetConfigFlags((io.ConfigFlags() &^ imgui.ConfigFlagsViewportsEnable) | imgui.ConfigFlagsNavEnableKeyboard)
		a.be.SetWindowSizeLimits(chromeMinW, chromeMinH, -1, -1)
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
	for _, path := range fontPaths() {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		cfg := imgui.NewFontConfig()
		if !base {
			cfg.SetMergeMode(true)
		}
		atlas.AddFontFromFileTTFV(path, fontPx, cfg, nil)
		base = false
	}
}
