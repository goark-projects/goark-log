package configfile

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
