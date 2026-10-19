package installer

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Log records what an installation put on the disk, so that the uninstaller removes exactly that
// and nothing else. It is stored as uninstall.json in the install folder.
type Log struct {
	Product      string   `json:"product"`
	Version      string   `json:"version"`
	Key          string   `json:"key,omitempty"`          // name of the Add/Remove Programs key
	Files        []string `json:"files"`                  // slash separated, relative to the install folder
	Dirs         []string `json:"dirs,omitempty"`         // folders the installer created, same form
	UserData     []string `json:"userData,omitempty"`     // removed on uninstall only when the user asks
	RootCreated  bool     `json:"rootCreated,omitempty"`  // the install folder itself did not exist before
	Shortcuts    []string `json:"shortcuts,omitempty"`    // absolute paths of the .lnk files
	ShortcutDirs []string `json:"shortcutDirs,omitempty"` // absolute paths of Start Menu folders created
	PathEntry    bool     `json:"pathEntry,omitempty"`    // the install folder was added to the user's PATH
}

// InUseError reports a file that could not be replaced because a program is running from it.
type InUseError struct {
	Path string
	Err  error
}

func (e *InUseError) Error() string { return fmt.Sprintf("%s is in use: %v", e.Path, e.Err) }
func (e *InUseError) Unwrap() error { return e.Err }

// Extract writes the files of the payload below dir, creating folders as needed, and returns the
// log of what was written (also on failure, listing what was written so far). A file that exists
// is replaced; one that is running is renamed out of the way first. progress, when set, is called
// before each file with the bytes done so far and the total.
func (p *Payload) Extract(dir string, progress func(done, total int64, name string)) (*Log, error) {
	lg := &Log{Product: p.Manifest.Product, Version: p.Manifest.Version, Key: p.Manifest.Key, UserData: append([]string(nil), p.Manifest.UserData...)}
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		lg.RootCreated = true
	}
	total := p.Size()
	var done int64
	for _, f := range p.files {
		dst := filepath.Join(dir, filepath.FromSlash(f.Name))
		if f.FileInfo().IsDir() {
			if err := mkdirTracked(dst, dir, lg); err != nil {
				return lg, err
			}
			continue
		}
		if progress != nil {
			progress(done, total, f.Name)
		}
		if err := mkdirTracked(filepath.Dir(dst), dir, lg); err != nil {
			return lg, err
		}
		if err := writeZipFile(dst, f); err != nil {
			return lg, fmt.Errorf("%s: %w", f.Name, err)
		}
		lg.Files = append(lg.Files, f.Name)
		done += int64(f.UncompressedSize64)
	}
	if progress != nil {
		progress(total, total, "")
	}
	return lg, nil
}

// mkdirTracked creates path (below root) and records the folders that did not exist before.
func mkdirTracked(path, root string, lg *Log) error {
	var missing []string
	for d := path; d != root && d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(d); err == nil {
			break
		}
		missing = append(missing, d)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	for _, d := range missing {
		if rel, err := filepath.Rel(root, d); err == nil {
			lg.Dirs = append(lg.Dirs, filepath.ToSlash(rel))
		}
	}
	return nil
}

func writeZipFile(dst string, f *zip.File) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	tmp := dst + ".setup-new"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, rc) // the reader checks the CRC at the end of the entry
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		// A running program cannot be overwritten but it can be renamed.
		old := dst + ".old"
		os.Remove(old)
		if err2 := os.Rename(dst, old); err2 != nil {
			os.Remove(tmp)
			return &InUseError{Path: dst, Err: err}
		}
		if err3 := os.Rename(tmp, dst); err3 != nil {
			os.Rename(old, dst)
			os.Remove(tmp)
			return err3
		}
	}
	os.Remove(dst + ".old")
	if !f.Modified.IsZero() {
		os.Chtimes(dst, f.Modified, f.Modified)
	}
	return nil
}

// WriteLog stores the log in dir.
func WriteLog(dir string, lg *Log) error {
	data, err := json.MarshalIndent(lg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, LogName), data, 0o644)
}

// ReadLog loads the log of the installation in dir. The error satisfies
// errors.Is(err, fs.ErrNotExist) when there is none. The listed paths are checked: a log that
// names anything outside the install folder is refused.
func ReadLog(dir string) (*Log, error) {
	data, err := os.ReadFile(filepath.Join(dir, LogName))
	if err != nil {
		return nil, err
	}
	var lg Log
	if err := json.Unmarshal(data, &lg); err != nil {
		return nil, fmt.Errorf("%s: %w", LogName, err)
	}
	for _, list := range [][]string{lg.Files, lg.Dirs, lg.UserData} {
		for _, name := range list {
			if !validRel(name) || name == "." {
				return nil, fmt.Errorf("%s: unsafe path %q", LogName, name)
			}
		}
	}
	return &lg, nil
}

// Absorb merges the log of an earlier installation into the new one: files the new version no
// longer has are removed, and the folders and the root flag are carried over.
func (lg *Log) Absorb(dir string, prev *Log) {
	if prev == nil {
		return
	}
	keep := make(map[string]bool, len(lg.Files))
	for _, f := range lg.Files {
		keep[f] = true
	}
	for _, f := range prev.Files {
		if !keep[f] {
			p := filepath.Join(dir, filepath.FromSlash(f))
			os.Remove(p)
			os.Remove(p + ".old")
		}
	}
	seen := make(map[string]bool, len(lg.Dirs))
	for _, d := range lg.Dirs {
		seen[d] = true
	}
	for _, d := range prev.Dirs {
		if !seen[d] {
			lg.Dirs = append(lg.Dirs, d)
		}
	}
	lg.RootCreated = lg.RootCreated || prev.RootCreated
	lg.Shortcuts = append([]string(nil), prev.Shortcuts...)
	lg.ShortcutDirs = append([]string(nil), prev.ShortcutDirs...)
	lg.PathEntry = prev.PathEntry
}

// UninstallOptions selects what Uninstall removes besides the installed files.
type UninstallOptions struct {
	RemoveUserData bool // the files the programs created (configuration, captures)
}

// Uninstall removes the files listed in lg from dir, the folders the installer created once they
// are empty, optionally the user data, and finally the uninstaller and the log. It returns the
// paths that could not be removed (typically files of a running program); then the uninstaller
// and a log that lists only those files stay, so that the uninstallation can be run again.
// Shortcuts, the registry entry and the PATH entry belong to the caller.
func Uninstall(dir string, lg *Log, opt UninstallOptions) []string {
	var failed, failedRel []string
	for _, rel := range lg.Files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		os.Remove(p + ".old")
		os.Remove(p + ".setup-new")
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			failed = append(failed, p)
			failedRel = append(failedRel, rel)
		}
	}
	if opt.RemoveUserData {
		for _, u := range lg.UserData {
			p := filepath.Join(dir, filepath.FromSlash(u))
			if err := os.RemoveAll(p); err != nil {
				failed = append(failed, p)
			}
		}
	}
	dirs := append([]string(nil), lg.Dirs...)
	depth := func(d string) int { return strings.Count(d, "/") }
	sort.Slice(dirs, func(i, j int) bool {
		if depth(dirs[i]) != depth(dirs[j]) {
			return depth(dirs[i]) > depth(dirs[j])
		}
		return dirs[i] > dirs[j]
	})
	for _, d := range dirs {
		os.Remove(filepath.Join(dir, filepath.FromSlash(d)))
	}
	if len(failed) > 0 {
		rest := *lg
		rest.Files = failedRel
		WriteLog(dir, &rest)
		return failed
	}
	os.Remove(filepath.Join(dir, UninstallerName))
	os.Remove(filepath.Join(dir, LogName))
	if lg.RootCreated {
		os.Remove(dir)
	}
	return nil
}
