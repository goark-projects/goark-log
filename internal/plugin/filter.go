package plugin

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	logfilter "goark.dev/log/internal/filter"
	configlevel "goark.dev/log/internal/level"
	"goark.dev/log/internal/textutil"
)

func (c FilterBuildConfig) filterOptions() ([]logfilter.FilterOption, error) {
	onMatch, err := parseFilterDecisionOrDefault(c.OnMatch, logfilter.FilterNeutral)
	if err != nil {
		return nil, err
	}
	onMismatch, err := parseFilterDecisionOrDefault(c.OnMismatch, logfilter.FilterDeny)
	if err != nil {
		return nil, err
	}
	return []logfilter.FilterOption{
		logfilter.WithFilterOnMatch(onMatch),
		logfilter.WithFilterOnMismatch(onMismatch),
	}, nil
}

func (c FilterBuildConfig) mapFilterOptions() ([]logfilter.MapFilterOption, map[string]string, error) {
	values := make(map[string]string, len(c.Values)+1)
	for key, value := range c.Values {
		values[key] = value
	}
	if strings.TrimSpace(c.Key) != "" {
		values[c.Key] = c.Value
	}
	operator, err := logfilter.ParseMapFilterOperator(c.Operator)
	if err != nil {
		return nil, nil, err
	}
	onMatch, err := parseFilterDecisionOrDefault(c.OnMatch, logfilter.FilterNeutral)
	if err != nil {
		return nil, nil, err
	}
	onMismatch, err := parseFilterDecisionOrDefault(c.OnMismatch, logfilter.FilterDeny)
	if err != nil {
		return nil, nil, err
	}
	return []logfilter.MapFilterOption{
		logfilter.WithMapFilterOperator(operator),
		logfilter.WithMapFilterOnMatch(onMatch),
		logfilter.WithMapFilterOnMismatch(onMismatch),
	}, values, nil
}

func (c FilterBuildConfig) regexOutcomeOptions() ([]logfilter.RegexFilterOption, error) {
	onMatch, err := parseFilterDecisionOrDefault(c.OnMatch, logfilter.FilterNeutral)
	if err != nil {
		return nil, err
	}
	onMismatch, err := parseFilterDecisionOrDefault(c.OnMismatch, logfilter.FilterDeny)
	if err != nil {
		return nil, err
	}
	return []logfilter.RegexFilterOption{
		logfilter.WithRegexOnMatch(onMatch),
		logfilter.WithRegexOnMismatch(onMismatch),
	}, nil
}

func buildThresholdFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	level, err := configlevel.ParseDefault(config.Level)
	if err != nil {
		return nil, err
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return logfilter.NewThresholdFilter(level, options...), nil
}

func buildLevelFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	level, err := configlevel.ParseDefault(config.Level)
	if err != nil {
		return nil, err
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return logfilter.NewLevelFilter(level, options...), nil
}

func buildLevelRangeFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	if config.MinLevel == "" || config.MaxLevel == "" {
		return nil, fmt.Errorf("goark-log: filter %q level range requires minLevel and maxLevel", config.Name)
	}
	min, err := configlevel.ParseDefault(config.MinLevel)
	if err != nil {
		return nil, err
	}
	max, err := configlevel.ParseDefault(config.MaxLevel)
	if err != nil {
		return nil, err
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return logfilter.NewLevelRangeFilter(min, max, options...)
}

func buildRegexFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	if strings.TrimSpace(config.Pattern) == "" {
		return nil, fmt.Errorf("goark-log: filter %q regex pattern is empty", config.Name)
	}
	options, err := config.regexOutcomeOptions()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(config.Field) != "" {
		field, err := parseRegexFilterField(config.Field)
		if err != nil {
			return nil, err
		}
		options = append(options, logfilter.WithRegexField(field))
	}
	if strings.TrimSpace(config.Key) != "" {
		options = append(options, logfilter.WithRegexAttrKey(config.Key))
	}
	return logfilter.NewRegexFilter(config.Pattern, options...)
}

func buildAttrFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return logfilter.NewAttrFilter(config.Key, config.Value, options...)
}

func buildDenyFilterPlugin(FilterBuildConfig) (logfilter.Filter, error) {
	return logfilter.NewDenyFilter(), nil
}

func buildCompositeFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	if len(config.Filters) == 0 {
		return nil, fmt.Errorf("goark-log: filter %q composite requires filterRefs", config.Name)
	}
	return logfilter.NewCompositeFilter(config.Filters...)
}

func buildMarkerFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return logfilter.NewMarkerFilter(textutil.FirstNonBlank(config.Marker, config.Value), options...)
}

func buildNoMarkerFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return logfilter.NewNoMarkerFilter(options...), nil
}

func buildMapFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	options, values, err := config.mapFilterOptions()
	if err != nil {
		return nil, err
	}
	return logfilter.NewMapFilter(values, options...)
}

func buildThreadContextMapFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	options, values, err := config.mapFilterOptions()
	if err != nil {
		return nil, err
	}
	return logfilter.NewThreadContextMapFilter(values, options...)
}

func buildThreadContextStackFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return logfilter.NewThreadContextStackFilter(textutil.FirstNonBlank(config.Value, config.Text, config.Pattern), options...)
}

func buildStructuredDataFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	options, values, err := config.mapFilterOptions()
	if err != nil {
		return nil, err
	}
	return logfilter.NewStructuredDataFilter(values, options...)
}

func buildThrowableFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return logfilter.NewThrowableFilter(textutil.FirstNonBlank(config.Pattern, config.Text, config.Value), options...)
}

func buildStringMatchFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return logfilter.NewStringMatchFilter(textutil.FirstNonBlank(config.Text, config.Value, config.Pattern), options...)
}

func buildTimeFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	start := textutil.FirstNonBlank(config.Start, "00:00:00")
	end := textutil.FirstNonBlank(config.End, "23:59:59.999999999")
	if strings.TrimSpace(config.Timezone) == "" {
		return logfilter.NewTimeFilter(start, end, options...)
	}
	location, err := time.LoadLocation(strings.TrimSpace(config.Timezone))
	if err != nil {
		return nil, fmt.Errorf("goark-log: filter %q timezone %q is invalid", config.Name, config.Timezone)
	}
	return logfilter.NewTimeFilterInLocation(start, end, location, options...)
}

func buildBurstFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	level, err := configlevel.ParseDefault(textutil.FirstNonBlank(config.Level, "warn"))
	if err != nil {
		return nil, err
	}
	rate := 10.0
	if strings.TrimSpace(config.Rate) != "" {
		parsed, err := logfilter.ParseFloat(config.Rate, "burst filter rate")
		if err != nil {
			return nil, err
		}
		rate = parsed
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	maxBurst := config.MaxBurst
	if maxBurst == 0 {
		maxBurst = int(rate * 10)
		if maxBurst <= 0 {
			maxBurst = 1
		}
	}
	return logfilter.NewBurstFilter(level, rate, maxBurst, options...)
}

func buildDynamicThresholdFilterPlugin(config FilterBuildConfig) (logfilter.Filter, error) {
	defaultLevel, err := configlevel.ParseDefault(textutil.FirstNonBlank(config.DefaultThreshold, config.Level, "error"))
	if err != nil {
		return nil, err
	}
	thresholds := make(map[string]slog.Level, len(config.Thresholds))
	for value, levelText := range config.Thresholds {
		level, err := configlevel.ParseDefault(levelText)
		if err != nil {
			return nil, err
		}
		thresholds[value] = level
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return logfilter.NewDynamicThresholdFilter(config.Key, defaultLevel, thresholds, options...)
}

func parseFilterDecisionOrDefault(value string, fallback logfilter.FilterDecision) (logfilter.FilterDecision, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	return logfilter.ParseFilterDecision(value)
}

func parseRegexFilterField(value string) (logfilter.RegexFilterField, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "message", "msg":
		return logfilter.RegexFieldMessage, nil
	case "logger", "name":
		return logfilter.RegexFieldLogger, nil
	case "attr", "attribute":
		return logfilter.RegexFieldAttr, nil
	default:
		return "", fmt.Errorf("unsupported regex filter field %q", value)
	}
}
