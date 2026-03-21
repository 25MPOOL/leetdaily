package filesystem

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/problemcache"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
)

type Repository struct {
	paths storage.Paths
}

func New(paths storage.Paths) (*Repository, error) {
	if err := paths.Validate(); err != nil {
		return nil, err
	}

	return &Repository{paths: paths}, nil
}

func (r *Repository) LoadConfig(ctx context.Context) (config.Config, error) {
	if err := checkContext(ctx); err != nil {
		return config.Config{}, err
	}

	var cfg config.Config
	if _, err := readJSONFile(r.paths.ConfigPath, &cfg); err != nil {
		return config.Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return config.Config{}, fmt.Errorf("validate config %q: %w", r.paths.ConfigPath, err)
	}

	return cfg, nil
}

func (r *Repository) LoadGuildSettings(ctx context.Context) (config.GuildSettings, storage.Version, error) {
	if err := checkContext(ctx); err != nil {
		return config.GuildSettings{}, storage.Version{}, err
	}

	var guilds config.GuildSettings
	version, err := readJSONFile(r.paths.GuildsPath, &guilds)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return config.GuildSettings{}, storage.Version{}, err
		}

		cfg, cfgErr := r.LoadConfig(ctx)
		if cfgErr != nil {
			return config.GuildSettings{}, storage.Version{}, err
		}
		return config.GuildSettings{Guilds: append([]config.Guild(nil), cfg.Guilds...)}, storage.Version{}, nil
	}

	if guilds.Guilds == nil {
		guilds.Guilds = []config.Guild{}
	}

	if err := guilds.Validate(); err != nil {
		return config.GuildSettings{}, storage.Version{}, fmt.Errorf("validate guild settings %q: %w", r.paths.GuildsPath, err)
	}

	return guilds, version, nil
}

func (r *Repository) SaveGuildSettings(ctx context.Context, guilds config.GuildSettings, version storage.Version) (storage.Version, error) {
	if err := checkContext(ctx); err != nil {
		return storage.Version{}, err
	}

	if guilds.Guilds == nil {
		guilds.Guilds = []config.Guild{}
	}

	if err := guilds.Validate(); err != nil {
		return storage.Version{}, fmt.Errorf("validate guild settings before save: %w", err)
	}

	return writeJSONWithVersion(r.paths.GuildsPath, guilds, version)
}

func (r *Repository) LoadState(ctx context.Context) (state.State, storage.Version, error) {
	if err := checkContext(ctx); err != nil {
		return state.State{}, storage.Version{}, err
	}

	var current state.State
	version, err := readJSONFile(r.paths.StatePath, &current)
	if err != nil {
		return state.State{}, storage.Version{}, err
	}

	if current.GuildStates == nil {
		current.GuildStates = map[string]state.GuildState{}
	}

	if err := current.Validate(); err != nil {
		return state.State{}, storage.Version{}, fmt.Errorf("validate state %q: %w", r.paths.StatePath, err)
	}

	return current, version, nil
}

func (r *Repository) SaveState(ctx context.Context, current state.State, version storage.Version) (storage.Version, error) {
	if err := checkContext(ctx); err != nil {
		return storage.Version{}, err
	}

	if current.GuildStates == nil {
		current.GuildStates = map[string]state.GuildState{}
	}

	if err := current.Validate(); err != nil {
		return storage.Version{}, fmt.Errorf("validate state before save: %w", err)
	}

	return writeJSONWithVersion(r.paths.StatePath, current, version)
}

func (r *Repository) LoadProblemCache(ctx context.Context) (problemcache.Cache, storage.Version, error) {
	if err := checkContext(ctx); err != nil {
		return problemcache.Cache{}, storage.Version{}, err
	}

	var cache problemcache.Cache
	version, err := readJSONFile(r.paths.ProblemsPath, &cache)
	if err != nil {
		return problemcache.Cache{}, storage.Version{}, err
	}

	if cache.Problems == nil {
		cache.Problems = []problemcache.Problem{}
	}

	if err := cache.Validate(); err != nil {
		return problemcache.Cache{}, storage.Version{}, fmt.Errorf("validate problem cache %q: %w", r.paths.ProblemsPath, err)
	}

	return cache, version, nil
}

func (r *Repository) SaveProblemCache(ctx context.Context, cache problemcache.Cache, version storage.Version) (storage.Version, error) {
	if err := checkContext(ctx); err != nil {
		return storage.Version{}, err
	}

	if cache.Problems == nil {
		cache.Problems = []problemcache.Problem{}
	}

	if err := cache.Validate(); err != nil {
		return storage.Version{}, fmt.Errorf("validate problem cache before save: %w", err)
	}

	return writeJSONWithVersion(r.paths.ProblemsPath, cache, version)
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}

	return ctx.Err()
}

func readJSONFile(path string, destination any) (storage.Version, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return storage.Version{}, fmt.Errorf("%w: %s", storage.ErrNotFound, path)
		}

		return storage.Version{}, fmt.Errorf("read %q: %w", path, err)
	}

	if err := storage.DecodeJSON(path, data, destination); err != nil {
		return storage.Version{}, err
	}

	return storage.VersionFromBytes(data), nil
}

func writeJSONWithVersion(path string, value any, expected storage.Version) (storage.Version, error) {
	data, err := storage.EncodeJSON(path, value)
	if err != nil {
		return storage.Version{}, err
	}

	currentData, err := os.ReadFile(path)
	switch {
	case err == nil:
		currentVersion := storage.VersionFromBytes(currentData)
		if expected.IsZero() || currentVersion != expected {
			return storage.Version{}, fmt.Errorf("%w: %s", storage.ErrConflict, path)
		}
	case errors.Is(err, os.ErrNotExist):
		if !expected.IsZero() {
			return storage.Version{}, fmt.Errorf("%w: %s", storage.ErrConflict, path)
		}
	default:
		return storage.Version{}, fmt.Errorf("read %q before save: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return storage.Version{}, fmt.Errorf("create parent directory for %q: %w", path, err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return storage.Version{}, fmt.Errorf("create temp file for %q: %w", path, err)
	}

	tempPath := tempFile.Name()
	keepTemp := true
	defer func() {
		if keepTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return storage.Version{}, fmt.Errorf("write temp file for %q: %w", path, err)
	}

	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return storage.Version{}, fmt.Errorf("sync temp file for %q: %w", path, err)
	}

	if err := tempFile.Close(); err != nil {
		return storage.Version{}, fmt.Errorf("close temp file for %q: %w", path, err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		return storage.Version{}, fmt.Errorf("rename temp file for %q: %w", path, err)
	}

	keepTemp = false
	return storage.VersionFromBytes(data), nil
}
