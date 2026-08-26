package goarklog

import (
	"context"
	"errors"
	"testing"
)

func TestScriptFilter_whenEvaluatorMatches_shouldReturnOnMatch(t *testing.T) {
	filter, err := NewScriptFilter(ScriptEvaluatorFunc(func(_ context.Context, event Event) (bool, error) {
		return event.Message == "keep", nil
	}), WithScriptFilterOnMatch(FilterAccept), WithScriptFilterOnMismatch(FilterDeny))
	if err != nil {
		t.Fatalf("NewScriptFilter() error = %v", err)
	}

	if got := filter.Decide(context.Background(), testEvent("keep", fixedTestTime())); got != FilterAccept {
		t.Fatalf("Decide(match) = %v, want accept", got)
	}
	if got := filter.Decide(context.Background(), testEvent("drop", fixedTestTime())); got != FilterDeny {
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

	if got := filter.Decide(context.Background(), testEvent("event", fixedTestTime())); got != FilterDeny {
		t.Fatalf("Decide(error) = %v, want deny", got)
	}
}
