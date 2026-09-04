package logvalue

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/bytedance/sonic"
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
	if err := sonic.Unmarshal(buf.Bytes(), &fields); err != nil {
		t.Fatalf("AppendJSONValue() invalid JSON: %v", err)
	}
	if fields["trace_id"] != "abc" || fields["elapsed"] != "1.5s" {
		t.Fatalf("JSON fields = %+v, want trace_id and elapsed", fields)
	}
}

func TestAppendFormatted_whenUnicodeMaxWidthProvided_shouldTruncateFromStartByRune(t *testing.T) {
	var buf bytes.Buffer

	AppendFormatted(&buf, "日志abc", FieldFormat{MinWidth: 5, MaxWidth: 2})

	if got, want := buf.String(), "   bc"; got != want {
		t.Fatalf("AppendFormatted() = %q, want %q", got, want)
	}
}

func TestAppendFormatted_whenTruncateFromEndEnabled_shouldKeepPrefix(t *testing.T) {
	var buf bytes.Buffer

	AppendFormatted(&buf, "日志abc", FieldFormat{MaxWidth: 2, TruncateFromEnd: true})

	if got, want := buf.String(), "日志"; got != want {
		t.Fatalf("AppendFormatted() = %q, want %q", got, want)
	}
}

func TestAppendFormatted_whenZeroPaddingEnabled_shouldPadWithZeros(t *testing.T) {
	var buf bytes.Buffer

	AppendFormatted(&buf, "42", FieldFormat{MinWidth: 5, ZeroPad: true})

	if got, want := buf.String(), "00042"; got != want {
		t.Fatalf("AppendFormatted() = %q, want %q", got, want)
	}
}

func TestEncodePatternValue_whenCRLFMode_shouldEscapeLineBreaks(t *testing.T) {
	if got, want := EncodePatternValue("a\r\nb", "crlf"), `a\r\nb`; got != want {
		t.Fatalf("EncodePatternValue(crlf) = %q, want %q", got, want)
	}
}
