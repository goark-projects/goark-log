package integration

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
)

func TestJSONAppender_whenNativeLoggerUsesFastPath_shouldWriteJSON(t *testing.T) {
	var out bytes.Buffer
	appender := NewJSONAppender(WithJSONAppenderWriter(&out))
	handler, err := NewHandler(Options{
		Appenders: []Appender{appender},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.fast")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if err := logger.LogAttrs(context.Background(), slog.LevelInfo, "event",
		slog.String("profile", "bench"),
		slog.Int("index", 7),
	); err != nil {
		t.Fatalf("LogAttrs() error = %v", err)
	}
	var got map[string]any
	if err := sonic.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v, output = %q", err, out.String())
	}
	if got["logger"] != "goark.fast" || got["msg"] != "event" || got["profile"] != "bench" || got["index"] != float64(7) {
		t.Fatalf("JSON output = %#v", got)
	}
}

func TestJSONAppender_whenNativeLoggerUsesFixedAttrFastPath_shouldWriteJSON(t *testing.T) {
	var out bytes.Buffer
	appender := NewJSONAppender(WithJSONAppenderWriter(&out))
	handler, err := NewHandler(Options{
		Appenders: []Appender{appender},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.fixed")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if err := logger.LogAttrs3(context.Background(), slog.LevelInfo, "event",
		slog.String("profile", "bench"),
		slog.Int("index", 7),
		slog.Duration("elapsed", time.Second),
	); err != nil {
		t.Fatalf("LogAttrs3() error = %v", err)
	}
	var got map[string]any
	if err := sonic.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v, output = %q", err, out.String())
	}
	if got["logger"] != "goark.fixed" || got["msg"] != "event" || got["profile"] != "bench" || got["index"] != float64(7) || got["elapsed"] != time.Second.String() {
		t.Fatalf("JSON output = %#v", got)
	}
}

func TestJSONAppender_whenDirectAndLayoutPathsUsed_shouldProduceSameFields(t *testing.T) {
	when := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	attrs := []slog.Attr{slog.String("profile", "bench"), slog.Int("index", 7)}
	var direct bytes.Buffer
	appendJSONEvent(&direct, when, slog.LevelInfo, "goark.fast", "event", attrs)
	var layout bytes.Buffer
	if err := (JSONLayout{}).Format(&layout, Event{
		Time:    when,
		Level:   slog.LevelInfo,
		Logger:  "goark.fast",
		Message: "event",
		Attrs:   attrs,
	}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if direct.String() != layout.String() {
		t.Fatalf("direct JSON = %q, layout JSON = %q", direct.String(), layout.String())
	}
}

func TestJSONFileAppender_whenBuffered_shouldFlushAndCloseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "app.json")
	appender, err := NewJSONFileAppender(path,
		WithJSONAppenderName("jsonFile"),
		WithJSONAppenderBufferSize(64*1024),
	)
	if err != nil {
		t.Fatalf("NewJSONFileAppender() error = %v", err)
	}
	if err := appender.Append(context.Background(), Event{
		Time:    fixedTestTime(),
		Level:   slog.LevelInfo,
		Logger:  "goark.json",
		Message: "file direct",
		Attrs:   []slog.Attr{slog.String("profile", "bench")},
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := appender.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), `"logger":"goark.json"`) ||
		!strings.Contains(string(content), `"profile":"bench"`) {
		t.Fatalf("JSON file content is wrong: %q", string(content))
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() should be idempotent, got %v", err)
	}
}

func TestJSONFileAppender_whenPathIsDirectory_shouldReject(t *testing.T) {
	_, err := NewJSONFileAppender(t.TempDir())
	if err == nil {
		t.Fatalf("NewJSONFileAppender() should reject directory path")
	}
}
