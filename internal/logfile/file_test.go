package logfile

import (
	"io/fs"
	"testing"
)

func TestParsePermissions_whenSupportedFormsUsed_shouldReturnMode(t *testing.T) {
	cases := []struct {
		value string
		want  fs.FileMode
	}{
		{value: "0640", want: 0o640},
		{value: "640", want: 0o640},
		{value: "rw-r-----", want: 0o640},
		{value: "---------", want: 0},
	}
	for _, tt := range cases {
		got, err := ParsePermissions(tt.value)
		if err != nil {
			t.Fatalf("ParsePermissions(%q) error = %v", tt.value, err)
		}
		if got != tt.want {
			t.Fatalf("ParsePermissions(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestParsePermissions_whenInvalidValueUsed_shouldReject(t *testing.T) {
	for _, value := range []string{"0999", "rwxr-x", "rwzr-----"} {
		if _, err := ParsePermissions(value); err == nil {
			t.Fatalf("ParsePermissions(%q) should reject invalid mode", value)
		}
	}
}
