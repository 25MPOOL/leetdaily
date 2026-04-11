package problemcache

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

var ErrRefillUsedStaleCache = errors.New("problem cache refill failed; using stale cache")

type Fetcher interface {
	FetchProblems(context.Context) ([]Problem, error)
}

func NeedsRefill(cache Cache, nextProblemNumber, threshold int, maxDifficulty Difficulty) bool {
	return CountProblemsFrom(cache, nextProblemNumber, maxDifficulty) < threshold
}

func CountProblemsFrom(cache Cache, nextProblemNumber int, maxDifficulty Difficulty) int {
	count := 0
	for _, problem := range cache.Problems {
		if problem.ProblemNumber >= nextProblemNumber && difficultyRank(problem.Difficulty) <= difficultyRank(maxDifficulty) {
			count++
		}
	}

	return count
}

func Refresh(ctx context.Context, now time.Time, cache Cache, nextProblemNumber, threshold int, maxDifficulty Difficulty, fetcher Fetcher) (Cache, bool, error) {
	if !NeedsRefill(cache, nextProblemNumber, threshold, maxDifficulty) {
		return cache, false, nil
	}

	problems, err := fetcher.FetchProblems(ctx)
	if err != nil {
		if HasProblemFrom(cache, nextProblemNumber, maxDifficulty) {
			return cache, false, fmt.Errorf("%w: %w", ErrRefillUsedStaleCache, err)
		}
		return Cache{}, false, fmt.Errorf("refill problem cache: %w", err)
	}

	refreshed := Cache{
		UpdatedAt: &now,
		Problems:  slices.Clone(problems),
	}
	if refreshed.Problems == nil {
		refreshed.Problems = []Problem{}
	}

	if err := refreshed.Validate(); err != nil {
		return Cache{}, false, fmt.Errorf("validate refilled problem cache: %w", err)
	}

	return refreshed, true, nil
}

func HasProblemFrom(cache Cache, nextProblemNumber int, maxDifficulty Difficulty) bool {
	_, err := SelectNextAtMost(cache, nextProblemNumber, maxDifficulty)
	return err == nil
}
