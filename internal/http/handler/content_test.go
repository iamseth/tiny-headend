package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/iamseth/tiny-headend/internal/service"
)

const (
	validContentJSON        = `{"title":"t","path":"/tmp/f.ts","size":1,"length":1}`
	invalidTitleContentJSON = `{"title":" ","path":"/tmp/f.ts","size":1,"length":1}`
	trailingContentJSON     = validContentJSON +
		`{"title":"t2","path":"/tmp/f2.ts","size":2,"length":2}`
)

type stubContentRepo struct {
	createFn func(context.Context, *service.Content) error
	getByID  func(context.Context, uint) (*service.Content, error)
	listFn   func(context.Context, int, int) ([]service.Content, error)
	updateFn func(context.Context, *service.Content) error
	deleteFn func(context.Context, uint) error
}

func (s *stubContentRepo) Create(ctx context.Context, c *service.Content) error {
	if s.createFn == nil {
		return nil
	}
	return s.createFn(ctx, c)
}

func (s *stubContentRepo) GetByID(ctx context.Context, id uint) (*service.Content, error) {
	if s.getByID == nil {
		return nil, service.ErrNotFound
	}
	return s.getByID(ctx, id)
}

func (s *stubContentRepo) List(ctx context.Context, limit, offset int) ([]service.Content, error) {
	if s.listFn == nil {
		return nil, nil
	}
	return s.listFn(ctx, limit, offset)
}

func (s *stubContentRepo) Update(ctx context.Context, c *service.Content) error {
	if s.updateFn == nil {
		return nil
	}
	return s.updateFn(ctx, c)
}

func (s *stubContentRepo) Delete(ctx context.Context, id uint) error {
	if s.deleteFn == nil {
		return nil
	}
	return s.deleteFn(ctx, id)
}

func newTestRouter(h *ContentHandler) nethttp.Handler {
	r := chi.NewRouter()
	r.Post("/content", h.Create)
	r.Get("/content", h.List)
	r.Get("/content/{id}", h.Get)
	r.Put("/content/{id}", h.Update)
	r.Delete("/content/{id}", h.Delete)
	return r
}

type failingResponseWriter struct {
	header nethttp.Header
	status int
}

func (w *failingResponseWriter) Header() nethttp.Header {
	if w.header == nil {
		w.header = make(nethttp.Header)
	}
	return w.header
}

func (w *failingResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

func (w *failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestContentHandlerGetInvalidIDReturnsBadRequest(t *testing.T) {
	testCases := []string{"/content/not-a-number", "/content/0"}

	for _, path := range testCases {
		t.Run(path, func(t *testing.T) {
			repo := &stubContentRepo{
				getByID: func(context.Context, uint) (*service.Content, error) {
					t.Fatalf("GetByID should not be called for invalid id")
					return nil, service.ErrNotFound
				},
			}
			h := NewContentHandler(service.NewContentService(repo))

			req := httptest.NewRequest(nethttp.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			newTestRouter(h).ServeHTTP(rec, req)

			if rec.Code != nethttp.StatusBadRequest {
				t.Fatalf("expected %d, got %d", nethttp.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestContentHandlerGetNotFoundReturnsNotFound(t *testing.T) {
	repo := &stubContentRepo{
		getByID: func(context.Context, uint) (*service.Content, error) {
			return nil, service.ErrNotFound
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	req := httptest.NewRequest(nethttp.MethodGet, "/content/123", nil)
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusNotFound {
		t.Fatalf("expected %d, got %d", nethttp.StatusNotFound, rec.Code)
	}
}

func TestContentHandlerCreateReturnsCreatedContentWithID(t *testing.T) {
	repo := &stubContentRepo{
		createFn: func(_ context.Context, c *service.Content) error {
			c.ID = 42
			return nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	body := bytes.NewBufferString(`{"title":"t","path":"/tmp/f.ts","size":123,"length":9.1}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/content", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("expected %d, got %d", nethttp.StatusCreated, rec.Code)
	}

	var got service.Content
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != 42 {
		t.Fatalf("expected id 42, got %d", got.ID)
	}
}

func TestContentHandlerCreateBadJSONReturnsBadRequest(t *testing.T) {
	repo := &stubContentRepo{
		createFn: func(context.Context, *service.Content) error {
			t.Fatalf("Create should not be called for bad json")
			return nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	body := bytes.NewBufferString(`{"title":"t",`)
	req := httptest.NewRequest(nethttp.MethodPost, "/content", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected %d, got %d", nethttp.StatusBadRequest, rec.Code)
	}
}

func TestContentHandlerCreateTooLargeBodyReturnsRequestEntityTooLarge(t *testing.T) {
	repo := &stubContentRepo{
		createFn: func(context.Context, *service.Content) error {
			t.Fatalf("Create should not be called for oversized body")
			return nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	body := bytes.NewBufferString(
		`{"title":"t","path":"` + strings.Repeat("a", maxContentBodyBytes) + `","size":1,"length":1}`,
	)
	req := httptest.NewRequest(nethttp.MethodPost, "/content", body)
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusRequestEntityTooLarge {
		t.Fatalf("expected %d, got %d", nethttp.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestContentHandlerCreateUnknownFieldReturnsBadRequest(t *testing.T) {
	repo := &stubContentRepo{
		createFn: func(context.Context, *service.Content) error {
			t.Fatalf("Create should not be called for unknown json fields")
			return nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	body := bytes.NewBufferString(`{"title":"t","path":"/tmp/f.ts","size":1,"length":1,"extra":"x"}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/content", body)
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected %d, got %d", nethttp.StatusBadRequest, rec.Code)
	}
}

func TestContentHandlerCreateTrailingJSONReturnsBadRequest(t *testing.T) {
	repo := &stubContentRepo{
		createFn: func(context.Context, *service.Content) error {
			t.Fatalf("Create should not be called for trailing json")
			return nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	body := bytes.NewBufferString(trailingContentJSON)
	req := httptest.NewRequest(nethttp.MethodPost, "/content", body)
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected %d, got %d", nethttp.StatusBadRequest, rec.Code)
	}
}

func TestContentHandlerCreateValidationErrorReturnsBadRequest(t *testing.T) {
	repo := &stubContentRepo{
		createFn: func(context.Context, *service.Content) error {
			t.Fatalf("Create should not be called for validation errors")
			return nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	body := bytes.NewBufferString(`{"title":" ","path":"/tmp/f.ts","size":1,"length":1}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/content", body)
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected %d, got %d", nethttp.StatusBadRequest, rec.Code)
	}
}

func TestContentHandlerGetValidationErrorReturnsBadRequest(t *testing.T) {
	repo := &stubContentRepo{
		getByID: func(context.Context, uint) (*service.Content, error) {
			return nil, service.ErrValidation("bad input")
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	req := httptest.NewRequest(nethttp.MethodGet, "/content/1", nil)
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected %d, got %d", nethttp.StatusBadRequest, rec.Code)
	}
}

func TestContentHandlerGetUnknownErrorReturnsInternalServerError(t *testing.T) {
	repo := &stubContentRepo{
		getByID: func(context.Context, uint) (*service.Content, error) {
			return nil, errors.New("boom")
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	req := httptest.NewRequest(nethttp.MethodGet, "/content/1", nil)
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", nethttp.StatusInternalServerError, rec.Code)
	}
}

func TestContentHandlerListReturnsContent(t *testing.T) {
	repo := &stubContentRepo{
		listFn: func(context.Context, int, int) ([]service.Content, error) {
			return []service.Content{
				{ID: 1, Title: "one", Path: "/tmp/one.ts", Size: 10, Length: 1.1},
				{ID: 2, Title: "two", Path: "/tmp/two.ts", Size: 20, Length: 2.2},
			}, nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	req := httptest.NewRequest(nethttp.MethodGet, "/content", nil)
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected %d, got %d", nethttp.StatusOK, rec.Code)
	}

	var got []service.Content
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
}

func TestContentHandlerListDefaultsAndParsesPagination(t *testing.T) {
	testCases := []struct {
		name           string
		path           string
		expectedLimit  int
		expectedOffset int
	}{
		{name: "defaults", path: "/content", expectedLimit: 100, expectedOffset: 0},
		{name: "explicit values", path: "/content?limit=5&offset=3", expectedLimit: 5, expectedOffset: 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubContentRepo{
				listFn: func(_ context.Context, limit, offset int) ([]service.Content, error) {
					if limit != tc.expectedLimit || offset != tc.expectedOffset {
						t.Fatalf("expected limit/offset %d/%d, got %d/%d", tc.expectedLimit, tc.expectedOffset, limit, offset)
					}
					return nil, nil
				},
			}
			h := NewContentHandler(service.NewContentService(repo))

			req := httptest.NewRequest(nethttp.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			newTestRouter(h).ServeHTTP(rec, req)

			if rec.Code != nethttp.StatusOK {
				t.Fatalf("expected %d, got %d", nethttp.StatusOK, rec.Code)
			}
		})
	}
}

func TestContentHandlerListInvalidPaginationReturnsBadRequest(t *testing.T) {
	testCases := []string{
		"/content?limit=0",
		"/content?limit=-1",
		"/content?limit=501",
		"/content?limit=nope",
		"/content?offset=-1",
		"/content?offset=nope",
	}

	for _, path := range testCases {
		t.Run(path, func(t *testing.T) {
			repo := &stubContentRepo{
				listFn: func(context.Context, int, int) ([]service.Content, error) {
					t.Fatalf("List should not be called for invalid pagination")
					return nil, nil
				},
			}
			h := NewContentHandler(service.NewContentService(repo))

			req := httptest.NewRequest(nethttp.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			newTestRouter(h).ServeHTTP(rec, req)

			if rec.Code != nethttp.StatusBadRequest {
				t.Fatalf("expected %d, got %d", nethttp.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestContentHandlerListNilSliceEncodesAsEmptyArray(t *testing.T) {
	repo := &stubContentRepo{
		listFn: func(context.Context, int, int) ([]service.Content, error) {
			return nil, nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	req := httptest.NewRequest(nethttp.MethodGet, "/content", nil)
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected %d, got %d", nethttp.StatusOK, rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("expected empty array response, got %q", rec.Body.String())
	}
}

func TestContentHandlerUpdateInvalidIDReturnsBadRequest(t *testing.T) {
	testCases := []string{"/content/not-a-number", "/content/0"}

	for _, path := range testCases {
		t.Run(path, func(t *testing.T) {
			repo := &stubContentRepo{
				updateFn: func(context.Context, *service.Content) error {
					t.Fatalf("Update should not be called for invalid id")
					return nil
				},
			}
			h := NewContentHandler(service.NewContentService(repo))

			req := httptest.NewRequest(nethttp.MethodPut, path, bytes.NewBufferString(validContentJSON))
			rec := httptest.NewRecorder()
			newTestRouter(h).ServeHTTP(rec, req)

			if rec.Code != nethttp.StatusBadRequest {
				t.Fatalf("expected %d, got %d", nethttp.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestContentHandlerUpdateBadJSONReturnsBadRequest(t *testing.T) {
	repo := &stubContentRepo{
		updateFn: func(context.Context, *service.Content) error {
			t.Fatalf("Update should not be called for bad json")
			return nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	req := httptest.NewRequest(nethttp.MethodPut, "/content/1", bytes.NewBufferString(`{"title":"t",`))
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected %d, got %d", nethttp.StatusBadRequest, rec.Code)
	}
}

func TestContentHandlerUpdateTooLargeBodyReturnsRequestEntityTooLarge(t *testing.T) {
	repo := &stubContentRepo{
		updateFn: func(context.Context, *service.Content) error {
			t.Fatalf("Update should not be called for oversized body")
			return nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	body := bytes.NewBufferString(
		`{"title":"t","path":"` + strings.Repeat("a", maxContentBodyBytes) + `","size":1,"length":1}`,
	)
	req := httptest.NewRequest(nethttp.MethodPut, "/content/1", body)
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusRequestEntityTooLarge {
		t.Fatalf("expected %d, got %d", nethttp.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestContentHandlerUpdateValidationErrorReturnsBadRequest(t *testing.T) {
	repo := &stubContentRepo{
		updateFn: func(context.Context, *service.Content) error {
			t.Fatalf("Update should not be called for service validation errors")
			return nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	req := httptest.NewRequest(nethttp.MethodPut, "/content/1", bytes.NewBufferString(invalidTitleContentJSON))
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected %d, got %d", nethttp.StatusBadRequest, rec.Code)
	}
}

func TestContentHandlerUpdateNotFoundReturnsNotFound(t *testing.T) {
	repo := &stubContentRepo{
		updateFn: func(context.Context, *service.Content) error {
			return service.ErrNotFound
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	req := httptest.NewRequest(nethttp.MethodPut, "/content/12", bytes.NewBufferString(validContentJSON))
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusNotFound {
		t.Fatalf("expected %d, got %d", nethttp.StatusNotFound, rec.Code)
	}
}

func TestContentHandlerUpdateReturnsUpdatedContent(t *testing.T) {
	repo := &stubContentRepo{
		updateFn: func(_ context.Context, c *service.Content) error {
			if c.ID != 12 {
				t.Fatalf("expected id 12, got %d", c.ID)
			}
			return nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	req := httptest.NewRequest(nethttp.MethodPut, "/content/12", bytes.NewBufferString(validContentJSON))
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected %d, got %d", nethttp.StatusOK, rec.Code)
	}

	var got service.Content
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != 12 || got.Title != "t" || got.Path != "/tmp/f.ts" || got.Size != 1 || got.Length != 1 {
		t.Fatalf("unexpected updated response: %+v", got)
	}
}

func TestContentHandlerDeleteInvalidIDReturnsBadRequest(t *testing.T) {
	testCases := []string{"/content/not-a-number", "/content/0"}

	for _, path := range testCases {
		t.Run(path, func(t *testing.T) {
			repo := &stubContentRepo{
				deleteFn: func(context.Context, uint) error {
					t.Fatalf("Delete should not be called for invalid id")
					return nil
				},
			}
			h := NewContentHandler(service.NewContentService(repo))

			req := httptest.NewRequest(nethttp.MethodDelete, path, nil)
			rec := httptest.NewRecorder()
			newTestRouter(h).ServeHTTP(rec, req)

			if rec.Code != nethttp.StatusBadRequest {
				t.Fatalf("expected %d, got %d", nethttp.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestContentHandlerDeleteNotFoundReturnsNotFound(t *testing.T) {
	repo := &stubContentRepo{
		deleteFn: func(context.Context, uint) error {
			return service.ErrNotFound
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	req := httptest.NewRequest(nethttp.MethodDelete, "/content/9", nil)
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusNotFound {
		t.Fatalf("expected %d, got %d", nethttp.StatusNotFound, rec.Code)
	}
}

func TestContentHandlerDeleteReturnsNoContent(t *testing.T) {
	repo := &stubContentRepo{
		deleteFn: func(_ context.Context, id uint) error {
			if id != 9 {
				t.Fatalf("expected id 9, got %d", id)
			}
			return nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	req := httptest.NewRequest(nethttp.MethodDelete, "/content/9", nil)
	rec := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusNoContent {
		t.Fatalf("expected %d, got %d", nethttp.StatusNoContent, rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rec.Body.String())
	}
}

func TestContentHandlerCreateLogsEncodeErrors(t *testing.T) {
	repo := &stubContentRepo{
		createFn: func(_ context.Context, c *service.Content) error {
			c.ID = 42
			return nil
		},
	}
	h := NewContentHandler(service.NewContentService(repo))

	var logBuf bytes.Buffer
	origLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	defer slog.SetDefault(origLogger)

	req := httptest.NewRequest(
		nethttp.MethodPost,
		"/content",
		bytes.NewBufferString(`{"title":"t","path":"/tmp/f.ts","size":123,"length":9.1}`),
	)
	w := &failingResponseWriter{}
	h.Create(w, req)

	if w.status != nethttp.StatusCreated {
		t.Fatalf("expected status %d, got %d", nethttp.StatusCreated, w.status)
	}
	if !strings.Contains(logBuf.String(), "encode create response") {
		t.Fatalf("expected encode error to be logged, got log: %q", logBuf.String())
	}
}
