package problemcache

import (
	"fmt"
	"strings"
	"time"
)

type Difficulty string

const (
	DifficultyEasy   Difficulty = "Easy"
	DifficultyMedium Difficulty = "Medium"
	DifficultyHard   Difficulty = "Hard"
)

type Cache struct {
	UpdatedAt *time.Time `json:"updated_at"`
	Problems  []Problem  `json:"problems"`
}

type Problem struct {
	ProblemNumber int        `json:"problem_number"`
	Title         string     `json:"title"`
	Slug          string     `json:"slug"`
	Difficulty    Difficulty `json:"difficulty"`
	IsPaidOnly    bool       `json:"is_paid_only"`
}

func (c Cache) Validate() error {
	if c.UpdatedAt != nil && c.UpdatedAt.IsZero() {
		return fmt.Errorf("updated_at must not be zero")
	}

	if len(c.Problems) > 0 && c.UpdatedAt == nil {
		return fmt.Errorf("updated_at is required when problems are present")
	}

	seenNumbers := make(map[int]struct{}, len(c.Problems))
	seenSlugs := make(map[string]struct{}, len(c.Problems))
	for i, problem := range c.Problems {
		if err := problem.Validate(); err != nil {
			return fmt.Errorf("problems[%d]: %w", i, err)
		}

		if _, ok := seenNumbers[problem.ProblemNumber]; ok {
			return fmt.Errorf("problems[%d]: duplicate problem_number %d", i, problem.ProblemNumber)
		}
		seenNumbers[problem.ProblemNumber] = struct{}{}

		if _, ok := seenSlugs[problem.Slug]; ok {
			return fmt.Errorf("problems[%d]: duplicate slug %q", i, problem.Slug)
		}
		seenSlugs[problem.Slug] = struct{}{}
	}

	return nil
}

func (c Cache) ByNumber() map[int]Problem {
	indexed := make(map[int]Problem, len(c.Problems))
	for _, problem := range c.Problems {
		indexed[problem.ProblemNumber] = problem
	}

	return indexed
}

func (p Problem) Validate() error {
	if p.ProblemNumber < 1 {
		return fmt.Errorf("problem_number must be greater than 0: %d", p.ProblemNumber)
	}

	if strings.TrimSpace(p.Title) == "" {
		return fmt.Errorf("title must not be empty")
	}

	if strings.TrimSpace(p.Slug) == "" {
		return fmt.Errorf("slug must not be empty")
	}

	switch p.Difficulty {
	case DifficultyEasy, DifficultyMedium, DifficultyHard:
	default:
		return fmt.Errorf("difficulty must be one of Easy/Medium/Hard: %q", p.Difficulty)
	}

	return nil
}

func (p Problem) URL() string {
	return fmt.Sprintf("https://leetcode.com/problems/%s", p.Slug)
}
