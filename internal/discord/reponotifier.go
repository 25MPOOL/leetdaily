package discord

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/storage"
)

// RepositoryNotifier implements job.Notifier by looking up each guild's
// notification channel from the repository at call time.
type RepositoryNotifier struct {
	repository storage.Repository
	client     *Client
	logger     *slog.Logger
}

func NewRepositoryNotifier(repository storage.Repository, client *Client, logger *slog.Logger) (*RepositoryNotifier, error) {
	if client == nil {
		return nil, fmt.Errorf("discord client must not be nil")
	}
	return &RepositoryNotifier{
		repository: repository,
		client:     client,
		logger:     logger,
	}, nil
}

func (n *RepositoryNotifier) NotifyFailure(ctx context.Context, guildID string, err error) error {
	guilds, _, loadErr := n.repository.LoadGuildSettings(ctx)
	if loadErr != nil {
		n.logger.Warn("skip failure notification because guild settings could not be loaded", "guild_id", guildID, "error", loadErr)
		return nil
	}

	guild, ok := findGuild(guilds.Guilds, guildID)
	if !ok {
		n.logger.Warn("skip missing notifier mapping", "guild_id", guildID)
		return nil
	}

	notifier, notifyErr := NewNotifier(n.client, guild.NotificationChannelID)
	if notifyErr != nil {
		n.logger.Warn("skip invalid notifier channel", "guild_id", guildID, "channel_id", guild.NotificationChannelID, "error", notifyErr)
		return nil
	}

	return notifier.NotifyFailure(ctx, guildID, err)
}

func findGuild(guilds []config.Guild, guildID string) (config.Guild, bool) {
	for _, g := range guilds {
		if g.GuildID == guildID {
			return g, true
		}
	}
	return config.Guild{}, false
}
