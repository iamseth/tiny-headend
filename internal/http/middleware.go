package http

import (
	"fmt"
	"log/slog"
	nethttp "net/http"
	"time"
)

type statusRecorder struct {
	nethttp.ResponseWriter
	statusCode int
	size       int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = nethttp.StatusOK
	}

	n, err := r.ResponseWriter.Write(b)
	r.size += n
	if err != nil {
		return n, fmt.Errorf("write response body: %w", err)
	}
	return n, nil
}

func requestLogger(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		start := time.Now()
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     nethttp.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", recorder.statusCode,
			"bytes", recorder.size,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}

func recoverPanic(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("panic recovered",
					"panic", recovered,
					"method", r.Method,
					"path", r.URL.Path,
				)
				nethttp.Error(w, "internal error", nethttp.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
