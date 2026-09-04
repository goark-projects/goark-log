package layout

import (
	"bytes"
	"testing"

	"goark.dev/log/internal/logevent"
)

func TestCharsetLayout_whenSingleByteCharsetUsed_shouldEncodeFormattedOutput(t *testing.T) {
	delegate, err := NewPatternLayout("%msg")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	layout, err := NewCharsetLayout(delegate, "ISO-8859-1")
	if err != nil {
		t.Fatalf("NewCharsetLayout() error = %v", err)
	}
	var output bytes.Buffer
	if err := layout.Format(&output, logevent.Event{Message: "caf\u00e9"}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if !bytes.Equal(output.Bytes(), []byte{'c', 'a', 'f', 0xe9}) {
		t.Fatalf("encoded output = %v", output.Bytes())
	}
}

func TestCharsetLayout_whenUTF8Used_shouldPreserveBytes(t *testing.T) {
	delegate, err := NewPatternLayout("%msg")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	layout, err := NewCharsetLayout(delegate, "UTF-8")
	if err != nil {
		t.Fatalf("NewCharsetLayout() error = %v", err)
	}
	var output bytes.Buffer
	if err := layout.Format(&output, logevent.Event{Message: "\u65e5\u5fd7"}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if output.String() != "\u65e5\u5fd7" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestCharsetLayout_whenCharsetUnknown_shouldReject(t *testing.T) {
	if _, err := NewCharsetLayout(TextLayout{}, "unknown-charset"); err == nil {
		t.Fatal("unknown charset should fail")
	}
}
