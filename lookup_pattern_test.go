package goarklog

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLookupResolver_whenEnvAndSystemLookupsUsed_shouldResolveText(t *testing.T) {
	t.Setenv("GOARK_LOG_PROFILE", "dev")
	resolver := NewLookupResolver()
	text, err := resolver.Resolve("profile=${env:GOARK_LOG_PROFILE},missing=${env:GOARK_LOG_MISSING:-local},pid=${sys:pid}")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !strings.Contains(text, "profile=dev") || !strings.Contains(text, "missing=local") || !strings.Contains(text, "pid=") {
		t.Fatalf("resolved text is wrong: %q", text)
	}
}

func TestLookupResolver_whenLookupMissingWithoutDefault_shouldReject(t *testing.T) {
	resolver := NewLookupResolver()
	_, err := resolver.Resolve("${env:GOARK_LOG_NOT_SET}")
	if err == nil {
		t.Fatalf("Resolve() should reject missing lookup without default")
	}
}

func TestPatternLayout_whenLog4jStyleTokensUsed_shouldRenderEventFields(t *testing.T) {
	layout, err := NewPatternLayout("%d{yyyy-MM-dd HH:mm:ss.SSS} %5p %c %X{trace_id} %ex %m %% %n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	event := testEvent("request done", fixedTestTime())
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

func TestPatternLayout_whenUnixMillisDateUsed_shouldRenderEpochMillis(t *testing.T) {
	layout, err := NewPatternLayout("%d{UNIX_MILLIS} %p %m%n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	var buf bytes.Buffer
	if err := layout.Format(&buf, testEvent("epoch", fixedTestTime())); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if !strings.HasPrefix(buf.String(), "1787624130123 INFO epoch\n") {
		t.Fatalf("formatted line = %q", buf.String())
	}
}

func TestPatternLayout_whenExtendedConvertersUsed_shouldRenderLog4jStyleOutput(t *testing.T) {
	layout, err := NewPatternLayout("%logger{2} %map %highlight{%p} %style{%m}{red} %notEmpty{%X{trace_id}} %uuid%n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	event := testEvent("request done", fixedTestTime())
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
	if !strings.Contains(buf.String(), "trace_id=trace-1") || !strings.Contains(buf.String(), "status=200") {
		t.Fatalf("map converter output = %q", buf.String())
	}
	if !strings.Contains(buf.String(), "INFO") || !strings.Contains(buf.String(), "request done") || !strings.Contains(buf.String(), "trace-1") {
		t.Fatalf("sub pattern converter output = %q", buf.String())
	}
	uuid := strings.TrimSpace(parts[len(parts)-1])
	if len(uuid) != 36 || uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		t.Fatalf("uuid = %q, want RFC4122 text form", uuid)
	}
}

func TestPatternLayout_whenCallerTokensUsed_shouldRenderGoLocation(t *testing.T) {
	pc := callerProgramCounter(t, "TestPatternLayout_whenCallerTokensUsed")
	layout, err := NewPatternLayout("%class|%method|%file|%line|%location|%marker%n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	event := testEvent("caller", fixedTestTime())
	event.PC = pc
	event.Attrs = []slog.Attr{slog.String("marker", "SQL")}

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	line := buf.String()
	for _, want := range []string{
		"goark.dev/log",
		"TestPatternLayout_whenCallerTokensUsed",
		"lookup_pattern_test.go",
		"|SQL\n",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("caller line should contain %q, got %q", want, line)
		}
	}
	if strings.Contains(line, "|0|") {
		t.Fatalf("caller line should include a non-zero source line, got %q", line)
	}
}

func TestHandler_whenContextAttrsUsed_shouldExposeMDCToPatternLayout(t *testing.T) {
	var out bytes.Buffer
	layout, err := NewPatternLayout("%X{trace_id} %X{span_id} %m%n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewConsoleAppender(WithConsoleWriter(&out), WithConsoleLayout(layout))},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	ctx := WithContextAttrs(context.Background(),
		slog.String("trace_id", "trace-1"),
		slog.String("span_id", "span-1"),
	)
	NewLogger(handler, "goark.web").InfoContext(ctx, "request done")
	if got := out.String(); got != "trace-1 span-1 request done\n" {
		t.Fatalf("context MDC output = %q", got)
	}
}

func TestLayout_whenPrimitiveAttrsUsed_shouldRenderWithoutSemanticDrift(t *testing.T) {
	event := testEvent("primitive attrs", fixedTestTime())
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
	if !strings.Contains(pattern.String(), `primitive attrs profile="bench worker" index=42 cached=true ratio=1.25 elapsed=10ms`) {
		t.Fatalf("pattern line is wrong: %q", pattern.String())
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

func TestPatternLayout_whenAttrsHaveExplicitSeparator_shouldNotDuplicateSpace(t *testing.T) {
	event := testEvent("attrs", fixedTestTime())
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

func TestNewConfigured_whenLookupsUsedInYaml_shouldExpandBeforeBuild(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOARK_LOG_DIR", filepath.ToSlash(filepath.Join(dir, "logs")))
	t.Setenv("GOARK_LOG_PATTERN", "%p %c %X{trace_id} %m%n")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, `
appenders:
  file:
    type: file
    fileName: "${env:GOARK_LOG_DIR}/lookup.log"
    layout:
      type: pattern
      pattern: "${env:GOARK_LOG_PATTERN}"
root:
  level: info
  appenderRefs: [file]
`)
	logger, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Info("lookup works", slog.String("trace_id", "trace-1"))
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "logs", "lookup.log"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), "INFO goark trace-1 lookup works") {
		t.Fatalf("lookup config output is wrong: %q", string(content))
	}
}
