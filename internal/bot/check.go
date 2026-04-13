package bot

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/problemcache"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
)

// Compile-time check: *discordgo.Session satisfies ReactionChecker.
var _ ReactionChecker = (*discordgo.Session)(nil)

// CheckRepository is the storage interface required by the /check handler.
type CheckRepository interface {
	LoadState(context.Context) (state.State, storage.Version, error)
	LoadGuildSettings(context.Context) (config.GuildSettings, storage.Version, error)
	LoadProblemCache(context.Context) (problemcache.Cache, storage.Version, error)
}

// ReactionChecker fetches users who reacted with a given emoji on a message.
type ReactionChecker interface {
	MessageReactions(channelID, messageID, emoji string, limit int, beforeID, afterID string, options ...discordgo.RequestOption) ([]*discordgo.User, error)
}

type checkHandler struct {
	repository     CheckRepository
	reactionClient ReactionChecker
	logger         *slog.Logger
}

func (h *checkHandler) handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Acknowledge immediately with a deferred response to avoid the 3-second timeout.
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		h.logger.Error("failed to send deferred response", "error", err)
		return
	}

	ctx := context.Background()
	content, err := h.buildResponse(ctx, i)
	if err != nil {
		h.logger.Error("failed to build /check response", "error", err)
		content = "エラーが発生しました。しばらく待ってから再試行してください。"
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	}); err != nil {
		h.logger.Error("failed to edit interaction response", "error", err)
	}
}

func (h *checkHandler) buildResponse(ctx context.Context, i *discordgo.InteractionCreate) (string, error) {
	guildSettings, _, err := h.repository.LoadGuildSettings(ctx)
	if err != nil {
		return "", fmt.Errorf("load guild settings: %w", err)
	}

	guild, ok := findGuild(guildSettings, i.GuildID)
	if !ok {
		return "このサーバーは設定されていません。", nil
	}

	if i.ChannelID != guild.NotificationChannelID {
		return fmt.Sprintf("このコマンドは <#%s> でのみ使用できます。", guild.NotificationChannelID), nil
	}

	if i.Member == nil || i.Member.User == nil {
		return "このコマンドはサーバー内でのみ使用できます。", nil
	}

	currentState, _, err := h.repository.LoadState(ctx)
	if err != nil {
		return "", fmt.Errorf("load state: %w", err)
	}

	guildState, _ := currentState.EnsureGuild(guild.GuildID, guild.StartProblemNumber)

	cache, _, err := h.repository.LoadProblemCache(ctx)
	if err != nil {
		return "", fmt.Errorf("load problem cache: %w", err)
	}
	problemsByNumber := cache.ByNumber()

	userID := i.Member.User.ID

	// Collect unsolved problems in ascending order.
	type unsolvedEntry struct {
		number  int
		problem problemcache.Problem
	}
	var unsolved []unsolvedEntry

	for problemNumber, pt := range guildState.PostedThreads {
		problem, ok := problemsByNumber[problemNumber]
		if !ok {
			continue
		}

		users, err := h.allReactionUsers(pt.ThreadID, pt.MessageID, "done_4:1461518237329133761")
		if err != nil {
			h.logger.Warn("failed to get reactions", "problem_number", problemNumber, "error", err)
			continue
		}

		if !containsUser(users, userID) {
			unsolved = append(unsolved, unsolvedEntry{number: problemNumber, problem: problem})
		}
	}

	if len(unsolved) == 0 {
		return "✅ 全問解決済みです！", nil
	}

	sort.Slice(unsolved, func(i, j int) bool {
		return unsolved[i].number < unsolved[j].number
	})

	var sb strings.Builder
	sb.WriteString("📋 未解決の問題一覧:\n")
	for _, entry := range unsolved {
		sb.WriteString(fmt.Sprintf("\n#N%d %s - %s - %s\n%s\n",
			entry.number,
			entry.problem.Title,
			entry.problem.Difficulty,
			entry.problem.Category,
			entry.problem.URL(),
		))
	}
	return sb.String(), nil
}

// allReactionUsers fetches all users who reacted, handling Discord's 100-user pagination.
func (h *checkHandler) allReactionUsers(channelID, messageID, emoji string) ([]*discordgo.User, error) {
	var all []*discordgo.User
	afterID := ""
	for {
		batch, err := h.reactionClient.MessageReactions(channelID, messageID, emoji, 100, "", afterID)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			break
		}
		afterID = batch[len(batch)-1].ID
	}
	return all, nil
}

func findGuild(settings config.GuildSettings, guildID string) (config.Guild, bool) {
	for _, g := range settings.Guilds {
		if g.GuildID == guildID {
			return g, true
		}
	}
	return config.Guild{}, false
}

func containsUser(users []*discordgo.User, userID string) bool {
	for _, u := range users {
		if u.ID == userID {
			return true
		}
	}
	return false
}
