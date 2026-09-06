package integration

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	. "goark.dev/log/internal/testsupport"
)

func TestJSONTemplateLayout_whenLayoutOptionsUsed_shouldApplyResolverDefaults(t *testing.T) {
	layout, err := NewJSONTemplateLayout(`{
  "level": {"$resolver": "level", "field": "severity"},
  "logger": {"$resolver": "logger", "precision": 2},
  "mdc": {"$resolver": "mdc"},
  "thrown": {"$resolver": "throwable"}
}`, WithJSONTemplateLayoutOptions(LayoutOptions{
		Compact:              true,
		PropertiesAsList:     true,
		StacktraceAsString:   true,
		IncludeNullDelimiter: true,
	}))
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}
	event := TestEvent("options", FixedTestTime())
	event.Logger = "goark.orm.mapper"
	event.Attrs = []slog.Attr{slog.String("trace_id", "trace-1")}
	event.Throwable = &Throwable{
		Type:    "errors.errorString",
		Message: "query failed",
		Stack:   []string{"goark.orm.query(query.go:10)"},
	}

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	output := buf.Bytes()
	if !bytes.HasSuffix(output, []byte{0}) {
		t.Fatalf("JSON template output = %q, want NUL delimiter", string(output))
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(bytes.TrimSuffix(output, []byte{0}), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v, json=%s", err, string(output))
	}
	if decoded["level"] != float64(6) {
		t.Fatalf("level = %#v, want syslog severity 6", decoded["level"])
	}
	if decoded["logger"] != "orm.mapper" {
		t.Fatalf("logger = %#v, want precision-trimmed name", decoded["logger"])
	}
	contextMap, ok := decoded["mdc"].([]any)
	if !ok || len(contextMap) != 1 {
		t.Fatalf("mdc = %#v, want list form", decoded["mdc"])
	}
	thrown, ok := decoded["thrown"].(string)
	if !ok || !strings.Contains(thrown, "query.go:10") {
		t.Fatalf("thrown = %#v, want stack string", decoded["thrown"])
	}
}

func TestJSONTemplateLayout_whenThrowableResolversUsed_shouldWriteDetails(t *testing.T) {
	root := errors.New("root")
	err := fmt.Errorf("wrap: %w", root)
	event := TestEvent("failed", FixedTestTime())
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
	if err := sonic.Unmarshal(buf.Bytes(), &decoded); err != nil {
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

func TestJSONTemplateLayout_whenDefaultTemplateUsed_shouldRenderLog4jStyleFields(t *testing.T) {
	layout, err := NewJSONTemplateLayout("")
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}
	event := TestEvent("template event", FixedTestTime())
	event.ThreadName = "worker-1"
	event.ContextStack = []string{"request", "sql"}
	event.EndOfBatch = true
	event.Attrs = append(event.Attrs, slog.String("trace_id", "trace-1"))

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(buf.Bytes(), &decoded); err != nil {
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

func TestJSONTemplateLayout_whenCustomResolverRegistered_shouldUseRegistry(t *testing.T) {
	registry := NewPluginRegistry()
	factory := func(
		config JSONTemplateResolverBuildConfig,
	) (JSONTemplateResolver, error) {
		var value string
		if err := sonic.Unmarshal(config.Options["value"], &value); err != nil {
			return nil, err
		}
		return ConstantJSONResolver(value), nil
	}
	if err := registry.RegisterJSONTemplateResolver("constant", factory); err != nil {
		t.Fatalf("RegisterJSONTemplateResolver() error = %v", err)
	}
	layout, err := NewJSONTemplateLayout(
		`{"custom":{"$resolver":"constant","value":"ok"}}`,
		WithJSONTemplateResolverRegistry(registry),
	)
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}

	var buf bytes.Buffer
	if err := layout.Format(&buf, TestEvent("custom", FixedTestTime())); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if strings.TrimSpace(buf.String()) != `{"custom":"ok"}` {
		t.Fatalf("output = %s, want custom resolver output", buf.String())
	}
}

func TestJSONTemplateLayout_whenMDCFlattenEnabled_shouldFlattenGroups(t *testing.T) {
	layout, err := NewJSONTemplateLayout(`{"mdc":{"$resolver":"mdc","flatten":true}}`)
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}
	event := TestEvent("flatten", FixedTestTime())
	event.Attrs = append(event.Attrs, slog.Group("request", slog.String("id", "req-1")))

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]map[string]any
	if err := sonic.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v, json=%s", err, buf.String())
	}
	if decoded["mdc"]["request.id"] != "req-1" {
		t.Fatalf("mdc = %#v, want flattened request.id", decoded["mdc"])
	}
}

func TestJSONTemplateLayout_whenAttrResolverHasNoKey_shouldReject(t *testing.T) {
	_, err := NewJSONTemplateLayout(`{"missing": {"$resolver": "attr"}}`)
	if err == nil || !strings.Contains(err.Error(), "requires key") {
		t.Fatalf("NewJSONTemplateLayout() error = %v, want attr key rejection", err)
	}
}
