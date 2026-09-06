package integration

import (
	"bytes"
	"context"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	. "goark.dev/log/internal/testsupport"
)

func TestFileAppender_whenCompleteJSONLayoutShared_shouldIsolateLifecycleState(t *testing.T) {
	dir := t.TempDir()
	shared := NewJSONLayout(LayoutOptions{Compact: true, Complete: true})
	first, err := NewFileAppender(
		filepath.Join(dir, "first.json"),
		WithFileLayout(shared),
		WithFileBufferSize(0),
	)
	if err != nil {
		t.Fatalf("NewFileAppender(first) error = %v", err)
	}
	second, err := NewFileAppender(
		filepath.Join(dir, "second.json"),
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
		{first, "first-1"},
		{second, "second-1"},
		{first, "first-2"},
		{second, "second-2"},
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
	AssertCompleteJSONMessages(t, filepath.Join(dir, "first.json"), "first-1", "first-2")
	AssertCompleteJSONMessages(t, filepath.Join(dir, "second.json"), "second-1", "second-2")
}

func TestJSONTemplateLayout_whenSourceAndProcessResolversUsed_shouldWriteObjects(t *testing.T) {
	layout, err := NewJSONTemplateLayout(`{
  "source": {"$resolver": "source"},
  "process": {"$resolver": "process"},
  "threadName": {"$resolver": "threadName"}
}`)
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}
	event := BenchmarkEvent()
	event.PC = CallerPC(0)
	event.ThreadName = "worker-1"

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v, json=%s", err, buf.String())
	}
	source, ok := decoded["source"].(map[string]any)
	if !ok || source["method"] == "" || source["file"] == "" {
		t.Fatalf("source resolver output = %#v", decoded["source"])
	}
	process, ok := decoded["process"].(map[string]any)
	if !ok || process["pid"] != float64(os.Getpid()) {
		t.Fatalf("process resolver output = %#v", decoded["process"])
	}
	if decoded["threadName"] != "worker-1" {
		t.Fatalf("threadName = %#v", decoded["threadName"])
	}
}

func TestJSONLayout_whenNonFiniteFloatAttrsUsed_shouldWriteValidJSON(t *testing.T) {
	event := BenchmarkEvent()
	event.Attrs = []slog.Attr{
		slog.Float64("nan", math.NaN()),
		slog.Float64("posInf", math.Inf(1)),
		slog.Float64("negInf", math.Inf(-1)),
	}

	var buf bytes.Buffer
	if err := (JSONLayout{}).Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := sonic.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON output is invalid: %v\n%s", err, buf.String())
	}
	for key, want := range map[string]string{
		"nan":    "NaN",
		"posInf": "+Inf",
		"negInf": "-Inf",
	} {
		if decoded[key] != want {
			t.Fatalf("%s = %#v, want %q", key, decoded[key], want)
		}
	}
}

func TestXMLLayout_whenEventHasSpecialChars_shouldEscapeOutput(t *testing.T) {
	event := BenchmarkEvent()
	event.Message = `service <started> & "ready"`
	event.Attrs = append(event.Attrs, slog.String("component", `api & "http"`))
	event.ContextStack = []string{"request"}

	var buf bytes.Buffer
	if err := (XMLLayout{}).Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	output := buf.String()
	for _, want := range []string{
		`<Event time="2026-08-25T10:15:30.123+08:00" level="INFO" logger="goark.bench"`,
		`<Message>service &lt;started&gt; &amp; &#34;ready&#34;</Message>`,
		`<Entry key="component">api &amp; &#34;http&#34;</Entry>`,
		`<ContextStack><Value>request</Value></ContextStack>`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("XML output missing %q: %s", want, output)
		}
	}
}

func TestRFC5424Layout_whenEventFormatted_shouldWriteSyslogLine(t *testing.T) {
	event := BenchmarkEvent()
	event.Attrs = append(event.Attrs, slog.String("traceId", `trace"1`))

	var buf bytes.Buffer
	if err := (RFC5424Layout{}).Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	output := buf.String()
	if !strings.HasPrefix(output, "<14>1 ") {
		t.Fatalf("RFC5424 output = %q, want priority prefix", output)
	}
	if !strings.Contains(output, `[goark@32473`) ||
		!strings.Contains(output, `traceId="trace\"1"`) {
		t.Fatalf("RFC5424 structured data = %q", output)
	}
	if !regexp.MustCompile(` service started\n$`).MatchString(output) {
		t.Fatalf("RFC5424 message = %q", output)
	}
}

func TestCSVLayout_whenEventHasCommaAndQuote_shouldQuoteFields(t *testing.T) {
	event := BenchmarkEvent()
	event.Message = `service, "started"`

	var buf bytes.Buffer
	if err := (CSVLayout{}).Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, `"service, ""started"""`) {
		t.Fatalf("CSV output = %q", output)
	}
}

func TestHTMLLayout_whenEventHasSpecialChars_shouldEscapeCells(t *testing.T) {
	event := BenchmarkEvent()
	event.Message = `<ready>&"ok"`

	var buf bytes.Buffer
	if err := (HTMLLayout{}).Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "&lt;ready&gt;&amp;&#34;ok&#34;") {
		t.Fatalf("HTML output = %q, want escaped message", output)
	}
}
