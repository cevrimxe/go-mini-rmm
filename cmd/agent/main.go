package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cevrimxe/go-mini-rmm/internal/agent/executor"
	"github.com/cevrimxe/go-mini-rmm/internal/agent/heartbeat"
	"github.com/cevrimxe/go-mini-rmm/internal/agent/updater"
)

var Version = "dev"

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "RMM server URL")
	agentKey := flag.String("key", "", "Agent key (ID) – sunucuda bu agent'ı tanımak için kullanılır")
	displayName := flag.String("name", "", "Görünen isim (kurulumda girilen, dashboard'da gösterilir)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if *agentKey == "" {
		slog.Error("agent key is required (-key flag)")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start heartbeat
	hb := heartbeat.New(*serverURL, *agentKey, *displayName, Version)
	go hb.Run(ctx)

	// Start WebSocket executor
	exec := executor.New(*serverURL, *agentKey)
	go exec.Run(ctx)

	// Start auto-updater
	upd := updater.New(*serverURL, Version)
	go upd.Run(ctx)

	slog.Info("agent started", "server", *serverURL, "version", Version)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("agent shutting down...")
	cancel()
}
