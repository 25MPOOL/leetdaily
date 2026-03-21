package provider

import (
	"context"
	"fmt"

	"github.com/nkoji21/leetdaily/internal/runtimecfg"
	"github.com/nkoji21/leetdaily/internal/storage"
	"github.com/nkoji21/leetdaily/internal/storage/filesystem"
	gcsrepo "github.com/nkoji21/leetdaily/internal/storage/gcs"
)

func NewRepository(ctx context.Context, cfg runtimecfg.Config) (storage.Repository, error) {
	if cfg.UsesGCS() {
		client, err := gcsrepo.NewGoogleClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("create GCS client: %w", err)
		}

		return NewRepositoryWithGCSClient(cfg, client)
	}

	return filesystem.New(storage.Paths{
		ConfigPath:   cfg.ConfigPath(),
		GuildsPath:   cfg.GuildsPath(),
		StatePath:    cfg.StatePath(),
		ProblemsPath: cfg.ProblemsPath(),
	})
}

func NewRepositoryWithGCSClient(cfg runtimecfg.Config, client gcsrepo.ObjectClient) (storage.Repository, error) {
	if !cfg.UsesGCS() {
		return nil, fmt.Errorf("GCS client requires GCS-backed runtime configuration")
	}

	return gcsrepo.New(client, cfg.GCSBucket, storage.Paths{
		ConfigPath:   cfg.ConfigPath(),
		GuildsPath:   cfg.GuildsPath(),
		StatePath:    cfg.StatePath(),
		ProblemsPath: cfg.ProblemsPath(),
	})
}
