package callsite

import "testing"

func TestBaseName_whenPathUsesDifferentSeparators_shouldReturnLastElement(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "unix", path: "/opt/goark/app.go", want: "app.go"},
		{name: "windows", path: `C:\goark\app.go`, want: "app.go"},
		{name: "plain", path: "app.go", want: "app.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BaseName(tt.path); got != tt.want {
				t.Fatalf("BaseName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestFrameFromPC_whenPCIsZero_shouldReturnEmptyFrame(t *testing.T) {
	if frame := FrameFromPC(0); !frame.IsZero() || frame.Location() != "" {
		t.Fatalf("FrameFromPC(0) = %+v, want empty frame", frame)
	}
}
