# OpenTelemetry & Wide Events: Learn by Doing in Nightfall

A hands-on guide to observability — traces, spans, and wide events — using your Go backend as the learning playground.

---

## Part 0: The Mental Model (5 min read)

Before touching code, internalize the key ideas from [loggingsucks.com](https://loggingsucks.com).

### The problem with classic logging

```
log.Printf("client connected")
log.Printf("join room: nightfall-123")
log.Printf("player ready")
log.Printf("game started")
```

17 log lines for one player session. 10,000 concurrent players = 170,000 lines/sec. When something breaks, you're grepping in the dark.

### The three pillars of modern observability

| Signal      | What it is                            | Example in Nightfall                       |
| ----------- | ------------------------------------- | ------------------------------------------ |
| **Logs**    | Discrete events, usually text         | `"client disconnected with error"`         |
| **Metrics** | Aggregated numbers over time          | `active_connections = 47`                  |
| **Traces**  | A request's journey across components | WebSocket connect → join room → game start |

### Key vocabulary

- **Trace**: The full journey of one logical operation (e.g., a player connecting and joining a game). Has a unique `trace_id`.
- **Span**: One unit of work within a trace (e.g., "accept WebSocket", "validate room", "assign role"). Spans are nested — a trace is a tree of spans.
- **Wide Event / Canonical Log Line**: Instead of many small log lines, you emit **one** rich JSON object per operation containing every relevant field. Popularized by Stripe.
- **Cardinality**: Number of unique values a field can have. `player_id` is high-cardinality (many unique values = very useful for debugging). `status` is low-cardinality.
- **OpenTelemetry (OTel)**: A vendor-neutral standard + SDK for emitting traces, metrics, and logs. It's the *delivery mechanism* — not the observability solution itself.

> [!IMPORTANT]
> OTel doesn't decide *what* to instrument. You still have to think about what context matters. Adding OTel without intention just gives you bad telemetry in a standardized format.

### What a wide event looks like for Nightfall

Instead of 5 separate log lines for a player session, you emit **one** event at the end:

```json
{
  "timestamp": "2026-03-04T19:14:16Z",
  "trace_id": "abc123def456",
  "service": "nightfall-backend",
  "version": "0.1.0",
  "event": "websocket_session",
  "remote_addr": "127.0.0.1:54321",
  "duration_ms": 4523,
  "outcome": "success",
  "player": {
    "id": "player_xyz",
    "room_id": "nightfall-abc",
    "role": "werewolf",
    "ready": true
  },
  "game": {
    "started": true,
    "player_count": 6,
    "phase": "night"
  },
  "messages_sent": 12,
  "messages_received": 8
}
```

One line. Every answer you'd ever need.

---

## Part 1: Set Up the Local Observability Stack

You need somewhere to *send* and *visualize* your telemetry. Here's the local stack:

```
Your Go App
    │
    ▼ (OTLP gRPC :4317)
OpenTelemetry Collector   ← receives all signals
    │          │
    ▼          ▼
  Tempo      Loki        ← stores traces / logs
    │          │
    └────┬─────┘
         ▼
       Grafana           ← you explore everything here
```

### Step 1.1 — Create the Docker Compose stack

Create `backend/observability/docker-compose.yml`:

```yaml
services:
  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.104.0
    command: ["--config=/etc/otel-config.yaml"]
    volumes:
      - ./otel-config.yaml:/etc/otel-config.yaml
    ports:
      - "4317:4317"   # OTLP gRPC (your app sends here)
      - "4318:4318"   # OTLP HTTP (alternative)
      - "8888:8888"   # Collector's own metrics
    depends_on:
      - tempo
      - loki

  tempo:
    image: grafana/tempo:2.7.0
    command: ["-config.file=/etc/tempo.yaml"]
    volumes:
      - ./tempo.yaml:/etc/tempo.yaml
      - tempo-data:/var/tempo
    ports:
      - "3200:3200"   # Tempo HTTP API
      - "4319:4317"   # Tempo's own OTLP gRPC (collector writes here)

  loki:
    image: grafana/loki:3.3.0
    command: ["-config.file=/etc/loki/loki.yaml"]
    volumes:
      - ./loki.yaml:/etc/loki/loki.yaml
      - loki-data:/loki
    ports:
      - "3100:3100"

  grafana:
    image: grafana/grafana:11.4.0
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
    volumes:
      - ./grafana-datasources.yaml:/etc/grafana/provisioning/datasources/datasources.yaml
    ports:
      - "3003:3000"   # Open http://localhost:3003
    depends_on:
      - tempo
      - loki

volumes:
  tempo-data:
  loki-data:
```

Create `backend/observability/otel-config.yaml`:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:

exporters:
  otlp/tempo:
    endpoint: tempo:4317
    tls:
      insecure: true
  loki:
    endpoint: http://loki:3100/loki/api/v1/push
    default_labels_enabled:
      exporter: false
      job: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/tempo]
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [loki]
```

Create `backend/observability/tempo.yaml`:

```yaml
server:
  http_listen_port: 3200

distributor:
  receivers:
    otlp:
      protocols:
        grpc:
          endpoint: 0.0.0.0:4317

storage:
  trace:
    backend: local
    local:
      path: /var/tempo/blocks
    wal:
      path: /var/tempo/wal
```

Create `backend/observability/loki.yaml`:

```yaml
auth_enabled: false

server:
  http_listen_port: 3100

common:
  instance_addr: 127.0.0.1
  path_prefix: /loki
  storage:
    filesystem:
      chunks_directory: /loki/chunks
      rules_directory: /loki/rules
  replication_factor: 1
  ring:
    kvstore:
      store: inmemory

schema_config:
  configs:
    - from: 2020-10-24
      store: tsdb
      object_store: filesystem
      schema: v13
      index:
        prefix: index_
        period: 24h
```

Create `backend/observability/grafana-datasources.yaml`:

```yaml
apiVersion: 1
datasources:
  - name: Tempo
    type: tempo
    url: http://tempo:3200
    isDefault: true
    jsonData:
      tracesToLogsV2:
        datasourceUid: loki
      serviceMap:
        datasourceUid: prometheus
  - name: Loki
    type: loki
    uid: loki
    url: http://loki:3100
```

### Step 1.2 — Start the stack

```bash
cd backend/observability
docker compose up -d
```

Open **http://localhost:3003** — you should see Grafana. Click **Explore** → select the **Tempo** datasource. It will be empty for now. That's OK.

> [!TIP]
> To stop the stack later: `docker compose down`. Your data is stored in named Docker volumes, so it persists across restarts.

---

## Part 2: Add OpenTelemetry to the Go Backend

### Step 2.1 — Install OTel SDK packages

```bash
cd backend
go get go.opentelemetry.io/otel \
       go.opentelemetry.io/otel/sdk/trace \
       go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc \
       go.opentelemetry.io/otel/sdk/log \
       go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc \
       go.opentelemetry.io/otel/log/global \
       google.golang.org/grpc
```

### Step 2.2 — Create the OTel bootstrap

Create `backend/internal/telemetry/telemetry.go`. This file sets up the OTel SDK: connects to the collector, configures the tracer and logger providers, and returns a shutdown function.

```go
package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Setup initializes OTel and returns a shutdown function.
// Call shutdown() before the process exits.
func Setup(ctx context.Context, serviceName, serviceVersion string) (shutdown func(context.Context) error, err error) {
	// Connect to the OTel Collector
	conn, err := grpc.NewClient(
		"localhost:4317",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to collector: %w", err)
	}

	// Describe this service (shows up in Grafana)
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// --- Trace provider ---
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		// Sample 100% of traces locally (use a ratio sampler in production)
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)

	// --- Log provider (for wide events / canonical log lines) ---
	logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}
	lp := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
		log.WithResource(res),
	)
	global.SetLoggerProvider(lp)

	shutdown = func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(ctx)
		_ = lp.Shutdown(ctx)
		return conn.Close()
	}
	return shutdown, nil
}
```

### Step 2.3 — Initialize OTel in `main.go`

Update `cmd/main.go` to call `telemetry.Setup` before starting the server:

```go
package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gabrielrnascimento/nightfall/backend/internal/lobby"
	"github.com/gabrielrnascimento/nightfall/backend/internal/telemetry"
)

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	// Bootstrap OpenTelemetry
	shutdown, err := telemetry.Setup(ctx, "nightfall-backend", "0.1.0")
	if err != nil {
		return err
	}
	defer func() {
		// Flush and close exporters before exit
		if err := shutdown(ctx); err != nil {
			log.Printf("failed to shutdown telemetry: %v", err)
		}
	}()

	l, err := net.Listen("tcp", "127.0.0.1:3001")
	if err != nil {
		return err
	}
	log.Printf("listening on ws://%v", l.Addr())

	s := &http.Server{
		Handler: lobby.Server{
			Logf: log.Printf,
		},
	}
	errC := make(chan error, 1)
	go func() { errC <- s.Serve(l) }()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errC:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.Shutdown(shutCtx)
}
```

---

## Part 3: Add Your First Span (Traces)

### What is a span?

Think of a span as a timed box around a unit of work. Spans know about their parent span, forming a tree. The whole tree is the trace.

```
[WebSocket session]          ← root span (trace)
  ├── [accept connection]    ← child span
  ├── [join room]            ← child span
  │     └── [validate room]  ← grandchild span
  └── [start game]           ← child span
```

### Step 3.1 — Instrument `ServeHTTP` with a trace

Update `internal/lobby/server.go`:

```go
package lobby

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"github.com/coder/websocket"
)

// tracer is the package-level tracer for the lobby package.
// "nightfall/lobby" is the instrumentation scope name — helps you filter in Grafana.
var tracer = otel.Tracer("nightfall/lobby")

type Server struct {
	Logf func(f string, v ...any)
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Start a root span for this entire WebSocket session
	ctx, span := tracer.Start(r.Context(), "websocket.session")
	defer span.End() // End the span when ServeHTTP returns (session ends)

	// Tag the span with initial context. These show up as attributes in Grafana.
	span.SetAttributes(
		attribute.String("net.peer.addr", r.RemoteAddr),
	)

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{
			"http://127.0.0.1:3000",
			"http://localhost:3000",
		},
	})
	if err != nil {
		// Mark the span as failed
		span.RecordError(err)
		span.SetStatus(codes.Error, "websocket accept failed")
		s.Logf("%v", err)
		return
	}
	defer c.CloseNow()

	s.Logf("client connected from %v", r.RemoteAddr)

	client := &Client{
		conn: c,
		send: make(chan []byte, 256),
	}

	ctx, cancel := context.WithCancel(ctx) // use the span's context, not r.Context()
	defer cancel()

	errChan := make(chan error, 1)
	go func() { errChan <- client.readPump(ctx) }()
	go func() { client.writePump(ctx) }()

	err = <-errChan

	if err == nil || websocket.CloseStatus(err) == websocket.StatusNormalClosure {
		span.SetStatus(codes.Ok, "")
		s.Logf("client disconnected normally from %v", r.RemoteAddr)
	} else {
		span.RecordError(err)
		span.SetStatus(codes.Error, "client disconnected with error")
		s.Logf("client disconnected with error from %v %v", r.RemoteAddr, err)
	}
}
```

### Step 3.2 — Run and see your first trace

```bash
cd backend
go run ./cmd/main.go
```

Open your app's frontend, connect a player. Then go to **Grafana → Explore → Tempo** and click **Search**. You'll see a `websocket.session` trace. Click it — you'll see the timeline of your span.

> [!TIP]
> This is the core OTel loop: **instrument → run → explore in Grafana**. Repeat for every interesting operation.

---

## Part 4: Wide Events — One Log Line to Rule Them All

Spans are great for timing and tracing. But for rich business context (player role, room state at disconnect, number of messages exchanged), you want a **wide event**.

### Step 4.1 — Create a wide event struct

Create `backend/internal/lobby/wide_event.go`:

```go
package lobby

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// SessionEvent is the canonical log line for one WebSocket session.
// Build it up throughout the session, then emit once at the end.
type SessionEvent struct {
	Timestamp  string `json:"timestamp"`
	TraceID    string `json:"trace_id"`   // links log to trace in Grafana
	SpanID     string `json:"span_id"`
	Service    string `json:"service"`
	Event      string `json:"event"`
	RemoteAddr string `json:"remote_addr"`
	DurationMs int64  `json:"duration_ms"`
	Outcome    string `json:"outcome"` // "success" | "error"
	Error      string `json:"error,omitempty"`

	// Business context — add fields here as you learn more about the domain
	Player *PlayerContext `json:"player,omitempty"`
	Room   *RoomContext   `json:"room,omitempty"`
	Stats  *SessionStats  `json:"stats,omitempty"`
}

type PlayerContext struct {
	ID   string `json:"id,omitempty"`
	Role string `json:"role,omitempty"`
}

type RoomContext struct {
	ID          string `json:"id,omitempty"`
	PlayerCount int    `json:"player_count,omitempty"`
	GameStarted bool   `json:"game_started,omitempty"`
}

type SessionStats struct {
	MessagesReceived int `json:"messages_received"`
	MessagesSent     int `json:"messages_sent"`
}

// Emit writes the event as a single structured JSON log line.
func (e *SessionEvent) Emit(ctx context.Context) {
	// Attach the trace/span IDs so Grafana can link logs ↔ traces
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		e.TraceID = span.SpanContext().TraceID().String()
		e.SpanID  = span.SpanContext().SpanID().String()
	}
	e.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)

	data, _ := json.Marshal(e)
	// slog writes structured JSON to stdout — the OTel log bridge will pick it up
	slog.InfoContext(ctx, string(data))
}
```

### Step 4.2 — Use the wide event in `ServeHTTP`

Update `ServeHTTP` to build up and emit a `SessionEvent`:

```go
func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "websocket.session")
	defer span.End()

	start := time.Now()

	// Initialize the wide event at the top of the request
	event := &SessionEvent{
		Service:    "nightfall-backend",
		Event:      "websocket.session",
		RemoteAddr: r.RemoteAddr,
	}
	// Emit the event at the very end, no matter what
	defer func() {
		event.DurationMs = time.Since(start).Milliseconds()
		event.Emit(ctx)
	}()

	span.SetAttributes(attribute.String("net.peer.addr", r.RemoteAddr))

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"http://127.0.0.1:3000", "http://localhost:3000"},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "websocket accept failed")
		event.Outcome = "error"
		event.Error   = err.Error()
		s.Logf("%v", err)
		return
	}
	defer c.CloseNow()

	client := &Client{conn: c, send: make(chan []byte, 256)}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 1)
	go func() { errChan <- client.readPump(ctx) }()
	go func() { client.writePump(ctx) }()

	err = <-errChan

	if err == nil || websocket.CloseStatus(err) == websocket.StatusNormalClosure {
		span.SetStatus(codes.Ok, "")
		event.Outcome = "success"
	} else {
		span.RecordError(err)
		span.SetStatus(codes.Error, "abnormal disconnect")
		event.Outcome = "error"
		event.Error   = err.Error()
	}
}
```

When the session ends, stdout gets one line like:

```json
{"timestamp":"2026-03-04T22:14:16Z","trace_id":"abc...","service":"nightfall-backend","event":"websocket.session","remote_addr":"127.0.0.1:54321","duration_ms":3201,"outcome":"success"}
```

---

## Part 5: Add Business Context (the real payoff)

A wide event with only network fields is not much better than regular logs. The power comes from adding **domain context** at each step of the session.

### Step 5.1 — Pass the event through the session

Pass `*SessionEvent` down into `client.go` so each handler can enrich it:

```go
// In readPump or wherever you process messages, do things like:
event.Player = &PlayerContext{
    ID:   msg.PlayerID,
    Role: assignedRole,
}
event.Room = &RoomContext{
    ID:          room.ID,
    PlayerCount: len(room.Players),
    GameStarted: room.GameStarted,
}
event.Stats = &SessionStats{
    MessagesReceived: client.msgReceived,
    MessagesSent:     client.msgSent,
}
```

### Step 5.2 — Explore the result in Grafana

In **Grafana → Explore → Loki**, run this query to find all sessions that ended with an error:

```logql
{service_name="nightfall-backend"} | json | outcome = "error"
```

Find all sessions for a specific player:

```logql
{service_name="nightfall-backend"} | json | player_id = "player_xyz"
```

Click a trace ID in a log line → Grafana jumps to the trace in Tempo. This is the log ↔ trace correlation.

---

## Part 6: Child Spans for Sub-operations

Not everything belongs in the root span. Add child spans around meaningful sub-operations:

```go
func (c *Client) handleJoinRoom(ctx context.Context, msg JoinMessage) error {
    // Create a child span — automatically connected to the parent
    ctx, span := tracer.Start(ctx, "room.join")
    defer span.End()

    span.SetAttributes(
        attribute.String("room.id", msg.RoomID),
        attribute.String("player.id", msg.PlayerID),
    )

    // ... your join logic ...

    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, "join failed")
        return err
    }

    span.SetAttributes(attribute.String("player.role", assignedRole))
    return nil
}
```

In Grafana's trace view, you'll see this as a nested bar under `websocket.session`.

---

## Part 7: Sampling Concepts (Theory Only)

> [!NOTE]
> At your current scale (college project), sample 100% of traces (`AlwaysSample()`). Read this section to understand what you'd do in production.

**Head sampling**: Decided at the *start* of a trace. Fast and cheap, but you might drop errored traces.

**Tail sampling**: Decided at the *end* of a trace, after you know the outcome. Done by the OTel Collector. Lets you keep 100% of errors and 1% of successes:

```yaml
# In otel-config.yaml (collector) — DO NOT add yet, just read
processors:
  tail_sampling:
    decision_wait: 10s
    policies:
      - name: keep-errors
        type: status_code
        status_code: { status_codes: [ERROR] }
      - name: probabilistic
        type: probabilistic
        probabilistic: { sampling_percentage: 10 }
```

---

## Summary Checklist

- [ ] **Part 1**: Docker Compose stack running (Grafana at http://localhost:3003)
- [ ] **Part 2**: OTel SDK installed, `telemetry.Setup` called in `main.go`
- [ ] **Part 3**: Root span in `ServeHTTP`, visible in Grafana Tempo
- [ ] **Part 4**: `SessionEvent` wide event emitted at session end
- [ ] **Part 5**: Player/room context added to the wide event, visible in Grafana Loki
- [ ] **Part 6**: Child span around a game action (e.g., `room.join`)
- [ ] **Stretch**: Add a child span around game message processing in `client.go`
- [ ] **Stretch**: Add `messages_received` / `messages_sent` counters to the wide event
- [ ] **Stretch**: Explore tail sampling config in the OTel Collector
