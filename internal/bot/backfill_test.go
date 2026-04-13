package bot

import (
	"context"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
)

func TestBackfillRecordsMissingThreads(t *testing.T) {
	t.Parallel()

	guild := testGuild()
	repo := &stubCheckRepository{
		guildSettings: config.GuildSettings{Guilds: []config.Guild{guild}},
		state: state.State{
			GuildStates: map[string]state.GuildState{
				guild.GuildID: {
					NextProblemNumber: 3,   // problems 1 and 2 have been posted
					PostedThreads:     nil, // but not yet tracked
				},
			},
		},
	}

	lister := &stubThreadLister{
		activeThreads: []*discordgo.Channel{
			{ID: "thread-1", Name: "#N1 Two Sum", ParentID: guild.ForumChannelID},
		},
		archivedThreads: []*discordgo.Channel{
			{
				ID:       "thread-2",
				Name:     "#N2 Add Two Numbers",
				ParentID: guild.ForumChannelID,
				ThreadMetadata: &discordgo.ThreadMetadata{
					ArchiveTimestamp: time.Now(),
				},
			},
		},
		messages: map[string][]*discordgo.Message{
			"thread-1": {{ID: "msg-1"}},
			"thread-2": {{ID: "msg-2"}},
		},
	}

	if err := backfill(context.Background(), repo, lister, nil); err != nil {
		t.Fatalf("backfill() error = %v", err)
	}

	gs := repo.state.GuildStates[guild.GuildID]
	if len(gs.PostedThreads) != 2 {
		t.Fatalf("PostedThreads len = %d, want 2", len(gs.PostedThreads))
	}
	if gs.PostedThreads[1].ThreadID != "thread-1" {
		t.Errorf("PostedThreads[1].ThreadID = %q, want %q", gs.PostedThreads[1].ThreadID, "thread-1")
	}
	if gs.PostedThreads[2].ThreadID != "thread-2" {
		t.Errorf("PostedThreads[2].ThreadID = %q, want %q", gs.PostedThreads[2].ThreadID, "thread-2")
	}
	if lister.reactionAddCalls != 2 {
		t.Errorf("reactionAddCalls = %d, want 2", lister.reactionAddCalls)
	}
}

func TestBackfillSkipsAlreadyTracked(t *testing.T) {
	t.Parallel()

	guild := testGuild()
	repo := &stubCheckRepository{
		guildSettings: config.GuildSettings{Guilds: []config.Guild{guild}},
		state: state.State{
			GuildStates: map[string]state.GuildState{
				guild.GuildID: {
					NextProblemNumber: 2,
					PostedThreads: map[int]state.PostedThread{
						1: {ThreadID: "thread-1", MessageID: "msg-1"},
					},
				},
			},
		},
	}

	lister := &stubThreadLister{
		activeThreads: []*discordgo.Channel{
			{ID: "thread-1", Name: "#N1 Two Sum", ParentID: guild.ForumChannelID},
		},
		messages: map[string][]*discordgo.Message{},
	}

	if err := backfill(context.Background(), repo, lister, nil); err != nil {
		t.Fatalf("backfill() error = %v", err)
	}

	if lister.reactionAddCalls != 0 {
		t.Errorf("reactionAddCalls = %d, want 0 (already tracked)", lister.reactionAddCalls)
	}
}

// --- stubs ---

type stubThreadLister struct {
	activeThreads    []*discordgo.Channel
	archivedThreads  []*discordgo.Channel
	messages         map[string][]*discordgo.Message
	reactionAddCalls int
}

func (s *stubThreadLister) GuildThreadsActive(_ string, _ ...discordgo.RequestOption) (*discordgo.ThreadsList, error) {
	return &discordgo.ThreadsList{Threads: s.activeThreads}, nil
}

func (s *stubThreadLister) ThreadsArchived(_ string, _ *time.Time, _ int, _ ...discordgo.RequestOption) (*discordgo.ThreadsList, error) {
	return &discordgo.ThreadsList{Threads: s.archivedThreads, HasMore: false}, nil
}

func (s *stubThreadLister) ChannelMessages(channelID string, _ int, _, _, _ string, _ ...discordgo.RequestOption) ([]*discordgo.Message, error) {
	return s.messages[channelID], nil
}

func (s *stubThreadLister) MessageReactionAdd(_, _, _ string, _ ...discordgo.RequestOption) error {
	s.reactionAddCalls++
	return nil
}

// stubCheckRepository needs to satisfy BackfillRepository as well.
// The methods are already defined in check_test.go (same package).
var _ BackfillRepository = (*stubCheckRepository)(nil)

// Verify stubThreadLister satisfies ThreadLister.
var _ ThreadLister = (*stubThreadLister)(nil)

// Verify stubCheckRepository satisfies storage.Repository (needed for bot.Repository).
var _ storage.Repository = (*stubCheckRepository)(nil)
