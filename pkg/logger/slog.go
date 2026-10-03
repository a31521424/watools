package logger

import (
	"context"
	"github.com/rs/zerolog"
	"log/slog"
	"strings"
)

// NewSlogAdapter routes v3 framework logs through the existing rotating log.
func NewSlogAdapter() *slog.Logger { return slog.New(slogAdapter{}) }

type slogAdapter struct {
	attrs []slog.Attr
	group string
}

func (h slogAdapter) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelDebug
}
func (h slogAdapter) Handle(_ context.Context, r slog.Record) error {
	level := zerolog.InfoLevel
	switch {
	case r.Level >= slog.LevelError:
		level = zerolog.ErrorLevel
	case r.Level >= slog.LevelWarn:
		level = zerolog.WarnLevel
	case r.Level < slog.LevelInfo:
		level = zerolog.DebugLevel
	}
	event := WaLogger.WithLevel(level)
	add := func(a slog.Attr) {
		// Wails debug logs include binding arguments/results and clipboard payloads.
		// Preserve call metadata without writing credentials or clipboard contents.
		if a.Key == "args" || a.Key == "result" {
			return
		}
		a.Value = a.Value.Resolve()
		key := a.Key
		if h.group != "" {
			key = h.group + "." + key
		}
		event.Interface(key, a.Value.Any())
	}
	for _, a := range h.attrs {
		add(a)
	}
	r.Attrs(func(a slog.Attr) bool { add(a); return true })
	event.Msg(r.Message)
	return nil
}
func (h slogAdapter) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return h
}
func (h slogAdapter) WithGroup(name string) slog.Handler {
	h.group = strings.Trim(h.group+"."+name, ".")
	return h
}
