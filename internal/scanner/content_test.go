package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/iamseth/tiny-headend/internal/service"
)

type stubContentService struct {
	mu      sync.Mutex
	list    []service.Content
	created []service.Content
}

func (s *stubContentService) Create(_ context.Context, c *service.Content) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *c
	cp.ID = uint(len(s.created) + 1)
	s.created = append(s.created, cp)
	c.ID = cp.ID
	return nil
}

func (s *stubContentService) List(_ context.Context, limit, offset int) ([]service.Content, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if offset >= len(s.list) {
		return []service.Content{}, nil
	}
	end := min(offset+limit, len(s.list))
	out := make([]service.Content, end-offset)
	copy(out, s.list[offset:end])
	return out, nil
}

func (s *stubContentService) Created() []service.Content {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]service.Content, len(s.created))
	copy(out, s.created)
	return out
}

func TestRunOnceSkipsKnownAndAddsNewFiles(t *testing.T) {
	dir := t.TempDir()
	known := filepath.Join(dir, "known.ts")
	if err := osWriteFile(known, []byte("old")); err != nil {
		t.Fatalf("write known file: %v", err)
	}

	newFile := filepath.Join(dir, "new-video.mp4")
	if err := osWriteFile(newFile, []byte("new")); err != nil {
		t.Fatalf("write new file: %v", err)
	}

	svc := &stubContentService{
		list: []service.Content{
			{ID: 10, Title: "known", Path: known, Size: 3, Length: 0},
		},
	}
	scanner, err := NewContentScanner(dir, 100*time.Millisecond, svc)
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}

	if err := scanner.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if err := scanner.RunOnce(context.Background()); err != nil {
		t.Fatalf("second run once: %v", err)
	}

	created := svc.Created()
	if len(created) != 1 {
		t.Fatalf("expected 1 created content, got %d", len(created))
	}
	if created[0].Path != newFile {
		t.Fatalf("expected created path %q, got %q", newFile, created[0].Path)
	}
	if created[0].Title != "new-video" {
		t.Fatalf("expected derived title new-video, got %q", created[0].Title)
	}
}

func TestRunOnceScansRecursively(t *testing.T) {
	dir := t.TempDir()
	nestedDir := filepath.Join(dir, "nested")
	if err := osMkdirAll(nestedDir); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	nestedFile := filepath.Join(nestedDir, "episode01.ts")
	if err := osWriteFile(nestedFile, []byte("a")); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	svc := &stubContentService{}
	scanner, err := NewContentScanner(dir, 100*time.Millisecond, svc)
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}

	if err := scanner.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}

	created := svc.Created()
	if len(created) != 1 {
		t.Fatalf("expected 1 created content, got %d", len(created))
	}
	if created[0].Path != nestedFile {
		t.Fatalf("expected nested file path %q, got %q", nestedFile, created[0].Path)
	}
}

func TestStartRunsInBackground(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "queued.ts")
	if err := osWriteFile(file, []byte("x")); err != nil {
		t.Fatalf("write file: %v", err)
	}

	svc := &stubContentService{}
	scanner, err := NewContentScanner(dir, 20*time.Millisecond, svc)
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}

	scanner.Start(t.Context())

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if len(svc.Created()) == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("scanner did not create content in background")
}

func osWriteFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write file %q: %w", path, err)
	}
	return nil
}

func osMkdirAll(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create directory %q: %w", path, err)
	}
	return nil
}
