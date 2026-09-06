package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	. "goark.dev/log/internal/testsupport"
)

func TestPatternLayout_whenExpressionConvertersUsed_shouldRenderLog4jStyleOutput(t *testing.T) {
	layout, err := NewPatternLayout(
		"%replace{%m}{secret}{***}|%enc{%m}{json}|" +
			"%equals{%X{tenant}}{root}{system}|" +
			"%equalsIgnoreCase{%X{mode}}{debug}{dbg}|" +
			"%maxLen{%logger}{8}|%repeat{%p}{2}|%sn|%throwable{short}%n",
	)
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	event := TestEvent("secret\nline", FixedTestTime())
	event.Logger = "goark.service.audit"
	event.Attrs = []slog.Attr{
		slog.String("tenant", "root"),
		slog.String("mode", "DEBUG"),
	}
	event.Throwable = NewThrowable(fmt.Errorf("outer: %w", fmt.Errorf("inner")))
	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	fields := strings.Split(strings.TrimSpace(buf.String()), "|")
	if len(fields) != 8 {
		t.Fatalf("fields = %v, want 8 from %q", fields, buf.String())
	}
	if fields[0] != "***\nline" ||
		fields[1] != `secret\nline` ||
		fields[2] != "system" ||
		fields[3] != "dbg" ||
		fields[4] != "goark.se" ||
		fields[5] != "INFOINFO" ||
		fields[7] != "outer: inner" {
		t.Fatalf("formatted fields = %#v", fields)
	}
	if _, err := strconv.ParseUint(fields[6], 10, 64); err != nil {
		t.Fatalf("sequence number = %q, want integer", fields[6])
	}
}

func TestNewConfigured_whenLookupsUsedInYaml_shouldExpandBeforeBuild(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOARK_LOG_DIR", filepath.ToSlash(filepath.Join(dir, "logs")))
	t.Setenv("GOARK_LOG_PATTERN", "%p %c %X{trace_id} %m%n")
	configPath := filepath.Join(dir, "goark-log.yml")
	WriteConfig(t, configPath, `
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
func TestPatternLayout_whenCallerTokensUsed_shouldRenderGoLocation(t *testing.T) {
	pc := callerProgramCounter(t, "TestPatternLayout_whenCallerTokensUsed")
	layout, err := NewPatternLayout("%class|%method|%file|%line|%location|%marker%n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	event := TestEvent("caller", FixedTestTime())
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
		"lookup_pattern_part2_test.go",
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

func TestPatternLayout_whenThrowableOptionsUsed_shouldHonorNoneShortAndFull(t *testing.T) {
	layout, err := NewPatternLayout("%throwable{none}|%throwable{short}|%throwable{full}%n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	event := TestEvent("throwable", FixedTestTime())
	event.Throwable = NewThrowableWithStack(errors.New("boom"))

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	fields := strings.SplitN(strings.TrimSpace(buf.String()), "|", 3)
	if len(fields) != 3 {
		t.Fatalf("fields = %v, want 3 from %q", fields, buf.String())
	}
	if fields[0] != "" || fields[1] != "boom" {
		t.Fatalf("throwable none/short = %#v, want empty and boom", fields[:2])
	}
	if !strings.Contains(fields[2], "boom") ||
		!strings.Contains(fields[2], "errors.errorString") ||
		!strings.Contains(fields[2], ".go:") {
		t.Fatalf("throwable full = %q, want type, message and stack frame", fields[2])
	}
}

func TestPatternLayout_whenRuntimeConvertersUsed_shouldRenderRelativeAndHost(t *testing.T) {
	layout, err := NewPatternLayout("%r|%relative|%host|%hostname%n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	var buf bytes.Buffer
	if err := layout.Format(&buf, TestEvent("runtime", FixedTestTime())); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	fields := strings.Split(strings.TrimSpace(buf.String()), "|")
	if len(fields) != 4 {
		t.Fatalf("fields = %v, want 4 from %q", fields, buf.String())
	}
	for _, value := range fields[:2] {
		relative, err := strconv.ParseInt(value, 10, 64)
		if err != nil || relative < 0 {
			t.Fatalf("relative value = %q, want non-negative milliseconds", value)
		}
	}
	if fields[2] == "" || fields[3] == "" || fields[2] != fields[3] {
		t.Fatalf("host fields = %q and %q, want same non-empty host", fields[2], fields[3])
	}
}

func TestHandler_whenContextAttrsUsed_shouldExposeMDCToPatternLayout(t *testing.T) {
	var out bytes.Buffer
	layout, err := NewPatternLayout("%X{trace_id} %X{span_id} %m%n")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	handler, err := NewHandler(Options{
		Appenders: []Appender{
			NewConsoleAppender(WithConsoleWriter(&out), WithConsoleLayout(layout)),
		},
		Root: RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
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

func TestPatternLayout_whenAnsiDisabled_shouldRenderPlainNestedPattern(t *testing.T) {
	layout, err := NewPatternLayoutWithOptions(
		"%style{%m}{red,bold}|%highlight{%p}%n",
		LayoutOptions{DisableANSI: true},
	)
	if err != nil {
		t.Fatalf("NewPatternLayoutWithOptions() error = %v", err)
	}
	event := TestEvent("styled", FixedTestTime())
	event.Level = slog.LevelWarn

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if buf.String() != "styled|WARN\n" {
		t.Fatalf("plain ANSI-disabled output = %q", buf.String())
	}
}

func TestLookupResolver_whenEnvAndSystemLookupsUsed_shouldResolveText(t *testing.T) {
	t.Setenv("GOARK_LOG_PROFILE", "dev")
	resolver := NewLookupResolver()
	text, err := resolver.Resolve(
		"profile=${env:GOARK_LOG_PROFILE},missing=${env:GOARK_LOG_MISSING:-local},pid=${sys:pid}",
	)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !strings.Contains(text, "profile=dev") || !strings.Contains(text, "missing=local") ||
		!strings.Contains(text, "pid=") {
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
