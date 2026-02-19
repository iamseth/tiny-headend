package service

import (
	"context"
	"fmt"
	"strings"
)

type Content struct {
	ID     uint    `json:"id"`
	Title  string  `json:"title"`
	Path   string  `json:"path"`
	Size   int64   `json:"size"`
	Length float64 `json:"length"`
}

type ContentRepo interface {
	Create(ctx context.Context, c *Content) error
	GetByID(ctx context.Context, id uint) (*Content, error)
	List(ctx context.Context, limit, offset int) ([]Content, error)
	Update(ctx context.Context, c *Content) error
	Delete(ctx context.Context, id uint) error
}

type ContentService struct {
	repo ContentRepo
}

func NewContentService(repo ContentRepo) *ContentService {
	return &ContentService{repo: repo}
}

func (s *ContentService) Create(ctx context.Context, c *Content) error {
	if err := validateContent(c); err != nil {
		return err
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return fmt.Errorf("create content: %w", err)
	}
	return nil
}

func (s *ContentService) Get(ctx context.Context, id uint) (*Content, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get content by id: %w", err)
	}
	return c, nil
}

func (s *ContentService) List(ctx context.Context, limit, offset int) ([]Content, error) {
	contents, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list content: %w", err)
	}
	return contents, nil
}

func (s *ContentService) Update(ctx context.Context, c *Content) error {
	if c == nil || c.ID == 0 {
		return ErrValidation("id must be greater than zero")
	}
	if err := validateContent(c); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, c); err != nil {
		return fmt.Errorf("update content: %w", err)
	}
	return nil
}

func (s *ContentService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete content: %w", err)
	}
	return nil
}

func validateContent(c *Content) error {
	if c == nil {
		return ErrValidation("content is required")
	}
	if strings.TrimSpace(c.Title) == "" {
		return ErrValidation("title is required")
	}
	if strings.TrimSpace(c.Path) == "" {
		return ErrValidation("path is required")
	}
	if c.Size < 0 {
		return ErrValidation("size must be non-negative")
	}
	if c.Length < 0 {
		return ErrValidation("length must be non-negative")
	}
	return nil
}
