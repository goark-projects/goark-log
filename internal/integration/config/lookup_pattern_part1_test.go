package integration

import (
	"bytes"
	"errors"
	"log/slog"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	. "goark.dev/log/internal/testsupport"
)

func TestLayout_whenPrimitiveAttrsUsed_shouldRenderWithoutSemanticDrift(t *testing.T) {
	event := TestEvent("primitive attrs", FixedTestTime())
	event.Attrs = []slog.Attr{
		slog.String("profile", "bench worker"),
		slog.Int("index", 42),
		slog.Bool("cached", true),
		slog.Float64("ratio", 1.25),
		slog.Duration("elapsed", 10*time.Millisecond),
	}

	var text bytes.Buffer
	if err := (TextLayout{}).Format(&text, event); err != nil {
		t.Fatalf("TextLayout.Format() error = %v", err)
	}
	textLine := text.String()
	for _, want := range []string{
		`profile="bench worker"`,
		"index=42",
		"cached=true",
		"ratio=1.25",
		"elapsed=10ms",
	} {
		if !strings.Contains(textLine, want) {
			t.Fatalf("text line should contain %q, got %q", want, textLine)
		}
	}

	layout, err := NewPatternLayout("%m%attrs%n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	var pattern bytes.Buffer
	if err := layout.Format(&pattern, event); err != nil {
		t.Fatalf("PatternLayout.Format() error = %v", err)
	}
	if !strings.Contains(
		pattern.String(),
		`primitive attrs profile="bench worker" index=42 cached=true ratio=1.25 elapsed=10ms`,
	) {
		t.Fatalf("pattern line is wrong: %q", pattern.String())
	}
}

func TestPatternLayout_whenExtendedConvertersUsed_shouldRenderLog4jStyleOutput(t *testing.T) {
	layout, err := NewPatternLayout(
		"%logger{2} %map %highlight{%p} %style{%m}{red} %notEmpty{%X{trace_id}} %uuid%n",
	)
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	event := TestEvent("request done", FixedTestTime())
	event.Logger = "dev.goark.web.audit"
	event.Attrs = []slog.Attr{slog.String("trace_id", "trace-1"), slog.Int("status", 200)}

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	parts := strings.Fields(buf.String())
	if len(parts) < 7 {
		t.Fatalf("formatted line = %q, want fields", buf.String())
	}
	if parts[0] != "web.audit" {
		t.Fatalf("logger precision = %q, want web.audit", parts[0])
	}
	if !strings.Contains(buf.String(), "trace_id=trace-1") ||
		!strings.Contains(buf.String(), "status=200") {
		t.Fatalf("map converter output = %q", buf.String())
	}
	if !strings.Contains(buf.String(), "INFO") || !strings.Contains(buf.String(), "request done") ||
		!strings.Contains(buf.String(), "trace-1") {
		t.Fatalf("sub pattern converter output = %q", buf.String())
	}
	uuid := strings.TrimSpace(parts[len(parts)-1])
	if len(uuid) != 36 || uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		t.Fatalf("uuid = %q, want RFC4122 text form", uuid)
	}
}

func TestPatternLayout_whenAnsiStyleConvertersUsed_shouldRenderEscapeSequences(t *testing.T) {
	styleLayout, err := NewPatternLayout("%style{%m}{red,bold}%n")
	if err != nil {
		t.Fatalf("NewPatternLayout(style) error = %v", err)
	}
	event := TestEvent("styled", FixedTestTime())
	var styled bytes.Buffer
	if err := styleLayout.Format(&styled, event); err != nil {
		t.Fatalf("Format(style) error = %v", err)
	}
	if styled.String() != "\x1b[31;1mstyled\x1b[0m\n" {
		t.Fatalf("style output = %q, want ANSI red bold", styled.String())
	}

	highlightLayout, err := NewPatternLayout("%highlight{%p}%n")
	if err != nil {
		t.Fatalf("NewPatternLayout(highlight) error = %v", err)
	}
	event.Level = slog.LevelWarn
	var highlighted bytes.Buffer
	if err := highlightLayout.Format(&highlighted, event); err != nil {
		t.Fatalf("Format(highlight) error = %v", err)
	}
	if highlighted.String() != "\x1b[33mWARN\x1b[0m\n" {
		t.Fatalf("highlight output = %q, want ANSI warning color", highlighted.String())
	}
}

func TestPatternLayout_whenAttrsHaveExplicitSeparator_shouldNotDuplicateSpace(t *testing.T) {
	event := TestEvent("attrs", FixedTestTime())
	event.Attrs = []slog.Attr{slog.String("trace_id", "trace-1")}
	cases := []struct {
		pattern string
		want    string
	}{
		{pattern: "%m%attrs%n", want: "attrs trace_id=trace-1\n"},
		{pattern: "%m %attrs%n", want: "attrs trace_id=trace-1\n"},
		{pattern: "%attrs%n", want: "trace_id=trace-1\n"},
	}
	for _, tt := range cases {
		layout, err := NewPatternLayout(tt.pattern)
		if err != nil {
			t.Fatalf("NewPatternLayout(%q) error = %v", tt.pattern, err)
		}
		var buf bytes.Buffer
		if err := layout.Format(&buf, event); err != nil {
			t.Fatalf("Format(%q) error = %v", tt.pattern, err)
		}
		if buf.String() != tt.want {
			t.Fatalf("Format(%q) = %q, want %q", tt.pattern, buf.String(), tt.want)
		}
	}
}

func TestPatternLayout_whenLog4jStyleTokensUsed_shouldRenderEventFields(t *testing.T) {
	layout, err := NewPatternLayout("%d{yyyy-MM-dd HH:mm:ss.SSS} %5p %c %X{trace_id} %ex %m %% %n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	event := TestEvent("request done", FixedTestTime())
	event.Level = slog.LevelInfo
	event.Logger = "goark.web"
	event.Attrs = []slog.Attr{
		slog.String("trace_id", "abc-123"),
		slog.Any("error", errors.New("boom")),
	}

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	want := "2026-08-25 10:15:30.123  INFO goark.web abc-123 boom request done % \n"
	if buf.String() != want {
		t.Fatalf("formatted line = %q, want %q", buf.String(), want)
	}
}

func TestPatternLayout_whenUnicodeValueIsTruncated_shouldKeepValidUTF8(t *testing.T) {
	layout, err := NewPatternLayout("%.1msg|%.-2msg|%4.2msg|%05msg|%maxLen{%msg}{2}%n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}

	var buf bytes.Buffer
	if err := layout.Format(&buf, TestEvent("界abc", FixedTestTime())); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	got := buf.String()
	if !utf8.ValidString(got) {
		t.Fatalf("formatted line is not valid UTF-8: %q", got)
	}
	want := "c|界a|  bc|0界abc|界a\n"
	if got != want {
		t.Fatalf("formatted line = %q, want %q", got, want)
	}
}

func callerProgramCounter(t *testing.T, name string) uintptr {
	t.Helper()
	var pcs [32]uintptr
	count := runtime.Callers(0, pcs[:])
	frames := runtime.CallersFrames(pcs[:count])
	for {
		frame, more := frames.Next()
		if strings.Contains(frame.Function, name) {
			return frame.PC
		}
		if !more {
			break
		}
	}
	t.Fatalf("caller frame %q not found", name)
	return 0
}

func TestPatternLayout_whenUnixMillisDateUsed_shouldRenderEpochMillis(t *testing.T) {
	layout, err := NewPatternLayout("%d{UNIX_MILLIS} %p %m%n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	var buf bytes.Buffer
	if err := layout.Format(&buf, TestEvent("epoch", FixedTestTime())); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if !strings.HasPrefix(buf.String(), "1787624130123 INFO epoch\n") {
		t.Fatalf("formatted line = %q", buf.String())
	}
}
