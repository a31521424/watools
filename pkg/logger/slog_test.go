package logger

import (
	"bytes"
	"context"
	"github.com/rs/zerolog"
	"log/slog"
	"strings"
	"testing"
)

func TestWailsSlogDoesNotRecordRuntimePayloads(t *testing.T) {
	old := WaLogger
	defer func() { WaLogger = old }()
	var output bytes.Buffer
	WaLogger = zerolog.New(&output)
	log := NewSlogAdapter().With("args", "synthetic-credential")
	log.LogAttrs(context.Background(), slog.LevelInfo, "Binding call complete:",
		slog.String("method", "GetPluginSecretApi"), slog.String("result", "synthetic-secret"))
	log.LogAttrs(context.Background(), slog.LevelInfo, "Runtime call:",
		slog.String("method", "Clipboard.SetText"), slog.String("args", "synthetic-clipboard"))
	text := output.String()
	for _, private := range []string{"synthetic-credential", "synthetic-secret", "synthetic-clipboard"} {
		if strings.Contains(text, private) {
			t.Fatalf("framework log exposed %s", private)
		}
	}
	for _, metadata := range []string{"GetPluginSecretApi", "Clipboard.SetText"} {
		if !strings.Contains(text, metadata) {
			t.Fatalf("missing call metadata %s", metadata)
		}
	}
}
