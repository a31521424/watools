package logger

import (
	"bytes"
	"fmt"
	"github.com/rs/zerolog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestErrorKeepsFieldsAndRedactsSecrets(t *testing.T) {
	old := WaLogger
	defer func() { WaLogger = old }()
	var output bytes.Buffer
	WaLogger = zerolog.New(&output)
	Error(fmt.Errorf("failure"), "message", "packageId", "watools.plugin.test", "token", "private")
	if !bytes.Contains(output.Bytes(), []byte(`"packageId":"watools.plugin.test"`)) || bytes.Contains(output.Bytes(), []byte("private")) {
		t.Fatal(output.String())
	}
}
func TestRotationDuringSessionIsBounded(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	writer := &rotatingWriter{dir: dir, maxSize: 10, keep: 3, now: func() time.Time { return now }}
	defer writer.Close()
	for i := 0; i < 10; i++ {
		if _, err := writer.Write([]byte("123456\n")); err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) > 3 {
			t.Fatalf("rotation temporarily retained %d files", len(entries))
		}
	}
	now = now.Add(24 * time.Hour)
	if _, err := writer.Write([]byte("new day\n")); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("retained %d files", len(entries))
	}
}
func TestLogQueryPaginationAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WATOOLS_CACHE_DIR", dir)
	logDir := filepath.Join(dir, "logs")
	os.MkdirAll(logDir, 0700)
	for _, name := range []string{"watools-a.log", "watools-b.log"} {
		var data bytes.Buffer
		for i := 0; i < 2000; i++ {
			fmt.Fprintf(&data, "{\"time\":\"2026-10-02T00:00:00Z\",\"message\":\"%d\"}\n", i)
		}
		os.WriteFile(filepath.Join(logDir, name), data.Bytes(), 0600)
	}
	result, err := QueryLogs(LogQuery{Files: []string{"watools-a.log", "watools-b.log"}, Cursor: "10", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalMatched != 4000 || len(result.Records) != 2 || result.Records[0].File != "watools-b.log" || result.Records[0].Line != 1990 || result.NextCursor != "12" {
		t.Fatalf("wrong pagination: %+v", result)
	}
}
