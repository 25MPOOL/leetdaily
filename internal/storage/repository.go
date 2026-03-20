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

type Paths struct {
	ConfigPath   string
	StatePath    string
	ProblemsPath string
}

type Repository interface {
	LoadConfig(context.Context) (config.Config, error)
	LoadState(context.Context) (state.State, error)
	SaveState(context.Context, state.State) error
	LoadProblemCache(context.Context) (problemcache.Cache, error)
	SaveProblemCache(context.Context, problemcache.Cache) error
}

func (p Paths) Validate() error {
	if strings.TrimSpace(p.ConfigPath) == "" {
		return fmt.Errorf("config path must not be empty")
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
