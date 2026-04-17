package problemcache

import (
	"fmt"
	"math"
)

func SelectNext(cache Cache, nextProblemNumber int) (Problem, error) {
	return selectNext(cache, nextProblemNumber, nil)
}

func SelectNextAtMost(cache Cache, nextProblemNumber int, maxDifficulty Difficulty) (Problem, error) {
	return selectNext(cache, nextProblemNumber, &maxDifficulty)
}

func selectNext(cache Cache, nextProblemNumber int, maxDifficulty *Difficulty) (Problem, error) {
	if nextProblemNumber < 1 {
		return Problem{}, fmt.Errorf("next problem number must be greater than 0: %d", nextProblemNumber)
	}

	indexed := cache.ByNumber()
	for number := nextProblemNumber; number <= maxProblemNumber(indexed); number++ {
		problem, ok := indexed[number]
		if !ok {
			continue
		}
		if maxDifficulty != nil && difficultyRank(problem.Difficulty) > difficultyRank(*maxDifficulty) {
			continue
		}
		if problem.IsPaidOnly {
			continue
		}
		return problem, nil
	}

	return Problem{}, fmt.Errorf("no problem found at or after #%d", nextProblemNumber)
}

func difficultyRank(d Difficulty) int {
	switch d {
	case DifficultyEasy:
		return 1
	case DifficultyMedium:
		return 2
	case DifficultyHard:
		return 3
	default:
		return math.MaxInt
	}
}

func maxProblemNumber(indexed map[int]Problem) int {
	max := 0
	for number := range indexed {
		if number > max {
			max = number
		}
	}

	return max
}
