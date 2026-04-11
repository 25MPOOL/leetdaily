package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/nkoji21/leetdaily/internal/discordid"
)

type Config struct {
	Timezone     string             `json:"timezone"`
	Retry        RetryConfig        `json:"retry"`
	ProblemCache ProblemCacheConfig `json:"problem_cache"`
	Guilds       []Guild            `json:"guilds"`
}

type GuildSettings struct {
	Guilds []Guild `json:"guilds"`
}

type RetryConfig struct {
	IntervalMinutes int `json:"interval_minutes"`
	MaxAttempts     int `json:"max_attempts"`
}

type ProblemCacheConfig struct {
	RefillThreshold int `json:"refill_threshold"`
}

type Guild struct {
	GuildID               string `json:"guild_id"`
	Enabled               bool   `json:"enabled"`
	ForumChannelID        string `json:"forum_channel_id"`
	NotificationChannelID string `json:"notification_channel_id"`
	StartProblemNumber    int    `json:"start_problem_number"`
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Timezone) == "" {
		return fmt.Errorf("timezone must not be empty")
	}

	if _, err := time.LoadLocation(c.Timezone); err != nil {
		return fmt.Errorf("invalid timezone %q: %w", c.Timezone, err)
	}

	if c.Retry.IntervalMinutes < 1 {
		return fmt.Errorf("retry.interval_minutes must be greater than 0: %d", c.Retry.IntervalMinutes)
	}

	if c.Retry.MaxAttempts < 1 {
		return fmt.Errorf("retry.max_attempts must be greater than 0: %d", c.Retry.MaxAttempts)
	}

	if c.ProblemCache.RefillThreshold < 1 {
		return fmt.Errorf("problem_cache.refill_threshold must be greater than 0: %d", c.ProblemCache.RefillThreshold)
	}

	if err := validateGuilds(c.Guilds); err != nil {
		return err
	}

	return nil
}

func (c Config) EnabledGuilds() []Guild {
	return enabledGuilds(c.Guilds)
}

func (c Config) Location() (*time.Location, error) {
	location, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", c.Timezone, err)
	}

	return location, nil
}

func (g GuildSettings) Validate() error {
	return validateGuilds(g.Guilds)
}

func (g GuildSettings) EnabledGuilds() []Guild {
	return enabledGuilds(g.Guilds)
}

func validateGuilds(guilds []Guild) error {
	seenGuilds := make(map[string]struct{}, len(guilds))
	for i, guild := range guilds {
		if err := guild.Validate(); err != nil {
			return fmt.Errorf("guilds[%d]: %w", i, err)
		}

		if _, ok := seenGuilds[guild.GuildID]; ok {
			return fmt.Errorf("guilds[%d]: duplicate guild_id %q", i, guild.GuildID)
		}
		seenGuilds[guild.GuildID] = struct{}{}
	}

	return nil
}

func enabledGuilds(guilds []Guild) []Guild {
	enabled := make([]Guild, 0, len(guilds))
	for _, guild := range guilds {
		if guild.Enabled {
			enabled = append(enabled, guild)
		}
	}

	return enabled
}

func (g *GuildSettings) Upsert(guild Guild) {
	for i := range g.Guilds {
		if g.Guilds[i].GuildID == guild.GuildID {
			g.Guilds[i] = guild
			return
		}
	}

	g.Guilds = append(g.Guilds, guild)
}

func (g Guild) Validate() error {
	if !discordid.IsValid(g.GuildID) {
		return fmt.Errorf("guild_id must be a numeric Discord ID: %q", g.GuildID)
	}

	if !discordid.IsValid(g.ForumChannelID) {
		return fmt.Errorf("forum_channel_id must be a numeric Discord ID: %q", g.ForumChannelID)
	}

	if !discordid.IsValid(g.NotificationChannelID) {
		return fmt.Errorf("notification_channel_id must be a numeric Discord ID: %q", g.NotificationChannelID)
	}

	if g.StartProblemNumber < 1 {
		return fmt.Errorf("start_problem_number must be greater than 0: %d", g.StartProblemNumber)
	}

	return nil
}
