package configfile

import (
	"bytes"
	"errors"
	"io"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

func decodeTOMLConfig(reader io.Reader, lookups *LookupResolver) (*fileConfig, error) {
	var raw map[string]any
	decoder := toml.NewDecoder(reader)
	if err := decoder.Decode(&raw); err != nil {
		if errors.Is(err, io.EOF) {
			return &fileConfig{}, nil
		}
		return nil, err
	}
	if len(raw) == 0 {
		return finalizeDecodedConfig(fileConfig{}, lookups)
	}
	data, err := yaml.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return decodeStructuredConfig(bytes.NewReader(data), lookups)
}
