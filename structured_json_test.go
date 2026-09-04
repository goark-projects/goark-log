package goarklog_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	goarklog "goark.dev/log"
)

type cyclicUnwrapError struct {
	message string
	cause   error
}

func (e *cyclicUnwrapError) Error() string { return e.message }

func (e *cyclicUnwrapError) Unwrap() error { return e.cause }

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
			if test.format == goarklog.StructuredFormatGELF {
				contextKey = "_trace_id"
			}
			if decoded[contextKey] != "trace-1" {
				t.Fatalf("context field %q = %#v", contextKey, decoded[contextKey])
			}
			if test.format == goarklog.StructuredFormatECS && decoded["@timestamp"] != "2026-09-04T02:20:30.123Z" {
				t.Fatalf("ECS timestamp = %#v", decoded["@timestamp"])
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

func TestStructuredJSONLayoutKeepsFirstMemberOnNameCollision(t *testing.T) {
	layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
		Format: goarklog.StructuredFormatLogstash,
		Rename: map[string]string{"logger_name": "message"},
		Add:    map[string]string{"message": "added"},
		Customizers: []goarklog.StructuredJSONCustomizer{
			goarklog.StructuredJSONCustomizerFunc(func(_ goarklog.Event, fields goarklog.StructuredJSONFieldAppender) {
				fields.Add("message", slog.StringValue("customized"))
			}),
		},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	if err := layout.Format(&output, goarklog.Event{Message: "original", Logger: "logger"}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if strings.Count(output.String(), `"message":`) != 1 {
		t.Fatalf("output contains duplicate message members: %s", output.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["message"] != "original" {
		t.Fatalf("message = %#v", decoded["message"])
	}
}

func TestStructuredJSONLayoutECSFieldFilters(t *testing.T) {
	tests := []struct {
		name    string
		include []string
		exclude []string
		wantLog bool
		wantPID bool
	}{
		{name: "include parent", include: []string{"process"}, wantLog: false, wantPID: false},
		{name: "include child", include: []string{"process.pid"}, wantLog: false, wantPID: true},
		{name: "exclude parent", exclude: []string{"process"}, wantLog: true, wantPID: false},
		{name: "exclude child", exclude: []string{"process.pid"}, wantLog: true, wantPID: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
				Format: goarklog.StructuredFormatECS, Include: test.include, Exclude: test.exclude,
			})
			if err != nil {
				t.Fatalf("NewStructuredJSONLayout() error = %v", err)
			}
			var output bytes.Buffer
			if err := layout.Format(&output, goarklog.Event{}); err != nil {
				t.Fatalf("Format() error = %v", err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			_, hasLog := decoded["log"]
			process, _ := decoded["process"].(map[string]any)
			_, hasPID := process["pid"]
			if hasLog != test.wantLog || hasPID != test.wantPID {
				t.Fatalf("decoded = %#v", decoded)
			}
		})
	}
}

func TestStructuredJSONLayoutRejectsConflictingNestedAddPaths(t *testing.T) {
	_, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
		Format: goarklog.StructuredFormatECS,
		Add:    map[string]string{"build": "42", "build.version": "1.2.3"},
	})
	if err == nil {
		t.Fatal("NewStructuredJSONLayout() error = nil")
	}
}

func TestStructuredJSONLayoutStacktraceLimits(t *testing.T) {
	layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
		Format: goarklog.StructuredFormatECS,
		Stacktrace: goarklog.StructuredStacktraceOptions{
			RootFirst:         true,
			MaxThrowableDepth: 1,
			IncludeHashes:     true,
		},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	event := goarklog.Event{Throwable: &goarklog.Throwable{
		Type: "outer", Message: "outer error", Stack: []string{"outer.go:10", "outer.go:11"},
		Cause: &goarklog.Throwable{Type: "root", Message: "root error", Stack: []string{"root.go:20", "root.go:21"}},
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
	if !strings.Contains(stack, "<#") || strings.Contains(stack, "outer.go:11") || strings.Contains(stack, "root.go:21") {
		t.Fatalf("stack trace = %q", stack)
	}
	for _, expected := range []string{"root: root error", "Wrapped by: <#", "outer: outer error", "\tat root.go:20", "\tat outer.go:10"} {
		if !strings.Contains(stack, expected) {
			t.Fatalf("stack trace does not contain %q: %q", expected, stack)
		}
	}
}

func TestStructuredJSONLayoutStacktraceMaximumLengthUsesEllipsis(t *testing.T) {
	layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
		Format:     goarklog.StructuredFormatECS,
		Stacktrace: goarklog.StructuredStacktraceOptions{MaxLength: 14},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	if err := layout.Format(&output, goarklog.Event{Throwable: &goarklog.Throwable{Type: "错误类型", Message: "异常消息"}}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	errorFields, _ := decoded["error"].(map[string]any)
	stack, _ := errorFields["stack_trace"].(string)
	if len(stack) > 14 || !strings.HasSuffix(stack, "...") || !utf8.ValidString(stack) {
		t.Fatalf("stack trace = %q", stack)
	}
}

func TestStructuredJSONLayoutStacktraceOmitsCommonFrames(t *testing.T) {
	layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
		Format: goarklog.StructuredFormatECS,
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	event := goarklog.Event{Throwable: &goarklog.Throwable{
		Type: "outer", Message: "outer error", Stack: []string{"outer.go:10", "shared.go:30"},
		Cause: &goarklog.Throwable{Type: "root", Message: "root error", Stack: []string{"root.go:20", "shared.go:30"}},
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
	if !strings.Contains(stack, "Caused by: root: root error") || !strings.Contains(stack, "... 1 more") {
		t.Fatalf("stack trace = %q", stack)
	}
}

func TestStructuredJSONLayoutRootFirstComparesCommonFramesWithWrapper(t *testing.T) {
	layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
		Format:     goarklog.StructuredFormatECS,
		Stacktrace: goarklog.StructuredStacktraceOptions{RootFirst: true},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	if err := layout.Format(&output, goarklog.Event{Throwable: &goarklog.Throwable{
		Type: "outer", Message: "outer", Stack: []string{"outer.go:10", "shared.go:30"},
		Cause: &goarklog.Throwable{Type: "root", Message: "root", Stack: []string{"root.go:20", "shared.go:30"}},
	}}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	errorFields, _ := decoded["error"].(map[string]any)
	stack, _ := errorFields["stack_trace"].(string)
	if strings.Count(stack, "shared.go:30") != 1 || !strings.Contains(stack, "root: root\n\tat root.go:20\n\t... 1 more") {
		t.Fatalf("stack trace = %q", stack)
	}
}

func TestStructuredJSONLayoutECSUsesNestedPathsAndMarkerArrays(t *testing.T) {
	layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
		Format:         goarklog.StructuredFormatECS,
		IncludeContext: true,
		ContextPrefix:  "metadata",
		Add:            map[string]string{"build.version": "1.2.3"},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	marker := goarklog.NewMarker("HTTP", goarklog.NewMarker("REQUEST"))
	var output bytes.Buffer
	if err := layout.Format(&output, goarklog.Event{
		Message: "handled",
		Attrs:   []slog.Attr{slog.String("request.id", "request-1")},
		Marker:  &marker,
	}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
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

func TestStructuredJSONLayoutLogstashUsesMarkerArray(t *testing.T) {
	layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{Format: goarklog.StructuredFormatLogstash})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	marker := goarklog.NewMarker("HTTP")
	var output bytes.Buffer
	if err := layout.Format(&output, goarklog.Event{Message: "handled", Marker: &marker}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	tags, ok := decoded["tags"].([]any)
	if !ok || len(tags) != 1 || tags[0] != "HTTP" {
		t.Fatalf("tags = %#v", decoded["tags"])
	}
}

func TestStructuredJSONLayoutGELFFullMessageIncludesLogMessage(t *testing.T) {
	layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{Format: goarklog.StructuredFormatGELF})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	if err := layout.Format(&output, goarklog.Event{
		Message:   "request failed",
		Throwable: &goarklog.Throwable{Type: "failure", Message: "broken"},
	}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["full_message"] != "request failed\n\nfailure: broken" {
		t.Fatalf("full_message = %#v", decoded["full_message"])
	}
}

func TestStructuredJSONLayoutLoggingSystemPrinterKeepsCommonFrames(t *testing.T) {
	layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
		Format: goarklog.StructuredFormatECS,
		Stacktrace: goarklog.StructuredStacktraceOptions{
			Printer: goarklog.StructuredStacktracePrinterLoggingSystem,
		},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	if err := layout.Format(&output, goarklog.Event{Throwable: &goarklog.Throwable{
		Type: "outer", Message: "outer", Stack: []string{"outer.go:10", "shared.go:30"},
		Cause: &goarklog.Throwable{Type: "root", Message: "root", Stack: []string{"root.go:20", "shared.go:30"}},
	}}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	errorFields, _ := decoded["error"].(map[string]any)
	stack, _ := errorFields["stack_trace"].(string)
	if strings.Count(stack, "shared.go:30") != 2 || strings.Contains(stack, "... 1 more") {
		t.Fatalf("stack trace = %q", stack)
	}
}

func TestStructuredJSONLayoutHandlesCircularThrowable(t *testing.T) {
	for _, printer := range []goarklog.StructuredStacktracePrinter{
		goarklog.StructuredStacktracePrinterStandard,
		goarklog.StructuredStacktracePrinterLoggingSystem,
	} {
		t.Run(string(printer), func(t *testing.T) {
			layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
				Format: goarklog.StructuredFormatECS, Stacktrace: goarklog.StructuredStacktraceOptions{Printer: printer},
			})
			if err != nil {
				t.Fatalf("NewStructuredJSONLayout() error = %v", err)
			}
			throwable := &goarklog.Throwable{Type: "cycle", Message: "broken"}
			throwable.Cause = throwable
			var output bytes.Buffer
			if err := layout.Format(&output, goarklog.Event{Throwable: throwable}); err != nil {
				t.Fatalf("Format() error = %v", err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			errorFields, _ := decoded["error"].(map[string]any)
			stack, _ := errorFields["stack_trace"].(string)
			if !strings.Contains(stack, "[CIRCULAR REFERENCE: cycle: broken]") {
				t.Fatalf("stack trace = %q", stack)
			}
		})
	}
}

func TestNewThrowableStopsCircularErrorChain(t *testing.T) {
	outer := &cyclicUnwrapError{message: "outer"}
	inner := &cyclicUnwrapError{message: "inner"}
	outer.cause = inner
	inner.cause = outer
	throwable := goarklog.NewThrowable(outer)
	if throwable == nil || throwable.Message != "outer" || throwable.Cause == nil || throwable.Cause.Message != "inner" {
		t.Fatalf("throwable = %#v", throwable)
	}
	if throwable.Cause.Cause != nil {
		t.Fatalf("circular cause was retained: %#v", throwable.Cause.Cause)
	}
}

func TestStructuredJSONLayoutRejectsUnknownFormat(t *testing.T) {
	if _, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{Format: "custom"}); err == nil {
		t.Fatal("NewStructuredJSONLayout() error = nil")
	}
}

func BenchmarkStructuredJSONLayoutECS(b *testing.B) {
	layout, err := goarklog.NewStructuredJSONLayout(goarklog.StructuredJSONOptions{
		Format:         goarklog.StructuredFormatECS,
		IncludeContext: true,
	})
	if err != nil {
		b.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	event := goarklog.Event{
		Time:    time.Date(2026, 9, 4, 10, 20, 30, 123000000, time.UTC),
		Level:   slog.LevelInfo,
		Logger:  "goark.dev.admin",
		Message: "request completed",
		Attrs:   []slog.Attr{slog.String("trace_id", "trace-1")},
	}
	var output bytes.Buffer
	b.ReportAllocs()
	for b.Loop() {
		output.Reset()
		if err := layout.Format(&output, event); err != nil {
			b.Fatal(err)
		}
	}
}
