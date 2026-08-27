package configxml

import "testing"

func TestInt_whenValueBlank_shouldReturnZero(t *testing.T) {
	got, err := Int(" ", "queueSize")
	if err != nil {
		t.Fatalf("Int(blank) error = %v", err)
	}
	if got != 0 {
		t.Fatalf("Int(blank) = %d, want 0", got)
	}
}

func TestInt_whenInvalid_shouldReturnFieldError(t *testing.T) {
	_, err := Int("abc", "queueSize")
	if err == nil || err.Error() != "queueSize is invalid" {
		t.Fatalf("Int(invalid) error = %v, want field error", err)
	}
}

func TestBoolPointerStrict_whenInvalid_shouldReturnError(t *testing.T) {
	_, err := BoolPointerStrict("maybe", "includeLocation")
	if err == nil || err.Error() != "includeLocation is invalid" {
		t.Fatalf("BoolPointerStrict(invalid) error = %v, want field error", err)
	}
}

func TestBoolPointer_whenBlank_shouldReturnNil(t *testing.T) {
	if got := BoolPointer(""); got != nil {
		t.Fatalf("BoolPointer(blank) = %v, want nil", *got)
	}
}
