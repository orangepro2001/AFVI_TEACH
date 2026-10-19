//go:build windows

package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"afvi/teach-imgui/internal/installer"
)

type installOptions struct {
	dir     string
	menu    bool
	desktop bool
	path    bool
	lang    string
}

// logf appends a line to %TEMP%\afvi-setup.log: the setup program has no console, so this is where
// a silent installation reports what went wrong.
func logf(format string, args ...any) {
	f, err := os.OpenFile(filepath.Join(os.TempDir(), "afvi-setup.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s  %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

// copyStub writes the first size bytes of this executable, the setup program without its
// payload, as the uninstaller.
func copyStub(self, dst string, size int64) error {
	in, err := os.Open(self)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".setup-new"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = io.CopyN(out, in, size)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	os.Remove(dst)
	return os.Rename(tmp, dst)
}

// doInstall installs the payload and everything that goes with it. A failure to put the files
// there is returned as err; trouble with the extras (shortcuts, the Add/Remove entry, PATH) is
// returned as warnings, the programs are installed all the same.
func doInstall(p *installer.Payload, o installOptions, progress func(done, total int64, name string)) (warnings []string, err error) {
	m := &p.Manifest
	prev, _ := installer.ReadLog(o.dir)
	lg, err := p.Extract(o.dir, progress)
	if err != nil {
		if prev == nil {
			installer.Uninstall(o.dir, lg, installer.UninstallOptions{})
		}
		return nil, err
	}
	lg.Absorb(o.dir, prev)
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	if err := copyStub(self, filepath.Join(o.dir, installer.UninstallerName), m.StubSize); err != nil {
		return nil, fmt.Errorf("%s: %w", installer.UninstallerName, err)
	}
	if err := installer.WriteLog(o.dir, lg); err != nil {
		return nil, err
	}

	// The shortcuts of an earlier installation go; the ones wanted now are made below.
	removeShortcuts(lg.Shortcuts, nil)
	lg.Shortcuts = nil
	warnings = append(warnings, makeShortcuts(m, o, lg)...)

	size := uint32(p.Size() / 1024)
	if err := writeUninstallEntry(m, o.dir, size, time.Now().Format("20060102")); err != nil {
		warnings = append(warnings, err.Error())
	}
	if o.path {
		changed, err := editUserPath(o.dir, true)
		if err != nil {
			warnings = append(warnings, err.Error())
		}
		lg.PathEntry = lg.PathEntry || changed
	}
	if err := installer.WriteLog(o.dir, lg); err != nil {
		return warnings, err
	}
	return warnings, nil
}

// makeShortcuts creates the Start Menu entries and, when asked, the desktop ones, and records
// them in the log.
func makeShortcuts(m *installer.Manifest, o installOptions, lg *installer.Log) (warnings []string) {
	if !o.menu && !o.desktop {
		return nil
	}
	var menuDir, desktopDir string
	if o.menu {
		base, err := knownFolder(&folderPrograms)
		if err != nil {
			return []string{err.Error()}
		}
		menuDir = filepath.Join(base, m.Folder)
		if _, err := os.Stat(menuDir); errors.Is(err, fs.ErrNotExist) {
			lg.ShortcutDirs = append(lg.ShortcutDirs, menuDir)
		}
		if err := os.MkdirAll(menuDir, 0o755); err != nil {
			return []string{err.Error()}
		}
	}
	if o.desktop {
		d, err := knownFolder(&folderDesktop)
		if err != nil {
			warnings = append(warnings, err.Error())
		} else {
			desktopDir = d
		}
	}
	for _, s := range m.Shortcuts {
		title := s.Title.For(o.lang)
		target := installer.Expand(s.Target, o.dir)
		if !strings.HasPrefix(s.Target, "{") && !filepath.IsAbs(target) {
			target = filepath.Join(o.dir, filepath.FromSlash(target))
		}
		args := installer.Expand(s.Args, o.dir)
		work := o.dir
		if s.WorkDir != "" {
			work = installer.Expand(s.WorkDir, o.dir)
		}
		icon := ""
		switch {
		case s.Icon != "":
			icon = filepath.Join(o.dir, filepath.FromSlash(s.Icon))
		case strings.EqualFold(filepath.Ext(target), ".exe") && strings.HasPrefix(strings.ToLower(target), strings.ToLower(o.dir)):
			icon = target
		}
		var dirs []string
		if menuDir != "" {
			dirs = append(dirs, menuDir)
		}
		if desktopDir != "" && s.Desktop {
			dirs = append(dirs, desktopDir)
		}
		for _, d := range dirs {
			lnk := filepath.Join(d, shortcutFileName(title))
			if err := createShortcut(lnk, target, args, work, icon, title); err != nil {
				warnings = append(warnings, err.Error())
				continue
			}
			lg.Shortcuts = append(lg.Shortcuts, lnk)
		}
	}
	return warnings
}

// removeShortcuts deletes .lnk files and the Start Menu folders that are empty afterwards.
func removeShortcuts(lnks, dirs []string) {
	for _, l := range lnks {
		os.Remove(l)
	}
	for _, d := range dirs {
		os.Remove(d)
	}
}

// startDetached starts a program that keeps running after this one exits.
func startDetached(exe, dir string, args ...string) error {
	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	return cmd.Start()
}

// scheduleSelfDelete removes this executable (a copy in %TEMP%) a moment after it has exited.
func scheduleSelfDelete(path string) {
	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		CmdLine:       `cmd.exe /c ping 127.0.0.1 -n 3 >nul & del /f /q "` + path + `"`,
	}
	cmd.Start()
}

// uninstall removes an installation: the files from the log, then the shortcuts, the PATH entry
// and the Add/Remove entry. It returns the files that could not be removed.
func uninstall(dir string, lg *installer.Log, purge bool) []string {
	failed := installer.Uninstall(dir, lg, installer.UninstallOptions{RemoveUserData: purge})
	removeShortcuts(lg.Shortcuts, lg.ShortcutDirs)
	if lg.PathEntry {
		if _, err := editUserPath(dir, false); err != nil {
			logf("PATH: %v", err)
		}
	}
	if len(failed) == 0 && lg.Key != "" {
		regDeleteTreeKey(uninstallKeyPath(lg.Key))
	}
	return failed
}
