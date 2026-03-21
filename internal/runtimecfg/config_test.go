package runtimecfg

import (
	"log/slog"
	"path/filepath"
	"testing"
)

func TestLoadFromEnvDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFromEnv(nil)
	if err != nil {
		t.Fatalf("LoadFromEnv(nil) returned error: %v", err)
	}

	if cfg.Mode != ModeHTTP {
		t.Fatalf("Mode = %q, want %q", cfg.Mode, ModeHTTP)
	}

	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}

	if cfg.HTTPPort != 8080 {
		t.Fatalf("HTTPPort = %d, want 8080", cfg.HTTPPort)
	}

	if cfg.DataDir != "." {
		t.Fatalf("DataDir = %q, want \".\"", cfg.DataDir)
	}

	if cfg.ConfigPath() != "config.json" {
		t.Fatalf("ConfigPath() = %q, want %q", cfg.ConfigPath(), "config.json")
	}

	if cfg.GuildsPath() != "guilds.json" {
		t.Fatalf("GuildsPath() = %q, want %q", cfg.GuildsPath(), "guilds.json")
	}

	if cfg.StatePath() != "state.json" {
		t.Fatalf("StatePath() = %q, want %q", cfg.StatePath(), "state.json")
	}

	if cfg.ProblemsPath() != "problems.json" {
		t.Fatalf("ProblemsPath() = %q, want %q", cfg.ProblemsPath(), "problems.json")
	}

	if cfg.UsesGCS() {
		t.Fatal("UsesGCS() = true, want false")
	}

	if cfg.DiscordBotToken != "" {
		t.Fatalf("DiscordBotToken = %q, want empty", cfg.DiscordBotToken)
	}
}

func TestLoadFromEnvCustomValues(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFromEnv(func(key string) (string, bool) {
		switch key {
		case "LEETDAILY_RUNTIME":
			return "job", true
		case "LEETDAILY_LOG_LEVEL":
			return "debug", true
		case "PORT":
			return "9090", true
		case "LEETDAILY_DATA_DIR":
			return "./var/data", true
		case "DISCORD_BOT_TOKEN":
			return "discord-token", true
		case "DISCORD_APPLICATION_PUBLIC_KEY":
			return "discord-public-key", true
		default:
			return "", false
		}
	})
	if err != nil {
		t.Fatalf("LoadFromEnv(custom) returned error: %v", err)
	}

	if cfg.Mode != ModeJob {
		t.Fatalf("Mode = %q, want %q", cfg.Mode, ModeJob)
	}

	if cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}

	if cfg.HTTPPort != 9090 {
		t.Fatalf("HTTPPort = %d, want 9090", cfg.HTTPPort)
	}

	wantDataDir := filepath.Join("var", "data")
	if cfg.DataDir != wantDataDir {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, wantDataDir)
	}

	wantConfigPath := filepath.Join(wantDataDir, "config.json")
	if cfg.ConfigPath() != wantConfigPath {
		t.Fatalf("ConfigPath() = %q, want %q", cfg.ConfigPath(), wantConfigPath)
	}

	wantGuildsPath := filepath.Join(wantDataDir, "guilds.json")
	if cfg.GuildsPath() != wantGuildsPath {
		t.Fatalf("GuildsPath() = %q, want %q", cfg.GuildsPath(), wantGuildsPath)
	}

	if cfg.HTTPAddr() != ":9090" {
		t.Fatalf("HTTPAddr() = %q, want %q", cfg.HTTPAddr(), ":9090")
	}

	if cfg.DiscordBotToken != "discord-token" {
		t.Fatalf("DiscordBotToken = %q, want %q", cfg.DiscordBotToken, "discord-token")
	}

	if cfg.DiscordAppKey != "discord-public-key" {
		t.Fatalf("DiscordAppKey = %q, want %q", cfg.DiscordAppKey, "discord-public-key")
	}
}

func TestLoadFromEnvGCSValues(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFromEnv(func(key string) (string, bool) {
		switch key {
		case "GCS_BUCKET":
			return "leetdaily-prod", true
		case "CONFIG_OBJECT":
			return "runtime/config.json", true
		case "STATE_OBJECT":
			return "runtime/state.json", true
		case "GUILDS_OBJECT":
			return "runtime/guilds.json", true
		case "PROBLEMS_OBJECT":
			return "runtime/problems.json", true
		default:
			return "", false
		}
	})
	if err != nil {
		t.Fatalf("LoadFromEnv(gcs) returned error: %v", err)
	}

	if !cfg.UsesGCS() {
		t.Fatal("UsesGCS() = false, want true")
	}

	if cfg.GCSBucket != "leetdaily-prod" {
		t.Fatalf("GCSBucket = %q, want %q", cfg.GCSBucket, "leetdaily-prod")
	}

	if cfg.ConfigPath() != "runtime/config.json" {
		t.Fatalf("ConfigPath() = %q, want %q", cfg.ConfigPath(), "runtime/config.json")
	}

	if cfg.StatePath() != "runtime/state.json" {
		t.Fatalf("StatePath() = %q, want %q", cfg.StatePath(), "runtime/state.json")
	}

	if cfg.GuildsPath() != "runtime/guilds.json" {
		t.Fatalf("GuildsPath() = %q, want %q", cfg.GuildsPath(), "runtime/guilds.json")
	}

	if cfg.ProblemsPath() != "runtime/problems.json" {
		t.Fatalf("ProblemsPath() = %q, want %q", cfg.ProblemsPath(), "runtime/problems.json")
	}
}

func TestLoadFromEnvRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "invalid mode",
			env: map[string]string{
				"LEETDAILY_RUNTIME": "worker",
			},
		},
		{
			name: "invalid log level",
			env: map[string]string{
				"LEETDAILY_LOG_LEVEL": "verbose",
			},
		},
		{
			name: "invalid port",
			env: map[string]string{
				"PORT": "99999",
			},
		},
		{
			name: "gcs object without bucket",
			env: map[string]string{
				"STATE_OBJECT": "runtime/state.json",
			},
		},
		{
			name: "guilds object without bucket",
			env: map[string]string{
				"GUILDS_OBJECT": "runtime/guilds.json",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := LoadFromEnv(func(key string) (string, bool) {
				value, ok := tc.env[key]
				return value, ok
			})
			if err == nil {
				t.Fatal("LoadFromEnv() returned nil error, want validation error")
			}
		})
	}
}
