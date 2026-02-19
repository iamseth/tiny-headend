package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadFromEnvOverridesDefaults(t *testing.T) {
	t.Setenv(envDBPath, "db.sqlite")
	t.Setenv(envHTTPAddr, ":9090")
	t.Setenv(envConfigPath, "/tmp/tiny-headend.yaml")
	t.Setenv(envScanPath, "/mnt/media")
	t.Setenv(envScanInterval, "45s")
	t.Setenv(envDBPingTimeout, "4s")
	t.Setenv(envHealthPingTimeout, "3s")
	t.Setenv(envReadHeaderTimeout, "1s")
	t.Setenv(envReadTimeout, "6s")
	t.Setenv(envWriteTimeout, "7s")
	t.Setenv(envIdleTimeout, "300s")
	t.Setenv(envMaxHeaderBytes, "65536")
	t.Setenv(envHealthLogInterval, "2m")
	t.Setenv(envServerShutdownTimer, "15s")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("load from env: %v", err)
	}

	want := Config{
		DBPath:            "db.sqlite",
		HTTPAddr:          ":9090",
		ConfigPath:        "/tmp/tiny-headend.yaml",
		ScanPath:          "/mnt/media",
		ScanInterval:      45 * time.Second,
		DBPingTimeout:     4 * time.Second,
		HealthPingTimeout: 3 * time.Second,
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       6 * time.Second,
		WriteTimeout:      7 * time.Second,
		IdleTimeout:       5 * time.Minute,
		MaxHeaderBytes:    65536,
		HealthLogInterval: 2 * time.Minute,
		ShutdownTimeout:   15 * time.Second,
	}
	if cfg != want {
		t.Fatalf("unexpected config: got %+v want %+v", cfg, want)
	}
}

func TestLoadFromEnvReturnsErrorOnInvalidDuration(t *testing.T) {
	t.Setenv(envReadTimeout, "not-a-duration")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
	if !strings.Contains(err.Error(), envReadTimeout) {
		t.Fatalf("expected error mentioning %s, got %v", envReadTimeout, err)
	}
}

func TestLoadFromEnvReturnsErrorOnEmptyRequiredValue(t *testing.T) {
	t.Setenv(envDBPath, "")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected error for empty required value")
	}
	if !strings.Contains(err.Error(), envDBPath) {
		t.Fatalf("expected error mentioning %s, got %v", envDBPath, err)
	}
}
