# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run the server
make run
# or
go run ./cmd/main.go

# Run the server with OTel telemetry enabled
make run-otel

# Build binary (use this, not go build directly — avoids dropping a stray main binary)
make build

# Run all tests
make test
# or
go test -v ./...

# Run tests for a specific package
go test -v ./internal/lobby/...
go test -v ./internal/game/...

# Run a single test
go test -v ./internal/lobby/ -run TestName

# Run tests with race detector (required for concurrency changes)
go test -race ./...

# Format all files with goimports
make fmt

# Run all linters (golangci-lint)
make lint

# Run fmt + lint together
make check
```

## Observability Stack

Start before running the backend (required for telemetry):

```bash
cd observability && docker compose up -d
```

- Grafana: http://localhost:3003
- Backend sends OTLP traces and logs to `localhost:4317` (OTel Collector via gRPC)

## Key Patterns

**Logging**: Use `slog` with context (`slog.InfoContext`, `slog.ErrorContext`) so trace/span IDs are automatically propagated to OTel.

**Tracing**: Each WebSocket session gets a span via `tracer.Start(r.Context(), ...)`. The tracer is package-level: `var tracer = otel.Tracer("nightfall/lobby")`.

**Wide events**: At the end of each WebSocket session, a `SessionEvent` is emitted via `Emit(ctx, message)` — a single structured log capturing the full session lifecycle (duration, outcome, player/room context). Add fields to `SessionEvent` in `wide_event.go` rather than scattering individual logs. The struct has no json tags — `Emit()` builds a flat slog `args` slice manually; add new fields there, not as struct tags.

**Hub concurrency**: `Hub` and `Room` use `sync.RWMutex`. Hub is injected via the `HubStore` interface — there is no global singleton. Create with `lobby.NewHub()` and pass to `lobby.NewServer(hub, logger)`. Always acquire the lock before reading/writing room state.

**Room handler locks**: Message handlers that only read room state use `room.mutex.RLock()`. Handlers that write room state (e.g. `handleStart` sets `gameStarted`) use `room.mutex.Lock()`. Don't downgrade a write lock to a read lock when modifying handlers.

**golangci-lint config** (`backend/.golangci.yml`): Uses v2 format — formatters (`goimports`) go under `formatters.enable`, not `linters.enable`; exclusions go under `linters.exclusions.rules`, not `issues.exclude-rules`; `gosimple` no longer exists (merged into `staticcheck`). Path patterns in YAML must use single quotes (e.g. `'_test\.go'`).

**Shutdown timeout**: `telemetry.ShutdownTimeout` (5s) is defined in `internal/telemetry/telemetry.go` — use it rather than a raw duration when calling the shutdown func from `cmd/main.go`.

## Testing

**E2E test logger**: Use `slog.New(slog.NewJSONHandler(io.Discard, nil))` for the logger in httptest-based tests. A `testWriter` that calls `t.Log()` races with server goroutines that may still be logging after the test function returns.

**Game role randomness**: `game.Start()` shuffles players via `rand.Shuffle` before assigning roles, so role-to-player mapping is non-deterministic. Test *invariants* (all players assigned, correct role set present) rather than exact mappings.

**OTel span injection (no SDK required)**: To test span context propagation (e.g. in `Emit`), inject a `trace.SpanContext` via `trace.ContextWithSpanContext(context.Background(), sc)`. See `wide_event_test.go` for the pattern.

**Hub in tests**: `Hub` is no longer a global singleton — unit tests create a `fakeHub` (implements `HubStore`) and E2E tests call `NewHub()` directly. No shared state between tests.

**`SessionEvent` testability**: `SessionEvent.buildArgs()` is an unexported pure helper — test it directly within `package telemetry` to assert args construction without `slog` side effects. Trace/span ID extraction from context happens in `Emit()`, not in `buildArgs()`.

**Testing `Emit`**: `Emit` calls `slog.InfoContext` (don't try to capture log output). Instead, assert the struct mutation: `Emit` writes `e.TraceID`/`e.SpanID` from the span in context. See `TestEmit_ExtractsSpanFromContext` in `wide_event_test.go`.

## Git

**Git hook**: Checks (fmt, lint, test) run on **push** via `.githooks/pre-push`, not pre-commit. Atomic commits are safe to make without running the full suite locally first.

**Git root**: Repo root is `nightfall/`, not `backend/`. File paths in git commands must be prefixed with `backend/` (e.g. `git add backend/internal/...`).

**Partial staging**: `printf 'y\nn\n' | git add -p <file>` works for hunk-level staging — useful when a file has independent concerns to split into separate commits.

