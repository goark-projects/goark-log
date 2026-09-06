package integration

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	. "goark.dev/log/internal/testsupport"
)

func TestDecodeStructuredConfig_whenCompositeFilterRefsConfigured_shouldApplyInOrder(t *testing.T) {
	config, err := DecodeStructuredConfig(strings.NewReader(`
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
	filters, err := config.BuildFilters(DefaultPluginRegistry())
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}
	info := TestEvent("request done", FixedTestTime())
	info.Level = slog.LevelInfo
	if decision := filters["chain"].Decide(context.Background(), info); decision != FilterNeutral {
		t.Fatalf("info Decide() = %v, want neutral", decision)
	}
	timeout := TestEvent("request timeout", FixedTestTime())
	timeout.Level = slog.LevelInfo
	if decision := filters["chain"].Decide(context.Background(), timeout); decision != FilterDeny {
		t.Fatalf("timeout Decide() = %v, want deny", decision)
	}
	debug := TestEvent("debug", FixedTestTime())
	debug.Level = slog.LevelDebug
	if decision := filters["chain"].Decide(context.Background(), debug); decision != FilterDeny {
		t.Fatalf("debug Decide() = %v, want deny", decision)
	}
}

func TestDecodeXMLConfig_whenLog4jStyleFiltersConfigured_shouldBuildFilters(t *testing.T) {
	config, err := DecodeXMLConfig(strings.NewReader(`
<Configuration>
  <Filters>
    <MarkerFilter name="marker" marker="SECURITY" onMatch="ACCEPT" onMismatch="DENY"/>
    <MapFilter name="map" operator="or" onMatch="ACCEPT" onMismatch="DENY">
      <KeyValuePair key="tenant" value="core"/>
    </MapFilter>
    <DynamicThresholdFilter name="dynamic" key="tenant"
        defaultThreshold="ERROR" onMatch="ACCEPT" onMismatch="DENY">
      <KeyValuePair key="core" value="DEBUG"/>
    </DynamicThresholdFilter>
  </Filters>
</Configuration>
`), NewLookupResolver())
	if err != nil {
		t.Fatalf("decodeXMLConfig() error = %v", err)
	}
	filters, err := config.BuildFilters(DefaultPluginRegistry())
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}
	event := TestEvent("xml", FixedTestTime())
	marker := NewMarker("LOGIN", NewMarker("SECURITY"))
	event.Marker = MarkerPointer(marker)
	event.Level = slog.LevelDebug
	event.Attrs = []slog.Attr{slog.String("tenant", "core")}

	for _, name := range []string{"marker", "map", "dynamic"} {
		if decision := filters[name].Decide(context.Background(), event); decision != FilterAccept {
			t.Fatalf("filter %s Decide() = %v, want %v", name, decision, FilterAccept)
		}
	}
}

func TestMapFilter_whenAndAndOrConfigured_shouldMatchAttributes(t *testing.T) {
	event := TestEvent("mapped", FixedTestTime())
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

func TestDynamicThresholdFilter_whenAttributeMatchesThreshold_shouldUseSpecificLevel(t *testing.T) {
	filter, err := NewDynamicThresholdFilter("tenant", slog.LevelError, map[string]slog.Level{
		"core": slog.LevelDebug,
	}, WithFilterOnMatch(FilterAccept), WithFilterOnMismatch(FilterDeny))
	if err != nil {
		t.Fatalf("NewDynamicThresholdFilter() error = %v", err)
	}
	debugEvent := TestEvent("debug", FixedTestTime())
	debugEvent.Level = slog.LevelDebug
	debugEvent.Attrs = []slog.Attr{slog.String("tenant", "core")}
	infoEvent := TestEvent("info", FixedTestTime())
	infoEvent.Level = slog.LevelInfo
	errorEvent := TestEvent("error", FixedTestTime())
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

func TestDecodeXMLConfig_whenCompositeFilterRefsConfigured_shouldBuildFilters(t *testing.T) {
	config, err := DecodeXMLConfig(strings.NewReader(`
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
func TestDecodeStructuredConfig_whenCompositeFilterHasCycle_shouldReject(t *testing.T) {
	config, err := DecodeStructuredConfig(strings.NewReader(`
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
	_, err = config.BuildFilters(DefaultPluginRegistry())
	if err == nil || !strings.Contains(err.Error(), "cyclic") {
		t.Fatalf("buildFilters() error = %v, want cyclic filterRefs rejection", err)
	}
}

func TestMarkerFilter_whenParentMarkerMatches_shouldUseOnMatch(t *testing.T) {
	filter, err := NewMarkerFilter("SECURITY",
		WithFilterOnMatch(FilterAccept),
		WithFilterOnMismatch(FilterDeny),
	)
	if err != nil {
		t.Fatalf("NewMarkerFilter() error = %v", err)
	}
	event := TestEvent("login", FixedTestTime())
	marker := NewMarker("LOGIN", NewMarker("SECURITY"))
	event.Marker = MarkerPointer(marker)

	if decision := filter.Decide(context.Background(), event); decision != FilterAccept {
		t.Fatalf("Decide() = %v, want %v", decision, FilterAccept)
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
	event := TestEvent("morning", time.Date(2026, 8, 25, 2, 30, 0, 0, time.UTC))

	if decision := filter.Decide(context.Background(), event); decision != FilterAccept {
		t.Fatalf("Decide() = %v, want %v", decision, FilterAccept)
	}
}
