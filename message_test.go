package goarklog

import (
	"log/slog"
	"strings"
	"testing"
)

func TestParameterizedMessage_whenPlaceholdersPresent_shouldReplaceInOrder(t *testing.T) {
	message := NewParameterizedMessage("user {} login from {}", "alice", "127.0.0.1")

	if got := message.String(); got != "user alice login from 127.0.0.1" {
		t.Fatalf("message = %q, want formatted text", got)
	}
}

func TestParameterizedMessage_whenPlaceholderEscaped_shouldKeepLiteral(t *testing.T) {
	message := NewParameterizedMessage(`literal \{} and {}`, "value")

	if got := message.String(); got != "literal {} and value" {
		t.Fatalf("message = %q, want escaped placeholder", got)
	}
}

func TestMapMessage_whenAttrsProvided_shouldExposeMessageAndAttrs(t *testing.T) {
	message := NewMapMessage(slog.String("user", "alice"), slog.Int("status", 200))

	if got := message.String(); !strings.Contains(got, "user=alice") || !strings.Contains(got, "status=200") {
		t.Fatalf("message = %q, want key value pairs", got)
	}
	attrs := message.Attrs()
	if len(attrs) != 2 || attrs[0].Key != "user" || attrs[1].Key != "status" {
		t.Fatalf("attrs = %+v, want message attrs", attrs)
	}
}

func TestStructuredDataMessage_whenCreated_shouldExposeStructuredAttrs(t *testing.T) {
	message := NewStructuredDataMessage("audit@32473", "login", "accepted", slog.String("user", "alice"))

	if got := message.String(); !strings.Contains(got, "[audit@32473") || !strings.Contains(got, "accepted") {
		t.Fatalf("message = %q, want structured data text", got)
	}
	attrs := message.Attrs()
	if len(attrs) != 3 {
		t.Fatalf("attrs = %+v, want id type and custom attrs", attrs)
	}
	if attrs[0].Key != StructuredDataIDAttrKey || attrs[1].Key != StructuredDataTypeAttrKey || attrs[2].Key != "user" {
		t.Fatalf("attrs = %+v, want structured attrs", attrs)
	}
}
