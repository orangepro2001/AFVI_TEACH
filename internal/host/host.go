// Package host runs a Program on one UI goroutine and connects it to the Vue front end
// over plain HTTP: commands are POSTed, frames are long-polled, images are fetched as raw
// pixels. The same handler serves the Wails window (AssetServer.Handler) and the browser
// during development and tests. Standard library plus internal/ui only.
package host

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"afvi/teach-imgui/internal/ui"
)

// Program is the application the host drives.
type Program interface {
	// Build builds one frame: it calls b.Begin/b.Finish for every slot and returns them.
	Build(b *ui.Builder) *Frame
	// Viewer returns the shared image viewer (may be nil).
	SharedViewer() *ui.Viewer
}

// Frame is the wire message the front end renders.
type Frame struct {
	Seq    int                 `json:"seq"`
	Title  string              `json:"title,omitempty"`
	Slots  map[string]*ui.Node `json:"slots"`
	Viewer *ui.ViewerOut       `json:"viewer,omitempty"`
	Meta   map[string]any      `json:"meta,omitempty"` // profile, theme, lang, fontPx, role ...
}

// Host owns the loop.
type Host struct {
	prog Program
	b    *ui.Builder

	in   chan ui.Cmd
	wake chan struct{}
	tick time.Duration

	mu     sync.Mutex
	cond   *sync.Cond
	seq    int
	last   []byte
	lastJS []byte // frame JSON without seq, for change detection
	img    imgSnap
	closed bool

	// Stats count the requests served (the Windows smoke test reads them after exit).
	Polls, Cmds, Imgs atomic.Int64
	// last frame timings in microseconds, bytes of the last frame JSON, frames published
	BuildUs, MarshalUs, FrameBytes, Frames, Steps atomic.Int64

	// Hook runs on the UI goroutine at the start of every loop turn (headless tests,
	// self-test drivers). It may feed commands with Do.
	Hook func()
}

type imgSnap struct {
	rev  int
	pyr  *pyramid
	w, h int
}

// New creates a host for prog. tick is the idle frame interval (default 33 ms).
func New(prog Program, tick time.Duration) *Host {
	if tick <= 0 {
		tick = 33 * time.Millisecond
	}
	h := &Host{prog: prog, b: ui.NewBuilder(), in: make(chan ui.Cmd, 1024), wake: make(chan struct{}, 1), tick: tick}
	h.cond = sync.NewCond(&h.mu)
	return h
}

// Do queues a command (safe from any goroutine).
func (h *Host) Do(c ui.Cmd) {
	select {
	case h.in <- c:
	default:
	}
	select {
	case h.wake <- struct{}{}:
	default:
	}
}

// Step runs exactly one UI turn synchronously: drain commands, build the frame, publish
// it if it changed. Run calls it in a loop; tests call it directly.
func (h *Host) Step() {
	if h.Hook != nil {
		h.Hook()
	}
	var cmds []ui.Cmd
	for {
		select {
		case c := <-h.in:
			cmds = append(cmds, c)
			continue
		default:
		}
		break
	}
	v := h.prog.SharedViewer()
	var widget []ui.Cmd
	for _, c := range cmds {
		switch c.Op {
		case "ptr", "view", "hover":
			if v != nil {
				v.Handle(c)
			}
		default:
			widget = append(widget, c)
		}
	}
	t0 := time.Now()
	h.b.Feed(widget)
	ui.Use(h.b)
	f := h.prog.Build(h.b)
	h.b.Drop()
	h.BuildUs.Store(time.Since(t0).Microseconds())
	h.Steps.Add(1)
	if f == nil {
		return
	}
	if v != nil {
		f.Viewer = v.Out()
	}
	t1 := time.Now()
	body, err := json.Marshal(f)
	h.MarshalUs.Store(time.Since(t1).Microseconds())
	h.FrameBytes.Store(int64(len(body)))
	if err != nil {
		body = []byte(fmt.Sprintf(`{"error":%q}`, err.Error()))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if v != nil && v.Rev != h.img.rev {
		pix, w, hh, gray := v.Pixels()
		h.img = imgSnap{rev: v.Rev, pyr: newPyramid(pix, w, hh, gray), w: w, h: hh}
	}
	if string(body) == string(h.lastJS) {
		return
	}
	h.lastJS = body
	h.seq++
	h.Frames.Add(1)
	f.Seq = h.seq
	h.last, _ = json.Marshal(f)
	h.cond.Broadcast()
}

// Run loops until ctx ends.
func (h *Host) Run(ctx context.Context) {
	t := time.NewTicker(h.tick)
	defer t.Stop()
	for {
		h.Step()
		select {
		case <-ctx.Done():
			h.mu.Lock()
			h.closed = true
			h.cond.Broadcast()
			h.mu.Unlock()
			return
		case <-t.C:
		case <-h.wake:
		}
	}
}

// Frame returns the latest frame JSON and its sequence number.
func (h *Host) Latest() (int, []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.seq, h.last
}

// WaitFrame blocks until a frame newer than seq exists (or the timeout passes).
func (h *Host) WaitFrame(seq int, timeout time.Duration) (int, []byte) {
	deadline := time.AfterFunc(timeout, func() { h.mu.Lock(); h.cond.Broadcast(); h.mu.Unlock() })
	defer deadline.Stop()
	start := time.Now()
	h.mu.Lock()
	defer h.mu.Unlock()
	for h.seq <= seq && !h.closed && time.Since(start) < timeout {
		h.cond.Wait()
	}
	return h.seq, h.last
}

// Handler serves /cmd (POST JSON array of commands), /poll?seq=N (long poll), /img
// (raw pixels) and, when static is non-nil, everything else from it.
func (h *Host) Handler(static http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/cmd", func(w http.ResponseWriter, r *http.Request) {
		h.Cmds.Add(1)
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var cmds []ui.Cmd
		if err := json.Unmarshal(body, &cmds); err != nil {
			var one ui.Cmd
			if err2 := json.Unmarshal(body, &one); err2 != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			cmds = []ui.Cmd{one}
		}
		for _, c := range cmds {
			h.Do(c)
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("/poll", func(w http.ResponseWriter, r *http.Request) {
		h.Polls.Add(1)
		seq, _ := strconv.Atoi(r.URL.Query().Get("seq"))
		n, body := h.WaitFrame(seq, 20*time.Second)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if n <= seq || body == nil {
			w.WriteHeader(204)
			return
		}
		w.Write(body)
	})
	mux.HandleFunc("/img", func(w http.ResponseWriter, r *http.Request) {
		h.Imgs.Add(1)
		h.mu.Lock()
		im := h.img
		h.mu.Unlock()
		if im.pyr == nil || im.pyr.src == nil {
			http.NotFound(w, r)
			return
		}
		lvl, _ := strconv.Atoi(r.URL.Query().Get("lvl"))
		pix, lw, lh, gray := im.pyr.get(lvl)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-W", strconv.Itoa(lw))
		w.Header().Set("X-H", strconv.Itoa(lh))
		w.Header().Set("X-Rev", strconv.Itoa(im.rev))
		w.Header().Set("X-Full-W", strconv.Itoa(im.w))
		w.Header().Set("X-Full-H", strconv.Itoa(im.h))
		if gray {
			w.Header().Set("X-Fmt", "gray")
		} else {
			w.Header().Set("X-Fmt", "rgba")
		}
		w.Write(pix)
	})
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"buildUs":%d,"marshalUs":%d,"frameBytes":%d,"frames":%d,"steps":%d,"polls":%d,"cmds":%d,"imgs":%d}`,
			h.BuildUs.Load(), h.MarshalUs.Load(), h.FrameBytes.Load(), h.Frames.Load(), h.Steps.Load(), h.Polls.Load(), h.Cmds.Load(), h.Imgs.Load())
	})
	if static != nil {
		mux.Handle("/", static)
	}
	return mux
}
