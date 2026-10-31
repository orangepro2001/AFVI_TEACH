package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func floatPtr(v float64) *float64 { return &v }

func newProviderConfig(t *testing.T, u *upstream, extra string) *Provider {
	t.Helper()
	cfg, err := ParseConfig([]byte(`{"providers":[{"name":"fake","base_url":"` + u.URL +
		`/v1","api_key":"sk-test","model":"m1"` + extra + `}]}`))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	return &cfg.Providers[0]
}

func TestChatBuildsRequest(t *testing.T) {
	u := newUpstream(t, "pong")
	p := newProviderConfig(t, u, `,"temperature":0.7,"max_tokens":128,"params":{"top_p":0.9}`)
	c := NewClient(0, 1)

	res, err := c.Chat(context.Background(), Target{Provider: p, Model: "m1"}, ChatOptions{
		Messages:    []Message{{Role: "user", Content: "hi"}},
		Temperature: floatPtr(0.2),
		MaxTokens:   42,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if res.Text != "pong" || res.PromptTokens != 11 || res.OutputTokens != 7 {
		t.Errorf("res = %+v", res)
	}
	last := u.last()
	if last.Path != "/v1/chat/completions" {
		t.Errorf("path = %q", last.Path)
	}
	if last.Auth != "Bearer sk-test" {
		t.Errorf("Authorization = %q", last.Auth)
	}
	body := last.Body
	if body["model"] != "m1" || body["stream"] != false {
		t.Errorf("model/stream = %v/%v", body["model"], body["stream"])
	}
	if got := body["temperature"]; got != 0.2 {
		t.Errorf("temperature = %v, want the per-call 0.2 (not the provider 0.7)", got)
	}
	if got := body["max_tokens"]; got != float64(42) {
		t.Errorf("max_tokens = %v, want 42", got)
	}
	if got := body["top_p"]; got != 0.9 {
		t.Errorf("top_p = %v, want the provider param 0.9", got)
	}
	msgs, _ := body["messages"].([]any)
	if len(msgs) != 1 {
		t.Errorf("messages = %v", body["messages"])
	}
}

func TestChatUsesProviderDefaults(t *testing.T) {
	u := newUpstream(t, "ok")
	p := newProviderConfig(t, u, `,"temperature":0.55,"max_tokens":77`)
	c := NewClient(0, 1)
	if _, err := c.Chat(context.Background(), Target{Provider: p, Model: "m1"}, ChatOptions{
		Messages: []Message{{Role: "user", Content: "hi"}},
	}); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	body := u.last().Body
	if body["temperature"] != 0.55 || body["max_tokens"] != float64(77) {
		t.Errorf("provider defaults not applied: temperature=%v max_tokens=%v",
			body["temperature"], body["max_tokens"])
	}
}

func TestChatRetriesServerErrors(t *testing.T) {
	u := newUpstream(t, "finally")
	u.set(func(u *upstream) { u.statuses = []int{500, 503} })
	p := newProviderConfig(t, u, "")
	c := NewClient(0, 3)
	res, err := c.Chat(context.Background(), Target{Provider: p, Model: "m1"}, ChatOptions{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if res.Text != "finally" {
		t.Errorf("Text = %q", res.Text)
	}
	if n := u.count(); n != 3 {
		t.Errorf("attempts = %d, want 3", n)
	}
}

func TestChatGivesUpAndHints(t *testing.T) {
	u := newUpstream(t, "")
	u.set(func(u *upstream) { u.statuses = []int{500, 500, 500} })
	p := newProviderConfig(t, u, "")
	c := NewClient(0, 2)
	_, err := c.Chat(context.Background(), Target{Provider: p, Model: "m1"}, ChatOptions{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if ae.Status != 500 || u.count() != 2 {
		t.Errorf("status = %d, attempts = %d", ae.Status, u.count())
	}
	if !strings.Contains(ae.Hint(), "provider side") {
		t.Errorf("hint = %q", ae.Hint())
	}
	if !strings.Contains(ae.Message, "fake upstream says no") {
		t.Errorf("message = %q", ae.Message)
	}
}

func TestChatDoesNotRetryAuthErrors(t *testing.T) {
	u := newUpstream(t, "")
	u.set(func(u *upstream) { u.statuses = []int{401} })
	p := newProviderConfig(t, u, "")
	c := NewClient(0, 3)
	_, err := c.Chat(context.Background(), Target{Provider: p, Model: "m1"}, ChatOptions{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("err = %v", err)
	}
	if u.count() != 1 {
		t.Errorf("attempts = %d, want 1", u.count())
	}
	if !strings.Contains(ae.Hint(), "authentication") {
		t.Errorf("hint = %q", ae.Hint())
	}
}

func TestChatTimeoutIsReported(t *testing.T) {
	u := newUpstream(t, "slow")
	u.set(func(u *upstream) { u.delayMs = 400 })
	p := newProviderConfig(t, u, `,"timeout":"60ms"`)
	c := NewClient(0, 1)
	_, err := c.Chat(context.Background(), Target{Provider: p, Model: "m1"}, ChatOptions{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if !strings.Contains(ae.Message, "timed out") {
		t.Errorf("message = %q", ae.Message)
	}
}

func TestChatHonoursContextCancel(t *testing.T) {
	u := newUpstream(t, "slow")
	u.set(func(u *upstream) { u.delayMs = 400 })
	p := newProviderConfig(t, u, "")
	c := NewClient(0, 3)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Chat(ctx, Target{Provider: p, Model: "m1"}, ChatOptions{
		Messages: []Message{{Role: "user", Content: "hi"}},
	}); err == nil {
		t.Fatal("expected an error for a cancelled context")
	}
}

func TestChatURLNormalisation(t *testing.T) {
	cases := map[string]string{
		"https://api.deepseek.com/v1":          "https://api.deepseek.com/v1/chat/completions",
		"https://api.deepseek.com/v1/":         "https://api.deepseek.com/v1/chat/completions",
		"https://api.deepseek.com":             "https://api.deepseek.com/chat/completions",
		"http://x.test/v1/chat/completions":    "http://x.test/v1/chat/completions",
		" http://x.test/v1/chat/completions/ ": "http://x.test/v1/chat/completions",
	}
	for in, want := range cases {
		p := &Provider{Name: "p", BaseURL: in}
		if got := p.chatURL(); got != want {
			t.Errorf("chatURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestListRemoteModels(t *testing.T) {
	u := newUpstream(t, "")
	u.set(func(u *upstream) {})
	// The fake only answers /chat/completions meaningfully; /models gets the
	// default shape, which has no data array -> error is acceptable, but a 200
	// with an empty list must not panic.
	p := newProviderConfig(t, u, "")
	c := NewClient(0, 1)
	_, err := c.ListRemote(context.Background(), p)
	if err != nil {
		t.Logf("ListRemote error (accepted): %v", err)
	}
	if u.last().Path != "/v1/models" {
		t.Errorf("path = %q", u.last().Path)
	}
}
