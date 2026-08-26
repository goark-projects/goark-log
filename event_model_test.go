package goarklog

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestEvent_whenMarkerThrowableAndContextStackUsed_shouldSnapshotLog4jFields(t *testing.T) {
	parent := NewMarker("DATABASE")
	marker := NewMarker("SQL", parent)
	err := errors.New("query failed")
	ctx := WithThreadName(
		WithContextStack(
			WithMarker(context.Background(), marker),
			"request-1",
			"query-users",
		),
		"worker-7",
	)

	record := slog.NewRecord(fixedTestTime(), slog.LevelError, "select failed", 0)
	record.AddAttrs(ThrowableAttr(err), slog.String("trace_id", "trace-1"))
	event := newEvent(ctx, "goark.orm", nil, nil, record)

	if event.Marker == nil || event.Marker.Name != "SQL" || !event.Marker.Contains("DATABASE") {
		t.Fatalf("event marker = %#v, want SQL with DATABASE parent", event.Marker)
	}
	if event.Throwable == nil || event.Throwable.Message != "query failed" {
		t.Fatalf("event throwable = %#v, want query failed", event.Throwable)
	}
	if got := strings.Join(event.ContextStack, "/"); got != "request-1/query-users" {
		t.Fatalf("context stack = %q", got)
	}
	if event.ThreadName != "worker-7" {
		t.Fatalf("thread name = %q, want worker-7", event.ThreadName)
	}
}

func TestPatternLayout_whenLog4jEventFieldsUsed_shouldRenderSnapshotFields(t *testing.T) {
	layout, err := NewPatternLayout("%marker|%ex|%thread|%ndc|%m%n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	event := testEvent("done", fixedTestTime())
	event.Marker = markerPointer(NewMarker("AUDIT"))
	event.Throwable = NewThrowable(errors.New("denied"))
	event.ThreadName = "worker-2"
	event.ContextStack = []string{"tenant-a", "request-9"}

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if got := buf.String(); got != "AUDIT|denied|worker-2|tenant-a request-9|done\n" {
		t.Fatalf("formatted line = %q", got)
	}
}
