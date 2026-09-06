package log_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"goark.dev/log"
)

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
			layout, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{
				Format: log.StructuredFormatECS, Include: test.include, Exclude: test.exclude,
			})
			if err != nil {
				t.Fatalf("NewStructuredJSONLayout() error = %v", err)
			}
			var output bytes.Buffer
			if err := layout.Format(&output, log.Event{}); err != nil {
				t.Fatalf("Format() error = %v", err)
			}
			var decoded map[string]any
			if err := sonic.Unmarshal(output.Bytes(), &decoded); err != nil {
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

func TestStructuredJSONLayoutStacktraceLimits(t *testing.T) {
	layout, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{
		Format: log.StructuredFormatECS,
		Stacktrace: log.StructuredStacktraceOptions{
			RootFirst:         true,
			MaxThrowableDepth: 1,
			IncludeHashes:     true,
		},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	event := log.Event{Throwable: &log.Throwable{
		Type: "outer", Message: "outer error", Stack: []string{"outer.go:10", "outer.go:11"},
		Cause: &log.Throwable{
			Type:    "root",
			Message: "root error",
			Stack:   []string{"root.go:20", "root.go:21"},
		},
	}}
	if err := layout.Format(&output, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	errorFields, _ := decoded["error"].(map[string]any)
	stack, _ := errorFields["stack_trace"].(string)
	if !strings.Contains(stack, "<#") || strings.Contains(stack, "outer.go:11") ||
		strings.Contains(stack, "root.go:21") {
		t.Fatalf("stack trace = %q", stack)
	}
	expectedParts := []string{
		"root: root error",
		"Wrapped by: <#",
		"outer: outer error",
		"\tat root.go:20",
		"\tat outer.go:10",
	}
	for _, expected := range expectedParts {
		if !strings.Contains(stack, expected) {
			t.Fatalf("stack trace does not contain %q: %q", expected, stack)
		}
	}
}

func TestStructuredJSONLayoutHandlesCircularThrowable(t *testing.T) {
	for _, printer := range []log.StructuredStacktracePrinter{
		log.StructuredStacktracePrinterStandard,
		log.StructuredStacktracePrinterLoggingSystem,
	} {
		t.Run(string(printer), func(t *testing.T) {
			layout, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{
				Format: log.StructuredFormatECS, Stacktrace: log.StructuredStacktraceOptions{Printer: printer},
			})
			if err != nil {
				t.Fatalf("NewStructuredJSONLayout() error = %v", err)
			}
			throwable := &log.Throwable{Type: "cycle", Message: "broken"}
			throwable.Cause = throwable
			var output bytes.Buffer
			if err := layout.Format(&output, log.Event{Throwable: throwable}); err != nil {
				t.Fatalf("Format() error = %v", err)
			}
			var decoded map[string]any
			if err := sonic.Unmarshal(output.Bytes(), &decoded); err != nil {
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

func TestStructuredJSONLayoutTransformsAndCustomizer(t *testing.T) {
	layout, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{
		Format:  log.StructuredFormatLogstash,
		Include: []string{"message", "fixed", "custom"},
		Rename:  map[string]string{"message": "msg"},
		Add:     map[string]string{"fixed": "value"},
		Customizers: []log.StructuredJSONCustomizer{
			log.StructuredJSONCustomizerFunc(
				func(_ log.Event, fields log.StructuredJSONFieldAppender) {
					fields.Add("custom", slog.StringValue("ok"))
				},
			),
		},
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	if err := layout.Format(&output, log.Event{Message: "hello"}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["msg"] != "hello" || decoded["fixed"] != "value" || decoded["custom"] != "ok" ||
		len(decoded) != 3 {
		t.Fatalf("decoded = %#v", decoded)
	}
}

func TestStructuredJSONLayoutStacktraceOmitsCommonFrames(t *testing.T) {
	layout, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{
		Format: log.StructuredFormatECS,
	})
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	event := log.Event{Throwable: &log.Throwable{
		Type: "outer", Message: "outer error", Stack: []string{"outer.go:10", "shared.go:30"},
		Cause: &log.Throwable{
			Type:    "root",
			Message: "root error",
			Stack:   []string{"root.go:20", "shared.go:30"},
		},
	}}
	if err := layout.Format(&output, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	errorFields, _ := decoded["error"].(map[string]any)
	stack, _ := errorFields["stack_trace"].(string)
	if !strings.Contains(stack, "Caused by: root: root error") ||
		!strings.Contains(stack, "... 1 more") {
		t.Fatalf("stack trace = %q", stack)
	}
}

func BenchmarkStructuredJSONLayoutECS(b *testing.B) {
	layout, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{
		Format:         log.StructuredFormatECS,
		IncludeContext: true,
	})
	if err != nil {
		b.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	event := log.Event{
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
func TestStructuredJSONLayoutGELFFullMessageIncludesLogMessage(t *testing.T) {
	layout, err := log.NewStructuredJSONLayout(
		log.StructuredJSONOptions{Format: log.StructuredFormatGELF},
	)
	if err != nil {
		t.Fatalf("NewStructuredJSONLayout() error = %v", err)
	}
	var output bytes.Buffer
	if err := layout.Format(&output, log.Event{
		Message:   "request failed",
		Throwable: &log.Throwable{Type: "failure", Message: "broken"},
	}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["full_message"] != "request failed\n\nfailure: broken" {
		t.Fatalf("full_message = %#v", decoded["full_message"])
	}
}

func TestNewThrowableStopsCircularErrorChain(t *testing.T) {
	outer := &cyclicUnwrapError{message: "outer"}
	inner := &cyclicUnwrapError{message: "inner"}
	outer.cause = inner
	inner.cause = outer
	throwable := log.NewThrowable(outer)
	if throwable == nil || throwable.Message != "outer" || throwable.Cause == nil ||
		throwable.Cause.Message != "inner" {
		t.Fatalf("throwable = %#v", throwable)
	}
	if throwable.Cause.Cause != nil {
		t.Fatalf("circular cause was retained: %#v", throwable.Cause.Cause)
	}
}

func TestStructuredJSONLayoutRejectsConflictingNestedAddPaths(t *testing.T) {
	_, err := log.NewStructuredJSONLayout(log.StructuredJSONOptions{
		Format: log.StructuredFormatECS,
		Add:    map[string]string{"build": "42", "build.version": "1.2.3"},
	})
	if err == nil {
		t.Fatal("NewStructuredJSONLayout() error = nil")
	}
}

func (e *cyclicUnwrapError) Error() string { return e.message }
