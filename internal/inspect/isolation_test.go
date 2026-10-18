package inspect

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInspectIsolatedFromSimulation 保证检查包与仿真包之间没有代码依赖。
func TestInspectIsolatedFromSimulation(t *testing.T) {
	// (a) internal/inspect 下的非测试 .go 文件不得导入 afvi/teach-imgui/ 前缀的包。
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("解析 %s 失败：%v", name, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(path, "afvi/teach-imgui/") {
				t.Fatalf("文件 %s 导入了仿真包 %s", name, path)
			}
		}
	}

	// (b) ../core 下的任何 .go 文件（含测试）不得导入 afvi/teach-imgui/internal/inspect。
	coreFiles, err := filepath.Glob(filepath.Join("..", "core", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range coreFiles {
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("解析 %s 失败：%v", name, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if path == "afvi/teach-imgui/internal/inspect" {
				t.Fatalf("文件 %s 导入了检查包 %s", name, path)
			}
		}
	}
}
