package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const maxBodyBytes = 32 << 20

// RunHTTP serves MCP over Streamable HTTP: POST /mcp with a JSON-RPC body (single
// message or batch) answers with JSON, or with an SSE stream when the client only
// accepts text/event-stream. GET / and GET /healthz are informational.
func RunHTTP(ctx context.Context, s *Server, addr, token string, logger *log.Logger) error {
	ln, err := net.Listen("tcp", normalizeAddr(addr))
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler:           newHandler(s, token),
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	logger.Printf("HTTP 模式监听 http://%s/mcp （Bearer 令牌: %s）", ln.Addr().String(), tokenState(token))

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Printf("HTTP 服务已停止")
	return nil
}

// NewHandler is the full HTTP surface, used by RunHTTP and by the tests.
func NewHandler(s *Server, token string) http.Handler {
	return newHandler(s, token)
}

func newHandler(s *Server, token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) { handleMCP(s, w, r) })
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { handleHealthz(s, w) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost || r.Method == http.MethodOptions:
			// A reverse proxy that mounts this server under a prefix (Tailscale
			// Serve --set-path, nginx location /) may deliver the request with
			// the prefix stripped, so accept MCP on the root as well.
			handleMCP(s, w, r)
		case strings.HasSuffix(r.URL.Path, "/healthz"):
			handleHealthz(s, w)
		case r.URL.Path != "/":
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found", "mcp_endpoint": "/mcp"})
		default:
			handleInfo(s, w)
		}
	})
	return withAuth(mux, token)
}

func handleHealthz(s *Server, w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"server":    serverName,
		"version":   serverVersion,
		"sessions":  s.sessions.Len(),
		"providers": len(s.cfg.Providers),
	})
}

func handleInfo(s *Server, w http.ResponseWriter) {
	names := make([]string, 0, len(s.tools))
	for _, t := range s.tools {
		names = append(names, t.Name)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"server":    serverName,
		"version":   serverVersion,
		"transport": "streamable-http",
		"endpoint":  "/mcp",
		"tools":     names,
		"targets":   s.cfg.Describe(),
	})
}

func handleMCP(s *Server, w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		writeCORS(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeCORS(w)
		w.Header().Set("Allow", "POST, OPTIONS")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "只支持 POST /mcp（Streamable HTTP）：服务端不主动推送消息，因此没有 GET/SSE 通道",
		})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "读取请求失败: " + err.Error()})
		return
	}
	// Mid-request notifications (progress) can only travel over an event stream.
	var emit emitFunc
	streaming := acceptsSSE(r) && bytes.Contains(body, []byte("progressToken"))
	if streaming {
		writeCORS(w)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		var mu sync.Mutex
		var flusher http.Flusher
		if f, ok := w.(http.Flusher); ok {
			flusher = f
		}
		emit = func(payload []byte) error {
			mu.Lock()
			defer mu.Unlock()
			if _, err := fmt.Fprintf(w, "event: message\ndata: %s\n\n", payload); err != nil {
				return err
			}
			if flusher != nil {
				flusher.Flush()
			}
			return nil
		}
		if flusher != nil {
			flusher.Flush()
		}
	}

	var resp []byte
	var has bool
	if strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		resp, has = s.batch(r.Context(), body, emit)
	} else {
		resp, has = s.HandleEmit(r.Context(), body, emit)
	}
	if !has {
		writeCORS(w)
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if streaming || wantsSSE(r) {
		writeCORS(w)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		if !streaming {
			w.WriteHeader(http.StatusOK)
		}
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", resp)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return
	}
	writeCORS(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// withAuth enforces the Bearer token when one is configured.
func withAuth(next http.Handler, token string) http.Handler {
	if token == "" {
		return next
	}
	expected := []byte(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health probes stay open (they expose no secrets); everything else needs
		// the token, including MCP on the root when a proxy strips the prefix.
		if r.URL.Path == "/healthz" || strings.HasSuffix(r.URL.Path, "/healthz") {
			next.ServeHTTP(w, r)
			return
		}
		got := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer"))
		if subtle.ConstantTimeCompare([]byte(got), expected) != 1 {
			writeCORS(w)
			w.Header().Set("WWW-Authenticate", `Bearer realm="`+serverName+`"`)
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"error": "需要 Authorization: Bearer <token>（启动时用 -token 或 LLM_MCP_TOKEN 设置）",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeCORS(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

func writeCORS(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Mcp-Session-Id, Accept")
	h.Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	h.Set("Access-Control-Expose-Headers", "Mcp-Session-Id")
}

func acceptsSSE(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/event-stream")
}

func wantsSSE(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if !strings.Contains(accept, "text/event-stream") {
		return false
	}
	return !strings.Contains(accept, "application/json")
}

// normalizeAddr turns "8765" into "127.0.0.1:8765".
func normalizeAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "127.0.0.1:8765"
	}
	if !strings.Contains(addr, ":") {
		return "127.0.0.1:" + addr
	}
	return addr
}

func tokenState(token string) string {
	if token == "" {
		return "未启用（建议只绑定 127.0.0.1）"
	}
	return "已启用"
}
