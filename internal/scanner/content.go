package scanner

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/iamseth/tiny-headend/internal/service"
)

const listPageSize = 500

type contentService interface {
	Create(ctx context.Context, c *service.Content) error
	List(ctx context.Context, limit, offset int) ([]service.Content, error)
}

type ContentScanner struct {
	path      string
	interval  time.Duration
	svc       contentService
	knownPath map[string]struct{}
	seeded    bool
}

func NewContentScanner(path string, interval time.Duration, svc contentService) (*ContentScanner, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("scan path is required")
	}
	if interval <= 0 {
		return nil, errors.New("scan interval must be greater than zero")
	}
	if svc == nil {
		return nil, errors.New("content service is required")
	}

	absPath, err := normalizePath(path)
	if err != nil {
		return nil, fmt.Errorf("normalize scan path: %w", err)
	}

	st, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat scan path: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("scan path is not a directory: %s", absPath)
	}

	return &ContentScanner{
		path:      absPath,
		interval:  interval,
		svc:       svc,
		knownPath: make(map[string]struct{}),
	}, nil
}

func (s *ContentScanner) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *ContentScanner) RunOnce(ctx context.Context) error {
	if ctx.Err() != nil {
		return fmt.Errorf("run once canceled: %w", ctx.Err())
	}

	if !s.seeded {
		if err := s.seedKnown(ctx); err != nil {
			return fmt.Errorf("seed known content: %w", err)
		}
		s.seeded = true
	}

	return s.scan(ctx)
}

func (s *ContentScanner) run(ctx context.Context) {
	if err := s.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("content scanner initial run failed", "error", err)
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("content scanner run failed", "error", err)
			}
		}
	}
}

func (s *ContentScanner) seedKnown(ctx context.Context) error {
	offset := 0
	for {
		contents, err := s.svc.List(ctx, listPageSize, offset)
		if err != nil {
			return fmt.Errorf("list existing content: %w", err)
		}
		if len(contents) == 0 {
			return nil
		}

		for _, c := range contents {
			if strings.TrimSpace(c.Path) == "" {
				continue
			}
			normalized, err := normalizePath(c.Path)
			if err != nil {
				slog.Warn("content scanner skipped invalid persisted path", "path", c.Path, "error", err)
				continue
			}
			s.knownPath[normalized] = struct{}{}
		}

		offset += len(contents)
		if len(contents) < listPageSize {
			return nil
		}
	}
}

func (s *ContentScanner) scan(ctx context.Context) error {
	err := filepath.WalkDir(s.path, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			slog.Warn("content scanner skipped unreadable path", "path", path, "error", walkErr)
			return nil
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		if ctx.Err() != nil {
			return fmt.Errorf("scan canceled: %w", ctx.Err())
		}

		normalized, err := normalizePath(path)
		if err != nil {
			slog.Warn("content scanner skipped invalid file path", "path", path, "error", err)
			return nil
		}
		if _, exists := s.knownPath[normalized]; exists {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			slog.Warn("content scanner skipped file info read", "path", normalized, "error", err)
			return nil
		}

		content := &service.Content{
			Title:  titleFromName(d.Name()),
			Path:   normalized,
			Size:   info.Size(),
			Length: 0,
		}
		if err := s.svc.Create(ctx, content); err != nil {
			slog.Error("content scanner failed to create content", "path", normalized, "error", err)
			return nil
		}

		s.knownPath[normalized] = struct{}{}
		slog.Info("content scanner discovered file", "path", normalized, "content_id", content.ID)
		return nil
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("scan canceled: %w", err)
		}
		return fmt.Errorf("walk scan path: %w", err)
	}
	return nil
}

func normalizePath(path string) (string, error) {
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}
	return absPath, nil
}

func titleFromName(name string) string {
	title := strings.TrimSpace(strings.TrimSuffix(name, filepath.Ext(name)))
	if title != "" {
		return title
	}
	return name
}
