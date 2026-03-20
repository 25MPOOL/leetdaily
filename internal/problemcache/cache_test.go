package problemcache

import (
	"context"
	"testing"
	"time"
)

func TestCacheValidateAndByNumber(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC)
	cache := Cache{
		UpdatedAt: &updatedAt,
		Problems: []Problem{
			{
				ProblemNumber: 1,
				Title:         "Two Sum",
				Slug:          "two-sum",
				Difficulty:    DifficultyEasy,
				IsPaidOnly:    false,
			},
			{
				ProblemNumber: 2,
				Title:         "Add Two Numbers",
				Slug:          "add-two-numbers",
				Difficulty:    DifficultyMedium,
				IsPaidOnly:    false,
			},
		},
	}

	if err := cache.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}

	indexed := cache.ByNumber()
	if len(indexed) != 2 {
		t.Fatalf("len(ByNumber()) = %d, want 2", len(indexed))
	}

	if indexed[2].URL() != "https://leetcode.com/problems/add-two-numbers" {
		t.Fatalf("URL() = %q, want %q", indexed[2].URL(), "https://leetcode.com/problems/add-two-numbers")
	}
}

func TestCacheValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		cache Cache
	}{
		{
			name: "missing updated at",
			cache: Cache{
				Problems: []Problem{
					{
						ProblemNumber: 1,
						Title:         "Two Sum",
						Slug:          "two-sum",
						Difficulty:    DifficultyEasy,
					},
				},
			},
		},
		{
			name: "duplicate problem number",
			cache: Cache{
				UpdatedAt: &updatedAt,
				Problems: []Problem{
					{
						ProblemNumber: 1,
						Title:         "Two Sum",
						Slug:          "two-sum",
						Difficulty:    DifficultyEasy,
					},
					{
						ProblemNumber: 1,
						Title:         "Another Problem",
						Slug:          "another-problem",
						Difficulty:    DifficultyMedium,
					},
				},
			},
		},
		{
			name: "invalid difficulty",
			cache: Cache{
				UpdatedAt: &updatedAt,
				Problems: []Problem{
					{
						ProblemNumber: 3,
						Title:         "Bad Problem",
						Slug:          "bad-problem",
						Difficulty:    "Legendary",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.cache.Validate(); err == nil {
				t.Fatal("Validate() returned nil error, want validation error")
			}
		})
	}
}

func TestSelectNextFreeSkipsPaidProblems(t *testing.T) {
	t.Parallel()

	cache := Cache{
		UpdatedAt: timePointer(time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC)),
		Problems: []Problem{
			{ProblemNumber: 10, Title: "Paid", Slug: "paid", Difficulty: DifficultyEasy, IsPaidOnly: true},
			{ProblemNumber: 11, Title: "Free", Slug: "free", Difficulty: DifficultyMedium, IsPaidOnly: false},
		},
	}

	problem, err := SelectNextFree(cache, 10)
	if err != nil {
		t.Fatalf("SelectNextFree() error = %v", err)
	}

	if problem.ProblemNumber != 11 {
		t.Fatalf("SelectNextFree().ProblemNumber = %d, want 11", problem.ProblemNumber)
	}
}

func TestRefreshBehaviors(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 6, 0, 0, 0, time.UTC)
	current := Cache{
		UpdatedAt: timePointer(time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC)),
		Problems: []Problem{
			{ProblemNumber: 1, Title: "One", Slug: "one", Difficulty: DifficultyEasy},
			{ProblemNumber: 2, Title: "Two", Slug: "two", Difficulty: DifficultyMedium, IsPaidOnly: true},
			{ProblemNumber: 3, Title: "Three", Slug: "three", Difficulty: DifficultyHard},
		},
	}

	t.Run("skip refill when enough free problems remain", func(t *testing.T) {
		t.Parallel()

		cache, refreshed, err := Refresh(context.Background(), now, current, 1, 2, stubFetcher{})
		if err != nil {
			t.Fatalf("Refresh() error = %v", err)
		}
		if refreshed {
			t.Fatal("Refresh() refreshed = true, want false")
		}
		if cache.UpdatedAt != current.UpdatedAt {
			t.Fatal("Refresh() returned different cache, want existing cache")
		}
	})

	t.Run("refill when threshold is not met", func(t *testing.T) {
		t.Parallel()

		cache, refreshed, err := Refresh(context.Background(), now, current, 3, 2, stubFetcher{
			problems: []Problem{
				{ProblemNumber: 3, Title: "Three", Slug: "three", Difficulty: DifficultyHard},
				{ProblemNumber: 4, Title: "Four", Slug: "four", Difficulty: DifficultyEasy},
			},
		})
		if err != nil {
			t.Fatalf("Refresh() error = %v", err)
		}
		if !refreshed {
			t.Fatal("Refresh() refreshed = false, want true")
		}
		if cache.UpdatedAt == nil || !cache.UpdatedAt.Equal(now) {
			t.Fatalf("Refresh().UpdatedAt = %v, want %v", cache.UpdatedAt, now)
		}
		if len(cache.Problems) != 2 {
			t.Fatalf("len(Refresh().Problems) = %d, want 2", len(cache.Problems))
		}
	})

	t.Run("keep existing cache when refill fails but free problems remain", func(t *testing.T) {
		t.Parallel()

		cache, refreshed, err := Refresh(context.Background(), now, current, 1, 5, stubFetcher{err: context.DeadlineExceeded})
		if err != nil {
			t.Fatalf("Refresh() error = %v, want nil", err)
		}
		if refreshed {
			t.Fatal("Refresh() refreshed = true, want false")
		}
		if len(cache.Problems) != len(current.Problems) {
			t.Fatalf("len(Refresh().Problems) = %d, want %d", len(cache.Problems), len(current.Problems))
		}
	})

	t.Run("fail when refill fails and no free problem remains", func(t *testing.T) {
		t.Parallel()

		_, _, err := Refresh(context.Background(), now, current, 4, 1, stubFetcher{err: context.DeadlineExceeded})
		if err == nil {
			t.Fatal("Refresh() error = nil, want refill error")
		}
	})
}

type stubFetcher struct {
	problems []Problem
	err      error
}

func (s stubFetcher) FetchProblems(context.Context) ([]Problem, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.problems, nil
}

func timePointer(value time.Time) *time.Time {
	return &value
}
