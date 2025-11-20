package logging

import (
	"io"
	"log/slog"
)

func New(out io.Writer) *slog.Logger {
	hnd := slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo})
	log := slog.New(hnd)
	return log
}
