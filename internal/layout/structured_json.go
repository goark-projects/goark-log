package layout

import (
	"bytes"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"goark.dev/log/internal/logcontext"
	"goark.dev/log/internal/logevent"
)

// StructuredFormat 标识受支持的结构化日志协议。
type StructuredFormat string

const (
	StructuredFormatECS      StructuredFormat = "ecs"
	StructuredFormatGELF     StructuredFormat = "gelf"
	StructuredFormatLogstash StructuredFormat = "logstash"
)

// StructuredStacktracePrinter 标识结构化日志的异常栈打印策略。
type StructuredStacktracePrinter string

const (
	StructuredStacktracePrinterStandard      StructuredStacktracePrinter = "standard"
	StructuredStacktracePrinterLoggingSystem StructuredStacktracePrinter = "logging-system"
)

// StructuredStacktraceOptions 描述异常链输出规则。
type StructuredStacktraceOptions struct {
	Printer             StructuredStacktracePrinter
	RootFirst           bool
	MaxLength           int
	MaxThrowableDepth   int
	IncludeCommonFrames bool
	IncludeHashes       bool
}

// StructuredECSOptions 描述 ECS 服务元数据。
type StructuredECSOptions struct {
	ServiceEnvironment string
	ServiceName        string
	ServiceNodeName    string
	ServiceVersion     string
}

// StructuredGELFOptions 描述 GELF 主机与服务元数据。
type StructuredGELFOptions struct {
	Host           string
	ServiceName    string
	ServiceVersion string
}

// StructuredJSONFieldAppender 接收自定义结构化成员。
type StructuredJSONFieldAppender interface {
	Add(key string, value slog.Value)
}

// StructuredJSONCustomizer 以显式 Go API 追加结构化成员。
type StructuredJSONCustomizer interface {
	Customize(event Event, fields StructuredJSONFieldAppender)
}

// StructuredJSONCustomizerFunc 把函数适配为 StructuredJSONCustomizer。
type StructuredJSONCustomizerFunc func(event Event, fields StructuredJSONFieldAppender)

func (f StructuredJSONCustomizerFunc) Customize(event Event, fields StructuredJSONFieldAppender) {
	if f != nil {
		f(event, fields)
	}
}

// StructuredJSONOptions 描述结构化 JSON 的编译参数。
type StructuredJSONOptions struct {
	Format         StructuredFormat
	Include        []string
	Exclude        []string
	Rename         map[string]string
	Add            map[string]string
	IncludeContext bool
	ContextPrefix  string
	Stacktrace     StructuredStacktraceOptions
	ECS            StructuredECSOptions
	GELF           StructuredGELFOptions
	Customizers    []StructuredJSONCustomizer
}

// StructuredJSONLayout 输出 Spring Boot 兼容的 ECS、GELF 或 Logstash JSON。
type StructuredJSONLayout struct {
	format         StructuredFormat
	include        map[string]struct{}
	exclude        map[string]struct{}
	rename         map[string]string
	add            []structuredStaticField
	nestedAdd      []*structuredPathNode
	includeContext bool
	contextPrefix  string
	stacktrace     StructuredStacktraceOptions
	ecs            StructuredECSOptions
	gelf           StructuredGELFOptions
	customizers    []StructuredJSONCustomizer
}

type structuredStaticField struct {
	key   string
	value string
}

// NewStructuredJSONLayout 编译结构化日志规则。
func NewStructuredJSONLayout(options StructuredJSONOptions) (*StructuredJSONLayout, error) {
	format := StructuredFormat(strings.ToLower(strings.TrimSpace(string(options.Format))))
	switch format {
	case StructuredFormatECS, StructuredFormatGELF, StructuredFormatLogstash:
	default:
		return nil, fmt.Errorf("goark-log: unsupported structured format %q", options.Format)
	}
	printer := StructuredStacktracePrinter(strings.ToLower(strings.TrimSpace(string(options.Stacktrace.Printer))))
	switch printer {
	case "", StructuredStacktracePrinterStandard, StructuredStacktracePrinterLoggingSystem:
	default:
		return nil, fmt.Errorf("goark-log: unsupported structured stacktrace printer %q", options.Stacktrace.Printer)
	}
	if options.Stacktrace.MaxLength < 0 {
		return nil, fmt.Errorf("goark-log: structured stacktrace maximum length must be positive")
	}
	if options.Stacktrace.MaxThrowableDepth < 0 {
		return nil, fmt.Errorf("goark-log: structured stacktrace maximum throwable depth must be positive")
	}
	options.Stacktrace.Printer = printer
	layout := &StructuredJSONLayout{
		format:         format,
		include:        compileFieldSet(options.Include),
		exclude:        compileFieldSet(options.Exclude),
		rename:         compileRename(options.Rename),
		includeContext: options.IncludeContext,
		contextPrefix:  strings.TrimSpace(options.ContextPrefix),
		stacktrace:     options.Stacktrace,
		ecs:            options.ECS,
		gelf:           options.GELF,
		customizers:    append([]StructuredJSONCustomizer(nil), options.Customizers...),
	}
	keys := make([]string, 0, len(options.Add))
	for key := range options.Add {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		layout.add = append(layout.add, structuredStaticField{key: key, value: options.Add[key]})
	}
	if format == StructuredFormatECS {
		attrs := make([]slog.Attr, 0, len(layout.add))
		for _, field := range layout.add {
			attrs = append(attrs, slog.String(field.key, field.value))
		}
		var err error
		layout.nestedAdd, err = compileStructuredPaths(attrs, "")
		if err != nil {
			return nil, err
		}
		layout.add = nil
	}
	return layout, nil
}

func compileFieldSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result[value] = struct{}{}
		}
	}
	return result
}

func compileRename(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for source, target := range values {
		source, target = strings.TrimSpace(source), strings.TrimSpace(target)
		if source != "" && target != "" {
			result[source] = target
		}
	}
	return result
}

func (l *StructuredJSONLayout) Format(buf *bytes.Buffer, event Event) error {
	if len(l.customizers) == 0 {
		return l.formatDefault(buf, event)
	}
	return l.formatCustomized(buf, event)
}

func (l *StructuredJSONLayout) formatDefault(buf *bytes.Buffer, event Event) error {
	buf.WriteByte('{')
	writer := structuredWriter{buf: buf, layout: l}
	if err := l.appendStructuredFields(&writer, event); err != nil {
		return err
	}
	buf.WriteString("}\n")
	return nil
}

func (l *StructuredJSONLayout) formatCustomized(buf *bytes.Buffer, event Event) error {
	buf.WriteByte('{')
	writer := structuredWriter{buf: buf, layout: l}
	if err := l.appendStructuredFields(&writer, event); err != nil {
		return err
	}
	for _, customizer := range l.customizers {
		if customizer != nil {
			customizer.Customize(event, &writer)
		}
	}
	buf.WriteString("}\n")
	return nil
}

func (l *StructuredJSONLayout) appendStructuredFields(writer *structuredWriter, event Event) error {
	switch l.format {
	case StructuredFormatECS:
		l.appendECS(writer, event)
	case StructuredFormatGELF:
		l.appendGELF(writer, event)
	case StructuredFormatLogstash:
		l.appendLogstash(writer, event)
	}
	if l.includeContext {
		if l.format == StructuredFormatECS {
			if !requiresNestedStructuredPaths(event.Attrs, l.contextPrefix) {
				for _, attr := range event.Attrs {
					if !isStructuredControlAttr(attr.Key) {
						writer.Add(attr.Key, attr.Value)
					}
				}
			} else {
				contextAttrs := make([]slog.Attr, 0, len(event.Attrs))
				for _, attr := range event.Attrs {
					if !isStructuredControlAttr(attr.Key) {
						contextAttrs = append(contextAttrs, attr)
					}
				}
				nodes, err := compileStructuredPaths(contextAttrs, l.contextPrefix)
				if err != nil {
					return err
				}
				writer.addStructuredPaths(nodes)
			}
		} else {
			for _, attr := range event.Attrs {
				if isStructuredControlAttr(attr.Key) {
					continue
				}
				key := joinStructuredPath(l.contextPrefix, attr.Key, "_")
				if l.format == StructuredFormatGELF {
					key = "_" + strings.TrimLeft(l.contextPrefix+attr.Key, "_")
					if key == "_id" || !validGELFField(key) {
						continue
					}
				}
				writer.Add(key, attr.Value)
			}
		}
	}
	writer.addStructuredPaths(l.nestedAdd)
	for _, field := range l.add {
		writer.Add(field.key, slog.StringValue(field.value))
	}
	return nil
}

func isStructuredControlAttr(key string) bool {
	switch key {
	case logevent.ThrowableAttrKey, logcontext.MarkerAttrKey, logcontext.ThreadNameAttrKey, logcontext.StackAttrKey:
		return true
	default:
		return false
	}
}

func validGELFField(key string) bool {
	for _, char := range key {
		if char == '_' || char == '.' || char == '-' || char >= '0' && char <= '9' ||
			char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' {
			continue
		}
		return false
	}
	return true
}

func (l *StructuredJSONLayout) enabled(path string) bool {
	if _, excluded := l.exclude[path]; excluded {
		return false
	}
	if len(l.include) == 0 {
		return true
	}
	_, included := l.include[path]
	return included
}

func (l *StructuredJSONLayout) objectEnabled(path string) bool {
	if _, excluded := l.exclude[path]; excluded {
		return false
	}
	if len(l.include) == 0 {
		return true
	}
	prefix := path + "."
	for included := range l.include {
		if included == path || strings.HasPrefix(included, prefix) {
			return true
		}
	}
	return false
}

var _ Layout = (*StructuredJSONLayout)(nil)
