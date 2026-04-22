// Package logging wires a process-wide slog.Logger using the options from
// config.LoggingCfg.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Options controls how New produces a Logger.
type Options struct {
	Level  string // debug | info | warn | error
	Format string // text | json
	Out    io.Writer
}

// New builds a slog.Logger for the given options. Unknown levels default to
// info; unknown formats default to text.
func New(opts Options) *slog.Logger {
	lvl := parseLevel(opts.Level)
	out := opts.Out
	if out == nil {
		out = os.Stderr
	}
	handlerOpts := &slog.HandlerOptions{Level: lvl}
	switch strings.ToLower(opts.Format) {
	case "json":
		return slog.New(slog.NewJSONHandler(out, handlerOpts))
	default:
		return slog.New(slog.NewTextHandler(out, handlerOpts))
	}
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
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
