// Package app is the AFVI teaching program and camera studio UI: panels hold the state and
// the logic, record their controls into a frame tree (internal/ui) and a Vue front end
// (frontend/, win32-vue components in a Wails window) renders it. This file is the
// command-line driver both cmd/teach-imgui and cmd/camera-gui call.
package app

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"afvi/teach-imgui/internal/access"
	"afvi/teach-imgui/internal/host"
	"afvi/teach-imgui/internal/model"
)

// Run runs the teaching program (align bench + guided operator pages); cmd/teach-imgui
// calls it with the built front end. It starts on the operator pages unless -login is given.
func Run(assets fs.FS) { runProfile(ProfileTeach, assets) }

// RunStudio runs the camera studio (light + defect synthesis); cmd/camera-gui calls it.
// It starts locked (login dialog) unless -login is given and only accepts an admin login.
func RunStudio(assets fs.FS) { runProfile(ProfileStudio, assets) }

// runProfile parses the command line, loads the machine configuration and runs the UI.
func runProfile(profile Profile, assets fs.FS) {
	cfgFlag := flag.String("config", "", "machine configuration file (afvi.toml); default: AFVI_CONFIG, next to the exe, or the working directory")
	machine := flag.String("machine", "", "machine name to activate (case-insensitive)")
	ref := flag.String("ref", "", "reference directory (contains FM1/FM2/BM); upserts the simulated REFERENCE machine")
	theme := flag.String("theme", "", "ui theme: dark | light (default: config, else light)")
	lang := flag.String("lang", "", "ui language: zh | en | ko (default: config, else zh)")
	modelID := flag.String("model", "", "model id to open first, e.g. FM1/25fbo015-15")
	exitAfter := flag.Duration("exitafter", 0, "close the window after this duration")
	fontsize := flag.Float64("fontsize", defaultFontPx, "UI font size in pixels (10-28)")
	linkAddr := flag.String("link", "", "live link address: the studio listens on it, the teach program dials it (default 127.0.0.1:47631, or AFVI_LINK); off disables the link")
	login := flag.String("login", "", "start with a login, user:password (admin or operator; the studio accepts admin only)")
	web := flag.String("web", "", "serve the UI over HTTP on this address (browser/test mode) instead of opening the window")
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
	ApplyTheme(themeName)

	a := NewApp(cfg, cfgPath, themeName, profile)
	a.exitAfter = *exitAfter
	startRole(flag.Visit, *login, false, a, profile)
	addr, on, err := resolveLinkAddr(*linkAddr, profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if on {
		a.StartLive(addr)
	}

	first := 0
	for i, m := range a.Models {
		if m.ID == *modelID {
			first = i
		}
	}
	a.LoadModel(first)

	h := host.New(a, 0)
	ctx, cancel := context.WithCancel(context.Background())
	go h.Run(ctx)
	go func() { // exit after / quit requests close the process in web mode and the window otherwise
		for ctx.Err() == nil {
			time.Sleep(100 * time.Millisecond)
			if a.exitAfter > 0 && time.Since(a.start) > a.exitAfter {
				a.Quit()
				break
			}
		}
	}()

	if *web != "" {
		serveWeb(*web, h, assets, a)
	} else if err := runWindow(h, a, assets); err != nil {
		fmt.Fprintln(os.Stderr, err)
		cancel()
		a.closeLive()
		a.closePanels()
		os.Exit(1)
	}
	cancel()
	if p := os.Getenv("AFVI_HOSTLOG"); p != "" {
		_ = os.WriteFile(p, []byte(fmt.Sprintf("polls=%d cmds=%d imgs=%d\n", h.Polls.Load(), h.Cmds.Load(), h.Imgs.Load())), 0o644)
	}
	a.closeLive()
	a.closePanels()
}

// serveWeb runs the UI as a web page (development and the Playwright suite). It returns
// when the app asks to quit.
func serveWeb(addr string, h *host.Host, assets fs.FS, a *App) {
	var static http.Handler
	if assets != nil {
		static = http.FileServer(http.FS(assets))
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("AFVI UI on http://" + ln.Addr().String())
	srv := &http.Server{Handler: h.Handler(static)}
	go func() {
		for !a.quit {
			time.Sleep(100 * time.Millisecond)
		}
		_ = srv.Close()
	}()
	_ = srv.Serve(ln)
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
