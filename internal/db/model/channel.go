package model

import (
	"context"
	"errors"

	"github.com/iamseth/tiny-headend/internal/service"
	"gorm.io/gorm"
)

type Channel struct {
	gorm.Model
	Title         string `gorm:"type:varchar(255);not null" json:"title"`
	ChannelNumber uint   `gorm:"not null" json:"channelNumber"`
	Description   string `gorm:"type:text;not null" json:"description"`
}

type ChannelRepo struct {
	db *gorm.DB
}

func NewChannelRepo(db *gorm.DB) *ChannelRepo {
	return &ChannelRepo{db: db}
}

func (r *ChannelRepo) Create(ctx context.Context, c *service.Channel) error {
	m := &Channel{
		Title:         c.Title,
		ChannelNumber: c.ChannelNumber,
		Description:   c.Description,
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	c.ID = m.ID
	return nil
}

func (r *ChannelRepo) GetByID(ctx context.Context, id uint) (*service.Channel, error) {
	var m Channel
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}
	return &service.Channel{
		ID:            m.ID,
		Title:         m.Title,
		ChannelNumber: m.ChannelNumber,
		Description:   m.Description,
	}, nil
}

func (r *ChannelRepo) List(ctx context.Context, limit, offset int) ([]service.Channel, error) {
	var ms []Channel
	if err := r.db.WithContext(ctx).Order("id ASC").Limit(limit).Offset(offset).Find(&ms).Error; err != nil {
		return nil, err
	}
	channels := make([]service.Channel, len(ms))
	for i, m := range ms {
		channels[i] = service.Channel{
			ID:            m.ID,
			Title:         m.Title,
			ChannelNumber: m.ChannelNumber,
			Description:   m.Description,
		}
	}
	return channels, nil
}

func (r *ChannelRepo) Update(ctx context.Context, c *service.Channel) error {
	res := r.db.WithContext(ctx).Model(&Channel{}).Where("id = ?", c.ID).Updates(map[string]any{
		"title":          c.Title,
		"channel_number": c.ChannelNumber,
		"description":    c.Description,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (r *ChannelRepo) Delete(ctx context.Context, id uint) error {
	res := r.db.WithContext(ctx).Delete(&Channel{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return service.ErrNotFound
	}
	return nil
}
