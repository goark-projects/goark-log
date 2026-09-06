package configfile

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
