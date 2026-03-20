package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/discord"
	"github.com/nkoji21/leetdaily/internal/job"
	"github.com/nkoji21/leetdaily/internal/leetcode"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
	"github.com/nkoji21/leetdaily/internal/storage/filesystem"
)

func TestJobFlowIntegration(t *testing.T) {
	t.Parallel()

	targetDate := mustDate(t, "2026-03-20")

	t.Run("initial success", func(t *testing.T) {
		t.Parallel()
		env := newIntegrationEnv(t)
		if err := env.runner.Run(context.Background(), targetDate); err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		if env.discord.threadCalls != 1 {
			t.Fatalf("threadCalls = %d, want 1", env.discord.threadCalls)
		}
		if env.discord.notificationCalls != 0 {
			t.Fatalf("notificationCalls = %d, want 0", env.discord.notificationCalls)
		}

		current, _, err := env.repository.LoadState(context.Background())
		if err != nil {
			t.Fatalf("LoadState() error = %v", err)
		}
		got := current.GuildStates[env.guild.GuildID]
		if got.Job.Status != state.JobStatusPosted {
			t.Fatalf("Job.Status = %q, want %q", got.Job.Status, state.JobStatusPosted)
		}
		if got.NextProblemNumber != 2 {
			t.Fatalf("NextProblemNumber = %d, want 2", got.NextProblemNumber)
		}
	})

	t.Run("retry then success", func(t *testing.T) {
		t.Parallel()
		env := newIntegrationEnv(t)
		env.discord.mu.Lock()
		env.discord.threadFailures = 1
		env.discord.mu.Unlock()

		if err := env.runner.Run(context.Background(), targetDate); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if env.discord.threadCalls != 2 {
			t.Fatalf("threadCalls = %d, want 2", env.discord.threadCalls)
		}
		if env.discord.notificationCalls != 0 {
			t.Fatalf("notificationCalls = %d, want 0", env.discord.notificationCalls)
		}
	})

	t.Run("final failure sends notification", func(t *testing.T) {
		t.Parallel()
		env := newIntegrationEnv(t)
		env.discord.mu.Lock()
		env.discord.threadFailures = 3
		env.discord.mu.Unlock()

		if err := env.runner.Run(context.Background(), targetDate); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if env.discord.notificationCalls != 1 {
			t.Fatalf("notificationCalls = %d, want 1", env.discord.notificationCalls)
		}
	})

	t.Run("same day skip avoids duplicate post", func(t *testing.T) {
		t.Parallel()
		env := newIntegrationEnv(t)

		now := time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC)
		problemNumber := 1
		threadID := "456789012345678901"
		current := state.State{
			GuildStates: map[string]state.GuildState{
				env.guild.GuildID: {
					NextProblemNumber:       2,
					LastPostedProblemNumber: &problemNumber,
					LastPostedAt:            &now,
					LastPostedThreadID:      &threadID,
					Job: state.JobState{
						TargetDate:    &targetDate,
						Status:        state.JobStatusPosted,
						ProblemNumber: &problemNumber,
					},
				},
			},
		}
		if _, err := env.repository.SaveState(context.Background(), current, storage.Version{}); err != nil {
			t.Fatalf("SaveState() error = %v", err)
		}

		if err := env.runner.Run(context.Background(), targetDate); err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		if env.discord.threadCalls != 0 {
			t.Fatalf("threadCalls = %d, want 0", env.discord.threadCalls)
		}
	})
}

type integrationEnv struct {
	repository *filesystem.Repository
	runner     *job.Runner
	guild      config.Guild
	discord    *fakeDiscord
}

func newIntegrationEnv(t *testing.T) *integrationEnv {
	t.Helper()

	root := t.TempDir()
	repository, err := filesystem.New(storage.Paths{
		ConfigPath:   filepath.Join(root, "config.json"),
		StatePath:    filepath.Join(root, "state.json"),
		ProblemsPath: filepath.Join(root, "problems.json"),
	})
	if err != nil {
		t.Fatalf("filesystem.New() error = %v", err)
	}

	guild := config.Guild{
		GuildID:               "123456789012345678",
		Enabled:               true,
		ForumChannelID:        "234567890123456789",
		NotificationChannelID: "345678901234567890",
		StartProblemNumber:    1,
	}
	cfg := config.Config{
		Timezone: "Asia/Tokyo",
		Retry: config.RetryConfig{
			IntervalMinutes: 5,
			MaxAttempts:     3,
		},
		ProblemCache: config.ProblemCacheConfig{
			RefillThreshold: 1,
		},
		Guilds: []config.Guild{guild},
	}
	writeJSON(t, filepath.Join(root, "config.json"), cfg)

	leetcodeServer := newFakeLeetCodeServer(t)
	discordServer, fakeDiscord := newFakeDiscordServer(t)

	discordClient, err := discord.NewClientWithBaseURL(discordServer.Client(), "token", discordServer.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL() error = %v", err)
	}
	notifier, err := discord.NewNotifier(discordClient, guild.NotificationChannelID)
	if err != nil {
		t.Fatalf("NewNotifier() error = %v", err)
	}

	leetcodeClient := leetcode.NewClientWithEndpoint(leetcodeServer.Client(), leetcodeServer.URL)
	runner, err := job.NewWithOptions(repository, leetcodeClient, discordClient, notifier, job.Options{
		Now: func() time.Time { return time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC) },
		Sleep: func(context.Context, time.Duration) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("job.New() error = %v", err)
	}

	return &integrationEnv{
		repository: repository,
		runner:     runner,
		guild:      guild,
		discord:    fakeDiscord,
	}
}

func newFakeLeetCodeServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"problemsetQuestionList": map[string]any{
					"total": 2,
					"questions": []map[string]any{
						{
							"frontendQuestionId": "1",
							"title":              "Two Sum",
							"titleSlug":          "two-sum",
							"difficulty":         "Easy",
							"isPaidOnly":         false,
						},
						{
							"frontendQuestionId": "2",
							"title":              "Add Two Numbers",
							"titleSlug":          "add-two-numbers",
							"difficulty":         "Medium",
							"isPaidOnly":         false,
						},
					},
				},
			},
		})
	}))

	t.Cleanup(server.Close)
	return server
}

type fakeDiscord struct {
	mu                sync.Mutex
	threadFailures    int
	threadCalls       int
	notificationCalls int
}

func newFakeDiscordServer(t *testing.T) (*httptest.Server, *fakeDiscord) {
	t.Helper()

	state := &fakeDiscord{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/channels/234567890123456789"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "234567890123456789",
				"available_tags": []map[string]any{
					{"id": "easy-1", "name": "Easy"},
					{"id": "medium-1", "name": "Medium"},
					{"id": "hard-1", "name": "Hard"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/channels/234567890123456789/threads":
			state.mu.Lock()
			state.threadCalls++
			shouldFail := state.threadCalls <= state.threadFailures
			state.mu.Unlock()
			if shouldFail {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": fmt.Sprintf("45678901234567890%d", state.threadCalls)})
		case r.Method == http.MethodPost && r.URL.Path == "/channels/345678901234567890/messages":
			state.mu.Lock()
			state.notificationCalls++
			state.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "message-1"})
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))

	t.Cleanup(server.Close)
	return server, state
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}

func mustDate(t *testing.T, value string) state.Date {
	t.Helper()
	date, err := state.ParseDate(value)
	if err != nil {
		t.Fatalf("ParseDate(%q) error = %v", value, err)
	}
	return date
}
