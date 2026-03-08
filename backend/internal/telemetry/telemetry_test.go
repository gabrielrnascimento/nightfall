package telemetry

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

type fakeHandler struct {
	enabled  bool
	records  []slog.Record
	attrs    []slog.Attr
	group    string
}

func (f *fakeHandler) Enabled(_ context.Context, _ slog.Level) bool { return f.enabled }
func (f *fakeHandler) Handle(_ context.Context, r slog.Record) error {
	f.records = append(f.records, r.Clone())
	return nil
}
func (f *fakeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &fakeHandler{enabled: f.enabled, attrs: attrs}
}
func (f *fakeHandler) WithGroup(name string) slog.Handler {
	return &fakeHandler{enabled: f.enabled, group: name}
}

func TestMultiHandler_Handle_DispatchesBothEnabled(t *testing.T) {
	a := &fakeHandler{enabled: true}
	b := &fakeHandler{enabled: true}
	m := &multiHandler{handlers: []slog.Handler{a, b}}

	r := slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0)
	_ = m.Handle(context.Background(), r)

	if len(a.records) != 1 {
		t.Errorf("handler a: want 1 record, got %d", len(a.records))
	}
	if len(b.records) != 1 {
		t.Errorf("handler b: want 1 record, got %d", len(b.records))
	}
}

func TestMultiHandler_Handle_SkipsDisabledHandler(t *testing.T) {
	enabled := &fakeHandler{enabled: true}
	disabled := &fakeHandler{enabled: false}
	m := &multiHandler{handlers: []slog.Handler{enabled, disabled}}

	r := slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0)
	_ = m.Handle(context.Background(), r)

	if len(enabled.records) != 1 {
		t.Errorf("enabled handler: want 1 record, got %d", len(enabled.records))
	}
	if len(disabled.records) != 0 {
		t.Errorf("disabled handler: want 0 records, got %d", len(disabled.records))
	}
}

func TestMultiHandler_Enabled_TrueIfAnyEnabled(t *testing.T) {
	m := &multiHandler{handlers: []slog.Handler{
		&fakeHandler{enabled: false},
		&fakeHandler{enabled: true},
	}}
	if !m.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("want true when at least one handler is enabled")
	}
}

func TestMultiHandler_Enabled_FalseIfAllDisabled(t *testing.T) {
	m := &multiHandler{handlers: []slog.Handler{
		&fakeHandler{enabled: false},
		&fakeHandler{enabled: false},
	}}
	if m.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("want false when all handlers are disabled")
	}
}

func TestMultiHandler_WithAttrs_ReturnNewMultiHandler(t *testing.T) {
	a := &fakeHandler{enabled: true}
	b := &fakeHandler{enabled: true}
	m := &multiHandler{handlers: []slog.Handler{a, b}}

	attrs := []slog.Attr{slog.String("key", "val")}
	result := m.WithAttrs(attrs)

	mh, ok := result.(*multiHandler)
	if !ok {
		t.Fatal("WithAttrs must return *multiHandler")
	}
	if len(mh.handlers) != 2 {
		t.Errorf("want 2 handlers, got %d", len(mh.handlers))
	}
	for i, h := range mh.handlers {
		fh, ok := h.(*fakeHandler)
		if !ok {
			t.Fatalf("handler %d is not *fakeHandler", i)
		}
		if len(fh.attrs) == 0 {
			t.Errorf("handler %d: attrs not propagated", i)
		}
	}
}

func TestMultiHandler_WithGroup_ReturnNewMultiHandler(t *testing.T) {
	a := &fakeHandler{enabled: true}
	b := &fakeHandler{enabled: true}
	m := &multiHandler{handlers: []slog.Handler{a, b}}

	result := m.WithGroup("grp")

	mh, ok := result.(*multiHandler)
	if !ok {
		t.Fatal("WithGroup must return *multiHandler")
	}
	if len(mh.handlers) != 2 {
		t.Errorf("want 2 handlers, got %d", len(mh.handlers))
	}
	for i, h := range mh.handlers {
		fh, ok := h.(*fakeHandler)
		if !ok {
			t.Fatalf("handler %d is not *fakeHandler", i)
		}
		if fh.group != "grp" {
			t.Errorf("handler %d: want group %q, got %q", i, "grp", fh.group)
		}
	}
}
