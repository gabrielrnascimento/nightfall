# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run the server
make run
# or
go run ./cmd/main.go

# Build binary
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

**Hub concurrency**: `Hub` and `Room` use `sync.RWMutex`. The global `hub` singleton is in `hub.go`. Always acquire the lock before reading/writing room state.

## Testing

**Game role randomness**: `game.Start()` shuffles players via `rand.Shuffle` before assigning roles, so role-to-player mapping is non-deterministic. Test *invariants* (all players assigned, correct role set present) rather than exact mappings.

**OTel span injection (no SDK required)**: To test span ID extraction in unit tests, inject a `trace.SpanContext` directly:
```go
sc := trace.NewSpanContext(trace.SpanContextConfig{
    TraceID:    traceID,
    SpanID:     spanID,
    TraceFlags: trace.FlagsSampled,
})
ctx := trace.ContextWithSpanContext(context.Background(), sc)
```

**Global `hub` singleton**: The `hub` in `hub.go` is package-level and shared across all parallel tests. Use a unique room name per test subtest to avoid cross-test contamination.

**`SessionEvent` testability**: `SessionEvent.buildArgs()` is an unexported pure helper — test it directly within `package lobby` to assert args construction without `slog` side effects. Trace/span ID extraction from context happens in `Emit()`, not in `buildArgs()`.

## Git

**Git root**: Repo root is `nightfall/`, not `backend/`. File paths in git commands must be prefixed with `backend/` (e.g. `git add backend/internal/...`).

**Partial staging**: `printf 'y\nn\n' | git add -p <file>` works for hunk-level staging — useful when a file has independent concerns to split into separate commits.

