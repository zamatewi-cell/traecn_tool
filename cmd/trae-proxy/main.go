package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
	"github.com/zamatewi-cell/traecn_tool/internal/db"
	"github.com/zamatewi-cell/traecn_tool/internal/openai"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
)

var version = "1.0.0"

func main() {
	configPath := flag.String("config", "config.json", "config file path")
	listen := flag.String("listen", "", "listen address (default: 127.0.0.1:9090, or 0.0.0.0:9090 if allow-lan)")
	allowLan := flag.Bool("allow-lan", false, "allow access from local area network (bind to 0.0.0.0)")
	logLevel := flag.String("log-level", "info", "log level (debug/info/warn/error)")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("trae-proxy v%s\n", version)
		os.Exit(0)
	}

	// Setup logger
	var level slog.Level
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	// Load config
	var cfg *config.Config
	if _, err := os.Stat(*configPath); err == nil {
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

	// Determine allow-lan setting
	if *allowLan {
		cfg.AllowLan = true
	}

	// Determine listen address with security convergence
	if *listen != "" {
		cfg.ListenAddr = *listen
	}
	if cfg.ListenAddr == "" {
		if cfg.AllowLan {
			cfg.ListenAddr = "0.0.0.0:9090"
		} else {
			cfg.ListenAddr = "127.0.0.1:9090"
		}
	} else {
		// Convergence check: if not allow-lan but listen is ':port' or '0.0.0.0:port', constrain to 127.0.0.1
		if !cfg.AllowLan {
			if len(cfg.ListenAddr) > 0 && cfg.ListenAddr[0] == ':' {
				logger.Warn("binding to all interfaces without allow_lan is disabled for security, constraining to 127.0.0.1", "original", cfg.ListenAddr)
				cfg.ListenAddr = "127.0.0.1" + cfg.ListenAddr
			} else if len(cfg.ListenAddr) >= 8 && cfg.ListenAddr[:8] == "0.0.0.0:" {
				logger.Warn("binding to 0.0.0.0 without allow_lan is disabled for security, constraining to 127.0.0.1", "original", cfg.ListenAddr)
				cfg.ListenAddr = "127.0.0.1:" + cfg.ListenAddr[8:]
			}
		} else {
			if len(cfg.ListenAddr) >= 10 && cfg.ListenAddr[:10] == "127.0.0.1:" {
				cfg.ListenAddr = "0.0.0.0:" + cfg.ListenAddr[10:]
			} else if len(cfg.ListenAddr) > 0 && cfg.ListenAddr[0] == ':' {
				cfg.ListenAddr = "0.0.0.0" + cfg.ListenAddr
			}
		}
	}

	// Initialize credential pool with proactive refresh + circuit breaking
	tp := auth.NewPool(&auth.PoolOptions{
		Refresher: auth.NewTokenRefresher(config.AgentDomain, nil),
		Logger:    logger,
	})

	if len(cfg.Accounts) > 0 {
		for _, acc := range cfg.Accounts {
			switch {
			case acc.Token != "":
				tp.AddAccountWithToken(acc.Name, acc.Token)
				logger.Info("added account (direct token)", "name", acc.Name)
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
	} else {
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
	logger.Info("  trae-proxy v" + version)
	logger.Info("  Trae CN -> OpenAI Compatible API")
	logger.Info("====================================")

	if err := server.ListenAndServe(cfg.ListenAddr); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
