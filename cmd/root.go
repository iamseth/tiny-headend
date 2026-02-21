package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/iamseth/tiny-headend/internal/config"
	"github.com/iamseth/tiny-headend/internal/db"
	"github.com/iamseth/tiny-headend/internal/db/model"
	tinyhttp "github.com/iamseth/tiny-headend/internal/http"
	"github.com/iamseth/tiny-headend/internal/service"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var (
	cfgFile   string
	appConfig config.Config
)

var registerFlagsOnce sync.Once

var rootCmd = &cobra.Command{
	Use:   "tiny-headend",
	Short: "A tiny headend server for streaming media content",
	Long: `Tiny Headend is a simple server that allows you to stream media content from your local machine.
It supports a variety of media formats and can be easily configured using a YAML file.

Configuration precedence:
  1. CLI flags
  2. Environment variables
  3. Built-in defaults

Durations use Go duration format (examples: 500ms, 5s, 2m).

Environment variables:
  TINY_HEADEND_DB_PATH
  TINY_HEADEND_HTTP_ADDR
  TINY_HEADEND_CONFIG_PATH
  TINY_HEADEND_DB_PING_TIMEOUT
  TINY_HEADEND_HEALTH_PING_TIMEOUT
  TINY_HEADEND_SERVER_READ_HEADER_TIMEOUT
  TINY_HEADEND_SERVER_READ_TIMEOUT
  TINY_HEADEND_SERVER_WRITE_TIMEOUT
  TINY_HEADEND_SERVER_IDLE_TIMEOUT
  TINY_HEADEND_SERVER_MAX_HEADER_BYTES
  TINY_HEADEND_HEALTH_LOG_INTERVAL
  TINY_HEADEND_SERVER_SHUTDOWN_TIMEOUT`,
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := db.Open(appConfig.DBPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer func() {
			if closeErr := db.Close(g); closeErr != nil {
				slog.Error("failed to close database", "error", closeErr)
			}
		}()

		pingCtx, cancelPing := context.WithTimeout(context.Background(), appConfig.DBPingTimeout)
		if err := db.Ping(pingCtx, g); err != nil {
			cancelPing()
			return fmt.Errorf("database ping failed: %w", err)
		}
		cancelPing()

		if err := db.Migrate(g); err != nil {
			return fmt.Errorf("failed to migrate database: %w", err)
		}

		healthCheck := func(ctx context.Context) error {
			pingCtx, cancel := context.WithTimeout(ctx, appConfig.HealthPingTimeout)
			defer cancel()
			return db.Ping(pingCtx, g)
		}

		deps := tinyhttp.Deps{
			Content:     service.NewContentService(model.NewContentRepo(g)),
			HealthCheck: healthCheck,
		}

		srv := tinyhttp.New(tinyhttp.Config{
			Addr:              appConfig.HTTPAddr,
			ReadHeaderTimeout: appConfig.ReadHeaderTimeout,
			ReadTimeout:       appConfig.ReadTimeout,
			WriteTimeout:      appConfig.WriteTimeout,
			IdleTimeout:       appConfig.IdleTimeout,
			MaxHeaderBytes:    appConfig.MaxHeaderBytes,
		}, deps)

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		errCh := make(chan error, 1)
		startPeriodicHealthLog(ctx, g, appConfig.HealthLogInterval)

		go func() {
			errCh <- srv.ListenAndServe()
		}()

		slog.Info("starting server", "addr", srv.Addr)
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
				return fmt.Errorf("server error: %w", err)
			}
		case <-ctx.Done():
			slog.Info("shutdown signal received", "signal", ctx.Err())
			shutdownCtx, cancel := context.WithTimeout(context.Background(), appConfig.ShutdownTimeout)
			defer cancel()

			if err := srv.Shutdown(shutdownCtx); err != nil {
				return fmt.Errorf("server shutdown error: %w", err)
			}
			slog.Info("server shutdown complete")

			if err := <-errCh; err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
				return fmt.Errorf("server error: %w", err)
			}
		}

		return nil
	},
}

func startPeriodicHealthLog(ctx context.Context, g *gorm.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats, err := db.Stats(g)
				if err != nil {
					slog.Error("runtime health check failed", "error", err)
					continue
				}

				slog.Info("runtime health",
					"goroutines", runtime.NumGoroutine(),
					"db_open_connections", stats.OpenConnections,
					"db_in_use_connections", stats.InUse,
					"db_idle_connections", stats.Idle,
					"db_wait_count", stats.WaitCount,
					"db_wait_duration_ms", stats.WaitDuration.Milliseconds(),
					"db_max_idle_closed", stats.MaxIdleClosed,
					"db_max_lifetime_closed", stats.MaxLifetimeClosed,
				)
			}
		}
	}()
}

func Execute() {
	loadedConfig, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	appConfig = loadedConfig

	registerFlagsOnce.Do(registerRootFlags)

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	err = rootCmd.Execute()
	if err != nil {
		slog.Error("command execution failed", "error", err)
		os.Exit(1)
	}

	slog.Info("config file", "path", cfgFile)
}

func registerRootFlags() {
	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		appConfig.ConfigPath,
		"config file (default is $HOME/.tiny-headend.yaml)",
	)
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
