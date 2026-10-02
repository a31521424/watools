package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestEveryConnectionHasBusyTimeout(t *testing.T) {
	t.Setenv("WATOOLS_DATA_DIR", t.TempDir()+"/state")
	t.Setenv("WATOOLS_CACHE_DIR", t.TempDir())
	database, err := NewWaDB()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var held []*sql.Conn
	defer func() {
		for _, conn := range held {
			conn.Close()
		}
	}()
	for i := 0; i < 4; i++ {
		conn, err := database.db.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		held = append(held, conn)
		var timeout int
		if err := conn.QueryRowContext(context.Background(), "PRAGMA busy_timeout").Scan(&timeout); err != nil {
			t.Fatal(err)
		}
		if timeout != 5000 {
			t.Fatalf("connection %d timeout=%d", i, timeout)
		}
	}
}
