package app

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"afvi/teach-imgui/internal/link"
)

// The live link joins the two programs at port level (docs/LINK.md): the camera studio
// listens and publishes the scenario it is drawing, the teaching program follows it and
// pulls the rendered images over the socket, so no image file is needed to see an effect.
// Only this file and the live_*.go files know about internal/link; the panels reach it
// through App.live.

const (
	studioProducer = "afvi-camera-studio 1"
	teachProducer  = "afvi-teach 1"

	liveQuiet       = 120 * time.Millisecond // the studio publishes this long after the last change ...
	liveMaxDelay    = 400 * time.Millisecond // ... or at most this long into a continuous burst of changes
	liveDialTimeout = 2 * time.Second
	liveRedial      = 2 * time.Second // pause between connection attempts
	liveFetchLimit  = 90 * time.Second
	liveRetryDelay  = 2 * time.Second // pause before the teach program asks again after a failed fetch
)

// Tickers are called once per frame for every panel (after finished jobs were applied);
// the defect panel uses it to publish (studio) or adopt (teach) the live scenario.
type Ticker interface{ Tick() }

// liveLink is this program's end of the live link: the server in the studio, the
// reconnecting client in the teaching program.
type liveLink struct {
	addr      string
	srv       *link.Server // studio
	listenErr string       // studio: why listening failed ("" = listening)
	cli       *liveClient  // teach
}

// resolveLinkAddr turns the -link flag (falling back to AFVI_LINK, then the default) into
// the address the studio listens on or the teaching program dials. A bare port number
// means 127.0.0.1:<port>; "off", "none", "0" and "false" disable the link; the teaching
// program dials 127.0.0.1 when the address names a wildcard host.
func resolveLinkAddr(value string, profile Profile) (string, bool, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		v = strings.TrimSpace(os.Getenv("AFVI_LINK"))
	}
	switch strings.ToLower(v) {
	case "":
		v = link.DefaultAddr
	case "off", "none", "no", "false", "0":
		return "", false, nil
	}
	if _, err := strconv.Atoi(v); err == nil {
		v = "127.0.0.1:" + v
	}
	host, port, err := net.SplitHostPort(v)
	if err != nil {
		return "", false, fmt.Errorf("-link %q: %w", v, err)
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return "", false, fmt.Errorf("-link %q: bad port %q", v, port)
	}
	if profile == ProfileTeach {
		switch host {
		case "", "0.0.0.0", "::":
			host = "127.0.0.1"
		}
	} else if host == "" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port), true, nil
}

// StartLive starts the live link on addr: the studio listens, the teaching program keeps
// dialling. A studio that cannot listen (port in use) keeps running and shows why.
func (a *App) StartLive(addr string) {
	if a.live != nil || addr == "" {
		return
	}
	l := &liveLink{addr: addr}
	if a.profile == ProfileStudio {
		srv, err := link.Listen(addr, studioProducer)
		if err != nil {
			l.listenErr = err.Error()
		} else {
			l.srv = srv
		}
	} else {
		l.cli = newLiveClient(addr, teachProducer)
	}
	a.live = l
}

// closeLive stops the link when the program exits.
func (a *App) closeLive() {
	l := a.live
	if l == nil {
		return
	}
	a.live = nil
	if l.srv != nil {
		_ = l.srv.Close()
	}
	if l.cli != nil {
		l.cli.Close()
	}
}

// tickPanels gives every panel that wants it its per-frame callback.
func (a *App) tickPanels() {
	for _, p := range a.allPanels {
		if t, ok := p.(Ticker); ok {
			t.Tick()
		}
	}
}
