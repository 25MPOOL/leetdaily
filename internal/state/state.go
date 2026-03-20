package state

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

type JobStatus string

const (
	JobStatusIdle    JobStatus = "idle"
	JobStatusPosting JobStatus = "posting"
	JobStatusPosted  JobStatus = "posted"
	JobStatusFailed  JobStatus = "failed"
)

type State struct {
	GuildStates map[string]GuildState `json:"guild_states"`
}

type GuildState struct {
	NextProblemNumber       int        `json:"next_problem_number"`
	LastPostedProblemNumber *int       `json:"last_posted_problem_number"`
	LastPostedAt            *time.Time `json:"last_posted_at"`
	LastPostedThreadID      *string    `json:"last_posted_thread_id"`
	Job                     JobState   `json:"job"`
}

type JobState struct {
	TargetDate       *Date      `json:"target_date"`
	Status           JobStatus  `json:"status"`
	ProblemNumber    *int       `json:"problem_number"`
	RetryCount       int        `json:"retry_count"`
	LastError        *string    `json:"last_error"`
	PostingStartedAt *time.Time `json:"posting_started_at"`
}

func New() State {
	return State{
		GuildStates: map[string]GuildState{},
	}
}

func DefaultGuildState(startProblemNumber int) GuildState {
	return GuildState{
		NextProblemNumber: startProblemNumber,
		Job: JobState{
			Status: JobStatusIdle,
		},
	}
}

func (s *State) EnsureGuild(guildID string, startProblemNumber int) (GuildState, bool) {
	if s.GuildStates == nil {
		s.GuildStates = map[string]GuildState{}
	}

	if guildState, ok := s.GuildStates[guildID]; ok {
		return guildState, false
	}

	guildState := DefaultGuildState(startProblemNumber)
	s.GuildStates[guildID] = guildState
	return guildState, true
}

func (s State) Validate() error {
	for guildID, guildState := range s.GuildStates {
		if !isSnowflake(guildID) {
			return fmt.Errorf("guild_states[%q]: key must be a numeric Discord ID", guildID)
		}

		if err := guildState.Validate(); err != nil {
			return fmt.Errorf("guild_states[%q]: %w", guildID, err)
		}
	}

	return nil
}

func (g GuildState) Validate() error {
	if g.NextProblemNumber < 1 {
		return fmt.Errorf("next_problem_number must be greater than 0: %d", g.NextProblemNumber)
	}

	if g.LastPostedProblemNumber != nil && *g.LastPostedProblemNumber < 1 {
		return fmt.Errorf("last_posted_problem_number must be greater than 0: %d", *g.LastPostedProblemNumber)
	}

	if g.LastPostedAt != nil && g.LastPostedAt.IsZero() {
		return fmt.Errorf("last_posted_at must not be zero")
	}

	if g.LastPostedThreadID != nil && !isSnowflake(*g.LastPostedThreadID) {
		return fmt.Errorf("last_posted_thread_id must be a numeric Discord ID: %q", *g.LastPostedThreadID)
	}

	if err := g.Job.Validate(); err != nil {
		return fmt.Errorf("job: %w", err)
	}

	return nil
}

func (j JobState) Validate() error {
	switch j.Status {
	case JobStatusIdle, JobStatusPosting, JobStatusPosted, JobStatusFailed:
	default:
		return fmt.Errorf("status must be one of idle/posting/posted/failed: %q", j.Status)
	}

	if j.TargetDate != nil && j.TargetDate.IsZero() {
		return fmt.Errorf("target_date must not be zero")
	}

	if j.ProblemNumber != nil && *j.ProblemNumber < 1 {
		return fmt.Errorf("problem_number must be greater than 0: %d", *j.ProblemNumber)
	}

	if j.RetryCount < 0 {
		return fmt.Errorf("retry_count must be greater than or equal to 0: %d", j.RetryCount)
	}

	if j.LastError != nil && strings.TrimSpace(*j.LastError) == "" {
		return fmt.Errorf("last_error must not be empty when present")
	}

	if j.PostingStartedAt != nil && j.PostingStartedAt.IsZero() {
		return fmt.Errorf("posting_started_at must not be zero")
	}

	return nil
}

func isSnowflake(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	return true
}
