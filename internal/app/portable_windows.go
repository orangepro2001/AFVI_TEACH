//go:build windows

package app

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// portableInstall replaces the files of a green distribution with the contents of the downloaded
// zip and starts the program again.
//
// A program cannot overwrite its own running executable, so the work is done by a small batch file
// that this function writes: it waits for this process to close, copies the unpacked files over the
// folder (never deleting anything it did not ship, so afvi.toml and captures survive), starts the
// program and deletes itself. The batch file runs detached, which is why the program can exit
// immediately after starting it.
func portableInstall(zipPath string) error {
	dir, err := executableDir()
	if err != nil {
		return err
	}
	stage := filepath.Join(dir, "updates", "staged")
	if err := os.RemoveAll(stage); err != nil {
		return err
	}
	if err := unzipTo(zipPath, stage); err != nil {
		return fmt.Errorf("unpacking the update: %w", err)
	}
	if _, err := os.Stat(filepath.Join(stage, "teach-imgui.exe")); err != nil {
		return fmt.Errorf("the update package has no teach-imgui.exe: %w", err)
	}
	script := filepath.Join(dir, "updates", "apply-update.cmd")
	if err := os.WriteFile(script, []byte(swapScript(stage, dir, filepath.Join(dir, "teach-imgui.exe"))), 0o644); err != nil {
		return err
	}
	cmd := exec.Command("cmd.exe", "/c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x00000008, // DETACHED_PROCESS: it must outlive this program
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting the update script: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// unzipTo unpacks a zip into dir.
func unzipTo(zipPath, dir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		name := filepath.Clean(f.Name)
		if strings.Contains(name, "..") || filepath.IsAbs(name) {
			return fmt.Errorf("refusing to unpack %q", f.Name)
		}
		target := filepath.Join(dir, name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		src, err := f.Open()
		if err != nil {
			return err
		}
		dst, err := os.Create(target)
		if err != nil {
			src.Close()
			return err
		}
		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			dst.Close()
			return err
		}
		src.Close()
		if err := dst.Close(); err != nil {
			return err
		}
	}
	return nil
}
