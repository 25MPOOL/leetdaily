package job

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/discord"
	"github.com/nkoji21/leetdaily/internal/problemcache"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
)

const stalePostingAfter = 30 * time.Minute

type Repository interface {
	LoadConfig(context.Context) (config.Config, error)
	LoadGuildSettings(context.Context) (config.GuildSettings, storage.Version, error)
	LoadState(context.Context) (state.State, storage.Version, error)
	SaveState(context.Context, state.State, storage.Version) (storage.Version, error)
	LoadProblemCache(context.Context) (problemcache.Cache, storage.Version, error)
	SaveProblemCache(context.Context, problemcache.Cache, storage.Version) (storage.Version, error)
}

type ForumPoster interface {
	EnsureDifficultyTags(context.Context, string) (map[problemcache.Difficulty]string, error)
	CreateForumThread(context.Context, discord.ForumThreadParams) (discord.Thread, error)
}

type Notifier interface {
	NotifyFailure(context.Context, string, error) error
}

type Sleeper func(context.Context, time.Duration) error

type Options struct {
	Now   func() time.Time
	Sleep Sleeper
}

type Runner struct {
	repository Repository
	fetcher    problemcache.Fetcher
	poster     ForumPoster
	notifier   Notifier
	now        func() time.Time
	sleep      Sleeper
}

func New(repository Repository, fetcher problemcache.Fetcher, poster ForumPoster, notifier Notifier) (*Runner, error) {
	return NewWithOptions(repository, fetcher, poster, notifier, Options{})
}

func NewWithOptions(repository Repository, fetcher problemcache.Fetcher, poster ForumPoster, notifier Notifier, options Options) (*Runner, error) {
	if repository == nil {
		return nil, fmt.Errorf("job repository must not be nil")
	}
	if fetcher == nil {
		return nil, fmt.Errorf("job problem fetcher must not be nil")
	}
	if poster == nil {
		return nil, fmt.Errorf("job forum poster must not be nil")
	}
	if notifier == nil {
		return nil, fmt.Errorf("job notifier must not be nil")
	}

	return &Runner{
		repository: repository,
		fetcher:    fetcher,
		poster:     poster,
		notifier:   notifier,
		now:        coalesceNow(options.Now),
		sleep:      coalesceSleep(options.Sleep),
	}, nil
}

// Now returns the current time using the runner's injectable clock.
func (r *Runner) Now() time.Time {
	return r.now()
}

func coalesceNow(now func() time.Time) func() time.Time {
	if now != nil {
		return now
	}

	return time.Now
}

func coalesceSleep(sleep Sleeper) Sleeper {
	if sleep != nil {
		return sleep
	}

	return func(ctx context.Context, d time.Duration) error {
		timer := time.NewTimer(d)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
}

func (r *Runner) Run(ctx context.Context, targetDate state.Date) error {
	cfg, err := r.repository.LoadConfig(ctx)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	guildSettings, _, err := r.repository.LoadGuildSettings(ctx)
	if err != nil {
		return fmt.Errorf("load guild settings: %w", err)
	}

	currentState, stateVersion, err := r.repository.LoadState(ctx)
	if err != nil {
		if storage.IsNotFound(err) {
			currentState = state.New()
		} else {
			return fmt.Errorf("load state: %w", err)
		}
	}

	cache, cacheVersion, err := r.repository.LoadProblemCache(ctx)
	if err != nil {
		if storage.IsNotFound(err) {
			cache = problemcache.Cache{}
		} else {
			return fmt.Errorf("load problem cache: %w", err)
		}
	}

	var errs []error
	for _, guild := range guildSettings.EnabledGuilds() {
		guildState, _ := currentState.EnsureGuild(guild.GuildID, guild.StartProblemNumber)

		gr := &guildRun{
			runner:       r,
			cfg:          cfg,
			state:        &currentState,
			stateVersion: stateVersion,
			guild:        guild,
			guildState:   guildState,
			targetDate:   targetDate,
		}

		newStateVersion, newCacheVersion, newCache, err := gr.execute(ctx, cache, cacheVersion)
		stateVersion = newStateVersion
		cache = newCache
		cacheVersion = newCacheVersion
		if err != nil {
			errs = append(errs, fmt.Errorf("guild %s: %w", guild.GuildID, err))
		}
	}

	return errors.Join(errs...)
}

// guildRun holds the per-guild execution context for a single Run iteration.
type guildRun struct {
	runner       *Runner
	cfg          config.Config
	state        *state.State
	stateVersion storage.Version
	guild        config.Guild
	guildState   state.GuildState
	targetDate   state.Date
}

func (gr *guildRun) execute(ctx context.Context, cache problemcache.Cache, cacheVersion storage.Version) (storage.Version, storage.Version, problemcache.Cache, error) {
	if shouldSkip(gr.guildState, gr.targetDate, gr.runner.now()) {
		return gr.stateVersion, cacheVersion, cache, nil
	}

	if recovered := recoverStalePosting(gr.guildState, gr.targetDate, gr.runner.now()); recovered.Job.Status != gr.guildState.Job.Status {
		gr.guildState = recovered
		gr.state.GuildStates[gr.guild.GuildID] = gr.guildState
		newVersion, err := gr.runner.repository.SaveState(ctx, *gr.state, gr.stateVersion)
		if err != nil {
			return gr.stateVersion, cacheVersion, cache, fmt.Errorf("save stale recovery state for guild %s: %w", gr.guild.GuildID, err)
		}
		gr.stateVersion = newVersion
	}

	// DifficultyMedium is the intentional cap: Hard problems are excluded by design.
	// To make this configurable, add a MaxDifficulty field to config.Guild.
	refreshedCache, refreshed, refreshErr := problemcache.Refresh(ctx, gr.runner.now(), cache, gr.guildState.NextProblemNumber, gr.cfg.ProblemCache.RefillThreshold, problemcache.DifficultyMedium, gr.runner.fetcher)
	if refreshErr != nil {
		notifyErr := gr.runner.notifier.NotifyFailure(ctx, gr.guild.GuildID, refreshErr)
		if !errors.Is(refreshErr, problemcache.ErrRefillUsedStaleCache) {
			return gr.stateVersion, cacheVersion, cache, errors.Join(refreshErr, notifyErr)
		}
		// Stale cache: continue with existing cache, but surface any notification failure.
		cache = refreshedCache
		if notifyErr != nil {
			return gr.stateVersion, cacheVersion, cache, errors.Join(refreshErr, notifyErr)
		}
	}
	if refreshed {
		cache = refreshedCache
		newVersion, err := gr.runner.repository.SaveProblemCache(ctx, cache, cacheVersion)
		if err != nil {
			return gr.stateVersion, cacheVersion, cache, fmt.Errorf("save refreshed problem cache: %w", err)
		}
		cacheVersion = newVersion
	}

	problem, err := problemcache.SelectNextAtMost(cache, gr.guildState.NextProblemNumber, problemcache.DifficultyMedium) // same cap as Refresh above
	if err != nil {
		notifyErr := gr.runner.notifier.NotifyFailure(ctx, gr.guild.GuildID, err)
		if notifyErr != nil {
			err = errors.Join(err, notifyErr)
		}
		return gr.stateVersion, cacheVersion, cache, err
	}

	newStateVersion, err := gr.post(ctx, problem)
	gr.stateVersion = newStateVersion
	return gr.stateVersion, cacheVersion, cache, err
}

func (gr *guildRun) post(ctx context.Context, problem problemcache.Problem) (storage.Version, error) {
	tags, err := gr.runner.poster.EnsureDifficultyTags(ctx, gr.guild.ForumChannelID)
	if err != nil {
		return gr.recordFailure(ctx, problem.ProblemNumber, gr.cfg.Retry.MaxAttempts, err)
	}

	for attempt := 1; attempt <= gr.cfg.Retry.MaxAttempts; attempt++ {
		startedAt := gr.runner.now()
		gr.guildState.Job = state.JobState{
			TargetDate:       &gr.targetDate,
			Status:           state.JobStatusPosting,
			ProblemNumber:    intPointer(problem.ProblemNumber),
			RetryCount:       attempt - 1,
			PostingStartedAt: &startedAt,
		}
		gr.state.GuildStates[gr.guild.GuildID] = gr.guildState

		newVersion, saveErr := gr.runner.repository.SaveState(ctx, *gr.state, gr.stateVersion)
		if saveErr != nil {
			return gr.stateVersion, fmt.Errorf("save posting state for guild %s: %w", gr.guild.GuildID, saveErr)
		}
		gr.stateVersion = newVersion

		thread, postErr := gr.runner.poster.CreateForumThread(ctx, discord.ForumThreadParams{
			ForumChannelID: gr.guild.ForumChannelID,
			TagID:          tags[problem.Difficulty],
			Title:          formatThreadTitle(problem),
			Body:           formatThreadBody(problem),
		})
		if postErr == nil {
			now := gr.runner.now()
			gr.guildState.LastPostedProblemNumber = intPointer(problem.ProblemNumber)
			gr.guildState.LastPostedAt = &now
			gr.guildState.LastPostedThreadID = &thread.ID
			gr.guildState.NextProblemNumber = problem.ProblemNumber + 1
			gr.guildState.Job = state.JobState{
				TargetDate:    &gr.targetDate,
				Status:        state.JobStatusPosted,
				ProblemNumber: intPointer(problem.ProblemNumber),
				RetryCount:    attempt - 1,
			}
			gr.state.GuildStates[gr.guild.GuildID] = gr.guildState
			newVersion, saveErr = gr.runner.repository.SaveState(ctx, *gr.state, gr.stateVersion)
			if saveErr != nil {
				return gr.stateVersion, fmt.Errorf("save posted state for guild %s: %w", gr.guild.GuildID, saveErr)
			}
			return newVersion, nil
		}

		lastErr := postErr.Error()
		gr.guildState.Job = state.JobState{
			TargetDate:    &gr.targetDate,
			Status:        state.JobStatusFailed,
			ProblemNumber: intPointer(problem.ProblemNumber),
			RetryCount:    attempt,
			LastError:     &lastErr,
		}
		gr.state.GuildStates[gr.guild.GuildID] = gr.guildState
		newVersion, saveErr = gr.runner.repository.SaveState(ctx, *gr.state, gr.stateVersion)
		if saveErr != nil {
			return gr.stateVersion, fmt.Errorf("save failed state for guild %s: %w", gr.guild.GuildID, saveErr)
		}
		gr.stateVersion = newVersion

		if attempt < gr.cfg.Retry.MaxAttempts {
			if sleepErr := gr.runner.sleep(ctx, time.Duration(gr.cfg.Retry.IntervalMinutes)*time.Minute); sleepErr != nil {
				return gr.stateVersion, sleepErr
			}
			continue
		}

		notifyErr := gr.runner.notifier.NotifyFailure(ctx, gr.guild.GuildID, postErr)
		if notifyErr != nil {
			return gr.stateVersion, errors.Join(postErr, notifyErr)
		}
		return gr.stateVersion, postErr
	}

	return gr.stateVersion, nil
}

func (gr *guildRun) recordFailure(ctx context.Context, problemNumber, retryCount int, cause error) (storage.Version, error) {
	lastErr := cause.Error()
	gr.guildState.Job = state.JobState{
		TargetDate:    &gr.targetDate,
		Status:        state.JobStatusFailed,
		ProblemNumber: intPointer(problemNumber),
		RetryCount:    retryCount,
		LastError:     &lastErr,
	}
	gr.state.GuildStates[gr.guild.GuildID] = gr.guildState
	newVersion, err := gr.runner.repository.SaveState(ctx, *gr.state, gr.stateVersion)
	if err != nil {
		return gr.stateVersion, err
	}

	notifyErr := gr.runner.notifier.NotifyFailure(ctx, gr.guild.GuildID, cause)
	if notifyErr != nil {
		return newVersion, errors.Join(cause, notifyErr)
	}
	return newVersion, cause
}

func shouldSkip(guildState state.GuildState, targetDate state.Date, now time.Time) bool {
	if guildState.Job.TargetDate == nil || !guildState.Job.TargetDate.Equal(targetDate.Time) {
		return false
	}

	if guildState.Job.Status == state.JobStatusPosted {
		return true
	}

	if guildState.Job.Status == state.JobStatusPosting && !isStale(guildState.Job.PostingStartedAt, now) {
		return true
	}

	return false
}

func recoverStalePosting(guildState state.GuildState, targetDate state.Date, now time.Time) state.GuildState {
	if guildState.Job.TargetDate == nil || !guildState.Job.TargetDate.Equal(targetDate.Time) {
		return guildState
	}

	if guildState.Job.Status != state.JobStatusPosting || !isStale(guildState.Job.PostingStartedAt, now) {
		return guildState
	}

	guildState.Job = state.JobState{
		TargetDate: &targetDate,
		Status:     state.JobStatusIdle,
	}
	return guildState
}

func isStale(startedAt *time.Time, now time.Time) bool {
	return startedAt != nil && now.Sub(*startedAt) >= stalePostingAfter
}

func formatThreadTitle(problem problemcache.Problem) string {
	return fmt.Sprintf("#N%d %s", problem.ProblemNumber, problem.Title)
}

func formatThreadBody(problem problemcache.Problem) string {
	if problem.Category != "" {
		return fmt.Sprintf("%s\n%s", problem.URL(), problem.Category)
	}
	return problem.URL()
}

func intPointer(value int) *int {
	return &value
}
