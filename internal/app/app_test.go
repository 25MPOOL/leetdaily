package app

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/nkoji21/leetdaily/internal/runtimecfg"
)

func TestAppRunSelectsHTTPRunner(t *testing.T) {
	t.Parallel()

	httpRunner := &stubRunner{}
	jobRunner := &stubRunner{}

	cfg := runtimecfg.Config{
		Mode:     runtimecfg.ModeHTTP,
		LogLevel: slog.LevelInfo,
		HTTPPort: 8080,
		DataDir:  ".",
	}

	err := New(cfg, nil, Dependencies{
		HTTPRunner: httpRunner,
		JobRunner:  jobRunner,
	}).Run(context.Background())
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if httpRunner.calls != 1 {
		t.Fatalf("http runner calls = %d, want 1", httpRunner.calls)
	}

	if jobRunner.calls != 0 {
		t.Fatalf("job runner calls = %d, want 0", jobRunner.calls)
	}
}

func TestAppRunSelectsJobRunner(t *testing.T) {
	t.Parallel()

	httpRunner := &stubRunner{}
	jobRunner := &stubRunner{}

	cfg := runtimecfg.Config{
		Mode:     runtimecfg.ModeJob,
		LogLevel: slog.LevelInfo,
		HTTPPort: 8080,
		DataDir:  ".",
	}

	err := New(cfg, nil, Dependencies{
		HTTPRunner: httpRunner,
		JobRunner:  jobRunner,
	}).Run(context.Background())
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if httpRunner.calls != 0 {
		t.Fatalf("http runner calls = %d, want 0", httpRunner.calls)
	}

	if jobRunner.calls != 1 {
		t.Fatalf("job runner calls = %d, want 1", jobRunner.calls)
	}
}

func TestAppRunPropagatesRunnerError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	jobRunner := &stubRunner{err: wantErr}

	cfg := runtimecfg.Config{
		Mode:     runtimecfg.ModeJob,
		LogLevel: slog.LevelInfo,
		HTTPPort: 8080,
		DataDir:  ".",
	}

	err := New(cfg, nil, Dependencies{
		JobRunner: jobRunner,
	}).Run(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
}

type stubRunner struct {
	calls int
	err   error
}

func (r *stubRunner) Run(context.Context) error {
	r.calls++
	return r.err
}
