package logvalue

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

func TestString_whenGroupValueProvided_shouldKeepStableOrder(t *testing.T) {
	value := slog.GroupValue(
		slog.String("region", "cn"),
		slog.Int("retry", 2),
	)

	if got, want := String(value), "region=cn,retry=2"; got != want {
		t.Fatalf("String(group) = %q, want %q", got, want)
	}
}

func TestAppendTextValue_whenStringNeedsQuoting_shouldQuote(t *testing.T) {
	var buf bytes.Buffer

	AppendTextValue(&buf, slog.StringValue("hello world"))

	if got, want := buf.String(), `"hello world"`; got != want {
		t.Fatalf("AppendTextValue() = %q, want %q", got, want)
	}
}

func TestAppendJSONValue_whenGroupProvided_shouldWriteObject(t *testing.T) {
	var buf bytes.Buffer

	AppendJSONValue(&buf, slog.GroupValue(
		slog.String("trace_id", "abc"),
		slog.Duration("elapsed", 1500*time.Millisecond),
	))

	var fields map[string]string
	if err := json.Unmarshal(buf.Bytes(), &fields); err != nil {
		t.Fatalf("AppendJSONValue() invalid JSON: %v", err)
	}
	if fields["trace_id"] != "abc" || fields["elapsed"] != "1.5s" {
		t.Fatalf("JSON fields = %+v, want trace_id and elapsed", fields)
	}
}

func TestAppendPadded_whenUnicodeMaxWidthProvided_shouldTruncateByRune(t *testing.T) {
	var buf bytes.Buffer

	AppendPadded(&buf, "日志abc", 5, 2, false)

	if got, want := buf.String(), "   日志"; got != want {
		t.Fatalf("AppendPadded() = %q, want %q", got, want)
	}
}

func TestEncodePatternValue_whenCRLFMode_shouldEscapeLineBreaks(t *testing.T) {
	if got, want := EncodePatternValue("a\r\nb", "crlf"), `a\r\nb`; got != want {
		t.Fatalf("EncodePatternValue(crlf) = %q, want %q", got, want)
	}
}
