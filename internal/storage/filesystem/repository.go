package filesystem

import (
	"context"
	"encoding/json"
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
	if err := readJSONFile(r.paths.ConfigPath, &cfg); err != nil {
		return config.Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return config.Config{}, fmt.Errorf("validate config %q: %w", r.paths.ConfigPath, err)
	}

	return cfg, nil
}

func (r *Repository) LoadState(ctx context.Context) (state.State, error) {
	if err := checkContext(ctx); err != nil {
		return state.State{}, err
	}

	var current state.State
	if err := readJSONFile(r.paths.StatePath, &current); err != nil {
		return state.State{}, err
	}

	if current.GuildStates == nil {
		current.GuildStates = map[string]state.GuildState{}
	}

	if err := current.Validate(); err != nil {
		return state.State{}, fmt.Errorf("validate state %q: %w", r.paths.StatePath, err)
	}

	return current, nil
}

func (r *Repository) SaveState(ctx context.Context, current state.State) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	if current.GuildStates == nil {
		current.GuildStates = map[string]state.GuildState{}
	}

	if err := current.Validate(); err != nil {
		return fmt.Errorf("validate state before save: %w", err)
	}

	return writeJSONAtomic(r.paths.StatePath, current)
}

func (r *Repository) LoadProblemCache(ctx context.Context) (problemcache.Cache, error) {
	if err := checkContext(ctx); err != nil {
		return problemcache.Cache{}, err
	}

	var cache problemcache.Cache
	if err := readJSONFile(r.paths.ProblemsPath, &cache); err != nil {
		return problemcache.Cache{}, err
	}

	if cache.Problems == nil {
		cache.Problems = []problemcache.Problem{}
	}

	if err := cache.Validate(); err != nil {
		return problemcache.Cache{}, fmt.Errorf("validate problem cache %q: %w", r.paths.ProblemsPath, err)
	}

	return cache, nil
}

func (r *Repository) SaveProblemCache(ctx context.Context, cache problemcache.Cache) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	if cache.Problems == nil {
		cache.Problems = []problemcache.Problem{}
	}

	if err := cache.Validate(); err != nil {
		return fmt.Errorf("validate problem cache before save: %w", err)
	}

	return writeJSONAtomic(r.paths.ProblemsPath, cache)
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}

	return ctx.Err()
}

func readJSONFile(path string, destination any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", storage.ErrNotFound, path)
		}

		return fmt.Errorf("read %q: %w", path, err)
	}

	if err := json.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("decode %q: %w", path, err)
	}

	return nil
}

func writeJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %q: %w", path, err)
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %q: %w", path, err)
	}
	data = append(data, '\n')

	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file for %q: %w", path, err)
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
		return fmt.Errorf("write temp file for %q: %w", path, err)
	}

	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("sync temp file for %q: %w", path, err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp file for %q: %w", path, err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("rename temp file for %q: %w", path, err)
	}

	keepTemp = false
	return nil
}
