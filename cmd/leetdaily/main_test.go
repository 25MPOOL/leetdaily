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

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/discord"
)

func TestMultiNotifierRoutesToGuildChannel(t *testing.T) {
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

	notifier := mustNotifier(client, []config.Guild{
		{GuildID: "111111111111111111", NotificationChannelID: "222222222222222222"},
		{GuildID: "333333333333333333", NotificationChannelID: "444444444444444444"},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := notifier.NotifyFailure(context.Background(), "333333333333333333", errors.New("boom")); err != nil {
		t.Fatalf("NotifyFailure() error = %v", err)
	}

	want := []string{"/channels/444444444444444444/messages"}
	if !slices.Equal(channels, want) {
		t.Fatalf("channels = %v, want %v", channels, want)
	}
}
