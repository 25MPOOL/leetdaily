package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/nkoji21/leetdaily/internal/runtimecfg"
)

type Runner interface {
	Run(context.Context) error
}

type Dependencies struct {
	HTTPRunner Runner
	JobRunner  Runner
}

type App struct {
	cfg        runtimecfg.Config
	logger     *slog.Logger
	httpRunner Runner
	jobRunner  Runner
}

func New(cfg runtimecfg.Config, logger *slog.Logger, deps Dependencies) *App {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	app := &App{
		cfg:    cfg,
		logger: logger,
		httpRunner: placeholderRunner{
			mode:   runtimecfg.ModeHTTP,
			logger: logger.With("component", "http_runtime"),
		},
		jobRunner: placeholderRunner{
			mode:   runtimecfg.ModeJob,
			logger: logger.With("component", "job_runtime"),
		},
	}

	if deps.HTTPRunner != nil {
		app.httpRunner = deps.HTTPRunner
	}

	if deps.JobRunner != nil {
		app.jobRunner = deps.JobRunner
	}

	return app
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("starting leetdaily", "mode", a.cfg.Mode, "data_dir", a.cfg.DataDir)

	switch a.cfg.Mode {
	case runtimecfg.ModeHTTP:
		return a.httpRunner.Run(ctx)
	case runtimecfg.ModeJob:
		return a.jobRunner.Run(ctx)
	default:
		return fmt.Errorf("unsupported runtime mode %q", a.cfg.Mode)
	}
}

type placeholderRunner struct {
	mode   runtimecfg.Mode
	logger *slog.Logger
}

func (r placeholderRunner) Run(context.Context) error {
	r.logger.Warn("runtime skeleton is ready but implementation is deferred", "mode", r.mode)
	return fmt.Errorf("%s runtime is not implemented yet", r.mode)
}
