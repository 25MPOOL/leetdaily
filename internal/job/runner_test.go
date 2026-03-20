package job

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/discord"
	"github.com/nkoji21/leetdaily/internal/problemcache"
	"github.com/nkoji21/leetdaily/internal/state"
	"github.com/nkoji21/leetdaily/internal/storage"
)

func TestRunnerSkipsAlreadyPostedGuild(t *testing.T) {
	t.Parallel()

	targetDate := mustDate(t, "2026-03-20")
	repository := &stubRepository{
		config: testConfig(),
		state: state.State{
			GuildStates: map[string]state.GuildState{
				"123456789012345678": {
					NextProblemNumber: 2,
					Job: state.JobState{
						TargetDate: &targetDate,
						Status:     state.JobStatusPosted,
					},
				},
			},
		},
		cache: problemcache.Cache{
			UpdatedAt: timePointer(time.Date(2026, 3, 20, 4, 0, 0, 0, time.UTC)),
			Problems:  []problemcache.Problem{},
		},
	}

	runner := newRunnerForTest(t, repository, stubFetcher{}, &stubPoster{}, &stubNotifier{})
	if err := runner.Run(context.Background(), targetDate); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if repository.saveStateCalls != 0 {
		t.Fatalf("saveStateCalls = %d, want 0", repository.saveStateCalls)
	}
}

func TestRunnerRecoversStalePostingAndPosts(t *testing.T) {
	t.Parallel()

	targetDate := mustDate(t, "2026-03-20")
	startedAt := time.Date(2026, 3, 20, 4, 0, 0, 0, time.UTC)
	repository := &stubRepository{
		config: testConfig(),
		state: state.State{
			GuildStates: map[string]state.GuildState{
				"123456789012345678": {
					NextProblemNumber: 1,
					Job: state.JobState{
						TargetDate:       &targetDate,
						Status:           state.JobStatusPosting,
						PostingStartedAt: &startedAt,
					},
				},
			},
		},
		cache: problemcache.Cache{
			UpdatedAt: timePointer(time.Date(2026, 3, 20, 4, 0, 0, 0, time.UTC)),
			Problems:  []problemcache.Problem{},
		},
	}
	poster := &stubPoster{
		tags: map[problemcache.Difficulty]string{
			problemcache.DifficultyEasy:   "easy-1",
			problemcache.DifficultyMedium: "medium-1",
			problemcache.DifficultyHard:   "hard-1",
		},
		threadID: "thread-1",
	}

	runner := newRunnerForTest(t, repository, stubFetcher{}, poster, &stubNotifier{})
	runner.now = func() time.Time { return time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC) }

	if err := runner.Run(context.Background(), targetDate); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := repository.state.GuildStates["123456789012345678"]
	if got.Job.Status != state.JobStatusPosted {
		t.Fatalf("Job.Status = %q, want %q", got.Job.Status, state.JobStatusPosted)
	}
	if got.NextProblemNumber != 2 {
		t.Fatalf("NextProblemNumber = %d, want 2", got.NextProblemNumber)
	}
	if got.LastPostedThreadID == nil || *got.LastPostedThreadID != "thread-1" {
		t.Fatalf("LastPostedThreadID = %v, want thread-1", got.LastPostedThreadID)
	}
}

func TestRunnerRetriesAndEventuallySucceeds(t *testing.T) {
	t.Parallel()

	targetDate := mustDate(t, "2026-03-20")
	repository := &stubRepository{
		config: testConfig(),
		state:  state.New(),
		cache:  testCache(),
	}
	poster := &stubPoster{
		tags: map[problemcache.Difficulty]string{
			problemcache.DifficultyEasy:   "easy-1",
			problemcache.DifficultyMedium: "medium-1",
			problemcache.DifficultyHard:   "hard-1",
		},
		threadID: "thread-1",
		threadErrs: []error{
			errors.New("temporary failure"),
		},
	}
	notifier := &stubNotifier{}

	runner := newRunnerForTest(t, repository, stubFetcher{}, poster, notifier)
	runner.sleep = func(context.Context, time.Duration) error { return nil }

	if err := runner.Run(context.Background(), targetDate); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := repository.state.GuildStates["123456789012345678"]
	if got.Job.Status != state.JobStatusPosted {
		t.Fatalf("Job.Status = %q, want %q", got.Job.Status, state.JobStatusPosted)
	}
	if notifier.calls != 0 {
		t.Fatalf("notifier.calls = %d, want 0", notifier.calls)
	}
}

func TestRunnerNotifiesAfterRetriesExhausted(t *testing.T) {
	t.Parallel()

	targetDate := mustDate(t, "2026-03-20")
	repository := &stubRepository{
		config: testConfig(),
		state:  state.New(),
		cache:  testCache(),
	}
	poster := &stubPoster{
		tags: map[problemcache.Difficulty]string{
			problemcache.DifficultyEasy:   "easy-1",
			problemcache.DifficultyMedium: "medium-1",
			problemcache.DifficultyHard:   "hard-1",
		},
		threadErrs: []error{
			errors.New("fail-1"),
			errors.New("fail-2"),
			errors.New("fail-3"),
		},
	}
	notifier := &stubNotifier{}

	runner := newRunnerForTest(t, repository, stubFetcher{}, poster, notifier)
	runner.sleep = func(context.Context, time.Duration) error { return nil }

	if err := runner.Run(context.Background(), targetDate); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := repository.state.GuildStates["123456789012345678"]
	if got.Job.Status != state.JobStatusFailed {
		t.Fatalf("Job.Status = %q, want %q", got.Job.Status, state.JobStatusFailed)
	}
	if got.Job.RetryCount != 3 {
		t.Fatalf("RetryCount = %d, want 3", got.Job.RetryCount)
	}
	if notifier.calls != 1 {
		t.Fatalf("notifier.calls = %d, want 1", notifier.calls)
	}
}

func TestRunnerRefreshesProblemCacheWhenThresholdIsLow(t *testing.T) {
	t.Parallel()

	targetDate := mustDate(t, "2026-03-20")
	cfg := testConfig()
	cfg.ProblemCache.RefillThreshold = 2
	repository := &stubRepository{
		config: cfg,
		state:  state.New(),
		cache: problemcache.Cache{
			UpdatedAt: timePointer(time.Date(2026, 3, 20, 4, 0, 0, 0, time.UTC)),
			Problems: []problemcache.Problem{
				{ProblemNumber: 1, Title: "One", Slug: "one", Difficulty: problemcache.DifficultyEasy},
			},
		},
	}
	fetcher := stubFetcher{
		problems: []problemcache.Problem{
			{ProblemNumber: 1, Title: "One", Slug: "one", Difficulty: problemcache.DifficultyEasy},
			{ProblemNumber: 2, Title: "Two", Slug: "two", Difficulty: problemcache.DifficultyEasy},
		},
	}
	poster := &stubPoster{
		tags: map[problemcache.Difficulty]string{
			problemcache.DifficultyEasy:   "easy-1",
			problemcache.DifficultyMedium: "medium-1",
			problemcache.DifficultyHard:   "hard-1",
		},
		threadID: "thread-1",
	}

	runner := newRunnerForTest(t, repository, fetcher, poster, &stubNotifier{})
	if err := runner.Run(context.Background(), targetDate); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if repository.saveCacheCalls == 0 {
		t.Fatal("saveCacheCalls = 0, want cache refresh save")
	}
}

func TestRunnerSavesRecoveredStalePostingBeforeEarlyFailure(t *testing.T) {
	t.Parallel()

	targetDate := mustDate(t, "2026-03-20")
	startedAt := time.Date(2026, 3, 20, 4, 0, 0, 0, time.UTC)
	repository := &stubRepository{
		config: testConfig(),
		state: state.State{
			GuildStates: map[string]state.GuildState{
				"123456789012345678": {
					NextProblemNumber: 1,
					Job: state.JobState{
						TargetDate:       &targetDate,
						Status:           state.JobStatusPosting,
						PostingStartedAt: &startedAt,
					},
				},
			},
		},
		cache: problemcache.Cache{
			UpdatedAt: timePointer(time.Date(2026, 3, 20, 4, 0, 0, 0, time.UTC)),
			Problems:  []problemcache.Problem{},
		},
	}
	notifier := &stubNotifier{}
	runner := newRunnerForTest(t, repository, stubFetcher{err: context.DeadlineExceeded}, &stubPoster{}, notifier)
	runner.now = func() time.Time { return time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC) }

	if err := runner.Run(context.Background(), targetDate); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := repository.state.GuildStates["123456789012345678"]
	if got.Job.Status != state.JobStatusIdle {
		t.Fatalf("Job.Status = %q, want %q", got.Job.Status, state.JobStatusIdle)
	}
	if repository.saveStateCalls == 0 {
		t.Fatal("saveStateCalls = 0, want stale recovery save")
	}
	if notifier.calls != 1 {
		t.Fatalf("notifier.calls = %d, want 1", notifier.calls)
	}
}

func TestRunnerContinuesWithStaleCacheWhenRefillFallbackOccurs(t *testing.T) {
	t.Parallel()

	targetDate := mustDate(t, "2026-03-20")
	cfg := testConfig()
	cfg.ProblemCache.RefillThreshold = 5
	repository := &stubRepository{
		config: cfg,
		state:  state.New(),
		cache:  testCache(),
	}
	poster := &stubPoster{
		tags: map[problemcache.Difficulty]string{
			problemcache.DifficultyEasy:   "easy-1",
			problemcache.DifficultyMedium: "medium-1",
			problemcache.DifficultyHard:   "hard-1",
		},
		threadID: "thread-1",
	}
	notifier := &stubNotifier{}
	runner := newRunnerForTest(t, repository, stubFetcher{err: context.DeadlineExceeded}, poster, notifier)

	if err := runner.Run(context.Background(), targetDate); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := repository.state.GuildStates["123456789012345678"]
	if got.Job.Status != state.JobStatusPosted {
		t.Fatalf("Job.Status = %q, want %q", got.Job.Status, state.JobStatusPosted)
	}
	if notifier.calls != 1 {
		t.Fatalf("notifier.calls = %d, want 1", notifier.calls)
	}
}

func newRunnerForTest(t *testing.T, repository *stubRepository, fetcher stubFetcher, poster *stubPoster, notifier *stubNotifier) *Runner {
	t.Helper()

	runner, err := New(repository, fetcher, poster, notifier)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	runner.now = func() time.Time { return time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC) }
	runner.sleep = func(context.Context, time.Duration) error { return nil }
	return runner
}

type stubRepository struct {
	config         config.Config
	state          state.State
	stateVersion   storage.Version
	cache          problemcache.Cache
	cacheVersion   storage.Version
	saveStateCalls int
	saveCacheCalls int
}

func (s *stubRepository) LoadConfig(context.Context) (config.Config, error) {
	return s.config, nil
}

func (s *stubRepository) LoadState(context.Context) (state.State, storage.Version, error) {
	return s.state, s.stateVersion, nil
}

func (s *stubRepository) SaveState(_ context.Context, current state.State, version storage.Version) (storage.Version, error) {
	s.saveStateCalls++
	s.state = current
	s.stateVersion = storage.Version{Token: version.Token + "s"}
	return s.stateVersion, nil
}

func (s *stubRepository) LoadProblemCache(context.Context) (problemcache.Cache, storage.Version, error) {
	return s.cache, s.cacheVersion, nil
}

func (s *stubRepository) SaveProblemCache(_ context.Context, cache problemcache.Cache, version storage.Version) (storage.Version, error) {
	s.saveCacheCalls++
	s.cache = cache
	s.cacheVersion = storage.Version{Token: version.Token + "c"}
	return s.cacheVersion, nil
}

type stubFetcher struct {
	problems []problemcache.Problem
	err      error
}

func (s stubFetcher) FetchProblems(context.Context) ([]problemcache.Problem, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.problems != nil {
		return s.problems, nil
	}
	return testCache().Problems, nil
}

type stubPoster struct {
	tags       map[problemcache.Difficulty]string
	threadID   string
	threadErrs []error
}

func (s *stubPoster) EnsureDifficultyTags(context.Context, string) (map[problemcache.Difficulty]string, error) {
	return s.tags, nil
}

func (s *stubPoster) CreateForumThread(context.Context, string, string, string, string) (discord.Thread, error) {
	if len(s.threadErrs) > 0 {
		err := s.threadErrs[0]
		s.threadErrs = s.threadErrs[1:]
		return discord.Thread{}, err
	}
	return discord.Thread{ID: s.threadID}, nil
}

type stubNotifier struct {
	calls int
}

func (s *stubNotifier) NotifyFailure(context.Context, string, error) error {
	s.calls++
	return nil
}

func testConfig() config.Config {
	return config.Config{
		Timezone: "Asia/Tokyo",
		Retry: config.RetryConfig{
			IntervalMinutes: 5,
			MaxAttempts:     3,
		},
		ProblemCache: config.ProblemCacheConfig{
			RefillThreshold: 1,
		},
		Guilds: []config.Guild{
			{
				GuildID:               "123456789012345678",
				Enabled:               true,
				ForumChannelID:        "234567890123456789",
				NotificationChannelID: "345678901234567890",
				StartProblemNumber:    1,
			},
		},
	}
}

func testCache() problemcache.Cache {
	return problemcache.Cache{
		UpdatedAt: timePointer(time.Date(2026, 3, 20, 4, 0, 0, 0, time.UTC)),
		Problems: []problemcache.Problem{
			{ProblemNumber: 1, Title: "One", Slug: "one", Difficulty: problemcache.DifficultyEasy},
			{ProblemNumber: 2, Title: "Two", Slug: "two", Difficulty: problemcache.DifficultyMedium, IsPaidOnly: true},
			{ProblemNumber: 3, Title: "Three", Slug: "three", Difficulty: problemcache.DifficultyHard},
		},
	}
}

func mustDate(t *testing.T, value string) state.Date {
	t.Helper()
	date, err := state.ParseDate(value)
	if err != nil {
		t.Fatalf("ParseDate(%q) error = %v", value, err)
	}
	return date
}

func timePointer(value time.Time) *time.Time {
	return &value
}
