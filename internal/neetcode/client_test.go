package neetcode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
}

func TestClientFetchProblemsSuccess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "problems.json")
	writeJSON(t, path, map[string]any{
		"problems": []map[string]any{
			{"problem_number": 1, "title": "Two Sum", "slug": "two-sum", "difficulty": "Easy", "category": "Arrays & Hashing"},
			{"problem_number": 2, "title": "Add Two Numbers", "slug": "add-two-numbers", "difficulty": "Medium", "category": "Linked List"},
			{"problem_number": 6, "title": "Encode and Decode Strings", "slug": "encode-and-decode-strings", "difficulty": "Medium", "category": "Arrays & Hashing", "is_paid_only": true, "neetcode_url": "https://neetcode.io/problems/string-encode-and-decode"},
		},
	})

	client := NewProblemSource(path)
	problems, err := client.FetchProblems(context.Background())
	if err != nil {
		t.Fatalf("FetchProblems() error = %v", err)
	}

	if len(problems) != 3 {
		t.Fatalf("len(FetchProblems()) = %d, want 3", len(problems))
	}

	if problems[0].ProblemNumber != 1 || problems[0].Slug != "two-sum" {
		t.Fatalf("problems[0] = %+v, want Two Sum", problems[0])
	}

	if !problems[2].IsPaidOnly || problems[2].NeetCodeURL == "" {
		t.Fatalf("problems[2] should be paid-only with NeetCodeURL set, got %+v", problems[2])
	}

	if got := problems[2].URL(); got != "https://neetcode.io/problems/string-encode-and-decode" {
		t.Fatalf("URL() = %q, want NeetCode URL", got)
	}
}

func TestClientFetchProblemsFileNotFound(t *testing.T) {
	t.Parallel()

	client := NewProblemSource("/nonexistent/path/problems.json")
	if _, err := client.FetchProblems(context.Background()); err == nil {
		t.Fatal("FetchProblems() returned nil error, want file-not-found error")
	}
}

func TestClientFetchProblemsMalformedJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "problems.json")
	if err := os.WriteFile(path, []byte("not json at all"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := NewProblemSource(path)
	if _, err := client.FetchProblems(context.Background()); err == nil {
		t.Fatal("FetchProblems() returned nil error, want JSON decode error")
	}
}

func TestClientFetchProblemsValidationFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "problems.json")
	writeJSON(t, path, map[string]any{
		"problems": []map[string]any{
			// invalid difficulty
			{"problem_number": 1, "title": "Bad Problem", "slug": "bad-problem", "difficulty": "Legendary", "category": "Unknown"},
		},
	})

	client := NewProblemSource(path)
	if _, err := client.FetchProblems(context.Background()); err == nil {
		t.Fatal("FetchProblems() returned nil error, want validation error")
	}
}

func TestClientFetchProblemsPaidOnlyMissingNeetCodeURL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "problems.json")
	writeJSON(t, path, map[string]any{
		"problems": []map[string]any{
			// paid_only but no neetcode_url
			{"problem_number": 1, "title": "Paid Problem", "slug": "paid-problem", "difficulty": "Easy", "category": "Arrays & Hashing", "is_paid_only": true},
		},
	})

	client := NewProblemSource(path)
	if _, err := client.FetchProblems(context.Background()); err == nil {
		t.Fatal("FetchProblems() returned nil error, want validation error for missing neetcode_url")
	}
}
