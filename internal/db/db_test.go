package db

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iamseth/tiny-headend/internal/db/model"
)

func TestOpenConfiguresSQLitePragmas(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tiny-headend-test.db")

	g, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() {
		if closeErr := Close(g); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()

	var journalMode string
	if err := g.Raw("PRAGMA journal_mode;").Scan(&journalMode).Error; err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if strings.ToLower(journalMode) != "wal" {
		t.Fatalf("expected journal_mode wal, got %q", journalMode)
	}

	var busyTimeout int
	if err := g.Raw("PRAGMA busy_timeout;").Scan(&busyTimeout).Error; err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("expected busy_timeout 5000, got %d", busyTimeout)
	}

	var foreignKeys int
	if err := g.Raw("PRAGMA foreign_keys;").Scan(&foreignKeys).Error; err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("expected foreign_keys 1, got %d", foreignKeys)
	}

	if err := Ping(context.Background(), g); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	stats, err := Stats(g)
	if err != nil {
		t.Fatalf("db stats: %v", err)
	}
	if stats.MaxOpenConnections != 1 {
		t.Fatalf("expected max open conns 1, got %d", stats.MaxOpenConnections)
	}
}

func TestMigrateCreatesContentAndChannelTables(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tiny-headend-migrate-test.db")

	g, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() {
		if closeErr := Close(g); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()

	if err := Migrate(g); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	if !g.Migrator().HasTable(&model.Content{}) {
		t.Fatalf("expected content table to exist after migration")
	}
	if !g.Migrator().HasTable(&model.Channel{}) {
		t.Fatalf("expected channel table to exist after migration")
	}
}
