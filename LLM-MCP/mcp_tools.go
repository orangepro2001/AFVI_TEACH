package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// args decodes tool arguments. Unknown fields are ignored so a client that sends
// extra keys still works.
func decodeArgs(raw json.RawMessage, dst any) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	return nil
}

// askArgs is the input of the `ask` tool.
type askArgs struct {
	Prompt      string   `json:"prompt"`
	System      string   `json:"system,omitempty"`
	Model       string   `json:"model,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Session     string   `json:"session,omitempty"`
	Verbose     bool     `json:"verbose,omitempty"`
}

// chatArgs is the input of the `chat` tool.
type chatArgs struct {
	Messages    []Message `json:"messages"`
	System      string    `json:"system,omitempty"`
	Model       string    `json:"model,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Session     string    `json:"session,omitempty"`
	Verbose     bool      `json:"verbose,omitempty"`
}

func (s *Server) toolAsk() Tool {
	return Tool{
		Name: "ask",
		Description: "Ask an external LLM one self-contained question and get its answer as plain text. " +
			"The answer streams back and progress notifications are sent while it generates, so a slow model " +
			"does not trip your request timeout - but keep each call small: ask for one thing at a time and " +
			"split work that would produce more than a few hundred lines. " +
			"No credentials are needed: the server holds the API keys; `model` only routes the question. " +
			"The remote model has no tools, no repo access and no memory beyond this call " +
			"unless you pass a `session` id: put every fact it needs into `prompt`. Good for a second opinion, " +
			"review of a snippet, translation, summarising long text, or generating boilerplate. " +
			s.routingHint(),
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{
					"type":        "string",
					"description": "The full question or task, including any code/text it must reason about.",
				},
				"system": map[string]any{
					"type":        "string",
					"description": "Optional system prompt (role, output format, language).",
				},
				"model": map[string]any{
					"type": "string",
					"description": "Optional routing override. Leave empty for ordinary/medium tasks; use the " +
						"expensive high-effort target for genuinely hard or high-stakes work. Accepts a provider " +
						"name, provider/alias, an alias, or a literal model id. " + s.routingHint(),
				},
				"temperature": map[string]any{
					"type":        "number",
					"description": "Sampling temperature 0..2. Defaults to the provider setting.",
					"minimum":     0,
					"maximum":     2,
				},
				"max_tokens": map[string]any{
					"type":        "integer",
					"description": "Upper bound for the answer length. Defaults to the provider setting.",
					"minimum":     1,
				},
				"session": map[string]any{
					"type": "string",
					"description": "Optional conversation id. Reuse it in later calls to keep the thread " +
						"(the server keeps the last ~60 messages). Omit for a stateless call.",
				},
				"verbose": map[string]any{
					"type":        "boolean",
					"description": "Append a footer with model, token usage and elapsed time.",
				},
			},
			"required":             []string{"prompt"},
			"additionalProperties": false,
		},
		Handler: s.handleAsk,
	}
}

func (s *Server) handleAsk(ctx context.Context, raw json.RawMessage, rep progressReporter) (string, error) {
	var a askArgs
	if err := decodeArgs(raw, &a); err != nil {
		return "", err
	}
	if strings.TrimSpace(a.Prompt) == "" {
		return "", errors.New("prompt is required")
	}
	msgs := s.sessions.Get(a.Session)
	if a.System != "" {
		msgs = withSystem(msgs, a.System)
	}
	msgs = append(msgs, Message{Role: "user", Content: a.Prompt})
	res, err := s.complete(ctx, a.Model, msgs, a.Temperature, a.MaxTokens, rep)
	if err != nil {
		return "", err
	}
	s.sessions.Append(a.Session,
		Message{Role: "user", Content: a.Prompt},
		Message{Role: "assistant", Content: res.Text})
	return renderResult(res, a.Verbose), nil
}

func (s *Server) toolChat() Tool {
	return Tool{
		Name: "chat",
		Description: "Send a whole OpenAI-style message list to the external LLM and get the next assistant " +
			"message. Use it when you need full control of the transcript (few-shot examples, continuing a " +
			"conversation you already hold, or several system messages). For one question prefer `ask`. " +
			s.routingHint(),
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"messages": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"role": map[string]any{
								"type": "string",
								"enum": []string{"system", "user", "assistant"},
							},
							"content": map[string]any{"type": "string"},
						},
						"required": []string{"role", "content"},
					},
					"description": "The transcript so far, oldest first.",
				},
				"system": map[string]any{
					"type":        "string",
					"description": "Optional system prompt prepended when the transcript has none.",
				},
				"model": map[string]any{
					"type":        "string",
					"description": "Optional routing override, same forms as in `ask`. " + s.routingHint(),
				},
				"temperature": map[string]any{"type": "number", "minimum": 0, "maximum": 2},
				"max_tokens":  map[string]any{"type": "integer", "minimum": 1},
				"session": map[string]any{
					"type":        "string",
					"description": "Optional conversation id: the stored history is prepended to `messages`.",
				},
				"verbose": map[string]any{
					"type":        "boolean",
					"description": "Append a footer with model, token usage and elapsed time.",
				},
			},
			"required": []string{"messages"},
		},
		Handler: s.handleChat,
	}
}

func (s *Server) handleChat(ctx context.Context, raw json.RawMessage, rep progressReporter) (string, error) {
	var a chatArgs
	if err := decodeArgs(raw, &a); err != nil {
		return "", err
	}
	if len(a.Messages) == 0 {
		return "", errors.New("messages is required")
	}
	for i, m := range a.Messages {
		if strings.TrimSpace(m.Content) == "" {
			return "", fmt.Errorf("messages[%d].content is empty", i)
		}
		if m.Role == "" {
			a.Messages[i].Role = "user"
		}
	}
	msgs := s.sessions.Get(a.Session)
	if a.System != "" {
		msgs = withSystem(msgs, a.System)
	}
	msgs = append(msgs, a.Messages...)
	res, err := s.complete(ctx, a.Model, msgs, a.Temperature, a.MaxTokens, rep)
	if err != nil {
		return "", err
	}
	stored := append([]Message{}, a.Messages...)
	stored = append(stored, Message{Role: "assistant", Content: res.Text})
	s.sessions.Append(a.Session, stored...)
	return renderResult(res, a.Verbose), nil
}

func (s *Server) toolListModels() Tool {
	return Tool{
		Name: "list_models",
		Description: "List the providers and models this server can reach (configured aliases, default model, " +
			"whether an API key is present). Set `remote` to also query each endpoint's /models API.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"remote": map[string]any{
					"type":        "boolean",
					"description": "Also ask every endpoint for its model list (one HTTP request per provider).",
				},
				"provider": map[string]any{
					"type":        "string",
					"description": "Only report this provider.",
				},
			},
		},
		Handler: s.handleListModels,
	}
}

func (s *Server) handleListModels(ctx context.Context, raw json.RawMessage, _ progressReporter) (string, error) {
	var a struct {
		Remote   bool   `json:"remote"`
		Provider string `json:"provider"`
	}
	if err := decodeArgs(raw, &a); err != nil {
		return "", err
	}
	out := struct {
		Server    string     `json:"server"`
		Version   string     `json:"version"`
		Default   string     `json:"default"`
		Providers []Describe `json:"providers"`
		Remote    map[string][]string
		Sessions  int `json:"sessions"`
	}{Server: serverName, Version: serverVersion, Default: s.cfg.Default,
		Sessions: s.sessions.Len()}
	if a.Remote {
		out.Remote = map[string][]string{}
	}
	for _, d := range s.cfg.Describe() {
		if a.Provider != "" && !strings.EqualFold(a.Provider, d.Name) {
			continue
		}
		out.Providers = append(out.Providers, d)
		if !a.Remote {
			continue
		}
		p := s.cfg.Provider(d.Name)
		ids, err := s.client.ListRemote(ctx, p)
		if err != nil {
			out.Remote[d.Name] = []string{"error: " + oneLine(err.Error())}
			continue
		}
		sort.Strings(ids)
		out.Remote[d.Name] = ids
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Server) toolResetSession() Tool {
	return Tool{
		Name:        "reset_session",
		Description: "Forget a stored conversation id created by `ask`/`chat` with `session`.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session": map[string]any{"type": "string", "description": "The conversation id."},
			},
			"required": []string{"session"},
		},
		Handler: func(_ context.Context, raw json.RawMessage, _ progressReporter) (string, error) {
			var a struct {
				Session string `json:"session"`
			}
			if err := decodeArgs(raw, &a); err != nil {
				return "", err
			}
			if strings.TrimSpace(a.Session) == "" {
				return "", errors.New("session is required")
			}
			if s.sessions.Reset(a.Session) {
				return "session " + a.Session + " dropped", nil
			}
			return "session " + a.Session + " did not exist", nil
		},
	}
}

// progressEvery is how often a waiting call reports progress. Clients reset their
// per-request timeout on every notification, so it must stay well below the
// smallest timeout in use (pi defaults to 60 s).
var progressEvery = 5 * time.Second

// complete resolves the target model and performs the request, streaming the
// answer and keeping the client's timeout alive with progress notifications.
func (s *Server) complete(ctx context.Context, modelSpec string, msgs []Message, temp *float64, maxTokens int, rep progressReporter) (ChatResult, error) {
	target, err := s.cfg.Resolve(modelSpec)
	if err != nil {
		return ChatResult{}, err
	}
	if target.Provider.Key() == "" {
		return ChatResult{}, fmt.Errorf(
			"provider %q has no API key: set api_key / api_key_env in the config or export LLM_MCP_API_KEY",
			target.Provider.Name)
	}
	start := time.Now()
	s.log.Printf("→ %s/%s (%d messages)", target.Provider.Name, target.Model, len(msgs))

	var tokens, chars atomic.Int64
	stop := make(chan struct{})
	defer close(stop)
	if rep.enabled() {
		go func() {
			tick := time.NewTicker(progressEvery)
			defer tick.Stop()
			for {
				select {
				case <-stop:
					return
				case <-ctx.Done():
					return
				case <-tick.C:
					secs := time.Since(start).Seconds()
					msg := fmt.Sprintf("等待 %s 响应（%d 秒）", target.Provider.Name, int(secs))
					if n := tokens.Load(); n > 0 {
						msg = fmt.Sprintf("生成中：%d tokens（%d 秒）", n, int(secs))
					} else if c := chars.Load(); c > 0 {
						msg = fmt.Sprintf("生成中：%d 字（%d 秒）", c, int(secs))
					}
					rep.report(secs, 0, msg)
				}
			}
		}()
	}

	res, err := s.client.ChatStream(ctx, target, ChatOptions{Messages: msgs, Temperature: temp, MaxTokens: maxTokens},
		func(d Delta) {
			if d.Tokens > 0 {
				tokens.Store(int64(d.Tokens))
			}
			chars.Store(int64(len([]rune(d.Text))))
		})
	if err != nil {
		return ChatResult{}, err
	}
	if res.Text == "" {
		return ChatResult{}, fmt.Errorf("provider %q returned an empty answer (finish_reason=%q)",
			target.Provider.Name, res.FinishReason)
	}
	elapsed := time.Since(start)
	s.log.Printf("← %s/%s %d→%d tokens in %s (streamed=%v)", target.Provider.Name, res.Model,
		res.PromptTokens, res.OutputTokens, elapsed.Round(time.Millisecond), res.Streamed)
	rep.report(elapsed.Seconds(), 0, fmt.Sprintf("完成：%d 字，用时 %d 秒", len([]rune(res.Text)), int(elapsed.Seconds())))
	return res, nil
}

// renderResult is the text an agent receives: the answer, a truncation note and
// an optional footer.
func renderResult(res ChatResult, verbose bool) string {
	text := res.Text
	if res.FinishReason == "length" {
		text += "\n\n[答案被截断：finish_reason=length，可提高 max_tokens 或分段提问]"
	}
	if verbose {
		text += fmt.Sprintf("\n\n[model=%s tokens=%d/%d elapsed=%s finish=%s]",
			res.Model, res.PromptTokens, res.OutputTokens,
			res.Elapsed.Round(time.Millisecond), res.FinishReason)
	}
	return text
}

// withSystem puts a system message first when the transcript has none yet.
func withSystem(msgs []Message, system string) []Message {
	out := make([]Message, 0, len(msgs)+1)
	out = append(out, Message{Role: "system", Content: system})
	return append(out, msgs...)
}
