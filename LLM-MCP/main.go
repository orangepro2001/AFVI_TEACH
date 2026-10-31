package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("llm-mcp", flag.ContinueOnError)
	var (
		configPath  = fs.String("config", "", "配置文件路径（默认 ./llm-mcp.json → 可执行文件旁 → ~/.config/llm-mcp/config.json；环境变量 "+ConfigEnv+"）")
		endpoint    = fs.String("endpoint", envOr("LLM_MCP_BASE_URL", ""), "API 端点/基址，如 https://api.deepseek.com/v1（覆盖配置里的同名 provider）")
		modelName   = fs.String("model", envOr("LLM_MCP_MODEL", ""), "模型名，如 deepseek-chat")
		providerNme = fs.String("name", envOr("LLM_MCP_NAME", "deepseek"), "-endpoint/-model 对应的 provider 名字")
		apiKeyFile  = fs.String("api-key-file", envOr("LLM_MCP_API_KEY_FILE", ""), "存放 API Key 的文件（推荐，权限 600）")
		apiKey      = fs.String("api-key", envOr("LLM_MCP_API_KEY", ""), "API Key（会留在 shell 历史里，优先用 -api-key-file）")
		httpAddr    = fs.String("http", envOr("LLM_MCP_HTTP", ""), "HTTP 模式监听地址，如 127.0.0.1:8765 或 8765；留空 = stdio")
		token       = fs.String("token", envOr("LLM_MCP_TOKEN", ""), "HTTP 模式的 Bearer 访问令牌（留空=不校验，只适合绑定回环地址）")
		logPath     = fs.String("log", envOr("LLM_MCP_LOG", ""), "日志文件（追加写入，权限 600）")
		verbose     = fs.Bool("verbose", false, "打印每次请求/应答")
		listOnly    = fs.Bool("list", false, "打印已配置的 provider 与模型后退出")
		dumpConfig  = fs.Bool("print-config", false, "打印生效配置（密钥打码）后退出")
		showVersion = fs.Bool("version", false, "打印版本后退出")
		timeout     = fs.String("timeout", envOr("LLM_MCP_TIMEOUT", ""), "单次请求超时，如 120s（默认 180s）")
		retries     = fs.Int("retries", 3, "单个请求最大尝试次数（429 / 5xx / 网络错误才重试）")
	)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s %s — 把 DeepSeek / 任意 OpenAI 兼容端点包装成 MCP 服务\n\n", serverName, serverVersion)
		fmt.Fprintf(os.Stderr, "用法:\n  llm-mcp                      # stdio 模式（给 MCP 客户端当子进程启动）\n")
		fmt.Fprintf(os.Stderr, "  llm-mcp -http 8765           # HTTP 模式（远程 Agent 连 http://127.0.0.1:8765/mcp）\n")
		fmt.Fprintf(os.Stderr, "  llm-mcp -endpoint URL -model NAME -api-key-file FILE\n\n参数:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Printf("%s %s\n", serverName, serverVersion)
		return 0
	}

	logger := log.New(os.Stderr, "["+serverName+"] ", log.LstdFlags)
	if *logPath != "" {
		f, err := os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open log %s: %v\n", *logPath, err)
			return 1
		}
		defer f.Close()
		logger = log.New(f, "", log.LstdFlags)
	}

	cfg, path, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "配置错误:", err)
		return 1
	}
	switch {
	case cfg == nil && *endpoint == "":
		fmt.Fprintln(os.Stderr, "没有找到配置文件，也没有给 -endpoint。最小用法：")
		fmt.Fprintln(os.Stderr, `  llm-mcp -endpoint https://api.deepseek.com/v1 -model deepseek-chat -api-key-file ~/.config/llm-mcp/api-key`)
		return 2
	case cfg == nil:
		cfg = &Config{Providers: []Provider{{Name: *providerNme}}}
		path = "(命令行参数)"
	}
	if cfg.Default == "" {
		cfg.Default = cfg.Providers[0].Name
	}
	if err := applyOverrides(cfg, *providerNme, *endpoint, *modelName, *apiKeyFile, *apiKey); err != nil {
		fmt.Fprintln(os.Stderr, "配置错误:", err)
		return 1
	}
	if *httpAddr != "" {
		cfg.HTTP = *httpAddr
	}
	if *token != "" {
		cfg.Token = *token
	}
	if *timeout != "" {
		cfg.Timeout = *timeout
	}
	if err := cfg.normalize(); err != nil {
		fmt.Fprintln(os.Stderr, "配置错误:", err)
		return 1
	}

	perRequest := 180 * time.Second
	if d, err := time.ParseDuration(cfg.Timeout); err == nil && d > 0 {
		perRequest = d
	} else if cfg.Timeout != "" {
		fmt.Fprintf(os.Stderr, "忽略非法 timeout %q\n", cfg.Timeout)
	}
	authToken, err := cfg.ResolveToken()
	if err != nil {
		fmt.Fprintln(os.Stderr, "配置错误:", err)
		return 1
	}

	logger.Printf("%s %s, 配置: %s, 默认: %s, 超时: %s", serverName, serverVersion, path, cfg.Default, perRequest)
	for _, w := range cfg.CheckSecrets() {
		logger.Printf("警告: %s", w)
	}
	if *verbose {
		for _, d := range cfg.Describe() {
			logger.Printf("模型 %s: %s（密钥来源: %s）", d.Name, d.DefaultModel, d.KeySource)
		}
	}
	if *listOnly {
		printConfigSummary(cfg)
		return 0
	}
	if *dumpConfig {
		fmt.Println(cfg.RedactedJSON())
		return 0
	}

	client := NewClient(perRequest, *retries)
	srv := NewServer(cfg, client, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.HTTP != "" {
		return report(RunHTTP(ctx, srv, cfg.HTTP, authToken, logger))
	}
	return report(RunStdio(ctx, srv, logger))
}

func report(err error) int {
	if err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, "退出:", err)
		return 1
	}
	return 0
}

// applyOverrides lets -endpoint/-model/-api-key-* patch the configuration
// without a file.
func applyOverrides(cfg *Config, name, endpoint, model, keyFile, key string) error {
	if endpoint == "" && model == "" && keyFile == "" && key == "" {
		return nil
	}
	p := cfg.Provider(name)
	if p == nil {
		cfg.Providers = append([]Provider{{Name: name}}, cfg.Providers...)
		p = &cfg.Providers[0]
	}
	if endpoint != "" {
		p.BaseURL = endpoint
	}
	if model != "" {
		p.Model = model
	}
	if keyFile != "" {
		p.APIKeyFile = keyFile
	}
	if key != "" {
		p.APIKey = key
	}
	if cfg.Default == "" || cfg.Provider(cfg.Default) == nil {
		cfg.Default = p.Name
	}
	if strings.TrimSpace(p.BaseURL) == "" {
		return fmt.Errorf("-endpoint 不能为空（或删掉 -endpoint 让配置文件的 base_url 生效）")
	}
	return nil
}

func printConfigSummary(cfg *Config) {
	fmt.Printf("%s %s — 默认: %s\n", serverName, serverVersion, cfg.Default)
	tok, err := cfg.ResolveToken()
	switch {
	case err != nil:
		fmt.Printf("HTTP 令牌: 读取失败（%v）\n", err)
	case tok == "":
		fmt.Printf("HTTP 令牌: 未启用（建议只绑 127.0.0.1）\n")
	default:
		fmt.Printf("HTTP 令牌: 已启用（%s）\n", tokenSource(cfg))
	}
	for _, d := range cfg.Describe() {
		mark := " "
		if d.Default {
			mark = "*"
		}
		key := "缺少密钥"
		if d.HasKey {
			key = d.KeySource
		}
		fmt.Printf("%s %-12s %-40s 模型=%s  密钥=%s\n", mark, d.Name, d.BaseURL, d.DefaultModel, key)
		for alias, id := range d.Models {
			fmt.Printf("    %-10s -> %s\n", alias, id)
		}
	}
}

func tokenSource(cfg *Config) string {
	if cfg.TokenFile != "" {
		return "file " + cfg.TokenFile
	}
	return "inline token"
}

func envOr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return def
}
