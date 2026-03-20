package problemcache

import (
	"context"
	"fmt"
	"slices"
	"time"
)

type Fetcher interface {
	FetchProblems(context.Context) ([]Problem, error)
}

func NeedsRefill(cache Cache, nextProblemNumber, threshold int) bool {
	return CountFreeProblemsFrom(cache, nextProblemNumber) < threshold
}

func CountFreeProblemsFrom(cache Cache, nextProblemNumber int) int {
	count := 0
	for _, problem := range cache.Problems {
		if problem.ProblemNumber >= nextProblemNumber && !problem.IsPaidOnly {
			count++
		}
	}

	return count
}

func Refresh(ctx context.Context, now time.Time, cache Cache, nextProblemNumber, threshold int, fetcher Fetcher) (Cache, bool, error) {
	if !NeedsRefill(cache, nextProblemNumber, threshold) {
		return cache, false, nil
	}

	problems, err := fetcher.FetchProblems(ctx)
	if err != nil {
		if HasFreeProblemFrom(cache, nextProblemNumber) {
			return cache, false, nil
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

func HasFreeProblemFrom(cache Cache, nextProblemNumber int) bool {
	_, err := SelectNextFree(cache, nextProblemNumber)
	return err == nil
}
