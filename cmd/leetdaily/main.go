package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/nkoji21/leetdaily/internal/app"
	"github.com/nkoji21/leetdaily/internal/bot"
	"github.com/nkoji21/leetdaily/internal/discord"
	"github.com/nkoji21/leetdaily/internal/httpruntime"
	"github.com/nkoji21/leetdaily/internal/job"
	"github.com/nkoji21/leetdaily/internal/logging"
	"github.com/nkoji21/leetdaily/internal/neetcode"
	"github.com/nkoji21/leetdaily/internal/runtimecfg"
	"github.com/nkoji21/leetdaily/internal/storage"
	"github.com/nkoji21/leetdaily/internal/storage/provider"
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

	deps, err := buildDependencies(ctx, cfg, logger)
	if err != nil {
		return err
	}

	if err := app.New(cfg, logger, deps).Run(ctx); err != nil {
		return fmt.Errorf("run application: %w", err)
	}

	return nil
}

func buildDependencies(ctx context.Context, cfg runtimecfg.Config, logger *slog.Logger) (app.Dependencies, error) {
	repository, err := provider.NewRepository(ctx, cfg)
	if err != nil {
		return app.Dependencies{}, fmt.Errorf("build repository: %w", err)
	}

	discordClient, err := discord.NewClient(nil, cfg.DiscordBotToken)
	if err != nil {
		return app.Dependencies{}, fmt.Errorf("build Discord client: %w", err)
	}

	location, err := loadRuntimeLocation(ctx, repository)
	if err != nil {
		return app.Dependencies{}, err
	}

	neetcodeClient := neetcode.NewProblemSource(cfg.NeetCodeProblemsPath())
	notifier, err := discord.NewRepositoryNotifier(repository, discordClient, logger)
	if err != nil {
		return app.Dependencies{}, fmt.Errorf("build notifier: %w", err)
	}
	jobRunner, err := job.New(
		repository,
		neetcodeClient,
		discordClient,
		notifier,
	)
	if err != nil {
		return app.Dependencies{}, fmt.Errorf("build job runner: %w", err)
	}

	httpRunner, err := httpruntime.New(cfg.HTTPAddr(), location, jobRunner)
	if err != nil {
		return app.Dependencies{}, fmt.Errorf("build HTTP runtime: %w", err)
	}

	deps := app.Dependencies{
		HTTPRunner: httpRunner,
		JobRunner:  job.NewDateRunner(jobRunner, location),
	}

	if cfg.Mode == runtimecfg.ModeBot {
		session, err := discordgo.New("Bot " + cfg.DiscordBotToken)
		if err != nil {
			return app.Dependencies{}, fmt.Errorf("create Discord session: %w", err)
		}

		botRunner, err := bot.New(bot.Config{
			Session:    session,
			Job:        jobRunner,
			Repository: repository,
			AppID:      cfg.DiscordApplicationID,
			Location:   location,
			HTTPAddr:   cfg.HTTPAddr(),
			Logger:     logger.With("component", "bot"),
		})
		if err != nil {
			return app.Dependencies{}, fmt.Errorf("build bot runtime: %w", err)
		}
		deps.BotRunner = botRunner
	}

	return deps, nil
}

func loadRuntimeLocation(ctx context.Context, repository storage.Repository) (*time.Location, error) {
	configValue, err := repository.LoadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load config for runtime wiring: %w", err)
	}

	if _, _, err = repository.LoadGuildSettings(ctx); err != nil {
		return nil, fmt.Errorf("load guild settings for runtime wiring: %w", err)
	}

	location, err := configValue.Location()
	if err != nil {
		return nil, fmt.Errorf("load configured timezone: %w", err)
	}

	return location, nil
}
