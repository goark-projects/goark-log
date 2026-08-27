package goarklog

import (
	"strings"
)

func (c FilterBuildConfig) filterOptions() ([]FilterOption, error) {
	onMatch, err := parseFilterDecisionOrDefault(c.OnMatch, FilterNeutral)
	if err != nil {
		return nil, err
	}
	onMismatch, err := parseFilterDecisionOrDefault(c.OnMismatch, FilterDeny)
	if err != nil {
		return nil, err
	}
	return []FilterOption{
		WithFilterOnMatch(onMatch),
		WithFilterOnMismatch(onMismatch),
	}, nil
}

func (c FilterBuildConfig) mapFilterOptions() ([]MapFilterOption, map[string]string, error) {
	values := make(map[string]string, len(c.Values)+1)
	for key, value := range c.Values {
		values[key] = value
	}
	if strings.TrimSpace(c.Key) != "" {
		values[c.Key] = c.Value
	}
	operator, err := ParseMapFilterOperator(c.Operator)
	if err != nil {
		return nil, nil, err
	}
	onMatch, err := parseFilterDecisionOrDefault(c.OnMatch, FilterNeutral)
	if err != nil {
		return nil, nil, err
	}
	onMismatch, err := parseFilterDecisionOrDefault(c.OnMismatch, FilterDeny)
	if err != nil {
		return nil, nil, err
	}
	return []MapFilterOption{
		WithMapFilterOperator(operator),
		WithMapFilterOnMatch(onMatch),
		WithMapFilterOnMismatch(onMismatch),
	}, values, nil
}

func (c FilterBuildConfig) regexOutcomeOptions() ([]RegexFilterOption, error) {
	onMatch, err := parseFilterDecisionOrDefault(c.OnMatch, FilterNeutral)
	if err != nil {
		return nil, err
	}
	onMismatch, err := parseFilterDecisionOrDefault(c.OnMismatch, FilterDeny)
	if err != nil {
		return nil, err
	}
	return []RegexFilterOption{
		WithRegexOnMatch(onMatch),
		WithRegexOnMismatch(onMismatch),
	}, nil
}
