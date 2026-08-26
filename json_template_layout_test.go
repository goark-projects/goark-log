package goarklog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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

func TestJSONTemplateLayout_whenTemplateFileUsed_shouldLoadLocalFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event-template.json")
	if err := os.WriteFile(path, []byte(`{"msg":{"$resolver":"message"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	layout, err := NewJSONTemplateLayoutFromFile(path)
	if err != nil {
		t.Fatalf("NewJSONTemplateLayoutFromFile() error = %v", err)
	}

	var buf bytes.Buffer
	if err := layout.Format(&buf, testEvent("from file", fixedTestTime())); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if strings.TrimSpace(buf.String()) != `{"msg":"from file"}` {
		t.Fatalf("output = %s, want file template output", buf.String())
	}
}

func TestBuildLayout_whenJSONTemplateURIConfigured_shouldLoadTemplateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event-template.json")
	if err := os.WriteFile(path, []byte(`{"level":{"$resolver":"level"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	layout, err := buildLayout(layoutConfig{Type: "jsonTemplate", EventTemplateURI: path}, DefaultPluginRegistry())
	if err != nil {
		t.Fatalf("buildLayout() error = %v", err)
	}

	var buf bytes.Buffer
	if err := layout.Format(&buf, testEvent("from config", fixedTestTime())); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if strings.TrimSpace(buf.String()) != `{"level":"INFO"}` {
		t.Fatalf("output = %s, want config template output", buf.String())
	}
}

func TestJSONTemplateLayout_whenRemoteTemplateURIUsed_shouldReject(t *testing.T) {
	_, err := NewJSONTemplateLayoutFromFile("https://example.invalid/template.json")
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("NewJSONTemplateLayoutFromFile() error = %v, want remote URI rejection", err)
	}
}

func TestJSONTemplateLayout_whenCustomResolverRegistered_shouldUseRegistry(t *testing.T) {
	registry := NewPluginRegistry()
	if err := registry.RegisterJSONTemplateResolver("constant", func(config JSONTemplateResolverBuildConfig) (JSONTemplateResolver, error) {
		var value string
		if err := json.Unmarshal(config.Options["value"], &value); err != nil {
			return nil, err
		}
		return constantJSONResolver(value), nil
	}); err != nil {
		t.Fatalf("RegisterJSONTemplateResolver() error = %v", err)
	}
	layout, err := NewJSONTemplateLayout(`{"custom":{"$resolver":"constant","value":"ok"}}`, WithJSONTemplateResolverRegistry(registry))
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}

	var buf bytes.Buffer
	if err := layout.Format(&buf, testEvent("custom", fixedTestTime())); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if strings.TrimSpace(buf.String()) != `{"custom":"ok"}` {
		t.Fatalf("output = %s, want custom resolver output", buf.String())
	}
}

func TestJSONTemplateLayout_whenThrowableResolversUsed_shouldWriteDetails(t *testing.T) {
	root := errors.New("root")
	err := fmt.Errorf("wrap: %w", root)
	event := testEvent("failed", fixedTestTime())
	event.Throwable = NewThrowableWithStack(err)
	layout, layoutErr := NewJSONTemplateLayout(`{
  "thrown": {"$resolver": "throwable"},
  "root": {"$resolver": "rootCause"},
  "stack": {"$resolver": "stackTrace"}
}`)
	if layoutErr != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", layoutErr)
	}

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v, json=%s", err, buf.String())
	}
	thrown, ok := decoded["thrown"].(map[string]any)
	if !ok || thrown["message"] != "wrap: root" {
		t.Fatalf("thrown = %#v, want throwable object", decoded["thrown"])
	}
	rootCause, ok := decoded["root"].(map[string]any)
	if !ok || rootCause["message"] != "root" {
		t.Fatalf("root = %#v, want root cause", decoded["root"])
	}
	stack, ok := decoded["stack"].([]any)
	if !ok || len(stack) == 0 {
		t.Fatalf("stack = %#v, want captured stack", decoded["stack"])
	}
}

func TestJSONTemplateLayout_whenMDCFlattenEnabled_shouldFlattenGroups(t *testing.T) {
	layout, err := NewJSONTemplateLayout(`{"mdc":{"$resolver":"mdc","flatten":true}}`)
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}
	event := testEvent("flatten", fixedTestTime())
	event.Attrs = append(event.Attrs, slog.Group("request", slog.String("id", "req-1")))

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v, json=%s", err, buf.String())
	}
	if decoded["mdc"]["request.id"] != "req-1" {
		t.Fatalf("mdc = %#v, want flattened request.id", decoded["mdc"])
	}
}

type constantJSONResolver string

func (r constantJSONResolver) AppendJSON(buf *bytes.Buffer, _ Event) {
	appendJSONString(buf, string(r))
}
