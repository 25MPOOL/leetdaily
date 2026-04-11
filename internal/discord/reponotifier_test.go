package discord_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

func newTestClient(t *testing.T) (*discord.Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "message-1"})
	}))
	t.Cleanup(server.Close)
	client, err := discord.NewClientWithBaseURL(server.Client(), "token", server.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL() error = %v", err)
	}
	return client, server
}

func TestNewRepositoryNotifierRejectsNilClient(t *testing.T) {
	t.Parallel()

	if _, err := discord.NewRepositoryNotifier(&stubRepository{}, nil, silentLogger()); err == nil {
		t.Fatal("NewRepositoryNotifier(nil client) got nil, want error")
	}
}

func TestRepositoryNotifierSkipsWhenGuildSettingsLoadFails(t *testing.T) {
	t.Parallel()

	client, _ := newTestClient(t)
	repo := &stubRepository{guildsErr: errors.New("backend unavailable")}
	notifier, err := discord.NewRepositoryNotifier(repo, client, silentLogger())
	if err != nil {
		t.Fatalf("NewRepositoryNotifier() error = %v", err)
	}

	// Should return nil (skip, not propagate) when settings cannot be loaded.
	if err := notifier.NotifyFailure(context.Background(), "123456789012345678", errors.New("job failed")); err != nil {
		t.Fatalf("NotifyFailure() error = %v, want nil (skip)", err)
	}
}

func TestRepositoryNotifierSkipsWhenGuildNotFound(t *testing.T) {
	t.Parallel()

	client, _ := newTestClient(t)
	repo := &stubRepository{
		guilds: config.GuildSettings{
			Guilds: []config.Guild{
				{GuildID: "111111111111111111", NotificationChannelID: "222222222222222222"},
			},
		},
	}
	notifier, err := discord.NewRepositoryNotifier(repo, client, silentLogger())
	if err != nil {
		t.Fatalf("NewRepositoryNotifier() error = %v", err)
	}

	// Guild ID not in settings: should skip silently.
	if err := notifier.NotifyFailure(context.Background(), "999999999999999999", errors.New("job failed")); err != nil {
		t.Fatalf("NotifyFailure() error = %v, want nil (skip)", err)
	}
}

func TestRepositoryNotifierSkipsWhenChannelIDIsInvalid(t *testing.T) {
	t.Parallel()

	client, _ := newTestClient(t)
	repo := &stubRepository{
		guilds: config.GuildSettings{
			Guilds: []config.Guild{
				{GuildID: "123456789012345678", NotificationChannelID: "not-a-snowflake"},
			},
		},
	}
	notifier, err := discord.NewRepositoryNotifier(repo, client, silentLogger())
	if err != nil {
		t.Fatalf("NewRepositoryNotifier() error = %v", err)
	}

	// Invalid channel ID: NewNotifier will fail, should skip silently.
	if err := notifier.NotifyFailure(context.Background(), "123456789012345678", errors.New("job failed")); err != nil {
		t.Fatalf("NotifyFailure() error = %v, want nil (skip)", err)
	}
}

func TestRepositoryNotifierForwardsNotificationToGuildChannel(t *testing.T) {
	t.Parallel()

	var gotPath, gotContent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			if content, ok := body["content"].(string); ok {
				gotContent = content
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "message-1"})
	}))
	defer server.Close()

	client, err := discord.NewClientWithBaseURL(server.Client(), "token", server.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL() error = %v", err)
	}

	repo := &stubRepository{
		guilds: config.GuildSettings{
			Guilds: []config.Guild{
				{GuildID: "123456789012345678", NotificationChannelID: "234567890123456789"},
			},
		},
	}
	notifier, err := discord.NewRepositoryNotifier(repo, client, silentLogger())
	if err != nil {
		t.Fatalf("NewRepositoryNotifier() error = %v", err)
	}

	if err := notifier.NotifyFailure(context.Background(), "123456789012345678", errors.New("job failed")); err != nil {
		t.Fatalf("NotifyFailure() error = %v", err)
	}

	if gotPath != "/channels/234567890123456789/messages" {
		t.Fatalf("request path = %q, want /channels/234567890123456789/messages", gotPath)
	}
	if !strings.Contains(gotContent, "123456789012345678") {
		t.Fatalf("message content = %q, want it to contain guild ID", gotContent)
	}
}
