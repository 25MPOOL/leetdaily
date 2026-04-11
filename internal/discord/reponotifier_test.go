package discord_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/discord"
	"github.com/nkoji21/leetdaily/internal/problemcache"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
)

// stubRepository implements storage.Repository with configurable LoadGuildSettings behavior.
type stubRepository struct {
	guilds    config.GuildSettings
	guildsErr error
}

func (r *stubRepository) LoadConfig(_ context.Context) (config.Config, error) {
	return config.Config{}, nil
}
func (r *stubRepository) LoadGuildSettings(_ context.Context) (config.GuildSettings, storage.Version, error) {
	return r.guilds, storage.Version{}, r.guildsErr
}
func (r *stubRepository) SaveGuildSettings(_ context.Context, _ config.GuildSettings, _ storage.Version) (storage.Version, error) {
	return storage.Version{}, nil
}
func (r *stubRepository) LoadState(_ context.Context) (state.State, storage.Version, error) {
	return state.State{}, storage.Version{}, nil
}
func (r *stubRepository) SaveState(_ context.Context, _ state.State, _ storage.Version) (storage.Version, error) {
	return storage.Version{}, nil
}
func (r *stubRepository) LoadProblemCache(_ context.Context) (problemcache.Cache, storage.Version, error) {
	return problemcache.Cache{}, storage.Version{}, nil
}
func (r *stubRepository) SaveProblemCache(_ context.Context, _ problemcache.Cache, _ storage.Version) (storage.Version, error) {
	return storage.Version{}, nil
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

func TestRepositoryNotifierSkipsWhenGuildSettingsLoadFails(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{guildsErr: errors.New("backend unavailable")}
	notifier := discord.NewRepositoryNotifier(repo, nil, silentLogger())

	// Should return nil (skip, not propagate) when settings cannot be loaded.
	if err := notifier.NotifyFailure(context.Background(), "123456789012345678", errors.New("job failed")); err != nil {
		t.Fatalf("NotifyFailure() error = %v, want nil (skip)", err)
	}
}

func TestRepositoryNotifierSkipsWhenGuildNotFound(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{
		guilds: config.GuildSettings{
			Guilds: []config.Guild{
				{GuildID: "111111111111111111", NotificationChannelID: "222222222222222222"},
			},
		},
	}
	notifier := discord.NewRepositoryNotifier(repo, nil, silentLogger())

	// Guild ID not in settings: should skip silently.
	if err := notifier.NotifyFailure(context.Background(), "999999999999999999", errors.New("job failed")); err != nil {
		t.Fatalf("NotifyFailure() error = %v, want nil (skip)", err)
	}
}

func TestRepositoryNotifierSkipsWhenChannelIDIsInvalid(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{
		guilds: config.GuildSettings{
			Guilds: []config.Guild{
				{GuildID: "123456789012345678", NotificationChannelID: "not-a-snowflake"},
			},
		},
	}
	notifier := discord.NewRepositoryNotifier(repo, nil, silentLogger())

	// Invalid channel ID: NewNotifier will fail, should skip silently.
	if err := notifier.NotifyFailure(context.Background(), "123456789012345678", errors.New("job failed")); err != nil {
		t.Fatalf("NotifyFailure() error = %v, want nil (skip)", err)
	}
}
