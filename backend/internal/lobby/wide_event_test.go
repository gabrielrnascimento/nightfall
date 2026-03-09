package lobby

import (
	"encoding/hex"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

// argsMap converts a flat key-value args slice to a map for easy assertions.
func argsMap(args []any) map[string]any {
	m := make(map[string]any, len(args)/2)
	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		m[key] = args[i+1]
	}
	return m
}

func baseEvent() *SessionEvent {
	return &SessionEvent{
		Service:    "nightfall",
		Event:      "session_end",
		RemoteAddr: "127.0.0.1:9999",
		DurationMs: 42,
		Outcome:    OutcomeSuccess,
	}
}

func TestBuildArgs_BaseFieldsAlwaysPresent(t *testing.T) {
	e := baseEvent()
	m := argsMap(e.buildArgs())

	for _, key := range []string{"service", "event", "remote_addr", "duration_ms", "outcome", "trace_id", "span_id"} {
		if _, ok := m[key]; !ok {
			t.Errorf("base field %q missing from args", key)
		}
	}
}

func TestBuildArgs_NoSpanInContext(t *testing.T) {
	e := baseEvent()
	m := argsMap(e.buildArgs())

	if got := m["trace_id"]; got != "" {
		t.Errorf("trace_id: want empty string, got %q", got)
	}
	if got := m["span_id"]; got != "" {
		t.Errorf("span_id: want empty string, got %q", got)
	}
}

func TestBuildArgs_ValidSpanInContext(t *testing.T) {
	traceIDBytes, _ := hex.DecodeString("0af7651916cd43dd8448eb211c80319c")
	spanIDBytes, _ := hex.DecodeString("b7ad6b7169203331")

	var traceID trace.TraceID
	var spanID trace.SpanID
	copy(traceID[:], traceIDBytes)
	copy(spanID[:], spanIDBytes)

	e := baseEvent()
	e.TraceID = traceID.String()
	e.SpanID = spanID.String()
	m := argsMap(e.buildArgs())
	if got := m["trace_id"]; got != traceID.String() {
		t.Errorf("trace_id: want %q, got %q", traceID.String(), got)
	}
	if got := m["span_id"]; got != spanID.String() {
		t.Errorf("span_id: want %q, got %q", spanID.String(), got)
	}
	if got := m["event"]; got != "session_end" {
		t.Errorf("event: want %q, got %q", "session_end", got)
	}
	if got := m["service"]; got != "nightfall" {
		t.Errorf("service: want %q, got %q", "nightfall", got)
	}
	if got := m["remote_addr"]; got != "127.0.0.1:9999" {
		t.Errorf("remote_addr: want %q, got %q", "127.0.0.1:9999", got)
	}
	if got := m["duration_ms"]; got != int64(42) {
		t.Errorf("duration_ms: want 42, got %v", got)
	}
	if got := m["outcome"]; got != OutcomeSuccess {
		t.Errorf("outcome: want %q, got %q", OutcomeSuccess, got)
	}
}

func TestBuildArgs_ErrorAbsentWhenEmpty(t *testing.T) {
	e := baseEvent()
	m := argsMap(e.buildArgs())
	if _, ok := m["error"]; ok {
		t.Error("error key should be absent when Error is empty")
	}
}

func TestBuildArgs_ErrorPresentWhenSet(t *testing.T) {
	e := baseEvent()
	e.Error = "something went wrong"
	m := argsMap(e.buildArgs())
	if got := m["error"]; got != "something went wrong" {
		t.Errorf("error: want %q, got %q", "something went wrong", got)
	}
}

func TestBuildArgs_PlayerNil(t *testing.T) {
	e := baseEvent()
	m := argsMap(e.buildArgs())
	if _, ok := m["player.id"]; ok {
		t.Error("player.id should be absent when Player is nil")
	}
}

func TestBuildArgs_PlayerSet(t *testing.T) {
	e := baseEvent()
	e.Player = &PlayerContext{ID: "player-42"}
	m := argsMap(e.buildArgs())
	if got := m["player.id"]; got != "player-42" {
		t.Errorf("player.id: want %q, got %q", "player-42", got)
	}
}

func TestBuildArgs_RoomNil(t *testing.T) {
	e := baseEvent()
	m := argsMap(e.buildArgs())
	if _, ok := m["room.id"]; ok {
		t.Error("room.id should be absent when Room is nil")
	}
	if _, ok := m["room.player_count"]; ok {
		t.Error("room.player_count should be absent when Room is nil")
	}
}

func TestBuildArgs_RoomSet(t *testing.T) {
	e := baseEvent()
	e.Room = &RoomContext{ID: "room-7", PlayerCount: 3}
	m := argsMap(e.buildArgs())
	if got := m["room.id"]; got != "room-7" {
		t.Errorf("room.id: want %q, got %q", "room-7", got)
	}
	if got := m["room.player_count"]; got != 3 {
		t.Errorf("room.player_count: want 3, got %v", got)
	}
}

func TestBuildArgs_StatsNil(t *testing.T) {
	e := baseEvent()
	m := argsMap(e.buildArgs())
	if _, ok := m["messages_received"]; ok {
		t.Error("messages_received should be absent when Stats is nil")
	}
	if _, ok := m["messages_sent"]; ok {
		t.Error("messages_sent should be absent when Stats is nil")
	}
}

func TestBuildArgs_StatsSet(t *testing.T) {
	e := baseEvent()
	e.Stats = &SessionStats{MessagesReceived: 10, MessagesSent: 5}
	m := argsMap(e.buildArgs())
	if got := m["messages_received"]; got != int64(10) {
		t.Errorf("messages_received: want 10, got %v", got)
	}
	if got := m["messages_sent"]; got != int64(5) {
		t.Errorf("messages_sent: want 5, got %v", got)
	}
}
