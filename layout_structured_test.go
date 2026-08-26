package goarklog

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"testing"
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

func mustAtoi(t *testing.T, value string) int {
	t.Helper()
	parsed, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("Atoi(%q) error = %v", value, err)
	}
	return parsed
}
