package state

import (
	"testing"
	"time"
)

func TestEnsureGuildInitializesMissingGuild(t *testing.T) {
	t.Parallel()

	current := New()
	guildState, created := current.EnsureGuild("123456789012345678", 51)
	if !created {
		t.Fatal("EnsureGuild() created = false, want true")
	}

	if guildState.NextProblemNumber != 51 {
		t.Fatalf("NextProblemNumber = %d, want 51", guildState.NextProblemNumber)
	}

	if guildState.Job.Status != JobStatusIdle {
		t.Fatalf("Job.Status = %q, want %q", guildState.Job.Status, JobStatusIdle)
	}

	second, created := current.EnsureGuild("123456789012345678", 999)
	if created {
		t.Fatal("EnsureGuild() created = true on existing guild, want false")
	}

	if second.NextProblemNumber != 51 {
		t.Fatalf("existing NextProblemNumber = %d, want 51", second.NextProblemNumber)
	}
}

func TestStateValidateAcceptsValidState(t *testing.T) {
	t.Parallel()

	targetDate, err := ParseDate("2026-03-20")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}

	lastPostedAt := time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC)
	postingStartedAt := time.Date(2026, 3, 20, 5, 0, 3, 0, time.UTC)
	lastPostedProblem := 137
	threadID := "234567890123456789"
	problemNumber := 137
	lastError := "discord permissions missing"

	current := State{
		GuildStates: map[string]GuildState{
			"123456789012345678": {
				NextProblemNumber:       138,
				LastPostedProblemNumber: &lastPostedProblem,
				LastPostedAt:            &lastPostedAt,
				LastPostedThreadID:      &threadID,
				Job: JobState{
					TargetDate:       &targetDate,
					Status:           JobStatusFailed,
					ProblemNumber:    &problemNumber,
					RetryCount:       3,
					LastError:        &lastError,
					PostingStartedAt: &postingStartedAt,
				},
			},
		},
	}

	if err := current.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}
}

func TestStateValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state State
	}{
		{
			name: "invalid guild id key",
			state: State{
				GuildStates: map[string]GuildState{
					"guild-1": DefaultGuildState(1),
				},
			},
		},
		{
			name: "invalid next problem number",
			state: State{
				GuildStates: map[string]GuildState{
					"123456789012345678": {
						NextProblemNumber: 0,
						Job: JobState{
							Status: JobStatusIdle,
						},
					},
				},
			},
		},
		{
			name: "invalid job status",
			state: State{
				GuildStates: map[string]GuildState{
					"123456789012345678": {
						NextProblemNumber: 1,
						Job: JobState{
							Status: "running",
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.state.Validate(); err == nil {
				t.Fatal("Validate() returned nil error, want validation error")
			}
		})
	}
}
