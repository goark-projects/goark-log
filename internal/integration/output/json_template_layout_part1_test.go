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
)

func TestFileAppender_whenCompleteJSONTemplateLayoutShared_shouldIsolateLifecycleState(
	t *testing.T,
) {
	shared, err := NewJSONTemplateLayout(
		`{"msg":{"$resolver":"message"}}`,
		WithJSONTemplateLayoutOptions(LayoutOptions{
			Compact:  true,
			Complete: true,
		}),
	)
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}
	dir := t.TempDir()
	first, err := NewFileAppender(
		filepath.Join(dir, "template-first.json"),
		WithFileLayout(shared),
		WithFileBufferSize(0),
	)
	if err != nil {
		t.Fatalf("NewFileAppender(first) error = %v", err)
	}
	second, err := NewFileAppender(
		filepath.Join(dir, "template-second.json"),
		WithFileLayout(shared),
		WithFileBufferSize(0),
	)
	if err != nil {
		t.Fatalf("NewFileAppender(second) error = %v", err)
	}
	t.Cleanup(func() {
		_ = first.Close()
		_ = second.Close()
	})

	for _, item := range []struct {
		appender *FileAppender
		message  string
	}{
		{first, "template-first-1"},
		{second, "template-second-1"},
		{first, "template-first-2"},
		{second, "template-second-2"},
	} {
		if err := item.appender.Append(
			context.Background(),
			TestEvent(item.message, FixedTestTime()),
		); err != nil {
			t.Fatalf("Append(%s) error = %v", item.message, err)
		}
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("Close(second) error = %v", err)
	}
	AssertCompleteJSONMessages(
		t,
		filepath.Join(dir, "template-first.json"),
		"template-first-1",
		"template-first-2",
	)
	AssertCompleteJSONMessages(
		t,
		filepath.Join(dir, "template-second.json"),
		"template-second-1",
		"template-second-2",
	)
}
func TestJSONTemplateLayout_whenCompleteOptionUsed_shouldWriteValidArray(t *testing.T) {
	layout, err := NewJSONTemplateLayout(
		`{"msg":{"$resolver":"message"}}`,
		WithJSONTemplateLayoutOptions(LayoutOptions{
			Compact:  true,
			Complete: true,
		}),
	)
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "template-complete.json")
	appender, err := NewFileAppender(path, WithFileLayout(layout), WithFileBufferSize(0))
	if err != nil {
		t.Fatalf("NewFileAppender() error = %v", err)
	}
	if err := appender.Append(context.Background(), TestEvent("first", FixedTestTime())); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if err := appender.Append(context.Background(), TestEvent("second", FixedTestTime())); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	var decoded []map[string]any
	content := ReadTextFile(t, path)
	if err := sonic.Unmarshal([]byte(content), &decoded); err != nil {
		t.Fatalf("complete JSON template output is invalid: %v\n%s", err, content)
	}
	if len(decoded) != 2 || decoded[0]["msg"] != "first" || decoded[1]["msg"] != "second" {
		t.Fatalf("decoded complete template = %#v, want two messages", decoded)
	}
}

func TestJSONTemplateLayout_whenCustomTemplateUsed_shouldResolveFieldsInTemplateOrder(
	t *testing.T,
) {
	layout, err := NewJSONTemplateLayout(`{
  "ts": {"$resolver": "timestamp", "format": "UNIX_MILLIS"},
  "lvl": {"$resolver": "level"},
  "trace": {"$resolver": "attr", "key": "trace_id"},
  "msg": {"$resolver": "message"},
  "static": "goark"
}`)
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}
	event := TestEvent("custom template", FixedTestTime())
	event.Attrs = append(event.Attrs, slog.String("trace_id", "trace-42"))

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	line := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(line, `{"ts":`) ||
		!strings.Contains(line, `"lvl":"INFO"`) ||
		!strings.Contains(line, `"trace":"trace-42"`) ||
		!strings.Contains(line, `"msg":"custom template"`) ||
		!strings.HasSuffix(line, `"static":"goark"}`) {
		t.Fatalf("custom template output is wrong: %s", line)
	}
}

func TestBuildLayout_whenJSONTemplateURIConfigured_shouldLoadTemplateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event-template.json")
	if err := os.WriteFile(path, []byte(`{"level":{"$resolver":"level"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	layout, err := BuildLayout(
		LayoutConfig{Type: "jsonTemplate", EventTemplateURI: path},
		DefaultPluginRegistry(),
	)
	if err != nil {
		t.Fatalf("buildLayout() error = %v", err)
	}

	var buf bytes.Buffer
	if err := layout.Format(&buf, TestEvent("from config", FixedTestTime())); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if strings.TrimSpace(buf.String()) != `{"level":"INFO"}` {
		t.Fatalf("output = %s, want config template output", buf.String())
	}
}

func TestJSONTemplateLayout_whenTemplateFileUsed_shouldLoadLocalFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event-template.json")
	if err := os.WriteFile(path, []byte(`{"msg":{"$resolver":"message"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	layout, err := NewJSONTemplateLayoutFromFile(path)
	if err != nil {
		t.Fatalf("NewJSONTemplateLayoutFromFile() error = %v", err)
	}

	var buf bytes.Buffer
	if err := layout.Format(&buf, TestEvent("from file", FixedTestTime())); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if strings.TrimSpace(buf.String()) != `{"msg":"from file"}` {
		t.Fatalf("output = %s, want file template output", buf.String())
	}
}

func TestJSONTemplateLayout_whenRemoteTemplateURIUsed_shouldReject(t *testing.T) {
	_, err := NewJSONTemplateLayoutFromFile("https://example.invalid/template.json")
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("NewJSONTemplateLayoutFromFile() error = %v, want remote URI rejection", err)
	}
}
