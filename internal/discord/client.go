package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nkoji21/leetdaily/internal/problemcache"
)

const defaultBaseURL = "https://discord.com/api/v10"

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	baseURL    string
	httpClient HTTPClient
	token      string
}

type Thread struct {
	ID string
}

type ForumTag struct {
	ID        string  `json:"id,omitempty"`
	Name      string  `json:"name"`
	Moderated *bool   `json:"moderated,omitempty"`
	EmojiID   *string `json:"emoji_id,omitempty"`
	EmojiName *string `json:"emoji_name,omitempty"`
}

type channel struct {
	ID            string     `json:"id"`
	AvailableTags []ForumTag `json:"available_tags"`
}

func NewClient(httpClient HTTPClient, token string) (*Client, error) {
	return NewClientWithBaseURL(httpClient, token, defaultBaseURL)
}

func NewClientWithBaseURL(httpClient HTTPClient, token, baseURL string) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("Discord bot token must not be empty")
	}

	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		token:      token,
	}, nil
}

func (c *Client) EnsureDifficultyTags(ctx context.Context, forumChannelID string) (map[problemcache.Difficulty]string, error) {
	var existing channel
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/channels/%s", forumChannelID), nil, &existing); err != nil {
		return nil, err
	}

	current := make(map[string]ForumTag, len(existing.AvailableTags))
	for _, tag := range existing.AvailableTags {
		current[normalizeTagName(tag.Name)] = tag
	}

	desired := []problemcache.Difficulty{
		problemcache.DifficultyEasy,
		problemcache.DifficultyMedium,
		problemcache.DifficultyHard,
	}
	updated := append([]ForumTag{}, existing.AvailableTags...)
	for _, difficulty := range desired {
		key := normalizeTagName(string(difficulty))
		if _, ok := current[key]; ok {
			continue
		}
		updated = append(updated, ForumTag{Name: string(difficulty)})
	}

	if len(updated) != len(existing.AvailableTags) {
		var patched channel
		if err := c.doJSON(ctx, http.MethodPatch, fmt.Sprintf("/channels/%s", forumChannelID), map[string]any{
			"available_tags": updated,
		}, &patched); err != nil {
			return nil, err
		}
		existing = patched
	}

	result := make(map[problemcache.Difficulty]string, len(desired))
	for _, tag := range existing.AvailableTags {
		switch normalizeTagName(tag.Name) {
		case normalizeTagName(string(problemcache.DifficultyEasy)):
			result[problemcache.DifficultyEasy] = tag.ID
		case normalizeTagName(string(problemcache.DifficultyMedium)):
			result[problemcache.DifficultyMedium] = tag.ID
		case normalizeTagName(string(problemcache.DifficultyHard)):
			result[problemcache.DifficultyHard] = tag.ID
		}
	}

	for _, difficulty := range desired {
		if strings.TrimSpace(result[difficulty]) == "" {
			return nil, fmt.Errorf("Discord forum channel %s is missing tag %s after ensure", forumChannelID, difficulty)
		}
	}

	return result, nil
}

func normalizeTagName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (c *Client) CreateForumThread(ctx context.Context, forumChannelID, tagID, title, body string) (Thread, error) {
	payload := map[string]any{
		"name":         title,
		"applied_tags": []string{tagID},
		"message": map[string]any{
			"content": body,
		},
	}

	var response struct {
		ID string `json:"id"`
	}
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/channels/%s/threads", forumChannelID), payload, &response); err != nil {
		return Thread{}, err
	}

	if strings.TrimSpace(response.ID) == "" {
		return Thread{}, fmt.Errorf("Discord thread create returned empty thread ID")
	}

	return Thread{ID: response.ID}, nil
}

func (c *Client) SendMessage(ctx context.Context, channelID, content string) error {
	return c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/channels/%s/messages", channelID), map[string]any{
		"content": content,
	}, nil)
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, destination any) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode Discord request: %w", err)
		}
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build Discord request: %w", err)
	}
	request.Header.Set("Authorization", "Bot "+c.token)
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send Discord request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Discord API %s %s returned status %d", method, path, response.StatusCode)
	}

	if destination == nil {
		return nil
	}

	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		return fmt.Errorf("decode Discord response: %w", err)
	}

	return nil
}
