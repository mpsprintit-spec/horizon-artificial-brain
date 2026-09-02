// Package logger provides structured logging defaults built on log/slog.
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

type Options struct {
	Environment, Level string
	Output             io.Writer
}

func New(opts Options) *slog.Logger {
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}
	level := new(slog.LevelVar)
	level.Set(parseLevel(opts.Level))
	handlerOpts := &slog.HandlerOptions{Level: level}
	if strings.EqualFold(opts.Environment, "development") {
		return slog.New(slog.NewTextHandler(out, handlerOpts))
	}
	return slog.New(slog.NewJSONHandler(out, handlerOpts))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
