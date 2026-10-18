// Package link is the live link between the camera studio and the teaching program: a
// small framed TCP protocol (docs/LINK.md) over which the studio publishes its current
// scenario (a snap.Meta description, no pixels) and serves rendered frames on request, so
// the teaching program can inspect what the studio draws without any image file.
//
// The package is the contract both programs are written against. It holds the message
// definitions, the framing, a Server (the studio side, which owns the scenario and a
// renderer) and a Client (the teaching side, which follows the scenario and pulls frames).
// It uses only the standard library and internal/snap.
package link

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"afvi/teach-imgui/internal/snap"
)

// Protocol is the wire protocol version; peers with a different version refuse each other.
const Protocol = 1

// DefaultAddr is where the studio listens and the teaching program dials by default.
const DefaultAddr = "127.0.0.1:47631"

// Roles announced in the hello message.
const (
	RoleStudio = "studio"
	RoleTeach  = "teach"
)

// Frame kinds a client can ask for. Test and good are the two kinds of the capture
// format; golden is the clean render of the same light setting (no defects, no noise),
// which the teaching program only needs for its golden/diff views.
const (
	KindTest   = snap.KindTest
	KindGood   = snap.KindGood
	KindGolden = "golden"
)

// Error codes carried by an error message.
const (
	CodeStale      = "stale"       // the requested revision is no longer the current one
	CodeNoScene    = "no-scene"    // nothing has been published yet
	CodeBadRequest = "bad-request" // unknown channel or kind, malformed request
	CodeRender     = "render"      // the renderer failed
)

// ErrStale is matched (errors.Is) by the error of a frame request whose revision was
// replaced by a newer scene while it was in flight.
var ErrStale = errors.New("link: the scene changed")

// ErrClosed is returned by calls on a connection that has ended.
var ErrClosed = errors.New("link: connection closed")

// RemoteError is an error message sent by the peer.
type RemoteError struct {
	Code    string
	Message string
}

func (e *RemoteError) Error() string { return "link: peer error " + e.Code + ": " + e.Message }

// Is lets errors.Is(err, ErrStale) recognise a stale reply.
func (e *RemoteError) Is(target error) bool { return target == ErrStale && e.Code == CodeStale }

// Hello is the first message each side sends after connecting.
type Hello struct {
	Protocol int    `json:"protocol"`
	Role     string `json:"role"`
	App      string `json:"app"`
	PID      int    `json:"pid"`
	// Session identifies the serving process (set by the studio); a restarted studio has a
	// new session, so revisions are only comparable within one session.
	Session string `json:"session,omitempty"`
}

// Scene is one published scenario: a revision number (strictly increasing within a
// session) and the description of the capture a camera would take (snap.Meta without
// images).
type Scene struct {
	Session string    `json:"-"`
	Rev     int64     `json:"rev"`
	Meta    snap.Meta `json:"meta"`
}

// frameRequest asks the studio to render one image of the current scene. Rev is the
// revision the client believes is current (0 = whatever is current).
type frameRequest struct {
	Rev     int64  `json:"rev"`
	Channel string `json:"channel"`
	Kind    string `json:"kind"`
}

// frameHeader precedes the pixels of a frame message; the body is Width*Height raw 8-bit
// gray bytes (Enc "raw").
type frameHeader struct {
	Rev     int64  `json:"rev"`
	Channel string `json:"channel"`
	Kind    string `json:"kind"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Enc     string `json:"enc"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type msgType uint8

const (
	typeHello    msgType = 1 // both directions, first message
	typeScene    msgType = 2 // studio -> teach, pushed after hello and on every change (id 0)
	typeGetFrame msgType = 3 // teach -> studio
	typeFrame    msgType = 4 // studio -> teach, answers a getFrame (same id)
	typeError    msgType = 5 // answers a getFrame (same id) or reports a connection problem (id 0)
)

// Every message is a 20 byte header, a JSON part and a binary body:
//
//	0   4  magic "AFVL"
//	4   1  protocol version
//	5   1  message type
//	6   2  reserved (0)
//	8   4  id (request/response matching, 0 for pushes), big endian
//	12  4  JSON length
//	16  4  body length
const headerSize = 20

var magic = [4]byte{'A', 'F', 'V', 'L'}

const (
	maxJSON = 64 << 20
	maxBody = 1 << 30
)

type message struct {
	typ  msgType
	id   uint32
	json []byte
	body []byte
}

// writeJSON marshals v and writes one message.
func writeJSON(w io.Writer, typ msgType, id uint32, v any, body []byte) error {
	var js []byte
	if v != nil {
		var err error
		if js, err = json.Marshal(v); err != nil {
			return err
		}
	}
	return writeRaw(w, typ, id, js, body)
}

// writeRaw writes one message whose JSON part is already marshalled. The header and JSON
// go out in one write, the (large) body in a second one.
func writeRaw(w io.Writer, typ msgType, id uint32, js, body []byte) error {
	if len(js) > maxJSON || len(body) > maxBody {
		return fmt.Errorf("link: message too large (%d + %d bytes)", len(js), len(body))
	}
	buf := make([]byte, headerSize, headerSize+len(js))
	copy(buf, magic[:])
	buf[4] = Protocol
	buf[5] = byte(typ)
	binary.BigEndian.PutUint32(buf[8:], id)
	binary.BigEndian.PutUint32(buf[12:], uint32(len(js)))
	binary.BigEndian.PutUint32(buf[16:], uint32(len(body)))
	buf = append(buf, js...)
	if _, err := w.Write(buf); err != nil {
		return err
	}
	if len(body) > 0 {
		if _, err := w.Write(body); err != nil {
			return err
		}
	}
	return nil
}

// readMessage reads one message; the body is allocated exactly once and owned by the caller.
func readMessage(r io.Reader) (message, error) {
	var hdr [headerSize]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return message{}, err
	}
	if [4]byte(hdr[0:4]) != magic {
		return message{}, errors.New("link: not a link peer (bad magic)")
	}
	if hdr[4] != Protocol {
		return message{}, fmt.Errorf("link: protocol version %d, want %d", hdr[4], Protocol)
	}
	jl := binary.BigEndian.Uint32(hdr[12:])
	bl := binary.BigEndian.Uint32(hdr[16:])
	if jl > maxJSON || bl > maxBody {
		return message{}, fmt.Errorf("link: message too large (%d + %d bytes)", jl, bl)
	}
	m := message{typ: msgType(hdr[5]), id: binary.BigEndian.Uint32(hdr[8:])}
	if jl > 0 {
		m.json = make([]byte, jl)
		if _, err := io.ReadFull(r, m.json); err != nil {
			return message{}, err
		}
	}
	if bl > 0 {
		m.body = make([]byte, bl)
		if _, err := io.ReadFull(r, m.body); err != nil {
			return message{}, err
		}
	}
	return m, nil
}

// handshake exchanges hello messages on a fresh connection and returns the peer's.
// Both sides write first, so neither waits for the other.
func handshake(rw io.ReadWriter, mine Hello) (Hello, error) {
	mine.Protocol = Protocol
	if err := writeJSON(rw, typeHello, 0, mine, nil); err != nil {
		return Hello{}, err
	}
	m, err := readMessage(rw)
	if err != nil {
		return Hello{}, err
	}
	if m.typ == typeError {
		var e errorBody
		_ = json.Unmarshal(m.json, &e)
		return Hello{}, &RemoteError{Code: e.Code, Message: e.Message}
	}
	if m.typ != typeHello {
		return Hello{}, fmt.Errorf("link: expected hello, got message type %d", m.typ)
	}
	var peer Hello
	if err := json.Unmarshal(m.json, &peer); err != nil {
		return Hello{}, fmt.Errorf("link: bad hello: %w", err)
	}
	if peer.Protocol != Protocol {
		return peer, fmt.Errorf("link: peer speaks protocol %d, want %d", peer.Protocol, Protocol)
	}
	return peer, nil
}

func pid() int { return os.Getpid() }
