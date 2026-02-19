package model

import (
	"context"
	"errors"

	"github.com/iamseth/tiny-headend/internal/service"
	"gorm.io/gorm"
)

type Content struct {
	gorm.Model
	Title  string  `gorm:"type:varchar(255);not null" json:"title"`
	Path   string  `gorm:"type:varchar(255);not null" json:"path"`
	Size   int64   `gorm:"not null" json:"size"`
	Length float64 `gorm:"not null" json:"length"`
}

type ContentRepo struct {
	db *gorm.DB
}

func NewContentRepo(db *gorm.DB) *ContentRepo {
	return &ContentRepo{db: db}
}

func (r *ContentRepo) Create(ctx context.Context, c *service.Content) error {
	m := &Content{
		Title:  c.Title,
		Size:   c.Size,
		Path:   c.Path,
		Length: c.Length,
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	c.ID = m.ID
	return nil
}

func (r *ContentRepo) GetByID(ctx context.Context, id uint) (*service.Content, error) {
	var m Content
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}
	return &service.Content{
		ID:     m.ID,
		Title:  m.Title,
		Size:   m.Size,
		Length: m.Length,
		Path:   m.Path,
	}, nil
}

func (r *ContentRepo) List(ctx context.Context, limit, offset int) ([]service.Content, error) {
	var ms []Content
	if err := r.db.WithContext(ctx).Order("id ASC").Limit(limit).Offset(offset).Find(&ms).Error; err != nil {
		return nil, err
	}
	contents := make([]service.Content, len(ms))
	for i, m := range ms {
		contents[i] = service.Content{
			ID:     m.ID,
			Title:  m.Title,
			Size:   m.Size,
			Length: m.Length,
			Path:   m.Path,
		}
	}
	return contents, nil
}

func (r *ContentRepo) Update(ctx context.Context, c *service.Content) error {
	res := r.db.WithContext(ctx).Model(&Content{}).Where("id = ?", c.ID).Updates(map[string]any{
		"title":  c.Title,
		"size":   c.Size,
		"length": c.Length,
		"path":   c.Path,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (r *ContentRepo) Delete(ctx context.Context, id uint) error {
	res := r.db.WithContext(ctx).Delete(&Content{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return service.ErrNotFound
	}
	return nil
}
