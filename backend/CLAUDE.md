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

**Wide events**: At the end of each WebSocket session, a `SessionEvent` is emitted via `Emit(ctx, message)` — a single structured log capturing the full session lifecycle (duration, outcome, player/room context). Add fields to `SessionEvent` in `wide_event.go` rather than scattering individual logs.

**Hub concurrency**: `Hub` and `Room` use `sync.RWMutex`. The global `hub` singleton is in `hub.go`. Always acquire the lock before reading/writing room state.
