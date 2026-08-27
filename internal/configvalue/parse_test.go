package configvalue

import (
	"testing"
	"time"
)

func TestByteSize_whenValidUnitsUsed_shouldParseBytes(t *testing.T) {
	cases := []struct {
		value string
		want  int64
	}{
		{value: "1.5MiB", want: 1572864},
		{value: "2kb", want: 2000},
		{value: "3GiB", want: 3 * 1024 * 1024 * 1024},
		{value: "0", want: 0},
	}
	for _, tt := range cases {
		got, err := ByteSize(tt.value)
		if err != nil {
			t.Fatalf("ByteSize(%q) error = %v", tt.value, err)
		}
		if got != tt.want {
			t.Fatalf("ByteSize(%q) = %d, want %d", tt.value, got, tt.want)
		}
	}
}

func TestByteSize_whenInvalidValueUsed_shouldReject(t *testing.T) {
	for _, value := range []string{"", "-1", "1xb", "9223372036854775808b"} {
		if _, err := ByteSize(value); err == nil {
			t.Fatalf("ByteSize(%q) should reject invalid value", value)
		}
	}
}

func TestRollingInterval_whenSupportedValuesUsed_shouldParse(t *testing.T) {
	cases := []struct {
		value string
		want  time.Duration
	}{
		{value: "daily", want: 24 * time.Hour},
		{value: "2days", want: 48 * time.Hour},
		{value: "30m", want: 30 * time.Minute},
		{value: "disabled", want: 0},
	}
	for _, tt := range cases {
		got, err := RollingInterval(tt.value)
		if err != nil {
			t.Fatalf("RollingInterval(%q) error = %v", tt.value, err)
		}
		if got != tt.want {
			t.Fatalf("RollingInterval(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestRollingMaxAge_whenSupportedValuesUsed_shouldParse(t *testing.T) {
	cases := []struct {
		value string
		want  time.Duration
	}{
		{value: "7d", want: 7 * 24 * time.Hour},
		{value: "2days", want: 48 * time.Hour},
		{value: "12h", want: 12 * time.Hour},
		{value: "off", want: 0},
	}
	for _, tt := range cases {
		got, err := RollingMaxAge(tt.value)
		if err != nil {
			t.Fatalf("RollingMaxAge(%q) error = %v", tt.value, err)
		}
		if got != tt.want {
			t.Fatalf("RollingMaxAge(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestMonitorInterval_whenSupportedValuesUsed_shouldParse(t *testing.T) {
	cases := []struct {
		value string
		want  time.Duration
	}{
		{value: "0", want: 0},
		{value: "false", want: 0},
		{value: "1.5", want: 1500 * time.Millisecond},
		{value: "20ms", want: 20 * time.Millisecond},
	}
	for _, tt := range cases {
		got, err := MonitorInterval(tt.value)
		if err != nil {
			t.Fatalf("MonitorInterval(%q) error = %v", tt.value, err)
		}
		if got != tt.want {
			t.Fatalf("MonitorInterval(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestMonitorInterval_whenInvalidValuesUsed_shouldReject(t *testing.T) {
	for _, value := range []string{"-1", "soon"} {
		if _, err := MonitorInterval(value); err == nil {
			t.Fatalf("MonitorInterval(%q) should reject invalid value", value)
		}
	}
}
