package camera

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func parseImports(t *testing.T, dir string) map[string][]string {
	t.Helper()
	out := make(map[string][]string)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir %s: %v", dir, err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ParseFile %s: %v", name, err)
		}
		var paths []string
		for _, imp := range f.Imports {
			paths = append(paths, strings.Trim(imp.Path.Value, `"`))
		}
		out[name] = paths
	}
	return out
}

func TestCameraImports(t *testing.T) {
	allowed := map[string]bool{
		"afvi/teach-imgui/internal/core":   true,
		"afvi/teach-imgui/internal/model":  true,
		"afvi/teach-imgui/internal/snap":   true,
		"afvi/teach-imgui/internal/config": true,
	}
	for file, imports := range parseImports(t, ".") {
		for _, p := range imports {
			if strings.HasPrefix(p, "afvi/teach-imgui/") {
				if !allowed[p] {
					t.Fatalf("%s imports %s", file, p)
				}
				continue
			}
			if strings.Contains(p, ".") && !strings.HasPrefix(p, "afvi/") {
				// standard library paths have no dot in the first element
				first := p
				if i := strings.Index(p, "/"); i >= 0 {
					first = p[:i]
				}
				if strings.Contains(first, ".") {
					t.Fatalf("%s imports non-stdlib %s", file, p)
				}
			}
		}
	}
}

func TestAppDoesNotImportCamera(t *testing.T) {
	for file, imports := range parseImports(t, filepath.Join("..", "app")) {
		for _, p := range imports {
			if strings.HasSuffix(p, "/internal/camera") {
				t.Fatalf("%s imports %s", file, p)
			}
		}
	}
}

func TestInspectKnowsNothingOfCapture(t *testing.T) {
	for file, imports := range parseImports(t, filepath.Join("..", "inspect")) {
		for _, p := range imports {
			if strings.HasSuffix(p, "/internal/snap") || strings.HasSuffix(p, "/internal/model") || strings.HasSuffix(p, "/internal/camera") {
				t.Fatalf("%s imports %s", file, p)
			}
		}
	}
}
