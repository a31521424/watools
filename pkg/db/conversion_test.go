package db

import (
	"testing"
	"time"
	"watools/pkg/models"
)

func TestApplicationConversionPreservesUsage(t *testing.T) {
	lastUsed := time.Now().Truncate(time.Second)
	app := ConvertApplicationCommand(Application{ID: "stable-id", Name: "App", Path: "/App", UsedCount: 42, LastUsedAt: models.ToOptionTime(lastUsed)})
	if app.UsedCount != 42 || app.LastUsedAt.IsNone() || !app.LastUsedAt.MustGet().Equal(lastUsed) {
		t.Fatalf("usage lost during conversion: count=%d lastUsed=%v", app.UsedCount, app.LastUsedAt)
	}
}
