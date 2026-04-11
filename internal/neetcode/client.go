package neetcode

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nkoji21/leetdaily/internal/problemcache"
)

// ProblemSource reads the NeetCode 150 problem list from a local JSON file.
type ProblemSource struct {
	path string
}

func NewProblemSource(path string) *ProblemSource {
	return &ProblemSource{path: path}
}

type problemEntry struct {
	ProblemNumber int                     `json:"problem_number"`
	Title         string                  `json:"title"`
	Slug          string                  `json:"slug"`
	Difficulty    problemcache.Difficulty `json:"difficulty"`
	Category      string                  `json:"category"`
	IsPaidOnly    bool                    `json:"is_paid_only"`
	NeetCodeURL   string                  `json:"neetcode_url"`
}

type problemList struct {
	Problems []problemEntry `json:"problems"`
}

func (s *ProblemSource) FetchProblems(_ context.Context) ([]problemcache.Problem, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("read NeetCode 150 problem list %q: %w", s.path, err)
	}

	var list problemList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("decode NeetCode 150 problem list %q: %w", s.path, err)
	}

	problems := make([]problemcache.Problem, 0, len(list.Problems))
	for i, entry := range list.Problems {
		p := problemcache.Problem{
			ProblemNumber: entry.ProblemNumber,
			Title:         entry.Title,
			Slug:          entry.Slug,
			Difficulty:    entry.Difficulty,
			IsPaidOnly:    entry.IsPaidOnly,
			NeetCodeURL:   entry.NeetCodeURL,
		}
		if err := p.Validate(); err != nil {
			return nil, fmt.Errorf("problems[%d]: %w", i, err)
		}
		problems = append(problems, p)
	}

	return problems, nil
}
