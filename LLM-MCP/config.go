package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Config is the whole llm-mcp configuration file (JSON, optional; every field can
// also be supplied by flags / environment variables).
//
//	{
//	  "default": "deepseek",
//	  "providers": [
//	    {
//	      "name": "deepseek",
//	      "base_url": "https://api.deepseek.com/v1",
//	      "api_key_env": "DEEPSEEK_API_KEY",
//	      "model": "deepseek-chat",
//	      "models": { "reasoner": "deepseek-reasoner" },
//	      "temperature": 0.3,
//	      "max_tokens": 4096,
//	      "timeout": "180s"
//	    }
//	  ]
//	}
type Config struct {
	// Default names the provider (or provider/model) used when a tool call does
	// not ask for one.
	Default   string     `json:"default,omitempty"`
	Providers []Provider `json:"providers"`

	// Server options; the matching command line flags win over these.
	HTTP  string `json:"http,omitempty"`
	Token string `json:"token,omitempty"`
	// TokenFile is a 0600 file holding the HTTP bearer token (preferred over Token).
	TokenFile string `json:"token_file,omitempty"`
	Timeout   string `json:"timeout,omitempty"`
}

// Provider is one OpenAI-compatible endpoint (DeepSeek, Moonshot, vLLM, Ollama,
// OpenRouter, LM Studio, ...).
type Provider struct {
	Name string `json:"name"`
	// BaseURL is the API root ("https://api.deepseek.com/v1") or a full
	// ".../chat/completions" URL. /chat/completions is appended when missing.
	BaseURL string `json:"base_url"`
	// The key comes from the first of: APIKey (inline, avoid), APIKeyFile (a
	// 0600 file, recommended), $APIKeyEnv, $LLM_MCP_API_KEY. All may use ${VAR}
	// and ~ expansion; APIKeyFile is read on every request so rotation needs no
	// restart.
	APIKey     string            `json:"api_key,omitempty"`
	APIKeyFile string            `json:"api_key_file,omitempty"`
	APIKeyEnv  string            `json:"api_key_env,omitempty"`
	Model      string            `json:"model,omitempty"`
	Models     map[string]string `json:"models,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	// Sampling defaults applied when a tool call does not set them.
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Timeout     string   `json:"timeout,omitempty"`
	// Params are merged into the request body verbatim (e.g.
	// {"top_p":0.9,"response_format":{"type":"json_object"}}).
	Params map[string]any `json:"params,omitempty"`
	// Cost is a free-form tier shown to clients ("low", "high", ...).
	Cost string `json:"cost,omitempty"`
	// Note is one line of guidance for the calling agent, shown in the server
	// instructions and in list_models. Use it to say what this target is for.
	Note string `json:"note,omitempty"`
}

// Target is a resolved (provider, model) pair.
type Target struct {
	Provider *Provider
	Model    string
}

// ConfigEnv is the environment variable naming the config file.
const ConfigEnv = "LLM_MCP_CONFIG"

// LoadConfig reads the configuration. Search order: explicit path, $LLM_MCP_CONFIG,
// ./llm-mcp.json, next to the executable, ~/.config/llm-mcp/config.json.
// It returns (nil, "", nil) when no file exists so the caller can fall back to
// the -endpoint/-api-key/-model flags.
func LoadConfig(explicit string) (*Config, string, error) {
	var candidates []string
	switch {
	case explicit != "":
		candidates = []string{explicit}
	case os.Getenv(ConfigEnv) != "":
		candidates = []string{os.Getenv(ConfigEnv)}
	default:
		candidates = append(candidates, "llm-mcp.json")
		if exe, err := os.Executable(); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(exe), "llm-mcp.json"))
		}
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(home, ".config", "llm-mcp", "config.json"))
		}
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, "", fmt.Errorf("read config %s: %w", path, err)
		}
		cfg, err := ParseConfig(data)
		if err != nil {
			return nil, path, fmt.Errorf("%s: %w", path, err)
		}
		return cfg, path, nil
	}
	if explicit != "" || os.Getenv(ConfigEnv) != "" {
		return nil, "", fmt.Errorf("config file not found: %s", candidates[0])
	}
	return nil, "", nil
}

// ParseConfig decodes and validates a configuration document. ${VAR} references
// are expanded first, so keys may stay out of the file.
func ParseConfig(data []byte) (*Config, error) {
	cfg := &Config{}
	if err := json.Unmarshal([]byte(os.ExpandEnv(string(data))), cfg); err != nil {
		return nil, err
	}
	if err := cfg.normalize(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// normalize fills defaults and validates every provider.
func (c *Config) normalize() error {
	c.Token = os.ExpandEnv(strings.TrimSpace(c.Token))
	c.TokenFile = expandPath(c.TokenFile)
	if len(c.Providers) == 0 {
		return errors.New("no providers configured")
	}
	seen := map[string]bool{}
	for i := range c.Providers {
		p := &c.Providers[i]
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			p.Name = fmt.Sprintf("provider%d", i+1)
		}
		if seen[p.Name] {
			return fmt.Errorf("duplicate provider name %q", p.Name)
		}
		seen[p.Name] = true
		p.BaseURL = os.ExpandEnv(strings.TrimSpace(p.BaseURL))
		if p.BaseURL == "" {
			return fmt.Errorf("provider %q: base_url is required", p.Name)
		}
		p.Model = strings.TrimSpace(os.ExpandEnv(p.Model))
		p.APIKey = os.ExpandEnv(p.APIKey)
		p.APIKeyFile = expandPath(p.APIKeyFile)
		p.Headers = expandMap(p.Headers)
		if len(p.Models) > 0 {
			expanded := make(map[string]string, len(p.Models))
			for k, v := range p.Models {
				expanded[k] = os.ExpandEnv(strings.TrimSpace(v))
			}
			p.Models = expanded
		}
		if p.Timeout != "" {
			if _, err := time.ParseDuration(p.Timeout); err != nil {
				return fmt.Errorf("provider %q: timeout %q: %w", p.Name, p.Timeout, err)
			}
		}
	}
	return nil
}

// expandPath expands ${VAR} and a leading ~ in a file path.
func expandPath(path string) string {
	path = strings.TrimSpace(os.ExpandEnv(path))
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

func expandMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[os.ExpandEnv(k)] = os.ExpandEnv(v)
	}
	return out
}

// Provider finds a provider by name (case-insensitive).
func (c *Config) Provider(name string) *Provider {
	name = strings.TrimSpace(name)
	for i := range c.Providers {
		if strings.EqualFold(c.Providers[i].Name, name) {
			return &c.Providers[i]
		}
	}
	return nil
}

// DefaultProvider is the provider named by "default" (a provider name, a
// "provider/model" spec, an alias or a model id), else the first one.
func (c *Config) DefaultProvider() *Provider {
	if spec := strings.TrimSpace(c.Default); spec != "" {
		name := spec
		if i := strings.Index(spec, "/"); i > 0 {
			name = spec[:i]
		}
		if p := c.Provider(name); p != nil {
			return p
		}
		for i := range c.Providers {
			p := &c.Providers[i]
			if p.Model == spec {
				return p
			}
			for _, id := range p.Models {
				if id == spec {
					return p
				}
			}
		}
	}
	if len(c.Providers) > 0 {
		return &c.Providers[0]
	}
	return nil
}

// DefaultModel is the provider's model, else its first alias target.
func (p *Provider) DefaultModel() string {
	if p.Model != "" {
		return p.Model
	}
	keys := make([]string, 0, len(p.Models))
	for k := range p.Models {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if p.Models[k] != "" {
			return p.Models[k]
		}
	}
	return ""
}

// ModelFor resolves an alias ("reasoner") to a model id, else returns it verbatim.
func (p *Provider) ModelFor(name string) string {
	if v, ok := p.Models[name]; ok && v != "" {
		return v
	}
	return name
}

// Key is the API key to send: inline value, then api_key_file, then
// $APIKeyEnv, then $LLM_MCP_API_KEY.
func (p *Provider) Key() string {
	key, _ := p.KeyWithError()
	return key
}

// KeyWithError is Key plus why a file could not be read (startup diagnostics).
func (p *Provider) KeyWithError() (string, string) {
	if p.APIKey != "" {
		return p.APIKey, ""
	}
	if p.APIKeyFile != "" {
		data, err := os.ReadFile(p.APIKeyFile)
		if err != nil {
			return "", fmt.Sprintf("provider %q: api_key_file %s: %v", p.Name, p.APIKeyFile, err)
		}
		if key := strings.TrimSpace(string(data)); key != "" {
			return key, ""
		}
		return "", fmt.Sprintf("provider %q: api_key_file %s is empty", p.Name, p.APIKeyFile)
	}
	if p.APIKeyEnv != "" {
		if key := os.Getenv(p.APIKeyEnv); key != "" {
			return key, ""
		}
		return "", fmt.Sprintf("provider %q: environment variable %s is empty", p.Name, p.APIKeyEnv)
	}
	return os.Getenv("LLM_MCP_API_KEY"), ""
}

// KeySource names where the key comes from, without revealing it.
func (p *Provider) KeySource() string {
	switch {
	case p.APIKey != "":
		return "inline api_key"
	case p.APIKeyFile != "":
		return "file " + p.APIKeyFile
	case p.APIKeyEnv != "":
		return "env " + p.APIKeyEnv
	}
	if os.Getenv("LLM_MCP_API_KEY") != "" {
		return "env LLM_MCP_API_KEY"
	}
	return "none"
}

// ResolveToken is the HTTP bearer token: inline value, else token_file.
func (c *Config) ResolveToken() (string, error) {
	if c.Token != "" {
		return c.Token, nil
	}
	if c.TokenFile == "" {
		return "", nil
	}
	data, err := os.ReadFile(c.TokenFile)
	if err != nil {
		return "", fmt.Errorf("token_file %s: %w", c.TokenFile, err)
	}
	tok := strings.TrimSpace(string(data))
	if tok == "" {
		return "", fmt.Errorf("token_file %s is empty", c.TokenFile)
	}
	return tok, nil
}

// CheckSecrets reports the startup problems that would make a call fail, and
// warns about a key file other users can read.
func (c *Config) CheckSecrets() []string {
	var out []string
	for i := range c.Providers {
		p := &c.Providers[i]
		if _, problem := p.KeyWithError(); problem != "" {
			out = append(out, problem)
			continue
		}
		if p.Key() == "" {
			out = append(out, fmt.Sprintf("provider %q has no API key (api_key_file / api_key_env / LLM_MCP_API_KEY)", p.Name))
		}
		if p.APIKeyFile == "" {
			continue
		}
		if w := loosePerms("provider "+p.Name, "api_key_file", p.APIKeyFile); w != "" {
			out = append(out, w)
		}
	}
	if w := loosePerms("", "token_file", c.TokenFile); w != "" {
		out = append(out, w)
	}
	return out
}

// loosePerms warns when a secret file is readable by group or others.
func loosePerms(owner, label, path string) string {
	if path == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	mode := info.Mode().Perm()
	if mode&0o077 == 0 {
		return ""
	}
	who := strings.TrimSpace(owner + " ")
	return fmt.Sprintf("%s%s is mode %04o, tighten it: chmod 600 %s", who, label+" "+path, mode, path)
}

// TimeoutDuration is the per-request budget of this provider (0 = client default).
func (p *Provider) TimeoutDuration() time.Duration {
	if d, err := time.ParseDuration(p.Timeout); err == nil && d > 0 {
		return d
	}
	return 0
}

// Resolve turns a model spec into a provider + model id. Accepted forms:
//
//	""                    -> cfg.Default (a provider, "provider/model", an alias or an id)
//	"deepseek"            -> that provider's default model
//	"deepseek/reasoner"   -> provider deepseek, alias/model "reasoner"
//	"reasoner"            -> first provider whose models map has that alias
//	"deepseek-chat"       -> the provider that lists that model id, else the default provider
func (c *Config) Resolve(spec string) (Target, error) {
	spec = strings.TrimSpace(os.ExpandEnv(spec))
	if spec == "" {
		spec = strings.TrimSpace(c.Default)
	}
	if spec == "" {
		p := c.DefaultProvider()
		if p == nil {
			return Target{}, errors.New("no provider configured")
		}
		m := p.DefaultModel()
		if m == "" {
			return Target{}, fmt.Errorf("provider %q has no model configured", p.Name)
		}
		return Target{Provider: p, Model: m}, nil
	}
	if i := strings.Index(spec, "/"); i > 0 && i < len(spec)-1 {
		if p := c.Provider(spec[:i]); p != nil {
			return Target{Provider: p, Model: p.ModelFor(spec[i+1:])}, nil
		}
	}
	if p := c.Provider(spec); p != nil {
		m := p.DefaultModel()
		if m == "" {
			return Target{}, fmt.Errorf("provider %q has no model configured", p.Name)
		}
		return Target{Provider: p, Model: m}, nil
	}
	for i := range c.Providers {
		p := &c.Providers[i]
		if v, ok := p.Models[spec]; ok && v != "" {
			return Target{Provider: p, Model: v}, nil
		}
	}
	for i := range c.Providers {
		p := &c.Providers[i]
		if p.Model == spec {
			return Target{Provider: p, Model: spec}, nil
		}
	}
	p := c.DefaultProvider()
	if p == nil {
		return Target{}, errors.New("no provider configured")
	}
	// Unknown name: treat it as a literal model id on the default provider, which
	// is what "-model some-new-model" needs.
	return Target{Provider: p, Model: spec}, nil
}

// Describe lists the configured models for humans (and for list_models).
type Describe struct {
	Name         string            `json:"name"`
	BaseURL      string            `json:"base_url"`
	DefaultModel string            `json:"default_model"`
	Models       map[string]string `json:"models,omitempty"`
	HasKey       bool              `json:"has_key"`
	KeySource    string            `json:"key_source"`
	Cost         string            `json:"cost,omitempty"`
	Note         string            `json:"note,omitempty"`
	Params       map[string]any    `json:"params,omitempty"`
	Default      bool              `json:"default,omitempty"`
}

// Describe returns one entry per provider, default first.
func (c *Config) Describe() []Describe {
	def := c.DefaultProvider()
	out := make([]Describe, 0, len(c.Providers))
	for i := range c.Providers {
		p := &c.Providers[i]
		d := Describe{
			Name:         p.Name,
			BaseURL:      p.BaseURL,
			DefaultModel: p.DefaultModel(),
			Models:       p.Models,
			HasKey:       p.Key() != "",
			KeySource:    p.KeySource(),
			Cost:         p.Cost,
			Note:         p.Note,
			Params:       p.Params,
			Default:      p == def,
		}
		if d.Default {
			out = append([]Describe{d}, out...)
			continue
		}
		out = append(out, d)
	}
	return out
}

// RedactedJSON renders the effective configuration with keys masked.
func (c *Config) RedactedJSON() string {
	type providerOut struct {
		Name        string            `json:"name"`
		BaseURL     string            `json:"base_url"`
		Model       string            `json:"model,omitempty"`
		Models      map[string]string `json:"models,omitempty"`
		APIKey      string            `json:"api_key,omitempty"`
		APIKeyFile  string            `json:"api_key_file,omitempty"`
		APIKeyEnv   string            `json:"api_key_env,omitempty"`
		Temperature *float64          `json:"temperature,omitempty"`
		MaxTokens   int               `json:"max_tokens,omitempty"`
		Timeout     string            `json:"timeout,omitempty"`
		Params      map[string]any    `json:"params,omitempty"`
		Cost        string            `json:"cost,omitempty"`
		Note        string            `json:"note,omitempty"`
	}
	out := struct {
		Default   string        `json:"default"`
		HTTP      string        `json:"http,omitempty"`
		Token     string        `json:"token,omitempty"`
		TokenFile string        `json:"token_file,omitempty"`
		Timeout   string        `json:"timeout,omitempty"`
		Providers []providerOut `json:"providers"`
	}{Default: c.Default, HTTP: c.HTTP, Token: maskSecret(c.Token), TokenFile: c.TokenFile, Timeout: c.Timeout}
	for i := range c.Providers {
		p := &c.Providers[i]
		out.Providers = append(out.Providers, providerOut{
			Name: p.Name, BaseURL: p.BaseURL, Model: p.Model, Models: p.Models,
			APIKey: maskSecret(p.APIKey), APIKeyFile: p.APIKeyFile, APIKeyEnv: p.APIKeyEnv,
			Temperature: p.Temperature, MaxTokens: p.MaxTokens,
			Timeout: p.Timeout, Params: p.Params, Cost: p.Cost, Note: p.Note,
		})
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return string(b)
}

func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-2:]
}
