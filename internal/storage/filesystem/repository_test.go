package filesystem

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
	"github.com/nkoji21/leetdaily/internal/storage/repositorytest"
)

func TestRepositorySuite(t *testing.T) {
	t.Parallel()

	repositorytest.RunRepositorySuite(t, "filesystem", func(t *testing.T) repositorytest.Harness {
		repository, paths := newTestRepository(t)
		return repositorytest.Harness{
			Repository: repository,
			SeedConfig: func(tb testing.TB, cfg config.Config) {
				tb.Helper()
				writeJSON(tb, paths.ConfigPath, cfg)
			},
		}
	})
}

func TestRepositorySaveUsesAtomicRenameWithoutLeavingTempFiles(t *testing.T) {
	t.Parallel()

	repository, paths := newTestRepository(t)

	if _, err := repository.SaveState(context.Background(), state.New(), storage.Version{}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	data, err := os.ReadFile(paths.StatePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", paths.StatePath, err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("saved state JSON is invalid: %v", err)
	}

	matches, err := filepath.Glob(paths.StatePath + ".*.tmp")
	if err != nil {
		t.Fatalf("filepath.Glob() error = %v", err)
	}

	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}

func newTestRepository(t *testing.T) (*Repository, storage.Paths) {
	t.Helper()

	root := t.TempDir()
	paths := storage.Paths{
		ConfigPath:   filepath.Join(root, "config.json"),
		StatePath:    filepath.Join(root, "state.json"),
		ProblemsPath: filepath.Join(root, "problems.json"),
	}

	repository, err := New(paths)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return repository, paths
}

func writeJSON(tb testing.TB, path string, value any) {
	tb.Helper()

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		tb.Fatalf("json.MarshalIndent() error = %v", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		tb.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}
