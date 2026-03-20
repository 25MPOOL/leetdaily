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
	Mode     Mode
	LogLevel slog.Level
	HTTPPort int
	DataDir  string
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
		Mode:     ModeHTTP,
		LogLevel: slog.LevelInfo,
		HTTPPort: 8080,
		DataDir:  ".",
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

	return nil
}

func (c Config) HTTPAddr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}

func (c Config) ConfigPath() string {
	return filepath.Join(c.DataDir, "config.json")
}

func (c Config) StatePath() string {
	return filepath.Join(c.DataDir, "state.json")
}

func (c Config) ProblemsPath() string {
	return filepath.Join(c.DataDir, "problems.json")
}

func parseLogLevel(raw string) (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(raw)))); err != nil {
		return 0, fmt.Errorf("parse LEETDAILY_LOG_LEVEL: %w", err)
	}

	return level, nil
}
