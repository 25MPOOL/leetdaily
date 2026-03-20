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
	"github.com/nkoji21/leetdaily/internal/httpruntime"
	"github.com/nkoji21/leetdaily/internal/job"
	"github.com/nkoji21/leetdaily/internal/leetcode"
	"github.com/nkoji21/leetdaily/internal/logging"
	"github.com/nkoji21/leetdaily/internal/runtimecfg"
	"github.com/nkoji21/leetdaily/internal/state"
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

	configValue, err := repository.LoadConfig(ctx)
	if err != nil {
		return app.Dependencies{}, fmt.Errorf("load config for runtime wiring: %w", err)
	}
	location, err = configValue.Location()
	if err != nil {
		return app.Dependencies{}, fmt.Errorf("load configured timezone: %w", err)
	}

	leetcodeClient := leetcode.NewClient(nil)
	jobRunner, err := job.New(
		repository,
		leetcodeClient,
		discordClient,
		mustNotifier(discordClient, configValue.Guilds, logger),
	)
	if err != nil {
		return app.Dependencies{}, fmt.Errorf("build job runner: %w", err)
	}

	httpRunner, err := httpruntime.New(cfg.HTTPAddr(), location, jobRunner)
	if err != nil {
		return app.Dependencies{}, fmt.Errorf("build HTTP runtime: %w", err)
	}

	return app.Dependencies{
		HTTPRunner: httpRunner,
		JobRunner:  jobModeRunner{runner: jobRunner, location: location},
	}, nil
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

type multiNotifier struct {
	notifiers []*discord.Notifier
	logger    *slog.Logger
}

func mustNotifier(client *discord.Client, guilds []config.Guild, logger *slog.Logger) *multiNotifier {
	byChannel := map[string]*discord.Notifier{}
	notifiers := make([]*discord.Notifier, 0, len(guilds))
	for _, guild := range guilds {
		if _, ok := byChannel[guild.NotificationChannelID]; ok {
			continue
		}
		notifier, err := discord.NewNotifier(client, guild.NotificationChannelID)
		if err != nil {
			logger.Warn("skip invalid notifier channel", "channel_id", guild.NotificationChannelID, "error", err)
			continue
		}
		byChannel[guild.NotificationChannelID] = notifier
		notifiers = append(notifiers, notifier)
	}
	return &multiNotifier{notifiers: notifiers, logger: logger}
}

func (m *multiNotifier) NotifyFailure(ctx context.Context, guildID string, err error) error {
	var notifyErr error
	for _, notifier := range m.notifiers {
		if currentErr := notifier.NotifyFailure(ctx, guildID, err); currentErr != nil {
			notifyErr = currentErr
		}
	}
	return notifyErr
}
