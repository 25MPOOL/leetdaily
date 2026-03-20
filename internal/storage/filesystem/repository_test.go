package filesystem

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/problemcache"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
)

func TestRepositoryLoadConfigAndRoundTripStateAndCache(t *testing.T) {
	t.Parallel()

	repository, paths := newTestRepository(t)

	cfg := config.Config{
		Timezone: "Asia/Tokyo",
		Retry: config.RetryConfig{
			IntervalMinutes: 5,
			MaxAttempts:     3,
		},
		ProblemCache: config.ProblemCacheConfig{
			RefillThreshold: 30,
		},
		Guilds: []config.Guild{
			{
				GuildID:               "123456789012345678",
				Enabled:               true,
				ForumChannelID:        "234567890123456789",
				NotificationChannelID: "345678901234567890",
				StartProblemNumber:    1,
			},
		},
	}
	writeJSON(t, paths.ConfigPath, cfg)

	loadedConfig, err := repository.LoadConfig(context.Background())
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loadedConfig.Timezone != "Asia/Tokyo" {
		t.Fatalf("LoadConfig().Timezone = %q, want %q", loadedConfig.Timezone, "Asia/Tokyo")
	}

	targetDate, err := state.ParseDate("2026-03-20")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}

	lastPostedAt := time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC)
	postingStartedAt := time.Date(2026, 3, 20, 5, 0, 3, 0, time.UTC)
	lastProblem := 137
	threadID := "456789012345678901"
	problemNumber := 137
	lastError := "missing permissions"

	wantState := state.State{
		GuildStates: map[string]state.GuildState{
			"123456789012345678": {
				NextProblemNumber:       138,
				LastPostedProblemNumber: &lastProblem,
				LastPostedAt:            &lastPostedAt,
				LastPostedThreadID:      &threadID,
				Job: state.JobState{
					TargetDate:       &targetDate,
					Status:           state.JobStatusPosted,
					ProblemNumber:    &problemNumber,
					RetryCount:       1,
					LastError:        &lastError,
					PostingStartedAt: &postingStartedAt,
				},
			},
		},
	}

	if err := repository.SaveState(context.Background(), wantState); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	gotState, err := repository.LoadState(context.Background())
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}

	if len(gotState.GuildStates) != 1 {
		t.Fatalf("len(LoadState().GuildStates) = %d, want 1", len(gotState.GuildStates))
	}

	if gotState.GuildStates["123456789012345678"].NextProblemNumber != 138 {
		t.Fatalf("LoadState().GuildStates[next].NextProblemNumber = %d, want 138", gotState.GuildStates["123456789012345678"].NextProblemNumber)
	}

	updatedAt := time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC)
	wantCache := problemcache.Cache{
		UpdatedAt: &updatedAt,
		Problems: []problemcache.Problem{
			{
				ProblemNumber: 1,
				Title:         "Two Sum",
				Slug:          "two-sum",
				Difficulty:    problemcache.DifficultyEasy,
			},
		},
	}

	if err := repository.SaveProblemCache(context.Background(), wantCache); err != nil {
		t.Fatalf("SaveProblemCache() error = %v", err)
	}

	gotCache, err := repository.LoadProblemCache(context.Background())
	if err != nil {
		t.Fatalf("LoadProblemCache() error = %v", err)
	}

	if len(gotCache.Problems) != 1 {
		t.Fatalf("len(LoadProblemCache().Problems) = %d, want 1", len(gotCache.Problems))
	}

	if gotCache.Problems[0].Slug != "two-sum" {
		t.Fatalf("LoadProblemCache().Problems[0].Slug = %q, want %q", gotCache.Problems[0].Slug, "two-sum")
	}
}

func TestRepositoryLoadReturnsNotFoundForMissingFiles(t *testing.T) {
	t.Parallel()

	repository, _ := newTestRepository(t)

	_, err := repository.LoadConfig(context.Background())
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("LoadConfig() error = %v, want ErrNotFound", err)
	}

	_, err = repository.LoadState(context.Background())
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("LoadState() error = %v, want ErrNotFound", err)
	}

	_, err = repository.LoadProblemCache(context.Background())
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("LoadProblemCache() error = %v, want ErrNotFound", err)
	}
}

func TestRepositorySaveUsesAtomicRenameWithoutLeavingTempFiles(t *testing.T) {
	t.Parallel()

	repository, paths := newTestRepository(t)

	if err := repository.SaveState(context.Background(), state.New()); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	data, err := os.ReadFile(paths.StatePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", paths.StatePath, err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("saved state JSON is invalid: %v", err)
	}

	matches, err := filepath.Glob(paths.StatePath + ".*.tmp")
	if err != nil {
		t.Fatalf("filepath.Glob() error = %v", err)
	}

	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}

func newTestRepository(t *testing.T) (*Repository, storage.Paths) {
	t.Helper()

	root := t.TempDir()
	paths := storage.Paths{
		ConfigPath:   filepath.Join(root, "config.json"),
		StatePath:    filepath.Join(root, "state.json"),
		ProblemsPath: filepath.Join(root, "problems.json"),
	}

	repository, err := New(paths)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return repository, paths
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
