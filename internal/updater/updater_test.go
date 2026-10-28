package updater

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	app      = "afvi-teach"
	plat     = "win-x64"
	channel  = "stable"
	testBase = "http://updates.test/release"
)

// fixture is a running fake server plus the key it signs with.
type fixture struct {
	srv  *httptest.Server
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
	// hits counts requests per path (see requests).
	hits int32
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{pub: pub, priv: priv}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.hits, 1)
		http.NotFound(w, r)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// base is the server URL with the /release prefix the service runs under.
func (f *fixture) base() string { return f.srv.URL + "/release" }

// client returns a client pinned to the fixture key.
func (f *fixture) client(t *testing.T) *Client {
	t.Helper()
	c, err := New(Config{
		BaseURL:  f.base(),
		App:      app,
		Platform: plat,
		Channel:  channel,
		Keys:     []string{base64.StdEncoding.EncodeToString(f.pub)},
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// manifest is a plausible release of the payload below.
func (f *fixture) manifest(code int64, payload []byte) Manifest {
	sum := sha256.Sum256(payload)
	return Manifest{
		App:            app,
		Platform:       plat,
		Channel:        channel,
		VersionCode:    code,
		VersionName:    fmt.Sprintf("0.1.%d", code),
		File:           fmt.Sprintf("afvi-teach-0.1.%d-win-x64.exe", code),
		Size:           int64(len(payload)),
		SHA256:         hex.EncodeToString(sum[:]),
		MinVersionCode: 0,
		Notes:          "fix: the window opens where it should",
		PublishedAt:    "2026-10-28T15:58:10Z",
	}
}

// sign fills in keyId and signature.
func (f *fixture) sign(m Manifest) *Manifest {
	m.KeyID = KeyID(f.pub)
	m.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(f.priv, signedBytes(&m)))
	return &m
}

// serveLatest makes /latest answer m (once per call chain; the handler is replaced).
func (f *fixture) serveLatest(t *testing.T, m *Manifest, code int) {
	t.Helper()
	f.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.hits, 1)
		if r.URL.Path != "/release/v1/apps/"+app+"/"+plat+"/latest" {
			http.NotFound(w, r)
			return
		}
		if code != http.StatusOK {
			w.WriteHeader(code)
			return
		}
		body, err := json.Marshal(m)
		if err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
}

func TestSignedBytesFormat(t *testing.T) {
	// The canonical text is the whole security contract: one wrong byte and every real
	// server answer is refused. Pinned here so a refactor cannot quietly change it.
	notes := "fix: the window opens where it should"
	sum := sha256.Sum256([]byte(notes))
	m := &Manifest{
		App: app, Platform: plat, Channel: channel,
		VersionCode: 112, VersionName: "0.1.12", File: "afvi-teach-0.1.12-win-x64.exe",
		Size: 15053152, SHA256: strings.Repeat("ab", 32), MinVersionCode: 0,
		Notes: notes, PublishedAt: "2026-10-28T15:58:10Z",
	}
	want := "relhub-release-v1\n" +
		"app=afvi-teach\n" +
		"platform=win-x64\n" +
		"channel=stable\n" +
		"versionCode=112\n" +
		"versionName=0.1.12\n" +
		"file=afvi-teach-0.1.12-win-x64.exe\n" +
		"size=15053152\n" +
		"sha256=" + strings.Repeat("ab", 32) + "\n" +
		"minVersionCode=0\n" +
		"publishedAt=2026-10-28T15:58:10Z\n" +
		"notesSha256=" + hex.EncodeToString(sum[:]) + "\n"
	if got := string(signedBytes(m)); got != want {
		t.Fatalf("signed bytes changed:\n got %q\nwant %q", got, want)
	}
	// an empty notes field signs the hash of the empty string
	m.Notes = ""
	empty := sha256.Sum256(nil)
	if !strings.HasSuffix(string(signedBytes(m)), "notesSha256="+hex.EncodeToString(empty[:])+"\n") {
		t.Fatalf("empty notes must sign the SHA-256 of the empty string: %q", signedBytes(m))
	}
}

func TestKeyIDMatchesTheServer(t *testing.T) {
	// The live server's document keyId for this public key is f5faa9f9858b789f
	// (GET /v1/info); the hint must be computed the same way or logs cannot be trusted.
	keys, err := ParseKeys("k4thZR+OVYSGAT/9j5F39csgGqTL0t3X6GYdNCmuRI8=")
	if err != nil {
		t.Fatal(err)
	}
	if got := KeyID(keys[0]); got != "f5faa9f9858b789f" {
		t.Fatalf("KeyID = %s, want f5faa9f9858b789f", got)
	}
}

func TestNewRejectsBadConfig(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))
	for name, cfg := range map[string]Config{
		"no key":          {BaseURL: testBase, App: app, Platform: plat},
		"ftp url":         {BaseURL: "ftp://host/x", App: app, Platform: plat, Keys: []string{key}},
		"no scheme":       {BaseURL: "host/release", App: app, Platform: plat, Keys: []string{key}},
		"upper app":       {BaseURL: testBase, App: "AFVI", Platform: plat, Keys: []string{key}},
		"bad platform":    {BaseURL: testBase, App: app, Platform: "WIN", Keys: []string{key}},
		"bad channel":     {BaseURL: testBase, App: app, Platform: plat, Channel: "Beta!", Keys: []string{key}},
		"short key":       {BaseURL: testBase, App: app, Platform: plat, Keys: []string{"AAAA"}},
		"garbage key":     {BaseURL: testBase, App: app, Platform: plat, Keys: []string{"not base64!!"}},
		"not enough bits": {BaseURL: testBase, App: app, Platform: plat, Keys: []string{base64.StdEncoding.EncodeToString([]byte{1, 2, 3})}},
	} {
		if _, err := New(cfg); err == nil {
			t.Errorf("%s: New accepted an unusable configuration", name)
		}
	}
}

func TestCheckFindsUpdate(t *testing.T) {
	f := newFixture(t)
	payload := []byte("installer bytes")
	f.serveLatest(t, f.sign(f.manifest(112, payload)), http.StatusOK)

	u, err := f.client(t).Check(context.Background(), 111)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if u.VersionCode != 112 || u.VersionName != "0.1.112" || u.Mandatory {
		t.Fatalf("unexpected update %+v", u)
	}
	if u.File != "afvi-teach-0.1.112-win-x64.exe" {
		t.Fatalf("unexpected file %q", u.File)
	}
}

func TestCheckAsksForTheInstalledVersion(t *testing.T) {
	f := newFixture(t)
	var query string
	f.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.WriteHeader(http.StatusNotFound)
	})
	if _, err := f.client(t).Check(context.Background(), 111); !errors.Is(err, ErrNoUpdate) {
		t.Fatalf("Check = %v, want ErrNoUpdate", err)
	}
	if query != "versionCode=111" {
		t.Fatalf("query = %q, want versionCode=111", query)
	}
}

func TestCheckNothingPublished(t *testing.T) {
	f := newFixture(t) // the default handler answers 404 for everything
	if _, err := f.client(t).Check(context.Background(), 1); !errors.Is(err, ErrNoUpdate) {
		t.Fatalf("Check = %v, want ErrNoUpdate", err)
	}
}

func TestCheckUpToDate(t *testing.T) {
	f := newFixture(t)
	payload := []byte("installer bytes")
	f.serveLatest(t, f.sign(f.manifest(112, payload)), http.StatusOK)
	if _, err := f.client(t).Check(context.Background(), 112); !errors.Is(err, ErrNoUpdate) {
		t.Fatalf("Check = %v, want ErrNoUpdate", err)
	}
	if _, err := f.client(t).Check(context.Background(), 200); !errors.Is(err, ErrNoUpdate) {
		t.Fatalf("Check from a newer build = %v, want ErrNoUpdate", err)
	}
}

func TestCheckMandatory(t *testing.T) {
	f := newFixture(t)
	payload := []byte("installer bytes")
	m := f.manifest(120, payload)
	m.MinVersionCode = 115
	f.serveLatest(t, f.sign(m), http.StatusOK)
	u, err := f.client(t).Check(context.Background(), 114)
	if err != nil {
		t.Fatal(err)
	}
	if !u.Mandatory {
		t.Fatal("minVersionCode above the installed build must make the update mandatory")
	}
}

func TestCheckRefusesTamperedManifest(t *testing.T) {
	f := newFixture(t)
	payload := []byte("installer bytes")
	m := f.sign(f.manifest(112, payload))
	m.Notes = "trust me, nothing to see here" // signed text no longer matches
	f.serveLatest(t, m, http.StatusOK)
	if _, err := f.client(t).Check(context.Background(), 1); err == nil ||
		!strings.Contains(err.Error(), "matches no pinned key") {
		t.Fatalf("Check = %v, want a signature failure", err)
	}
}

func TestCheckRefusesUnsignedAndForeignManifests(t *testing.T) {
	f := newFixture(t)
	payload := []byte("installer bytes")

	// signed by the pinned key, but for another app: a valid signature is not a licence
	foreign := f.manifest(112, payload)
	foreign.App = "go-nx"
	f.serveLatest(t, f.sign(foreign), http.StatusOK)
	if _, err := f.client(t).Check(context.Background(), 1); err == nil ||
		!strings.Contains(err.Error(), "not afvi-teach") {
		t.Fatalf("Check = %v, want the app mismatch to be refused", err)
	}

	// signed by someone else's key
	otherPub, otherPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	m := f.manifest(112, payload)
	foreignSig := base64.StdEncoding.EncodeToString(ed25519.Sign(otherPriv, signedBytes(&m)))
	m.Signature = foreignSig
	m.KeyID = KeyID(otherPub)
	f.serveLatest(t, &m, http.StatusOK)
	if _, err := f.client(t).Check(context.Background(), 1); err == nil ||
		!strings.Contains(err.Error(), "matches no pinned key") {
		t.Fatalf("Check = %v, want a signature failure", err)
	}
}

func TestCheckRefusesBrokenFields(t *testing.T) {
	f := newFixture(t)
	payload := []byte("installer bytes")
	for name, breakIt := range map[string]func(*Manifest){
		"dotdot file":  func(m *Manifest) { m.File = "../evil.exe" },
		"zero size":    func(m *Manifest) { m.Size = 0 },
		"huge size":    func(m *Manifest) { m.Size = maxPackage + 1 },
		"upper sha256": func(m *Manifest) { m.SHA256 = strings.ToUpper(m.SHA256) },
		"short sha":    func(m *Manifest) { m.SHA256 = "abc" },
		"bad time":     func(m *Manifest) { m.PublishedAt = "yesterday" },
		"min above":    func(m *Manifest) { m.MinVersionCode = m.VersionCode + 1 },
		"no name":      func(m *Manifest) { m.VersionName = "" },
		"control char": func(m *Manifest) { m.VersionName = "0.1.1\n" },
	} {
		m := f.manifest(112, payload)
		breakIt(&m)
		f.serveLatest(t, f.sign(m), http.StatusOK)
		if _, err := f.client(t).Check(context.Background(), 1); err == nil {
			t.Errorf("%s: Check accepted a manifest that breaks the field rules", name)
		}
	}
}

func TestCheckRejectsGarbageBody(t *testing.T) {
	f := newFixture(t)
	f.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>not json"))
	})
	if _, err := f.client(t).Check(context.Background(), 1); err == nil ||
		!strings.Contains(err.Error(), "malformed manifest") {
		t.Fatalf("Check = %v, want a malformed-manifest error", err)
	}
}

// packageHandler serves payload at the release download path, with Range support.
func (f *fixture) packageHandler(t *testing.T, m *Manifest, payload []byte, ranges *int32) {
	t.Helper()
	f.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.hits, 1)
		if r.URL.Path != fmt.Sprintf("/release/v1/apps/%s/%s/releases/%d/download", app, plat, m.VersionCode) {
			http.NotFound(w, r)
			return
		}
		if ranges != nil {
			atomic.AddInt32(ranges, 1)
		}
		w.Header().Set("ETag", `"`+m.SHA256+`"`)
		if rng := r.Header.Get("Range"); rng != "" && r.Header.Get("If-Range") == `"`+m.SHA256+`"` {
			var start int64
			if _, err := fmt.Sscanf(rng, "bytes=%d-", &start); err == nil && start < int64(len(payload)) {
				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(payload)-1, len(payload)))
				w.WriteHeader(http.StatusPartialContent)
				_, _ = w.Write(payload[start:])
				return
			}
		}
		_, _ = w.Write(payload)
	})
}

func TestDownloadWritesTheVerifiedPackage(t *testing.T) {
	f := newFixture(t)
	payload := bytes.Repeat([]byte("afvi"), 400_000)
	m := f.sign(f.manifest(112, payload))
	f.packageHandler(t, m, payload, nil)
	dest := filepath.Join(t.TempDir(), "sub", "AFVI-Teach-Setup-0.1.112.exe")

	var lastDone, lastTotal int64
	err := f.client(t).Download(context.Background(), &Update{Manifest: *m}, dest, func(done, total int64) {
		lastDone, lastTotal = done, total
	})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("downloaded %d bytes, want %d", len(got), len(payload))
	}
	if lastDone != int64(len(payload)) || lastTotal != int64(len(payload)) {
		t.Fatalf("progress ended at %d/%d, want %d", lastDone, lastTotal, len(payload))
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatal("the .part file must be gone after a successful download")
	}
}

func TestDownloadResumes(t *testing.T) {
	f := newFixture(t)
	payload := bytes.Repeat([]byte("resume-me-"), 100_000)
	m := f.sign(f.manifest(112, payload))
	var ranges int32
	f.packageHandler(t, m, payload, &ranges)

	dir := t.TempDir()
	dest := filepath.Join(dir, "setup.exe")
	half := int64(len(payload) / 2)
	if err := os.WriteFile(dest+".part", payload[:half], 0o644); err != nil {
		t.Fatal(err)
	}
	if err := f.client(t).Download(context.Background(), &Update{Manifest: *m}, dest, nil); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if n := atomic.LoadInt32(&ranges); n == 0 {
		t.Fatal("a half-downloaded .part must be resumed with a Range request")
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("resumed download does not equal the package")
	}
}

func TestDownloadRestartsWhenThePackageChanged(t *testing.T) {
	f := newFixture(t)
	payload := bytes.Repeat([]byte("new-build-"), 50_000)
	m := f.sign(f.manifest(112, payload))
	// a stale .part from an older build: the If-Range tag no longer matches, so the server
	// answers 200 with the whole package and the tail must not be spliced onto old bytes
	f.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "" && r.Header.Get("If-Range") != `"`+m.SHA256+`"` {
			_, _ = w.Write(payload) // 200, full body
			return
		}
		_, _ = w.Write(payload)
	})
	dest := filepath.Join(t.TempDir(), "setup.exe")
	if err := os.WriteFile(dest+".part", bytes.Repeat([]byte("old-"), 1000), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := f.client(t).Download(context.Background(), &Update{Manifest: *m}, dest, nil); err != nil {
		t.Fatalf("Download: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("a 200 answer must replace the partial file, not append to it")
	}
}

func TestDownloadRefusesWrongBytes(t *testing.T) {
	f := newFixture(t)
	payload := []byte("the real package")
	m := f.sign(f.manifest(112, payload))
	f.packageHandler(t, m, []byte("something else entirely"), nil)
	dir := t.TempDir()
	dest := filepath.Join(dir, "setup.exe")
	err := f.client(t).Download(context.Background(), &Update{Manifest: *m}, dest, nil)
	if err == nil || !errors.Is(err, errCorrupt) {
		t.Fatalf("Download = %v, want a corrupt-package failure", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("a package that failed the checksum must not be left in place")
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatal("the broken .part file must be removed")
	}
}

func TestDownloadIsIdempotent(t *testing.T) {
	f := newFixture(t)
	payload := []byte("already here")
	m := f.sign(f.manifest(112, payload))
	dir := t.TempDir()
	dest := filepath.Join(dir, "setup.exe")
	if err := os.WriteFile(dest, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	before := atomic.LoadInt32(&f.hits)
	if err := f.client(t).Download(context.Background(), &Update{Manifest: *m}, dest, nil); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if atomic.LoadInt32(&f.hits) != before {
		t.Fatal("a file that already matches must not be downloaded again")
	}
}

func TestCheckGivesUpOnASlowServer(t *testing.T) {
	f := newFixture(t)
	release := make(chan struct{})
	defer close(release)
	f.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // hold the answer until the context is gone
	})
	c, err := New(Config{
		BaseURL: f.base(), App: app, Platform: plat,
		Keys: []string{base64.StdEncoding.EncodeToString(f.pub)},
		HTTP: &http.Client{Timeout: 50 * time.Millisecond},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Check(context.Background(), 1); err == nil {
		t.Fatal("Check must respect the client timeout")
	}
}

func TestURLs(t *testing.T) {
	c, err := New(Config{
		BaseURL: testBase + "/", App: app, Platform: plat, Channel: "beta",
		Keys: []string{base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := c.URL(111), testBase+"/v1/apps/afvi-teach/win-x64/latest?versionCode=111"; got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
	m := &Manifest{VersionCode: 112}
	if got, want := c.DownloadURL(m), testBase+"/v1/apps/afvi-teach/win-x64/releases/112/download"; got != want {
		t.Fatalf("DownloadURL = %q, want %q", got, want)
	}
}

// TestLiveServer checks the client against a real release-publish server, which is the only
// way to know the canonical text and the key hint agree byte for byte:
//
//	AFVI_UPDATE_LIVE=http://127.0.0.1:18790/release go test ./internal/updater -run Live
//
// It skips without the variable, so the normal test run needs no network.
func TestLiveServer(t *testing.T) {
	base := os.Getenv("AFVI_UPDATE_LIVE")
	if base == "" {
		t.Skip("set AFVI_UPDATE_LIVE=<base url> to check a running release-publish server")
	}
	c, err := New(Config{
		BaseURL:  base,
		App:      "afvi-teach",
		Platform: "win-x64",
		Keys:     []string{"k4thZR+OVYSGAT/9j5F39csgGqTL0t3X6GYdNCmuRI8="}, // the server's key, pinned
	})
	if err != nil {
		t.Fatal(err)
	}
	// from 0 an update must exist and its signature must verify
	u, err := c.Check(context.Background(), 0)
	if err != nil {
		t.Fatalf("Check(0): %v", err)
	}
	t.Logf("offered %s (%d), %d bytes, keyId %s", u.VersionName, u.VersionCode, u.Size, u.KeyID)
	if u.VersionCode < 1 {
		t.Fatalf("implausible version code %d", u.VersionCode)
	}
	// the package downloads and verifies against the signed sha256
	dest := filepath.Join(t.TempDir(), "setup.exe")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := c.Download(ctx, u, dest, nil); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if st, err := os.Stat(dest); err != nil || st.Size() != u.Size {
		t.Fatalf("downloaded file: %v (%v bytes, want %d)", err, st, u.Size)
	}
	// from the version it offers there is nothing newer
	if _, err := c.Check(context.Background(), u.VersionCode); !errors.Is(err, ErrNoUpdate) {
		t.Fatalf("Check(%d) = %v, want ErrNoUpdate", u.VersionCode, err)
	}
	// and the signature really is checked: a key that did not sign it must be refused
	otherKey := base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))
	wrong, err := New(Config{BaseURL: base, App: "afvi-teach", Platform: "win-x64", Keys: []string{otherKey}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrong.Check(context.Background(), 0); err == nil {
		t.Fatal("a manifest signed by another key was accepted")
	}
}
