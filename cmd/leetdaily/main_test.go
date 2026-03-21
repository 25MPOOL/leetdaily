package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/discord"
	"github.com/nkoji21/leetdaily/internal/problemcache"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
)

func TestRepositoryNotifierRoutesToGuildChannel(t *testing.T) {
	t.Parallel()

	var channels []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		channels = append(channels, r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "message-1"})
	}))
	defer server.Close()

	client, err := discord.NewClientWithBaseURL(server.Client(), "token", server.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL() error = %v", err)
	}

	notifier := newNotifier(&stubRepository{
		guilds: config.GuildSettings{
			Guilds: []config.Guild{
				{GuildID: "111111111111111111", NotificationChannelID: "222222222222222222"},
				{GuildID: "333333333333333333", NotificationChannelID: "444444444444444444"},
			},
		},
	}, client, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := notifier.NotifyFailure(context.Background(), "333333333333333333", errors.New("boom")); err != nil {
		t.Fatalf("NotifyFailure() error = %v", err)
	}

	want := []string{"/channels/444444444444444444/messages"}
	if !slices.Equal(channels, want) {
		t.Fatalf("channels = %v, want %v", channels, want)
	}
}

func TestLoadRuntimeLocationValidatesGuildSettings(t *testing.T) {
	t.Parallel()

	wantLocation, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("LoadLocation() error = %v", err)
	}

	location, err := loadRuntimeLocation(context.Background(), &stubRepository{
		cfg: config.Config{Timezone: "Asia/Tokyo"},
		guilds: config.GuildSettings{
			Guilds: []config.Guild{
				{
					GuildID:               "111111111111111111",
					Enabled:               true,
					ForumChannelID:        "222222222222222222",
					NotificationChannelID: "333333333333333333",
					StartProblemNumber:    1,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("loadRuntimeLocation() error = %v", err)
	}
	if location.String() != wantLocation.String() {
		t.Fatalf("location = %q, want %q", location.String(), wantLocation.String())
	}
}

func TestLoadRuntimeLocationReturnsGuildSettingsError(t *testing.T) {
	t.Parallel()

	_, err := loadRuntimeLocation(context.Background(), &stubRepository{
		cfg:      config.Config{Timezone: "Asia/Tokyo"},
		guildErr: errors.New("duplicate guild"),
	})
	if err == nil {
		t.Fatal("loadRuntimeLocation() error = nil, want error")
	}
	if got, want := err.Error(), "load guild settings for runtime wiring: duplicate guild"; got != want {
		t.Fatalf("loadRuntimeLocation() error = %q, want %q", got, want)
	}
}

type stubRepository struct {
	cfg       config.Config
	configErr error
	guilds    config.GuildSettings
	guildErr  error
}

func (s *stubRepository) LoadConfig(context.Context) (config.Config, error) {
	if s.configErr != nil {
		return config.Config{}, s.configErr
	}
	return s.cfg, nil
}

func (s *stubRepository) LoadGuildSettings(context.Context) (config.GuildSettings, storage.Version, error) {
	if s.guildErr != nil {
		return config.GuildSettings{}, storage.Version{}, s.guildErr
	}
	return s.guilds, storage.Version{}, nil
}

func (s *stubRepository) SaveGuildSettings(context.Context, config.GuildSettings, storage.Version) (storage.Version, error) {
	return storage.Version{}, nil
}

func (s *stubRepository) LoadState(context.Context) (state.State, storage.Version, error) {
	return state.State{}, storage.Version{}, nil
}

func (s *stubRepository) SaveState(context.Context, state.State, storage.Version) (storage.Version, error) {
	return storage.Version{}, nil
}

func (s *stubRepository) LoadProblemCache(context.Context) (problemcache.Cache, storage.Version, error) {
	return problemcache.Cache{}, storage.Version{}, nil
}

func (s *stubRepository) SaveProblemCache(context.Context, problemcache.Cache, storage.Version) (storage.Version, error) {
	return storage.Version{}, nil
}
