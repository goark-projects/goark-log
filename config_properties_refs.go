package goarklog

import (
	"fmt"
	"strconv"
	"strings"
)

func splitFilterPairProperty(key string) (string, string, string, bool) {
	if !strings.HasPrefix(key, "filter.") {
		return "", "", "", false
	}
	filterID, field, ok := splitPropertyID(strings.TrimPrefix(key, "filter."))
	if !ok {
		return "", "", "", false
	}
	pairID, pairField, ok := splitPropertyID(field)
	if !ok {
		return "", "", "", false
	}
	normalized := normalizeKind(pairID)
	if !strings.HasPrefix(normalized, "keyvaluepair") && !strings.HasPrefix(normalized, "kv") {
		return "", "", "", false
	}
	switch strings.ToLower(pairField) {
	case "key", "value":
		return filterID, pairID, strings.ToLower(pairField), true
	default:
		return "", "", "", false
	}
}

func splitPropertyID(key string) (string, string, bool) {
	id, field, ok := strings.Cut(key, ".")
	id = strings.TrimSpace(id)
	field = strings.TrimSpace(field)
	if !ok || id == "" || field == "" {
		return "", "", false
	}
	return id, field, true
}

func propertyAppenderRefs(value string) appenderRefs {
	values := propertyList(value)
	refs := make(appenderRefs, 0, len(values))
	for _, ref := range values {
		refs = append(refs, appenderRefConfig{Ref: ref})
	}
	return refs
}

func applyAppenderRefProperty(refs *appenderRefs, key string, value string) error {
	id, field, ok := splitPropertyID(key)
	if !ok {
		return nil
	}
	ref := findPropertyAppenderRef(refs, id)
	switch field {
	case "ref":
		ref.Ref = value
	case "level":
		ref.Level = value
	case "includeLocation", "include-location":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		ref.IncludeLocation = &parsed
	case "filters", "filterRefs", "filter-refs":
		ref.FilterRefs = propertyList(value)
	}
	return nil
}

func findPropertyAppenderRef(refs *appenderRefs, id string) *appenderRefConfig {
	for index := range *refs {
		if (*refs)[index].ID == id || ((*refs)[index].ID == "" && (*refs)[index].Ref == id) {
			if (*refs)[index].ID == "" {
				(*refs)[index].ID = id
			}
			return &(*refs)[index]
		}
	}
	*refs = append(*refs, appenderRefConfig{ID: id, Ref: id})
	return &(*refs)[len(*refs)-1]
}

func propertyList(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parsePropertyInt(value string, field string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("goark-log: properties %s value %q is invalid integer", field, value)
	}
	return parsed, nil
}

func parsePropertyBool(value string, field string) (bool, error) {
	parsed, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(value)))
	if err != nil {
		return false, fmt.Errorf("goark-log: properties %s value %q is invalid boolean", field, value)
	}
	return parsed, nil
}
