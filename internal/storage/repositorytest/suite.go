package repositorytest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/problemcache"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
)

type Harness struct {
	Repository storage.Repository
	SeedConfig func(testing.TB, config.Config)
}

func RunRepositorySuite(t *testing.T, name string, newHarness func(*testing.T) Harness) {
	t.Helper()

	t.Run(name+"/round_trip", func(t *testing.T) {
		t.Parallel()

		harness := newHarness(t)
		repository := harness.Repository

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
		harness.SeedConfig(t, cfg)

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

		stateVersion, err := repository.SaveState(context.Background(), wantState, storage.Version{})
		if err != nil {
			t.Fatalf("SaveState() error = %v", err)
		}

		gotState, loadedStateVersion, err := repository.LoadState(context.Background())
		if err != nil {
			t.Fatalf("LoadState() error = %v", err)
		}

		if len(gotState.GuildStates) != 1 {
			t.Fatalf("len(LoadState().GuildStates) = %d, want 1", len(gotState.GuildStates))
		}

		if gotState.GuildStates["123456789012345678"].NextProblemNumber != 138 {
			t.Fatalf("LoadState().GuildStates[next].NextProblemNumber = %d, want 138", gotState.GuildStates["123456789012345678"].NextProblemNumber)
		}

		if loadedStateVersion != stateVersion {
			t.Fatalf("LoadState() version = %#v, want %#v", loadedStateVersion, stateVersion)
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

		cacheVersion, err := repository.SaveProblemCache(context.Background(), wantCache, storage.Version{})
		if err != nil {
			t.Fatalf("SaveProblemCache() error = %v", err)
		}

		gotCache, loadedCacheVersion, err := repository.LoadProblemCache(context.Background())
		if err != nil {
			t.Fatalf("LoadProblemCache() error = %v", err)
		}

		if len(gotCache.Problems) != 1 {
			t.Fatalf("len(LoadProblemCache().Problems) = %d, want 1", len(gotCache.Problems))
		}

		if gotCache.Problems[0].Slug != "two-sum" {
			t.Fatalf("LoadProblemCache().Problems[0].Slug = %q, want %q", gotCache.Problems[0].Slug, "two-sum")
		}

		if loadedCacheVersion != cacheVersion {
			t.Fatalf("LoadProblemCache() version = %#v, want %#v", loadedCacheVersion, cacheVersion)
		}
	})

	t.Run(name+"/not_found", func(t *testing.T) {
		t.Parallel()

		repository := newHarness(t).Repository

		_, err := repository.LoadConfig(context.Background())
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("LoadConfig() error = %v, want ErrNotFound", err)
		}

		_, _, err = repository.LoadState(context.Background())
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("LoadState() error = %v, want ErrNotFound", err)
		}

		_, _, err = repository.LoadProblemCache(context.Background())
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("LoadProblemCache() error = %v, want ErrNotFound", err)
		}
	})

	t.Run(name+"/state_conflict", func(t *testing.T) {
		t.Parallel()

		repository := newHarness(t).Repository

		initial := state.New()
		version, err := repository.SaveState(context.Background(), initial, storage.Version{})
		if err != nil {
			t.Fatalf("SaveState(initial) error = %v", err)
		}

		current, loadedVersion, err := repository.LoadState(context.Background())
		if err != nil {
			t.Fatalf("LoadState() error = %v", err)
		}

		current.GuildStates["1"] = state.GuildState{
			NextProblemNumber: 10,
			Job: state.JobState{
				Status: state.JobStatusIdle,
			},
		}
		newVersion, err := repository.SaveState(context.Background(), current, loadedVersion)
		if err != nil {
			t.Fatalf("SaveState(updated) error = %v", err)
		}

		if newVersion == version {
			t.Fatalf("SaveState(updated) version = %#v, want different from %#v", newVersion, version)
		}

		current.GuildStates["2"] = state.GuildState{
			NextProblemNumber: 20,
			Job: state.JobState{
				Status: state.JobStatusIdle,
			},
		}
		_, err = repository.SaveState(context.Background(), current, loadedVersion)
		if !errors.Is(err, storage.ErrConflict) {
			t.Fatalf("SaveState(stale version) error = %v, want ErrConflict", err)
		}
	})

	t.Run(name+"/problem_cache_conflict", func(t *testing.T) {
		t.Parallel()

		repository := newHarness(t).Repository

		initialUpdatedAt := time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC)
		initial := problemcache.Cache{
			UpdatedAt: &initialUpdatedAt,
			Problems:  []problemcache.Problem{},
		}
		version, err := repository.SaveProblemCache(context.Background(), initial, storage.Version{})
		if err != nil {
			t.Fatalf("SaveProblemCache(initial) error = %v", err)
		}

		current, loadedVersion, err := repository.LoadProblemCache(context.Background())
		if err != nil {
			t.Fatalf("LoadProblemCache() error = %v", err)
		}

		current.Problems = append(current.Problems, problemcache.Problem{
			ProblemNumber: 1,
			Title:         "Two Sum",
			Slug:          "two-sum",
			Difficulty:    problemcache.DifficultyEasy,
		})
		newVersion, err := repository.SaveProblemCache(context.Background(), current, loadedVersion)
		if err != nil {
			t.Fatalf("SaveProblemCache(updated) error = %v", err)
		}

		if newVersion == version {
			t.Fatalf("SaveProblemCache(updated) version = %#v, want different from %#v", newVersion, version)
		}

		current.Problems = append(current.Problems, problemcache.Problem{
			ProblemNumber: 2,
			Title:         "Add Two Numbers",
			Slug:          "add-two-numbers",
			Difficulty:    problemcache.DifficultyMedium,
		})
		_, err = repository.SaveProblemCache(context.Background(), current, loadedVersion)
		if !errors.Is(err, storage.ErrConflict) {
			t.Fatalf("SaveProblemCache(stale version) error = %v, want ErrConflict", err)
		}
	})
}
