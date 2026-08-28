package integration

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"testing"
)

func TestThreadContextStackFilter_whenStackContainsValue_shouldMatch(t *testing.T) {
	filter, err := NewThreadContextStackFilter("request", WithFilterOnMatch(FilterAccept), WithFilterOnMismatch(FilterDeny))
	if err != nil {
		t.Fatalf("NewThreadContextStackFilter() error = %v", err)
	}
	event := benchmarkEvent()
	event.ContextStack = []string{"bootstrap", "request"}

	if got := filter.Decide(context.Background(), event); got != FilterAccept {
		t.Fatalf("Decide() = %v, want ACCEPT", got)
	}
}

func TestThrowableFilter_whenErrorMatchesPattern_shouldMatch(t *testing.T) {
	filter, err := NewThrowableFilter("timeout|deadline", WithFilterOnMatch(FilterAccept), WithFilterOnMismatch(FilterDeny))
	if err != nil {
		t.Fatalf("NewThrowableFilter() error = %v", err)
	}
	event := benchmarkEvent()
	event.Throwable = NewThrowable(errors.New("request timeout"))

	if got := filter.Decide(context.Background(), event); got != FilterAccept {
		t.Fatalf("Decide() = %v, want ACCEPT", got)
	}
}

func TestStructuredDataFilter_whenAttrMatches_shouldMatch(t *testing.T) {
	filter, err := NewStructuredDataFilter(map[string]string{"tenant": "goark"}, WithMapFilterOnMatch(FilterAccept), WithMapFilterOnMismatch(FilterDeny))
	if err != nil {
		t.Fatalf("NewStructuredDataFilter() error = %v", err)
	}
	event := benchmarkEvent()
	event.Attrs = append(event.Attrs, slog.String("tenant", "goark"))

	if got := filter.Decide(context.Background(), event); got != FilterAccept {
		t.Fatalf("Decide() = %v, want ACCEPT", got)
	}
}

func TestNewConfigured_whenStructuredFiltersConfigured_shouldBuild(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
filters:
  stack:
    type: ThreadContextStackFilter
    value: request
    onMatch: accept
    onMismatch: deny
  thrown:
    type: ThrowableFilter
    pattern: timeout
  structured:
    type: StructuredDataFilter
    values:
      tenant: goark
appenders:
  console:
    type: console
root:
  level: info
  appenderRefs: [console]
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfiguredHandler() error = %v", err)
	}
}
