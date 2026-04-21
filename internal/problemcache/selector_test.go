package problemcache

import (
	"testing"
	"time"
)

func TestSelectNext(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cache := Cache{
		UpdatedAt: &updatedAt,
		Problems: []Problem{
			{ProblemNumber: 1, LeetCodeNumber: 1, Title: "Two Sum", Slug: "two-sum", Difficulty: DifficultyEasy},
			{ProblemNumber: 3, LeetCodeNumber: 3, Title: "Longest Substring", Slug: "longest-substring-without-repeating-characters", Difficulty: DifficultyMedium},
			{ProblemNumber: 5, LeetCodeNumber: 5, Title: "Longest Palindrome", Slug: "longest-palindromic-substring", Difficulty: DifficultyMedium},
		},
	}

	cases := []struct {
		name              string
		nextProblemNumber int
		wantNumber        int
		wantErr           bool
	}{
		{"first problem", 1, 1, false},
		{"skips gap", 2, 3, false},
		{"exact match on gap", 3, 3, false},
		{"last problem", 5, 5, false},
		{"beyond last", 6, 0, true},
		{"zero nextProblemNumber", 0, 0, true},
		{"negative nextProblemNumber", -1, 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := SelectNext(cache, tc.nextProblemNumber)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("SelectNext(%d) got nil, want error", tc.nextProblemNumber)
				}
				return
			}
			if err != nil {
				t.Fatalf("SelectNext(%d) error = %v", tc.nextProblemNumber, err)
			}
			if got.ProblemNumber != tc.wantNumber {
				t.Fatalf("SelectNext(%d).ProblemNumber = %d, want %d", tc.nextProblemNumber, got.ProblemNumber, tc.wantNumber)
			}
		})
	}
}

func TestSelectNextEmptyCache(t *testing.T) {
	t.Parallel()

	_, err := SelectNext(Cache{}, 1)
	if err == nil {
		t.Fatal("SelectNext with empty cache: got nil, want error")
	}
}

func TestSelectNextAtMost(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cache := Cache{
		UpdatedAt: &updatedAt,
		Problems: []Problem{
			{ProblemNumber: 1, LeetCodeNumber: 1, Title: "Two Sum", Slug: "two-sum", Difficulty: DifficultyEasy},
			{ProblemNumber: 2, LeetCodeNumber: 2, Title: "Add Two Numbers", Slug: "add-two-numbers", Difficulty: DifficultyMedium},
			{ProblemNumber: 3, LeetCodeNumber: 4, Title: "Median of Two Arrays", Slug: "median-of-two-sorted-arrays", Difficulty: DifficultyHard},
		},
	}

	cases := []struct {
		name          string
		nextNumber    int
		maxDifficulty Difficulty
		wantNumber    int
		wantErr       bool
	}{
		{"easy filter returns easy", 1, DifficultyEasy, 1, false},
		{"medium filter returns easy", 1, DifficultyMedium, 1, false},
		{"medium filter returns medium", 2, DifficultyMedium, 2, false},
		{"medium filter skips hard", 3, DifficultyMedium, 0, true},
		{"hard filter returns hard", 3, DifficultyHard, 3, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := SelectNextAtMost(cache, tc.nextNumber, tc.maxDifficulty)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("SelectNextAtMost(%d, %s) got nil, want error", tc.nextNumber, tc.maxDifficulty)
				}
				return
			}
			if err != nil {
				t.Fatalf("SelectNextAtMost(%d, %s) error = %v", tc.nextNumber, tc.maxDifficulty, err)
			}
			if got.ProblemNumber != tc.wantNumber {
				t.Fatalf("SelectNextAtMost(%d, %s).ProblemNumber = %d, want %d", tc.nextNumber, tc.maxDifficulty, got.ProblemNumber, tc.wantNumber)
			}
		})
	}
}
