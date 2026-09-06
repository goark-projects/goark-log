package log_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"goark.dev/log"
)

func TestStructuredJSONLayoutFormats(t *testing.T) {
	tests := []struct {
		format log.StructuredFormat
		want   []string
	}{
		{log.StructuredFormatECS, []string{"@timestamp", "log", "ecs"}},
		{log.StructuredFormatGELF, []string{"version", "short_message", "_level_name"}},
		{log.StructuredFormatLogstash, []string{"@version", "logger_name", "level_value"}},
	}
	for _, test := range tests {
		t.Run(string(test.format), func(t *testing.T) {
			layout, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{
				Format:         test.format,
				IncludeContext: true,
			})
			if err != nil {
				t.Fatalf("NewStructuredJSONLayout() error = %v", err)
			}
			var output bytes.Buffer
			event := log.Event{
				Time: time.Date(
					2026,
					9,
					4,
					10,
					20,
					30,
					123000000,
					time.FixedZone("CST", 8*60*60),
				),
				Level:      slog.LevelInfo,
				Message:    "started",
				Logger:     "goark.dev.admin",
				ThreadName: "main",
				Attrs:      []slog.Attr{slog.String("trace_id", "trace-1")},
			}
			if err := layout.Format(&output, event); err != nil {
				t.Fatalf("Format() error = %v", err)
			}
			var decoded map[string]any
			if err := sonic.Unmarshal(output.Bytes(), &decoded); err != nil {
				t.Fatalf("Unmarshal() error = %v, output=%s", err, output.String())
			}
			for _, key := range test.want {
				if _, found := decoded[key]; !found {
					t.Fatalf("output missing %q: %s", key, output.String())
				}
			}
			if test.format == log.StructuredFormatECS {
				logFields, ok := decoded["log"].(map[string]any)
				if !ok || logFields["level"] != "INFO" || logFields["logger"] != "goark.dev.admin" {
					t.Fatalf("ECS log object = %#v", decoded["log"])
				}
			}
			contextKey := "trace_id"
			if test.format == log.StructuredFormatGELF {
				contextKey = "_trace_id"
			}
			if decoded[contextKey] != "trace-1" {
				t.Fatalf("context field %q = %#v", contextKey, decoded[contextKey])
			}
			if test.format == log.StructuredFormatECS &&
				decoded["@timestamp"] != "2026-09-04T02:20:30.123Z" {
				t.Fatalf("ECS timestamp = %#v", decoded["@timestamp"])
			}
		})
	}
}

func TestStructuredJSONLayoutECSUsesNestedPathsAndMarkerArrays(t *testing.T) {
	layout, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{
		Format:         log.StructuredFormatECS,
		IncludeContext: true,
		ContextPrefix:  "metadata",
		Add:            map[string]string{"build.version": "1.2.3"},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	marker := log.NewMarker("HTTP", log.NewMarker("REQUEST"))
	var output bytes.Buffer
	if err := layout.Format(&output, log.Event{
		Message: "handled",
		Attrs:   []slog.Attr{slog.String("request.id", "request-1")},
		Marker:  &marker,
	}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	build, _ := decoded["build"].(map[string]any)
	metadata, _ := decoded["metadata"].(map[string]any)
	request, _ := metadata["request"].(map[string]any)
	if build["version"] != "1.2.3" || request["id"] != "request-1" {
		t.Fatalf("decoded = %#v", decoded)
	}
	tags, ok := decoded["tags"].([]any)
	if !ok || len(tags) != 2 || tags[0] != "HTTP" || tags[1] != "REQUEST" {
		t.Fatalf("tags = %#v", decoded["tags"])
	}
}

func TestStructuredJSONLayoutKeepsFirstMemberOnNameCollision(t *testing.T) {
	layout, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{
		Format: log.StructuredFormatLogstash,
		Rename: map[string]string{"logger_name": "message"},
		Add:    map[string]string{"message": "added"},
		Customizers: []log.StructuredJSONCustomizer{
			log.StructuredJSONCustomizerFunc(
				func(_ log.Event, fields log.StructuredJSONFieldAppender) {
					fields.Add("message", slog.StringValue("customized"))
				},
			),
		},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	if err := layout.Format(&output, log.Event{Message: "original", Logger: "logger"}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if strings.Count(output.String(), `"message":`) != 1 {
		t.Fatalf("output contains duplicate message members: %s", output.String())
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["message"] != "original" {
		t.Fatalf("message = %#v", decoded["message"])
	}
}

func TestStructuredJSONLayoutLoggingSystemPrinterKeepsCommonFrames(t *testing.T) {
	layout, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{
		Format: log.StructuredFormatECS,
		Stacktrace: log.StructuredStacktraceOptions{
			Printer: log.StructuredStacktracePrinterLoggingSystem,
		},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	if err := layout.Format(&output, log.Event{Throwable: &log.Throwable{
		Type: "outer", Message: "outer", Stack: []string{"outer.go:10", "shared.go:30"},
		Cause: &log.Throwable{
			Type: "root", Message: "root", Stack: []string{"root.go:20", "shared.go:30"},
		},
	}}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	errorFields, _ := decoded["error"].(map[string]any)
	stack, _ := errorFields["stack_trace"].(string)
	if strings.Count(stack, "shared.go:30") != 2 || strings.Contains(stack, "... 1 more") {
		t.Fatalf("stack trace = %q", stack)
	}
}

func TestStructuredJSONLayoutRootFirstComparesCommonFramesWithWrapper(t *testing.T) {
	layout, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{
		Format:     log.StructuredFormatECS,
		Stacktrace: log.StructuredStacktraceOptions{RootFirst: true},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	if err := layout.Format(&output, log.Event{Throwable: &log.Throwable{
		Type: "outer", Message: "outer", Stack: []string{"outer.go:10", "shared.go:30"},
		Cause: &log.Throwable{
			Type: "root", Message: "root", Stack: []string{"root.go:20", "shared.go:30"},
		},
	}}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	errorFields, _ := decoded["error"].(map[string]any)
	stack, _ := errorFields["stack_trace"].(string)
	if strings.Count(stack, "shared.go:30") != 1 ||
		!strings.Contains(stack, "root: root\n\tat root.go:20\n\t... 1 more") {
		t.Fatalf("stack trace = %q", stack)
	}
}

func TestStructuredJSONLayoutStacktraceMaximumLengthUsesEllipsis(t *testing.T) {
	layout, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{
		Format:     log.StructuredFormatECS,
		Stacktrace: log.StructuredStacktraceOptions{MaxLength: 14},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	event := log.Event{
		Throwable: &log.Throwable{Type: "错误类型", Message: "异常消息"},
	}
	if err := layout.Format(&output, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	errorFields, _ := decoded["error"].(map[string]any)
	stack, _ := errorFields["stack_trace"].(string)
	if len(stack) > 14 || !strings.HasSuffix(stack, "...") || !utf8.ValidString(stack) {
		t.Fatalf("stack trace = %q", stack)
	}
}

func TestStructuredJSONLayoutLogstashUsesMarkerArray(t *testing.T) {
	layout, err := log.NewStructuredJSONLayout(
		log.StructuredJSONOptions{Format: log.StructuredFormatLogstash},
	)
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	marker := log.NewMarker("HTTP")
	var output bytes.Buffer
	if err := layout.Format(&output, log.Event{Message: "handled", Marker: &marker}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	tags, ok := decoded["tags"].([]any)
	if !ok || len(tags) != 1 || tags[0] != "HTTP" {
		t.Fatalf("tags = %#v", decoded["tags"])
	}
}

func TestStructuredJSONLayoutRejectsUnknownFormat(t *testing.T) {
	if _, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{Format: "custom"}); err == nil {
		t.Fatal("NewStructuredJSONLayout() error = nil")
	}
}

type cyclicUnwrapError struct {
	message string
	cause   error
}

func (e *cyclicUnwrapError) Unwrap() error { return e.cause }
