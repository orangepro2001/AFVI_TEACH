package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// upstream is a fake OpenAI-compatible endpoint that records every request.
type upstream struct {
	*httptest.Server

	mu       sync.Mutex
	requests []upstreamRequest
	statuses []int // consumed one per request; empty = 200
	status   int
	reply    string
	delayMs  int
	// sse makes the fake honour `"stream": true` with real SSE frames.
	sse          bool
	chunkDelayMs int
}

type upstreamRequest struct {
	Path string
	Auth string
	Body map[string]any
	Raw  string
}

func newUpstream(t *testing.T, reply string) *upstream {
	t.Helper()
	u := &upstream{reply: reply}
	u.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := map[string]any{}
		_ = json.Unmarshal(raw, &body)

		u.mu.Lock()
		u.requests = append(u.requests, upstreamRequest{
			Path: r.URL.Path,
			Auth: r.Header.Get("Authorization"),
			Body: body,
			Raw:  string(raw),
		})
		status := u.status
		if len(u.statuses) > 0 {
			status = u.statuses[0]
			u.statuses = u.statuses[1:]
		}
		reply, delay := u.reply, u.delayMs
		sse, chunkDelay := u.sse, u.chunkDelayMs
		u.mu.Unlock()

		if sse && body["stream"] == true {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher, _ := w.(http.Flusher)
			pieces := splitPieces(reply, 4)
			for _, piece := range pieces {
				enc, _ := json.Marshal(map[string]any{
					"model":   body["model"],
					"choices": []any{map[string]any{"delta": map[string]any{"content": piece}}},
				})
				fmt.Fprintf(w, "data: %s\n\n", enc)
				if flusher != nil {
					flusher.Flush()
				}
				if chunkDelay > 0 {
					time.Sleep(time.Duration(chunkDelay) * time.Millisecond)
				}
			}
			fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":11,"completion_tokens":7}}`)
			fmt.Fprint(w, "data: [DONE]\n\n")
			if flusher != nil {
				flusher.Flush()
			}
			return
		}

		if delay > 0 {
			select {
			case <-r.Context().Done():
			case <-time.After(time.Duration(delay) * time.Millisecond):
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if status != 0 && status != http.StatusOK {
			w.WriteHeader(status)
			msg := `{"error":{"message":"fake upstream says no","type":"invalid_request_error"}}`
			if status == http.StatusUnauthorized {
				msg = `{"error":{"message":"invalid api key"}}`
			}
			_, _ = io.WriteString(w, msg)
			return
		}
		out := map[string]any{
			"model": body["model"],
			"choices": []any{map[string]any{
				"message":       map[string]any{"content": reply, "reasoning_content": ""},
				"finish_reason": "stop",
			}},
			"usage": map[string]any{"prompt_tokens": 11, "completion_tokens": 7},
		}
		_ = json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(u.Close)
	return u
}

// splitPieces cuts s into at most n roughly equal pieces.
func splitPieces(s string, n int) []string {
	r := []rune(s)
	if n < 1 || len(r) == 0 {
		return []string{s}
	}
	size := (len(r) + n - 1) / n
	var out []string
	for i := 0; i < len(r); i += size {
		end := i + size
		if end > len(r) {
			end = len(r)
		}
		out = append(out, string(r[i:end]))
	}
	return out
}

func (u *upstream) set(fn func(u *upstream)) {
	u.mu.Lock()
	defer u.mu.Unlock()
	fn(u)
}

func (u *upstream) count() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return len(u.requests)
}

func (u *upstream) last() upstreamRequest {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.requests) == 0 {
		return upstreamRequest{}
	}
	return u.requests[len(u.requests)-1]
}

func (u *upstream) messages() []map[string]any {
	last := u.last()
	raw, _ := last.Body["messages"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, m := range raw {
		if mm, ok := m.(map[string]any); ok {
			out = append(out, mm)
		}
	}
	return out
}

// newTestServer wires a Server against a fake upstream.
func newTestServer(t *testing.T, u *upstream) *Server {
	t.Helper()
	cfg, err := ParseConfig([]byte(`{
	  "default": "fake",
	  "providers": [
	    {"name": "fake", "base_url": "` + u.URL + `/v1", "api_key": "sk-test",
	     "model": "fake-chat", "models": {"big": "fake-chat-big"}, "max_tokens": 256}
	  ]
	}`))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	logger := log.New(io.Discard, "", 0)
	return NewServer(cfg, NewClient(0, 3), logger)
}

// rpc sends one JSON-RPC message and decodes the response.
func rpc(t *testing.T, s *Server, msg string) (map[string]any, bool) {
	t.Helper()
	resp, has := s.Handle(context.Background(), []byte(msg))
	if !has {
		return nil, false
	}
	out := map[string]any{}
	if err := json.Unmarshal(resp, &out); err != nil {
		t.Fatalf("响应不是 JSON: %v\n%s", err, resp)
	}
	return out, true
}

// callTool invokes a tool and returns (text, isError).
func callTool(t *testing.T, s *Server, name, args string) (string, bool) {
	t.Helper()
	resp, ok := rpc(t, s, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"`+
		name+`","arguments":`+args+`}}`)
	if !ok {
		t.Fatal("tools/call 没有响应")
	}
	if errBox, ok := resp["error"]; ok {
		t.Fatalf("tools/call 返回协议错误: %v", errBox)
	}
	result, _ := resp["result"].(map[string]any)
	content, _ := result["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("result.content 结构不对: %v", result)
	}
	block, _ := content[0].(map[string]any)
	text, _ := block["text"].(string)
	isErr, _ := result["isError"].(bool)
	return text, isErr
}
