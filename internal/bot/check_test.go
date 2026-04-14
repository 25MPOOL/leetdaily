package bot

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/problemcache"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
)

func TestCheckHandlerAllSolved(t *testing.T) {
	t.Parallel()

	repo := &stubCheckRepository{
		guildSettings: config.GuildSettings{Guilds: []config.Guild{testGuild()}},
		state: state.State{
			GuildStates: map[string]state.GuildState{
				"123456789012345678": {
					NextProblemNumber: 2,
					PostedThreads: map[int]state.PostedThread{
						1: {ThreadID: "thread-1", MessageID: "msg-1"},
					},
				},
			},
		},
		cache: problemcache.Cache{
			UpdatedAt: timePtr(time.Now()),
			Problems: []problemcache.Problem{
				{ProblemNumber: 1, Title: "Two Sum", Slug: "two-sum", Difficulty: problemcache.DifficultyEasy, Category: "Arrays & Hashing"},
			},
		},
	}
	// User "user-1" has already reacted.
	reactions := &stubReactionChecker{
		users: map[string][]*discordgo.User{
			"msg-1": {{ID: "user-1"}},
		},
	}

	h := &checkHandler{repository: repo, reactionClient: reactions}

	interaction := makeInteraction("123456789012345678", "345678901234567890", "user-1")
	content, err := h.buildResponse(context.Background(), interaction)
	if err != nil {
		t.Fatalf("buildResponse() error = %v", err)
	}

	if content != "✅ 全問解決済みです！" {
		t.Fatalf("content = %q, want all-solved message", content)
	}
}

func TestCheckHandlerUnsolvedProblems(t *testing.T) {
	t.Parallel()

	repo := &stubCheckRepository{
		guildSettings: config.GuildSettings{Guilds: []config.Guild{testGuild()}},
		state: state.State{
			GuildStates: map[string]state.GuildState{
				"123456789012345678": {
					NextProblemNumber: 3,
					PostedThreads: map[int]state.PostedThread{
						1: {ThreadID: "thread-1", MessageID: "msg-1"},
						2: {ThreadID: "thread-2", MessageID: "msg-2"},
					},
				},
			},
		},
		cache: problemcache.Cache{
			UpdatedAt: timePtr(time.Now()),
			Problems: []problemcache.Problem{
				{ProblemNumber: 1, Title: "Two Sum", Slug: "two-sum", Difficulty: problemcache.DifficultyEasy, Category: "Arrays & Hashing"},
				{ProblemNumber: 2, Title: "Add Two Numbers", Slug: "add-two-numbers", Difficulty: problemcache.DifficultyMedium, Category: "Linked List"},
			},
		},
	}
	// User "user-1" solved problem 1 but not problem 2.
	reactions := &stubReactionChecker{
		users: map[string][]*discordgo.User{
			"msg-1": {{ID: "user-1"}},
			"msg-2": {},
		},
	}

	h := &checkHandler{repository: repo, reactionClient: reactions}

	interaction := makeInteraction("123456789012345678", "345678901234567890", "user-1")
	content, err := h.buildResponse(context.Background(), interaction)
	if err != nil {
		t.Fatalf("buildResponse() error = %v", err)
	}

	if content == "✅ 全問解決済みです！" {
		t.Fatal("expected unsolved list, got all-solved message")
	}
	// Problem 1 should be absent; problem 2 should be present.
	if contains(content, "Two Sum") {
		t.Fatal("content should not contain solved problem 'Two Sum'")
	}
	if !contains(content, "Add Two Numbers") {
		t.Fatal("content should contain unsolved problem 'Add Two Numbers'")
	}
}

func TestCheckHandlerReactionErrorCountsAsUnsolved(t *testing.T) {
	t.Parallel()

	repo := &stubCheckRepository{
		guildSettings: config.GuildSettings{Guilds: []config.Guild{testGuild()}},
		state: state.State{
			GuildStates: map[string]state.GuildState{
				"123456789012345678": {
					NextProblemNumber: 2,
					PostedThreads: map[int]state.PostedThread{
						1: {ThreadID: "thread-1", MessageID: "msg-1"},
					},
				},
			},
		},
		cache: problemcache.Cache{
			UpdatedAt: timePtr(time.Now()),
			Problems: []problemcache.Problem{
				{ProblemNumber: 1, Title: "Two Sum", Slug: "two-sum", Difficulty: problemcache.DifficultyEasy, Category: "Arrays & Hashing"},
			},
		},
	}
	// Reaction fetch returns an error.
	reactions := &stubReactionChecker{errOnMessageID: "msg-1"}

	h := &checkHandler{repository: repo, reactionClient: reactions, logger: slog.Default()}

	interaction := makeInteraction("123456789012345678", "345678901234567890", "user-1")
	content, err := h.buildResponse(context.Background(), interaction)
	if err != nil {
		t.Fatalf("buildResponse() error = %v", err)
	}

	if content == "✅ 全問解決済みです！" {
		t.Fatal("reaction fetch error should not produce all-solved message")
	}
}

func TestCheckHandlerNoPostedThreads(t *testing.T) {
	t.Parallel()

	repo := &stubCheckRepository{
		guildSettings: config.GuildSettings{Guilds: []config.Guild{testGuild()}},
		state: state.State{
			GuildStates: map[string]state.GuildState{
				"123456789012345678": {
					NextProblemNumber: 1,
					PostedThreads:     map[int]state.PostedThread{},
				},
			},
		},
		cache: problemcache.Cache{},
	}

	h := &checkHandler{repository: repo, reactionClient: &stubReactionChecker{}}

	interaction := makeInteraction("123456789012345678", "345678901234567890", "user-1")
	content, err := h.buildResponse(context.Background(), interaction)
	if err != nil {
		t.Fatalf("buildResponse() error = %v", err)
	}

	if content == "✅ 全問解決済みです！" {
		t.Fatal("empty PostedThreads should not produce all-solved message")
	}
}

func TestCheckHandlerWrongChannel(t *testing.T) {
	t.Parallel()

	repo := &stubCheckRepository{
		guildSettings: config.GuildSettings{Guilds: []config.Guild{testGuild()}},
		state:         state.State{GuildStates: map[string]state.GuildState{}},
		cache:         problemcache.Cache{},
	}

	h := &checkHandler{repository: repo, reactionClient: &stubReactionChecker{}}

	// Use a different channel ID.
	interaction := makeInteraction("123456789012345678", "999999999999999999", "user-1")
	content, err := h.buildResponse(context.Background(), interaction)
	if err != nil {
		t.Fatalf("buildResponse() error = %v", err)
	}

	if !contains(content, "345678901234567890") {
		t.Fatalf("expected channel restriction message, got %q", content)
	}
}

// --- stubs ---

type stubCheckRepository struct {
	guildSettings config.GuildSettings
	state         state.State
	cache         problemcache.Cache
}

func (s *stubCheckRepository) LoadState(context.Context) (state.State, storage.Version, error) {
	return s.state, storage.Version{}, nil
}

func (s *stubCheckRepository) LoadGuildSettings(context.Context) (config.GuildSettings, storage.Version, error) {
	return s.guildSettings, storage.Version{}, nil
}

func (s *stubCheckRepository) LoadProblemCache(context.Context) (problemcache.Cache, storage.Version, error) {
	return s.cache, storage.Version{}, nil
}

func (s *stubCheckRepository) SaveState(_ context.Context, st state.State, v storage.Version) (storage.Version, error) {
	s.state = st
	return v, nil
}

func (s *stubCheckRepository) SaveGuildSettings(_ context.Context, gs config.GuildSettings, v storage.Version) (storage.Version, error) {
	s.guildSettings = gs
	return v, nil
}

func (s *stubCheckRepository) LoadConfig(context.Context) (config.Config, error) {
	return config.Config{}, nil
}

func (s *stubCheckRepository) SaveProblemCache(_ context.Context, c problemcache.Cache, v storage.Version) (storage.Version, error) {
	s.cache = c
	return v, nil
}

type stubReactionChecker struct {
	// users maps messageID -> list of users who reacted.
	users map[string][]*discordgo.User
	// errOnMessageID returns an error when this messageID is queried.
	errOnMessageID string
}

func (s *stubReactionChecker) MessageReactions(_, messageID, _ string, _ int, _, _ string, _ ...discordgo.RequestOption) ([]*discordgo.User, error) {
	if s.errOnMessageID != "" && messageID == s.errOnMessageID {
		return nil, fmt.Errorf("simulated reaction fetch error")
	}
	return s.users[messageID], nil
}

func testGuild() config.Guild {
	return config.Guild{
		GuildID:               "123456789012345678",
		Enabled:               true,
		ForumChannelID:        "234567890123456789",
		NotificationChannelID: "345678901234567890",
		StartProblemNumber:    1,
	}
}

func makeInteraction(guildID, channelID, userID string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			GuildID:   guildID,
			ChannelID: channelID,
			Member: &discordgo.Member{
				User: &discordgo.User{ID: userID},
			},
		},
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func timePtr(t time.Time) *time.Time { return &t }
