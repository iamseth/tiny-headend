package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	envDBPath              = "TINY_HEADEND_DB_PATH"
	envHTTPAddr            = "TINY_HEADEND_HTTP_ADDR"
	envConfigPath          = "TINY_HEADEND_CONFIG_PATH"
	envScanPath            = "TINY_HEADEND_SCAN_PATH"
	envScanInterval        = "TINY_HEADEND_SCAN_INTERVAL"
	envDBPingTimeout       = "TINY_HEADEND_DB_PING_TIMEOUT"
	envHealthPingTimeout   = "TINY_HEADEND_HEALTH_PING_TIMEOUT"
	envReadHeaderTimeout   = "TINY_HEADEND_SERVER_READ_HEADER_TIMEOUT"
	envReadTimeout         = "TINY_HEADEND_SERVER_READ_TIMEOUT"
	envWriteTimeout        = "TINY_HEADEND_SERVER_WRITE_TIMEOUT"
	envIdleTimeout         = "TINY_HEADEND_SERVER_IDLE_TIMEOUT"
	envMaxHeaderBytes      = "TINY_HEADEND_SERVER_MAX_HEADER_BYTES"
	envHealthLogInterval   = "TINY_HEADEND_HEALTH_LOG_INTERVAL"
	envServerShutdownTimer = "TINY_HEADEND_SERVER_SHUTDOWN_TIMEOUT"

	defaultDBPath            = "tiny-headend.db"
	defaultHTTPAddr          = ":8080"
	defaultConfigPath        = "$HOME/.tiny-headend.yaml"
	defaultScanPath          = ""
	defaultScanInterval      = 30 * time.Second
	defaultDBPingTimeout     = 3 * time.Second
	defaultHealthPingTimeout = 2 * time.Second
	defaultReadHeaderTimeout = 2 * time.Second
	defaultReadTimeout       = 5 * time.Second
	defaultWriteTimeout      = 10 * time.Second
	defaultIdleTimeout       = 120 * time.Second
	defaultMaxHeaderBytes    = 1 << 20
	defaultHealthLogInterval = 60 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
)

type Config struct {
	DBPath            string
	HTTPAddr          string
	ConfigPath        string
	ScanPath          string
	ScanInterval      time.Duration
	DBPingTimeout     time.Duration
	HealthPingTimeout time.Duration
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	HealthLogInterval time.Duration
	ShutdownTimeout   time.Duration
}

func Default() Config {
	return Config{
		DBPath:            defaultDBPath,
		HTTPAddr:          defaultHTTPAddr,
		ConfigPath:        defaultConfigPath,
		ScanPath:          defaultScanPath,
		ScanInterval:      defaultScanInterval,
		DBPingTimeout:     defaultDBPingTimeout,
		HealthPingTimeout: defaultHealthPingTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
		HealthLogInterval: defaultHealthLogInterval,
		ShutdownTimeout:   defaultShutdownTimeout,
	}
}

func LoadFromEnv() (Config, error) {
	cfg := Default()

	if err := loadStringConfig(&cfg); err != nil {
		return Config{}, err
	}
	if err := loadDurationConfig(&cfg); err != nil {
		return Config{}, err
	}
	if err := loadNumericConfig(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func loadStringConfig(cfg *Config) error {
	var err error

	cfg.DBPath, err = loadString(envDBPath, cfg.DBPath, true)
	if err != nil {
		return err
	}

	cfg.HTTPAddr, err = loadString(envHTTPAddr, cfg.HTTPAddr, true)
	if err != nil {
		return err
	}

	cfg.ConfigPath, err = loadString(envConfigPath, cfg.ConfigPath, true)
	if err != nil {
		return err
	}

	cfg.ScanPath, err = loadString(envScanPath, cfg.ScanPath, false)
	if err != nil {
		return err
	}

	return nil
}

func loadDurationConfig(cfg *Config) error {
	var err error

	cfg.ScanInterval, err = loadDuration(envScanInterval, cfg.ScanInterval)
	if err != nil {
		return err
	}

	cfg.DBPingTimeout, err = loadDuration(envDBPingTimeout, cfg.DBPingTimeout)
	if err != nil {
		return err
	}

	cfg.HealthPingTimeout, err = loadDuration(envHealthPingTimeout, cfg.HealthPingTimeout)
	if err != nil {
		return err
	}

	cfg.ReadHeaderTimeout, err = loadDuration(envReadHeaderTimeout, cfg.ReadHeaderTimeout)
	if err != nil {
		return err
	}

	cfg.ReadTimeout, err = loadDuration(envReadTimeout, cfg.ReadTimeout)
	if err != nil {
		return err
	}

	cfg.WriteTimeout, err = loadDuration(envWriteTimeout, cfg.WriteTimeout)
	if err != nil {
		return err
	}

	cfg.IdleTimeout, err = loadDuration(envIdleTimeout, cfg.IdleTimeout)
	if err != nil {
		return err
	}

	cfg.HealthLogInterval, err = loadDuration(envHealthLogInterval, cfg.HealthLogInterval)
	if err != nil {
		return err
	}

	cfg.ShutdownTimeout, err = loadDuration(envServerShutdownTimer, cfg.ShutdownTimeout)
	if err != nil {
		return err
	}

	return nil
}

func loadNumericConfig(cfg *Config) error {
	var err error

	cfg.MaxHeaderBytes, err = loadInt(envMaxHeaderBytes, cfg.MaxHeaderBytes)
	if err != nil {
		return err
	}

	return nil
}

func loadString(key, defaultValue string, required bool) (string, error) {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}

	value := strings.TrimSpace(raw)
	if value == "" && required {
		return "", fmt.Errorf("environment variable %s must not be empty", key)
	}

	return value, nil
}

func loadDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}

	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("environment variable %s must not be empty", key)
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse environment variable %s as duration: %w", key, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("environment variable %s must be greater than zero", key)
	}

	return duration, nil
}

func loadInt(key string, defaultValue int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}

	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("environment variable %s must not be empty", key)
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse environment variable %s as int: %w", key, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("environment variable %s must be greater than zero", key)
	}

	return parsed, nil
}
