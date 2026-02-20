package scanner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/iamseth/tiny-headend/internal/service"
)

type stubContentService struct {
	mu              sync.Mutex
	list            []service.Content
	created         []service.Content
	listErr         error
	listErrByOffset map[int]error
	listCalls       []listCall
	createErr       error
	createErrByPath map[string]error
	createCalls     int
	onCreate        func()
}

type listCall struct {
	limit  int
	offset int
}

func (s *stubContentService) Create(_ context.Context, c *service.Content) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.createCalls++
	if s.onCreate != nil {
		s.onCreate()
	}
	if s.createErr != nil {
		return s.createErr
	}
	if err, ok := s.createErrByPath[c.Path]; ok {
		return err
	}

	cp := *c
	cp.ID = uint(len(s.created) + 1)
	s.created = append(s.created, cp)
	c.ID = cp.ID
	return nil
}

func (s *stubContentService) List(_ context.Context, limit, offset int) ([]service.Content, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.listCalls = append(s.listCalls, listCall{limit: limit, offset: offset})
	if s.listErr != nil {
		return nil, s.listErr
	}
	if err, ok := s.listErrByOffset[offset]; ok {
		return nil, err
	}

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

func (s *stubContentService) ListCalls() []listCall {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]listCall, len(s.listCalls))
	copy(out, s.listCalls)
	return out
}

func (s *stubContentService) CreateCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.createCalls
}

func TestNewContentScannerValidation(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "file.ts")
	if err := osWriteFile(filePath, []byte("x")); err != nil {
		t.Fatalf("write file: %v", err)
	}

	validSvc := &stubContentService{}
	tests := []struct {
		name     string
		path     string
		interval time.Duration
		svc      contentService
		wantErr  string
	}{
		{
			name:     "path required",
			path:     "   ",
			interval: time.Second,
			svc:      validSvc,
			wantErr:  "scan path is required",
		},
		{
			name:     "interval required",
			path:     dir,
			interval: 0,
			svc:      validSvc,
			wantErr:  "scan interval must be greater than zero",
		},
		{
			name:     "service required",
			path:     dir,
			interval: time.Second,
			svc:      nil,
			wantErr:  "content service is required",
		},
		{
			name:     "must be directory",
			path:     filePath,
			interval: time.Second,
			svc:      validSvc,
			wantErr:  "scan path is not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewContentScanner(tt.path, tt.interval, tt.svc)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error to contain %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestNewContentScannerStoresAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	relative, err := filepath.Rel(cwd, dir)
	if err != nil {
		t.Fatalf("relative path: %v", err)
	}

	scanner, err := NewContentScanner(relative, time.Second, &stubContentService{})
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}

	if scanner.path != dir {
		t.Fatalf("expected absolute path %q, got %q", dir, scanner.path)
	}
}

func TestNewContentScannerReturnsStatError(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "does-not-exist")
	_, err := NewContentScanner(missingPath, time.Second, &stubContentService{})
	if err == nil {
		t.Fatalf("expected error for missing scan path")
	}
	if !strings.Contains(err.Error(), "stat scan path") {
		t.Fatalf("expected stat error, got %q", err.Error())
	}
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

func TestRunOnceReturnsErrorWhenContextCanceled(t *testing.T) {
	scanner, err := NewContentScanner(t.TempDir(), 100*time.Millisecond, &stubContentService{})
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = scanner.RunOnce(ctx)
	if err == nil {
		t.Fatalf("expected canceled run error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}

func TestRunOnceReturnsSeedError(t *testing.T) {
	expectedErr := errors.New("list failed")
	scanner, err := NewContentScanner(
		t.TempDir(),
		100*time.Millisecond,
		&stubContentService{listErr: expectedErr},
	)
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}

	err = scanner.RunOnce(context.Background())
	if err == nil {
		t.Fatalf("expected run once error")
	}
	if !strings.Contains(err.Error(), "seed known content: list existing content") {
		t.Fatalf("expected wrapped seed/list error, got %q", err.Error())
	}
}

func TestRunOnceReturnsScanCanceledError(t *testing.T) {
	dir := t.TempDir()
	if err := osWriteFile(filepath.Join(dir, "a.ts"), []byte("a")); err != nil {
		t.Fatalf("write file a: %v", err)
	}
	if err := osWriteFile(filepath.Join(dir, "b.ts"), []byte("b")); err != nil {
		t.Fatalf("write file b: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	svc := &stubContentService{
		onCreate: cancel,
	}

	scanner, err := NewContentScanner(dir, 100*time.Millisecond, svc)
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}

	err = scanner.RunOnce(ctx)
	if err == nil {
		t.Fatalf("expected canceled scan error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}

func TestRunOnceContinuesWhenCreateFails(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "new.ts")
	if err := osWriteFile(file, []byte("new")); err != nil {
		t.Fatalf("write file: %v", err)
	}

	svc := &stubContentService{createErr: errors.New("create failed")}
	scanner, err := NewContentScanner(dir, 100*time.Millisecond, svc)
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}

	if err := scanner.RunOnce(context.Background()); err != nil {
		t.Fatalf("first run once: %v", err)
	}
	if err := scanner.RunOnce(context.Background()); err != nil {
		t.Fatalf("second run once: %v", err)
	}

	if len(svc.Created()) != 0 {
		t.Fatalf("expected no created content when create fails")
	}
	if svc.CreateCalls() != 2 {
		t.Fatalf("expected two create attempts, got %d", svc.CreateCalls())
	}
}

func TestRunOnceSkipsMissingRootPath(t *testing.T) {
	dir := t.TempDir()
	svc := &stubContentService{}
	scanner, err := NewContentScanner(dir, 100*time.Millisecond, svc)
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("remove scan root: %v", err)
	}

	if err := scanner.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(svc.Created()) != 0 {
		t.Fatalf("expected no created content, got %d", len(svc.Created()))
	}
}

func TestRunHandlesInitialErrorAndStopsOnCancel(t *testing.T) {
	scanner, err := NewContentScanner(
		t.TempDir(),
		20*time.Millisecond,
		&stubContentService{listErr: errors.New("list failed")},
	)
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		scanner.run(ctx)
		close(done)
	}()

	time.Sleep(40 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("scanner run did not stop after cancel")
	}
}

func TestSeedKnownUsesPaginationAndSkipsEmptyPath(t *testing.T) {
	dir := t.TempDir()
	list := make([]service.Content, listPageSize+1)
	for i := range list {
		list[i] = service.Content{
			ID:    uint(i + 1),
			Title: fmt.Sprintf("video-%03d", i),
			Path:  filepath.Join(dir, fmt.Sprintf("video-%03d.ts", i)),
			Size:  int64(i + 1),
		}
	}
	list[123].Path = " "

	svc := &stubContentService{list: list}
	scanner, err := NewContentScanner(dir, 100*time.Millisecond, svc)
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}

	if err := scanner.seedKnown(context.Background()); err != nil {
		t.Fatalf("seed known: %v", err)
	}

	if len(scanner.knownPath) != listPageSize {
		t.Fatalf("expected %d known paths, got %d", listPageSize, len(scanner.knownPath))
	}
	if _, ok := scanner.knownPath[" "]; ok {
		t.Fatalf("unexpected empty path in known set")
	}

	calls := svc.ListCalls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 list calls, got %d", len(calls))
	}
	if calls[0].limit != listPageSize || calls[0].offset != 0 {
		t.Fatalf("expected first list call limit=%d offset=0, got limit=%d offset=%d", listPageSize, calls[0].limit, calls[0].offset)
	}
	if calls[1].limit != listPageSize || calls[1].offset != listPageSize {
		t.Fatalf("expected second list call limit=%d offset=%d, got limit=%d offset=%d", listPageSize, listPageSize, calls[1].limit, calls[1].offset)
	}
}

func TestScanSkipsUnreadablePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based walk errors are not stable on windows")
	}

	dir := t.TempDir()
	readableFile := filepath.Join(dir, "readable.ts")
	if err := osWriteFile(readableFile, []byte("ok")); err != nil {
		t.Fatalf("write readable file: %v", err)
	}

	unreadableDir := filepath.Join(dir, "unreadable")
	if err := osMkdirAll(unreadableDir); err != nil {
		t.Fatalf("mkdir unreadable dir: %v", err)
	}
	if err := osWriteFile(filepath.Join(unreadableDir, "hidden.ts"), []byte("hidden")); err != nil {
		t.Fatalf("write hidden file: %v", err)
	}
	if err := os.Chmod(unreadableDir, 0o000); err != nil {
		t.Fatalf("chmod unreadable dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(unreadableDir, 0o755)
	})

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
	if created[0].Path != readableFile {
		t.Fatalf("expected created path %q, got %q", readableFile, created[0].Path)
	}
}

func TestNormalizePathReturnsErrorWhenWorkingDirMissing(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	missingWD := t.TempDir()
	if err := os.Chdir(missingWD); err != nil {
		t.Fatalf("chdir to temp dir: %v", err)
	}
	if err := os.RemoveAll(missingWD); err != nil {
		t.Fatalf("remove temp dir: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("restore working directory: %v", chdirErr)
		}
	}()

	if _, err := normalizePath("relative.ts"); err == nil {
		t.Fatalf("expected normalizePath to fail for missing cwd")
	}
}

func TestTitleFromName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strip extension",
			input: "movie.mp4",
			want:  "movie",
		},
		{
			name:  "fallback when stripped title empty",
			input: ".mp4",
			want:  ".mp4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := titleFromName(tt.input); got != tt.want {
				t.Fatalf("titleFromName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
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
