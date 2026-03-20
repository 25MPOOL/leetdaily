package problemcache

import (
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
