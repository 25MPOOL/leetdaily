package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nkoji21/leetdaily/internal/app"
	"github.com/nkoji21/leetdaily/internal/logging"
	"github.com/nkoji21/leetdaily/internal/runtimecfg"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "leetdaily: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := runtimecfg.Load()
	if err != nil {
		return fmt.Errorf("load runtime config: %w", err)
	}

	logger := logging.New(cfg.LogLevel, os.Stdout)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.New(cfg, logger, app.Dependencies{}).Run(ctx); err != nil {
		return fmt.Errorf("run application: %w", err)
	}

	return nil
}
