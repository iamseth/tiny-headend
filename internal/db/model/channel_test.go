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

func newTestChannelRepo(t *testing.T) *ChannelRepo {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test-channel.db")
	g, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if err := g.AutoMigrate(&Channel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	return NewChannelRepo(g)
}

func TestChannelRepoCreateSetsID(t *testing.T) {
	repo := newTestChannelRepo(t)
	c := &service.Channel{
		Title:         "ABC",
		ChannelNumber: 7,
		Description:   "news",
	}

	if err := repo.Create(context.Background(), c); err != nil {
		t.Fatalf("create channel: %v", err)
	}

	if c.ID == 0 {
		t.Fatalf("expected created channel ID to be set")
	}
}

func TestChannelRepoGetByIDNotFound(t *testing.T) {
	repo := newTestChannelRepo(t)

	_, err := repo.GetByID(context.Background(), 999999)
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestChannelRepoListIncludesFields(t *testing.T) {
	repo := newTestChannelRepo(t)
	c := &service.Channel{
		Title:         "ABC",
		ChannelNumber: 7,
		Description:   "news",
	}
	if err := repo.Create(context.Background(), c); err != nil {
		t.Fatalf("create channel: %v", err)
	}

	got, err := repo.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("list channels: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 channel row, got %d", len(got))
	}
	if got[0].Title != c.Title || got[0].ChannelNumber != c.ChannelNumber || got[0].Description != c.Description {
		t.Fatalf("expected %+v, got %+v", c, got[0])
	}
}

func TestChannelRepoListUsesStableIDOrder(t *testing.T) {
	repo := newTestChannelRepo(t)
	ctx := context.Background()

	first := &service.Channel{Title: "ABC", ChannelNumber: 7, Description: "news"}
	second := &service.Channel{Title: "NBC", ChannelNumber: 8, Description: "sports"}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create first channel: %v", err)
	}
	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("create second channel: %v", err)
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

func TestChannelRepoUpdatePersistsValues(t *testing.T) {
	repo := newTestChannelRepo(t)
	ctx := context.Background()

	c := &service.Channel{
		Title:         "ABC",
		ChannelNumber: 7,
		Description:   "news",
	}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("create channel: %v", err)
	}

	c.Title = "ABC HD"
	c.ChannelNumber = 107
	c.Description = "24-hour news"
	if err := repo.Update(ctx, c); err != nil {
		t.Fatalf("update channel: %v", err)
	}

	got, err := repo.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("get channel: %v", err)
	}
	if got.Title != c.Title || got.ChannelNumber != c.ChannelNumber || got.Description != c.Description {
		t.Fatalf("expected updated channel %+v, got %+v", c, got)
	}
}

func TestChannelRepoUpdateNotFound(t *testing.T) {
	repo := newTestChannelRepo(t)

	err := repo.Update(context.Background(), &service.Channel{
		ID:            999999,
		Title:         "missing",
		ChannelNumber: 1,
		Description:   "missing",
	})
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestChannelRepoDeleteNotFound(t *testing.T) {
	repo := newTestChannelRepo(t)

	err := repo.Delete(context.Background(), 999999)
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
