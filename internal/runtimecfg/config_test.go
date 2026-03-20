package runtimecfg

import (
	"log/slog"
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

	if cfg.StatePath() != "state.json" {
		t.Fatalf("StatePath() = %q, want %q", cfg.StatePath(), "state.json")
	}

	if cfg.ProblemsPath() != "problems.json" {
		t.Fatalf("ProblemsPath() = %q, want %q", cfg.ProblemsPath(), "problems.json")
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

	if cfg.DataDir != "var/data" {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, "var/data")
	}

	if cfg.ConfigPath() != "var/data/config.json" {
		t.Fatalf("ConfigPath() = %q, want %q", cfg.ConfigPath(), "var/data/config.json")
	}

	if cfg.HTTPAddr() != ":9090" {
		t.Fatalf("HTTPAddr() = %q, want %q", cfg.HTTPAddr(), ":9090")
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
