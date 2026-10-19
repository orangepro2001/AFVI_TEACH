// mkinstaller assembles the setup program: it appends the installation payload (the three
// executables, the documentation, the example configuration and optionally the reference data)
// to the setup stub that cmd/setup builds. It runs on the build machine, so it is plain Go and
// needs neither cgo nor Windows. scripts/package-wsl.sh and scripts/package.ps1 call it.
//
//	go run ./cmd/mkinstaller -stub dist/setup-stub.exe -out dist/AFVI-Teach-Setup.exe -version 1.0
//	go run ./cmd/mkinstaller ... -reference reference     (also ship the reference data)
package main

import (
	"debug/pe"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"afvi/teach-imgui/internal/installer"
)

func main() {
	root := flag.String("root", ".", "repository root")
	bin := flag.String("bin", "", "folder with teach-imgui.exe, camera-gui.exe and camera.exe (default <root>/bin)")
	stub := flag.String("stub", "", "the setup program that is built from cmd/setup (required)")
	out := flag.String("out", "", "setup executable to write (required)")
	version := flag.String("version", "dev", "version shown by the setup program")
	reference := flag.String("reference", "", "folder with the reference data to ship as <install folder>/reference (optional)")
	flag.Parse()
	if *stub == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	if *bin == "" {
		*bin = filepath.Join(*root, "bin")
	}
	if err := run(*root, *bin, *stub, *out, *version, *reference); err != nil {
		fmt.Fprintln(os.Stderr, "mkinstaller:", err)
		os.Exit(1)
	}
}

func run(root, bin, stub, out, version, reference string) error {
	var files []installer.File
	add := func(name, src string) error {
		if st, err := os.Stat(src); err != nil || !st.Mode().IsRegular() {
			return fmt.Errorf("%s is missing", src)
		}
		files = append(files, installer.File{Name: name, Src: src})
		return nil
	}
	for _, exe := range []string{"teach-imgui.exe", "camera-gui.exe", "camera.exe"} {
		src := filepath.Join(bin, exe)
		if err := add(exe, src); err != nil {
			return err
		}
		if err := checkImports(src); err != nil {
			return err
		}
	}
	if err := add("afvi.example.toml", filepath.Join(root, "configs", "afvi.example.toml")); err != nil {
		return err
	}
	docs := map[string]string{
		"docs/README.md":          filepath.Join(root, "README.md"),
		"docs/MANUAL.md":          filepath.Join(root, "docs", "MANUAL.md"),
		"docs/USAGE.md":           filepath.Join(root, "docs", "USAGE.md"),
		"docs/LINK.md":            filepath.Join(root, "docs", "LINK.md"),
		"docs/INSPECT_RULES.md":   filepath.Join(root, "docs", "INSPECT_RULES.md"),
		"docs/SCENARIO.md":        filepath.Join(root, "docs", "SCENARIO.md"),
		"docs/REFERENCE_SETUP.md": filepath.Join(root, "docs", "REFERENCE_SETUP.md"),
	}
	names := make([]string, 0, len(docs))
	for n := range docs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if err := add(n, docs[n]); err != nil {
			return err
		}
	}
	if reference != "" {
		n := 0
		err := filepath.WalkDir(reference, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !d.Type().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(reference, p)
			if err != nil {
				return err
			}
			files = append(files, installer.File{Name: "reference/" + filepath.ToSlash(rel), Src: p})
			n++
			return nil
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("%s holds no files", reference)
		}
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := installer.Build(out, stub, manifest(version, reference != ""), files); err != nil {
		return err
	}
	st, err := os.Stat(out)
	if err != nil {
		return err
	}
	fmt.Printf("%s: %d files, %.1f MB\n", out, len(files), float64(st.Size())/1e6)
	return nil
}

// libDLL matches the MinGW runtime libraries (libgcc_s_seh-1.dll, libstdc++-6.dll,
// libwinpthread-1.dll...), which a clean Windows does not have.
var libDLL = regexp.MustCompile(`(?i)^lib.*\.dll$`)

// checkImports refuses an executable that needs a DLL that is not part of Windows: a program
// built without static linking starts on the build machine and fails everywhere else.
func checkImports(exe string) error {
	f, err := pe.Open(exe)
	if err != nil {
		return fmt.Errorf("%s: %w", exe, err)
	}
	defer f.Close()
	libs, err := f.ImportedLibraries()
	if err != nil {
		return fmt.Errorf("%s: %w", exe, err)
	}
	var bad []string
	for _, l := range libs {
		if libDLL.MatchString(l) {
			bad = append(bad, l)
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("%s needs %s, which a clean Windows does not have; build with static linking (scripts/build-wsl.sh does)", exe, strings.Join(bad, ", "))
	}
	return nil
}

func manifest(version string, withReference bool) installer.Manifest {
	t := func(zh, en, ko string) installer.Text { return installer.Text{"zh": zh, "en": en, "ko": ko} }
	m := installer.Manifest{
		Product:   "AFVI Teach",
		Version:   version,
		Publisher: "AFVI",
		Folder:    "AFVI Teach",
		Key:       "AFVITeach",
		Main:      "teach-imgui.exe",
		UserData:  []string{"afvi.toml", "captures"},
		Components: []installer.Text{
			t("Color AFVI Master Teach（teach-imgui.exe）：教学程序，对齐与缺陷检出",
				"Color AFVI Master Teach (teach-imgui.exe): teaching program, alignment and defect detection",
				"Color AFVI Master Teach (teach-imgui.exe): 교육 프로그램, 정렬 및 결함 검출"),
			t("Color AFVI Camera Studio（camera-gui.exe）：相机合成工作室",
				"Color AFVI Camera Studio (camera-gui.exe): camera synthesis studio",
				"Color AFVI Camera Studio (camera-gui.exe): 카메라 합성 스튜디오"),
			t("camera.exe：命令行相机模拟器（两个程序也用它保存抓拍）",
				"camera.exe: command-line camera simulator (both programs use it to save captures)",
				"camera.exe: 명령줄 카메라 시뮬레이터 (두 프로그램이 캡처 저장에 사용)"),
			t("使用说明文档和机台配置示例（afvi.example.toml）",
				"Documentation and the example machine configuration (afvi.example.toml)",
				"사용 설명서와 장비 설정 예제 (afvi.example.toml)"),
		},
		Shortcuts: []installer.Shortcut{
			{Title: t("Color AFVI Master Teach", "Color AFVI Master Teach", "Color AFVI Master Teach"), Target: "teach-imgui.exe", Desktop: true},
			{Title: t("Color AFVI Camera Studio", "Color AFVI Camera Studio", "Color AFVI Camera Studio"), Target: "camera-gui.exe", Desktop: true},
			{
				Title:   t("AFVI 命令行（camera.exe）", "AFVI Command Prompt (camera.exe)", "AFVI 명령 프롬프트 (camera.exe)"),
				Target:  "{comspec}",
				Args:    `/k set "PATH={dir};%PATH%" & camera.exe -h`,
				WorkDir: "{home}",
			},
			{Title: t("使用说明文档", "Documentation", "사용 설명서"), Target: "docs"},
			{Title: t("卸载 AFVI Teach", "Uninstall AFVI Teach", "AFVI Teach 제거"), Target: installer.UninstallerName, Args: "-uninstall"},
		},
	}
	if withReference {
		m.Components = append(m.Components, t(
			"机台参考数据（reference 文件夹）",
			"Machine reference data (reference folder)",
			"장비 참조 데이터 (reference 폴더)"))
	} else {
		m.Notes = []installer.Text{t(
			"首次使用需要机台数据：把 reference 文件夹放进安装文件夹，或在程序里打开机台配置设置数据路径，详见 docs\\REFERENCE_SETUP.md。",
			"The programs need machine data on first use: put the reference folder into the install folder, or open the machine configuration in the program and set the data paths (see docs\\REFERENCE_SETUP.md).",
			"처음 사용할 때는 장비 데이터가 필요합니다. reference 폴더를 설치 폴더에 넣거나 프로그램의 장비 설정에서 데이터 경로를 지정하세요 (docs\\REFERENCE_SETUP.md 참고).",
		)}
	}
	return m
}
