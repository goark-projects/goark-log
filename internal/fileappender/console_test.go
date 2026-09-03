package fileappender

import (
	"os"
	"testing"
)

func TestNewConsoleAppender_whenWriterUnset_shouldUseStdout(t *testing.T) {
	appender := NewConsoleAppender()
	if appender.writer != os.Stdout {
		t.Fatalf("default writer = %v, want os.Stdout", appender.writer)
	}
}

func TestNewConsoleAppender_whenWriterNil_shouldUseStdout(t *testing.T) {
	appender := NewConsoleAppender(WithConsoleWriter(nil))
	if appender.writer != os.Stdout {
		t.Fatalf("nil writer fallback = %v, want os.Stdout", appender.writer)
	}
}
