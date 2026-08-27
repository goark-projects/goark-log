package configprops

import (
	"reflect"
	"strings"
	"testing"
)

func TestRead_whenPropertiesContainSupportedSeparators_shouldParse(t *testing.T) {
	values, err := Read(strings.NewReader("#comment\na = 1\nb:2\nc three\n"))
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	want := map[string]string{"a": "1", "b": "2", "c": "three"}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("Read() = %#v, want %#v", values, want)
	}
}

func TestCollectAliases_whenNamesConfigured_shouldResolveNames(t *testing.T) {
	aliases := CollectAliases(map[string]string{
		"appender.0.name": "console",
		"logger.1.name":   "service",
	})
	if got := aliases.AppenderName("0"); got != "console" {
		t.Fatalf("AppenderName() = %q, want console", got)
	}
	if got := aliases.LoggerName("1"); got != "service" {
		t.Fatalf("LoggerName() = %q, want service", got)
	}
}

func TestSplitFilterPairKey_whenKeyValuePairUsed_shouldSplit(t *testing.T) {
	filterID, pairID, field, ok := SplitFilterPairKey("filter.f1.kv0.key")
	if !ok {
		t.Fatalf("SplitFilterPairKey() should match")
	}
	if filterID != "f1" || pairID != "kv0" || field != "key" {
		t.Fatalf("SplitFilterPairKey() = %q %q %q", filterID, pairID, field)
	}
}

func TestList_whenCommaAndSemicolonUsed_shouldTrimValues(t *testing.T) {
	got := List(" a, b ; ; c ")
	if !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("List() = %#v", got)
	}
}

func TestIntAndBool_whenInvalidValuesUsed_shouldReject(t *testing.T) {
	if _, err := Int("bad", "x"); err == nil {
		t.Fatalf("Int() should reject invalid integer")
	}
	if _, err := Bool("maybe", "x"); err == nil {
		t.Fatalf("Bool() should reject invalid boolean")
	}
}
