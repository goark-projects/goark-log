package goarklog

import (
	"fmt"
	"sort"
	"strings"
)

func applyFilterProperty(config *fileConfig, key string, value string) error {
	id, field, ok := splitPropertyID(key)
	if !ok {
		return nil
	}
	filter := config.Filters[id]
	switch {
	case field == "type":
		filter.Type = value
	case field == "level":
		filter.Level = value
	case field == "minLevel" || field == "min-level":
		filter.MinLevel = value
	case field == "maxLevel" || field == "max-level":
		filter.MaxLevel = value
	case field == "marker":
		filter.Marker = value
	case field == "text":
		filter.Text = value
	case field == "operator":
		filter.Operator = value
	case field == "start":
		filter.Start = value
	case field == "end":
		filter.End = value
	case field == "timezone":
		filter.Timezone = value
	case field == "rate":
		filter.Rate = value
	case field == "maxBurst" || field == "max-burst":
		parsed, err := parsePropertyInt(value, key)
		if err != nil {
			return err
		}
		filter.MaxBurst = parsed
	case field == "field":
		filter.Field = value
	case field == "key":
		filter.Key = value
	case field == "value":
		filter.Value = value
	case strings.HasPrefix(field, "values."):
		mapKey := strings.TrimSpace(strings.TrimPrefix(field, "values."))
		if mapKey == "" {
			return fmt.Errorf("goark-log: properties filter.%s.%s has empty values key", id, field)
		}
		if filter.Values == nil {
			filter.Values = make(map[string]string)
		}
		filter.Values[mapKey] = value
	case strings.HasPrefix(field, "thresholds."):
		mapKey := strings.TrimSpace(strings.TrimPrefix(field, "thresholds."))
		if mapKey == "" {
			return fmt.Errorf("goark-log: properties filter.%s.%s has empty thresholds key", id, field)
		}
		if filter.Thresholds == nil {
			filter.Thresholds = make(map[string]string)
		}
		filter.Thresholds[mapKey] = value
	case field == "filters" || field == "filterRefs" || field == "filter-refs":
		filter.FilterRefs = propertyList(value)
	case field == "defaultThreshold" || field == "default-threshold":
		filter.DefaultThreshold = value
	case field == "pattern":
		filter.Pattern = value
	case field == "onMatch" || field == "on-match":
		filter.OnMatch = value
	case field == "onMismatch" || field == "on-mismatch":
		filter.OnMismatch = value
	}
	config.Filters[id] = filter
	return nil
}

type propertyFilterPair struct {
	key      string
	value    string
	hasKey   bool
	hasValue bool
}

func applyFilterKeyValuePairs(config *fileConfig, values map[string]string) error {
	pairsByFilter := make(map[string]map[string]propertyFilterPair)
	for key, value := range values {
		filterID, pairID, field, ok := splitFilterPairProperty(key)
		if !ok {
			continue
		}
		pairs := pairsByFilter[filterID]
		if pairs == nil {
			pairs = make(map[string]propertyFilterPair)
			pairsByFilter[filterID] = pairs
		}
		pair := pairs[pairID]
		switch field {
		case "key":
			pair.key = value
			pair.hasKey = true
		case "value":
			pair.value = value
			pair.hasValue = true
		}
		pairs[pairID] = pair
	}
	filterIDs := make([]string, 0, len(pairsByFilter))
	for filterID := range pairsByFilter {
		filterIDs = append(filterIDs, filterID)
	}
	sort.Strings(filterIDs)
	for _, filterID := range filterIDs {
		filter := config.Filters[filterID]
		pairIDs := make([]string, 0, len(pairsByFilter[filterID]))
		for pairID := range pairsByFilter[filterID] {
			pairIDs = append(pairIDs, pairID)
		}
		sort.Strings(pairIDs)
		for _, pairID := range pairIDs {
			pair := pairsByFilter[filterID][pairID]
			if !pair.hasKey && !pair.hasValue {
				continue
			}
			if !pair.hasKey || strings.TrimSpace(pair.key) == "" || !pair.hasValue {
				return fmt.Errorf("goark-log: properties filter.%s.%s requires key and value", filterID, pairID)
			}
			filter.KeyValuePair = append(filter.KeyValuePair, keyValuePairConfig{
				Key:   pair.key,
				Value: pair.value,
			})
		}
		config.Filters[filterID] = filter
	}
	return nil
}
