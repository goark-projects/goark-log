package goarklog

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestMarkerFilter_whenParentMarkerMatches_shouldUseOnMatch(t *testing.T) {
	filter, err := NewMarkerFilter("SECURITY",
		WithFilterOnMatch(FilterAccept),
		WithFilterOnMismatch(FilterDeny),
	)
	if err != nil {
		t.Fatalf("NewMarkerFilter() error = %v", err)
	}
	event := testEvent("login", fixedTestTime())
	marker := NewMarker("LOGIN", NewMarker("SECURITY"))
	event.Marker = markerPointer(marker)

	if decision := filter.Decide(context.Background(), event); decision != FilterAccept {
		t.Fatalf("Decide() = %v, want %v", decision, FilterAccept)
	}
}

func TestNoMarkerFilter_whenEventHasMarker_shouldUseOnMismatch(t *testing.T) {
	filter := NewNoMarkerFilter(
		WithFilterOnMatch(FilterAccept),
		WithFilterOnMismatch(FilterDeny),
	)
	unmarked := testEvent("plain", fixedTestTime())
	marked := testEvent("marked", fixedTestTime())
	marker := NewMarker("AUDIT")
	marked.Marker = markerPointer(marker)

	if decision := filter.Decide(context.Background(), unmarked); decision != FilterAccept {
		t.Fatalf("unmarked Decide() = %v, want %v", decision, FilterAccept)
	}
	if decision := filter.Decide(context.Background(), marked); decision != FilterDeny {
		t.Fatalf("marked Decide() = %v, want %v", decision, FilterDeny)
	}
}

func TestMapFilter_whenAndAndOrConfigured_shouldMatchAttributes(t *testing.T) {
	event := testEvent("mapped", fixedTestTime())
	event.Attrs = []slog.Attr{slog.String("tenant", "core"), slog.Int("status", 200)}
	andFilter, err := NewMapFilter(map[string]string{"tenant": "core", "status": "200"},
		WithMapFilterOnMatch(FilterAccept),
		WithMapFilterOnMismatch(FilterDeny),
	)
	if err != nil {
		t.Fatalf("NewMapFilter(and) error = %v", err)
	}
	orFilter, err := NewMapFilter(map[string]string{"tenant": "edge", "status": "200"},
		WithMapFilterOperator(MapFilterOr),
		WithMapFilterOnMatch(FilterAccept),
		WithMapFilterOnMismatch(FilterDeny),
	)
	if err != nil {
		t.Fatalf("NewMapFilter(or) error = %v", err)
	}

	if decision := andFilter.Decide(context.Background(), event); decision != FilterAccept {
		t.Fatalf("and Decide() = %v, want %v", decision, FilterAccept)
	}
	if decision := orFilter.Decide(context.Background(), event); decision != FilterAccept {
		t.Fatalf("or Decide() = %v, want %v", decision, FilterAccept)
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

	if decision := filter.Decide(context.Background(), testEvent("request timeout", fixedTestTime())); decision != FilterAccept {
		t.Fatalf("matched Decide() = %v, want %v", decision, FilterAccept)
	}
	if decision := filter.Decide(context.Background(), testEvent("request done", fixedTestTime())); decision != FilterDeny {
		t.Fatalf("mismatched Decide() = %v, want %v", decision, FilterDeny)
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
		event := testEvent("night", eventTime)
		if decision := filter.Decide(context.Background(), event); decision != FilterAccept {
			t.Fatalf("Decide(%s) = %v, want %v", eventTime, decision, FilterAccept)
		}
	}
	noon := testEvent("day", time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC))
	if decision := filter.Decide(context.Background(), noon); decision != FilterDeny {
		t.Fatalf("Decide(noon) = %v, want %v", decision, FilterDeny)
	}
}

func TestTimeFilter_whenLocationConfigured_shouldCompareInConfiguredLocation(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	filter, err := NewTimeFilterInLocation("10:00", "11:00", location,
		WithFilterOnMatch(FilterAccept),
		WithFilterOnMismatch(FilterDeny),
	)
	if err != nil {
		t.Fatalf("NewTimeFilterInLocation() error = %v", err)
	}
	event := testEvent("morning", time.Date(2026, 8, 25, 2, 30, 0, 0, time.UTC))

	if decision := filter.Decide(context.Background(), event); decision != FilterAccept {
		t.Fatalf("Decide() = %v, want %v", decision, FilterAccept)
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
	event := testEvent("info", fixedTestTime())
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
	errorEvent := testEvent("error", fixedTestTime())
	errorEvent.Level = slog.LevelError
	if decision := filter.Decide(context.Background(), errorEvent); decision != FilterNeutral {
		t.Fatalf("error Decide() = %v, want %v", decision, FilterNeutral)
	}
}

func TestDynamicThresholdFilter_whenAttributeMatchesThreshold_shouldUseSpecificLevel(t *testing.T) {
	filter, err := NewDynamicThresholdFilter("tenant", slog.LevelError, map[string]slog.Level{
		"core": slog.LevelDebug,
	}, WithFilterOnMatch(FilterAccept), WithFilterOnMismatch(FilterDeny))
	if err != nil {
		t.Fatalf("NewDynamicThresholdFilter() error = %v", err)
	}
	debugEvent := testEvent("debug", fixedTestTime())
	debugEvent.Level = slog.LevelDebug
	debugEvent.Attrs = []slog.Attr{slog.String("tenant", "core")}
	infoEvent := testEvent("info", fixedTestTime())
	infoEvent.Level = slog.LevelInfo
	errorEvent := testEvent("error", fixedTestTime())
	errorEvent.Level = slog.LevelError

	if decision := filter.Decide(context.Background(), debugEvent); decision != FilterAccept {
		t.Fatalf("debug Decide() = %v, want %v", decision, FilterAccept)
	}
	if decision := filter.Decide(context.Background(), infoEvent); decision != FilterDeny {
		t.Fatalf("info Decide() = %v, want %v", decision, FilterDeny)
	}
	if decision := filter.Decide(context.Background(), errorEvent); decision != FilterAccept {
		t.Fatalf("error Decide() = %v, want %v", decision, FilterAccept)
	}
}

func TestDecodeStructuredConfig_whenLog4jStyleFiltersConfigured_shouldBuildFilters(t *testing.T) {
	config, err := decodeStructuredConfig(strings.NewReader(`
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
	filters, err := config.buildFilters(DefaultPluginRegistry())
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}
	event := testEvent("request timeout", time.Date(2026, 8, 25, 10, 30, 0, 0, time.UTC))
	marker := NewMarker("LOGIN", NewMarker("SECURITY"))
	event.Marker = markerPointer(marker)
	event.Level = slog.LevelDebug
	event.Attrs = []slog.Attr{slog.String("tenant", "core")}

	for name, filter := range filters {
		if decision := filter.Decide(context.Background(), event); decision != FilterAccept {
			t.Fatalf("filter %s Decide() = %v, want %v", name, decision, FilterAccept)
		}
	}
}

func TestDecodeStructuredConfig_whenCompositeFilterRefsConfigured_shouldApplyInOrder(t *testing.T) {
	config, err := decodeStructuredConfig(strings.NewReader(`
filters:
  allow-info:
    type: ThresholdFilter
    level: info
    onMatch: neutral
    onMismatch: deny
  deny-timeout:
    type: StringMatchFilter
    text: timeout
    onMatch: deny
    onMismatch: neutral
  chain:
    type: CompositeFilter
    filterRefs: [allow-info, deny-timeout]
`), NewLookupResolver())
	if err != nil {
		t.Fatalf("decodeStructuredConfig() error = %v", err)
	}
	filters, err := config.buildFilters(DefaultPluginRegistry())
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}
	info := testEvent("request done", fixedTestTime())
	info.Level = slog.LevelInfo
	if decision := filters["chain"].Decide(context.Background(), info); decision != FilterNeutral {
		t.Fatalf("info Decide() = %v, want neutral", decision)
	}
	timeout := testEvent("request timeout", fixedTestTime())
	timeout.Level = slog.LevelInfo
	if decision := filters["chain"].Decide(context.Background(), timeout); decision != FilterDeny {
		t.Fatalf("timeout Decide() = %v, want deny", decision)
	}
	debug := testEvent("debug", fixedTestTime())
	debug.Level = slog.LevelDebug
	if decision := filters["chain"].Decide(context.Background(), debug); decision != FilterDeny {
		t.Fatalf("debug Decide() = %v, want deny", decision)
	}
}

func TestDecodeStructuredConfig_whenCompositeFilterHasCycle_shouldReject(t *testing.T) {
	config, err := decodeStructuredConfig(strings.NewReader(`
filters:
  a:
    type: CompositeFilter
    filterRefs: [b]
  b:
    type: CompositeFilter
    filterRefs: [a]
`), NewLookupResolver())
	if err != nil {
		t.Fatalf("decodeStructuredConfig() error = %v", err)
	}
	_, err = config.buildFilters(DefaultPluginRegistry())
	if err == nil || !strings.Contains(err.Error(), "cyclic") {
		t.Fatalf("buildFilters() error = %v, want cyclic filterRefs rejection", err)
	}
}

func TestDecodePropertiesConfig_whenKeyValuePairFiltersConfigured_shouldBuildFilters(t *testing.T) {
	config, err := decodePropertiesConfig(strings.NewReader(`
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
	filters, err := config.buildFilters(DefaultPluginRegistry())
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}
	event := testEvent("mapped", fixedTestTime())
	event.Level = slog.LevelDebug
	event.Attrs = []slog.Attr{slog.String("tenant", "core")}

	for _, name := range []string{"map", "dynamic"} {
		if decision := filters[name].Decide(context.Background(), event); decision != FilterNeutral {
			t.Fatalf("filter %s Decide() = %v, want %v", name, decision, FilterNeutral)
		}
	}
}

func TestDecodePropertiesConfig_whenCompositeFilterRefsConfigured_shouldBuildFilters(t *testing.T) {
	config, err := decodePropertiesConfig(strings.NewReader(`
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
	filters, err := config.buildFilters(DefaultPluginRegistry())
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}
	event := testEvent("request timeout", fixedTestTime())
	event.Level = slog.LevelInfo
	if decision := filters["chain"].Decide(context.Background(), event); decision != FilterDeny {
		t.Fatalf("Decide() = %v, want deny", decision)
	}
}

func TestDecodeXMLConfig_whenLog4jStyleFiltersConfigured_shouldBuildFilters(t *testing.T) {
	config, err := decodeXMLConfig(strings.NewReader(`
<Configuration>
  <Filters>
    <MarkerFilter name="marker" marker="SECURITY" onMatch="ACCEPT" onMismatch="DENY"/>
    <MapFilter name="map" operator="or" onMatch="ACCEPT" onMismatch="DENY">
      <KeyValuePair key="tenant" value="core"/>
    </MapFilter>
    <DynamicThresholdFilter name="dynamic" key="tenant" defaultThreshold="ERROR" onMatch="ACCEPT" onMismatch="DENY">
      <KeyValuePair key="core" value="DEBUG"/>
    </DynamicThresholdFilter>
  </Filters>
</Configuration>
`), NewLookupResolver())
	if err != nil {
		t.Fatalf("decodeXMLConfig() error = %v", err)
	}
	filters, err := config.buildFilters(DefaultPluginRegistry())
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}
	event := testEvent("xml", fixedTestTime())
	marker := NewMarker("LOGIN", NewMarker("SECURITY"))
	event.Marker = markerPointer(marker)
	event.Level = slog.LevelDebug
	event.Attrs = []slog.Attr{slog.String("tenant", "core")}

	for _, name := range []string{"marker", "map", "dynamic"} {
		if decision := filters[name].Decide(context.Background(), event); decision != FilterAccept {
			t.Fatalf("filter %s Decide() = %v, want %v", name, decision, FilterAccept)
		}
	}
}

func TestDecodeXMLConfig_whenCompositeFilterRefsConfigured_shouldBuildFilters(t *testing.T) {
	config, err := decodeXMLConfig(strings.NewReader(`
<Configuration>
  <Filters>
    <ThresholdFilter name="allow" level="INFO" onMatch="NEUTRAL" onMismatch="DENY"/>
    <StringMatchFilter name="text" text="timeout" onMatch="DENY" onMismatch="NEUTRAL"/>
    <CompositeFilter name="chain">
      <FilterRef ref="allow"/>
      <FilterRef ref="text"/>
    </CompositeFilter>
  </Filters>
</Configuration>
`), NewLookupResolver())
	if err != nil {
		t.Fatalf("decodeXMLConfig() error = %v", err)
	}
	filters, err := config.buildFilters(DefaultPluginRegistry())
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}
	event := testEvent("request timeout", fixedTestTime())
	event.Level = slog.LevelInfo
	if decision := filters["chain"].Decide(context.Background(), event); decision != FilterDeny {
		t.Fatalf("Decide() = %v, want deny", decision)
	}
}
