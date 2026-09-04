package integration

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"time"

	goarklog "goark.dev/log"
	"goark.dev/log/internal/configfile"
	internallayout "goark.dev/log/internal/layout"
	internallogevent "goark.dev/log/internal/logevent"
	internalnativelogger "goark.dev/log/internal/nativelogger"
	internalrollingfile "goark.dev/log/internal/rollingfile"
	internalrouter "goark.dev/log/internal/router"
)

const (
	EnvConfigPath = goarklog.EnvConfigPath

	ConfigSourceExplicit = goarklog.ConfigSourceExplicit
	ConfigSourceEnv      = goarklog.ConfigSourceEnv
	ConfigSourceBoot     = goarklog.ConfigSourceBoot
	ConfigSourceFile     = goarklog.ConfigSourceFile
	ConfigSourceDefault  = goarklog.ConfigSourceDefault

	LevelAll   = goarklog.LevelAll
	LevelTrace = goarklog.LevelTrace
	LevelFatal = goarklog.LevelFatal
	LevelOff   = goarklog.LevelOff

	FilterNeutral = goarklog.FilterNeutral
	FilterAccept  = goarklog.FilterAccept
	FilterDeny    = goarklog.FilterDeny

	MapFilterAnd = goarklog.MapFilterAnd
	MapFilterOr  = goarklog.MapFilterOr

	AsyncOverflowBlock        = goarklog.AsyncOverflowBlock
	AsyncOverflowDrop         = goarklog.AsyncOverflowDrop
	AsyncOverflowDropDebug    = goarklog.AsyncOverflowDropDebug
	AsyncOverflowSyncFallback = goarklog.AsyncOverflowSyncFallback

	AsyncWaitBlock = goarklog.AsyncWaitBlock
	AsyncWaitSleep = goarklog.AsyncWaitSleep
	AsyncWaitYield = goarklog.AsyncWaitYield
	AsyncWaitSpin  = goarklog.AsyncWaitSpin

	RollingFileIndexNoMax = goarklog.RollingFileIndexNoMax
	RollingFileIndexMax   = goarklog.RollingFileIndexMax
	RollingFileIndexMin   = goarklog.RollingFileIndexMin

	StructuredDataIDAttrKey   = goarklog.StructuredDataIDAttrKey
	StructuredDataTypeAttrKey = goarklog.StructuredDataTypeAttrKey

	defaultThreadName = internallogevent.DefaultThreadName
)

type (
	Appender              = goarklog.Appender
	AppenderRef           = goarklog.AppenderRef
	AsyncAppender         = goarklog.AsyncAppender
	AsyncErrorHandlerFunc = goarklog.AsyncErrorHandlerFunc
	AsyncLoggerOptions    = goarklog.AsyncLoggerOptions
	AsyncWaitOptions      = goarklog.AsyncWaitOptions
	Event                 = goarklog.Event
	FileAppender          = goarklog.FileAppender
	Filter                = goarklog.Filter
	FilterDecision        = goarklog.FilterDecision
	FilterFunc            = goarklog.FilterFunc
	Layout                = goarklog.Layout
	LayoutOptions         = goarklog.LayoutOptions
	Options               = goarklog.Options
	PatternLayout         = goarklog.PatternLayout
	PluginRegistry        = goarklog.PluginRegistry
	RollingDeleteAction   = goarklog.RollingDeleteAction
	RollingFileAppender   = goarklog.RollingFileAppender
	RollingFileIndexMode  = goarklog.RollingFileIndexMode
	RollingFileOption     = goarklog.RollingFileOption
	RootLogger            = goarklog.RootLogger
	LoggerRule            = goarklog.LoggerRule
	LoggerConfiguration   = goarklog.LoggerConfiguration
	Marker                = goarklog.Marker
	Message               = goarklog.Message
	MessageFactoryFunc    = goarklog.MessageFactoryFunc
	TextLayout            = goarklog.TextLayout
	CSVLayout             = goarklog.CSVLayout
	GELFLayout            = goarklog.GELFLayout
	HTMLLayout            = goarklog.HTMLLayout
	JSONLayout            = goarklog.JSONLayout
	RFC5424Layout         = goarklog.RFC5424Layout
	Throwable             = goarklog.Throwable
	XMLLayout             = goarklog.XMLLayout
	YAMLLayout            = goarklog.YAMLLayout

	AppenderBuildConfig             = goarklog.AppenderBuildConfig
	ConfigLoadOption                = goarklog.ConfigLoadOption
	ConfigResult                    = goarklog.ConfigResult
	ConfigSource                    = goarklog.ConfigSource
	FilterBuildConfig               = goarklog.FilterBuildConfig
	JSONTemplateResolver            = goarklog.JSONTemplateResolver
	JSONTemplateResolverBuildConfig = goarklog.JSONTemplateResolverBuildConfig
	LayoutBuildConfig               = goarklog.LayoutBuildConfig
	PluginRegistrarFunc             = goarklog.PluginRegistrarFunc
	PropertyMap                     = goarklog.PropertyMap
	ScriptEvaluatorFunc             = goarklog.ScriptEvaluatorFunc
	FilteredAppender                = goarklog.FilteredAppender

	layoutConfig = configfile.LayoutConfig
)

type fileConfig struct {
	*configfile.Config
}

var (
	ConfigureDefault              = goarklog.ConfigureDefault
	ContextAttrs                  = goarklog.ContextAttrs
	DefaultOptions                = goarklog.DefaultOptions
	DefaultPluginRegistry         = goarklog.DefaultPluginRegistry
	LevelName                     = goarklog.LevelName
	LoadOptions                   = goarklog.LoadOptions
	NewAppenderRef                = goarklog.NewAppenderRef
	NewAsyncAppender              = goarklog.NewAsyncAppender
	NewAttrFilter                 = goarklog.NewAttrFilter
	NewBurstFilter                = goarklog.NewBurstFilter
	NewConsoleAppender            = goarklog.NewConsoleAppender
	NewConfigReloader             = goarklog.NewConfigReloader
	NewConfiguredLoggerContext    = goarklog.NewConfiguredLoggerContext
	NewDynamicThresholdFilter     = goarklog.NewDynamicThresholdFilter
	NewFileAppender               = goarklog.NewFileAppender
	NewFilteredAppender           = goarklog.NewFilteredAppender
	NewHandler                    = goarklog.NewHandler
	NewJSONAppender               = goarklog.NewJSONAppender
	NewJSONFileAppender           = goarklog.NewJSONFileAppender
	NewJSONLayout                 = goarklog.NewJSONLayout
	NewJSONTemplateLayout         = goarklog.NewJSONTemplateLayout
	NewJSONTemplateLayoutFromFile = goarklog.NewJSONTemplateLayoutFromFile
	NewLevelRegistry              = goarklog.NewLevelRegistry
	NewLogger                     = goarklog.NewLogger
	NewLoggerContext              = goarklog.NewLoggerContext
	NewLookupResolver             = goarklog.NewLookupResolver
	NewMapFilter                  = goarklog.NewMapFilter
	NewMapMessage                 = goarklog.NewMapMessage
	NewMarker                     = goarklog.NewMarker
	NewMarkerFilter               = goarklog.NewMarkerFilter
	NewNativeLogger               = goarklog.NewNativeLogger
	NewNoMarkerFilter             = goarklog.NewNoMarkerFilter
	NewParameterizedMessage       = goarklog.NewParameterizedMessage
	NewPatternLayout              = goarklog.NewPatternLayout
	NewPatternLayoutWithOptions   = goarklog.NewPatternLayoutWithOptions
	NewPluginRegistry             = goarklog.NewPluginRegistry
	NewPluginSet                  = goarklog.NewPluginSet
	NewRegexFilter                = goarklog.NewRegexFilter
	NewRollingFileAppender        = goarklog.NewRollingFileAppender
	NewScriptFilter               = goarklog.NewScriptFilter
	NewSimpleMessage              = goarklog.NewSimpleMessage
	NewStatusLogger               = goarklog.NewStatusLogger
	NewStringMatchFilter          = goarklog.NewStringMatchFilter
	NewStructuredDataFilter       = goarklog.NewStructuredDataFilter
	NewStructuredDataMessage      = goarklog.NewStructuredDataMessage
	NewThreadContextStackFilter   = goarklog.NewThreadContextStackFilter
	NewThresholdFilter            = goarklog.NewThresholdFilter
	NewThrowable                  = goarklog.NewThrowable
	NewThrowableFilter            = goarklog.NewThrowableFilter
	NewThrowableWithStack         = goarklog.NewThrowableWithStack
	NewTimeFilter                 = goarklog.NewTimeFilter
	NewTimeFilterInLocation       = goarklog.NewTimeFilterInLocation
	ThrowableAttr                 = goarklog.ThrowableAttr

	NewConfigured                    = goarklog.NewConfigured
	NewConfiguredHandler             = goarklog.NewConfiguredHandler
	ParseByteSize                    = goarklog.ParseByteSize
	ParseLevel                       = goarklog.ParseLevel
	ParseMonitorInterval             = goarklog.ParseMonitorInterval
	ParseRollingInterval             = goarklog.ParseRollingInterval
	RegisterLayout                   = goarklog.RegisterLayout
	RegisterLevel                    = goarklog.RegisterLevel
	WithAppenderRefFilters           = goarklog.WithAppenderRefFilters
	WithAppenderRefLevel             = goarklog.WithAppenderRefLevel
	WithAppenderRefLocation          = goarklog.WithAppenderRefLocation
	WithAsyncBatchSize               = goarklog.WithAsyncBatchSize
	WithAsyncCloseAppenders          = goarklog.WithAsyncCloseAppenders
	WithAsyncErrorHandler            = goarklog.WithAsyncErrorHandler
	WithAsyncOverflowStrategy        = goarklog.WithAsyncOverflowStrategy
	WithAsyncQueueSize               = goarklog.WithAsyncQueueSize
	WithAsyncWaitOptions             = goarklog.WithAsyncWaitOptions
	WithAsyncWaitStrategy            = goarklog.WithAsyncWaitStrategy
	WithBootPropertyResolver         = goarklog.WithBootPropertyResolver
	WithConfigEnvKey                 = goarklog.WithConfigEnvKey
	WithConfigLookups                = goarklog.WithConfigLookups
	WithConfigPath                   = goarklog.WithConfigPath
	WithConfigWorkingDir             = goarklog.WithConfigWorkingDir
	WithConsoleLayout                = goarklog.WithConsoleLayout
	WithConsoleName                  = goarklog.WithConsoleName
	WithConsoleWriter                = goarklog.WithConsoleWriter
	WithFileBufferSize               = goarklog.WithFileBufferSize
	WithFileCreateOnDemand           = goarklog.WithFileCreateOnDemand
	WithFileFlushOnWrite             = goarklog.WithFileFlushOnWrite
	WithFileLayout                   = goarklog.WithFileLayout
	WithFilterOnMatch                = goarklog.WithFilterOnMatch
	WithFilterOnMismatch             = goarklog.WithFilterOnMismatch
	WithJSONAppenderBufferSize       = goarklog.WithJSONAppenderBufferSize
	WithJSONAppenderName             = goarklog.WithJSONAppenderName
	WithJSONAppenderWriter           = goarklog.WithJSONAppenderWriter
	WithJSONTemplateLayoutOptions    = goarklog.WithJSONTemplateLayoutOptions
	WithJSONTemplateResolverRegistry = goarklog.WithJSONTemplateResolverRegistry
	WithLoggerCaller                 = goarklog.WithLoggerCaller
	WithLoggerContextStatus          = goarklog.WithLoggerContextStatus
	WithLoggerMessageFactory         = goarklog.WithLoggerMessageFactory
	WithMapFilterOnMatch             = goarklog.WithMapFilterOnMatch
	WithMapFilterOnMismatch          = goarklog.WithMapFilterOnMismatch
	WithMapFilterOperator            = goarklog.WithMapFilterOperator
	WithPluginAppender               = goarklog.WithPluginAppender
	WithPluginJSONTemplateResolver   = goarklog.WithPluginJSONTemplateResolver
	WithPluginLayout                 = goarklog.WithPluginLayout
	WithPluginLookup                 = goarklog.WithPluginLookup
	WithPluginRegistry               = goarklog.WithPluginRegistry
	WithRegexOnMatch                 = goarklog.WithRegexOnMatch
	WithRegexOnMismatch              = goarklog.WithRegexOnMismatch
	WithRolloverOnStartup            = goarklog.WithRolloverOnStartup
	WithRollingActionQueueSize       = goarklog.WithRollingActionQueueSize
	WithRollingAsyncActions          = goarklog.WithRollingAsyncActions
	WithRollingCronSchedule          = goarklog.WithRollingCronSchedule
	WithRollingDeleteActions         = goarklog.WithRollingDeleteActions
	WithRollingDirectWrite           = goarklog.WithRollingDirectWrite
	WithRollingFileAppend            = goarklog.WithRollingFileAppend
	WithRollingFileBufferSize        = goarklog.WithRollingFileBufferSize
	WithRollingFileCreateOnDemand    = goarklog.WithRollingFileCreateOnDemand
	WithRollingFileIndexMode         = goarklog.WithRollingFileIndexMode
	WithRollingFileLayout            = goarklog.WithRollingFileLayout
	WithRollingFilePattern           = goarklog.WithRollingFilePattern
	WithRollingGzip                  = goarklog.WithRollingGzip
	WithRollingInterval              = goarklog.WithRollingInterval
	WithRollingMaxBackups            = goarklog.WithRollingMaxBackups
	WithRollingMaxSize               = goarklog.WithRollingMaxSize
	WithRollingTotalSizeCap          = goarklog.WithRollingTotalSizeCap
	WithRollingCleanHistoryOnStart   = goarklog.WithRollingCleanHistoryOnStart
	WithRoutingAttrKey               = goarklog.WithRoutingAttrKey
	WithRoutingDefault               = goarklog.WithRoutingDefault
	WithRoutingKeyFunc               = goarklog.WithRoutingKeyFunc
	WithScriptFilterOnMatch          = goarklog.WithScriptFilterOnMatch
	WithScriptFilterOnMismatch       = goarklog.WithScriptFilterOnMismatch
	WithStatusBufferSize             = goarklog.WithStatusBufferSize
	WithStatusLevel                  = goarklog.WithStatusLevel
	WithStatusWriter                 = goarklog.WithStatusWriter
	WithContextAttrs                 = goarklog.WithContextAttrs
	WithContextStack                 = goarklog.WithContextStack
	WithMarker                       = goarklog.WithMarker
	WithPluginFilter                 = goarklog.WithPluginFilter
	WithThreadName                   = goarklog.WithThreadName
	NewFailoverAppender              = goarklog.NewFailoverAppender
	NewRoutingAppender               = goarklog.NewRoutingAppender
	NewRewriteAppender               = goarklog.NewRewriteAppender
)

func buildLayout(config layoutConfig, registry *PluginRegistry) (Layout, error) {
	return configfile.BuildLayout(config, registry)
}

func appendJSONEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs []slog.Attr) {
	internallayout.AppendJSONEvent(buf, when, level, logger, message, attrs)
}

func newEvent(ctx context.Context, logger string, handlerAttrs []slog.Attr, groups []string, record slog.Record) Event {
	return internallogevent.New(ctx, logger, handlerAttrs, groups, record)
}

func decodeStructuredConfig(reader io.Reader, lookups *goarklog.LookupResolver) (*fileConfig, error) {
	config, err := configfile.DecodeStructured(reader, lookups)
	if err != nil {
		return nil, err
	}
	return &fileConfig{Config: config}, nil
}

func decodeTOMLConfig(reader io.Reader, lookups *goarklog.LookupResolver) (*fileConfig, error) {
	config, err := configfile.DecodeTOML(reader, lookups)
	if err != nil {
		return nil, err
	}
	return &fileConfig{Config: config}, nil
}

func decodeXMLConfig(reader io.Reader, lookups *goarklog.LookupResolver) (*fileConfig, error) {
	config, err := configfile.DecodeXML(reader, lookups)
	if err != nil {
		return nil, err
	}
	return &fileConfig{Config: config}, nil
}

func decodePropertiesConfig(reader io.Reader, lookups *goarklog.LookupResolver) (*fileConfig, error) {
	config, err := configfile.DecodeProperties(reader, lookups)
	if err != nil {
		return nil, err
	}
	return &fileConfig{Config: config}, nil
}

func (c *fileConfig) buildFilters(registry *PluginRegistry) (map[string]Filter, error) {
	if c == nil || c.Config == nil {
		return nil, nil
	}
	return c.Config.BuildFilters(registry)
}

func callerPC(skip int) uintptr {
	return internalnativelogger.CallerPC(skip)
}

func withRollingClock(clock func() time.Time) RollingFileOption {
	return internalrollingfile.WithRollingClock(clock)
}

func closeAppenderList(appenders []Appender) error {
	return internalrouter.CloseAppenders(appenders, isAsyncAppender)
}

func isAsyncAppender(appender Appender) bool {
	switch value := appender.(type) {
	case *AsyncAppender:
		return true
	case *FilteredAppender:
		return isAsyncAppender(value.Delegate())
	default:
		return false
	}
}

func benchmarkEvent() Event {
	event := testEvent("service started", fixedTestTime())
	event.Logger = "goark.bench"
	event.Attrs = []slog.Attr{
		slog.String("profile", "bench"),
		slog.Int("index", 42),
		slog.Duration("elapsed", 10*time.Millisecond),
	}
	return event
}

func levelPointer(level slog.Level) *slog.Level {
	return &level
}

func markerPointer(marker goarklog.Marker) *goarklog.Marker {
	return &marker
}
