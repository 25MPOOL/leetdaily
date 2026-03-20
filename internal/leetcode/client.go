package leetcode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nkoji21/leetdaily/internal/problemcache"
)

const defaultGraphQLEndpoint = "https://leetcode.com/graphql"

type Client struct {
	endpoint   string
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		endpoint:   defaultGraphQLEndpoint,
		httpClient: httpClient,
	}
}

func NewClientWithEndpoint(httpClient *http.Client, endpoint string) *Client {
	client := NewClient(httpClient)
	if strings.TrimSpace(endpoint) != "" {
		client.endpoint = strings.TrimSpace(endpoint)
	}

	return client
}

func (c *Client) FetchProblems(ctx context.Context) ([]problemcache.Problem, error) {
	var all []problemcache.Problem
	skip := 0

	for {
		response, err := c.queryProblemset(ctx, skip)
		if err != nil {
			return nil, err
		}

		for _, question := range response.Data.ProblemsetQuestionList.Questions {
			problem, err := normalizeQuestion(question)
			if err != nil {
				return nil, err
			}
			all = append(all, problem)
		}

		total := response.Data.ProblemsetQuestionList.Total
		skip += len(response.Data.ProblemsetQuestionList.Questions)
		if skip >= total || len(response.Data.ProblemsetQuestionList.Questions) == 0 {
			break
		}
	}

	return all, nil
}

func (c *Client) queryProblemset(ctx context.Context, skip int) (problemsetResponse, error) {
	requestBody := graphqlRequest{
		Query: problemsetQuestionListQuery,
		Variables: graphqlVariables{
			CategorySlug: "",
			Limit:        1000,
			Skip:         skip,
			Filters:      graphqlFilters{},
		},
		OperationName: "problemsetQuestionList",
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return problemsetResponse{}, fmt.Errorf("encode LeetCode GraphQL request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return problemsetResponse{}, fmt.Errorf("build LeetCode GraphQL request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return problemsetResponse{}, fmt.Errorf("send LeetCode GraphQL request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return problemsetResponse{}, fmt.Errorf("LeetCode GraphQL returned status %d", response.StatusCode)
	}

	var decoded problemsetResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return problemsetResponse{}, fmt.Errorf("decode LeetCode GraphQL response: %w", err)
	}

	if len(decoded.Errors) > 0 {
		return problemsetResponse{}, fmt.Errorf("LeetCode GraphQL error: %s", decoded.Errors[0].Message)
	}

	return decoded, nil
}

func normalizeQuestion(question question) (problemcache.Problem, error) {
	difficulty, err := normalizeDifficulty(question.Difficulty)
	if err != nil {
		return problemcache.Problem{}, err
	}

	problem := problemcache.Problem{
		ProblemNumber: question.FrontendID,
		Title:         strings.TrimSpace(question.Title),
		Slug:          strings.TrimSpace(question.TitleSlug),
		Difficulty:    difficulty,
		IsPaidOnly:    question.IsPaidOnly,
	}

	if err := problem.Validate(); err != nil {
		return problemcache.Problem{}, fmt.Errorf("normalize LeetCode problem %d: %w", question.FrontendID, err)
	}

	return problem, nil
}

func normalizeDifficulty(raw string) (problemcache.Difficulty, error) {
	switch strings.TrimSpace(raw) {
	case "Easy":
		return problemcache.DifficultyEasy, nil
	case "Medium":
		return problemcache.DifficultyMedium, nil
	case "Hard":
		return problemcache.DifficultyHard, nil
	default:
		return "", fmt.Errorf("unsupported LeetCode difficulty %q", raw)
	}
}

const problemsetQuestionListQuery = `
query problemsetQuestionList($categorySlug: String, $limit: Int, $skip: Int, $filters: QuestionListFilterInput) {
  problemsetQuestionList: questionList(
    categorySlug: $categorySlug
    limit: $limit
    skip: $skip
    filters: $filters
  ) {
    total: totalNum
    questions: data {
      frontendQuestionId
      title
      titleSlug
      difficulty
      isPaidOnly
    }
  }
}`

type graphqlRequest struct {
	Query         string           `json:"query"`
	Variables     graphqlVariables `json:"variables"`
	OperationName string           `json:"operationName"`
}

type graphqlVariables struct {
	CategorySlug string         `json:"categorySlug"`
	Limit        int            `json:"limit"`
	Skip         int            `json:"skip"`
	Filters      graphqlFilters `json:"filters"`
}

type graphqlFilters struct{}

type problemsetResponse struct {
	Data struct {
		ProblemsetQuestionList struct {
			Total     int        `json:"total"`
			Questions []question `json:"questions"`
		} `json:"problemsetQuestionList"`
	} `json:"data"`
	Errors []graphqlError `json:"errors"`
}

type graphqlError struct {
	Message string `json:"message"`
}

type question struct {
	FrontendID int    `json:"frontendQuestionId,string"`
	Title      string `json:"title"`
	TitleSlug  string `json:"titleSlug"`
	Difficulty string `json:"difficulty"`
	IsPaidOnly bool   `json:"isPaidOnly"`
}
