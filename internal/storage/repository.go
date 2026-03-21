package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/problemcache"
	"github.com/nkoji21/leetdaily/internal/state"
)

var ErrNotFound = errors.New("storage: not found")
var ErrConflict = errors.New("storage: conflict")

type Version struct {
	Token string
}

type Paths struct {
	ConfigPath   string
	GuildsPath   string
	StatePath    string
	ProblemsPath string
}

type Repository interface {
	LoadConfig(context.Context) (config.Config, error)
	LoadGuildSettings(context.Context) (config.GuildSettings, Version, error)
	SaveGuildSettings(context.Context, config.GuildSettings, Version) (Version, error)
	LoadState(context.Context) (state.State, Version, error)
	SaveState(context.Context, state.State, Version) (Version, error)
	LoadProblemCache(context.Context) (problemcache.Cache, Version, error)
	SaveProblemCache(context.Context, problemcache.Cache, Version) (Version, error)
}

func (p Paths) Validate() error {
	if strings.TrimSpace(p.ConfigPath) == "" {
		return fmt.Errorf("config path must not be empty")
	}

	if strings.TrimSpace(p.GuildsPath) == "" {
		return fmt.Errorf("guilds path must not be empty")
	}

	if strings.TrimSpace(p.StatePath) == "" {
		return fmt.Errorf("state path must not be empty")
	}

	if strings.TrimSpace(p.ProblemsPath) == "" {
		return fmt.Errorf("problems path must not be empty")
	}

	return nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

func (v Version) IsZero() bool {
	return strings.TrimSpace(v.Token) == ""
}
