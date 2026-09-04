package layout

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"goark.dev/log/internal/layoutsupport"
	"goark.dev/log/internal/logcontext"
	"goark.dev/log/internal/logevent"
	"goark.dev/log/internal/logvalue"
)

// StructuredFormat 标识受支持的结构化日志协议。
type StructuredFormat string

const (
	StructuredFormatECS      StructuredFormat = "ecs"
	StructuredFormatGELF     StructuredFormat = "gelf"
	StructuredFormatLogstash StructuredFormat = "logstash"
)

// StructuredStacktraceOptions 描述异常链输出规则。
type StructuredStacktraceOptions struct {
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
	buf.WriteByte('{')
	writer := structuredWriter{buf: buf, layout: l}
	switch l.format {
	case StructuredFormatECS:
		l.appendECS(&writer, event)
	case StructuredFormatGELF:
		l.appendGELF(&writer, event)
	case StructuredFormatLogstash:
		l.appendLogstash(&writer, event)
	}
	if l.includeContext {
		for _, attr := range event.Attrs {
			if isStructuredControlAttr(attr.Key) {
				continue
			}
			key := l.contextPrefix + attr.Key
			if l.format == StructuredFormatGELF || l.format == StructuredFormatLogstash {
				key = "_" + strings.TrimLeft(key, "_")
			}
			if l.format == StructuredFormatGELF && (key == "_id" || !validGELFField(key)) {
				continue
			}
			writer.Add(key, attr.Value)
		}
	}
	for _, field := range l.add {
		writer.Add(field.key, slog.StringValue(field.value))
	}
	for _, customizer := range l.customizers {
		if customizer != nil {
			customizer.Customize(event, &writer)
		}
	}
	buf.WriteString("}\n")
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

func (l *StructuredJSONLayout) appendECS(writer *structuredWriter, event Event) {
	writer.Add("@timestamp", slog.TimeValue(layoutsupport.EventTime(event.Time)))
	if parent, ok := writer.beginObject("log", "log"); ok {
		writer.addPath("log.level", "level", slog.StringValue(levelName(event.Level)))
		writer.addPath("log.logger", "logger", slog.StringValue(event.Logger))
		writer.endObject(parent)
	}
	if parent, ok := writer.beginObject("process", "process"); ok {
		writer.addPath("process.pid", "pid", slog.IntValue(os.Getpid()))
		if threadParent, threadOK := writer.beginObject("process.thread", "thread"); threadOK {
			writer.addPath("process.thread.name", "name", slog.StringValue(eventThreadName(event)))
			writer.endObject(threadParent)
		}
		writer.endObject(parent)
	}
	serviceConfigured := strings.TrimSpace(l.ecs.ServiceName) != "" || strings.TrimSpace(l.ecs.ServiceVersion) != "" ||
		strings.TrimSpace(l.ecs.ServiceEnvironment) != "" || strings.TrimSpace(l.ecs.ServiceNodeName) != ""
	if parent, ok := writer.beginObjectIf(serviceConfigured, "service", "service"); ok {
		writer.addNonEmptyPath("service.name", "name", l.ecs.ServiceName)
		writer.addNonEmptyPath("service.version", "version", l.ecs.ServiceVersion)
		writer.addNonEmptyPath("service.environment", "environment", l.ecs.ServiceEnvironment)
		if nodeParent, nodeOK := writer.beginObjectIf(strings.TrimSpace(l.ecs.ServiceNodeName) != "", "service.node", "node"); nodeOK {
			writer.addNonEmptyPath("service.node.name", "name", l.ecs.ServiceNodeName)
			writer.endObject(nodeParent)
		}
		writer.endObject(parent)
	}
	writer.Add("message", slog.StringValue(event.Message))
	if parent, ok := writer.beginObject("ecs", "ecs"); ok {
		writer.addPath("ecs.version", "version", slog.StringValue("8.11"))
		writer.endObject(parent)
	}
	if event.Throwable != nil {
		if parent, ok := writer.beginObject("error", "error"); ok {
			writer.addNonEmptyPath("error.type", "type", event.Throwable.Type)
			writer.addNonEmptyPath("error.message", "message", event.Throwable.Message)
			writer.addPath("error.stack_trace", "stack_trace", slog.StringValue(formatStructuredStacktrace(event.Throwable, l.stacktrace)))
			writer.endObject(parent)
		}
	}
}

func (l *StructuredJSONLayout) appendGELF(writer *structuredWriter, event Event) {
	when := layoutsupport.EventTime(event.Time)
	writer.Add("version", slog.StringValue("1.1"))
	message := event.Message
	if strings.TrimSpace(message) == "" {
		message = "(blank)"
	}
	writer.Add("short_message", slog.StringValue(message))
	writer.Add("timestamp", slog.Float64Value(float64(when.UnixMilli())/1000))
	writer.Add("level", slog.IntValue(syslogSeverity(event.Level)))
	writer.Add("_level_name", slog.StringValue(levelName(event.Level)))
	writer.Add("_process_pid", slog.IntValue(os.Getpid()))
	writer.Add("_process_thread_name", slog.StringValue(eventThreadName(event)))
	host := l.gelf.Host
	if strings.TrimSpace(host) == "" {
		host = layoutsupport.HostName()
	}
	writer.addNonEmpty("host", host)
	writer.addNonEmpty("_service_name", l.gelf.ServiceName)
	writer.addNonEmpty("_service_version", l.gelf.ServiceVersion)
	writer.Add("_log_logger", slog.StringValue(event.Logger))
	l.appendError(writer, event, "_error_type", "_error_message", "_error_stack_trace")
	if event.Throwable != nil {
		writer.Add("full_message", slog.StringValue(formatStructuredStacktrace(event.Throwable, l.stacktrace)))
	}
}

func (l *StructuredJSONLayout) appendLogstash(writer *structuredWriter, event Event) {
	writer.Add("@timestamp", slog.TimeValue(layoutsupport.EventTime(event.Time)))
	writer.Add("@version", slog.StringValue("1"))
	writer.Add("message", slog.StringValue(event.Message))
	writer.Add("logger_name", slog.StringValue(event.Logger))
	writer.Add("thread_name", slog.StringValue(eventThreadName(event)))
	writer.Add("level", slog.StringValue(levelName(event.Level)))
	writer.Add("level_value", slog.IntValue(logstashLevelValue(event.Level)))
	if marker := eventMarkerString(event); marker != "" {
		writer.Add("tags", slog.StringValue(marker))
	}
	if event.Throwable != nil {
		writer.Add("stack_trace", slog.StringValue(formatStructuredStacktrace(event.Throwable, l.stacktrace)))
	}
}

func (l *StructuredJSONLayout) appendError(writer *structuredWriter, event Event, typeKey, messageKey, stackKey string) {
	if event.Throwable == nil {
		return
	}
	writer.addNonEmpty(typeKey, event.Throwable.Type)
	writer.addNonEmpty(messageKey, event.Throwable.Message)
	writer.Add(stackKey, slog.StringValue(formatStructuredStacktrace(event.Throwable, l.stacktrace)))
}

func logstashLevelValue(level slog.Level) int {
	switch {
	case level >= slog.LevelError:
		return 40000
	case level >= slog.LevelWarn:
		return 30000
	case level >= slog.LevelInfo:
		return 20000
	default:
		return 10000
	}
}

type structuredWriter struct {
	buf    *bytes.Buffer
	layout *StructuredJSONLayout
	comma  bool
}

func (w *structuredWriter) Add(key string, value slog.Value) {
	w.addPath(key, key, value)
}

func (w *structuredWriter) addPath(path, key string, value slog.Value) {
	path = strings.TrimSpace(path)
	key = strings.TrimSpace(key)
	if path == "" || !w.layout.enabled(path) {
		return
	}
	if renamed := w.layout.rename[path]; renamed != "" {
		key = renamed
	}
	logvalue.AppendJSONFieldValue(w.buf, key, value, w.comma)
	w.comma = true
}

func (w *structuredWriter) addNonEmpty(key, value string) {
	if strings.TrimSpace(value) != "" {
		w.Add(key, slog.StringValue(value))
	}
}

func (w *structuredWriter) addNonEmptyPath(path, key, value string) {
	if strings.TrimSpace(value) != "" {
		w.addPath(path, key, slog.StringValue(value))
	}
}

func (w *structuredWriter) beginObject(path, key string) (bool, bool) {
	return w.beginObjectIf(true, path, key)
}

func (w *structuredWriter) beginObjectIf(condition bool, path, key string) (bool, bool) {
	if !condition || !w.layout.objectEnabled(path) {
		return false, false
	}
	if renamed := w.layout.rename[path]; renamed != "" {
		key = renamed
	}
	parentComma := w.comma
	logvalue.AppendJSONKey(w.buf, key, parentComma)
	w.buf.WriteByte('{')
	w.comma = false
	return parentComma, true
}

func (w *structuredWriter) endObject(parentComma bool) {
	w.buf.WriteByte('}')
	w.comma = true
	_ = parentComma
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

func formatStructuredStacktrace(throwable *Throwable, options StructuredStacktraceOptions) string {
	if throwable == nil {
		return ""
	}
	chain := make([]*Throwable, 0, 4)
	for current := throwable; current != nil; current = current.Cause {
		chain = append(chain, current)
		if options.MaxThrowableDepth > 0 && len(chain) >= options.MaxThrowableDepth {
			break
		}
	}
	if options.RootFirst {
		for left, right := 0, len(chain)-1; left < right; left, right = left+1, right-1 {
			chain[left], chain[right] = chain[right], chain[left]
		}
	}
	var builder strings.Builder
	for index, current := range chain {
		if index > 0 {
			if options.RootFirst {
				builder.WriteString("\nWrapped by: ")
			} else {
				builder.WriteString("\nCaused by: ")
			}
		}
		if current.Type != "" {
			builder.WriteString(current.Type)
			builder.WriteString(": ")
		}
		builder.WriteString(current.Message)
		frames := current.Stack
		if !options.IncludeCommonFrames && index > 0 {
			frames = trimCommonFrames(frames, chain[index-1].Stack)
		}
		for _, frame := range frames {
			builder.WriteString("\n\tat ")
			builder.WriteString(frame)
			if options.IncludeHashes {
				builder.WriteString(" #")
				builder.WriteString(strconv.FormatUint(uint64(fnv1a(frame)), 16))
			}
		}
	}
	result := builder.String()
	if options.MaxLength > 0 && len(result) > options.MaxLength {
		result = result[:options.MaxLength]
		for !utf8.ValidString(result) {
			result = result[:len(result)-1]
		}
	}
	return result
}

func trimCommonFrames(frames, parent []string) []string {
	end, parentEnd := len(frames), len(parent)
	for end > 0 && parentEnd > 0 && frames[end-1] == parent[parentEnd-1] {
		end--
		parentEnd--
	}
	return frames[:end]
}

func fnv1a(value string) uint32 {
	const prime = 16777619
	hash := uint32(2166136261)
	for index := 0; index < len(value); index++ {
		hash ^= uint32(value[index])
		hash *= prime
	}
	return hash
}

var _ Layout = (*StructuredJSONLayout)(nil)
var _ StructuredJSONFieldAppender = (*structuredWriter)(nil)
