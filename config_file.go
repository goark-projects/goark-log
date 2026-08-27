package goarklog

type fileConfig struct {
	Configuration     *fileConfig               `yaml:"configuration"`
	Status            string                    `yaml:"status"`
	MonitorInterval   string                    `yaml:"monitorInterval"`
	MonitorKebab      string                    `yaml:"monitor-interval"`
	Properties        map[string]string         `yaml:"properties"`
	CustomLevels      map[string]string         `yaml:"customLevels"`
	CustomLevelsKebab map[string]string         `yaml:"custom-levels"`
	Appenders         map[string]appenderConfig `yaml:"appenders"`
	Filters           map[string]filterConfig   `yaml:"filters"`
	FilterRefs        []string                  `yaml:"filterRefs"`
	FilterRefsKebab   []string                  `yaml:"filter-refs"`
	AsyncLogger       asyncLoggerConfig         `yaml:"asyncLogger"`
	AsyncLoggerKebab  asyncLoggerConfig         `yaml:"async-logger"`
	Async             asyncLoggerConfig         `yaml:"async"`
	Root              loggerConfig              `yaml:"root"`
	Loggers           map[string]loggerConfig   `yaml:"loggers"`
	Goark             struct {
		Log *fileConfig `yaml:"log"`
	} `yaml:"goark"`
}

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

type rollingConfig struct {
	FilePattern          string                `yaml:"filePattern"`
	FilePatternKebab     string                `yaml:"file-pattern"`
	MaxSize              string                `yaml:"maxSize"`
	MaxSizeKebab         string                `yaml:"max-size"`
	Interval             string                `yaml:"interval"`
	Cron                 string                `yaml:"cron"`
	CronSchedule         string                `yaml:"cronSchedule"`
	CronScheduleKebab    string                `yaml:"cron-schedule"`
	OnStartup            bool                  `yaml:"onStartup"`
	OnStartupKebab       bool                  `yaml:"on-startup"`
	MaxBackups           *int                  `yaml:"maxBackups"`
	MaxBackupsKebab      *int                  `yaml:"max-backups"`
	MaxAge               string                `yaml:"maxAge"`
	MaxAgeKebab          string                `yaml:"max-age"`
	Gzip                 bool                  `yaml:"gzip"`
	Compress             bool                  `yaml:"compress"`
	DirectWrite          bool                  `yaml:"directWrite"`
	DirectWriteKebab     bool                  `yaml:"direct-write"`
	AsyncActions         bool                  `yaml:"asyncActions"`
	AsyncActionsKebab    bool                  `yaml:"async-actions"`
	ActionQueueSize      int                   `yaml:"actionQueueSize"`
	ActionQueueSizeKebab int                   `yaml:"action-queue-size"`
	Policies             rollingPoliciesConfig `yaml:"policies"`
	Strategy             rollingStrategyConfig `yaml:"strategy"`
}

type rollingPoliciesConfig struct {
	Size                         rollingSizePolicyConfig    `yaml:"size"`
	SizeKebab                    rollingSizePolicyConfig    `yaml:"size-based-triggering-policy"`
	SizeBasedTriggeringPolicy    rollingSizePolicyConfig    `yaml:"sizeBasedTriggeringPolicy"`
	SizeBasedTriggeringPolicyXML rollingSizePolicyConfig    `yaml:"SizeBasedTriggeringPolicy"`
	Time                         rollingTimePolicyConfig    `yaml:"time"`
	TimeKebab                    rollingTimePolicyConfig    `yaml:"time-based-triggering-policy"`
	TimeBasedTriggeringPolicy    rollingTimePolicyConfig    `yaml:"timeBasedTriggeringPolicy"`
	TimeBasedTriggeringPolicyXML rollingTimePolicyConfig    `yaml:"TimeBasedTriggeringPolicy"`
	Cron                         rollingCronPolicyConfig    `yaml:"cron"`
	CronKebab                    rollingCronPolicyConfig    `yaml:"cron-triggering-policy"`
	CronTriggeringPolicy         rollingCronPolicyConfig    `yaml:"cronTriggeringPolicy"`
	CronTriggeringPolicyXML      rollingCronPolicyConfig    `yaml:"CronTriggeringPolicy"`
	Startup                      rollingStartupPolicyConfig `yaml:"startup"`
	StartupKebab                 rollingStartupPolicyConfig `yaml:"on-startup-triggering-policy"`
	OnStartupTriggeringPolicy    rollingStartupPolicyConfig `yaml:"onStartupTriggeringPolicy"`
	OnStartupTriggeringPolicyXML rollingStartupPolicyConfig `yaml:"OnStartupTriggeringPolicy"`
}

type rollingSizePolicyConfig struct {
	Size         string `yaml:"size"`
	MaxSize      string `yaml:"maxSize"`
	MaxSizeKebab string `yaml:"max-size"`
}

type rollingTimePolicyConfig struct {
	Interval string `yaml:"interval"`
	Every    string `yaml:"every"`
	Unit     string `yaml:"unit"`
	Modulate *bool  `yaml:"modulate"`
}

type rollingCronPolicyConfig struct {
	Schedule          string `yaml:"schedule"`
	Cron              string `yaml:"cron"`
	CronSchedule      string `yaml:"cronSchedule"`
	CronKebab         string `yaml:"cron-schedule"`
	EvaluateOnStartup bool   `yaml:"evaluateOnStartup"`
}

type rollingStartupPolicyConfig struct {
	Enabled *bool `yaml:"enabled"`
}

type rollingStrategyConfig struct {
	Type                 string                      `yaml:"type"`
	Max                  *int                        `yaml:"max"`
	MaxBackups           *int                        `yaml:"maxBackups"`
	MaxBackupsKebab      *int                        `yaml:"max-backups"`
	MaxAge               string                      `yaml:"maxAge"`
	MaxAgeKebab          string                      `yaml:"max-age"`
	FileIndex            string                      `yaml:"fileIndex"`
	FileIndexKebab       string                      `yaml:"file-index"`
	DirectWrite          bool                        `yaml:"directWrite"`
	DirectWriteKebab     bool                        `yaml:"direct-write"`
	AsyncActions         bool                        `yaml:"asyncActions"`
	AsyncActionsKebab    bool                        `yaml:"async-actions"`
	ActionQueueSize      int                         `yaml:"actionQueueSize"`
	ActionQueueSizeKebab int                         `yaml:"action-queue-size"`
	Compression          rollingCompressionConfig    `yaml:"compression"`
	Delete               rollingDeleteActionConfig   `yaml:"delete"`
	DeleteActions        []rollingDeleteActionConfig `yaml:"deleteActions"`
	DeleteActionsKebab   []rollingDeleteActionConfig `yaml:"delete-actions"`
}

type rollingCompressionConfig struct {
	Gzip     bool `yaml:"gzip"`
	Compress bool `yaml:"compress"`
	Async    bool `yaml:"async"`
}

type rollingDeleteActionConfig struct {
	BasePath                    string                              `yaml:"basePath"`
	BasePathKebab               string                              `yaml:"base-path"`
	MaxDepth                    *int                                `yaml:"maxDepth"`
	MaxDepthKebab               *int                                `yaml:"max-depth"`
	MaxCount                    *int                                `yaml:"maxCount"`
	MaxCountKebab               *int                                `yaml:"max-count"`
	MaxSize                     string                              `yaml:"maxSize"`
	MaxSizeKebab                string                              `yaml:"max-size"`
	Glob                        string                              `yaml:"glob"`
	Age                         string                              `yaml:"age"`
	Async                       bool                                `yaml:"async"`
	IfFileName                  rollingDeleteFileNameConfig         `yaml:"ifFileName"`
	IfFileNameKebab             rollingDeleteFileNameConfig         `yaml:"if-file-name"`
	IfLastModified              rollingDeleteLastModifiedConfig     `yaml:"ifLastModified"`
	IfLastModifiedKebab         rollingDeleteLastModifiedConfig     `yaml:"if-last-modified"`
	IfAccumulatedFileCount      rollingDeleteAccumulatedCountConfig `yaml:"ifAccumulatedFileCount"`
	IfAccumulatedFileCountKebab rollingDeleteAccumulatedCountConfig `yaml:"if-accumulated-file-count"`
	IfAccumulatedFileSize       rollingDeleteAccumulatedSizeConfig  `yaml:"ifAccumulatedFileSize"`
	IfAccumulatedFileSizeKebab  rollingDeleteAccumulatedSizeConfig  `yaml:"if-accumulated-file-size"`
}

type rollingDeleteFileNameConfig struct {
	Glob string `yaml:"glob"`
}

type rollingDeleteLastModifiedConfig struct {
	Age string `yaml:"age"`
}

type rollingDeleteAccumulatedCountConfig struct {
	Exceeds int `yaml:"exceeds"`
}

type rollingDeleteAccumulatedSizeConfig struct {
	Exceeds string `yaml:"exceeds"`
}

type asyncLoggerConfig struct {
	Enabled               *bool  `yaml:"enabled"`
	QueueSize             int    `yaml:"queueSize"`
	QueueSizeKebab        int    `yaml:"queue-size"`
	BatchSize             int    `yaml:"batchSize"`
	BatchSizeKebab        int    `yaml:"batch-size"`
	OverflowStrategy      string `yaml:"overflowStrategy"`
	OverflowStrategyKebab string `yaml:"overflow-strategy"`
	WaitStrategy          string `yaml:"waitStrategy"`
	WaitStrategyKebab     string `yaml:"wait-strategy"`
	WaitRetries           int    `yaml:"waitRetries"`
	WaitRetriesKebab      int    `yaml:"wait-retries"`
	SleepTime             string `yaml:"sleepTime"`
	SleepTimeKebab        string `yaml:"sleep-time"`
	Timeout               string `yaml:"timeout"`
	IncludeLocation       *bool  `yaml:"includeLocation"`
	IncludeLocationKebab  *bool  `yaml:"include-location"`
}

type loggerConfig struct {
	Level                string       `yaml:"level"`
	AppenderRefs         appenderRefs `yaml:"appenderRefs"`
	AppenderRefsKebab    appenderRefs `yaml:"appender-refs"`
	Refs                 appenderRefs `yaml:"refs"`
	Filters              []string     `yaml:"filters"`
	FilterRefs           []string     `yaml:"filterRefs"`
	FilterRefsKebab      []string     `yaml:"filter-refs"`
	Additivity           *bool        `yaml:"additivity"`
	IncludeLocation      *bool        `yaml:"includeLocation"`
	IncludeLocationKebab *bool        `yaml:"include-location"`
}

type appenderRefs []appenderRefConfig

type appenderRefConfig struct {
	ID                   string   `yaml:"-"`
	Ref                  string   `yaml:"ref"`
	Level                string   `yaml:"level"`
	IncludeLocation      *bool    `yaml:"includeLocation"`
	IncludeLocationKebab *bool    `yaml:"include-location"`
	Filters              []string `yaml:"filters"`
	FilterRefs           []string `yaml:"filterRefs"`
	FilterRefsKebab      []string `yaml:"filter-refs"`
}
