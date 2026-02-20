package service

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stubChannelRepo struct {
	createCalled bool
	getCalled    bool
	listCalled   bool
	updateCalled bool
	deleteCalled bool
	createErr    error
	getErr       error
	listErr      error
	updateErr    error
	deleteErr    error
	getChannel   *Channel
	listChannels []Channel
	gotGetID     uint
	gotDeleteID  uint
	gotLimit     int
	gotOffset    int
}

func (s *stubChannelRepo) Create(_ context.Context, _ *Channel) error {
	s.createCalled = true
	return s.createErr
}

func (s *stubChannelRepo) GetByID(_ context.Context, id uint) (*Channel, error) {
	s.getCalled = true
	s.gotGetID = id
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getChannel != nil {
		return s.getChannel, nil
	}
	return &Channel{}, nil
}

func (s *stubChannelRepo) List(_ context.Context, limit, offset int) ([]Channel, error) {
	s.listCalled = true
	s.gotLimit = limit
	s.gotOffset = offset
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.listChannels, nil
}

func (s *stubChannelRepo) Update(_ context.Context, _ *Channel) error {
	s.updateCalled = true
	return s.updateErr
}

func (s *stubChannelRepo) Delete(_ context.Context, id uint) error {
	s.deleteCalled = true
	s.gotDeleteID = id
	return s.deleteErr
}

func TestChannelServiceCreateValidatesInput(t *testing.T) {
	tests := []struct {
		name string
		in   *Channel
	}{
		{name: "nil channel", in: nil},
		{name: "empty title", in: &Channel{Title: "", ChannelNumber: 1, Description: "desc"}},
		{name: "whitespace title", in: &Channel{Title: " ", ChannelNumber: 1, Description: "desc"}},
		{name: "zero channel number", in: &Channel{Title: "ABC", ChannelNumber: 0, Description: "desc"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubChannelRepo{}
			svc := NewChannelService(repo)

			err := svc.Create(context.Background(), tc.in)
			var ve ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected validation error, got %v", err)
			}
			if repo.createCalled {
				t.Fatalf("repo create should not be called for invalid input")
			}
		})
	}
}

func TestChannelServiceCreateCallsRepoForValidInput(t *testing.T) {
	repo := &stubChannelRepo{}
	svc := NewChannelService(repo)

	err := svc.Create(context.Background(), &Channel{
		Title:         "ABC",
		ChannelNumber: 7,
		Description:   "news",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !repo.createCalled {
		t.Fatalf("expected repo create to be called")
	}
}

func TestChannelServiceCreateWrapsRepoError(t *testing.T) {
	repoErr := errors.New("db create failed")
	repo := &stubChannelRepo{createErr: repoErr}
	svc := NewChannelService(repo)

	err := svc.Create(context.Background(), &Channel{
		Title:         "ABC",
		ChannelNumber: 7,
		Description:   "news",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repo error, got %v", err)
	}
	if !strings.Contains(err.Error(), "create channel") {
		t.Fatalf("expected contextual message, got %v", err)
	}
}

func TestChannelServiceGetReturnsChannel(t *testing.T) {
	expected := &Channel{ID: 42, Title: "ABC", ChannelNumber: 9, Description: "sports"}
	repo := &stubChannelRepo{getChannel: expected}
	svc := NewChannelService(repo)

	got, err := svc.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !repo.getCalled {
		t.Fatalf("expected repo get to be called")
	}
	if repo.gotGetID != 42 {
		t.Fatalf("expected id 42, got %d", repo.gotGetID)
	}
	if got.ID != expected.ID || got.Title != expected.Title || got.ChannelNumber != expected.ChannelNumber || got.Description != expected.Description {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
}

func TestChannelServiceGetWrapsRepoError(t *testing.T) {
	repoErr := errors.New("db get failed")
	repo := &stubChannelRepo{getErr: repoErr}
	svc := NewChannelService(repo)

	_, err := svc.Get(context.Background(), 8)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !repo.getCalled {
		t.Fatalf("expected repo get to be called")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repo error, got %v", err)
	}
	if !strings.Contains(err.Error(), "get channel by id") {
		t.Fatalf("expected contextual message, got %v", err)
	}
}

func TestChannelServiceListReturnsChannels(t *testing.T) {
	repo := &stubChannelRepo{
		listChannels: []Channel{
			{ID: 1, Title: "ABC", ChannelNumber: 7, Description: "news"},
			{ID: 2, Title: "NBC", ChannelNumber: 8, Description: "sports"},
		},
	}
	svc := NewChannelService(repo)

	got, err := svc.List(context.Background(), 10, 2)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !repo.listCalled {
		t.Fatalf("expected repo list to be called")
	}
	if repo.gotLimit != 10 || repo.gotOffset != 2 {
		t.Fatalf("expected limit=10 offset=2, got limit=%d offset=%d", repo.gotLimit, repo.gotOffset)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(got))
	}
}

func TestChannelServiceListWrapsRepoError(t *testing.T) {
	repoErr := errors.New("db list failed")
	repo := &stubChannelRepo{listErr: repoErr}
	svc := NewChannelService(repo)

	_, err := svc.List(context.Background(), 10, 0)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !repo.listCalled {
		t.Fatalf("expected repo list to be called")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repo error, got %v", err)
	}
	if !strings.Contains(err.Error(), "list channels") {
		t.Fatalf("expected contextual message, got %v", err)
	}
}

func TestChannelServiceUpdateValidatesID(t *testing.T) {
	repo := &stubChannelRepo{}
	svc := NewChannelService(repo)

	err := svc.Update(context.Background(), &Channel{
		ID:            0,
		Title:         "ABC",
		ChannelNumber: 7,
		Description:   "news",
	})
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if repo.updateCalled {
		t.Fatalf("repo update should not be called")
	}
}

func TestChannelServiceUpdateValidatesFields(t *testing.T) {
	repo := &stubChannelRepo{}
	svc := NewChannelService(repo)

	err := svc.Update(context.Background(), &Channel{
		ID:            1,
		Title:         "",
		ChannelNumber: 7,
		Description:   "news",
	})
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if repo.updateCalled {
		t.Fatalf("repo update should not be called")
	}
}

func TestChannelServiceUpdateCallsRepoForValidInput(t *testing.T) {
	repo := &stubChannelRepo{}
	svc := NewChannelService(repo)

	err := svc.Update(context.Background(), &Channel{
		ID:            1,
		Title:         "ABC",
		ChannelNumber: 7,
		Description:   "news",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !repo.updateCalled {
		t.Fatalf("expected repo update to be called")
	}
}

func TestChannelServiceUpdateWrapsRepoError(t *testing.T) {
	repoErr := errors.New("db update failed")
	repo := &stubChannelRepo{updateErr: repoErr}
	svc := NewChannelService(repo)

	err := svc.Update(context.Background(), &Channel{
		ID:            1,
		Title:         "ABC",
		ChannelNumber: 7,
		Description:   "news",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !repo.updateCalled {
		t.Fatalf("expected repo update to be called")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repo error, got %v", err)
	}
	if !strings.Contains(err.Error(), "update channel") {
		t.Fatalf("expected contextual message, got %v", err)
	}
}

func TestChannelServiceDeleteSuccess(t *testing.T) {
	repo := &stubChannelRepo{}
	svc := NewChannelService(repo)

	if err := svc.Delete(context.Background(), 99); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !repo.deleteCalled {
		t.Fatalf("expected repo delete to be called")
	}
	if repo.gotDeleteID != 99 {
		t.Fatalf("expected id 99, got %d", repo.gotDeleteID)
	}
}

func TestChannelServiceDeleteWrapsRepoError(t *testing.T) {
	repoErr := errors.New("db delete failed")
	repo := &stubChannelRepo{deleteErr: repoErr}
	svc := NewChannelService(repo)

	err := svc.Delete(context.Background(), 99)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !repo.deleteCalled {
		t.Fatalf("expected repo delete to be called")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repo error, got %v", err)
	}
	if !strings.Contains(err.Error(), "delete channel") {
		t.Fatalf("expected contextual message, got %v", err)
	}
}
