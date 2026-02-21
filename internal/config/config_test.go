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

func TestLoadFromEnvReturnsErrorOnInvalidBool(t *testing.T) {

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected error for invalid bool")
	}
}

func TestLoadStringConfigReturnsErrorForRequiredVariables(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "db path", key: envDBPath},
		{name: "http addr", key: envHTTPAddr},
		{name: "config path", key: envConfigPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, "   ")
			cfg := Default()
			err := loadStringConfig(&cfg)
			if err == nil {
				t.Fatalf("expected error for %s", tt.key)
			}
			if !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("expected error mentioning %s, got %v", tt.key, err)
			}
		})
	}
}
func TestLoadDurationConfigReturnsErrorForEachDurationVariable(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "db ping timeout", key: envDBPingTimeout},
		{name: "health ping timeout", key: envHealthPingTimeout},
		{name: "read header timeout", key: envReadHeaderTimeout},
		{name: "read timeout", key: envReadTimeout},
		{name: "write timeout", key: envWriteTimeout},
		{name: "idle timeout", key: envIdleTimeout},
		{name: "health log interval", key: envHealthLogInterval},
		{name: "shutdown timeout", key: envServerShutdownTimer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, "0s")
			cfg := Default()
			err := loadDurationConfig(&cfg)
			if err == nil {
				t.Fatalf("expected error for %s", tt.key)
			}
			if !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("expected error mentioning %s, got %v", tt.key, err)
			}
		})
	}
}

func TestLoadNumericConfigReturnsErrorOnInvalidValue(t *testing.T) {
	t.Setenv(envMaxHeaderBytes, "0")
	cfg := Default()

	err := loadNumericConfig(&cfg)
	if err == nil {
		t.Fatal("expected error for max header bytes")
	}
	if !strings.Contains(err.Error(), envMaxHeaderBytes) {
		t.Fatalf("expected error mentioning %s, got %v", envMaxHeaderBytes, err)
	}
}

func TestLoadDuration(t *testing.T) {
	const key = "TEST_DURATION"
	const fallback = 5 * time.Second

	t.Run("uses default when unset", func(t *testing.T) {
		got, err := loadDuration(key, fallback)
		if err != nil {
			t.Fatalf("load duration: %v", err)
		}
		if got != fallback {
			t.Fatalf("expected %v, got %v", fallback, got)
		}
	})

	t.Run("returns error on empty value", func(t *testing.T) {
		t.Setenv(key, "   ")
		_, err := loadDuration(key, fallback)
		if err == nil {
			t.Fatal("expected error for empty duration")
		}
	})

	t.Run("returns error on invalid value", func(t *testing.T) {
		t.Setenv(key, "nope")
		_, err := loadDuration(key, fallback)
		if err == nil {
			t.Fatal("expected parse error for duration")
		}
	})

	t.Run("returns error on non-positive value", func(t *testing.T) {
		t.Setenv(key, "-1s")
		_, err := loadDuration(key, fallback)
		if err == nil {
			t.Fatal("expected non-positive duration error")
		}
	})

	t.Run("parses valid value", func(t *testing.T) {
		t.Setenv(key, " 12s ")
		got, err := loadDuration(key, fallback)
		if err != nil {
			t.Fatalf("load duration: %v", err)
		}
		if got != 12*time.Second {
			t.Fatalf("expected 12s, got %v", got)
		}
	})
}

func TestLoadBool(t *testing.T) {
	const key = "TEST_BOOL"

	t.Run("uses default when unset", func(t *testing.T) {
		got, err := loadBool(key, true)
		if err != nil {
			t.Fatalf("load bool: %v", err)
		}
		if !got {
			t.Fatal("expected default true")
		}
	})

	t.Run("returns error on empty value", func(t *testing.T) {
		t.Setenv(key, " ")
		_, err := loadBool(key, false)
		if err == nil {
			t.Fatal("expected error for empty bool")
		}
	})

	t.Run("returns error on invalid value", func(t *testing.T) {
		t.Setenv(key, "maybe")
		_, err := loadBool(key, false)
		if err == nil {
			t.Fatal("expected parse error for bool")
		}
	})

	t.Run("parses valid value", func(t *testing.T) {
		t.Setenv(key, " true ")
		got, err := loadBool(key, false)
		if err != nil {
			t.Fatalf("load bool: %v", err)
		}
		if !got {
			t.Fatal("expected parsed true")
		}
	})
}

func TestLoadInt(t *testing.T) {
	const key = "TEST_INT"

	t.Run("uses default when unset", func(t *testing.T) {
		got, err := loadInt(key, 123)
		if err != nil {
			t.Fatalf("load int: %v", err)
		}
		if got != 123 {
			t.Fatalf("expected 123, got %d", got)
		}
	})

	t.Run("returns error on empty value", func(t *testing.T) {
		t.Setenv(key, " ")
		_, err := loadInt(key, 1)
		if err == nil {
			t.Fatal("expected error for empty int")
		}
	})

	t.Run("returns error on invalid value", func(t *testing.T) {
		t.Setenv(key, "abc")
		_, err := loadInt(key, 1)
		if err == nil {
			t.Fatal("expected parse error for int")
		}
	})

	t.Run("returns error on non-positive value", func(t *testing.T) {
		t.Setenv(key, "0")
		_, err := loadInt(key, 1)
		if err == nil {
			t.Fatal("expected non-positive int error")
		}
	})

	t.Run("parses valid value", func(t *testing.T) {
		t.Setenv(key, " 321 ")
		got, err := loadInt(key, 1)
		if err != nil {
			t.Fatalf("load int: %v", err)
		}
		if got != 321 {
			t.Fatalf("expected 321, got %d", got)
		}
	})
}
