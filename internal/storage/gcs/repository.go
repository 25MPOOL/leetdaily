package gcs

import (
	"context"
	"fmt"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/problemcache"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
)

type ReadObjectResult struct {
	Data       []byte
	Generation int64
}

type WriteObjectOptions struct {
	DoesNotExist    bool
	MatchGeneration int64
}

type ObjectClient interface {
	ReadObject(context.Context, string, string) (ReadObjectResult, error)
	WriteObject(context.Context, string, string, []byte, WriteObjectOptions) (int64, error)
}

type Repository struct {
	client ObjectClient
	bucket string
	paths  storage.Paths
}

func New(client ObjectClient, bucket string, paths storage.Paths) (*Repository, error) {
	if client == nil {
		return nil, fmt.Errorf("GCS client must not be nil")
	}

	if err := paths.Validate(); err != nil {
		return nil, err
	}

	if bucket == "" {
		return nil, fmt.Errorf("GCS bucket must not be empty")
	}

	return &Repository{
		client: client,
		bucket: bucket,
		paths:  paths,
	}, nil
}

func (r *Repository) LoadConfig(ctx context.Context) (config.Config, error) {
	data, _, err := r.readObject(ctx, r.paths.ConfigPath)
	if err != nil {
		return config.Config{}, err
	}

	var cfg config.Config
	if err := storage.DecodeJSON(r.paths.ConfigPath, data, &cfg); err != nil {
		return config.Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return config.Config{}, fmt.Errorf("validate config %q: %w", r.paths.ConfigPath, err)
	}

	return cfg, nil
}

func (r *Repository) LoadGuildSettings(ctx context.Context) (config.GuildSettings, storage.Version, error) {
	data, version, err := r.readObject(ctx, r.paths.GuildsPath)
	if err != nil {
		if !storage.IsNotFound(err) {
			return config.GuildSettings{}, storage.Version{}, err
		}

		cfg, cfgErr := r.LoadConfig(ctx)
		if cfgErr != nil {
			return config.GuildSettings{}, storage.Version{}, err
		}
		return config.GuildSettings{Guilds: append([]config.Guild(nil), cfg.Guilds...)}, storage.Version{}, nil
	}

	var guilds config.GuildSettings
	if err := storage.DecodeJSON(r.paths.GuildsPath, data, &guilds); err != nil {
		return config.GuildSettings{}, storage.Version{}, err
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
	if guilds.Guilds == nil {
		guilds.Guilds = []config.Guild{}
	}

	if err := guilds.Validate(); err != nil {
		return storage.Version{}, fmt.Errorf("validate guild settings before save: %w", err)
	}

	return r.writeObject(ctx, r.paths.GuildsPath, guilds, version)
}

func (r *Repository) LoadState(ctx context.Context) (state.State, storage.Version, error) {
	data, version, err := r.readObject(ctx, r.paths.StatePath)
	if err != nil {
		return state.State{}, storage.Version{}, err
	}

	var current state.State
	if err := storage.DecodeJSON(r.paths.StatePath, data, &current); err != nil {
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
	if current.GuildStates == nil {
		current.GuildStates = map[string]state.GuildState{}
	}

	if err := current.Validate(); err != nil {
		return storage.Version{}, fmt.Errorf("validate state before save: %w", err)
	}

	return r.writeObject(ctx, r.paths.StatePath, current, version)
}

func (r *Repository) LoadProblemCache(ctx context.Context) (problemcache.Cache, storage.Version, error) {
	data, version, err := r.readObject(ctx, r.paths.ProblemsPath)
	if err != nil {
		return problemcache.Cache{}, storage.Version{}, err
	}

	var cache problemcache.Cache
	if err := storage.DecodeJSON(r.paths.ProblemsPath, data, &cache); err != nil {
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
	if cache.Problems == nil {
		cache.Problems = []problemcache.Problem{}
	}

	if err := cache.Validate(); err != nil {
		return storage.Version{}, fmt.Errorf("validate problem cache before save: %w", err)
	}

	return r.writeObject(ctx, r.paths.ProblemsPath, cache, version)
}

func (r *Repository) readObject(ctx context.Context, object string) ([]byte, storage.Version, error) {
	result, err := r.client.ReadObject(ctx, r.bucket, object)
	if err != nil {
		return nil, storage.Version{}, normalizeError(err, object)
	}

	return result.Data, versionFromGeneration(result.Generation), nil
}

func (r *Repository) writeObject(ctx context.Context, object string, value any, version storage.Version) (storage.Version, error) {
	data, err := storage.EncodeJSON(object, value)
	if err != nil {
		return storage.Version{}, err
	}

	generation, err := r.client.WriteObject(ctx, r.bucket, object, data, writeOptionsFromVersion(version))
	if err != nil {
		return storage.Version{}, normalizeError(err, object)
	}

	return versionFromGeneration(generation), nil
}
