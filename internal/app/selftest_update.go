package app

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"afvi/teach-imgui/internal/updater"
	"afvi/teach-imgui/internal/version"
)

// The update self-test runs the whole client path - check, signature check, download, verify -
// against a server and a key made up on the spot, so it needs no network and cannot install
// anything: it never presses "install".
const updateSelftestSize = 96 << 10

// updateSelftestInstall starts a stand-in update server that offers one signed release of
// payloadBytes, points the app's update state at it and returns the server to stop. The
// manifest is signed with a fresh key that only this test pins.
func updateSelftestInstall(a *App) (*httptest.Server, *updater.Update, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	payload := make([]byte, updateSelftestSize)
	for i := range payload {
		payload[i] = byte(i*7 + i/251)
	}
	sum := sha256.Sum256(payload)
	code := version.Code() + 1
	m := updater.Manifest{
		App:         updateApp,
		Platform:    updatePlat,
		Channel:     updChannel,
		VersionCode: code,
		VersionName: fmt.Sprintf("%s-selftest", version.Name),
		File:        "AFVI-Teach-Setup-selftest.exe",
		Size:        int64(len(payload)),
		SHA256:      hex.EncodeToString(sum[:]),
		Notes:       "selftest release: the update dialog, the download and the checksum, without a network",
		PublishedAt: time.Now().UTC().Format(time.RFC3339),
	}
	updater.Sign(&m, priv)

	mux := http.NewServeMux()
	base := "/release/v1/apps/" + updateApp + "/" + updatePlat
	mux.HandleFunc(base+"/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(m)
	})
	mux.HandleFunc(fmt.Sprintf("%s/releases/%d/download", base, code), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"`+m.SHA256+`"`)
		_, _ = w.Write(payload)
	})
	srv := httptest.NewServer(mux)

	cfg := updater.Config{
		BaseURL:  srv.URL + "/release",
		App:      updateApp,
		Platform: updatePlat,
		Channel:  updChannel,
		Keys:     []string{keyBase64(pub)},
	}
	client, err := updater.New(cfg)
	if err != nil {
		srv.Close()
		return nil, nil, err
	}
	a.update = updateState{
		cfg:    cfg,
		client: client,
		open:   true,
	}
	return srv, &updater.Update{Manifest: m}, nil
}

// keyBase64 encodes a public key the way the pinned keys are written.
func keyBase64(pub ed25519.PublicKey) string { return base64.StdEncoding.EncodeToString(pub) }

// updateSelftestCheck is the check of the update step: the offer arrived, the package is on
// disk and verified, the dialog is what it should be, and nothing was installed.
func updateSelftestCheck(a *App, want *updater.Update) (map[string]any, error) {
	u := &a.update
	if u.checking || u.downloading {
		return nil, fmt.Errorf("still busy (checking %v, downloading %v)", u.checking, u.downloading)
	}
	if u.off || u.client == nil {
		return nil, fmt.Errorf("the update check is off (%s)", u.offWhy)
	}
	if u.lastErr != "" {
		return nil, fmt.Errorf("check failed: %s", u.lastErr)
	}
	if u.offer == nil {
		return nil, fmt.Errorf("no offer, expected %s", want.VersionName)
	}
	if u.offer.VersionCode != want.VersionCode || u.offer.VersionName != want.VersionName {
		return nil, fmt.Errorf("offered %s (%d), want %s (%d)",
			u.offer.VersionName, u.offer.VersionCode, want.VersionName, want.VersionCode)
	}
	if u.dlErr != "" {
		return nil, fmt.Errorf("download failed: %s", u.dlErr)
	}
	if !u.ready {
		return nil, fmt.Errorf("the package was not downloaded and verified (ready %v)", u.ready)
	}
	st, err := os.Stat(u.installerAt)
	if err != nil {
		return nil, fmt.Errorf("the downloaded package is missing: %w", err)
	}
	if st.Size() != want.Size {
		return nil, fmt.Errorf("the package is %d bytes, want %d", st.Size(), want.Size)
	}
	return map[string]any{
		"version":     u.offer.VersionName,
		"versionCode": u.offer.VersionCode,
		"bytes":       st.Size(),
		"verified":    true,
		"summary":     a.updateSummary(),
	}, nil
}
