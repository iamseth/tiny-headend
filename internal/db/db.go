package db

import (
	"context"
	"database/sql"
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/iamseth/tiny-headend/internal/db/model"
)

func Open(path string) (*gorm.DB, error) {
	g, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open gorm sqlite db: %w", err)
	}

	sqlDB, err := g.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db from gorm db: %w", err)
	}

	configurePool(sqlDB)

	if err := configureSQLite(g); err != nil {
		return nil, err
	}

	return g, nil
}

func Close(g *gorm.DB) error {
	sqlDB, err := g.DB()
	if err != nil {
		return fmt.Errorf("get sql db from gorm db: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("close sql db: %w", err)
	}
	return nil
}

func Migrate(g *gorm.DB) error {
	if err := g.AutoMigrate(&model.Content{}); err != nil {
		return fmt.Errorf("auto-migrate db schema: %w", err)
	}
	return nil
}

func Ping(ctx context.Context, g *gorm.DB) error {
	sqlDB, err := g.DB()
	if err != nil {
		return fmt.Errorf("get sql db from gorm db: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping sql db: %w", err)
	}
	return nil
}

func Stats(g *gorm.DB) (sql.DBStats, error) {
	sqlDB, err := g.DB()
	if err != nil {
		return sql.DBStats{}, fmt.Errorf("get sql db from gorm db: %w", err)
	}
	return sqlDB.Stats(), nil
}

func configurePool(db *sql.DB) {
	// SQLite handles concurrent writes poorly with many open connections.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
}

func configureSQLite(g *gorm.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA foreign_keys = ON;",
	}

	for _, stmt := range pragmas {
		if err := g.Exec(stmt).Error; err != nil {
			return fmt.Errorf("sqlite pragma failed (%s): %w", stmt, err)
		}
	}

	return nil
}
