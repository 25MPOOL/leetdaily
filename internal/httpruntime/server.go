package httpruntime

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nkoji21/leetdaily/internal/state"
)

type DailyJob interface {
	Run(context.Context, state.Date) error
}

type Server struct {
	httpServer *http.Server
	job        DailyJob
	now        func() time.Time
	location   *time.Location
}

type Options struct {
	DiscordInteractions http.Handler
}

func New(addr string, location *time.Location, job DailyJob) (*Server, error) {
	return NewWithOptions(addr, location, job, Options{})
}

func NewWithOptions(addr string, location *time.Location, job DailyJob, options Options) (*Server, error) {
	if job == nil {
		return nil, fmt.Errorf("HTTP runtime job must not be nil")
	}
	if location == nil {
		location = time.UTC
	}

	server := &Server{
		job:      job,
		now:      time.Now,
		location: location,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.handleHealthz)
	mux.HandleFunc("POST /run", server.handleRun)
	if options.DiscordInteractions != nil {
		mux.Handle("POST /discord/interactions", options.DiscordInteractions)
	}

	server.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server, nil
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	targetDate, err := state.ParseDate(s.now().In(s.location).Format("2006-01-02"))
	if err != nil {
		http.Error(w, "failed to resolve target date", http.StatusInternalServerError)
		return
	}

	if err := s.job.Run(r.Context(), targetDate); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("scheduled\n"))
}
