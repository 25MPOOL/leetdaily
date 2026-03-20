package logging

import (
	"io"
	"log/slog"
)

func New(level slog.Level, output io.Writer) *slog.Logger {
	if output == nil {
		output = io.Discard
	}

	return slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: level,
	}))
}
