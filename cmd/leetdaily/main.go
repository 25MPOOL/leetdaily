package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nkoji21/leetdaily/internal/app"
	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/discord"
	"github.com/nkoji21/leetdaily/internal/discordapp"
	"github.com/nkoji21/leetdaily/internal/httpruntime"
	"github.com/nkoji21/leetdaily/internal/job"
	"github.com/nkoji21/leetdaily/internal/leetcode"
	"github.com/nkoji21/leetdaily/internal/logging"
	"github.com/nkoji21/leetdaily/internal/runtimecfg"
	"github.com/nkoji21/leetdaily/internal/state"
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

	location, err := time.LoadLocation("UTC")
	if err != nil {
		return app.Dependencies{}, err
	}

	location, err = loadRuntimeLocation(ctx, repository)
	if err != nil {
		return app.Dependencies{}, err
	}

	leetcodeClient := leetcode.NewClient(nil)
	jobRunner, err := job.New(
		repository,
		leetcodeClient,
		discordClient,
		newNotifier(repository, discordClient, logger),
	)
	if err != nil {
		return app.Dependencies{}, fmt.Errorf("build job runner: %w", err)
	}

	httpOptions := httpruntime.Options{}
	if cfg.DiscordAppKey != "" {
		interactionHandler, err := discordapp.NewHandler(cfg.DiscordAppKey, repository)
		if err != nil {
			return app.Dependencies{}, fmt.Errorf("build Discord interaction handler: %w", err)
		}
		httpOptions.DiscordInteractions = interactionHandler
	}

	httpRunner, err := httpruntime.NewWithOptions(cfg.HTTPAddr(), location, jobRunner, httpOptions)
	if err != nil {
		return app.Dependencies{}, fmt.Errorf("build HTTP runtime: %w", err)
	}

	return app.Dependencies{
		HTTPRunner: httpRunner,
		JobRunner:  jobModeRunner{runner: jobRunner, location: location},
	}, nil
}

func loadRuntimeLocation(ctx context.Context, repository storage.Repository) (*time.Location, error) {
	configValue, err := repository.LoadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load config for runtime wiring: %w", err)
	}

	if _, err := loadGuildSettings(ctx, repository); err != nil {
		return nil, fmt.Errorf("load guild settings for runtime wiring: %w", err)
	}

	location, err := configValue.Location()
	if err != nil {
		return nil, fmt.Errorf("load configured timezone: %w", err)
	}

	return location, nil
}

type jobModeRunner struct {
	runner   *job.Runner
	location *time.Location
}

func (r jobModeRunner) Run(ctx context.Context) error {
	targetDate, err := state.ParseDate(time.Now().In(r.location).Format("2006-01-02"))
	if err != nil {
		return err
	}

	return r.runner.Run(ctx, targetDate)
}

type repositoryNotifier struct {
	repository storage.Repository
	client     *discord.Client
	logger     *slog.Logger
}

func newNotifier(repository storage.Repository, client *discord.Client, logger *slog.Logger) *repositoryNotifier {
	return &repositoryNotifier{
		repository: repository,
		client:     client,
		logger:     logger,
	}
}

func (n *repositoryNotifier) NotifyFailure(ctx context.Context, guildID string, err error) error {
	guilds, notifyErr := loadGuildSettings(ctx, n.repository)
	if notifyErr != nil {
		n.logger.Warn("skip failure notification because guild settings could not be loaded", "guild_id", guildID, "error", notifyErr)
		return nil
	}

	var guild config.Guild
	ok := false
	for _, candidate := range guilds.Guilds {
		if candidate.GuildID == guildID {
			guild = candidate
			ok = true
			break
		}
	}
	if !ok {
		n.logger.Warn("skip missing notifier mapping", "guild_id", guildID)
		return nil
	}

	notifier, notifyErr := discord.NewNotifier(n.client, guild.NotificationChannelID)
	if notifyErr != nil {
		n.logger.Warn("skip invalid notifier channel", "guild_id", guildID, "channel_id", guild.NotificationChannelID, "error", notifyErr)
		return nil
	}
	return notifier.NotifyFailure(ctx, guildID, err)
}

func loadGuildSettings(ctx context.Context, repository storage.Repository) (config.GuildSettings, error) {
	guilds, _, err := repository.LoadGuildSettings(ctx)
	if err != nil {
		return config.GuildSettings{}, err
	}
	return guilds, nil
}
