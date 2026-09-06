package integration

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	. "goark.dev/log/internal/testsupport"
)

func TestDecodeStructuredConfig_whenLog4jStyleFiltersConfigured_shouldBuildFilters(t *testing.T) {
	config, err := DecodeStructuredConfig(strings.NewReader(`
filters:
  marker:
    type: MarkerFilter
    marker: SECURITY
    onMatch: accept
    onMismatch: deny
  mapped:
    type: MapFilter
    operator: or
    KeyValuePair:
      - key: tenant
        value: core
    onMatch: accept
    onMismatch: deny
  text:
    type: StringMatchFilter
    text: timeout
    onMatch: accept
    onMismatch: deny
  time:
    type: TimeFilter
    start: "10:00"
    end: "11:00"
    timezone: "UTC"
    onMatch: accept
    onMismatch: deny
  dynamic:
    type: DynamicThresholdFilter
    key: tenant
    defaultThreshold: error
    KeyValuePair:
      - key: core
        value: debug
    onMatch: accept
    onMismatch: deny
`), NewLookupResolver())
	if err != nil {
		t.Fatalf("decodeStructuredConfig() error = %v", err)
	}
	filters, err := config.BuildFilters(DefaultPluginRegistry())
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}
	event := TestEvent("request timeout", time.Date(2026, 8, 25, 10, 30, 0, 0, time.UTC))
	marker := NewMarker("LOGIN", NewMarker("SECURITY"))
	event.Marker = MarkerPointer(marker)
	event.Level = slog.LevelDebug
	event.Attrs = []slog.Attr{slog.String("tenant", "core")}

	for name, filter := range filters {
		if decision := filter.Decide(context.Background(), event); decision != FilterAccept {
			t.Fatalf("filter %s Decide() = %v, want %v", name, decision, FilterAccept)
		}
	}
}

func TestDecodePropertiesConfig_whenKeyValuePairFiltersConfigured_shouldBuildFilters(t *testing.T) {
	config, err := DecodePropertiesConfig(strings.NewReader(`
filter.map.type = MapFilter
filter.map.operator = and
filter.map.keyValuePair0.type = KeyValuePair
filter.map.keyValuePair0.key = tenant
filter.map.keyValuePair0.value = core
filter.dynamic.type = DynamicThresholdFilter
filter.dynamic.key = tenant
filter.dynamic.defaultThreshold = error
filter.dynamic.kv0.type = KeyValuePair
filter.dynamic.kv0.key = core
filter.dynamic.kv0.value = debug
`), NewLookupResolver())
	if err != nil {
		t.Fatalf("decodePropertiesConfig() error = %v", err)
	}
	filters, err := config.BuildFilters(DefaultPluginRegistry())
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}
	event := TestEvent("mapped", FixedTestTime())
	event.Level = slog.LevelDebug
	event.Attrs = []slog.Attr{slog.String("tenant", "core")}

	for _, name := range []string{"map", "dynamic"} {
		if decision := filters[name].Decide(context.Background(), event); decision != FilterNeutral {
			t.Fatalf("filter %s Decide() = %v, want %v", name, decision, FilterNeutral)
		}
	}
}

func TestBurstFilter_whenBurstExhausted_shouldDenyLowPriorityOnly(t *testing.T) {
	filter, err := NewBurstFilter(slog.LevelWarn, 0.000001, 2,
		WithFilterOnMatch(FilterNeutral),
		WithFilterOnMismatch(FilterDeny),
	)
	if err != nil {
		t.Fatalf("NewBurstFilter() error = %v", err)
	}
	event := TestEvent("info", FixedTestTime())
	event.Level = slog.LevelInfo

	if decision := filter.Decide(context.Background(), event); decision != FilterNeutral {
		t.Fatalf("first Decide() = %v, want %v", decision, FilterNeutral)
	}
	if decision := filter.Decide(context.Background(), event); decision != FilterNeutral {
		t.Fatalf("second Decide() = %v, want %v", decision, FilterNeutral)
	}
	if decision := filter.Decide(context.Background(), event); decision != FilterDeny {
		t.Fatalf("third Decide() = %v, want %v", decision, FilterDeny)
	}
	errorEvent := TestEvent("error", FixedTestTime())
	errorEvent.Level = slog.LevelError
	if decision := filter.Decide(context.Background(), errorEvent); decision != FilterNeutral {
		t.Fatalf("error Decide() = %v, want %v", decision, FilterNeutral)
	}
}

func TestDecodePropertiesConfig_whenCompositeFilterRefsConfigured_shouldBuildFilters(t *testing.T) {
	config, err := DecodePropertiesConfig(strings.NewReader(`
filter.allow.type = ThresholdFilter
filter.allow.level = info
filter.allow.onMismatch = deny
filter.text.type = StringMatchFilter
filter.text.text = timeout
filter.text.onMatch = deny
filter.text.onMismatch = neutral
filter.chain.type = CompositeFilter
filter.chain.filterRefs = allow,text
`), NewLookupResolver())
	if err != nil {
		t.Fatalf("decodePropertiesConfig() error = %v", err)
	}
	filters, err := config.BuildFilters(DefaultPluginRegistry())
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}
	event := TestEvent("request timeout", FixedTestTime())
	event.Level = slog.LevelInfo
	if decision := filters["chain"].Decide(context.Background(), event); decision != FilterDeny {
		t.Fatalf("Decide() = %v, want deny", decision)
	}
}

func TestTimeFilter_whenRangeCrossesMidnight_shouldMatchBothSides(t *testing.T) {
	filter, err := NewTimeFilter("22:00", "02:00",
		WithFilterOnMatch(FilterAccept),
		WithFilterOnMismatch(FilterDeny),
	)
	if err != nil {
		t.Fatalf("NewTimeFilter() error = %v", err)
	}

	for _, eventTime := range []time.Time{
		time.Date(2026, 8, 25, 23, 30, 0, 0, time.UTC),
		time.Date(2026, 8, 26, 1, 30, 0, 0, time.UTC),
	} {
		event := TestEvent("night", eventTime)
		if decision := filter.Decide(context.Background(), event); decision != FilterAccept {
			t.Fatalf("Decide(%s) = %v, want %v", eventTime, decision, FilterAccept)
		}
	}
	noon := TestEvent("day", time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC))
	if decision := filter.Decide(context.Background(), noon); decision != FilterDeny {
		t.Fatalf("Decide(noon) = %v, want %v", decision, FilterDeny)
	}
}

func TestNoMarkerFilter_whenEventHasMarker_shouldUseOnMismatch(t *testing.T) {
	filter := NewNoMarkerFilter(
		WithFilterOnMatch(FilterAccept),
		WithFilterOnMismatch(FilterDeny),
	)
	unmarked := TestEvent("plain", FixedTestTime())
	marked := TestEvent("marked", FixedTestTime())
	marker := NewMarker("AUDIT")
	marked.Marker = MarkerPointer(marker)

	if decision := filter.Decide(context.Background(), unmarked); decision != FilterAccept {
		t.Fatalf("unmarked Decide() = %v, want %v", decision, FilterAccept)
	}
	if decision := filter.Decide(context.Background(), marked); decision != FilterDeny {
		t.Fatalf("marked Decide() = %v, want %v", decision, FilterDeny)
	}
}

func TestStringMatchFilter_whenMessageContainsText_shouldUseOnMatch(t *testing.T) {
	filter, err := NewStringMatchFilter("timeout",
		WithFilterOnMatch(FilterAccept),
		WithFilterOnMismatch(FilterDeny),
	)
	if err != nil {
		t.Fatalf("NewStringMatchFilter() error = %v", err)
	}

	timeoutEvent := TestEvent("request timeout", FixedTestTime())
	if decision := filter.Decide(context.Background(), timeoutEvent); decision != FilterAccept {
		t.Fatalf("matched Decide() = %v, want %v", decision, FilterAccept)
	}
	doneEvent := TestEvent("request done", FixedTestTime())
	if decision := filter.Decide(context.Background(), doneEvent); decision != FilterDeny {
		t.Fatalf("mismatched Decide() = %v, want %v", decision, FilterDeny)
	}
}
