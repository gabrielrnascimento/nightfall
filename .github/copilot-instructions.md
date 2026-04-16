# Copilot Instructions for Nightfall

## Build, test, and lint commands

### Backend (`backend/`)

Run from `backend/`:

```bash
make fmt            # goimports
make lint           # golangci-lint
make test           # go test -race ./...
make testv          # verbose test run
make run            # run server on ws://127.0.0.1:3001
make run-otel       # run server with OpenTelemetry enabled
```

Single test examples:

```bash
go test -race -v ./internal/lobby -run TestName
go test -race -v ./internal/telemetry -run TestEmit_ExtractsSpanFromContext
```

### Frontend (`frontend/`)

Run from `frontend/`:

```bash
pnpm install
pnpm build          # tsc app.ts -> dist/
pnpm dev            # build watch + live-server on port 3000
```

There is currently no frontend test script in `package.json`.

### Git hooks

After clone, enable repo hooks:

```bash
git config core.hooksPath .githooks
```

The `pre-push` hook runs backend `make fmt`, `make lint`, and `make test` when pushed commits touch `backend/`.

## High-level architecture

- The backend entrypoint (`backend/cmd/main.go`) creates a `lobby.Hub`, starts an HTTP server on `127.0.0.1:3001`, and serves WebSocket traffic via `lobby.NewServer(hub, slog.Default())`.
- WebSocket session handling lives in `backend/internal/lobby/`: `Server.ServeHTTP` accepts connections, creates a `Client`, and runs `readPump` + `writePump` concurrently.
- Room state is managed by `Hub`/`Room` in `hub.go`: rooms are created on demand, tracked in memory, and removed when empty.
- Message handling is split by type in `client.go` + `handlers.go` (`join`, `leave`, `start`, `ready`) and serialized through explicit message structs in `types.go`.
- Game-role assignment is in `backend/internal/game/game.go`; `handleStart` locks room state, marks the game as started, then assigns roles.
- Telemetry is optional (`ENABLE_OTEL=true`) and configured in `backend/internal/telemetry/telemetry.go`; each WebSocket session emits one structured wide event (`SessionEvent`) in `wide_event.go`.
- Frontend is a single-file TypeScript client (`frontend/app.ts`) that connects directly to the backend WebSocket, logs raw messages, and toggles UI state based on protocol message types.

## Key conventions in this codebase

- **Hub injection, not globals:** depend on `HubStore` and pass `NewHub()` into `NewServer(...)` instead of using singleton state.
- **Lock discipline matters:** use `Room`/`Hub` mutexes for all shared state access; handlers that mutate room state (`handleStart`) use write locks.
- **Session-end wide event pattern:** add session analytics fields to `SessionEvent` + `buildArgs()` and emit once at session end, instead of scattering many ad-hoc logs.
- **Use context-aware slog calls:** prefer `slog.InfoContext`/`ErrorContext` in request/session flows to preserve trace correlation.
- **Origin/port coupling is strict:** backend WebSocket accepts origins on `http://127.0.0.1:3000` and `http://localhost:3000`, so frontend dev server must run on port `3000`.
- **Protocol work should update both code and docs:** keep `backend/internal/lobby/types.go`, handlers/client dispatch, frontend `app.ts`, and `docs/WEBSOCKET_PROTOCOL.md` aligned when adding or changing message types.
