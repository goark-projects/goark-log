package integration

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"time"

	"goark.dev/log"
	"goark.dev/log/internal/configfile"
	internallayout "goark.dev/log/internal/layout"
	internallogevent "goark.dev/log/internal/logevent"
	internalnativelogger "goark.dev/log/internal/nativelogger"
	internalrollingfile "goark.dev/log/internal/rollingfile"
	internalrouter "goark.dev/log/internal/router"
)

const (
	EnvConfigPath = log.EnvConfigPath

	ConfigSourceExplicit = log.ConfigSourceExplicit
	ConfigSourceEnv      = log.ConfigSourceEnv
	ConfigSourceBoot     = log.ConfigSourceBoot
	ConfigSourceFile     = log.ConfigSourceFile
	ConfigSourceDefault  = log.ConfigSourceDefault

	LevelAll   = log.LevelAll
	LevelTrace = log.LevelTrace
	LevelFatal = log.LevelFatal
	LevelOff   = log.LevelOff

	FilterNeutral = log.FilterNeutral
	FilterAccept  = log.FilterAccept
	FilterDeny    = log.FilterDeny

	MapFilterAnd = log.MapFilterAnd
	MapFilterOr  = log.MapFilterOr

	AsyncOverflowBlock        = log.AsyncOverflowBlock
	AsyncOverflowDrop         = log.AsyncOverflowDrop
	AsyncOverflowDropDebug    = log.AsyncOverflowDropDebug
	AsyncOverflowSyncFallback = log.AsyncOverflowSyncFallback

	AsyncWaitBlock = log.AsyncWaitBlock
	AsyncWaitSleep = log.AsyncWaitSleep
	AsyncWaitYield = log.AsyncWaitYield
	AsyncWaitSpin  = log.AsyncWaitSpin

	RollingFileIndexNoMax = log.RollingFileIndexNoMax
	RollingFileIndexMax   = log.RollingFileIndexMax
	RollingFileIndexMin   = log.RollingFileIndexMin

	StructuredDataIDAttrKey   = log.StructuredDataIDAttrKey
	StructuredDataTypeAttrKey = log.StructuredDataTypeAttrKey

	defaultThreadName = internallogevent.DefaultThreadName
)

type (
	Appender              = log.Appender
	AppenderRef           = log.AppenderRef
	AsyncAppender         = log.AsyncAppender
	AsyncErrorHandlerFunc = log.AsyncErrorHandlerFunc
	AsyncLoggerOptions    = log.AsyncLoggerOptions
	AsyncWaitOptions      = log.AsyncWaitOptions
	Event                 = log.Event
	FileAppender          = log.FileAppender
	Filter                = log.Filter
	FilterDecision        = log.FilterDecision
	FilterFunc            = log.FilterFunc
	Layout                = log.Layout
	LayoutOptions         = log.LayoutOptions
	Options               = log.Options
	PatternLayout         = log.PatternLayout
	PluginRegistry        = log.PluginRegistry
	RollingDeleteAction   = log.RollingDeleteAction
	RollingFileAppender   = log.RollingFileAppender
	RollingFileIndexMode  = log.RollingFileIndexMode
	RollingFileOption     = log.RollingFileOption
	RootLogger            = log.RootLogger
	LoggerRule            = log.LoggerRule
	LoggerConfiguration   = log.LoggerConfiguration
	Marker                = log.Marker
	Message               = log.Message
	MessageFactoryFunc    = log.MessageFactoryFunc
	TextLayout            = log.TextLayout
	CSVLayout             = log.CSVLayout
	GELFLayout            = log.GELFLayout
	HTMLLayout            = log.HTMLLayout
	JSONLayout            = log.JSONLayout
	RFC5424Layout         = log.RFC5424Layout
	Throwable             = log.Throwable
	XMLLayout             = log.XMLLayout
	YAMLLayout            = log.YAMLLayout

	AppenderBuildConfig             = log.AppenderBuildConfig
	ConfigLoadOption                = log.ConfigLoadOption
	ConfigResult                    = log.ConfigResult
	ConfigSource                    = log.ConfigSource
	FilterBuildConfig               = log.FilterBuildConfig
	JSONTemplateResolver            = log.JSONTemplateResolver
	JSONTemplateResolverBuildConfig = log.JSONTemplateResolverBuildConfig
	LayoutBuildConfig               = log.LayoutBuildConfig
	PluginRegistrarFunc             = log.PluginRegistrarFunc
	PropertyMap                     = log.PropertyMap
	ScriptEvaluatorFunc             = log.ScriptEvaluatorFunc
	FilteredAppender                = log.FilteredAppender

	layoutConfig = configfile.LayoutConfig
)

type fileConfig struct {
	*configfile.Config
}

var (
	ConfigureDefault              = log.ConfigureDefault
	ContextAttrs                  = log.ContextAttrs
	DefaultOptions                = log.DefaultOptions
	DefaultPluginRegistry         = log.DefaultPluginRegistry
	LevelName                     = log.LevelName
	LoadOptions                   = log.LoadOptions
	NewAppenderRef                = log.NewAppenderRef
	NewAsyncAppender              = log.NewAsyncAppender
	NewAttrFilter                 = log.NewAttrFilter
	NewBurstFilter                = log.NewBurstFilter
	NewConsoleAppender            = log.NewConsoleAppender
	NewConfigReloader             = log.NewConfigReloader
	NewConfiguredLoggerContext    = log.NewConfiguredLoggerContext
	NewDynamicThresholdFilter     = log.NewDynamicThresholdFilter
	NewFileAppender               = log.NewFileAppender
	NewFilteredAppender           = log.NewFilteredAppender
	NewHandler                    = log.NewHandler
	NewJSONAppender               = log.NewJSONAppender
	NewJSONFileAppender           = log.NewJSONFileAppender
	NewJSONLayout                 = log.NewJSONLayout
	NewJSONTemplateLayout         = log.NewJSONTemplateLayout
	NewJSONTemplateLayoutFromFile = log.NewJSONTemplateLayoutFromFile
	NewLevelRegistry              = log.NewLevelRegistry
	NewLogger                     = log.NewLogger
	NewLoggerContext              = log.NewLoggerContext
	NewLookupResolver             = log.NewLookupResolver
	NewMapFilter                  = log.NewMapFilter
	NewMapMessage                 = log.NewMapMessage
	NewMarker                     = log.NewMarker
	NewMarkerFilter               = log.NewMarkerFilter
	NewNativeLogger               = log.NewNativeLogger
	NewNoMarkerFilter             = log.NewNoMarkerFilter
	NewParameterizedMessage       = log.NewParameterizedMessage
	NewPatternLayout              = log.NewPatternLayout
	NewPatternLayoutWithOptions   = log.NewPatternLayoutWithOptions
	NewPluginRegistry             = log.NewPluginRegistry
	NewPluginSet                  = log.NewPluginSet
	NewRegexFilter                = log.NewRegexFilter
	NewRollingFileAppender        = log.NewRollingFileAppender
	NewScriptFilter               = log.NewScriptFilter
	NewSimpleMessage              = log.NewSimpleMessage
	NewStatusLogger               = log.NewStatusLogger
	NewStringMatchFilter          = log.NewStringMatchFilter
	NewStructuredDataFilter       = log.NewStructuredDataFilter
	NewStructuredDataMessage      = log.NewStructuredDataMessage
	NewThreadContextStackFilter   = log.NewThreadContextStackFilter
	NewThresholdFilter            = log.NewThresholdFilter
	NewThrowable                  = log.NewThrowable
	NewThrowableFilter            = log.NewThrowableFilter
	NewThrowableWithStack         = log.NewThrowableWithStack
	NewTimeFilter                 = log.NewTimeFilter
	NewTimeFilterInLocation       = log.NewTimeFilterInLocation
	ThrowableAttr                 = log.ThrowableAttr

	NewConfigured                    = log.NewConfigured
	NewConfiguredHandler             = log.NewConfiguredHandler
	ParseByteSize                    = log.ParseByteSize
	ParseLevel                       = log.ParseLevel
	ParseMonitorInterval             = log.ParseMonitorInterval
	ParseRollingInterval             = log.ParseRollingInterval
	RegisterLayout                   = log.RegisterLayout
	RegisterLevel                    = log.RegisterLevel
	WithAppenderRefFilters           = log.WithAppenderRefFilters
	WithAppenderRefLevel             = log.WithAppenderRefLevel
	WithAppenderRefLocation          = log.WithAppenderRefLocation
	WithAsyncBatchSize               = log.WithAsyncBatchSize
	WithAsyncCloseAppenders          = log.WithAsyncCloseAppenders
	WithAsyncErrorHandler            = log.WithAsyncErrorHandler
	WithAsyncOverflowStrategy        = log.WithAsyncOverflowStrategy
	WithAsyncQueueSize               = log.WithAsyncQueueSize
	WithAsyncWaitOptions             = log.WithAsyncWaitOptions
	WithAsyncWaitStrategy            = log.WithAsyncWaitStrategy
	WithBootPropertyResolver         = log.WithBootPropertyResolver
	WithConfigEnvKey                 = log.WithConfigEnvKey
	WithConfigLookups                = log.WithConfigLookups
	WithConfigPath                   = log.WithConfigPath
	WithConfigData                   = log.WithConfigData
	WithConfigWorkingDir             = log.WithConfigWorkingDir
	WithConsoleLayout                = log.WithConsoleLayout
	WithConsoleName                  = log.WithConsoleName
	WithConsoleWriter                = log.WithConsoleWriter
	WithFileBufferSize               = log.WithFileBufferSize
	WithFileCreateOnDemand           = log.WithFileCreateOnDemand
	WithFileFlushOnWrite             = log.WithFileFlushOnWrite
	WithFileLayout                   = log.WithFileLayout
	WithFilterOnMatch                = log.WithFilterOnMatch
	WithFilterOnMismatch             = log.WithFilterOnMismatch
	WithJSONAppenderBufferSize       = log.WithJSONAppenderBufferSize
	WithJSONAppenderName             = log.WithJSONAppenderName
	WithJSONAppenderWriter           = log.WithJSONAppenderWriter
	WithJSONTemplateLayoutOptions    = log.WithJSONTemplateLayoutOptions
	WithJSONTemplateResolverRegistry = log.WithJSONTemplateResolverRegistry
	WithLoggerCaller                 = log.WithLoggerCaller
	WithLoggerContextStatus          = log.WithLoggerContextStatus
	WithLoggerMessageFactory         = log.WithLoggerMessageFactory
	WithMapFilterOnMatch             = log.WithMapFilterOnMatch
	WithMapFilterOnMismatch          = log.WithMapFilterOnMismatch
	WithMapFilterOperator            = log.WithMapFilterOperator
	WithPluginAppender               = log.WithPluginAppender
	WithPluginJSONTemplateResolver   = log.WithPluginJSONTemplateResolver
	WithPluginLayout                 = log.WithPluginLayout
	WithPluginLookup                 = log.WithPluginLookup
	WithPluginRegistry               = log.WithPluginRegistry
	WithRegexOnMatch                 = log.WithRegexOnMatch
	WithRegexOnMismatch              = log.WithRegexOnMismatch
	WithRolloverOnStartup            = log.WithRolloverOnStartup
	WithRollingActionQueueSize       = log.WithRollingActionQueueSize
	WithRollingAsyncActions          = log.WithRollingAsyncActions
	WithRollingCronSchedule          = log.WithRollingCronSchedule
	WithRollingDeleteActions         = log.WithRollingDeleteActions
	WithRollingDirectWrite           = log.WithRollingDirectWrite
	WithRollingFileAppend            = log.WithRollingFileAppend
	WithRollingFileBufferSize        = log.WithRollingFileBufferSize
	WithRollingFileCreateOnDemand    = log.WithRollingFileCreateOnDemand
	WithRollingFileIndexMode         = log.WithRollingFileIndexMode
	WithRollingFileLayout            = log.WithRollingFileLayout
	WithRollingFilePattern           = log.WithRollingFilePattern
	WithRollingGzip                  = log.WithRollingGzip
	WithRollingInterval              = log.WithRollingInterval
	WithRollingMaxBackups            = log.WithRollingMaxBackups
	WithRollingMaxSize               = log.WithRollingMaxSize
	WithRollingTotalSizeCap          = log.WithRollingTotalSizeCap
	WithRollingCleanHistoryOnStart   = log.WithRollingCleanHistoryOnStart
	WithRoutingAttrKey               = log.WithRoutingAttrKey
	WithRoutingDefault               = log.WithRoutingDefault
	WithRoutingKeyFunc               = log.WithRoutingKeyFunc
	WithScriptFilterOnMatch          = log.WithScriptFilterOnMatch
	WithScriptFilterOnMismatch       = log.WithScriptFilterOnMismatch
	WithStatusBufferSize             = log.WithStatusBufferSize
	WithStatusLevel                  = log.WithStatusLevel
	WithStatusWriter                 = log.WithStatusWriter
	WithContextAttrs                 = log.WithContextAttrs
	WithContextStack                 = log.WithContextStack
	WithMarker                       = log.WithMarker
	WithPluginFilter                 = log.WithPluginFilter
	WithThreadName                   = log.WithThreadName
	NewFailoverAppender              = log.NewFailoverAppender
	NewRoutingAppender               = log.NewRoutingAppender
	NewRewriteAppender               = log.NewRewriteAppender
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

func decodeStructuredConfig(reader io.Reader, lookups *log.LookupResolver) (*fileConfig, error) {
	config, err := configfile.DecodeStructured(reader, lookups)
	if err != nil {
		return nil, err
	}
	return &fileConfig{Config: config}, nil
}

func decodeXMLConfig(reader io.Reader, lookups *log.LookupResolver) (*fileConfig, error) {
	config, err := configfile.DecodeXML(reader, lookups)
	if err != nil {
		return nil, err
	}
	return &fileConfig{Config: config}, nil
}

func decodePropertiesConfig(reader io.Reader, lookups *log.LookupResolver) (*fileConfig, error) {
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

func markerPointer(marker log.Marker) *log.Marker {
	return &marker
}
