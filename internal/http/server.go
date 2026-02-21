package http

import (
	"context"
	nethttp "net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/iamseth/tiny-headend/internal/http/handler"
	"github.com/iamseth/tiny-headend/internal/service"
)

// Deps holds the dependencies for the server.
type Deps struct {
	Content     *service.ContentService
	HealthCheck func(ctx context.Context) error
}

// Config holds the configuration for the server.
type Config struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
}

func New(cfg Config, deps Deps) *nethttp.Server {
	router := chi.NewRouter()
	router.Use(requestLogger, recoverPanic)

	contentH := handler.NewContentHandler(deps.Content)
	healthH := handler.NewHealthHandler(deps.HealthCheck)
	router.Get("/healthz", healthH.Get)
	router.Post("/content", contentH.Create)
	router.Get("/content", contentH.List)
	router.Get("/content/{id}", contentH.Get)
	router.Put("/content/{id}", contentH.Update)
	router.Delete("/content/{id}", contentH.Delete)

	return &nethttp.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
}
