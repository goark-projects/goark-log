package textutil

import (
	"reflect"
	"testing"
	"time"
)

type namedStrings []string

func TestFirstNonBlank_whenMixedValuesUsed_shouldTrimFirstNonBlank(t *testing.T) {
	if got := FirstNonBlank(" ", "\tvalue ", "next"); got != "value" {
		t.Fatalf("FirstNonBlank() = %q, want %q", got, "value")
	}
}

func TestFirstSlice_whenNamedSliceUsed_shouldReturnSnapshot(t *testing.T) {
	input := namedStrings{"a", "b"}
	got := FirstSlice(input)
	input[0] = "changed"
	if !reflect.DeepEqual(got, namedStrings{"a", "b"}) {
		t.Fatalf("FirstSlice() = %#v, want snapshot", got)
	}
}

func TestFirstTrimmedStrings_whenValuesUsed_shouldTrimSnapshot(t *testing.T) {
	got := FirstTrimmedStrings([]string{" a ", "", "b"})
	if !reflect.DeepEqual(got, []string{"a", "", "b"}) {
		t.Fatalf("FirstTrimmedStrings() = %#v", got)
	}
}

func TestOptionalDuration_whenInvalidValueUsed_shouldReturnMinusOne(t *testing.T) {
	if got := OptionalDuration("bad"); got != -1 {
		t.Fatalf("OptionalDuration() = %v, want -1", got)
	}
	if got := OptionalDuration("2s"); got != 2*time.Second {
		t.Fatalf("OptionalDuration() = %v, want 2s", got)
	}
}

func TestNormalizeKind_whenSeparatorsUsed_shouldRemoveSeparators(t *testing.T) {
	if got := NormalizeKind(" Rolling-File "); got != "rollingfile" {
		t.Fatalf("NormalizeKind() = %q, want rollingfile", got)
	}
}

func TestSortedKeys_whenMapUsed_shouldSortKeys(t *testing.T) {
	got := SortedKeys(map[string]int{"b": 1, "a": 2})
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("SortedKeys() = %#v", got)
	}
}
