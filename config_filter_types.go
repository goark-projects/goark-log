package goarklog

type filterConfig struct {
	Type               string               `yaml:"type"`
	Level              string               `yaml:"level"`
	MinLevel           string               `yaml:"minLevel"`
	MinLevelKebab      string               `yaml:"min-level"`
	MaxLevel           string               `yaml:"maxLevel"`
	MaxLevelKebab      string               `yaml:"max-level"`
	Marker             string               `yaml:"marker"`
	Text               string               `yaml:"text"`
	Operator           string               `yaml:"operator"`
	Start              string               `yaml:"start"`
	End                string               `yaml:"end"`
	Timezone           string               `yaml:"timezone"`
	Rate               string               `yaml:"rate"`
	MaxBurst           int                  `yaml:"maxBurst"`
	MaxBurstKebab      int                  `yaml:"max-burst"`
	Field              string               `yaml:"field"`
	Key                string               `yaml:"key"`
	Value              string               `yaml:"value"`
	Values             map[string]string    `yaml:"values"`
	Thresholds         map[string]string    `yaml:"thresholds"`
	Filters            []string             `yaml:"filters"`
	FilterRefs         []string             `yaml:"filterRefs"`
	FilterRefsKebab    []string             `yaml:"filter-refs"`
	KeyValuePair       []keyValuePairConfig `yaml:"KeyValuePair"`
	KeyValuePairs      []keyValuePairConfig `yaml:"keyValuePairs"`
	KeyValuePairsKebab []keyValuePairConfig `yaml:"key-value-pairs"`
	DefaultThreshold   string               `yaml:"defaultThreshold"`
	DefaultKebab       string               `yaml:"default-threshold"`
	Pattern            string               `yaml:"pattern"`
	OnMatch            string               `yaml:"onMatch"`
	OnMatchKebab       string               `yaml:"on-match"`
	OnMismatch         string               `yaml:"onMismatch"`
	OnMismatchKebab    string               `yaml:"on-mismatch"`
}

type keyValuePairConfig struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}
