package leetcode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientFetchProblemsPaginatesAndNormalizes(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}

		var request graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request error = %v", err)
		}

		skip := request.Variables.Skip
		var response any
		switch skip {
		case 0:
			response = map[string]any{
				"data": map[string]any{
					"problemsetQuestionList": map[string]any{
						"total": 3,
						"questions": []map[string]any{
							{
								"frontendQuestionId": "1",
								"title":              "Two Sum",
								"titleSlug":          "two-sum",
								"difficulty":         "Easy",
								"isPaidOnly":         false,
							},
							{
								"frontendQuestionId": "2",
								"title":              "Add Two Numbers",
								"titleSlug":          "add-two-numbers",
								"difficulty":         "Medium",
								"isPaidOnly":         true,
							},
						},
					},
				},
			}
		case 2:
			response = map[string]any{
				"data": map[string]any{
					"problemsetQuestionList": map[string]any{
						"total": 3,
						"questions": []map[string]any{
							{
								"frontendQuestionId": "3",
								"title":              "Longest Substring Without Repeating Characters",
								"titleSlug":          "longest-substring-without-repeating-characters",
								"difficulty":         "Medium",
								"isPaidOnly":         false,
							},
						},
					},
				},
			}
		default:
			t.Fatalf("unexpected skip = %d", skip)
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode response error = %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithEndpoint(server.Client(), server.URL)
	problems, err := client.FetchProblems(context.Background())
	if err != nil {
		t.Fatalf("FetchProblems() error = %v", err)
	}

	if len(problems) != 3 {
		t.Fatalf("len(FetchProblems()) = %d, want 3", len(problems))
	}

	if problems[1].ProblemNumber != 2 || !problems[1].IsPaidOnly {
		t.Fatalf("problems[1] = %#v, want paid problem #2", problems[1])
	}
}

func TestClientFetchProblemsRejectsGraphQLError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{"message": "upstream failed"},
			},
		})
	}))
	defer server.Close()

	client := NewClientWithEndpoint(server.Client(), server.URL)
	if _, err := client.FetchProblems(context.Background()); err == nil {
		t.Fatal("FetchProblems() returned nil error, want GraphQL error")
	}
}

func TestClientFetchProblemsRejectsUnknownDifficulty(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"problemsetQuestionList": map[string]any{
					"total": 1,
					"questions": []map[string]any{
						{
							"frontendQuestionId": "1",
							"title":              "Impossible Problem",
							"titleSlug":          "impossible-problem",
							"difficulty":         "Legendary",
							"isPaidOnly":         false,
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClientWithEndpoint(server.Client(), server.URL)
	if _, err := client.FetchProblems(context.Background()); err == nil {
		t.Fatal("FetchProblems() returned nil error, want difficulty error")
	}
}
