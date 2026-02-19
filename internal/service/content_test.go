package service

import (
	"context"
	"errors"
	"testing"
)

type stubRepo struct {
	createCalled bool
	updateCalled bool
	createErr    error
	updateErr    error
}

func (s *stubRepo) Create(context.Context, *Content) error {
	s.createCalled = true
	return s.createErr
}

func (s *stubRepo) GetByID(context.Context, uint) (*Content, error) {
	return &Content{}, nil
}

func (s *stubRepo) List(context.Context, int, int) ([]Content, error) {
	return nil, nil
}

func (s *stubRepo) Update(context.Context, *Content) error {
	s.updateCalled = true
	return s.updateErr
}

func (s *stubRepo) Delete(context.Context, uint) error {
	return nil
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
