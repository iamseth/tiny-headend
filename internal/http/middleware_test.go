package http

import (
	"bytes"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLoggerLogsRequest(t *testing.T) {
	var logBuf bytes.Buffer
	origLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	defer slog.SetDefault(origLogger)

	h := requestLogger(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(nethttp.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(nethttp.MethodGet, "/content?limit=1", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "tiny-headend-test")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("expected %d, got %d", nethttp.StatusCreated, rec.Code)
	}

	logLine := logBuf.String()
	wants := []string{
		"http request",
		"method=GET",
		"path=/content",
		"query=\"limit=1\"",
		"status=201",
		"bytes=2",
		"remote_addr=127.0.0.1:12345",
		"user_agent=tiny-headend-test",
	}
	for _, want := range wants {
		if !strings.Contains(logLine, want) {
			t.Fatalf("expected log output to contain %q, got %q", want, logLine)
		}
	}
}

func TestRecoverPanicReturnsInternalServerError(t *testing.T) {
	var logBuf bytes.Buffer
	origLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	defer slog.SetDefault(origLogger)

	h := requestLogger(recoverPanic(nethttp.HandlerFunc(func(nethttp.ResponseWriter, *nethttp.Request) {
		panic("boom")
	})))

	req := httptest.NewRequest(nethttp.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", nethttp.StatusInternalServerError, rec.Code)
	}

	logLine := logBuf.String()
	if !strings.Contains(logLine, "panic recovered") {
		t.Fatalf("expected panic recovery log, got %q", logLine)
	}
	if !strings.Contains(logLine, "http request") {
		t.Fatalf("expected request log, got %q", logLine)
	}
}
