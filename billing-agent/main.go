package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := LoadConfig()
	log := NewLogger()
	tx := NewTransport(cfg, log)

	src, err := NewProcessMetrics(cfg.TenantID, cfg.ServiceID, cfg.RevisionID)
	if err != nil {
		log.Error("config error", map[string]any{"err": err.Error()})
		os.Exit(1)
	}

	agent := NewAgent(cfg, log, tx, src)

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	agent.Run(ctx)
}
