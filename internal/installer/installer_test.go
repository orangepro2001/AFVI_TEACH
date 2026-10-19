package installer

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func testManifest() Manifest {
	return Manifest{
		Product: "AFVI Teach", Version: "1.2.3", Publisher: "AFVI", Folder: "AFVI Teach", Key: "AFVITeach",
		Main:     "bin/app.exe",
		UserData: []string{"afvi.toml", "captures"},
		Shortcuts: []Shortcut{
			{Title: Text{"en": "App", "zh": "应用"}, Target: "bin/app.exe", Desktop: true},
		},
	}
}

// buildSetup writes a setup executable made of a fake stub and the given files.
func buildSetup(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	stub := filepath.Join(dir, "stub.exe")
	if err := os.WriteFile(stub, bytes.Repeat([]byte("MZ stub "), 1000), 0o644); err != nil {
		t.Fatal(err)
	}
	var list []File
	for name, content := range files {
		src := filepath.Join(dir, "src", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		list = append(list, File{Name: name, Src: src})
	}
	out := filepath.Join(dir, "setup.exe")
	if err := Build(out, stub, testManifest(), list); err != nil {
		t.Fatalf("Build: %v", err)
	}
	return out
}

var sampleFiles = map[string]string{
	"bin/app.exe":     "the program",
	"bin/tool.exe":    "the tool",
	"docs/README.md":  "# readme",
	"docs/sub/a.md":   "deep",
	"example.toml":    "x = 1",
	"reference/a.bin": strings.Repeat("0123456789", 5000),
}

func readAll(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			data, _ := os.ReadFile(p)
			rel, _ := filepath.Rel(root, p)
			out[filepath.ToSlash(rel)] = string(data)
		}
		return nil
	})
	return out
}

func TestBuildOpenExtractRoundTrip(t *testing.T) {
	setup := buildSetup(t, sampleFiles)
	p, err := Open(setup)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer p.Close()
	if p.Manifest.Product != "AFVI Teach" || p.Manifest.Version != "1.2.3" || p.Manifest.Format != ManifestFormat {
		t.Errorf("manifest %+v", p.Manifest)
	}
	names := p.Names()
	sort.Strings(names)
	var want []string
	for n := range sampleFiles {
		want = append(want, n)
	}
	sort.Strings(want)
	if !reflect.DeepEqual(names, want) {
		t.Errorf("names %v, want %v", names, want)
	}

	dir := filepath.Join(t.TempDir(), "AFVI Teach")
	var lastDone, lastTotal int64
	lg, err := p.Extract(dir, func(done, total int64, name string) { lastDone, lastTotal = done, total })
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got := readAll(t, dir); !reflect.DeepEqual(got, sampleFiles) {
		t.Errorf("extracted files differ: %d files, want %d", len(got), len(sampleFiles))
	}
	if lastDone != lastTotal || lastTotal != p.Size() {
		t.Errorf("progress ended at %d/%d, payload is %d bytes", lastDone, lastTotal, p.Size())
	}
	if !lg.RootCreated || len(lg.Files) != len(sampleFiles) {
		t.Errorf("log %+v", lg)
	}
	for _, d := range []string{"bin", "docs", "docs/sub", "reference"} {
		found := false
		for _, x := range lg.Dirs {
			found = found || x == d
		}
		if !found {
			t.Errorf("log does not list the created folder %q: %v", d, lg.Dirs)
		}
	}
	if left, _ := filepath.Glob(filepath.Join(dir, "*", "*.setup-new")); len(left) != 0 {
		t.Errorf("temporary files left: %v", left)
	}
}

// uninstall.exe is the first StubSize bytes of the setup executable: exactly the stub, no payload.
func TestStubSizeIsTheStubThatWasBuiltIn(t *testing.T) {
	setup := buildSetup(t, sampleFiles)
	p, err := Open(setup)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	stub := bytes.Repeat([]byte("MZ stub "), 1000)
	if p.Manifest.StubSize != int64(len(stub)) {
		t.Fatalf("StubSize %d, want %d", p.Manifest.StubSize, len(stub))
	}
	data, err := os.ReadFile(setup)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data[:p.Manifest.StubSize], stub) {
		t.Error("the bytes in front of the payload are not the stub")
	}
	if _, err := Open(filepath.Join(t.TempDir(), "missing.exe")); err == nil {
		t.Error("a missing file was opened")
	}
}

func TestOpenRejectsAnExecutableWithoutPayload(t *testing.T) {
	plain := filepath.Join(t.TempDir(), "plain.exe")
	if err := os.WriteFile(plain, bytes.Repeat([]byte("MZ"), 5000), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(plain); err == nil {
		t.Error("an executable without a payload was opened")
	}
}

func TestOpenRejectsUnsafeNamesAndMissingManifest(t *testing.T) {
	write := func(entries map[string]string) string {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		for name, content := range entries {
			w, err := zw.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			w.Write([]byte(content))
		}
		zw.Close()
		p := filepath.Join(t.TempDir(), "x.exe")
		os.WriteFile(p, buf.Bytes(), 0o644)
		return p
	}
	if _, err := Open(write(map[string]string{"a.txt": "x"})); err == nil {
		t.Error("a payload without manifest was opened")
	}
	good := `{"format":"afvi-setup","product":"P","version":"1","folder":"P","key":"P","main":"a.exe"}`
	for _, evil := range []string{"../evil.txt", "/abs.txt", "a/../../evil.txt", `C:\evil.txt`, `..\evil.txt`, "a:stream"} {
		if _, err := Open(write(map[string]string{"manifest.json": good, evil: "x"})); err == nil {
			t.Errorf("name %q was accepted", evil)
		}
	}
	if _, err := Open(write(map[string]string{"manifest.json": good, "a.exe": "x"})); err != nil {
		t.Errorf("a clean payload was refused: %v", err)
	}
	bad := `{"format":"other","product":"P","version":"1","folder":"P","key":"P","main":"a.exe"}`
	if _, err := Open(write(map[string]string{"manifest.json": bad})); err == nil {
		t.Error("a manifest of another format was accepted")
	}
}

func TestBuildRefusesBadInput(t *testing.T) {
	dir := t.TempDir()
	stub := filepath.Join(dir, "stub.exe")
	os.WriteFile(stub, []byte("MZ"), 0o644)
	src := filepath.Join(dir, "a.txt")
	os.WriteFile(src, []byte("a"), 0o644)
	out := filepath.Join(dir, "out.exe")
	for _, name := range []string{"../a", "/a", "manifest.json", "uninstall.json", "uninstall.exe", ""} {
		if err := Build(out, stub, testManifest(), []File{{Name: name, Src: src}}); err == nil {
			t.Errorf("name %q was accepted", name)
		}
	}
	if err := Build(out, stub, testManifest(), []File{{Name: "a", Src: src}, {Name: "a", Src: src}}); err == nil {
		t.Error("a duplicate name was accepted")
	}
	if err := Build(out, stub, testManifest(), []File{{Name: "a", Src: filepath.Join(dir, "missing")}}); err == nil {
		t.Error("a missing source was accepted")
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("a failed build left an output file")
	}
	m := testManifest()
	m.Version = ""
	if err := Build(out, stub, m, nil); err == nil {
		t.Error("a manifest without version was accepted")
	}
}

func TestUpgradeKeepsUserDataAndDropsObsoleteFiles(t *testing.T) {
	oldSetup := buildSetup(t, map[string]string{"bin/app.exe": "v1", "bin/old.dll": "obsolete", "docs/a.md": "a"})
	newSetup := buildSetup(t, map[string]string{"bin/app.exe": "v2", "docs/a.md": "a2"})
	dir := filepath.Join(t.TempDir(), "inst")

	p1, err := Open(oldSetup)
	if err != nil {
		t.Fatal(err)
	}
	lg1, err := p1.Extract(dir, nil)
	p1.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteLog(dir, lg1); err != nil {
		t.Fatal(err)
	}
	// What the programs create.
	os.WriteFile(filepath.Join(dir, "afvi.toml"), []byte("mine"), 0o644)
	os.MkdirAll(filepath.Join(dir, "captures", "m"), 0o755)
	os.WriteFile(filepath.Join(dir, "captures", "m", "red.png"), []byte("png"), 0o644)

	prev, err := ReadLog(dir)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := Open(newSetup)
	if err != nil {
		t.Fatal(err)
	}
	lg2, err := p2.Extract(dir, nil)
	p2.Close()
	if err != nil {
		t.Fatal(err)
	}
	lg2.Absorb(dir, prev)

	got := readAll(t, dir)
	if got["bin/app.exe"] != "v2" || got["docs/a.md"] != "a2" {
		t.Errorf("files not upgraded: %v", got)
	}
	if _, ok := got["bin/old.dll"]; ok {
		t.Error("the obsolete file is still there")
	}
	if got["afvi.toml"] != "mine" || got["captures/m/red.png"] != "png" {
		t.Errorf("user data was touched: %v", got)
	}
	if !lg2.RootCreated {
		t.Error("the root flag of the first installation was lost")
	}
}

func TestUninstallRemovesOnlyWhatWasInstalled(t *testing.T) {
	setup := buildSetup(t, sampleFiles)
	p, err := Open(setup)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	parent := t.TempDir()
	dir := filepath.Join(parent, "inst")
	lg, err := p.Extract(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, UninstallerName), []byte("copy"), 0o644)
	WriteLog(dir, lg)
	os.WriteFile(filepath.Join(dir, "afvi.toml"), []byte("mine"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs", "mine.txt"), []byte("user file in an installed folder"), 0o644)
	os.MkdirAll(filepath.Join(dir, "captures"), 0o755)
	os.WriteFile(filepath.Join(dir, "captures", "c.png"), []byte("c"), 0o644)

	read, err := ReadLog(dir)
	if err != nil {
		t.Fatal(err)
	}
	if failed := Uninstall(dir, read, UninstallOptions{}); len(failed) != 0 {
		t.Fatalf("failed: %v", failed)
	}
	got := readAll(t, dir)
	want := map[string]string{"afvi.toml": "mine", "docs/mine.txt": "user file in an installed folder", "captures/c.png": "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("after uninstall: %v, want only the user's files %v", got, want)
	}
	if _, err := ReadLog(dir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the log is still there: %v", err)
	}

	// Removing the user data as well empties the folder, which is then removed because the
	// installer had created it.
	os.RemoveAll(filepath.Join(dir, "docs")) // the user cleaned up the folder that kept their file
	lg2, err := p.Extract(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	lg2.RootCreated = true
	os.WriteFile(filepath.Join(dir, UninstallerName), []byte("copy"), 0o644)
	if failed := Uninstall(dir, lg2, UninstallOptions{RemoveUserData: true}); len(failed) != 0 {
		t.Fatalf("failed: %v", failed)
	}
	if left := readAll(t, dir); len(left) != 0 {
		t.Errorf("files left: %v", left)
	}
	if _, err := os.Stat(dir); err == nil {
		t.Error("the install folder was created by the installer and is empty, but still exists")
	}
}

func TestUninstallKeepsARetryLogWhenAFileIsInUse(t *testing.T) {
	if os.PathSeparator != '\\' {
		// On Windows a directory with the file's name cannot be removed as a file; emulate
		// "cannot be removed" everywhere by making the target a non-empty directory.
	}
	dir := filepath.Join(t.TempDir(), "inst")
	os.MkdirAll(filepath.Join(dir, "bin", "stuck.exe"), 0o755)
	os.WriteFile(filepath.Join(dir, "bin", "stuck.exe", "inner"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "bin", "ok.exe"), []byte("ok"), 0o644)
	os.WriteFile(filepath.Join(dir, UninstallerName), []byte("u"), 0o644)
	lg := &Log{Product: "P", Files: []string{"bin/ok.exe", "bin/stuck.exe"}, Dirs: []string{"bin"}}
	failed := Uninstall(dir, lg, UninstallOptions{})
	if len(failed) != 1 || filepath.Base(failed[0]) != "stuck.exe" {
		t.Fatalf("failed %v", failed)
	}
	if _, err := os.Stat(filepath.Join(dir, "bin", "ok.exe")); err == nil {
		t.Error("the removable file was kept")
	}
	if _, err := os.Stat(filepath.Join(dir, UninstallerName)); err != nil {
		t.Error("the uninstaller was removed although files are left")
	}
	retry, err := ReadLog(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(retry.Files, []string{"bin/stuck.exe"}) {
		t.Errorf("retry log lists %v", retry.Files)
	}
}

func TestReadLogRefusesPathsOutsideTheInstallFolder(t *testing.T) {
	for _, evil := range []string{"../x", "/etc/passwd", "a/../../x", ".", "", `..\\x`, `C:\\x`, "a:b"} {
		dir := t.TempDir()
		data := `{"product":"P","files":["` + evil + `"]}`
		os.WriteFile(filepath.Join(dir, LogName), []byte(data), 0o644)
		if _, err := ReadLog(dir); err == nil {
			t.Errorf("log naming %q was accepted", evil)
		}
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, LogName), []byte(`{"product":"P","userData":["../x"]}`), 0o644)
	if _, err := ReadLog(dir); err == nil {
		t.Error("user data outside the folder was accepted")
	}
	if _, err := ReadLog(t.TempDir()); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a missing log gives %v", err)
	}
}

func TestPathEditing(t *testing.T) {
	const dir = `C:\Users\me\AppData\Local\Programs\AFVI Teach`
	cases := []struct {
		name, in, wantAdd string
		changed           bool
	}{
		{"empty", "", dir, true},
		{"normal", `C:\Windows;C:\Tools`, `C:\Windows;C:\Tools;` + dir, true},
		{"trailing semicolon", `C:\Windows;`, `C:\Windows;` + dir, true},
		{"already there", `C:\Windows;` + dir, `C:\Windows;` + dir, false},
		{"other case and slash", `C:\Windows;c:\users\ME\appdata\local\programs\afvi teach\`, `C:\Windows;c:\users\ME\appdata\local\programs\afvi teach\`, false},
		{"quoted", `"` + dir + `";C:\Windows`, `"` + dir + `";C:\Windows`, false},
	}
	for _, c := range cases {
		got, changed := PathAdd(c.in, dir)
		if got != c.wantAdd || changed != c.changed {
			t.Errorf("%s: PathAdd(%q) = %q, %v; want %q, %v", c.name, c.in, got, changed, c.wantAdd, c.changed)
		}
	}
	if got, changed := PathRemove(`C:\Windows;`+dir+`;C:\Tools`, dir); got != `C:\Windows;C:\Tools` || !changed {
		t.Errorf("remove: %q, %v", got, changed)
	}
	if got, changed := PathRemove(`C:\Windows;%USERPROFILE%\bin;;`, dir); got != `C:\Windows;%USERPROFILE%\bin;;` || changed {
		t.Errorf("an unrelated PATH must stay exactly as it is: %q, %v", got, changed)
	}
	if got, _ := PathRemove(dir, dir); got != "" {
		t.Errorf("removing the only entry gives %q", got)
	}
	if PathContains("", "") || PathContains(`C:\a`, "") {
		t.Error("an empty folder is never on the PATH")
	}
}

func TestTextsHaveAllLanguagesAndPlaceholders(t *testing.T) {
	for key := range texts["en"] {
		enHolders := placeholders(texts["en"][key])
		for _, lang := range Languages {
			s, ok := texts[lang][key]
			if !ok || s == "" {
				t.Errorf("[%s] %q is missing", lang, key)
				continue
			}
			if got := placeholders(s); !reflect.DeepEqual(got, enHolders) {
				t.Errorf("[%s] %q uses placeholders %v, English uses %v", lang, key, got, enHolders)
			}
		}
	}
	for _, lang := range Languages {
		for key := range texts[lang] {
			if _, ok := texts["en"][key]; !ok {
				t.Errorf("[%s] %q has no English text", lang, key)
			}
		}
	}
	if got := T("zh", "copying", "name", "a.exe"); got != "正在复制 a.exe" {
		t.Errorf("T = %q", got)
	}
	if got := T("fr", "install"); got != "Install" {
		t.Errorf("an unknown language falls back to English, got %q", got)
	}
	if got := T("en", "no.such.key"); got != "no.such.key" {
		t.Errorf("an unknown key comes back as itself, got %q", got)
	}
}

func placeholders(s string) []string {
	var out []string
	for {
		i := strings.IndexByte(s, '{')
		if i < 0 {
			break
		}
		j := strings.IndexByte(s[i:], '}')
		if j < 0 {
			break
		}
		out = append(out, s[i:i+j+1])
		s = s[i+j+1:]
	}
	sort.Strings(out)
	return out
}

func TestLangFromLANGID(t *testing.T) {
	for id, want := range map[uint16]string{0x0804: "zh", 0x0404: "zh", 0x0412: "ko", 0x0409: "en", 0x0407: "en", 0x0411: "en"} {
		if got := LangFromLANGID(id); got != want {
			t.Errorf("LangFromLANGID(%#x) = %q, want %q", id, got, want)
		}
	}
	if ValidLang("ko") != "ko" || ValidLang("fr") != "" {
		t.Error("ValidLang")
	}
}

func TestTextFallbackAndExpand(t *testing.T) {
	if got := (Text{"en": "E", "zh": "Z"}).For("zh"); got != "Z" {
		t.Error(got)
	}
	if got := (Text{"en": "E"}).For("ko"); got != "E" {
		t.Error(got)
	}
	if got := (Text{"zh": "Z"}).For("ko"); got != "Z" {
		t.Error(got)
	}
	if got := Expand(`{dir}\bin;{dir}`, `C:\x`); got != `C:\x\bin;C:\x` {
		t.Error(got)
	}
}
