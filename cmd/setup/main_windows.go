//go:build windows

// Command setup is the installer of AFVI Teach: a self-extracting Windows program whose payload
// (the programs, the documents and a manifest) is appended by cmd/mkinstaller. The same
// executable, stripped of its payload, is installed as uninstall.exe. See docs/BUILD.md.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"afvi/teach-imgui/internal/installer"
)

const productFallback = "AFVI Teach"

const (
	exitOK        = 0
	exitFailed    = 1
	exitUsage     = 2
	exitCancelled = 3
)

const usageText = `AFVI Teach setup

setup.exe                      install with a window
setup.exe -silent              install without a window
  -dir <folder>                install folder (default: %LOCALAPPDATA%\Programs\AFVI Teach)
  -menu=false  -desktop=false  leave out the Start Menu / desktop shortcuts
  -path                        add the install folder to the user's PATH
  -launch                      start the program when done (silent mode)
  -lang zh|en|ko               language (default: that of Windows)
  -selftest <folder>           install with the window driven by the program, write screenshots to <folder>

setup.exe -uninstall [-dir <folder>] [-silent] [-purge]
  -purge                       also delete afvi.toml and the captures folder

Exit code: 0 done, 1 failed, 2 usage, 3 cancelled.
The log is %TEMP%\afvi-setup.log.`

type cli struct {
	silent, uninstall, purge, fromTemp bool
	menu, desktop, path, launch        bool
	dir, lang, selftest                string
	parent                             uint
}

// The window and its message loop belong to one OS thread, the main one: keep the main goroutine
// on it.
func init() { runtime.LockOSThread() }

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	var c cli
	fl := flag.NewFlagSet("setup", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	fl.BoolVar(&c.silent, "silent", false, "")
	fl.BoolVar(&c.uninstall, "uninstall", false, "")
	fl.BoolVar(&c.purge, "purge", false, "")
	fl.BoolVar(&c.fromTemp, "fromtemp", false, "")
	fl.BoolVar(&c.menu, "menu", true, "")
	fl.BoolVar(&c.desktop, "desktop", true, "")
	fl.BoolVar(&c.path, "path", false, "")
	fl.BoolVar(&c.launch, "launch", false, "")
	fl.StringVar(&c.dir, "dir", "", "")
	fl.StringVar(&c.lang, "lang", "", "")
	fl.StringVar(&c.selftest, "selftest", "", "")
	fl.UintVar(&c.parent, "parent", 0, "")
	if err := fl.Parse(args); err != nil || fl.NArg() > 0 {
		text := usageText
		code := exitOK
		if err != nil && !errors.Is(err, flag.ErrHelp) {
			text = err.Error() + "\n\n" + usageText
			code = exitUsage
		} else if err == nil {
			text = fmt.Sprintf("unexpected argument %q\n\n%s", fl.Arg(0), usageText)
			code = exitUsage
		}
		messageBox(0, text, productFallback, mbOK|mbIconInfo)
		return code
	}

	self, err := os.Executable()
	if err != nil {
		messageBox(0, err.Error(), productFallback, mbOK|mbIconError)
		return exitFailed
	}
	// uninstall.exe started without arguments (a double click in the install folder) uninstalls.
	if len(args) == 0 && strings.EqualFold(filepath.Base(self), installer.UninstallerName) {
		c.uninstall = true
	}
	lang := installer.ValidLang(c.lang)
	if lang == "" {
		lang = installer.LangFromLANGID(userLangID())
	}
	if c.uninstall {
		return runUninstall(&c, self, lang)
	}
	return runSetup(&c, self, lang)
}

// --- install ------------------------------------------------------------------------------

func runSetup(c *cli, self, lang string) int {
	p, err := installer.Open(self)
	if err != nil {
		logf("open %s: %v", self, err)
		messageBox(0, err.Error(), productFallback, mbOK|mbIconError)
		return exitFailed
	}
	defer p.Close()

	prevDir, prevVer := existingInstall(&p.Manifest)
	dir := c.dir
	if dir == "" {
		dir = prevDir
	}
	if dir == "" {
		dir = defaultInstallDir(p.Manifest.Folder)
	}

	if c.silent {
		o := installOptions{dir: filepath.Clean(dir), menu: c.menu, desktop: c.desktop, path: c.path, lang: lang}
		if !filepath.IsAbs(o.dir) {
			logf("the install folder %q is not an absolute path", dir)
			return exitUsage
		}
		logf("silent install of %s %s to %s", p.Manifest.Product, p.Manifest.Version, o.dir)
		warnings, err := doInstall(p, o, nil)
		for _, w := range warnings {
			logf("warning: %s", w)
		}
		if err != nil {
			logf("failed: %v", err)
			return exitFailed
		}
		logf("done")
		if c.launch {
			if err := startDetached(filepath.Join(o.dir, filepath.FromSlash(p.Manifest.Main)), o.dir); err != nil {
				logf("start %s: %v", p.Manifest.Main, err)
			}
		}
		return exitOK
	}
	return runWizard(p, lang, dir, prevVer, c.selftest)
}

// --- uninstall ----------------------------------------------------------------------------

func runUninstall(c *cli, self, lang string) int {
	dir := c.dir
	if dir == "" {
		dir = filepath.Dir(self)
	}
	dir = filepath.Clean(dir)
	lg, err := installer.ReadLog(dir)
	if err != nil {
		logf("uninstall %s: %v", dir, err)
		if !c.silent {
			messageBox(0, installer.T(lang, "notInstalled", "product", productFallback), productFallback, mbOK|mbIconWarning)
		}
		return exitFailed
	}
	product := lg.Product
	if product == "" {
		product = productFallback
	}

	if !c.fromTemp {
		// This program lives in the folder that is about to be deleted: carry on from a copy.
		if !c.silent && messageBox(0, installer.T(lang, "uninstallConfirm", "product", product), product, mbYesNo|mbIconQuest|mbDefButton2) != idYes {
			return exitCancelled
		}
		tmp := filepath.Join(os.TempDir(), fmt.Sprintf("afvi-uninstall-%d.exe", os.Getpid()))
		if err := copyStub(self, tmp, fileSize(self)); err != nil {
			logf("copy %s: %v", self, err)
			messageBox(0, err.Error(), product, mbOK|mbIconError)
			return exitFailed
		}
		args := []string{"-uninstall", "-fromtemp", "-dir", dir, "-lang", lang, "-parent", strconv.Itoa(os.Getpid())}
		if c.silent {
			args = append(args, "-silent")
		}
		if c.purge {
			args = append(args, "-purge")
		}
		cmd := exec.Command(tmp, args...)
		cmd.Dir = os.TempDir() // never inside the folder that is going to be deleted
		if err := cmd.Start(); err != nil {
			logf("start %s: %v", tmp, err)
			os.Remove(tmp)
			messageBox(0, err.Error(), product, mbOK|mbIconError)
			return exitFailed
		}
		return exitOK
	}

	waitForProcess(uint32(c.parent))
	defer scheduleSelfDelete(self)

	purge := c.purge
	if !c.silent && !purge && hasUserData(dir, lg) {
		purge = messageBox(0, installer.T(lang, "uninstallData"), product, mbYesNo|mbIconQuest|mbDefButton2) == idYes
	}
	logf("uninstall %s from %s (purge=%v)", product, dir, purge)
	failed := uninstall(dir, lg, purge)
	if len(failed) > 0 {
		logf("not removed: %s", strings.Join(failed, "; "))
		if !c.silent {
			messageBox(0, installer.T(lang, "uninstallPartial", "product", product, "files", strings.Join(failed, "\n")), product, mbOK|mbIconWarning)
		}
		return exitFailed
	}
	logf("uninstalled")
	if !c.silent {
		messageBox(0, installer.T(lang, "uninstalled", "product", product), product, mbOK|mbIconInfo)
	}
	return exitOK
}

func fileSize(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}

// hasUserData reports whether the configuration or captures that the programs created exist.
func hasUserData(dir string, lg *installer.Log) bool {
	for _, u := range lg.UserData {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(u))); !errors.Is(err, fs.ErrNotExist) {
			return true
		}
	}
	return false
}

// waitForProcess returns when the process with the given id has ended (or after 15 s).
func waitForProcess(pid uint32) {
	if pid == 0 {
		return
	}
	const synchronize = 0x00100000
	h, err := syscall.OpenProcess(synchronize, false, pid)
	if err != nil {
		return
	}
	defer syscall.CloseHandle(h)
	syscall.WaitForSingleObject(h, 15000)
}
