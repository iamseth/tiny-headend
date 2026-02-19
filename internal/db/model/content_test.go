package model

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/iamseth/tiny-headend/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestRepo(t *testing.T) *ContentRepo {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	g, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if err := g.AutoMigrate(&Content{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	return NewContentRepo(g)
}

func TestContentRepoCreateSetsID(t *testing.T) {
	repo := newTestRepo(t)
	c := &service.Content{
		Title:  "title",
		Path:   "/tmp/file.ts",
		Size:   10,
		Length: 1.5,
	}

	if err := repo.Create(context.Background(), c); err != nil {
		t.Fatalf("create content: %v", err)
	}

	if c.ID == 0 {
		t.Fatalf("expected created content ID to be set")
	}
}

func TestContentRepoGetByIDNotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.GetByID(context.Background(), 999999)
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestContentRepoListIncludesPath(t *testing.T) {
	repo := newTestRepo(t)
	c := &service.Content{
		Title:  "title",
		Path:   "/tmp/path.ts",
		Size:   10,
		Length: 1.5,
	}

	if err := repo.Create(context.Background(), c); err != nil {
		t.Fatalf("create content: %v", err)
	}

	got, err := repo.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("list content: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 content row, got %d", len(got))
	}
	if got[0].Path != c.Path {
		t.Fatalf("expected path %q, got %q", c.Path, got[0].Path)
	}
}

func TestContentRepoListUsesStableIDOrder(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	first := &service.Content{Title: "first", Path: "/tmp/1.ts", Size: 1, Length: 1}
	second := &service.Content{Title: "second", Path: "/tmp/2.ts", Size: 2, Length: 2}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create first content: %v", err)
	}
	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("create second content: %v", err)
	}

	pageOne, err := repo.List(ctx, 1, 0)
	if err != nil {
		t.Fatalf("list page one: %v", err)
	}
	pageTwo, err := repo.List(ctx, 1, 1)
	if err != nil {
		t.Fatalf("list page two: %v", err)
	}

	if len(pageOne) != 1 || len(pageTwo) != 1 {
		t.Fatalf("expected one row per page, got %d and %d", len(pageOne), len(pageTwo))
	}
	if pageOne[0].ID != first.ID {
		t.Fatalf("expected first page ID %d, got %d", first.ID, pageOne[0].ID)
	}
	if pageTwo[0].ID != second.ID {
		t.Fatalf("expected second page ID %d, got %d", second.ID, pageTwo[0].ID)
	}
}

func TestContentRepoUpdatePersistsZeroValues(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	c := &service.Content{
		Title:  "before",
		Path:   "/tmp/before.ts",
		Size:   10,
		Length: 1.5,
	}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("create content: %v", err)
	}

	c.Title = ""
	c.Path = ""
	c.Size = 0
	c.Length = 0
	if err := repo.Update(ctx, c); err != nil {
		t.Fatalf("update content: %v", err)
	}

	got, err := repo.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("get content: %v", err)
	}
	if got.Title != "" || got.Path != "" || got.Size != 0 || got.Length != 0 {
		t.Fatalf("expected zero/empty values after update, got: %+v", got)
	}
}

func TestContentRepoUpdateNotFound(t *testing.T) {
	repo := newTestRepo(t)

	err := repo.Update(context.Background(), &service.Content{
		ID:     999999,
		Title:  "missing",
		Path:   "/tmp/missing.ts",
		Size:   1,
		Length: 1,
	})
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestContentRepoDeleteNotFound(t *testing.T) {
	repo := newTestRepo(t)

	err := repo.Delete(context.Background(), 999999)
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}
