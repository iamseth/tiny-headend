package http

import (
	"bytes"
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iamseth/tiny-headend/internal/service"
)

const validContentJSON = `{"title":"t","path":"/tmp/f.ts","size":1,"length":1}`

type serverStubContentRepo struct{}

func (serverStubContentRepo) Create(_ context.Context, c *service.Content) error {
	c.ID = 1
	return nil
}

func (serverStubContentRepo) GetByID(context.Context, uint) (*service.Content, error) {
	return nil, service.ErrNotFound
}

func (serverStubContentRepo) List(context.Context, int, int) ([]service.Content, error) {
	return nil, nil
}

func (serverStubContentRepo) Update(context.Context, *service.Content) error {
	return nil
}

func (serverStubContentRepo) Delete(context.Context, uint) error {
	return nil
}

func TestNewConfiguresServerAndRoutes(t *testing.T) {
	cfg := Config{
		Addr:              ":1234",
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      4 * time.Second,
		IdleTimeout:       5 * time.Second,
		MaxHeaderBytes:    1024,
	}

	srv := New(cfg, Deps{
		Content:     service.NewContentService(serverStubContentRepo{}),
		HealthCheck: func(context.Context) error { return nil },
	})

	if srv.Addr != cfg.Addr {
		t.Fatalf("expected addr %q, got %q", cfg.Addr, srv.Addr)
	}
	if srv.ReadHeaderTimeout != cfg.ReadHeaderTimeout {
		t.Fatalf("expected read header timeout %s, got %s", cfg.ReadHeaderTimeout, srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout != cfg.ReadTimeout {
		t.Fatalf("expected read timeout %s, got %s", cfg.ReadTimeout, srv.ReadTimeout)
	}
	if srv.WriteTimeout != cfg.WriteTimeout {
		t.Fatalf("expected write timeout %s, got %s", cfg.WriteTimeout, srv.WriteTimeout)
	}
	if srv.IdleTimeout != cfg.IdleTimeout {
		t.Fatalf("expected idle timeout %s, got %s", cfg.IdleTimeout, srv.IdleTimeout)
	}
	if srv.MaxHeaderBytes != cfg.MaxHeaderBytes {
		t.Fatalf("expected max header bytes %d, got %d", cfg.MaxHeaderBytes, srv.MaxHeaderBytes)
	}

	healthReq := httptest.NewRequest(nethttp.MethodGet, "/healthz", nil)
	healthRec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != nethttp.StatusOK {
		t.Fatalf("expected %d, got %d", nethttp.StatusOK, healthRec.Code)
	}

	createReq := httptest.NewRequest(
		nethttp.MethodPost,
		"/content",
		bytes.NewBufferString(validContentJSON),
	)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(createRec, createReq)
	if createRec.Code != nethttp.StatusCreated {
		t.Fatalf("expected %d, got %d", nethttp.StatusCreated, createRec.Code)
	}
}
