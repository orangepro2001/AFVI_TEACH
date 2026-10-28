// Package updater asks a release-publish ("relhub") server whether a newer build of this
// program exists, checks the Ed25519 signature of the answer and downloads the package with
// resume. Standard library only, like the other non-UI packages of this module.
//
// The wire format is documented in that service's docs/API.md: GET
// <base>/v1/apps/<app>/<platform>/latest?versionCode=<installed> answers the manifest of the
// release the channel offers (404 when nothing is published there) and
// <base>/v1/apps/<app>/<platform>/releases/<code>/download serves the package with HEAD,
// Range and If-Range.
//
// A manifest signs a canonical UTF-8 text (see signedBytes); the package bytes are covered
// by the sha256 inside that text. A manifest whose signature does not verify against one of
// the pinned keys is refused, and so is one that names another app, platform or channel: a
// valid signature on another app's manifest is still a valid signature.
package updater

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// maxPackage bounds what a manifest may offer, so a broken or hostile answer cannot make the
// program fill a disk: the installer is about 15 MB.
const maxPackage = 16 << 30

var (
	identRe  = regexp.MustCompile(`^[a-z][a-z0-9-]{0,15}$`)
	fileRe   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	hexRe    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	platform = regexp.MustCompile(`^[a-z][a-z0-9-]{0,23}$`)
)

// ErrNoUpdate is returned by Check when the installed build is the one the channel offers.
var ErrNoUpdate = errors.New("no update available")

// errCorrupt marks bytes that cannot be the package the manifest describes (the server sent
// more than it announced). The partial file is dropped then: it can never verify, and keeping
// it would make every later attempt resume from the wrong offset. An interrupted transfer, in
// contrast, keeps its .part file - that is what resume is for.
var errCorrupt = errors.New("the package does not match the manifest")

// Manifest is one release as the server describes and signs it.
type Manifest struct {
	App            string `json:"app"`
	Platform       string `json:"platform"`
	Channel        string `json:"channel"`
	VersionCode    int64  `json:"versionCode"`
	VersionName    string `json:"versionName"`
	File           string `json:"file"`
	Size           int64  `json:"size"`
	SHA256         string `json:"sha256"`
	MinVersionCode int64  `json:"minVersionCode"`
	Notes          string `json:"notes"`
	PublishedAt    string `json:"publishedAt"`
	KeyID          string `json:"keyId"`
	Signature      string `json:"signature"`
}

// Update is a verified offer, ready to download.
type Update struct {
	Manifest
	// Mandatory reports that the channel's floor is above the installed build, so the
	// program should not keep working on the old version.
	Mandatory bool
}

// Config names the app and the server, and pins the signing keys.
type Config struct {
	BaseURL  string   // e.g. https://host/release (the prefix is kept)
	App      string   // e.g. afvi-teach
	Platform string   // e.g. win-x64
	Channel  string   // e.g. stable; empty means stable
	Keys     []string // base64 std-alphabet 32-byte Ed25519 public keys
	HTTP     *http.Client
}

// Client talks to one server; create it with New.
type Client struct {
	base     string
	app      string
	platform string
	channel  string
	keys     []ed25519.PublicKey
	hc       *http.Client
}

// New validates the configuration and returns a client. A client without a pinned key is
// refused: an update nobody signed must never be installed.
func New(cfg Config) (*Client, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("updater: bad base URL %q (need http:// or https://)", cfg.BaseURL)
	}
	if !identRe.MatchString(cfg.App) {
		return nil, fmt.Errorf("updater: bad app id %q", cfg.App)
	}
	if !platform.MatchString(cfg.Platform) {
		return nil, fmt.Errorf("updater: bad platform %q", cfg.Platform)
	}
	channel := cfg.Channel
	if channel == "" {
		channel = "stable"
	}
	if !identRe.MatchString(channel) {
		return nil, fmt.Errorf("updater: bad channel %q", cfg.Channel)
	}
	keys, err := ParseKeys(cfg.Keys...)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, errors.New("updater: no signing key pinned, refusing to run")
	}
	hc := cfg.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{base: base, app: cfg.App, platform: cfg.Platform, channel: channel, keys: keys, hc: hc}, nil
}

// ParseKeys decodes base64 Ed25519 public keys; empty entries are skipped.
func ParseKeys(encoded ...string) ([]ed25519.PublicKey, error) {
	var keys []ed25519.PublicKey
	for _, s := range encoded {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			// tolerate the URL-safe alphabet and missing padding
			raw, err = base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
		}
		if err != nil {
			return nil, fmt.Errorf("updater: bad public key: %w", err)
		}
		if len(raw) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("updater: public key is %d bytes, want %d", len(raw), ed25519.PublicKeySize)
		}
		keys = append(keys, ed25519.PublicKey(raw))
	}
	return keys, nil
}

// KeyID is the short id the server puts in a manifest as a hint: the first 8 bytes of the
// SHA-256 of the raw public key, in hex. It is never used for trust, only for logs.
func KeyID(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return hex.EncodeToString(sum[:8])
}

// URL is the manifest address for an installed version code.
func (c *Client) URL(currentCode int64) string {
	return fmt.Sprintf("%s/v1/apps/%s/%s/latest?versionCode=%d", c.base, c.app, c.platform, currentCode)
}

// Check asks the server for the current release and returns it when it is newer than
// currentCode. It returns ErrNoUpdate when the channel offers nothing newer, and refuses
// (with an error) any answer whose signature or fields do not hold up.
func (c *Client) Check(ctx context.Context, currentCode int64) (*Update, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL(currentCode), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("updater: cannot reach %s: %w", c.base, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, ErrNoUpdate
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("updater: %s answered HTTP %d", c.base, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("updater: reading the manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("updater: malformed manifest: %w", err)
	}
	if err := c.verify(&m); err != nil {
		return nil, err
	}
	if m.VersionCode <= currentCode {
		return nil, ErrNoUpdate
	}
	return &Update{Manifest: m, Mandatory: m.MinVersionCode > currentCode}, nil
}

// verify checks the fields a client must not take on trust and the signature over them.
func (c *Client) verify(m *Manifest) error {
	if m.App != c.app || m.Platform != c.platform || m.Channel != c.channel {
		return fmt.Errorf("updater: the manifest is for %s/%s/%s, not %s/%s/%s",
			m.App, m.Platform, m.Channel, c.app, c.platform, c.channel)
	}
	if m.VersionCode < 1 {
		return fmt.Errorf("updater: manifest versionCode %d is not positive", m.VersionCode)
	}
	if n := len([]rune(m.VersionName)); n < 1 || n > 64 || strings.IndexFunc(m.VersionName, unicode.IsControl) >= 0 {
		return fmt.Errorf("updater: bad versionName %q", m.VersionName)
	}
	if !fileRe.MatchString(m.File) || strings.Contains(m.File, "..") {
		return fmt.Errorf("updater: bad file name %q", m.File)
	}
	if m.Size < 1 || m.Size > maxPackage {
		return fmt.Errorf("updater: implausible package size %d", m.Size)
	}
	if !hexRe.MatchString(m.SHA256) {
		return fmt.Errorf("updater: bad sha256 %q", m.SHA256)
	}
	if m.MinVersionCode < 0 || m.MinVersionCode > m.VersionCode {
		return fmt.Errorf("updater: minVersionCode %d is outside 0..%d", m.MinVersionCode, m.VersionCode)
	}
	if _, err := time.Parse(time.RFC3339, m.PublishedAt); err != nil {
		return fmt.Errorf("updater: bad publishedAt %q", m.PublishedAt)
	}
	sig, err := base64.StdEncoding.DecodeString(m.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("updater: bad signature for %s", m.VersionName)
	}
	msg := signedBytes(m)
	for _, k := range c.keys {
		if ed25519.Verify(k, msg, sig) {
			return nil
		}
	}
	return fmt.Errorf("updater: the signature of %s matches no pinned key (keyId %s)", m.VersionName, m.KeyID)
}

// SignedBytes is the exact text a manifest signs: UTF-8, every line ending in "\n", including
// the last one. The keyId is a hint and is not covered. It is exported so tools, tests and the
// program's self-test can produce (and reproduce) a signature for a local server.
func (m *Manifest) SignedBytes() []byte { return signedBytes(m) }

// Sign fills in KeyID and Signature for m with priv. A real deployment signs on the server; this
// exists for tests and for a stand-in server (the program's self-test) that proves the client
// verifies what it is given.
func Sign(m *Manifest, priv ed25519.PrivateKey) {
	m.KeyID = KeyID(priv.Public().(ed25519.PublicKey))
	m.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, signedBytes(m)))
}

// signedBytes is the exact text the server signs: UTF-8, every line ending in "\n",
// including the last one. The keyId is a hint and is not covered.
func signedBytes(m *Manifest) []byte {
	notesSum := sha256.Sum256([]byte(m.Notes))
	var b strings.Builder
	fmt.Fprintf(&b, "relhub-release-v1\n")
	fmt.Fprintf(&b, "app=%s\n", m.App)
	fmt.Fprintf(&b, "platform=%s\n", m.Platform)
	fmt.Fprintf(&b, "channel=%s\n", m.Channel)
	fmt.Fprintf(&b, "versionCode=%d\n", m.VersionCode)
	fmt.Fprintf(&b, "versionName=%s\n", m.VersionName)
	fmt.Fprintf(&b, "file=%s\n", m.File)
	fmt.Fprintf(&b, "size=%d\n", m.Size)
	fmt.Fprintf(&b, "sha256=%s\n", m.SHA256)
	fmt.Fprintf(&b, "minVersionCode=%d\n", m.MinVersionCode)
	fmt.Fprintf(&b, "publishedAt=%s\n", m.PublishedAt)
	fmt.Fprintf(&b, "notesSha256=%s\n", hex.EncodeToString(notesSum[:]))
	return []byte(b.String())
}

// DownloadURL is the address of a release package, pinned to its version code (what the
// server serves with resume support; it is not the channel and never changes).
func (c *Client) DownloadURL(m *Manifest) string {
	return fmt.Sprintf("%s/v1/apps/%s/%s/releases/%d/download", c.base, c.app, c.platform, m.VersionCode)
}

// Progress reports how many bytes of the package are on disk.
type Progress func(done, total int64)

// Download fetches the package of u to dest, resuming a partial file, and returns only when
// the size and the sha256 match the manifest. The bytes land next to dest as dest+".part"
// and are renamed when they check out, so an interrupted download never looks finished.
//
// A file that already matches is left alone (a second call is cheap).
func (c *Client) Download(ctx context.Context, u *Update, dest string, progress Progress) error {
	if ok, err := matches(dest, u.Size, u.SHA256); err != nil {
		return err
	} else if ok {
		return nil
	}
	part := dest + ".part"
	var have int64
	if st, err := os.Stat(part); err == nil {
		have = st.Size()
	} else if !os.IsNotExist(err) {
		return err
	}
	if have > u.Size {
		// nonsense from an older attempt: start over
		if err := os.Remove(part); err != nil && !os.IsNotExist(err) {
			return err
		}
		have = 0
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if have > 0 && progress != nil {
		progress(have, u.Size)
	}
	f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if have >= u.Size {
		if err := f.Truncate(have); err != nil {
			return err
		}
	}
	if have < u.Size {
		if err := c.fetch(ctx, u, f, have, progress); err != nil {
			if errors.Is(err, errCorrupt) {
				_ = f.Close()
				_ = os.Remove(part)
			}
			return err
		}
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if ok, err := matches(part, u.Size, u.SHA256); err != nil {
		return err
	} else if !ok {
		_ = os.Remove(part)
		return fmt.Errorf("updater: %s did not match the manifest (size or sha256), removed", filepath.Base(part))
	}
	if err := os.Rename(part, dest); err != nil {
		return err
	}
	return nil
}

// fetch streams the missing bytes of the package into f.
func (c *Client) fetch(ctx context.Context, u *Update, f *os.File, have int64, progress Progress) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.DownloadURL(&u.Manifest), nil)
	if err != nil {
		return err
	}
	if have > 0 {
		// If-Range pins the resume to this exact package: when the file changed the server
		// answers 200 with the whole thing instead of splicing two builds together.
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", have))
		req.Header.Set("If-Range", `"`+u.SHA256+`"`)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("updater: download failed: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		// full body (a fresh download, or the resume was refused)
		if err := f.Truncate(0); err != nil {
			return err
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		have = 0
	case http.StatusPartialContent:
		if _, err := f.Seek(have, io.SeekStart); err != nil {
			return err
		}
	default:
		return fmt.Errorf("updater: download answered HTTP %d", resp.StatusCode)
	}
	buf := make([]byte, 256<<10)
	done := have
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			done += int64(n)
			if progress != nil {
				progress(done, u.Size)
			}
			if done > u.Size {
				return fmt.Errorf("%w: the server sent more than the %d bytes it announced", errCorrupt, u.Size)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return fmt.Errorf("updater: download interrupted at %d of %d bytes: %w", done, u.Size, rerr)
		}
	}
	if done != u.Size {
		return fmt.Errorf("updater: download ends at %d of %d bytes", done, u.Size)
	}
	return nil
}

// matches reports whether path holds exactly the expected package.
func matches(path string, size int64, sum string) (bool, error) {
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if st.Size() != size {
		return false, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}
	return hex.EncodeToString(h.Sum(nil)) == sum, nil
}
