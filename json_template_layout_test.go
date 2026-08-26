package goarklog

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestJSONTemplateLayout_whenDefaultTemplateUsed_shouldRenderLog4jStyleFields(t *testing.T) {
	layout, err := NewJSONTemplateLayout("")
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}
	event := testEvent("template event", fixedTestTime())
	event.ThreadName = "worker-1"
	event.ContextStack = []string{"request", "sql"}
	event.EndOfBatch = true
	event.Attrs = append(event.Attrs, slog.String("trace_id", "trace-1"))

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v: %s", err, buf.String())
	}
	if decoded["level"] != "INFO" ||
		decoded["loggerName"] != "goark.test" ||
		decoded["message"] != "template event" ||
		decoded["thread"] != "worker-1" ||
		decoded["endOfBatch"] != true {
		t.Fatalf("default template decoded output is wrong: %#v", decoded)
	}
	contextMap, ok := decoded["contextMap"].(map[string]any)
	if !ok || contextMap["trace_id"] != "trace-1" {
		t.Fatalf("contextMap = %#v, want trace_id", decoded["contextMap"])
	}
}

func TestJSONTemplateLayout_whenCustomTemplateUsed_shouldResolveFieldsInTemplateOrder(t *testing.T) {
	layout, err := NewJSONTemplateLayout(`{
  "ts": {"$resolver": "timestamp", "format": "UNIX_MILLIS"},
  "lvl": {"$resolver": "level"},
  "trace": {"$resolver": "attr", "key": "trace_id"},
  "msg": {"$resolver": "message"},
  "static": "goark"
}`)
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}
	event := testEvent("custom template", fixedTestTime())
	event.Attrs = append(event.Attrs, slog.String("trace_id", "trace-42"))

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	line := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(line, `{"ts":`) ||
		!strings.Contains(line, `"lvl":"INFO"`) ||
		!strings.Contains(line, `"trace":"trace-42"`) ||
		!strings.Contains(line, `"msg":"custom template"`) ||
		!strings.HasSuffix(line, `"static":"goark"}`) {
		t.Fatalf("custom template output is wrong: %s", line)
	}
}

func TestJSONTemplateLayout_whenAttrResolverHasNoKey_shouldReject(t *testing.T) {
	_, err := NewJSONTemplateLayout(`{"missing": {"$resolver": "attr"}}`)
	if err == nil || !strings.Contains(err.Error(), "requires key") {
		t.Fatalf("NewJSONTemplateLayout() error = %v, want attr key rejection", err)
	}
}
