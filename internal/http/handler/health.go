package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	nethttp "net/http"
)

type healthCheckFunc func(context.Context) error

type HealthHandler struct {
	check healthCheckFunc
}

func NewHealthHandler(check healthCheckFunc) *HealthHandler {
	return &HealthHandler{check: check}
}

func (h *HealthHandler) Get(w nethttp.ResponseWriter, r *nethttp.Request) {
	status := nethttp.StatusOK
	resp := map[string]string{"status": "ok"}

	if h.check != nil {
		if err := h.check(r.Context()); err != nil {
			status = nethttp.StatusServiceUnavailable
			resp["status"] = "unhealthy"
			slog.Error("health check failed", "error", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encode health response", "error", err)
	}
}
