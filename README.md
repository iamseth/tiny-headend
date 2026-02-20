# tiny-headend

[![Build Status](https://github.com/iamseth/tiny-headend/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/iamseth/tiny-headend/actions/workflows/ci.yml)
[![Code Coverage](https://codecov.io/gh/iamseth/tiny-headend/branch/master/graph/badge.svg)](https://codecov.io/gh/iamseth/tiny-headend)
[![Go Report Card](https://goreportcard.com/badge/github.com/iamseth/tiny-headend)](https://goreportcard.com/report/github.com/iamseth/tiny-headend)
[![Go Version](https://img.shields.io/github/go-mod/go-version/iamseth/tiny-headend)](https://github.com/iamseth/tiny-headend/blob/master/go.mod)

# Configuration

Runtime config is resolved in this order:

1. CLI flags
2. Environment variables
3. Built-in defaults

Durations use Go duration format (examples: `500ms`, `5s`, `2m`).
Automatic scanning is disabled by default; content can always be added manually via the API.

| Environment Variable | Description | Default |
|---|---|---|
| `TINY_HEADEND_DB_PATH` | SQLite database file path | `tiny-headend.db` |
| `TINY_HEADEND_HTTP_ADDR` | HTTP bind address | `:8080` |
| `TINY_HEADEND_CONFIG_PATH` | Default value for `--config` | `$HOME/.tiny-headend.yaml` |
| `TINY_HEADEND_SCAN_ENABLED` | Default value for `--scan-enabled` | `false` |
| `TINY_HEADEND_SCAN_PATH` | Default value for `--scan-path` | `""` |
| `TINY_HEADEND_SCAN_INTERVAL` | Default value for `--scan-interval` | `30s` |
| `TINY_HEADEND_DB_PING_TIMEOUT` | Startup DB ping timeout | `3s` |
| `TINY_HEADEND_HEALTH_PING_TIMEOUT` | Health check DB ping timeout | `2s` |
| `TINY_HEADEND_SERVER_READ_HEADER_TIMEOUT` | HTTP server read-header timeout | `2s` |
| `TINY_HEADEND_SERVER_READ_TIMEOUT` | HTTP server read timeout | `5s` |
| `TINY_HEADEND_SERVER_WRITE_TIMEOUT` | HTTP server write timeout | `10s` |
| `TINY_HEADEND_SERVER_IDLE_TIMEOUT` | HTTP server idle timeout | `120s` |
| `TINY_HEADEND_SERVER_MAX_HEADER_BYTES` | HTTP server max header bytes | `1048576` |
| `TINY_HEADEND_HEALTH_LOG_INTERVAL` | Periodic runtime health log interval | `60s` |
| `TINY_HEADEND_SERVER_SHUTDOWN_TIMEOUT` | Graceful shutdown timeout | `10s` |

Example:

```bash
export TINY_HEADEND_HTTP_ADDR=":9090"
export TINY_HEADEND_DB_PATH="/var/lib/tiny-headend/tiny-headend.db"
export TINY_HEADEND_SCAN_ENABLED="true"
export TINY_HEADEND_SCAN_PATH="/srv/media"
export TINY_HEADEND_SCAN_INTERVAL="45s"
./bin/tiny-headend
```

# API

## Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/healthz` | Health check |
| `POST` | `/content` | Create content |
| `GET` | `/content` | List content |
| `GET` | `/content/{id}` | Get content by ID |
| `PUT` | `/content/{id}` | Full update by ID |
| `DELETE` | `/content/{id}` | Delete by ID |
