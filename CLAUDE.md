# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Nightfall is a multiplayer game lobby system with a Go WebSocket backend and a plain TypeScript frontend (no framework). The backend exposes a WebSocket endpoint; the frontend is a dev/test client UI.

## Repository Structure

```
nightfall/
├── backend/          # Go WebSocket server
├── frontend/         # TypeScript browser client (dev UI)
├── docs/             # Protocol and learning docs
│   ├── WEBSOCKET_PROTOCOL.md
│   └── OTEL_LEARNING.md
```

## Architecture

### Backend (`backend/`)

- **Entry point**: `cmd/main.go` — sets up telemetry, starts HTTP server on `ws://127.0.0.1:3001`
- **`internal/lobby/`** — core WebSocket logic:
  - `server.go` — `Server` implements `http.Handler`; accepts WebSocket connections, spawns read/write goroutines per client
  - `hub.go` — global `Hub` (singleton) holds named `Room`s; each `Room` holds a set of `*Client`s with RW mutex
  - `client.go` — per-connection state; `readPump` and `writePump` goroutines
  - `types.go` — message type definitions
- **`internal/telemetry/`** — OpenTelemetry setup: traces via OTLP/gRPC to collector at `localhost:4317`, logs via `otelslog` bridge; uses a `multiHandler` that fans out to both OTel and stdout JSON. Also owns `wide_event.go` — `SessionEvent` struct emitted as a structured slog log at session end (wide-event pattern).
- **`observability/`** — Docker Compose stack: OTel Collector → Tempo (traces) + Loki (logs) → Grafana at `http://localhost:3003`

### Frontend (`frontend/`)

- Single `app.ts` compiled to `dist/` — vanilla TypeScript browser client
- No framework; talks directly to the backend WebSocket

### WebSocket Protocol

All messages are JSON with a `type` field. See `docs/WEBSOCKET_PROTOCOL.md` for the full spec.

**Client → Server**: `join`, `leave`, `start`, `ready`
**Server → Client**: `joined`, `left`, `user_joined`, `user_left`, `game_started`, `user_ready`
