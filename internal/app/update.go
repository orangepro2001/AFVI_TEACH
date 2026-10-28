package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"afvi/teach-imgui/internal/updater"
	"afvi/teach-imgui/internal/version"
)

// Defaults of the in-program update check.
const (
	// updateURL is the release-publish service the shipped programs ask. It is the Tailscale
	// Serve front of the build server, so a machine that is on the tailnet reaches it over
	// HTTPS; -update / AFVI_UPDATE_URL point another build at another server, "off" disables
	// the whole thing.
	updateURL   = "https://tony-ai-pc-1.tail9f7f1f.ts.net/release"
	updateApp   = "afvi-teach"
	updatePlat  = "win-x64"
	updChannel  = "stable"
	updateFirst = 20 * time.Second  // first check: after the window is up and the model loaded
	updateEvery = 6 * time.Hour     // then this often
	updateRetry = 20 * time.Minute  // after a failure (network, wrong clock, server down)
	updateJobCh = "update-check"    // Jobs keys: one check and one download at a time
	updateJobDl = "update-download" //
)

// updateKeys are the pinned Ed25519 public keys of the update service. A manifest that does
// not verify against one of them is refused, whatever it says. Rotation: ship the next key
// here *before* the server switches (docs/BUILD.md), or add it for one run with
// AFVI_UPDATE_KEYS=<base64>[,<base64>].
var updateKeys = []string{"k4thZR+OVYSGAT/9j5F39csgGqTL0t3X6GYdNCmuRI8="} // keyId f5faa9f9858b789f

// updateState is the whole update feature: the client, what the server offers, how far the
// download is and what the dialog shows. It only changes on the UI thread, except for the
// byte counters a download goroutine keeps (atomic), which tickUpdate copies in.
type updateState struct {
	cfg     updater.Config
	client  *updater.Client
	off     bool   // the check is off (-update off, a dev build, a bad configuration)
	offWhy  string // i18n key explaining off
	lastErr string // last failure, shown in the dialog; "" = none

	checking   bool
	nextCheck  time.Time
	lastCheck  time.Time
	offer      *updater.Update // what the server offers (nil = nothing newer)
	ignoredFor int64           // version code the user said "never this one" to

	downloading bool
	dlDone      atomic.Int64
	dlTotal     atomic.Int64
	installerAt string // verified package on disk
	portable    bool   // this program runs from a green (no-install) folder
	ready       bool   // installerAt holds the offered package
	dlErr       string

	open       bool // the dialog was asked to open this frame
	closeReq   bool // the dialog should close (the self-test ends this way)
	installing bool
}

// setupUpdate configures the update client from the command line and the environment. It runs
// once, before the first frame; a configuration problem leaves the feature off with a reason
// instead of failing the program.
func (a *App) setupUpdate(urlOverride string) {
	u := &a.update
	u.cfg = updater.Config{BaseURL: updateURL, App: updateApp, Platform: updatePlat, Channel: updChannel, Keys: updateKeys}
	if v := os.Getenv("AFVI_UPDATE_URL"); v != "" {
		u.cfg.BaseURL = v
	}
	if urlOverride != "" {
		u.cfg.BaseURL = urlOverride
	}
	if extra := os.Getenv("AFVI_UPDATE_KEYS"); extra != "" {
		u.cfg.Keys = append(append([]string{}, u.cfg.Keys...), splitList(extra)...)
	}
	switch u.cfg.BaseURL {
	case "off", "none", "-":
		u.off, u.offWhy = true, "update.off.flag"
		return
	}
	if !version.Known() {
		// An unstamped build cannot name itself, so it must not decide that it is old.
		u.off, u.offWhy = true, "update.off.dev"
		return
	}
	// The green distribution is a different package (a zip over its own folder), so it asks the
	// service under its own platform name: the two never offer each other's download.
	u.portable = isPortable()
	if u.portable {
		u.cfg.Platform = portablePlatform
	}
	client, err := updater.New(u.cfg)
	if err != nil {
		u.off, u.offWhy = true, "update.off.config"
		u.lastErr = err.Error()
		return
	}
	u.client = client
	u.nextCheck = time.Now().Add(updateFirst)
}

// splitList splits a comma/space separated environment list.
func splitList(s string) []string {
	var out []string
	field := ""
	for _, r := range s {
		if r == ',' || r == ' ' || r == '\t' || r == '\n' {
			if field != "" {
				out = append(out, field)
				field = ""
			}
			continue
		}
		field += string(r)
	}
	if field != "" {
		out = append(out, field)
	}
	return out
}

// tickUpdate starts the periodic check and keeps the dialog's byte counters current. It runs
// from App.Frame, on the UI thread, before the panels are drawn.
func (a *App) tickUpdate() {
	u := &a.update
	if u.installing {
		return
	}
	if u.client != nil && !u.checking && !u.nextCheck.IsZero() && !time.Now().Before(u.nextCheck) {
		a.startUpdateCheck(false)
	}
}

// startUpdateCheck asks the server whether a newer build exists. manual = the user asked from
// the menu, which also reports "up to date".
func (a *App) startUpdateCheck(manual bool) {
	u := &a.update
	if u.client == nil || u.checking {
		if manual {
			u.open = true
		}
		return
	}
	u.checking = true
	u.lastErr = ""
	client, current := u.client, version.Code()
	a.Jobs.Run(updateJobCh, func() func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		offer, err := client.Check(ctx, current)
		return func() {
			u := &a.update
			u.checking = false
			u.lastCheck = time.Now()
			switch {
			case errors.Is(err, updater.ErrNoUpdate):
				u.offer = nil
				u.nextCheck = time.Now().Add(updateEvery)
			case err != nil:
				// the machine may be offline: keep the previous offer, try again later
				u.lastErr = err.Error()
				u.nextCheck = time.Now().Add(updateRetry)
			default:
				u.offer = offer
				u.nextCheck = time.Now().Add(updateEvery)
				if offer != nil && offer.VersionCode != u.ignoredFor {
					u.open = true // tell the user once; the download starts right away
					a.startUpdateDownload()
				}
			}
		}
	})
}

// startUpdateDownload fetches the offered package into the captures folder next to the
// executable, where the installer can run it from. The bytes are verified against the signed
// manifest before anything is offered as ready.
func (a *App) startUpdateDownload() {
	u := &a.update
	if u.downloading || u.ready || u.offer == nil || u.client == nil {
		return
	}
	dest, err := updateDownloadPath(u.offer)
	if err != nil {
		u.dlErr = err.Error()
		return
	}
	u.downloading = true
	u.dlErr = ""
	u.dlDone.Store(0)
	u.dlTotal.Store(u.offer.Size)
	client, offer := u.client, u.offer
	a.Jobs.Run(updateJobDl, func() func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		err := client.Download(ctx, offer, dest, func(done, total int64) {
			a.update.dlDone.Store(done)
			a.update.dlTotal.Store(total)
		})
		return func() {
			u := &a.update
			u.downloading = false
			if err != nil {
				u.dlErr = err.Error()
				return
			}
			u.installerAt = dest
			u.ready = true
			a.SetHint(T("update.ready", "version", offer.VersionName))
		}
	})
}

// updateDownloadPath is where the package is written: next to the program, so a user can also
// find it, in a subfolder the installer/uninstaller leaves alone.
func updateDownloadPath(offer *updater.Update) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(filepath.Dir(exe), "updates")
	return filepath.Join(dir, offer.File), nil
}

// requestInstall hands the verified package to the installer and closes this program: the
// installer replaces the files (a running exe is renamed to .old) and starts the new one.
func (a *App) requestInstall() {
	u := &a.update
	if !u.ready || u.installing || u.installerAt == "" {
		return
	}
	var err error
	if u.portable {
		err = portableInstall(u.installerAt)
	} else {
		err = launchInstaller(u.installerAt)
	}
	if err != nil {
		u.dlErr = err.Error()
		a.SetHint(T("update.err.install", "msg", err.Error()))
		return
	}
	u.installing = true
	a.SetHint(T("update.installing"))
	// give the frame a moment to show the hint before the window goes away
	go func() {
		time.Sleep(400 * time.Millisecond)
		a.be.SetShouldClose(true)
	}()
}

// openUpdateDialog shows the dialog from the menu: it reports "up to date" and "checking" too,
// which the automatic path never does (it only speaks up when there is something to install).
func (a *App) openUpdateDialog() {
	u := &a.update
	u.open = true
	if u.client != nil && u.offer == nil && !u.checking {
		a.startUpdateCheck(true)
	}
}

// updateSummary is the one line the status bar / dialog headline shows.
func (a *App) updateSummary() string {
	u := &a.update
	switch {
	case u.off:
		return T(u.offWhy)
	case u.ready && u.portable:
		return T("update.ready.portable", "version", u.offer.VersionName)
	case u.checking:
		return T("update.checking")
	case u.ready:
		return T("update.ready", "version", u.offer.VersionName)
	case u.downloading:
		return T("update.downloading", "pct", int(100*float64(u.dlDone.Load())/float64(max64(u.dlTotal.Load(), 1))))
	case u.offer != nil:
		return T("update.available", "version", u.offer.VersionName)
	case u.lastErr != "":
		return T("update.err.check", "msg", u.lastErr)
	case !u.lastCheck.IsZero():
		return T("update.uptodate")
	default:
		return T("update.idle", "version", version.String())
	}
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
