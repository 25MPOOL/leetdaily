package discord_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nkoji21/leetdaily/internal/discord"
)

func TestNotifierNotifyFailure(t *testing.T) {
	t.Parallel()

	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			if content, ok := body["content"].(string); ok {
				gotBody = content
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "msg-1"})
	}))
	defer server.Close()

	client, err := discord.NewClientWithBaseURL(server.Client(), "token", server.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL() error = %v", err)
	}

	notifier, err := discord.NewNotifier(client, "111111111111111111")
	if err != nil {
		t.Fatalf("NewNotifier() error = %v", err)
	}

	if err := notifier.NotifyFailure(context.Background(), "guild-1", errors.New("something broke")); err != nil {
		t.Fatalf("NotifyFailure() error = %v", err)
	}

	if !strings.Contains(gotBody, "guild-1") {
		t.Errorf("NotifyFailure() message = %q, want it to contain guild ID", gotBody)
	}
	if !strings.Contains(gotBody, "something broke") {
		t.Errorf("NotifyFailure() message = %q, want it to contain the error", gotBody)
	}
}

func TestNotifierNotifyFailureRequiresError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "msg-1"})
	}))
	defer server.Close()

	client, err := discord.NewClientWithBaseURL(server.Client(), "token", server.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL() error = %v", err)
	}

	notifier, err := discord.NewNotifier(client, "111111111111111111")
	if err != nil {
		t.Fatalf("NewNotifier() error = %v", err)
	}

	if err := notifier.NotifyFailure(context.Background(), "guild-1", nil); err == nil {
		t.Fatal("NotifyFailure() with nil error: got nil, want error")
	}
}

func TestNewNotifierRejectsNilClient(t *testing.T) {
	t.Parallel()

	if _, err := discord.NewNotifier(nil, "111111111111111111"); err == nil {
		t.Fatal("NewNotifier(nil, ...) got nil, want error")
	}
}

func TestNewNotifierRejectsEmptyChannel(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer server.Close()

	client, err := discord.NewClientWithBaseURL(server.Client(), "token", server.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL() error = %v", err)
	}

	if _, err := discord.NewNotifier(client, ""); err == nil {
		t.Fatal("NewNotifier(client, \"\") got nil, want error")
	}
}
