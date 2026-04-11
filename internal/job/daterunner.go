package job

import (
	"context"
	"time"

	"github.com/nkoji21/leetdaily/internal/state"
)

// DateRunner wraps Runner and resolves today's date in the given location
// before delegating to Runner.Run. It is used in job mode where the caller
// does not supply an explicit date.
type DateRunner struct {
	runner   *Runner
	location *time.Location
}

func NewDateRunner(runner *Runner, location *time.Location) *DateRunner {
	return &DateRunner{runner: runner, location: location}
}

func (r *DateRunner) Run(ctx context.Context) error {
	targetDate, err := state.ParseDate(time.Now().In(r.location).Format("2006-01-02"))
	if err != nil {
		return err
	}

	return r.runner.Run(ctx, targetDate)
}
