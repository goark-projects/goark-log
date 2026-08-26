package goarklog

import (
	"testing"
	"time"
)

func TestParseMonitorInterval(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "disabled", value: "off", want: 0},
		{name: "seconds", value: "5", want: 5 * time.Second},
		{name: "duration", value: "25ms", want: 25 * time.Millisecond},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseMonitorInterval(test.value)
			if err != nil {
				t.Fatalf("ParseMonitorInterval() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("ParseMonitorInterval() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestParseMonitorInterval_whenInvalid_shouldReject(t *testing.T) {
	if _, err := ParseMonitorInterval("-1"); err == nil {
		t.Fatalf("ParseMonitorInterval() should reject negative value")
	}
	if _, err := ParseMonitorInterval("soon"); err == nil {
		t.Fatalf("ParseMonitorInterval() should reject invalid value")
	}
}
