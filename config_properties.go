package goarklog

import (
	"io"

	"goark.dev/log/internal/configprops"
)

func decodePropertiesConfig(reader io.Reader, lookups *LookupResolver) (*fileConfig, error) {
	values, err := configprops.Read(reader)
	if err != nil {
		return nil, err
	}
	config, err := propertiesToFileConfig(values)
	if err != nil {
		return nil, err
	}
	return finalizeDecodedConfig(config, lookups)
}

func propertyAppenderRefs(value string) appenderRefs {
	values := configprops.List(value)
	refs := make(appenderRefs, 0, len(values))
	for _, ref := range values {
		refs = append(refs, appenderRefConfig{Ref: ref})
	}
	return refs
}

func applyAppenderRefProperty(refs *appenderRefs, key string, value string) error {
	id, field, ok := configprops.SplitID(key)
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
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		ref.IncludeLocation = &parsed
	case "filters", "filterRefs", "filter-refs":
		ref.FilterRefs = configprops.List(value)
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

func propertiesToFileConfig(values map[string]string) (fileConfig, error) {
	aliases := configprops.CollectAliases(values)
	config := fileConfig{
		Properties:   make(map[string]string),
		CustomLevels: make(map[string]string),
		Appenders:    make(map[string]appenderConfig),
		Filters:      make(map[string]filterConfig),
		Loggers:      make(map[string]loggerConfig),
	}
	for key, value := range values {
		if err := applyProperty(&config, aliases, key, value); err != nil {
			return fileConfig{}, err
		}
	}
	if err := applyFilterKeyValuePairs(&config, values); err != nil {
		return fileConfig{}, err
	}
	if len(config.Properties) == 0 {
		config.Properties = nil
	}
	if len(config.CustomLevels) == 0 {
		config.CustomLevels = nil
	}
	return config, nil
}
