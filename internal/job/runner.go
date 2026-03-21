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
	CreateForumThread(context.Context, string, string, string, string) (discord.Thread, error)
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

	for _, guild := range guildSettings.EnabledGuilds() {
		guildState, _ := currentState.EnsureGuild(guild.GuildID, guild.StartProblemNumber)

		if shouldSkip(guildState, targetDate, r.now()) {
			continue
		}

		if recovered := recoverStalePosting(guildState, targetDate, r.now()); recovered.Job.Status != guildState.Job.Status {
			guildState = recovered
			currentState.GuildStates[guild.GuildID] = guildState
			stateVersion, err = r.repository.SaveState(ctx, currentState, stateVersion)
			if err != nil {
				return fmt.Errorf("save stale recovery state for guild %s: %w", guild.GuildID, err)
			}
		}

		refreshedCache, refreshed, refreshErr := problemcache.Refresh(ctx, r.now(), cache, guildState.NextProblemNumber, cfg.ProblemCache.RefillThreshold, r.fetcher)
		if refreshErr != nil {
			if errors.Is(refreshErr, problemcache.ErrRefillUsedStaleCache) {
				cache = refreshedCache
				notifyErr := r.notifier.NotifyFailure(ctx, guild.GuildID, refreshErr)
				if notifyErr != nil {
					refreshErr = errors.Join(refreshErr, notifyErr)
				}
			} else {
				notifyErr := r.notifier.NotifyFailure(ctx, guild.GuildID, refreshErr)
				if notifyErr != nil {
					refreshErr = errors.Join(refreshErr, notifyErr)
				}
				continue
			}
		}
		if refreshed {
			cache = refreshedCache
			cacheVersion, err = r.repository.SaveProblemCache(ctx, cache, cacheVersion)
			if err != nil {
				return fmt.Errorf("save refreshed problem cache: %w", err)
			}
		}

		problem, err := problemcache.SelectNextFree(cache, guildState.NextProblemNumber)
		if err != nil {
			notifyErr := r.notifier.NotifyFailure(ctx, guild.GuildID, err)
			if notifyErr != nil {
				err = errors.Join(err, notifyErr)
			}
			continue
		}

		guildState, stateVersion, err = r.processGuild(ctx, cfg, currentState, stateVersion, guild, guildState, targetDate, problem)
		currentState.GuildStates[guild.GuildID] = guildState
		if err != nil {
			continue
		}
	}

	return nil
}

func (r *Runner) processGuild(
	ctx context.Context,
	cfg config.Config,
	currentState state.State,
	stateVersion storage.Version,
	guild config.Guild,
	guildState state.GuildState,
	targetDate state.Date,
	problem problemcache.Problem,
) (state.GuildState, storage.Version, error) {
	tags, err := r.poster.EnsureDifficultyTags(ctx, guild.ForumChannelID)
	if err != nil {
		return r.failGuild(ctx, currentState, stateVersion, guild, guildState, targetDate, problem.ProblemNumber, cfg.Retry.MaxAttempts, err)
	}

	for attempt := 1; attempt <= cfg.Retry.MaxAttempts; attempt++ {
		startedAt := r.now()
		guildState.Job = state.JobState{
			TargetDate:       &targetDate,
			Status:           state.JobStatusPosting,
			ProblemNumber:    intPointer(problem.ProblemNumber),
			RetryCount:       attempt - 1,
			PostingStartedAt: &startedAt,
		}
		currentState.GuildStates[guild.GuildID] = guildState

		stateVersion, err = r.repository.SaveState(ctx, currentState, stateVersion)
		if err != nil {
			return guildState, stateVersion, fmt.Errorf("save posting state for guild %s: %w", guild.GuildID, err)
		}

		thread, err := r.poster.CreateForumThread(ctx, guild.ForumChannelID, tags[problem.Difficulty], formatThreadTitle(problem), formatThreadBody(problem))
		if err == nil {
			now := r.now()
			guildState.LastPostedProblemNumber = intPointer(problem.ProblemNumber)
			guildState.LastPostedAt = &now
			guildState.LastPostedThreadID = &thread.ID
			guildState.NextProblemNumber = problem.ProblemNumber + 1
			guildState.Job = state.JobState{
				TargetDate:    &targetDate,
				Status:        state.JobStatusPosted,
				ProblemNumber: intPointer(problem.ProblemNumber),
				RetryCount:    attempt - 1,
			}
			currentState.GuildStates[guild.GuildID] = guildState
			stateVersion, err = r.repository.SaveState(ctx, currentState, stateVersion)
			if err != nil {
				return guildState, stateVersion, fmt.Errorf("save posted state for guild %s: %w", guild.GuildID, err)
			}
			return guildState, stateVersion, nil
		}

		lastErr := err.Error()
		guildState.Job = state.JobState{
			TargetDate:    &targetDate,
			Status:        state.JobStatusFailed,
			ProblemNumber: intPointer(problem.ProblemNumber),
			RetryCount:    attempt,
			LastError:     &lastErr,
		}
		currentState.GuildStates[guild.GuildID] = guildState
		var saveErr error
		stateVersion, saveErr = r.repository.SaveState(ctx, currentState, stateVersion)
		if saveErr != nil {
			return guildState, stateVersion, fmt.Errorf("save failed state for guild %s: %w", guild.GuildID, saveErr)
		}

		if attempt < cfg.Retry.MaxAttempts {
			if sleepErr := r.sleep(ctx, time.Duration(cfg.Retry.IntervalMinutes)*time.Minute); sleepErr != nil {
				return guildState, stateVersion, sleepErr
			}
			continue
		}

		notifyErr := r.notifier.NotifyFailure(ctx, guild.GuildID, err)
		if notifyErr != nil {
			return guildState, stateVersion, errors.Join(err, notifyErr)
		}
		return guildState, stateVersion, err
	}

	return guildState, stateVersion, nil
}

func (r *Runner) failGuild(
	ctx context.Context,
	currentState state.State,
	stateVersion storage.Version,
	guild config.Guild,
	guildState state.GuildState,
	targetDate state.Date,
	problemNumber int,
	retryCount int,
	cause error,
) (state.GuildState, storage.Version, error) {
	lastErr := cause.Error()
	guildState.Job = state.JobState{
		TargetDate:    &targetDate,
		Status:        state.JobStatusFailed,
		ProblemNumber: intPointer(problemNumber),
		RetryCount:    retryCount,
		LastError:     &lastErr,
	}
	currentState.GuildStates[guild.GuildID] = guildState
	var err error
	stateVersion, err = r.repository.SaveState(ctx, currentState, stateVersion)
	if err != nil {
		return guildState, stateVersion, err
	}

	notifyErr := r.notifier.NotifyFailure(ctx, guild.GuildID, cause)
	if notifyErr != nil {
		return guildState, stateVersion, errors.Join(cause, notifyErr)
	}
	return guildState, stateVersion, cause
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
	return fmt.Sprintf("#%d %s", problem.ProblemNumber, problem.Title)
}

func formatThreadBody(problem problemcache.Problem) string {
	return problem.URL()
}

func intPointer(value int) *int {
	return &value
}
