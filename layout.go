package goarklog

import (
	"bytes"
	"io"
	"log/slog"
	"time"

	internallayout "goark.dev/log/internal/layout"
)

const (
	// DefaultSpringBootPattern 是默认控制台输出格式，风格对齐 Spring Boot。
	DefaultSpringBootPattern = internallayout.DefaultSpringBootPattern
)

// Layout 把日志事件编码为字节。
type Layout = internallayout.Layout

// LayoutOptions 描述通用结构化布局参数。
type LayoutOptions = internallayout.LayoutOptions

// TextLayout 输出稳定的 key=value 文本。
type TextLayout = internallayout.TextLayout

// CSVLayout 输出单行 CSV，字段顺序固定。
type CSVLayout = internallayout.CSVLayout

// HTMLLayout 输出 HTML 表格行，适合文件或控制台片段组合。
type HTMLLayout = internallayout.HTMLLayout

// GELFLayout 输出 Graylog Extended Log Format 单行 JSON。
type GELFLayout = internallayout.GELFLayout

// RFC5424Layout 输出 RFC 5424 syslog 单行事件。
type RFC5424Layout = internallayout.RFC5424Layout

// SyslogLayout 是 RFC5424Layout 的语义别名。
type SyslogLayout = internallayout.SyslogLayout

// JSONLayout 输出 JSON 事件。
type JSONLayout = internallayout.JSONLayout

// XMLLayout 输出单事件 XML 片段。
type XMLLayout = internallayout.XMLLayout

// YAMLLayout 输出单事件 YAML 文档。
type YAMLLayout = internallayout.YAMLLayout

// PatternLayout 支持常用日志 pattern 占位符。
type PatternLayout = internallayout.PatternLayout

// JSONTemplateLayout 按 JSON 事件模板输出日志事件。
type JSONTemplateLayout = internallayout.JSONTemplateLayout

// JSONTemplateLayoutOption 调整 JSONTemplateLayout 编译行为。
type JSONTemplateLayoutOption = internallayout.JSONTemplateLayoutOption

// JSONTemplateResolver 是 JSON Template 字段值编码器。
type JSONTemplateResolver = internallayout.JSONTemplateResolver

// JSONTemplateResolverFactory 从配置构建 JSON Template resolver。
type JSONTemplateResolverFactory = internallayout.JSONTemplateResolverFactory

// JSONTemplateResolverBuildConfig 是 JSON Template resolver 插件的构建输入。
type JSONTemplateResolverBuildConfig = internallayout.JSONTemplateResolverBuildConfig

// NewDefaultLayout 创建默认 Spring Boot 风格布局。
func NewDefaultLayout() Layout {
	return internallayout.NewDefaultLayout()
}

// NewCSVLayout 创建可配置 CSV 布局。
func NewCSVLayout(options LayoutOptions) CSVLayout {
	return internallayout.NewCSVLayout(options)
}

// NewHTMLLayout 创建可配置 HTML 布局。
func NewHTMLLayout(options LayoutOptions) HTMLLayout {
	return internallayout.NewHTMLLayout(options)
}

// NewGELFLayout 创建可配置 GELF 布局。
func NewGELFLayout(options LayoutOptions) GELFLayout {
	return internallayout.NewGELFLayout(options)
}

// NewJSONLayout 创建可配置 JSON 布局。
func NewJSONLayout(options LayoutOptions) JSONLayout {
	return internallayout.NewJSONLayout(options)
}

// NewXMLLayout 创建可配置 XML 布局。
func NewXMLLayout(options LayoutOptions) XMLLayout {
	return internallayout.NewXMLLayout(options)
}

// NewYAMLLayout 创建可配置 YAML 布局。
func NewYAMLLayout(options LayoutOptions) YAMLLayout {
	return internallayout.NewYAMLLayout(options)
}

// NewPatternLayout 编译 pattern，避免热路径反复解析。
func NewPatternLayout(pattern string) (*PatternLayout, error) {
	return internallayout.NewPatternLayout(pattern)
}

// NewPatternLayoutWithOptions 使用指定布局参数编译 pattern。
func NewPatternLayoutWithOptions(pattern string, options LayoutOptions) (*PatternLayout, error) {
	return internallayout.NewPatternLayoutWithOptions(pattern, options)
}

// WithJSONTemplateResolverRegistry 设置用于解析自定义 resolver 的插件注册表。
func WithJSONTemplateResolverRegistry(registry *PluginRegistry) JSONTemplateLayoutOption {
	if registry == nil {
		registry = DefaultPluginRegistry()
	}
	return internallayout.WithJSONTemplateResolverLookup(registry.jsonTemplateResolverFactory)
}

// WithJSONTemplateLayoutOptions 设置 JSON Template 布局的通用输出参数。
func WithJSONTemplateLayoutOptions(layoutOptions LayoutOptions) JSONTemplateLayoutOption {
	return internallayout.WithJSONTemplateLayoutOptions(layoutOptions)
}

// NewJSONTemplateLayout 从 JSON 事件模板编译布局。
func NewJSONTemplateLayout(template string, options ...JSONTemplateLayoutOption) (*JSONTemplateLayout, error) {
	return internallayout.NewJSONTemplateLayout(template, jsonTemplateLayoutOptions(options...)...)
}

// NewJSONTemplateLayoutFromFile 从本地文件编译 JSON 事件模板。
func NewJSONTemplateLayoutFromFile(path string, options ...JSONTemplateLayoutOption) (*JSONTemplateLayout, error) {
	return internallayout.NewJSONTemplateLayoutFromFile(path, jsonTemplateLayoutOptions(options...)...)
}

func jsonTemplateLayoutOptions(options ...JSONTemplateLayoutOption) []JSONTemplateLayoutOption {
	merged := make([]JSONTemplateLayoutOption, 0, len(options)+1)
	merged = append(merged, WithJSONTemplateResolverRegistry(DefaultPluginRegistry()))
	return append(merged, options...)
}

func appendJSONEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs []slog.Attr) {
	internallayout.AppendJSONEvent(buf, when, level, logger, message, attrs)
}

func appendJSONFixedEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs [3]slog.Attr, count int) {
	internallayout.AppendJSONFixedEvent(buf, when, level, logger, message, attrs, count)
}

func writeLayoutHeader(writer io.Writer, layout Layout) (int, error) {
	return internallayout.WriteHeader(writer, layout)
}

func writeLayoutFooter(writer io.Writer, layout Layout) (int, error) {
	return internallayout.WriteFooter(writer, layout)
}
