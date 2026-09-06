package integration

import (
	"context"
	"errors"
	"testing"

	. "goark.dev/log/internal/testsupport"
)

func TestScriptFilter_whenEvaluatorMatches_shouldReturnOnMatch(t *testing.T) {
	filter, err := NewScriptFilter(
		ScriptEvaluatorFunc(func(_ context.Context, event Event) (bool, error) {
			return event.Message == "keep", nil
		}),
		WithScriptFilterOnMatch(FilterAccept),
		WithScriptFilterOnMismatch(FilterDeny),
	)
	if err != nil {
		t.Fatalf("NewScriptFilter() error = %v", err)
	}

	keepEvent := TestEvent("keep", FixedTestTime())
	if got := filter.Decide(context.Background(), keepEvent); got != FilterAccept {
		t.Fatalf("Decide(match) = %v, want accept", got)
	}
	dropEvent := TestEvent("drop", FixedTestTime())
	if got := filter.Decide(context.Background(), dropEvent); got != FilterDeny {
		t.Fatalf("Decide(mismatch) = %v, want deny", got)
	}
}

func TestScriptFilter_whenEvaluatorFails_shouldDenyByDefault(t *testing.T) {
	filter, err := NewScriptFilter(ScriptEvaluatorFunc(func(context.Context, Event) (bool, error) {
		return false, errors.New("script failed")
	}))
	if err != nil {
		t.Fatalf("NewScriptFilter() error = %v", err)
	}

	event := TestEvent("event", FixedTestTime())
	if got := filter.Decide(context.Background(), event); got != FilterDeny {
		t.Fatalf("Decide(error) = %v, want deny", got)
	}
}
