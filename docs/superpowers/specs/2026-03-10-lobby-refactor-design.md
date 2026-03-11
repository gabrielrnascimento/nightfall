# Lobby Package Refactor — Design Spec

**Date:** 2026-03-10
**Status:** Approved
**Branch:** wip

---

## Problem Statement

`backend/internal/lobby/` is a functional MVP with four correctness and testability problems:

1. **Global Hub singleton** — all tests share state; no test isolation possible without unique room names as a workaround
2. **String-format JSON** — `fmt.Sprintf` builds JSON with no escaping; a player name containing `"` produces malformed protocol messages (e.g. `{"type":"user_joined","name":"say "hi""}`)
3. **Manual goroutine coordination** — `errChan` approach can leak `writePump` goroutine; no clean cancellation propagation between pumps
4. **No unit tests** — `server_test.go` is integration-only; handlers can't be tested without a live WebSocket connection

---

## Goals

- Eliminate the global hub singleton via dependency injection
- Fix JSON safety by using `json.Marshal` over typed structs
- Improve goroutine lifecycle with `errgroup`
- Enable handler unit testing without WebSocket infrastructure
- Introduce room cleanup when the last client leaves

---

## Non-Goals

- Changes to the WebSocket protocol (message types remain unchanged)
- Changes to telemetry / observability setup
- Game logic changes (`internal/game/`)

---

## Architecture

### HubStore Interface

The core enabler for DI and testability. The `Hub` concrete type implements this interface; tests use a `fakeHub`.

```go
type HubStore interface {
    GetOrCreateRoom(name string) *Room
    Broadcast(room *Room, msg []byte, exclude *Client)
    RemoveClient(room *Room, c *Client)
}
```

**Design decisions:**
- `GetOrCreateRoom` combines the lookup-or-create into a single atomic operation under the hub mutex, preventing races where two goroutines both see a missing room.
- `RemoveClient` is on the interface (not just `Room`) so the hub can perform cleanup (delete room from map when empty) in a coordinated way.
- `Broadcast` is on the interface to allow fake hubs in tests to record calls without needing real channels.

### Dependency Injection via NewServer

```
main.go
  └── hub := lobby.NewHub()
  └── srv := lobby.NewServer(hub)        ← injects hub
        └── Server.ServeHTTP(...)
              └── client := &Client{hub: hub, ...}
                    └── handlers(c, hub, ...)  ← hub threaded through
```

### Handler Extraction

All four message handlers move out of `client.go` into `handlers.go` with pure, testable signatures:

```go
func handleJoin(c *Client, hub HubStore, name, roomName string) ([]byte, error)
func handleLeave(c *Client, hub HubStore) ([]byte, error)
func handleStart(c *Client, hub HubStore) ([]byte, error)
func handleReady(c *Client, hub HubStore) ([]byte, error)
```

Return `([]byte, error)` — the caller sends the byte slice to `c.send`; errors propagate up and close the connection.

### Goroutine Lifecycle with errgroup

Current (problematic):
```
goroutine: readPump → errChan
goroutine: writePump → nothing (leaks on readPump exit)
```

After:
```
eg, ctx := errgroup.WithContext(ctx)
eg.Go(func() error { return readPump(ctx) })
eg.Go(func() error { writePump(ctx); return nil })
eg.Wait() → blocks until first goroutine returns, ctx cancelled for the other
```

The context cancellation ensures `writePump` exits cleanly when `readPump` returns (and vice versa).

### Room Cleanup

`Hub.RemoveClient` now deletes the room from `hub.rooms` when `len(room.clients) == 0` after removal. This prevents unbounded memory growth in long-running servers with many transient rooms.

### JSON Safety

Before:
```go
fmt.Sprintf(`{"type":"user_joined","name":"%s"}`, c.name)
// if name = `say "hi"` → produces invalid JSON
```

After:
```go
json.Marshal(UserJoinedMsg{Type: "user_joined", Name: c.name})
// name is properly escaped by the JSON encoder
```

---

## File Plan

| File | Action | What changes |
|------|--------|-------------|
| `hub.go` | Modify | `HubStore` interface; remove `var hub` global; room cleanup; `map[*Client]struct{}` |
| `server.go` | Modify | `hub HubStore` field; `NewServer(hub HubStore) *Server` |
| `client.go` | Modify | Remove handler methods; `errgroup`; `hub HubStore` field |
| `handlers.go` | **New** | `handleJoin/Leave/Start/Ready` returning `([]byte, error)` |
| `types.go` | Modify | Response structs with JSON tags |
| `hub_test.go` | **New** | Unit tests for Room/Hub operations |
| `handlers_test.go` | **New** | Table-driven unit tests with `fakeHub` |
| `server_test.go` | Modify | Trim to E2E happy-path; unique room names |
| `cmd/main.go` | Modify | `lobby.NewHub()` + `lobby.NewServer(hub)` |
| `go.mod` | Modify (if needed) | Add `golang.org/x/sync` for errgroup |

---

## Testing Strategy

### Unit Tests (`hub_test.go`)
- `TestRoom_AddClient` — client appears in room's client map
- `TestRoom_RemoveClient_Cleanup` — room is deleted from hub when last client leaves
- `TestRoom_Broadcast_Excludes` — excluded client doesn't receive; others do

### Unit Tests (`handlers_test.go`)
- `fakeHub` implements `HubStore`; records `Broadcast` calls and manages a simple in-memory room map
- Table-driven per handler covering valid inputs, invalid inputs, and edge cases

### E2E Tests (`server_test.go`)
- Keep: happy-path join/leave, start/game_started broadcast, multi-client user_joined notification
- Remove: error scenario tests (invalid JSON, unknown type) — these are now covered at the handler level

---

## Open Questions

None — design is complete and implementation-ready.
