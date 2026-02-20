package service

import (
	"context"
	"fmt"
	"strings"
)

type Channel struct {
	ID            uint   `json:"id"`
	Title         string `json:"title"`
	ChannelNumber uint   `json:"channelNumber"`
	Description   string `json:"description"`
}

type ChannelRepo interface {
	Create(ctx context.Context, c *Channel) error
	GetByID(ctx context.Context, id uint) (*Channel, error)
	List(ctx context.Context, limit, offset int) ([]Channel, error)
	Update(ctx context.Context, c *Channel) error
	Delete(ctx context.Context, id uint) error
}

type ChannelService struct {
	repo ChannelRepo
}

func NewChannelService(repo ChannelRepo) *ChannelService {
	return &ChannelService{repo: repo}
}

func (s *ChannelService) Create(ctx context.Context, c *Channel) error {
	if err := validateChannel(c); err != nil {
		return err
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return fmt.Errorf("create channel: %w", err)
	}
	return nil
}

func (s *ChannelService) Get(ctx context.Context, id uint) (*Channel, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get channel by id: %w", err)
	}
	return c, nil
}

func (s *ChannelService) List(ctx context.Context, limit, offset int) ([]Channel, error) {
	channels, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	return channels, nil
}

func (s *ChannelService) Update(ctx context.Context, c *Channel) error {
	if c == nil || c.ID == 0 {
		return ErrValidation("id must be greater than zero")
	}
	if err := validateChannel(c); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, c); err != nil {
		return fmt.Errorf("update channel: %w", err)
	}
	return nil
}

func (s *ChannelService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete channel: %w", err)
	}
	return nil
}

func validateChannel(c *Channel) error {
	if c == nil {
		return ErrValidation("channel is required")
	}
	if strings.TrimSpace(c.Title) == "" {
		return ErrValidation("title is required")
	}
	if c.ChannelNumber == 0 {
		return ErrValidation("channel number must be greater than zero")
	}
	return nil
}
