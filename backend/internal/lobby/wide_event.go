package lobby

import (
	"context"
	"log/slog"

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

func (e *SessionEvent) Emit(ctx context.Context, message string) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		e.TraceID = span.SpanContext().TraceID().String()
		e.SpanID = span.SpanContext().SpanID().String()
	}

	args := []any{
		"service", e.Service,
		"event", e.Event,
		"remote_addr", e.RemoteAddr,
		"duration_ms", e.DurationMs,
		"outcome", e.Outcome,
	}

	if e.Error != "" {
		args = append(args, "error", e.Error)
	}

	slog.InfoContext(ctx, message, args...)
}
