package layoutsupport

import (
	"testing"
	"time"
)

func TestEventTime_whenZeroUsed_shouldReturnCurrentTime(t *testing.T) {
	before := time.Now()
	got := EventTime(time.Time{})
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("EventTime() = %v, want between %v and %v", got, before, after)
	}
}

func TestEventTime_whenNonZeroUsed_shouldReturnInput(t *testing.T) {
	when := time.Unix(123, 0)
	if got := EventTime(when); !got.Equal(when) {
		t.Fatalf("EventTime() = %v, want %v", got, when)
	}
}
