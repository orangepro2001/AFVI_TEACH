// Command mkicon regenerates the application icons of the two GUI programs:
//
//	go run ./cmd/mkicon
//
// It writes assets/icons/teach.ico and assets/icons/studio.ico, which the
// Windows resource scripts cmd/teach-imgui/teach.rc and
// cmd/camera-gui/studio.rc embed into the executables.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"afvi/teach-imgui/internal/appicon"
)

func main() {
	fs := flag.NewFlagSet("mkicon", flag.ExitOnError)
	out := fs.String("out", "assets/icons", "folder for teach.ico and studio.ico")
	pngDir := fs.String("png", "", "when set, also write <kind>-256.png there")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "mkicon:", err)
		os.Exit(2)
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "mkicon: unexpected arguments:", fs.Args())
		os.Exit(2)
	}
	if err := run(*out, *pngDir); err != nil {
		fmt.Fprintln(os.Stderr, "mkicon:", err)
		os.Exit(1)
	}
}

func run(out, pngDir string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	for _, kind := range []appicon.Kind{appicon.Teach, appicon.Studio} {
		data, err := appicon.ICO(kind)
		if err != nil {
			return err
		}
		path := filepath.Join(out, kind.String()+".ico")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
		fmt.Printf("mkicon: wrote %s (%d bytes)\n", path, len(data))
		if pngDir == "" {
			continue
		}
		if err := os.MkdirAll(pngDir, 0o755); err != nil {
			return err
		}
		img := appicon.Render(kind, 256)
		pngPath := filepath.Join(pngDir, kind.String()+"-256.png")
		f, err := os.Create(pngPath)
		if err != nil {
			return err
		}
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		err = enc.Encode(f, img)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		fmt.Printf("mkicon: wrote %s\n", pngPath)
	}
	return nil
}
