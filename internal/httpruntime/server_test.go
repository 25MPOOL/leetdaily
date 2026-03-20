package httpruntime

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nkoji21/leetdaily/internal/state"
)

func TestHandlers(t *testing.T) {
	t.Parallel()

	job := &stubJob{}
	location, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("LoadLocation() error = %v", err)
	}

	server, err := New(":0", location, job)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	server.now = func() time.Time { return time.Date(2026, 3, 20, 7, 0, 0, 0, time.UTC) }

	t.Run("healthz", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		server.httpServer.Handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
	})

	t.Run("run", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/run", nil)
		server.httpServer.Handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
		}
		if job.calls != 1 {
			t.Fatalf("job.calls = %d, want 1", job.calls)
		}
		if job.lastDate.String() != "2026-03-20" {
			t.Fatalf("job.lastDate = %s, want 2026-03-20", job.lastDate)
		}
	})

	t.Run("run failure", func(t *testing.T) {
		job.err = errors.New("boom")
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/run", nil)
		server.httpServer.Handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
		}
	})
}

type stubJob struct {
	calls    int
	lastDate state.Date
	err      error
}

func (s *stubJob) Run(_ context.Context, date state.Date) error {
	s.calls++
	s.lastDate = date
	if s.err != nil {
		return s.err
	}
	return nil
}
