package integration

import (
	"context"
	"log/slog"
	"testing"

	. "goark.dev/log/internal/testsupport"
)

func TestFailoverAppender_whenPrimaryFails_shouldWriteFirstHealthyFailover(t *testing.T) {
	primary := NewFailingAppender("primary")
	failover := NewRecordingAppender("failover")
	appender, err := NewFailoverAppender(primary, []Appender{failover})
	if err != nil {
		t.Fatalf("NewFailoverAppender() error = %v", err)
	}

	if err := appender.Append(
		context.Background(),
		TestEvent("failover event", FixedTestTime()),
	); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if !failover.Contains("failover event") {
		t.Fatalf("failover events = %+v, want failover event", failover.Events())
	}
}

func TestRoutingAppender_whenRouteKeyMatches_shouldWriteMatchedRoute(t *testing.T) {
	audit := NewRecordingAppender("audit")
	defaultRoute := NewRecordingAppender("default")
	appender, err := NewRoutingAppender(
		map[string]Appender{"audit": audit},
		WithRoutingDefault(defaultRoute),
		WithRoutingAttrKey("kind"),
	)
	if err != nil {
		t.Fatalf("NewRoutingAppender() error = %v", err)
	}

	event := TestEvent("audit event", FixedTestTime())
	event.Attrs = []slog.Attr{slog.String("kind", "audit")}
	if err := appender.Append(context.Background(), event); err != nil {
		t.Fatalf("Append(audit) error = %v", err)
	}
	if err := appender.Append(
		context.Background(),
		TestEvent("default event", FixedTestTime()),
	); err != nil {
		t.Fatalf("Append(default) error = %v", err)
	}
	if !audit.Contains("audit event") || audit.Contains("default event") {
		t.Fatalf("audit events = %+v, want only audit event", audit.Events())
	}
	if !defaultRoute.Contains("default event") || defaultRoute.Contains("audit event") {
		t.Fatalf("default events = %+v, want only default event", defaultRoute.Events())
	}
}

func TestRewriteAppender_whenPolicyConfigured_shouldRewriteEventBeforeDelegate(t *testing.T) {
	delegate := NewRecordingAppender("delegate")
	appender, err := NewRewriteAppender(
		delegate,
		func(_ context.Context, event Event) (Event, error) {
			event.Message = "rewritten " + event.Message
			event.Attrs = append(event.Attrs, slog.String("rewritten", "true"))
			return event, nil
		},
	)
	if err != nil {
		t.Fatalf("NewRewriteAppender() error = %v", err)
	}

	if err := appender.Append(context.Background(), TestEvent("event", FixedTestTime())); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	events := delegate.Events()
	if len(events) != 1 || events[0].Message != "rewritten event" {
		t.Fatalf("rewritten events = %+v", events)
	}
	if value, ok := events[0].Attr("rewritten"); !ok || value.String() != "true" {
		t.Fatalf("rewritten attr missing from event: %+v", events[0])
	}
}
