package problemcache

import (
	"context"
	"errors"
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
				ProblemNumber:  1,
				LeetCodeNumber: 1,
				Title:          "Two Sum",
				Slug:           "two-sum",
				Difficulty:     DifficultyEasy,
			},
			{
				ProblemNumber:  2,
				LeetCodeNumber: 2,
				Title:          "Add Two Numbers",
				Slug:           "add-two-numbers",
				Difficulty:     DifficultyMedium,
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
						ProblemNumber:  1,
						LeetCodeNumber: 1,
						Title:          "Two Sum",
						Slug:           "two-sum",
						Difficulty:     DifficultyEasy,
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
						ProblemNumber:  1,
						LeetCodeNumber: 1,
						Title:          "Two Sum",
						Slug:           "two-sum",
						Difficulty:     DifficultyEasy,
					},
					{
						ProblemNumber:  1,
						LeetCodeNumber: 100,
						Title:          "Another Problem",
						Slug:           "another-problem",
						Difficulty:     DifficultyMedium,
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
						ProblemNumber:  3,
						LeetCodeNumber: 3,
						Title:          "Bad Problem",
						Slug:           "bad-problem",
						Difficulty:     "Legendary",
					},
				},
			},
		},
		{
			name: "zero leetcode number",
			cache: Cache{
				UpdatedAt: &updatedAt,
				Problems: []Problem{
					{
						ProblemNumber:  1,
						LeetCodeNumber: 0,
						Title:          "Two Sum",
						Slug:           "two-sum",
						Difficulty:     DifficultyEasy,
					},
				},
			},
		},
		{
			name: "negative leetcode number",
			cache: Cache{
				UpdatedAt: &updatedAt,
				Problems: []Problem{
					{
						ProblemNumber:  1,
						LeetCodeNumber: -1,
						Title:          "Two Sum",
						Slug:           "two-sum",
						Difficulty:     DifficultyEasy,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.cache.Validate(); err == nil {
				t.Fatal("Validate() returned nil error, want validation error")
			}
		})
	}
}

func TestSelectNextReturnsNextProblem(t *testing.T) {
	t.Parallel()

	cache := Cache{
		UpdatedAt: timePointer(time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC)),
		Problems: []Problem{
			{ProblemNumber: 10, LeetCodeNumber: 10, Title: "First", Slug: "first", Difficulty: DifficultyEasy},
			{ProblemNumber: 11, LeetCodeNumber: 11, Title: "Second", Slug: "second", Difficulty: DifficultyMedium},
		},
	}

	problem, err := SelectNext(cache, 10)
	if err != nil {
		t.Fatalf("SelectNext() error = %v", err)
	}

	if problem.ProblemNumber != 10 {
		t.Fatalf("SelectNext().ProblemNumber = %d, want 10", problem.ProblemNumber)
	}
}

func TestSelectNextAtMostFiltersHard(t *testing.T) {
	t.Parallel()

	cache := Cache{
		UpdatedAt: timePointer(time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC)),
		Problems: []Problem{
			{ProblemNumber: 1, LeetCodeNumber: 1, Title: "Easy", Slug: "easy", Difficulty: DifficultyEasy},
			{ProblemNumber: 2, LeetCodeNumber: 2, Title: "Medium", Slug: "medium", Difficulty: DifficultyMedium},
			{ProblemNumber: 3, LeetCodeNumber: 3, Title: "Hard", Slug: "hard", Difficulty: DifficultyHard},
		},
	}

	t.Run("easy passes", func(t *testing.T) {
		t.Parallel()
		p, err := SelectNextAtMost(cache, 1, DifficultyMedium)
		if err != nil {
			t.Fatalf("SelectNextAtMost() error = %v", err)
		}
		if p.ProblemNumber != 1 {
			t.Fatalf("ProblemNumber = %d, want 1", p.ProblemNumber)
		}
	})

	t.Run("medium passes", func(t *testing.T) {
		t.Parallel()
		p, err := SelectNextAtMost(cache, 2, DifficultyMedium)
		if err != nil {
			t.Fatalf("SelectNextAtMost() error = %v", err)
		}
		if p.ProblemNumber != 2 {
			t.Fatalf("ProblemNumber = %d, want 2", p.ProblemNumber)
		}
	})

	t.Run("hard is skipped", func(t *testing.T) {
		t.Parallel()
		_, err := SelectNextAtMost(cache, 3, DifficultyMedium)
		if err == nil {
			t.Fatal("SelectNextAtMost() returned nil error, want error for hard-only cache")
		}
	})

	t.Run("hard skipped and next easy selected", func(t *testing.T) {
		t.Parallel()
		hardThenEasy := Cache{
			UpdatedAt: timePointer(time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC)),
			Problems: []Problem{
				{ProblemNumber: 5, LeetCodeNumber: 5, Title: "Hard", Slug: "hard", Difficulty: DifficultyHard},
				{ProblemNumber: 6, LeetCodeNumber: 6, Title: "Easy", Slug: "easy2", Difficulty: DifficultyEasy},
			},
		}
		p, err := SelectNextAtMost(hardThenEasy, 5, DifficultyMedium)
		if err != nil {
			t.Fatalf("SelectNextAtMost() error = %v", err)
		}
		if p.ProblemNumber != 6 {
			t.Fatalf("ProblemNumber = %d, want 6 (hard skipped)", p.ProblemNumber)
		}
	})

}

func TestRefreshBehaviors(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 6, 0, 0, 0, time.UTC)
	current := Cache{
		UpdatedAt: timePointer(time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC)),
		Problems: []Problem{
			{ProblemNumber: 1, LeetCodeNumber: 1, Title: "One", Slug: "one", Difficulty: DifficultyEasy},
			{ProblemNumber: 2, LeetCodeNumber: 2, Title: "Two", Slug: "two", Difficulty: DifficultyMedium},
			{ProblemNumber: 3, LeetCodeNumber: 3, Title: "Three", Slug: "three", Difficulty: DifficultyHard},
		},
	}

	t.Run("skip refill when enough free problems remain", func(t *testing.T) {
		t.Parallel()

		cache, refreshed, err := Refresh(context.Background(), now, current, 1, 2, DifficultyMedium, stubFetcher{})
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

		cache, refreshed, err := Refresh(context.Background(), now, current, 3, 2, DifficultyMedium, stubFetcher{
			problems: []Problem{
				{ProblemNumber: 3, LeetCodeNumber: 3, Title: "Three", Slug: "three", Difficulty: DifficultyHard},
				{ProblemNumber: 4, LeetCodeNumber: 4, Title: "Four", Slug: "four", Difficulty: DifficultyEasy},
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

		cache, refreshed, err := Refresh(context.Background(), now, current, 1, 5, DifficultyMedium, stubFetcher{err: context.DeadlineExceeded})
		if !errors.Is(err, ErrRefillUsedStaleCache) {
			t.Fatalf("Refresh() error = %v, want ErrRefillUsedStaleCache", err)
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

		_, _, err := Refresh(context.Background(), now, current, 4, 1, DifficultyMedium, stubFetcher{err: context.DeadlineExceeded})
		if err == nil {
			t.Fatal("Refresh() error = nil, want refill error")
		}
	})
}

func TestCategoryProgress(t *testing.T) {
	t.Parallel()

	updatedAt := timePointer(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	cache := Cache{
		UpdatedAt: updatedAt,
		Problems: []Problem{
			{ProblemNumber: 1, LeetCodeNumber: 1, Title: "A", Slug: "a", Difficulty: DifficultyEasy, Category: "Arrays & Hashing"},
			{ProblemNumber: 2, LeetCodeNumber: 2, Title: "B", Slug: "b", Difficulty: DifficultyMedium, Category: "Arrays & Hashing"},
			{ProblemNumber: 3, LeetCodeNumber: 3, Title: "C", Slug: "c", Difficulty: DifficultyHard, Category: "Arrays & Hashing"},
			{ProblemNumber: 4, LeetCodeNumber: 4, Title: "D", Slug: "d", Difficulty: DifficultyEasy, Category: "Two Pointers"},
			{ProblemNumber: 5, LeetCodeNumber: 5, Title: "E", Slug: "e", Difficulty: DifficultyMedium, Category: "Two Pointers"},
		},
	}

	cases := []struct {
		name          string
		problem       Problem
		maxDifficulty Difficulty
		wantPos       int
		wantTotal     int
	}{
		{
			name:          "first in category",
			problem:       cache.Problems[0],
			maxDifficulty: DifficultyMedium,
			wantPos:       1,
			wantTotal:     2, // Hard excluded
		},
		{
			name:          "second in category",
			problem:       cache.Problems[1],
			maxDifficulty: DifficultyMedium,
			wantPos:       2,
			wantTotal:     2, // Hard excluded
		},
		{
			name:          "hard excluded from count",
			problem:       cache.Problems[1],
			maxDifficulty: DifficultyHard,
			wantPos:       2,
			wantTotal:     3, // Hard included
		},
		{
			name:          "different category",
			problem:       cache.Problems[3],
			maxDifficulty: DifficultyMedium,
			wantPos:       1,
			wantTotal:     2,
		},
		{
			name:          "empty category returns zero",
			problem:       Problem{ProblemNumber: 1, LeetCodeNumber: 1, Title: "X", Slug: "x", Difficulty: DifficultyEasy, Category: ""},
			maxDifficulty: DifficultyMedium,
			wantPos:       0,
			wantTotal:     0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pos, total := CategoryProgress(cache, tc.problem, tc.maxDifficulty)
			if pos != tc.wantPos || total != tc.wantTotal {
				t.Fatalf("CategoryProgress() = (%d, %d), want (%d, %d)", pos, total, tc.wantPos, tc.wantTotal)
			}
		})
	}
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
