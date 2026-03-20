package problemcache

import "fmt"

func SelectNextFree(cache Cache, nextProblemNumber int) (Problem, error) {
	if nextProblemNumber < 1 {
		return Problem{}, fmt.Errorf("next problem number must be greater than 0: %d", nextProblemNumber)
	}

	indexed := cache.ByNumber()
	for number := nextProblemNumber; number <= maxProblemNumber(indexed); number++ {
		problem, ok := indexed[number]
		if !ok || problem.IsPaidOnly {
			continue
		}
		return problem, nil
	}

	return Problem{}, fmt.Errorf("no free problem found at or after #%d", nextProblemNumber)
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
