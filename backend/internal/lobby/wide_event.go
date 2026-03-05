package lobby

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type SessionEvent struct {
	Timestamp  string `json:"timestamp"`
	TraceID    string `json:"trace_id"`
	SpanID     string `json:"span_id"`
	Service    string `json:"service"`
	Event      string `json:"event"`
	RemoteAddr string `json:"remote_addr"`
	DurationMs int64  `json:"duration_ms"`
	Outcome    string `json:"outcome"`
	Error      string `json:"error,omitempty"`

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
	MessagesSent     int64 `json:"messages_sent,omitempty"`
	MessagesReceived int64 `json:"messages_received,omitempty"`
}

func (e *SessionEvent) Emit(ctx context.Context) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		e.TraceID = span.SpanContext().TraceID().String()
		e.SpanID = span.SpanContext().SpanID().String()
	}
	e.Timestamp = time.Now().Format(time.RFC3339Nano)

	data, _ := json.Marshal(e)
	slog.InfoContext(ctx, string(data))
}
