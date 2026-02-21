# AGENTS.md

Guidance for coding agents working in this repository.

## Project Snapshot

- Language: Go (`go 1.25.6`)
- App type: small HTTP API + CLI entrypoint
- Main binary: `tiny-headend`
- Core stack: `chi` router, `cobra` CLI, `gorm` with SQLite

## Repository Layout

- `cmd/`: CLI bootstrap and runtime wiring
- `internal/config/`: config defaults and environment loading
- `internal/http/`: HTTP server, middleware, handlers
- `internal/service/`: business logic and validation
- `internal/db/`: DB layer and models

Keep new code inside `internal/` unless there is a strong reason to expose it publicly.

## Build, Test, and Validation

Prefer `make` targets over ad-hoc commands:

- `make test` - run unit tests with coverage output
- `make build` - build `bin/tiny-headend`
- `make fmt` - format code
- `make vet` - static checks via `go vet`
- `make tidy` - normalize/verify dependencies

Default validation for code changes:

1. `make fmt`
2. `make test`
3. `make build`

Run narrower commands only when explicitly asked or when iterating quickly.

## Coding Conventions

- Follow existing package boundaries and dependency direction: handler -> service -> db/model.
- Prefer small, focused functions and explicit error wrapping with `%w`.
- Use `context.Context` for service/db-facing operations.
- For HTTP handlers, preserve existing response behavior and status code conventions.
- Keep JSON contracts stable unless the task explicitly requires API changes.
- Avoid introducing global mutable state.

## HTTP/API Patterns To Preserve

- Set `Content-Type: application/json` for JSON responses.
- Validate and bound user input (IDs, pagination, body size, unknown JSON fields).
- Map service errors to HTTP status codes consistently (validation -> 400, not found -> 404, unexpected -> 500).

## Config and Runtime Behavior

- Configuration precedence is: CLI flags, then env vars, then defaults.
- Durations are Go duration strings (for example `500ms`, `5s`, `2m`).
- Keep config additions coherent across:
  - `internal/config/config.go`
  - CLI/help text in `cmd/root.go`
  - `README.md` configuration docs

## Testing Expectations

- Add or update tests in the same area as the code change.
- Prefer table-driven tests where they improve readability.
- Keep tests deterministic; avoid timing-sensitive or environment-dependent behavior.

## Agent Workflow

- Read this file and `README.md` before making non-trivial changes.
- Make minimal, targeted edits; avoid unrelated refactors.
- If behavior changes, update docs in the same change.
- Do not commit or push unless explicitly requested by the user.
