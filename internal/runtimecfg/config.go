package runtimecfg

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type LookupEnv func(string) (string, bool)

type Mode string

const (
	ModeHTTP Mode = "http"
	ModeJob  Mode = "job"
)

type Config struct {
	Mode            Mode
	LogLevel        slog.Level
	HTTPPort        int
	DataDir         string
	DiscordBotToken string
	DiscordAppKey   string
	GCSBucket       string
	ConfigObject    string
	GuildsObject    string
	StateObject     string
	ProblemsObject  string
}

func Load() (Config, error) {
	return LoadFromEnv(os.LookupEnv)
}

func LoadFromEnv(lookup LookupEnv) (Config, error) {
	if lookup == nil {
		lookup = func(string) (string, bool) {
			return "", false
		}
	}

	cfg := Config{
		Mode:           ModeHTTP,
		LogLevel:       slog.LevelInfo,
		HTTPPort:       8080,
		DataDir:        ".",
		ConfigObject:   "config.json",
		GuildsObject:   "guilds.json",
		StateObject:    "state.json",
		ProblemsObject: "problems.json",
	}

	if raw, ok := lookup("LEETDAILY_RUNTIME"); ok && strings.TrimSpace(raw) != "" {
		cfg.Mode = Mode(strings.ToLower(strings.TrimSpace(raw)))
	}

	if raw, ok := lookup("LEETDAILY_LOG_LEVEL"); ok && strings.TrimSpace(raw) != "" {
		level, err := parseLogLevel(raw)
		if err != nil {
			return Config{}, err
		}
		cfg.LogLevel = level
	}

	if raw, ok := lookup("PORT"); ok && strings.TrimSpace(raw) != "" {
		port, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return Config{}, fmt.Errorf("parse PORT: %w", err)
		}
		cfg.HTTPPort = port
	}

	if raw, ok := lookup("LEETDAILY_DATA_DIR"); ok && strings.TrimSpace(raw) != "" {
		cfg.DataDir = filepath.Clean(strings.TrimSpace(raw))
	}

	if raw, ok := lookup("DISCORD_BOT_TOKEN"); ok && strings.TrimSpace(raw) != "" {
		cfg.DiscordBotToken = strings.TrimSpace(raw)
	}

	if raw, ok := lookup("DISCORD_APPLICATION_PUBLIC_KEY"); ok && strings.TrimSpace(raw) != "" {
		cfg.DiscordAppKey = strings.TrimSpace(raw)
	}

	if raw, ok := lookup("GCS_BUCKET"); ok && strings.TrimSpace(raw) != "" {
		cfg.GCSBucket = strings.TrimSpace(raw)
	}

	if raw, ok := lookup("CONFIG_OBJECT"); ok && strings.TrimSpace(raw) != "" {
		cfg.ConfigObject = strings.TrimSpace(raw)
	}

	if raw, ok := lookup("GUILDS_OBJECT"); ok && strings.TrimSpace(raw) != "" {
		cfg.GuildsObject = strings.TrimSpace(raw)
	}

	if raw, ok := lookup("STATE_OBJECT"); ok && strings.TrimSpace(raw) != "" {
		cfg.StateObject = strings.TrimSpace(raw)
	}

	if raw, ok := lookup("PROBLEMS_OBJECT"); ok && strings.TrimSpace(raw) != "" {
		cfg.ProblemsObject = strings.TrimSpace(raw)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	switch c.Mode {
	case ModeHTTP, ModeJob:
	default:
		return fmt.Errorf("unsupported LEETDAILY_RUNTIME %q", c.Mode)
	}

	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535: %d", c.HTTPPort)
	}

	if strings.TrimSpace(c.DataDir) == "" {
		return fmt.Errorf("LEETDAILY_DATA_DIR must not be empty")
	}

	if c.UsesGCS() {
		if strings.TrimSpace(c.ConfigObject) == "" {
			return fmt.Errorf("CONFIG_OBJECT must not be empty when GCS_BUCKET is set")
		}
		if strings.TrimSpace(c.GuildsObject) == "" {
			return fmt.Errorf("GUILDS_OBJECT must not be empty when GCS_BUCKET is set")
		}
		if strings.TrimSpace(c.StateObject) == "" {
			return fmt.Errorf("STATE_OBJECT must not be empty when GCS_BUCKET is set")
		}
		if strings.TrimSpace(c.ProblemsObject) == "" {
			return fmt.Errorf("PROBLEMS_OBJECT must not be empty when GCS_BUCKET is set")
		}
		return nil
	}

	if strings.TrimSpace(c.GCSBucket) == "" {
		if strings.TrimSpace(c.ConfigObject) != "config.json" {
			return fmt.Errorf("CONFIG_OBJECT requires GCS_BUCKET")
		}
		if strings.TrimSpace(c.GuildsObject) != "guilds.json" {
			return fmt.Errorf("GUILDS_OBJECT requires GCS_BUCKET")
		}
		if strings.TrimSpace(c.StateObject) != "state.json" {
			return fmt.Errorf("STATE_OBJECT requires GCS_BUCKET")
		}
		if strings.TrimSpace(c.ProblemsObject) != "problems.json" {
			return fmt.Errorf("PROBLEMS_OBJECT requires GCS_BUCKET")
		}
	}

	return nil
}

func (c Config) HTTPAddr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}

func (c Config) ConfigPath() string {
	if c.UsesGCS() {
		return c.ConfigObject
	}

	return filepath.Join(c.DataDir, "config.json")
}

func (c Config) StatePath() string {
	if c.UsesGCS() {
		return c.StateObject
	}

	return filepath.Join(c.DataDir, "state.json")
}

func (c Config) GuildsPath() string {
	if c.UsesGCS() {
		return c.GuildsObject
	}

	return filepath.Join(c.DataDir, "guilds.json")
}

func (c Config) ProblemsPath() string {
	if c.UsesGCS() {
		return c.ProblemsObject
	}

	return filepath.Join(c.DataDir, "problems.json")
}

func (c Config) UsesGCS() bool {
	return strings.TrimSpace(c.GCSBucket) != ""
}

func parseLogLevel(raw string) (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(raw)))); err != nil {
		return 0, fmt.Errorf("parse LEETDAILY_LOG_LEVEL: %w", err)
	}

	return level, nil
}
