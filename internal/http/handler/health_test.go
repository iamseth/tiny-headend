package handler

import (
	"context"
	"encoding/json"
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandlerGetHealthy(t *testing.T) {
	h := NewHealthHandler(func(context.Context) error { return nil })

	req := httptest.NewRequest(nethttp.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected %d, got %d", nethttp.StatusOK, rec.Code)
	}

	var got map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", got["status"])
	}
}

func TestHealthHandlerGetUnhealthy(t *testing.T) {
	h := NewHealthHandler(func(context.Context) error { return errors.New("db down") })

	req := httptest.NewRequest(nethttp.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	if rec.Code != nethttp.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", nethttp.StatusServiceUnavailable, rec.Code)
	}

	var got map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["status"] != "unhealthy" {
		t.Fatalf("expected status unhealthy, got %q", got["status"])
	}
}
