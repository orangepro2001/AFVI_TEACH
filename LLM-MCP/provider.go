package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Message is one chat message in OpenAI wire format.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatOptions are the per-call knobs of one completion.
type ChatOptions struct {
	Messages    []Message
	Temperature *float64
	MaxTokens   int
}

// Delta is one incremental update of a streamed completion. The Text and
// Reasoning fields are cumulative, Tokens is what the provider reported so far
// (0 when it reports nothing).
type Delta struct {
	Text      string
	Reasoning string
	Tokens    int
	Finish    string
}

// ChatResult is what came back from the endpoint.
type ChatResult struct {
	Text         string
	Reasoning    string
	FinishReason string
	PromptTokens int
	OutputTokens int
	Model        string
	Elapsed      time.Duration
	Streamed     bool
}

// APIError is a non-2xx answer or a transport failure of one request.
type APIError struct {
	Provider string
	Status   int
	Code     string
	Message  string
	Body     string
}

func (e *APIError) Error() string {
	if e.Status == 0 {
		return fmt.Sprintf("%s: %s", e.Provider, e.Message)
	}
	return fmt.Sprintf("%s: HTTP %d: %s", e.Provider, e.Status, e.Message)
}

// Hint turns a failed request into something an agent can act on.
func (e *APIError) Hint() string {
	switch e.Status {
	case 0:
		return "endpoint unreachable — check base_url/network (base_url must be the API root, e.g. https://api.deepseek.com/v1)"
	case 401, 403:
		return "authentication failed — set api_key / api_key_env (or LLM_MCP_API_KEY) for this provider"
	case 402:
		return "insufficient balance on the provider account"
	case 404:
		return "endpoint or model not found — check base_url (…/v1) and the model name"
	case 429:
		return "rate limited — retry later or lower concurrency"
	}
	if e.Status >= 500 {
		return "provider side error — retry later"
	}
	return ""
}

// chatURL is the /chat/completions URL of the provider.
func (p *Provider) chatURL() string {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	return base + "/chat/completions"
}

// modelsURL is the /models URL of the provider.
func (p *Provider) modelsURL() string {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	base = strings.TrimSuffix(base, "/chat/completions")
	return base + "/models"
}

// buildBody renders the request body. Streaming asks for usage in the last chunk
// so the answer can still report token counts.
func buildBody(p *Provider, model string, opts ChatOptions, stream bool) ([]byte, error) {
	body := map[string]any{
		"model":    model,
		"messages": opts.Messages,
		"stream":   stream,
	}
	if stream {
		body["stream_options"] = map[string]any{"include_usage": true}
	}
	temp := opts.Temperature
	if temp == nil {
		temp = p.Temperature
	}
	if temp != nil {
		body["temperature"] = *temp
	}
	if opts.MaxTokens > 0 {
		body["max_tokens"] = opts.MaxTokens
	} else if p.MaxTokens > 0 {
		body["max_tokens"] = p.MaxTokens
	}
	for k, v := range p.Params {
		if _, exists := body[k]; !exists {
			body[k] = v
		}
	}
	return json.Marshal(body)
}

// Chat sends one completion and waits for the whole answer. Used for small calls
// and as the fallback when an endpoint does not stream.
func (c *Client) Chat(ctx context.Context, t Target, opts ChatOptions) (ChatResult, error) {
	p := t.Provider
	if p == nil {
		return ChatResult{}, errors.New("no provider resolved")
	}
	model := firstNonEmpty(t.Model, p.DefaultModel())
	payload, err := buildBody(p, model, opts, false)
	if err != nil {
		return ChatResult{}, err
	}
	res, err := c.send(ctx, t, payload, func(resp *http.Response) (ChatResult, error) {
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		if err != nil {
			return ChatResult{}, &APIError{Provider: p.Name, Status: resp.StatusCode, Message: err.Error()}
		}
		return parseCompletion(p.Name, raw, model)
	})
	if err != nil {
		return ChatResult{}, err
	}
	res.Streamed = false
	return res, nil
}

// ChatStream sends one completion with stream:true and reports progress through
// onDelta while the answer arrives. An endpoint that ignores stream and answers
// with plain JSON is handled transparently.
func (c *Client) ChatStream(ctx context.Context, t Target, opts ChatOptions, onDelta func(Delta)) (ChatResult, error) {
	p := t.Provider
	if p == nil {
		return ChatResult{}, errors.New("no provider resolved")
	}
	model := firstNonEmpty(t.Model, p.DefaultModel())
	payload, err := buildBody(p, model, opts, true)
	if err != nil {
		return ChatResult{}, err
	}
	return c.send(ctx, t, payload, func(resp *http.Response) (ChatResult, error) {
		ctype := strings.ToLower(resp.Header.Get("Content-Type"))
		if !strings.Contains(ctype, "event-stream") {
			// Provider ignored `stream` (or answered with an error object).
			raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
			if err != nil {
				return ChatResult{}, &APIError{Provider: p.Name, Status: resp.StatusCode, Message: err.Error()}
			}
			res, err := parseCompletion(p.Name, raw, model)
			if err != nil {
				return ChatResult{}, err
			}
			res.Streamed = false
			if onDelta != nil {
				onDelta(Delta{Text: res.Text, Reasoning: res.Reasoning, Tokens: res.OutputTokens, Finish: res.FinishReason})
			}
			return res, nil
		}
		return readSSE(p.Name, resp.Body, model, onDelta)
	})
}

// send POSTs the payload, retries retryable failures and applies the timeouts.
func (c *Client) send(ctx context.Context, t Target, payload []byte, parse func(*http.Response) (ChatResult, error)) (ChatResult, error) {
	p := t.Provider
	timeout := p.TimeoutDuration()
	if timeout <= 0 {
		timeout = c.timeout
	}
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	start := time.Now()
	var lastErr error
	for attempt := 0; attempt < c.retries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(400<<uint(attempt-1)) * time.Millisecond
			select {
			case <-ctx.Done():
				return ChatResult{}, ctx.Err()
			case <-time.After(delay):
			}
		}
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		resp, err := c.post(reqCtx, cancel, p, payload)
		if err == nil {
			res, perr := parse(resp)
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			cancel()
			if perr == nil {
				res.Elapsed = time.Since(start)
				return res, nil
			}
			err = perr
		} else {
			cancel()
		}
		lastErr = err
		if !retryable(err) {
			return ChatResult{}, err
		}
		if ctx.Err() != nil {
			return ChatResult{}, ctx.Err()
		}
	}
	return ChatResult{}, lastErr
}

func (c *Client) post(reqCtx context.Context, cancel context.CancelFunc, p *Provider, payload []byte) (*http.Response, error) {
	_ = cancel // kept for symmetry; the caller owns cancellation
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, p.chatURL(), bytes.NewReader(payload))
	if err != nil {
		return nil, &APIError{Provider: p.Name, Status: 0, Message: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")
	if key := p.Key(); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if reqCtx.Err() != nil && errors.Is(reqCtx.Err(), context.Canceled) {
			return nil, context.Canceled
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(reqCtx.Err(), context.DeadlineExceeded) {
			return nil, &APIError{Provider: p.Name, Status: 0,
				Message: "request timed out (raise \"timeout\" on the provider or pass a smaller max_tokens)"}
		}
		return nil, &APIError{Provider: p.Name, Status: 0, Message: err.Error()}
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		return nil, httpError(p.Name, resp.StatusCode, raw)
	}
	return resp, nil
}

func httpError(provider string, status int, raw []byte) *APIError {
	ae := &APIError{Provider: provider, Status: status, Body: truncate(string(raw), 600)}
	var parsed struct {
		Error struct {
			Message string          `json:"message"`
			Type    string          `json:"type"`
			Code    json.RawMessage `json:"code"`
		} `json:"error"`
		Message string `json:"message"`
		Detail  string `json:"detail"`
	}
	if json.Unmarshal(raw, &parsed) == nil {
		ae.Message = firstNonEmpty(parsed.Error.Message, parsed.Message, parsed.Detail, strings.TrimSpace(string(raw)))
		ae.Code = strings.Trim(parsed.Error.Type+" "+string(parsed.Error.Code), ` "'`)
	} else {
		ae.Message = strings.TrimSpace(string(raw))
	}
	if ae.Message == "" {
		ae.Message = http.StatusText(status)
	}
	return ae
}

func retryable(err error) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.Status == 0 || ae.Status == 429 || ae.Status >= 500
	}
	return false
}

// parseCompletion reads a whole (non-streamed) chat completion body.
func parseCompletion(provider string, raw []byte, model string) (ChatResult, error) {
	var out struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
				Refusal          string `json:"refusal"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage usage `json:"usage"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return ChatResult{}, &APIError{Provider: provider, Status: 200,
			Message: "response is not a JSON chat completion: " + truncate(err.Error()+" "+string(raw), 300)}
	}
	if len(out.Choices) == 0 {
		return ChatResult{}, &APIError{Provider: provider, Status: 200,
			Message: "response has no choices: " + truncate(string(raw), 300)}
	}
	ch := out.Choices[0]
	text := strings.TrimSpace(ch.Message.Content)
	if text == "" && ch.Message.Refusal != "" {
		text = strings.TrimSpace(ch.Message.Refusal)
	}
	if text == "" {
		text = strings.TrimSpace(ch.Message.ReasoningContent)
	}
	return ChatResult{
		Text:         text,
		Reasoning:    strings.TrimSpace(ch.Message.ReasoningContent),
		FinishReason: ch.FinishReason,
		PromptTokens: out.Usage.PromptTokens,
		OutputTokens: out.Usage.CompletionTokens,
		Model:        firstNonEmpty(out.Model, model),
	}, nil
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// streamChunk is one SSE frame of an OpenAI-compatible stream.
type streamChunk struct {
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			Refusal          string `json:"refusal"`
		} `json:"delta"`
		Message struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage usage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// readSSE consumes an event stream, accumulating the answer and reporting each
// update.
func readSSE(provider string, body io.Reader, model string, onDelta func(Delta)) (ChatResult, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64<<10), 8<<20)

	res := ChatResult{Model: model, Streamed: true}
	var text, reasoning strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") || strings.HasPrefix(line, "event:") ||
			strings.HasPrefix(line, "id:") || strings.HasPrefix(line, "retry:") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			// A malformed frame must not kill a working stream.
			continue
		}
		if chunk.Error != nil {
			return ChatResult{}, &APIError{Provider: provider, Status: 200,
				Message: firstNonEmpty(chunk.Error.Message, "stream error"), Code: chunk.Error.Type}
		}
		if chunk.Model != "" {
			res.Model = chunk.Model
		}
		if chunk.Usage.CompletionTokens > 0 || chunk.Usage.PromptTokens > 0 {
			res.PromptTokens = chunk.Usage.PromptTokens
			res.OutputTokens = chunk.Usage.CompletionTokens
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		ch := chunk.Choices[0]
		piece := ch.Delta.Content
		think := ch.Delta.ReasoningContent
		if piece == "" && think == "" && ch.Message.Content != "" {
			// Some proxies stream the final message instead of deltas.
			piece = ch.Message.Content
			think = ch.Message.ReasoningContent
		}
		if think != "" {
			reasoning.WriteString(think)
		}
		if piece != "" {
			text.WriteString(piece)
		}
		if ch.FinishReason != "" {
			res.FinishReason = ch.FinishReason
		}
		if onDelta != nil && (piece != "" || think != "") {
			onDelta(Delta{Text: text.String(), Reasoning: reasoning.String(),
				Tokens: res.OutputTokens, Finish: res.FinishReason})
		}
	}
	if err := scanner.Err(); err != nil {
		if res.Text == "" && text.Len() == 0 {
			return ChatResult{}, &APIError{Provider: provider, Status: 0, Message: "stream interrupted: " + err.Error()}
		}
	}
	out := strings.TrimSpace(text.String())
	if out == "" {
		out = strings.TrimSpace(reasoning.String())
	}
	res.Text = out
	res.Reasoning = strings.TrimSpace(reasoning.String())
	if onDelta != nil {
		onDelta(Delta{Text: res.Text, Reasoning: res.Reasoning, Tokens: res.OutputTokens, Finish: res.FinishReason})
	}
	return res, nil
}

// ListRemote asks the provider's /models endpoint for model ids.
func (c *Client) ListRemote(ctx context.Context, p *Provider) ([]string, error) {
	timeout := p.TimeoutDuration()
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, p.modelsURL(), nil)
	if err != nil {
		return nil, err
	}
	if key := p.Key(); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &APIError{Provider: p.Name, Status: 0, Message: err.Error()}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, httpError(p.Name, resp.StatusCode, raw)
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, &APIError{Provider: p.Name, Status: resp.StatusCode, Message: "unexpected /models answer"}
	}
	ids := make([]string, 0, len(out.Data))
	for _, d := range out.Data {
		if d.ID != "" {
			ids = append(ids, d.ID)
		}
	}
	return ids, nil
}

// Client owns the HTTP connection pool and the retry budget.
type Client struct {
	http    *http.Client
	timeout time.Duration
	retries int
}

// NewClient builds the shared client. retries counts attempts (>= 1).
func NewClient(timeout time.Duration, retries int) *Client {
	if retries < 1 {
		retries = 1
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 8
	return &Client{
		http:    &http.Client{Transport: transport},
		timeout: timeout,
		retries: retries,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
