package service

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stubRepo struct {
	createCalled  bool
	getByIDCalled bool
	listCalled    bool
	updateCalled  bool
	deleteCalled  bool
	createErr     error
	getByIDErr    error
	listErr       error
	updateErr     error
	deleteErr     error
	getContent    *Content
	listContents  []Content
	gotGetID      uint
	gotLimit      int
	gotOffset     int
	gotDeleteID   uint
}

func (s *stubRepo) Create(context.Context, *Content) error {
	s.createCalled = true
	return s.createErr
}

func (s *stubRepo) GetByID(_ context.Context, id uint) (*Content, error) {
	s.getByIDCalled = true
	s.gotGetID = id
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	if s.getContent != nil {
		return s.getContent, nil
	}
	return &Content{}, nil
}

func (s *stubRepo) List(_ context.Context, limit, offset int) ([]Content, error) {
	s.listCalled = true
	s.gotLimit = limit
	s.gotOffset = offset
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.listContents, nil
}

func (s *stubRepo) Update(context.Context, *Content) error {
	s.updateCalled = true
	return s.updateErr
}

func (s *stubRepo) Delete(_ context.Context, id uint) error {
	s.deleteCalled = true
	s.gotDeleteID = id
	return s.deleteErr
}

func TestContentServiceCreateValidatesInput(t *testing.T) {
	cases := []struct {
		name string
		in   *Content
	}{
		{name: "nil content", in: nil},
		{name: "empty title", in: &Content{Title: "", Path: "/tmp/f.ts", Size: 1, Length: 1}},
		{name: "whitespace title", in: &Content{Title: " ", Path: "/tmp/f.ts", Size: 1, Length: 1}},
		{name: "empty path", in: &Content{Title: "ok", Path: "", Size: 1, Length: 1}},
		{name: "negative size", in: &Content{Title: "ok", Path: "/tmp/f.ts", Size: -1, Length: 1}},
		{name: "negative length", in: &Content{Title: "ok", Path: "/tmp/f.ts", Size: 1, Length: -1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubRepo{}
			svc := NewContentService(repo)

			err := svc.Create(context.Background(), tc.in)
			var ve ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected validation error, got: %v", err)
			}
			if repo.createCalled {
				t.Fatalf("repo create should not be called for invalid input")
			}
		})
	}
}

func TestContentServiceCreateCallsRepoForValidInput(t *testing.T) {
	repo := &stubRepo{}
	svc := NewContentService(repo)

	err := svc.Create(context.Background(), &Content{
		Title:  "title",
		Path:   "/tmp/file.ts",
		Size:   10,
		Length: 1.5,
	})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !repo.createCalled {
		t.Fatalf("expected repo create to be called")
	}
}

func TestContentServiceCreateWrapsRepoError(t *testing.T) {
	repoErr := errors.New("db create failed")
	repo := &stubRepo{createErr: repoErr}
	svc := NewContentService(repo)

	err := svc.Create(context.Background(), &Content{
		Title:  "title",
		Path:   "/tmp/file.ts",
		Size:   10,
		Length: 1.5,
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !repo.createCalled {
		t.Fatalf("expected repo create to be called")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repo error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "create content") {
		t.Fatalf("expected contextual message, got: %v", err)
	}
}

func TestContentServiceUpdateValidatesID(t *testing.T) {
	repo := &stubRepo{}
	svc := NewContentService(repo)

	err := svc.Update(context.Background(), &Content{
		ID:     0,
		Title:  "title",
		Path:   "/tmp/file.ts",
		Size:   10,
		Length: 1.5,
	})
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got: %v", err)
	}
	if repo.updateCalled {
		t.Fatalf("repo update should not be called for invalid id")
	}
}

func TestContentServiceUpdateValidatesNilContent(t *testing.T) {
	repo := &stubRepo{}
	svc := NewContentService(repo)

	err := svc.Update(context.Background(), nil)
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got: %v", err)
	}
	if repo.updateCalled {
		t.Fatalf("repo update should not be called for nil content")
	}
}

func TestContentServiceUpdateValidatesFields(t *testing.T) {
	repo := &stubRepo{}
	svc := NewContentService(repo)

	err := svc.Update(context.Background(), &Content{
		ID:     1,
		Title:  "",
		Path:   "/tmp/file.ts",
		Size:   10,
		Length: 1.5,
	})
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got: %v", err)
	}
	if repo.updateCalled {
		t.Fatalf("repo update should not be called for invalid fields")
	}
}

func TestContentServiceUpdateCallsRepoForValidInput(t *testing.T) {
	repo := &stubRepo{}
	svc := NewContentService(repo)

	err := svc.Update(context.Background(), &Content{
		ID:     1,
		Title:  "title",
		Path:   "/tmp/file.ts",
		Size:   10,
		Length: 1.5,
	})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !repo.updateCalled {
		t.Fatalf("expected repo update to be called")
	}
}

func TestContentServiceUpdateWrapsRepoError(t *testing.T) {
	repoErr := errors.New("db update failed")
	repo := &stubRepo{updateErr: repoErr}
	svc := NewContentService(repo)

	err := svc.Update(context.Background(), &Content{
		ID:     1,
		Title:  "title",
		Path:   "/tmp/file.ts",
		Size:   10,
		Length: 1.5,
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !repo.updateCalled {
		t.Fatalf("expected repo update to be called")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repo error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "update content") {
		t.Fatalf("expected contextual message, got: %v", err)
	}
}

func TestContentServiceGetReturnsContent(t *testing.T) {
	expected := &Content{ID: 42, Title: "t", Path: "/tmp/f.ts", Size: 1, Length: 1}
	repo := &stubRepo{getContent: expected}
	svc := NewContentService(repo)

	got, err := svc.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !repo.getByIDCalled {
		t.Fatalf("expected repo get by id to be called")
	}
	if repo.gotGetID != 42 {
		t.Fatalf("expected id 42, got %d", repo.gotGetID)
	}
	if got.ID != expected.ID || got.Title != expected.Title {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
}

func TestContentServiceGetWrapsRepoError(t *testing.T) {
	repoErr := errors.New("db get failed")
	repo := &stubRepo{getByIDErr: repoErr}
	svc := NewContentService(repo)

	_, err := svc.Get(context.Background(), 7)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !repo.getByIDCalled {
		t.Fatalf("expected repo get by id to be called")
	}
	if repo.gotGetID != 7 {
		t.Fatalf("expected id 7, got %d", repo.gotGetID)
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repo error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "get content by id") {
		t.Fatalf("expected contextual message, got: %v", err)
	}
}

func TestContentServiceListReturnsContents(t *testing.T) {
	repo := &stubRepo{
		listContents: []Content{
			{ID: 1, Title: "a", Path: "/a.ts", Size: 1, Length: 1},
			{ID: 2, Title: "b", Path: "/b.ts", Size: 2, Length: 2},
		},
	}
	svc := NewContentService(repo)

	got, err := svc.List(context.Background(), 50, 10)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !repo.listCalled {
		t.Fatalf("expected repo list to be called")
	}
	if repo.gotLimit != 50 || repo.gotOffset != 10 {
		t.Fatalf("expected limit=50 offset=10, got limit=%d offset=%d", repo.gotLimit, repo.gotOffset)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
}

func TestContentServiceListWrapsRepoError(t *testing.T) {
	repoErr := errors.New("db list failed")
	repo := &stubRepo{listErr: repoErr}
	svc := NewContentService(repo)

	_, err := svc.List(context.Background(), 10, 0)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !repo.listCalled {
		t.Fatalf("expected repo list to be called")
	}
	if repo.gotLimit != 10 || repo.gotOffset != 0 {
		t.Fatalf("expected limit=10 offset=0, got limit=%d offset=%d", repo.gotLimit, repo.gotOffset)
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repo error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "list content") {
		t.Fatalf("expected contextual message, got: %v", err)
	}
}

func TestContentServiceDeleteSuccess(t *testing.T) {
	repo := &stubRepo{}
	svc := NewContentService(repo)

	if err := svc.Delete(context.Background(), 99); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !repo.deleteCalled {
		t.Fatalf("expected repo delete to be called")
	}
	if repo.gotDeleteID != 99 {
		t.Fatalf("expected id 99, got %d", repo.gotDeleteID)
	}
}

func TestContentServiceDeleteWrapsRepoError(t *testing.T) {
	repoErr := errors.New("db delete failed")
	repo := &stubRepo{deleteErr: repoErr}
	svc := NewContentService(repo)

	err := svc.Delete(context.Background(), 99)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !repo.deleteCalled {
		t.Fatalf("expected repo delete to be called")
	}
	if repo.gotDeleteID != 99 {
		t.Fatalf("expected id 99, got %d", repo.gotDeleteID)
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repo error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "delete content") {
		t.Fatalf("expected contextual message, got: %v", err)
	}
}

func TestValidationErrorErrorReturnsMessage(t *testing.T) {
	err := ValidationError{Msg: "validation failed"}
	if got := err.Error(); got != "validation failed" {
		t.Fatalf("expected message to round-trip, got %q", got)
	}
}
