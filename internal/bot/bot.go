package bot

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/nkoji21/leetdaily/internal/state"
)

// DailyJob runs the daily problem posting for a given date.
type DailyJob interface {
	Run(context.Context, state.Date) error
}

// Repository combines all storage operations needed by the bot.
type Repository interface {
	BackfillRepository
	CheckRepository
}

// Config holds all dependencies for creating a Bot.
type Config struct {
	Session    *discordgo.Session
	Job        DailyJob
	Repository Repository
	AppID      string
	Location   *time.Location
	HTTPAddr   string
	Logger     *slog.Logger
}

// Bot manages the discordgo Gateway connection, the HTTP server for Cloud Scheduler,
// and the /check slash command handler.
type Bot struct {
	session    *discordgo.Session
	job        DailyJob
	repository Repository
	appID      string
	location   *time.Location
	httpServer *http.Server
	logger     *slog.Logger
	check      *checkHandler
}

var checkCommand = &discordgo.ApplicationCommand{
	Name:        "check",
	Description: "未解決の問題一覧を表示します",
}

// New creates a Bot from the provided Config.
func New(cfg Config) (*Bot, error) {
	if cfg.Session == nil {
		return nil, fmt.Errorf("bot: session must not be nil")
	}
	if cfg.Job == nil {
		return nil, fmt.Errorf("bot: job must not be nil")
	}
	if cfg.Repository == nil {
		return nil, fmt.Errorf("bot: repository must not be nil")
	}
	if cfg.AppID == "" {
		return nil, fmt.Errorf("bot: app ID must not be empty")
	}
	if cfg.HTTPAddr == "" {
		return nil, fmt.Errorf("bot: HTTP addr must not be empty")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	location := cfg.Location
	if location == nil {
		location = time.UTC
	}

	b := &Bot{
		session:    cfg.Session,
		job:        cfg.Job,
		repository: cfg.Repository,
		appID:      cfg.AppID,
		location:   location,
		logger:     logger,
		check: &checkHandler{
			repository:     cfg.Repository,
			reactionClient: cfg.Session,
			logger:         logger.With("handler", "check"),
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", b.handleHealthz)
	mux.HandleFunc("POST /run", b.handleRun)
	b.httpServer = &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: mux,
	}

	return b, nil
}

// Run opens the Gateway connection, registers the /check command, runs backfill,
// starts the HTTP server, and blocks until ctx is cancelled.
func (b *Bot) Run(ctx context.Context) error {
	b.session.AddHandler(b.handleInteraction)

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("open Discord gateway: %w", err)
	}
	defer b.session.Close() //nolint:errcheck

	b.logger.Info("Discord gateway connected")

	if _, err := b.session.ApplicationCommandCreate(b.appID, "", checkCommand); err != nil {
		return fmt.Errorf("register /check command: %w", err)
	}
	b.logger.Info("registered /check command")

	// Backfill existing threads that predate PostedThreads tracking.
	if err := backfill(ctx, b.repository, b.session, b.logger.With("component", "backfill")); err != nil {
		b.logger.Warn("backfill encountered errors", "error", err)
	}

	errCh := make(chan error, 1)
	go func() {
		b.logger.Info("HTTP server starting", "addr", b.httpServer.Addr)
		if err := b.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := b.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("HTTP server shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	data, ok := i.Data.(discordgo.ApplicationCommandInteractionData)
	if !ok {
		return
	}
	if data.Name == "check" {
		b.check.handle(s, i)
	}
}

func (b *Bot) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (b *Bot) handleRun(w http.ResponseWriter, r *http.Request) {
	targetDate, err := state.ParseDate(time.Now().In(b.location).Format("2006-01-02"))
	if err != nil {
		http.Error(w, "failed to resolve target date", http.StatusInternalServerError)
		return
	}

	if err := b.job.Run(r.Context(), targetDate); err != nil {
		b.logger.Error("job run failed", "error", err)
		http.Error(w, "job execution failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("scheduled\n"))
}

// Compile-time checks: *discordgo.Session must satisfy ThreadLister.
var _ ThreadLister = (*discordgo.Session)(nil)

// Compile-time checks: Repository must embed BackfillRepository and CheckRepository.
var _ BackfillRepository = (Repository)(nil)
var _ CheckRepository = (Repository)(nil)
