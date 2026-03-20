package discord

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnsureDifficultyTagsCreatesMissingTags(t *testing.T) {
	t.Parallel()

	var patchedTags []ForumTag
	moderated := true
	emojiName := "sparkles"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/channels/forum-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "forum-1",
				"available_tags": []map[string]any{
					{"id": "easy-1", "name": "Easy", "moderated": true, "emoji_name": "sparkles"},
				},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/channels/forum-1":
			var payload struct {
				AvailableTags []ForumTag `json:"available_tags"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode patch error = %v", err)
			}
			patchedTags = payload.AvailableTags
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "forum-1",
				"available_tags": []map[string]any{
					{"id": "easy-1", "name": "Easy"},
					{"id": "medium-1", "name": "Medium"},
					{"id": "hard-1", "name": "Hard"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClientWithBaseURL(server.Client(), "token", server.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL() error = %v", err)
	}

	tags, err := client.EnsureDifficultyTags(context.Background(), "forum-1")
	if err != nil {
		t.Fatalf("EnsureDifficultyTags() error = %v", err)
	}

	if len(patchedTags) != 3 {
		t.Fatalf("len(patchedTags) = %d, want 3", len(patchedTags))
	}
	if patchedTags[0].Moderated == nil || *patchedTags[0].Moderated != moderated {
		t.Fatalf("patchedTags[0].Moderated = %v, want %v", patchedTags[0].Moderated, moderated)
	}
	if patchedTags[0].EmojiName == nil || *patchedTags[0].EmojiName != emojiName {
		t.Fatalf("patchedTags[0].EmojiName = %v, want %q", patchedTags[0].EmojiName, emojiName)
	}
	if tags["Medium"] != "medium-1" {
		t.Fatalf("tags[Medium] = %q, want %q", tags["Medium"], "medium-1")
	}
}

func TestEnsureDifficultyTagsUsesNormalizedNamesWhenTagsAlreadyExist(t *testing.T) {
	t.Parallel()

	patchCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/channels/forum-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "forum-1",
				"available_tags": []map[string]any{
					{"id": "easy-1", "name": " easy "},
					{"id": "medium-1", "name": "MEDIUM"},
					{"id": "hard-1", "name": "Hard"},
				},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/channels/forum-1":
			patchCalls++
			t.Fatalf("unexpected patch request")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClientWithBaseURL(server.Client(), "token", server.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL() error = %v", err)
	}

	tags, err := client.EnsureDifficultyTags(context.Background(), "forum-1")
	if err != nil {
		t.Fatalf("EnsureDifficultyTags() error = %v", err)
	}

	if patchCalls != 0 {
		t.Fatalf("patchCalls = %d, want 0", patchCalls)
	}
	if tags["Easy"] != "easy-1" || tags["Medium"] != "medium-1" || tags["Hard"] != "hard-1" {
		t.Fatalf("tags = %#v, want all normalized difficulty tags", tags)
	}
}

func TestCreateForumThreadAndSendNotification(t *testing.T) {
	t.Parallel()

	var threadPayload map[string]any
	var notificationPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/channels/forum-1/threads":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &threadPayload)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "thread-1"})
		case "/channels/channel-1/messages":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &notificationPayload)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "message-1"})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClientWithBaseURL(server.Client(), "token", server.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL() error = %v", err)
	}

	thread, err := client.CreateForumThread(context.Background(), "forum-1", "easy-1", "Two Sum", "Discuss this problem")
	if err != nil {
		t.Fatalf("CreateForumThread() error = %v", err)
	}
	if thread.ID != "thread-1" {
		t.Fatalf("thread.ID = %q, want %q", thread.ID, "thread-1")
	}

	notifier, err := NewNotifier(client, "channel-1")
	if err != nil {
		t.Fatalf("NewNotifier() error = %v", err)
	}
	if err := notifier.NotifyFailure(context.Background(), "guild-1", errors.New("missing permissions")); err != nil {
		t.Fatalf("NotifyFailure() error = %v", err)
	}

	if threadPayload["name"] != "Two Sum" {
		t.Fatalf("threadPayload[name] = %#v, want %q", threadPayload["name"], "Two Sum")
	}
	if notificationPayload["content"] == "" {
		t.Fatal("notification content is empty")
	}
}
