package goarklog

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestXMLLayout_whenEventHasSpecialChars_shouldEscapeOutput(t *testing.T) {
	event := benchmarkEvent()
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

func TestCSVLayout_whenEventHasCommaAndQuote_shouldQuoteFields(t *testing.T) {
	event := benchmarkEvent()
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

func TestJSONTemplateLayout_whenSourceAndProcessResolversUsed_shouldWriteObjects(t *testing.T) {
	layout, err := NewJSONTemplateLayout(`{
  "source": {"$resolver": "source"},
  "process": {"$resolver": "process"},
  "threadName": {"$resolver": "threadName"}
}`)
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}
	event := benchmarkEvent()
	event.PC = callerPC(0)
	event.ThreadName = "worker-1"

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v, json=%s", err, buf.String())
	}
	source, ok := decoded["source"].(map[string]any)
	if !ok || source["method"] == "" || source["file"] == "" {
		t.Fatalf("source resolver output = %#v", decoded["source"])
	}
	process, ok := decoded["process"].(map[string]any)
	if !ok || process["pid"] != float64(mustAtoi(t, processIDString)) {
		t.Fatalf("process resolver output = %#v", decoded["process"])
	}
	if decoded["threadName"] != "worker-1" {
		t.Fatalf("threadName = %#v", decoded["threadName"])
	}
}

func TestGELFLayout_whenEventFormatted_shouldWriteGELFJSON(t *testing.T) {
	event := benchmarkEvent()
	event.Attrs = append(event.Attrs, slog.String("traceId", "trace-1"))

	var buf bytes.Buffer
	if err := (GELFLayout{}).Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v, json=%s", err, buf.String())
	}
	if decoded["version"] != "1.1" || decoded["short_message"] != "service started" {
		t.Fatalf("GELF output = %#v", decoded)
	}
	if decoded["_traceId"] != "trace-1" {
		t.Fatalf("_traceId = %#v, want trace-1", decoded["_traceId"])
	}
}

func TestJSONLayout_whenOptionsEnabled_shouldWriteListPropertiesAndStackString(t *testing.T) {
	event := benchmarkEvent()
	event.Attrs = []slog.Attr{slog.String("traceId", "trace-1")}
	event.Throwable = &Throwable{
		Type:    "errors.errorString",
		Message: "query failed",
		Stack:   []string{"goark.orm.query(query.go:10)"},
	}
	layout := NewJSONLayout(LayoutOptions{
		Compact:              true,
		PropertiesAsList:     true,
		StacktraceAsString:   true,
		IncludeNullDelimiter: true,
	})

	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	output := buf.Bytes()
	if !bytes.HasSuffix(output, []byte{0}) {
		t.Fatalf("JSON output = %q, want NUL delimiter", string(output))
	}
	if bytes.Contains(output, []byte{'\n'}) {
		t.Fatalf("JSON output = %q, compact layout should not add newline", string(output))
	}
	var decoded map[string]any
	if err := json.Unmarshal(bytes.TrimSuffix(output, []byte{0}), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v, json=%s", err, string(output))
	}
	contextMap, ok := decoded["contextMap"].([]any)
	if !ok || len(contextMap) != 1 {
		t.Fatalf("contextMap = %#v, want one property item", decoded["contextMap"])
	}
	item, ok := contextMap[0].(map[string]any)
	if !ok || item["key"] != "traceId" || item["value"] != "trace-1" {
		t.Fatalf("contextMap[0] = %#v, want traceId property", contextMap[0])
	}
	thrown, ok := decoded["thrown"].(string)
	if !ok || !strings.Contains(thrown, "query.go:10") {
		t.Fatalf("thrown = %#v, want stack string", decoded["thrown"])
	}
}

func TestFileAppender_whenCompleteJSONLayoutUsed_shouldWriteValidArray(t *testing.T) {
	path := filepath.Join(t.TempDir(), "complete.json")
	appender, err := NewFileAppender(
		path,
		WithFileLayout(NewJSONLayout(LayoutOptions{Compact: true, Complete: true})),
		WithFileBufferSize(0),
	)
	if err != nil {
		t.Fatalf("NewFileAppender() error = %v", err)
	}
	event := benchmarkEvent()
	if err := appender.Append(context.Background(), event); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	event.Message = "second event"
	if err := appender.Append(context.Background(), event); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	var decoded []map[string]any
	content := readTextFile(t, path)
	if err := json.Unmarshal([]byte(content), &decoded); err != nil {
		t.Fatalf("complete JSON output is invalid: %v\n%s", err, content)
	}
	if len(decoded) != 2 || decoded[1]["msg"] != "second event" {
		t.Fatalf("decoded complete JSON = %#v, want two events", decoded)
	}
}

func TestRFC5424Layout_whenEventFormatted_shouldWriteSyslogLine(t *testing.T) {
	event := benchmarkEvent()
	event.Attrs = append(event.Attrs, slog.String("traceId", `trace"1`))

	var buf bytes.Buffer
	if err := (RFC5424Layout{}).Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	output := buf.String()
	if !strings.HasPrefix(output, "<14>1 ") {
		t.Fatalf("RFC5424 output = %q, want priority prefix", output)
	}
	if !strings.Contains(output, `[goark@32473`) || !strings.Contains(output, `traceId="trace\"1"`) {
		t.Fatalf("RFC5424 structured data = %q", output)
	}
	if !regexp.MustCompile(` service started\n$`).MatchString(output) {
		t.Fatalf("RFC5424 message = %q", output)
	}
}

func TestYAMLLayout_whenEventFormatted_shouldWriteYAMLDocument(t *testing.T) {
	event := benchmarkEvent()
	event.Attrs = append(event.Attrs, slog.String("traceId", "trace-1"))

	var buf bytes.Buffer
	if err := (YAMLLayout{}).Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var decoded map[string]any
	if err := yaml.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("YAML output invalid: %v\n%s", err, buf.String())
	}
	if decoded["message"] != "service started" {
		t.Fatalf("message = %#v, want service started", decoded["message"])
	}
	contextMap, ok := decoded["contextMap"].(map[string]any)
	if !ok || contextMap["traceId"] != "trace-1" {
		t.Fatalf("contextMap = %#v, want traceId", decoded["contextMap"])
	}
}

func TestHTMLLayout_whenEventHasSpecialChars_shouldEscapeCells(t *testing.T) {
	event := benchmarkEvent()
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

func TestBuildLayout_whenStructuredLayoutTypesUsed_shouldResolveBuiltIns(t *testing.T) {
	for _, kind := range []string{"gelf", "rfc5424", "syslog", "yaml", "html"} {
		layout, err := buildLayout(layoutConfig{Type: kind}, DefaultPluginRegistry())
		if err != nil {
			t.Fatalf("buildLayout(%q) error = %v", kind, err)
		}
		if layout == nil {
			t.Fatalf("buildLayout(%q) returned nil layout", kind)
		}
	}
}

func mustAtoi(t *testing.T, value string) int {
	t.Helper()
	parsed, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("Atoi(%q) error = %v", value, err)
	}
	return parsed
}
