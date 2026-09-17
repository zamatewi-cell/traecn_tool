package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
	"github.com/zamatewi-cell/traecn_tool/internal/openai"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
)

var version = "0.1.0"

func main() {
	configPath := flag.String("config", "config.json", "config file path")
	listen := flag.String("listen", ":9090", "listen address")
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

	if *listen != ":9090" {
		cfg.ListenAddr = *listen
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

	// Start proxy
	traeProxy := proxy.NewTraeProxy(tp, logger)
	traeProxy.SetProtection(cfg.Protect)
	server := openai.NewServer(traeProxy, logger, nil)

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
