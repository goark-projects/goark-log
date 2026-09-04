package goarklog_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	goarklog "goark.dev/log"
)

func TestStructuredJSONLayoutFormats(t *testing.T) {
	tests := []struct {
		format goarklog.StructuredFormat
		want   []string
	}{
		{goarklog.StructuredFormatECS, []string{"@timestamp", "log", "ecs"}},
		{goarklog.StructuredFormatGELF, []string{"version", "short_message", "_level_name"}},
		{goarklog.StructuredFormatLogstash, []string{"@version", "logger_name", "level_value"}},
	}
	for _, test := range tests {
		t.Run(string(test.format), func(t *testing.T) {
			layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
				Format:         test.format,
				IncludeContext: true,
			})
			if err != nil {
				t.Fatalf("NewStructuredJSONLayout() error = %v", err)
			}
			var output bytes.Buffer
			event := goarklog.Event{
				Time:       time.Date(2026, 9, 4, 10, 20, 30, 123000000, time.FixedZone("CST", 8*60*60)),
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
			if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
				t.Fatalf("Unmarshal() error = %v, output=%s", err, output.String())
			}
			for _, key := range test.want {
				if _, found := decoded[key]; !found {
					t.Fatalf("output missing %q: %s", key, output.String())
				}
			}
			if test.format == goarklog.StructuredFormatECS {
				logFields, ok := decoded["log"].(map[string]any)
				if !ok || logFields["level"] != "INFO" || logFields["logger"] != "goark.dev.admin" {
					t.Fatalf("ECS log object = %#v", decoded["log"])
				}
			}
			contextKey := "trace_id"
			if test.format != goarklog.StructuredFormatECS {
				contextKey = "_trace_id"
			}
			if decoded[contextKey] != "trace-1" {
				t.Fatalf("context field %q = %#v", contextKey, decoded[contextKey])
			}
		})
	}
}

func TestStructuredJSONLayoutTransformsAndCustomizer(t *testing.T) {
	layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
		Format:  goarklog.StructuredFormatLogstash,
		Include: []string{"message", "fixed", "custom"},
		Rename:  map[string]string{"message": "msg"},
		Add:     map[string]string{"fixed": "value"},
		Customizers: []goarklog.StructuredJSONCustomizer{
			goarklog.StructuredJSONCustomizerFunc(func(_ goarklog.Event, fields goarklog.StructuredJSONFieldAppender) {
				fields.Add("custom", slog.StringValue("ok"))
			}),
		},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	if err := layout.Format(&output, goarklog.Event{Message: "hello"}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["msg"] != "hello" || decoded["fixed"] != "value" || decoded["custom"] != "ok" || len(decoded) != 3 {
		t.Fatalf("decoded = %#v", decoded)
	}
}

func TestStructuredJSONLayoutStacktraceLimits(t *testing.T) {
	layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
		Format: goarklog.StructuredFormatECS,
		Stacktrace: goarklog.StructuredStacktraceOptions{
			RootFirst:         true,
			MaxLength:         80,
			MaxThrowableDepth: 1,
			IncludeHashes:     true,
		},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	event := goarklog.Event{Throwable: &goarklog.Throwable{
		Type: "outer", Message: "outer error", Stack: []string{"outer.go:10"},
		Cause: &goarklog.Throwable{Type: "root", Message: "root error", Stack: []string{"root.go:20"}},
	}}
	if err := layout.Format(&output, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	errorFields, _ := decoded["error"].(map[string]any)
	stack, _ := errorFields["stack_trace"].(string)
	if len(stack) > 80 || !strings.Contains(stack, "<#") {
		t.Fatalf("stack trace = %q", stack)
	}
}

func TestStructuredJSONLayoutRejectsUnknownFormat(t *testing.T) {
	if _, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{Format: "custom"}); err == nil {
		t.Fatal("NewStructuredJSONLayout() error = nil")
	}
}
