package lobby

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeError   Outcome = "error"
)

type SessionEvent struct {
	TraceID    string
	SpanID     string
	Service    string
	Event      string
	RemoteAddr string
	DurationMs int64
	Outcome    Outcome
	Error      string

	Player *PlayerContext
	Room   *RoomContext
	Stats  *SessionStats
}

type PlayerContext struct {
	ID   string
	Role string
}

type RoomContext struct {
	ID          string
	PlayerCount int
	GameStarted bool
}

type SessionStats struct {
	MessagesSent     int64
	MessagesReceived int64
}

func (e *SessionEvent) buildArgs(ctx context.Context) []any {
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
		"trace_id", e.TraceID,
		"span_id", e.SpanID,
	}

	if e.Error != "" {
		args = append(args, "error", e.Error)
	}

	if e.Player != nil {
		args = append(args, "player.id", e.Player.ID)
	}

	if e.Room != nil {
		args = append(args, "room.id", e.Room.ID, "room.player_count", e.Room.PlayerCount)
	}

	if e.Stats != nil {
		args = append(args, "messages_received", e.Stats.MessagesReceived, "messages_sent", e.Stats.MessagesSent)
	}

	return args
}

func (e *SessionEvent) Emit(ctx context.Context, message string) {
	slog.InfoContext(ctx, message, e.buildArgs(ctx)...)
}
