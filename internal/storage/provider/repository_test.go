package provider

import (
	"context"
	"testing"

	"github.com/nkoji21/leetdaily/internal/runtimecfg"
	"github.com/nkoji21/leetdaily/internal/storage/filesystem"
	gcsrepo "github.com/nkoji21/leetdaily/internal/storage/gcs"
)

type noopGCSClient struct{}

func (noopGCSClient) ReadObject(context.Context, string, string) (gcsrepo.ReadObjectResult, error) {
	return gcsrepo.ReadObjectResult{}, nil
}

func (noopGCSClient) WriteObject(context.Context, string, string, []byte, gcsrepo.WriteObjectOptions) (int64, error) {
	return 1, nil
}

func TestNewRepositoryReturnsFilesystemBackendWithoutGCSConfig(t *testing.T) {
	t.Parallel()

	repository, err := NewRepository(context.Background(), runtimecfg.Config{
		DataDir:  ".",
		HTTPPort: 8080,
		Mode:     runtimecfg.ModeHTTP,
	})
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	if _, ok := repository.(*filesystem.Repository); !ok {
		t.Fatalf("NewRepository() type = %T, want *filesystem.Repository", repository)
	}
}

func TestNewRepositoryWithGCSClientReturnsGCSBackend(t *testing.T) {
	t.Parallel()

	repository, err := NewRepositoryWithGCSClient(runtimecfg.Config{
		Mode:           runtimecfg.ModeHTTP,
		HTTPPort:       8080,
		DataDir:        ".",
		GCSBucket:      "leetdaily-prod",
		ConfigObject:   "config.json",
		StateObject:    "state.json",
		ProblemsObject: "problems.json",
	}, noopGCSClient{})
	if err != nil {
		t.Fatalf("NewRepositoryWithGCSClient() error = %v", err)
	}

	if _, ok := repository.(*gcsrepo.Repository); !ok {
		t.Fatalf("NewRepositoryWithGCSClient() type = %T, want *gcs.Repository", repository)
	}
}
