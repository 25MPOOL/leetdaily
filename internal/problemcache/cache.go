package problemcache

import (
	"fmt"
	"net/url"
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
	Category      string     `json:"category"`
	IsPaidOnly    bool       `json:"is_paid_only"`
	NeetCodeURL   string     `json:"neetcode_url"`
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

	if p.IsPaidOnly && strings.TrimSpace(p.NeetCodeURL) == "" {
		return fmt.Errorf("neetcode_url must not be empty for paid-only problems")
	}

	return nil
}

// CategoryProgress returns (position, total) for the given problem within its category,
// counting only problems at or below maxDifficulty.
// position is 1-indexed. If the category is empty, returns (0, 0).
func CategoryProgress(cache Cache, problem Problem, maxDifficulty Difficulty) (position int, total int) {
	if problem.Category == "" {
		return 0, 0
	}
	maxRank := difficultyRank(maxDifficulty)
	for _, p := range cache.Problems {
		if p.Category != problem.Category {
			continue
		}
		if difficultyRank(p.Difficulty) > maxRank {
			continue
		}
		total++
		if p.ProblemNumber <= problem.ProblemNumber {
			position++
		}
	}
	return position, total
}

func (p Problem) URL() string {
	return fmt.Sprintf("https://leetcode.com/problems/%s", p.Slug)
}

// NeetCodeSolutionURL returns the NeetCode solution page URL for this problem.
// Returns an empty string if NeetCodeURL is not set.
func (p Problem) NeetCodeSolutionURL() string {
	if p.NeetCodeURL == "" {
		return ""
	}
	u, err := url.Parse(p.NeetCodeURL)
	if err != nil {
		return ""
	}
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/") + "/solution"
}
