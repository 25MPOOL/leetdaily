package bot

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
)

// BackfillRepository is the storage interface required by the backfill process.
type BackfillRepository interface {
	LoadState(context.Context) (state.State, storage.Version, error)
	SaveState(context.Context, state.State, storage.Version) (storage.Version, error)
	LoadGuildSettings(context.Context) (config.GuildSettings, storage.Version, error)
}

// ThreadLister fetches threads and messages from Discord channels.
type ThreadLister interface {
	GuildThreadsActive(guildID string, options ...discordgo.RequestOption) (*discordgo.ThreadsList, error)
	ThreadsArchived(channelID string, before *time.Time, limit int, options ...discordgo.RequestOption) (*discordgo.ThreadsList, error)
	ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string, options ...discordgo.RequestOption) ([]*discordgo.Message, error)
	MessageReactionAdd(channelID, messageID, emojiID string, options ...discordgo.RequestOption) error
}

var threadTitlePattern = regexp.MustCompile(`^#N(\d+)\s`)

// backfill populates PostedThreads in state for any posted problems that are missing entries.
// This handles threads that were posted before the PostedThreads feature was introduced.
func backfill(ctx context.Context, repo BackfillRepository, lister ThreadLister, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	guildSettings, _, err := repo.LoadGuildSettings(ctx)
	if err != nil {
		return fmt.Errorf("load guild settings: %w", err)
	}

	currentState, stateVersion, err := repo.LoadState(ctx)
	if err != nil {
		if storage.IsNotFound(err) {
			return nil // no state yet, nothing to backfill
		}
		return fmt.Errorf("load state: %w", err)
	}

	modified := false
	for _, guild := range guildSettings.EnabledGuilds() {
		guildState, _ := currentState.EnsureGuild(guild.GuildID, guild.StartProblemNumber)

		changed, err := backfillGuild(lister, logger, guild, &guildState)
		if err != nil {
			logger.Warn("backfill failed for guild", "guild_id", guild.GuildID, "error", err)
			continue
		}
		if changed {
			currentState.GuildStates[guild.GuildID] = guildState
			modified = true
		}
	}

	if !modified {
		return nil
	}

	if _, err := repo.SaveState(ctx, currentState, stateVersion); err != nil {
		return fmt.Errorf("save state after backfill: %w", err)
	}
	return nil
}

func backfillGuild(lister ThreadLister, logger *slog.Logger, guild config.Guild, guildState *state.GuildState) (bool, error) {
	if guildState.NextProblemNumber <= guild.StartProblemNumber {
		return false, nil // nothing posted yet
	}

	threads, err := collectForumThreads(lister, guild.GuildID, guild.ForumChannelID)
	if err != nil {
		return false, fmt.Errorf("collect forum threads: %w", err)
	}

	if guildState.PostedThreads == nil {
		guildState.PostedThreads = map[int]state.PostedThread{}
	}

	changed := false
	for _, thread := range threads {
		matches := threadTitlePattern.FindStringSubmatch(thread.Name)
		if matches == nil {
			continue
		}
		problemNumber, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		if _, already := guildState.PostedThreads[problemNumber]; already {
			continue // already tracked
		}

		// Fetch the first message in the thread to get its ID.
		// Discord returns messages newest-first, so we use afterID="" and get oldest by fetching with limit=1 around "0".
		messages, err := lister.ChannelMessages(thread.ID, 1, "", "0", "")
		if err != nil {
			logger.Warn("backfill: failed to fetch messages for thread", "thread_id", thread.ID, "error", err)
			continue
		}
		if len(messages) == 0 {
			logger.Warn("backfill: no messages found in thread", "thread_id", thread.ID)
			continue
		}
		messageID := messages[0].ID

		// Add the done_4 reaction on behalf of the bot.
		if err := lister.MessageReactionAdd(thread.ID, messageID, "done_4:1461518237329133761"); err != nil {
			logger.Warn("backfill: failed to add reaction", "thread_id", thread.ID, "error", err)
		}

		guildState.PostedThreads[problemNumber] = state.PostedThread{
			ThreadID:  thread.ID,
			MessageID: messageID,
		}
		changed = true
		logger.Info("backfill: recorded thread", "problem_number", problemNumber, "thread_id", thread.ID)
	}

	return changed, nil
}

func collectForumThreads(lister ThreadLister, guildID, forumChannelID string) ([]*discordgo.Channel, error) {
	var threads []*discordgo.Channel

	// Active threads.
	active, err := lister.GuildThreadsActive(guildID)
	if err != nil {
		return nil, fmt.Errorf("list active threads: %w", err)
	}
	for _, t := range active.Threads {
		if t.ParentID == forumChannelID {
			threads = append(threads, t)
		}
	}

	// Archived threads (paginate until exhausted).
	var before *time.Time
	for {
		archived, err := lister.ThreadsArchived(forumChannelID, before, 100)
		if err != nil {
			return nil, fmt.Errorf("list archived threads: %w", err)
		}
		threads = append(threads, archived.Threads...)
		if !archived.HasMore {
			break
		}
		if len(archived.Threads) > 0 {
			ts := archived.Threads[len(archived.Threads)-1].ThreadMetadata.ArchiveTimestamp
			before = &ts
		}
	}

	return threads, nil
}
