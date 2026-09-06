package configfile

type appenderConfig struct {
	Type                  string             `yaml:"type"`
	Target                string             `yaml:"target"`
	URL                   string             `yaml:"url"`
	Method                string             `yaml:"method"`
	Address               string             `yaml:"address"`
	Network               string             `yaml:"network"`
	Facility              string             `yaml:"facility"`
	AppName               string             `yaml:"appName"`
	AppNameKebab          string             `yaml:"app-name"`
	ConnectTimeout        string             `yaml:"connectTimeout"`
	ConnectTimeoutKebab   string             `yaml:"connect-timeout"`
	WriteTimeout          string             `yaml:"writeTimeout"`
	WriteTimeoutKebab     string             `yaml:"write-timeout"`
	FileName              string             `yaml:"fileName"`
	FileNameKebab         string             `yaml:"file-name"`
	Path                  string             `yaml:"path"`
	Layout                layoutConfig       `yaml:"layout"`
	Rolling               rollingConfig      `yaml:"rolling"`
	AppenderRefs          appenderRefs       `yaml:"appenderRefs"`
	AppenderRefsKebab     appenderRefs       `yaml:"appender-refs"`
	Refs                  appenderRefs       `yaml:"refs"`
	Primary               string             `yaml:"primary"`
	PrimaryKebab          string             `yaml:"primary-ref"`
	Failovers             []string           `yaml:"failovers"`
	FailoversKebab        []string           `yaml:"failover-refs"`
	RouteKey              string             `yaml:"routeKey"`
	RouteKeyKebab         string             `yaml:"route-key"`
	DefaultRoute          string             `yaml:"defaultRoute"`
	DefaultRouteKebab     string             `yaml:"default-route"`
	Routes                map[string]string  `yaml:"routes"`
	Rewrite               rewriteBuildConfig `yaml:"rewrite"`
	QueueSize             int                `yaml:"queueSize"`
	QueueSizeKebab        int                `yaml:"queue-size"`
	BatchSize             int                `yaml:"batchSize"`
	BatchSizeKebab        int                `yaml:"batch-size"`
	OverflowStrategy      string             `yaml:"overflowStrategy"`
	OverflowStrategyKebab string             `yaml:"overflow-strategy"`
	WaitStrategy          string             `yaml:"waitStrategy"`
	WaitStrategyKebab     string             `yaml:"wait-strategy"`
	WaitRetries           int                `yaml:"waitRetries"`
	WaitRetriesKebab      int                `yaml:"wait-retries"`
	SleepTime             string             `yaml:"sleepTime"`
	SleepTimeKebab        string             `yaml:"sleep-time"`
	Timeout               string             `yaml:"timeout"`
	BufferSize            string             `yaml:"bufferSize"`
	BufferSizeKebab       string             `yaml:"buffer-size"`
	FlushOnWrite          bool               `yaml:"flushOnWrite"`
	FlushOnWriteKebab     bool               `yaml:"flush-on-write"`
	Append                *bool              `yaml:"append"`
	CreateOnDemand        bool               `yaml:"createOnDemand"`
	CreateOnDemandKebab   bool               `yaml:"create-on-demand"`
	FilePermissions       string             `yaml:"filePermissions"`
	FilePermissionsKebab  string             `yaml:"file-permissions"`
	Filters               []string           `yaml:"filters"`
	FilterRefs            []string           `yaml:"filterRefs"`
	FilterRefsKebab       []string           `yaml:"filter-refs"`
}

type layoutConfig struct {
	Type                      string `yaml:"type"`
	Pattern                   string `yaml:"pattern"`
	EventTemplate             string `yaml:"eventTemplate"`
	EventTemplateKebab        string `yaml:"event-template"`
	EventTemplateURI          string `yaml:"eventTemplateUri"`
	EventTemplateURIKebab     string `yaml:"event-template-uri"`
	EventTemplatePath         string `yaml:"eventTemplatePath"`
	EventTemplatePathKebab    string `yaml:"event-template-path"`
	Compact                   bool   `yaml:"compact"`
	EventEOL                  bool   `yaml:"eventEol"`
	EventEOLKebab             bool   `yaml:"event-eol"`
	Complete                  bool   `yaml:"complete"`
	IncludeStacktrace         bool   `yaml:"includeStacktrace"`
	IncludeStacktraceKebab    bool   `yaml:"include-stacktrace"`
	StacktraceAsString        bool   `yaml:"stacktraceAsString"`
	StacktraceAsStringKebab   bool   `yaml:"stacktrace-as-string"`
	PropertiesAsList          bool   `yaml:"propertiesAsList"`
	PropertiesAsListKebab     bool   `yaml:"properties-as-list"`
	IncludeNullDelimiter      bool   `yaml:"includeNullDelimiter"`
	IncludeNullDelimiterKebab bool   `yaml:"include-null-delimiter"`
	DisableANSI               bool   `yaml:"disableAnsi"`
	DisableANSIKebab          bool   `yaml:"disable-ansi"`
	Header                    string `yaml:"header"`
	Footer                    string `yaml:"footer"`
}

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
