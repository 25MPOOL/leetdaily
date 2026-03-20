package config

import "testing"

func TestConfigValidateAndEnabledGuilds(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Timezone: "Asia/Tokyo",
		Retry: RetryConfig{
			IntervalMinutes: 5,
			MaxAttempts:     3,
		},
		ProblemCache: ProblemCacheConfig{
			RefillThreshold: 30,
		},
		Guilds: []Guild{
			{
				GuildID:               "123456789012345678",
				Enabled:               true,
				ForumChannelID:        "234567890123456789",
				NotificationChannelID: "345678901234567890",
				StartProblemNumber:    1,
			},
			{
				GuildID:               "456789012345678901",
				Enabled:               false,
				ForumChannelID:        "567890123456789012",
				NotificationChannelID: "678901234567890123",
				StartProblemNumber:    51,
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}

	enabled := cfg.EnabledGuilds()
	if len(enabled) != 1 {
		t.Fatalf("len(EnabledGuilds()) = %d, want 1", len(enabled))
	}

	if enabled[0].GuildID != "123456789012345678" {
		t.Fatalf("EnabledGuilds()[0].GuildID = %q, want %q", enabled[0].GuildID, "123456789012345678")
	}
}

func TestConfigValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "invalid timezone",
			cfg: Config{
				Timezone: "Mars/Olympus",
				Retry: RetryConfig{
					IntervalMinutes: 5,
					MaxAttempts:     3,
				},
				ProblemCache: ProblemCacheConfig{
					RefillThreshold: 30,
				},
			},
		},
		{
			name: "duplicate guild",
			cfg: Config{
				Timezone: "Asia/Tokyo",
				Retry: RetryConfig{
					IntervalMinutes: 5,
					MaxAttempts:     3,
				},
				ProblemCache: ProblemCacheConfig{
					RefillThreshold: 30,
				},
				Guilds: []Guild{
					{
						GuildID:               "123456789012345678",
						Enabled:               true,
						ForumChannelID:        "234567890123456789",
						NotificationChannelID: "345678901234567890",
						StartProblemNumber:    1,
					},
					{
						GuildID:               "123456789012345678",
						Enabled:               true,
						ForumChannelID:        "567890123456789012",
						NotificationChannelID: "678901234567890123",
						StartProblemNumber:    2,
					},
				},
			},
		},
		{
			name: "invalid guild snowflake",
			cfg: Config{
				Timezone: "Asia/Tokyo",
				Retry: RetryConfig{
					IntervalMinutes: 5,
					MaxAttempts:     3,
				},
				ProblemCache: ProblemCacheConfig{
					RefillThreshold: 30,
				},
				Guilds: []Guild{
					{
						GuildID:               "guild-1",
						Enabled:               true,
						ForumChannelID:        "234567890123456789",
						NotificationChannelID: "345678901234567890",
						StartProblemNumber:    1,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.cfg.Validate(); err == nil {
				t.Fatal("Validate() returned nil error, want validation error")
			}
		})
	}
}
