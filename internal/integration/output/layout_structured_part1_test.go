package integration

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	. "goark.dev/log/internal/testsupport"
	"gopkg.in/yaml.v3"
)

func TestJSONLayout_whenOptionsEnabled_shouldWriteListPropertiesAndStackString(t *testing.T) {
	event := BenchmarkEvent()
	event.Attrs = []slog.Attr{slog.String("traceId", "trace-1")}
	event.Throwable = &Throwable{
		Type:    "errors.errorString",
		Message: "query failed",
		Stack:   []string{"goark.orm.query(query.go:10)"},
	}
	layout := NewJSONLayout(LayoutOptions{
		Compact:              true,
		PropertiesAsList:     true,
		StacktraceAsString:   true,
		IncludeNullDelimiter: true,
	})

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	output := buf.Bytes()
	if !bytes.HasSuffix(output, []byte{0}) {
		t.Fatalf("JSON output = %q, want NUL delimiter", string(output))
	}
	if bytes.Contains(output, []byte{'\n'}) {
		t.Fatalf("JSON output = %q, compact layout should not add newline", string(output))
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(bytes.TrimSuffix(output, []byte{0}), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v, json=%s", err, string(output))
	}
	contextMap, ok := decoded["contextMap"].([]any)
	if !ok || len(contextMap) != 1 {
		t.Fatalf("contextMap = %#v, want one property item", decoded["contextMap"])
	}
	item, ok := contextMap[0].(map[string]any)
	if !ok || item["key"] != "traceId" || item["value"] != "trace-1" {
		t.Fatalf("contextMap[0] = %#v, want traceId property", contextMap[0])
	}
	thrown, ok := decoded["thrown"].(string)
	if !ok || !strings.Contains(thrown, "query.go:10") {
		t.Fatalf("thrown = %#v, want stack string", decoded["thrown"])
	}
}

func TestFileAppender_whenCompleteJSONLayoutUsed_shouldWriteValidArray(t *testing.T) {
	path := filepath.Join(t.TempDir(), "complete.json")
	appender, err := NewFileAppender(
		path,
		WithFileLayout(NewJSONLayout(LayoutOptions{Compact: true, Complete: true})),
		WithFileBufferSize(0),
	)
	if err != nil {
		t.Fatalf("NewFileAppender() error = %v", err)
	}
	event := BenchmarkEvent()
	if err := appender.Append(context.Background(), event); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	event.Message = "second event"
	if err := appender.Append(context.Background(), event); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	var decoded []map[string]any
	content := ReadTextFile(t, path)
	if err := sonic.Unmarshal([]byte(content), &decoded); err != nil {
		t.Fatalf("complete JSON output is invalid: %v\n%s", err, content)
	}
	if len(decoded) != 2 || decoded[1]["msg"] != "second event" {
		t.Fatalf("decoded complete JSON = %#v, want two events", decoded)
	}
}

func TestFileAppender_whenCompleteJSONLayoutCreateOnDemand_shouldWriteValidArray(t *testing.T) {
	path := filepath.Join(t.TempDir(), "complete-lazy.json")
	appender, err := NewFileAppender(
		path,
		WithFileLayout(NewJSONLayout(LayoutOptions{Compact: true, Complete: true})),
		WithFileCreateOnDemand(true),
		WithFileBufferSize(0),
	)
	if err != nil {
		t.Fatalf("NewFileAppender() error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file should not exist before first append, stat error = %v", err)
	}
	if err := appender.Append(
		context.Background(),
		TestEvent("lazy-first", FixedTestTime()),
	); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if err := appender.Append(
		context.Background(),
		TestEvent("lazy-second", FixedTestTime()),
	); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	AssertCompleteJSONMessages(t, path, "lazy-first", "lazy-second")
}

func TestYAMLLayout_whenEventFormatted_shouldWriteYAMLDocument(t *testing.T) {
	event := BenchmarkEvent()
	event.Attrs = append(event.Attrs, slog.String("traceId", "trace-1"))

	var buf bytes.Buffer
	if err := (YAMLLayout{}).Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := yaml.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("YAML output invalid: %v\n%s", err, buf.String())
	}
	if decoded["message"] != "service started" {
		t.Fatalf("message = %#v, want service started", decoded["message"])
	}
	contextMap, ok := decoded["contextMap"].(map[string]any)
	if !ok || contextMap["traceId"] != "trace-1" {
		t.Fatalf("contextMap = %#v, want traceId", decoded["contextMap"])
	}
}

func TestGELFLayout_whenEventFormatted_shouldWriteGELFJSON(t *testing.T) {
	event := BenchmarkEvent()
	event.Attrs = append(event.Attrs, slog.String("traceId", "trace-1"))

	var buf bytes.Buffer
	if err := (GELFLayout{}).Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v, json=%s", err, buf.String())
	}
	if decoded["version"] != "1.1" || decoded["short_message"] != "service started" {
		t.Fatalf("GELF output = %#v", decoded)
	}
	if decoded["_traceId"] != "trace-1" {
		t.Fatalf("_traceId = %#v, want trace-1", decoded["_traceId"])
	}
}

func TestConsoleAppender_whenCompleteJSONLayoutUsed_shouldWriteValidArray(t *testing.T) {
	var out bytes.Buffer
	appender := NewConsoleAppender(
		WithConsoleWriter(&out),
		WithConsoleLayout(NewJSONLayout(LayoutOptions{Compact: true, Complete: true})),
	)
	if err := appender.Append(
		context.Background(),
		TestEvent("console-first", FixedTestTime()),
	); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if err := appender.Append(
		context.Background(),
		TestEvent("console-second", FixedTestTime()),
	); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	AssertCompleteJSONContentMessages(t, out.String(), "console-first", "console-second")
}

func TestBuildLayout_whenStructuredLayoutTypesUsed_shouldResolveBuiltIns(t *testing.T) {
	for _, kind := range []string{"gelf", "rfc5424", "syslog", "yaml", "html"} {
		layout, err := BuildLayout(LayoutConfig{Type: kind}, DefaultPluginRegistry())
		if err != nil {
			t.Fatalf("buildLayout(%q) error = %v", kind, err)
		}
		if layout == nil {
			t.Fatalf("buildLayout(%q) returned nil layout", kind)
		}
	}
}
