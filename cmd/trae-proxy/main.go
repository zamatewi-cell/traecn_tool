package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
	"github.com/zamatewi-cell/traecn_tool/internal/db"
	"github.com/zamatewi-cell/traecn_tool/internal/openai"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
	"github.com/zamatewi-cell/traecn_tool/internal/version"
)

// isLoopbackHost checks if host is a loopback address (127.0.0.0/8, ::1, or localhost)
func isLoopbackHost(host string) bool {
	if host == "" {
		return false // empty host (e.g. ":9090") means all interfaces
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false // unknown hostname or invalid IP, treat as non-loopback
	}
	return ip.IsLoopback()
}

// sanitizeListenAddr determines the final listen address and whether it exposes the server
// to external/LAN networks (i.e. non-loopback).
func sanitizeListenAddr(addr string, allowLan bool, logger *slog.Logger) (string, bool) {
	if addr == "" {
		if allowLan {
			return "0.0.0.0:9090", true
		}
		return "127.0.0.1:9090", false
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasPrefix(addr, ":") {
			host = ""
			port = strings.TrimPrefix(addr, ":")
		} else {
			host = addr
			port = "9090"
		}
	}

	// Empty host (e.g. ":9090") binds to all network interfaces on the machine (INADDR_ANY / IPv6 wildcard)
	if host == "" {
		return net.JoinHostPort("0.0.0.0", port), true
	}

	isLoopback := isLoopbackHost(host)

	// If allow_lan is explicitly enabled but user specified a loopback host, expand to 0.0.0.0
	if allowLan && isLoopback {
		return net.JoinHostPort("0.0.0.0", port), true
	}

	// If host is not loopback (e.g. 192.168.x.x, 10.x.x.x, 0.0.0.0, [::]), it is definitively a LAN exposure.
	if !isLoopback {
		return net.JoinHostPort(host, port), true
	}

	// Pure loopback address (127.0.0.1, localhost, etc.)
	return net.JoinHostPort(host, port), false
}

func parseLogLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func main() {
	configPath := flag.String("config", "config.json", "config file path or 'stdin'/'-' to read from stdin")
	listen := flag.String("listen", "", "listen address (default: 127.0.0.1:9090, or 0.0.0.0:9090 if allow-lan)")
	allowLan := flag.Bool("allow-lan", false, "allow access from local area network (bind to 0.0.0.0)")
	insecureNoAuth := flag.Bool("insecure-no-auth", false, "allow LAN exposure without API Key authentication (INSECURE)")
	logLevel := flag.String("log-level", "info", "log level (debug/info/warn/error)")
	apiKey := flag.String("api-key", "", "API key for authentication (optional, comma-separated for multiple keys)")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		ver := strings.TrimPrefix(version.Version, "v")
		fmt.Printf("trae-proxy v%s\n", ver)
		os.Exit(0)
	}

	cliLogLevelSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "log-level" {
			cliLogLevelSet = true
		}
		if f.Name == "api-key" {
			trimmed := strings.TrimSpace(*apiKey)
			if trimmed == "" {
				fmt.Fprintln(os.Stderr, "FATAL: -api-key flag was explicitly provided but contains no valid key! Refusing to start in insecure fallback mode.")
				os.Exit(1)
			}
		}
	})

	// Setup dynamic logger
	logLevelVar := &slog.LevelVar{}
	logLevelVar.Set(parseLogLevel(*logLevel))
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevelVar}))

	// Load config (supports file, stdin, or default fallback)
	var cfg *config.Config
	var err error
	if *configPath == "-" || *configPath == "stdin" {
		inputBytes, errRead := io.ReadAll(os.Stdin)
		if errRead != nil {
			logger.Error("failed to read config from stdin", "error", errRead)
			os.Exit(1)
		}
		cfg, err = config.ParseConfig(inputBytes)
		if err != nil {
			logger.Error("failed to parse stdin config", "error", err)
			os.Exit(1)
		}
		logger.Info("loaded config from stdin (pipeline mode, zero disk footprint)")
	} else if _, errStat := os.Stat(*configPath); errStat == nil {
		cfg, err = config.LoadConfig(*configPath)
		if err != nil {
			logger.Error("failed to load config", "error", err)
			os.Exit(1)
		}
		logger.Info("loaded config", "path", *configPath)
	} else {
		cfg = config.DefaultConfig()
		logger.Info("using default config (auto-detecting Trae CN token)")
	}

	// Apply configuration log_level if not explicitly overridden by CLI
	if !cliLogLevelSet && cfg.LogLevel != "" {
		logLevelVar.Set(parseLogLevel(cfg.LogLevel))
		logger.Debug("applied log level from configuration", "level", cfg.LogLevel)
	}

	// Determine allow-lan setting
	if *allowLan {
		cfg.AllowLan = true
	}

	// Apply CLI listen flag override if provided
	if *listen != "" {
		cfg.ListenAddr = *listen
	}

	// Apply CLI apiKey flag override if provided
	if *apiKey != "" {
		for _, k := range strings.Split(*apiKey, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				cfg.APIKeys = append(cfg.APIKeys, k)
			}
		}
	}

	// Clean and filter empty api_keys
	var cleanKeys []string
	for _, k := range cfg.APIKeys {
		if tk := strings.TrimSpace(k); tk != "" {
			cleanKeys = append(cleanKeys, tk)
		}
	}
	cfg.APIKeys = cleanKeys

	// Enforce IP-level loopback security convergence
	resolvedAddr, isLAN := sanitizeListenAddr(cfg.ListenAddr, cfg.AllowLan, logger)
	cfg.ListenAddr = resolvedAddr

	// Safety check: prevent network exposure without API Key authentication
	if isLAN && len(cfg.APIKeys) == 0 && !cfg.InsecureNoAuth && !*insecureNoAuth {
		logger.Error("FATAL: server is configured to bind to a non-loopback address ("+resolvedAddr+") but no api_keys are configured! Binding to network interfaces without authentication is insecure and rejected.")
		logger.Error("To fix: configure 'api_keys' in configuration, or explicitly pass -insecure-no-auth if you really want to expose without auth.")
		os.Exit(1)
	}

	// Structured event callback for token refresh
	onTokenRefreshed := func(accountName string, tok *auth.TokenInfo) {
		payload := map[string]interface{}{
			"event":              "token_refreshed",
			"account":            accountName,
			"user_id":            tok.UserID,
			"token":              tok.AccessToken,
			"refresh_token":      tok.RefreshToken,
			"expires_at":         tok.ExpiresAt.Format(time.RFC3339),
			"refresh_expires_at": tok.RefreshExpiresAt.Format(time.RFC3339),
			"timestamp":          time.Now().Unix(),
		}
		raw, _ := json.Marshal(payload)
		fmt.Printf("__TRAE_EVENT__:%s\n", string(raw))
	}

	// Initialize credential pool with proactive refresh + circuit breaking
	tp := auth.NewPool(&auth.PoolOptions{
		Refresher:        auth.NewTokenRefresher(config.AgentDomain, nil),
		Logger:           logger,
		OnTokenRefreshed: onTokenRefreshed,
	})

	hasExplicitEmptyAccounts := (cfg.Accounts != nil && len(cfg.Accounts) == 0)
	autoDiscoverDisabled := (cfg.AutoDiscover != nil && !*cfg.AutoDiscover)

	if len(cfg.Accounts) > 0 {
		for _, acc := range cfg.Accounts {
			switch {
			case acc.Token != "":
				var exp, refExp time.Time
				if acc.ExpiresAt != "" {
					exp, _ = time.Parse(time.RFC3339, acc.ExpiresAt)
				}
				if acc.RefreshExpiresAt != "" {
					refExp, _ = time.Parse(time.RFC3339, acc.RefreshExpiresAt)
				}
				tp.AddAccountWithCredentials(acc.Name, acc.Token, acc.RefreshToken, acc.UserID, exp, refExp)
				logger.Info("added account (credentials)", "name", acc.Name)
			case acc.EnvVar != "":
				if err := tp.AddAccountFromEnv(acc.Name, acc.EnvVar); err != nil {
					logger.Warn("failed to add account from env", "name", acc.Name, "env_var", acc.EnvVar, "error", err)
				} else {
					logger.Info("added account (env var)", "name", acc.Name, "env_var", acc.EnvVar)
				}
			default:
				storagePath := acc.StoragePath
				if storagePath == "" {
					storagePath = auth.DefaultStoragePath()
				}
				if err := tp.AddAccount(acc.Name, storagePath); err != nil {
					logger.Warn("failed to add account", "name", acc.Name, "error", err)
				} else {
					logger.Info("added account (storage)", "name", acc.Name)
				}
			}
		}
	} else if hasExplicitEmptyAccounts || autoDiscoverDisabled {
		logger.Error("FATAL: no valid accounts configured and auto-discovery is explicitly disabled. Refusing to sniff local accounts.")
		os.Exit(1)
	} else {
		logger.Info("no accounts explicitly configured, falling back to local Trae credential auto-discovery")
		sniffed := auth.SniffAccounts()
		for _, s := range sniffed {
			if err := tp.AddSource(s.Name, s.Source); err != nil {
				logger.Warn("failed to add sniffed credential", "name", s.Name, "error", err)
			} else {
				logger.Info("auto-detected credential", "name", s.Name, "source", s.Source.Type)
			}
		}
		if len(tp.GetAccounts()) == 0 {
			logger.Error("no Trae CN credential found",
				"sniffed_paths", auth.StorageCandidates(),
				"env_vars", auth.EnvTokenVars)
			logger.Info("create config.json to configure tokens manually (token / env_var / storage_path)")
			os.Exit(1)
		}
	}

	// Initialize SQLite persistence store
	sqliteStore, err := db.InitGlobalStore(filepath.Join("data", "trae_proxy.db"))
	if err != nil {
		logger.Warn("failed to initialize sqlite store, activity logging disabled", "error", err)
	} else {
		defer sqliteStore.Close()
		logger.Info("initialized SQLite persistence store", "path", "data/trae_proxy.db")
	}

	// Start proxy
	traeProxy := proxy.NewTraeProxy(tp, logger)
	traeProxy.SetProtection(cfg.Protect)
	if cfg.RequestTimeout > 0 {
		traeProxy.SetRequestTimeout(time.Duration(cfg.RequestTimeout) * time.Second)
		logger.Info("configured request timeout", "seconds", cfg.RequestTimeout)
	}

	var srvCfg *openai.ServerConfig
	if len(cfg.APIKeys) > 0 {
		srvCfg = &openai.ServerConfig{
			APIKeys: cfg.APIKeys,
		}
		logger.Info("API Key authentication enabled", "keys_count", len(cfg.APIKeys))
	} else {
		logger.Info("API Key authentication disabled (open access on local loopback)")
	}
	server := openai.NewServer(traeProxy, logger, srvCfg)

	// Best-effort dynamic model registry refresh (falls back to builtin list).
	go traeProxy.RefreshModelRegistry()

	logger.Info("====================================")
	logger.Info("  trae-proxy v" + strings.TrimPrefix(version.Version, "v"))
	logger.Info("  Trae CN -> OpenAI Compatible API")
	logger.Info("====================================")

	if err := server.ListenAndServe(cfg.ListenAddr); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
