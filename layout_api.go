package goarklog

import internallayout "goark.dev/log/internal/layout"

// DefaultSpringBootPattern 是默认控制台输出格式，风格对齐 Spring Boot。
const DefaultSpringBootPattern = internallayout.DefaultSpringBootPattern

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

// StructuredFormat 标识受支持的结构化日志协议。
type StructuredFormat = internallayout.StructuredFormat

const (
	StructuredFormatECS      = internallayout.StructuredFormatECS
	StructuredFormatGELF     = internallayout.StructuredFormatGELF
	StructuredFormatLogstash = internallayout.StructuredFormatLogstash
)

// StructuredStacktracePrinter 标识结构化日志的异常栈打印策略。
type StructuredStacktracePrinter = internallayout.StructuredStacktracePrinter

const (
	StructuredStacktracePrinterStandard      = internallayout.StructuredStacktracePrinterStandard
	StructuredStacktracePrinterLoggingSystem = internallayout.StructuredStacktracePrinterLoggingSystem
)

// StructuredStacktraceOptions 描述异常链输出规则。
type StructuredStacktraceOptions = internallayout.StructuredStacktraceOptions

// StructuredECSOptions 描述 ECS 服务元数据。
type StructuredECSOptions = internallayout.StructuredECSOptions

// StructuredGELFOptions 描述 GELF 主机与服务元数据。
type StructuredGELFOptions = internallayout.StructuredGELFOptions

// StructuredJSONOptions 描述结构化 JSON 的编译参数。
type StructuredJSONOptions = internallayout.StructuredJSONOptions

// StructuredJSONLayout 输出 Spring Boot 兼容的结构化 JSON。
type StructuredJSONLayout = internallayout.StructuredJSONLayout

// StructuredJSONFieldAppender 接收自定义结构化成员。
type StructuredJSONFieldAppender = internallayout.StructuredJSONFieldAppender

// StructuredJSONCustomizer 以显式 Go API 追加结构化成员。
type StructuredJSONCustomizer = internallayout.StructuredJSONCustomizer

// StructuredJSONCustomizerFunc 把函数适配为 StructuredJSONCustomizer。
type StructuredJSONCustomizerFunc = internallayout.StructuredJSONCustomizerFunc

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

// NewStructuredJSONLayout 编译 ECS、GELF 或 Logstash 布局。
func NewStructuredJSONLayout(options StructuredJSONOptions) (*StructuredJSONLayout, error) {
	return internallayout.NewStructuredJSONLayout(options)
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

// NewCharsetLayout 使用指定字符集编码布局结果，UTF-8 不增加转换层。
func NewCharsetLayout(layout Layout, charset string) (Layout, error) {
	return internallayout.NewCharsetLayout(layout, charset)
}

// WithJSONTemplateResolverRegistry 设置用于解析自定义 resolver 的插件注册表。
func WithJSONTemplateResolverRegistry(registry *PluginRegistry) JSONTemplateLayoutOption {
	if registry == nil {
		registry = DefaultPluginRegistry()
	}
	return internallayout.WithJSONTemplateResolverLookup(registry.JSONTemplateResolverFactory)
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
