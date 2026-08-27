package goarklog

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

func buildThresholdFilterPlugin(config FilterBuildConfig) (Filter, error) {
	level, err := ParseLevel(config.Level)
	if err != nil {
		return nil, err
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewThresholdFilter(level, options...), nil
}

func buildLevelFilterPlugin(config FilterBuildConfig) (Filter, error) {
	level, err := ParseLevel(config.Level)
	if err != nil {
		return nil, err
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewLevelFilter(level, options...), nil
}

func buildLevelRangeFilterPlugin(config FilterBuildConfig) (Filter, error) {
	if config.MinLevel == "" || config.MaxLevel == "" {
		return nil, fmt.Errorf("goark-log: filter %q level range requires minLevel and maxLevel", config.Name)
	}
	min, err := ParseLevel(config.MinLevel)
	if err != nil {
		return nil, err
	}
	max, err := ParseLevel(config.MaxLevel)
	if err != nil {
		return nil, err
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewLevelRangeFilter(min, max, options...)
}

func buildRegexFilterPlugin(config FilterBuildConfig) (Filter, error) {
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
		options = append(options, WithRegexField(field))
	}
	if strings.TrimSpace(config.Key) != "" {
		options = append(options, WithRegexAttrKey(config.Key))
	}
	return NewRegexFilter(config.Pattern, options...)
}

func buildAttrFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewAttrFilter(config.Key, config.Value, options...)
}

func buildDenyFilterPlugin(FilterBuildConfig) (Filter, error) {
	return NewDenyFilter(), nil
}

func buildCompositeFilterPlugin(config FilterBuildConfig) (Filter, error) {
	if len(config.Filters) == 0 {
		return nil, fmt.Errorf("goark-log: filter %q composite requires filterRefs", config.Name)
	}
	return NewCompositeFilter(config.Filters...)
}

func buildMarkerFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewMarkerFilter(firstNonBlank(config.Marker, config.Value), options...)
}

func buildNoMarkerFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewNoMarkerFilter(options...), nil
}

func buildMapFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, values, err := config.mapFilterOptions()
	if err != nil {
		return nil, err
	}
	return NewMapFilter(values, options...)
}

func buildThreadContextMapFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, values, err := config.mapFilterOptions()
	if err != nil {
		return nil, err
	}
	return NewThreadContextMapFilter(values, options...)
}

func buildThreadContextStackFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewThreadContextStackFilter(firstNonBlank(config.Value, config.Text, config.Pattern), options...)
}

func buildStructuredDataFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, values, err := config.mapFilterOptions()
	if err != nil {
		return nil, err
	}
	return NewStructuredDataFilter(values, options...)
}

func buildThrowableFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewThrowableFilter(firstNonBlank(config.Pattern, config.Text, config.Value), options...)
}

func buildStringMatchFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewStringMatchFilter(firstNonBlank(config.Text, config.Value, config.Pattern), options...)
}

func buildTimeFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	start := firstNonBlank(config.Start, "00:00:00")
	end := firstNonBlank(config.End, "23:59:59.999999999")
	if strings.TrimSpace(config.Timezone) == "" {
		return NewTimeFilter(start, end, options...)
	}
	location, err := time.LoadLocation(strings.TrimSpace(config.Timezone))
	if err != nil {
		return nil, fmt.Errorf("goark-log: filter %q timezone %q is invalid", config.Name, config.Timezone)
	}
	return NewTimeFilterInLocation(start, end, location, options...)
}

func buildBurstFilterPlugin(config FilterBuildConfig) (Filter, error) {
	level, err := ParseLevel(firstNonBlank(config.Level, "warn"))
	if err != nil {
		return nil, err
	}
	rate := 10.0
	if strings.TrimSpace(config.Rate) != "" {
		parsed, err := parseFloat(config.Rate, "burst filter rate")
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
	return NewBurstFilter(level, rate, maxBurst, options...)
}

func buildDynamicThresholdFilterPlugin(config FilterBuildConfig) (Filter, error) {
	defaultLevel, err := ParseLevel(firstNonBlank(config.DefaultThreshold, config.Level, "error"))
	if err != nil {
		return nil, err
	}
	thresholds := make(map[string]slog.Level, len(config.Thresholds))
	for value, levelText := range config.Thresholds {
		level, err := ParseLevel(levelText)
		if err != nil {
			return nil, err
		}
		thresholds[value] = level
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewDynamicThresholdFilter(config.Key, defaultLevel, thresholds, options...)
}
