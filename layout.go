package goarklog

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"html"
	"log/slog"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"goark.dev/log/internal/jsoncodec"
)

const (
	// DefaultSpringBootPattern 是默认控制台输出格式，风格对齐 Spring Boot。
	DefaultSpringBootPattern = "%d %5level %pid --- [%thread] %logger : %msg%attrs%n"
	defaultTimeFormat        = "2006-01-02T15:04:05.000Z07:00"
)

var processIDString = strconv.Itoa(os.Getpid())
var patternSequence atomic.Uint64

// Layout 把日志事件编码为字节。
type Layout interface {
	Format(buf *bytes.Buffer, event Event) error
}

// NewDefaultLayout 创建默认 Spring Boot 风格布局。
func NewDefaultLayout() Layout {
	layout, _ := NewPatternLayout(DefaultSpringBootPattern)
	return layout
}

// TextLayout 输出稳定的 key=value 文本。
type TextLayout struct{}

func (TextLayout) Format(buf *bytes.Buffer, event Event) error {
	appendKey(buf, "time")
	buf.Write(event.Time.AppendFormat(buf.AvailableBuffer(), defaultTimeFormat))
	appendKeyValue(buf, "level", levelName(event.Level))
	appendKeyValue(buf, "logger", event.Logger)
	appendKeyValue(buf, "msg", event.Message)
	for _, attr := range event.Attrs {
		appendKeyValueAttr(buf, attr.Key, attr.Value)
	}
	buf.WriteByte('\n')
	return nil
}

// JSONLayout 输出 JSON 事件。
type JSONLayout struct {
	options LayoutOptions
	state   *jsonLayoutState
}

// NewJSONLayout 创建可配置 JSON 布局。
func NewJSONLayout(options LayoutOptions) JSONLayout {
	layout := JSONLayout{options: options}
	if options.Complete {
		layout.state = &jsonLayoutState{}
	}
	return layout
}

func (l JSONLayout) Format(buf *bytes.Buffer, event Event) error {
	appendJSONCompleteSeparator(buf, l.options, l.state)
	appendJSONLayoutEvent(buf, event, l.options)
	return nil
}

func (l JSONLayout) AppendHeader(buf *bytes.Buffer) error {
	appendJSONCompleteHeader(buf, l.options, l.state)
	return nil
}

func (l JSONLayout) AppendFooter(buf *bytes.Buffer) error {
	appendJSONCompleteFooter(buf, l.options)
	return nil
}

type jsonLayoutState struct {
	events atomic.Uint64
}

func appendJSONCompleteSeparator(buf *bytes.Buffer, options LayoutOptions, state *jsonLayoutState) {
	if !options.Complete || state == nil || state.events.Add(1) <= 1 {
		return
	}
	buf.WriteByte(',')
	if options.EventEOL || !options.Compact {
		buf.WriteByte('\n')
	}
}

func appendJSONCompleteHeader(buf *bytes.Buffer, options LayoutOptions, state *jsonLayoutState) {
	if state != nil {
		state.events.Store(0)
	}
	if !options.Complete {
		return
	}
	header := options.Header
	if strings.TrimSpace(header) == "" {
		header = "["
	}
	buf.WriteString(header)
	if options.EventEOL || !options.Compact {
		buf.WriteByte('\n')
	}
}

func appendJSONCompleteFooter(buf *bytes.Buffer, options LayoutOptions) {
	if !options.Complete {
		return
	}
	footer := options.Footer
	if strings.TrimSpace(footer) == "" {
		footer = "]"
	}
	buf.WriteString(footer)
}

func appendJSONEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs []slog.Attr) {
	buf.WriteByte('{')
	appendJSONFieldTime(buf, "time", when, defaultTimeFormat, false)
	appendJSONFieldString(buf, "level", levelName(level), true)
	appendJSONFieldString(buf, "logger", logger, true)
	appendJSONFieldString(buf, "msg", message, true)
	for _, attr := range attrs {
		appendJSONFieldValue(buf, attr.Key, attr.Value, true)
	}
	buf.WriteString("}\n")
}

func appendJSONLayoutEvent(buf *bytes.Buffer, event Event, options LayoutOptions) {
	buf.WriteByte('{')
	appendJSONFieldTime(buf, "time", event.Time, defaultTimeFormat, false)
	appendJSONFieldString(buf, "level", levelName(event.Level), true)
	appendJSONFieldString(buf, "logger", event.Logger, true)
	appendJSONFieldString(buf, "msg", event.Message, true)
	if options.PropertiesAsList {
		appendJSONAttrsListField(buf, "contextMap", event.Attrs, true)
	} else {
		for _, attr := range event.Attrs {
			appendJSONFieldValue(buf, attr.Key, attr.Value, true)
		}
	}
	if event.Throwable != nil && (options.IncludeStacktrace || options.StacktraceAsString) {
		appendJSONKey(buf, "thrown", true)
		if options.StacktraceAsString {
			appendJSONString(buf, throwableStackString(event.Throwable))
		} else {
			appendThrowableJSON(buf, event.Throwable)
		}
	}
	buf.WriteByte('}')
	appendLayoutTerminator(buf, options)
}

func appendJSONFixedEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs [3]slog.Attr, count int) {
	buf.WriteByte('{')
	appendJSONFieldTime(buf, "time", when, defaultTimeFormat, false)
	appendJSONFieldString(buf, "level", levelName(level), true)
	appendJSONFieldString(buf, "logger", logger, true)
	appendJSONFieldString(buf, "msg", message, true)
	for index := 0; index < count && index < len(attrs); index++ {
		attr := attrs[index]
		appendJSONFieldValue(buf, attr.Key, attr.Value, true)
	}
	buf.WriteString("}\n")
}

// PatternLayout 支持 Log4j2 风格的基础占位符子集。
type PatternLayout struct {
	tokens []patternToken
}

type patternToken struct {
	kind      patternTokenKind
	literal   string
	format    string
	key       string
	minWidth  int
	maxWidth  int
	precision int
	repeat    int
	leftAlign bool
	timeUnix  timeUnixMode
	child     *PatternLayout
	regex     *regexp.Regexp
	value     string
	repl      string
	ignore    bool
}

type patternTokenKind int

const (
	tokenLiteral patternTokenKind = iota
	tokenTime
	tokenLevel
	tokenPID
	tokenThread
	tokenLogger
	tokenMessage
	tokenAttrs
	tokenAttr
	tokenError
	tokenNewline
	tokenMarker
	tokenContextStack
	tokenCallerClass
	tokenCallerMethod
	tokenCallerFile
	tokenCallerLine
	tokenCallerLocation
	tokenUUID
	tokenSubPattern
	tokenHighlight
	tokenStyle
	tokenNotEmpty
	tokenReplace
	tokenEncode
	tokenEquals
	tokenMaxLen
	tokenRepeat
	tokenSequence
)

type timeUnixMode uint8

const (
	timeUnixNone timeUnixMode = iota
	timeUnixSeconds
	timeUnixMillis
	timeUnixMicros
	timeUnixNanos
)

// NewPatternLayout 编译 pattern，避免热路径反复解析。
func NewPatternLayout(pattern string) (*PatternLayout, error) {
	if strings.TrimSpace(pattern) == "" {
		pattern = DefaultSpringBootPattern
	}
	tokens, err := compilePattern(pattern)
	if err != nil {
		return nil, err
	}
	return &PatternLayout{tokens: tokens}, nil
}

func (l *PatternLayout) Format(buf *bytes.Buffer, event Event) error {
	if l == nil {
		return NewDefaultLayout().Format(buf, event)
	}
	var caller callerCache
	for _, token := range l.tokens {
		appendPatternToken(buf, token, event, &caller)
	}
	return nil
}

func compilePattern(pattern string) ([]patternToken, error) {
	tokens := make([]patternToken, 0, 16)
	for len(pattern) > 0 {
		index := strings.IndexByte(pattern, '%')
		if index < 0 {
			tokens = append(tokens, patternToken{kind: tokenLiteral, literal: pattern})
			break
		}
		if index > 0 {
			tokens = append(tokens, patternToken{kind: tokenLiteral, literal: pattern[:index]})
			pattern = pattern[index:]
			continue
		}
		token, size, err := readPatternToken(pattern)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
		pattern = pattern[size:]
	}
	return tokens, nil
}

func readPatternToken(pattern string) (patternToken, int, error) {
	if strings.HasPrefix(pattern, "%%") {
		return patternToken{kind: tokenLiteral, literal: "%"}, 2, nil
	}
	index := 1
	token := patternToken{}
	if index < len(pattern) && pattern[index] == '-' {
		token.leftAlign = true
		index++
	}
	for index < len(pattern) && isPatternDigit(pattern[index]) {
		token.minWidth = token.minWidth*10 + int(pattern[index]-'0')
		index++
	}
	if index < len(pattern) && pattern[index] == '.' {
		index++
		for index < len(pattern) && isPatternDigit(pattern[index]) {
			token.maxWidth = token.maxWidth*10 + int(pattern[index]-'0')
			index++
		}
	}
	converterStart := index
	if index < len(pattern) && pattern[index] == 'X' {
		index++
	} else {
		for index < len(pattern) && isPatternLetter(pattern[index]) {
			index++
		}
	}
	if converterStart == index {
		return patternToken{}, 0, fmt.Errorf("goark-log: unsupported pattern token near %q", pattern)
	}
	converter := pattern[converterStart:index]
	options := []string(nil)
	if index < len(pattern) && pattern[index] == '{' {
		for index < len(pattern) && pattern[index] == '{' {
			option, next, err := readPatternOption(pattern, index)
			if err != nil {
				return patternToken{}, 0, err
			}
			options = append(options, option)
			index = next
		}
	}
	if err := configurePatternToken(&token, converter, options); err != nil {
		return patternToken{}, 0, err
	}
	return token, index, nil
}

func readPatternOption(pattern string, start int) (string, int, error) {
	depth := 0
	for index := start; index < len(pattern); index++ {
		switch pattern[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return pattern[start+1 : index], index + 1, nil
			}
		}
	}
	return "", 0, fmt.Errorf("goark-log: pattern option is not closed near %q", pattern[start:])
}

func configurePatternToken(token *patternToken, converter string, options []string) error {
	normalized := strings.ToLower(converter)
	option := firstPatternOption(options)
	switch {
	case converter == "C" || normalized == "class":
		token.kind = tokenCallerClass
	case converter == "M" || normalized == "method":
		token.kind = tokenCallerMethod
	case converter == "F" || normalized == "file":
		token.kind = tokenCallerFile
	case converter == "L" || normalized == "line":
		token.kind = tokenCallerLine
	case converter == "l" || normalized == "location":
		token.kind = tokenCallerLocation
	case normalized == "d" || normalized == "date":
		token.kind = tokenTime
		token.format, token.timeUnix = normalizeTimePattern(option)
	case normalized == "level" || normalized == "p":
		token.kind = tokenLevel
	case normalized == "pid" || normalized == "processid":
		token.kind = tokenPID
	case normalized == "thread" || normalized == "t":
		token.kind = tokenThread
	case normalized == "logger" || converter == "c":
		token.kind = tokenLogger
		token.precision = parsePatternPrecision(option)
	case normalized == "msg" || normalized == "message" || converter == "m":
		token.kind = tokenMessage
	case normalized == "attrs" || normalized == "kvp" || normalized == "map":
		token.kind = tokenAttrs
	case converter == "X" || normalized == "mdc":
		if strings.TrimSpace(option) == "" {
			token.kind = tokenAttrs
			return nil
		}
		token.kind = tokenAttr
		token.key = strings.TrimSpace(option)
	case normalized == "ex" || normalized == "throwable" || normalized == "exception":
		token.kind = tokenError
		token.key = strings.ToLower(strings.TrimSpace(option))
	case normalized == "marker":
		token.kind = tokenMarker
	case normalized == "ndc" || normalized == "x":
		token.kind = tokenContextStack
	case normalized == "n":
		token.kind = tokenNewline
	case normalized == "uuid":
		token.kind = tokenUUID
	case normalized == "sequencenumber" || normalized == "sn":
		token.kind = tokenSequence
	case normalized == "highlight":
		child, err := NewPatternLayout(option)
		if err != nil {
			return err
		}
		token.kind = tokenHighlight
		token.child = child
	case normalized == "style":
		child, err := NewPatternLayout(option)
		if err != nil {
			return err
		}
		token.kind = tokenStyle
		token.child = child
		token.value = patternOption(options, 1)
	case normalized == "notempty":
		child, err := NewPatternLayout(option)
		if err != nil {
			return err
		}
		token.kind = tokenNotEmpty
		token.child = child
	case normalized == "replace":
		if len(options) < 3 {
			return fmt.Errorf("goark-log: replace pattern converter requires pattern, regex and replacement")
		}
		child, err := NewPatternLayout(options[0])
		if err != nil {
			return err
		}
		expression, err := regexp.Compile(options[1])
		if err != nil {
			return fmt.Errorf("goark-log: replace pattern regex %q is invalid: %w", options[1], err)
		}
		token.kind = tokenReplace
		token.child = child
		token.regex = expression
		token.repl = options[2]
	case normalized == "enc" || normalized == "encode":
		child, err := NewPatternLayout(option)
		if err != nil {
			return err
		}
		token.kind = tokenEncode
		token.child = child
		token.value = strings.ToLower(strings.TrimSpace(patternOption(options, 1)))
	case normalized == "equals" || normalized == "equalsignorecase":
		if len(options) < 3 {
			return fmt.Errorf("goark-log: %s pattern converter requires pattern, test and substitution", converter)
		}
		child, err := NewPatternLayout(options[0])
		if err != nil {
			return err
		}
		token.kind = tokenEquals
		token.child = child
		token.value = options[1]
		token.repl = options[2]
		token.ignore = normalized == "equalsignorecase"
	case normalized == "maxlen" || normalized == "maxlength":
		if len(options) < 2 {
			return fmt.Errorf("goark-log: maxLen pattern converter requires pattern and length")
		}
		child, err := NewPatternLayout(options[0])
		if err != nil {
			return err
		}
		limit, err := strconv.Atoi(strings.TrimSpace(options[1]))
		if err != nil || limit < 0 {
			return fmt.Errorf("goark-log: maxLen pattern length %q is invalid", options[1])
		}
		token.kind = tokenMaxLen
		token.child = child
		token.repeat = limit
	case normalized == "repeat":
		if len(options) < 2 {
			return fmt.Errorf("goark-log: repeat pattern converter requires pattern and count")
		}
		child, err := NewPatternLayout(options[0])
		if err != nil {
			return err
		}
		count, err := strconv.Atoi(strings.TrimSpace(options[1]))
		if err != nil || count < 0 {
			return fmt.Errorf("goark-log: repeat pattern count %q is invalid", options[1])
		}
		token.kind = tokenRepeat
		token.child = child
		token.repeat = count
	default:
		return fmt.Errorf("goark-log: unsupported pattern converter %q", converter)
	}
	return nil
}

func appendPatternToken(buf *bytes.Buffer, token patternToken, event Event, caller *callerCache) {
	if token.kind == tokenLiteral {
		buf.WriteString(token.literal)
		return
	}
	if token.kind == tokenAttrs && token.minWidth == 0 && token.maxWidth == 0 {
		appendPatternAttrs(buf, event.Attrs)
		return
	}
	if token.kind == tokenNewline && token.minWidth == 0 && token.maxWidth == 0 {
		buf.WriteByte('\n')
		return
	}
	if token.kind == tokenTime && token.minWidth == 0 && token.maxWidth == 0 {
		appendPatternTime(buf, token, event)
		return
	}
	if token.kind == tokenPID && token.minWidth == 0 && token.maxWidth == 0 {
		buf.WriteString(processIDString)
		return
	}
	value := patternTokenString(token, event, caller)
	appendPadded(buf, value, token.minWidth, token.maxWidth, token.leftAlign)
}

func appendPatternTime(buf *bytes.Buffer, token patternToken, event Event) {
	when := event.Time
	if when.IsZero() {
		when = time.Now()
	}
	switch token.timeUnix {
	case timeUnixSeconds:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.Unix(), 10))
	case timeUnixMillis:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMilli(), 10))
	case timeUnixMicros:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMicro(), 10))
	case timeUnixNanos:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixNano(), 10))
	default:
		buf.Write(when.AppendFormat(buf.AvailableBuffer(), token.format))
	}
}

func patternTokenString(token patternToken, event Event, caller *callerCache) string {
	switch token.kind {
	case tokenTime:
		when := event.Time
		if when.IsZero() {
			when = time.Now()
		}
		switch token.timeUnix {
		case timeUnixSeconds:
			return strconv.FormatInt(when.Unix(), 10)
		case timeUnixMillis:
			return strconv.FormatInt(when.UnixMilli(), 10)
		case timeUnixMicros:
			return strconv.FormatInt(when.UnixMicro(), 10)
		case timeUnixNanos:
			return strconv.FormatInt(when.UnixNano(), 10)
		default:
			return when.Format(token.format)
		}
	case tokenLevel:
		return levelName(event.Level)
	case tokenPID:
		return processIDString
	case tokenThread:
		return eventThreadName(event)
	case tokenLogger:
		return loggerNameWithPrecision(event.Logger, token.precision)
	case tokenMessage:
		return event.Message
	case tokenAttr:
		value, ok := event.Attr(token.key)
		if !ok {
			return ""
		}
		return attrValueString(value)
	case tokenAttrs:
		var attrBuf bytes.Buffer
		appendPatternAttrs(&attrBuf, event.Attrs)
		return attrBuf.String()
	case tokenError:
		return eventErrorStringWithOption(event, token.key)
	case tokenNewline:
		return "\n"
	case tokenMarker:
		return eventMarkerString(event)
	case tokenContextStack:
		return contextStackString(event.ContextStack)
	case tokenCallerClass:
		return caller.resolve(event).class
	case tokenCallerMethod:
		return caller.resolve(event).method
	case tokenCallerFile:
		return caller.resolve(event).file
	case tokenCallerLine:
		frame := caller.resolve(event)
		if frame.line == 0 {
			return ""
		}
		return strconv.Itoa(frame.line)
	case tokenCallerLocation:
		return caller.resolve(event).location()
	case tokenUUID:
		return newPatternUUID()
	case tokenSequence:
		return strconv.FormatUint(patternSequence.Add(1), 10)
	case tokenSubPattern:
		return formatChildPattern(token.child, event)
	case tokenHighlight:
		return applyANSIStyle(formatChildPattern(token.child, event), highlightStyle(event.Level))
	case tokenStyle:
		return applyANSIStyle(formatChildPattern(token.child, event), token.value)
	case tokenNotEmpty:
		value := formatChildPattern(token.child, event)
		if strings.TrimSpace(value) == "" {
			return ""
		}
		return value
	case tokenReplace:
		return token.regex.ReplaceAllString(formatChildPattern(token.child, event), token.repl)
	case tokenEncode:
		return encodePatternValue(formatChildPattern(token.child, event), token.value)
	case tokenEquals:
		value := formatChildPattern(token.child, event)
		matched := value == token.value
		if token.ignore {
			matched = strings.EqualFold(value, token.value)
		}
		if matched {
			return token.repl
		}
		return value
	case tokenMaxLen:
		return maxPatternLength(formatChildPattern(token.child, event), token.repeat)
	case tokenRepeat:
		return strings.Repeat(formatChildPattern(token.child, event), token.repeat)
	default:
		return ""
	}
}

func firstPatternOption(options []string) string {
	return patternOption(options, 0)
}

func patternOption(options []string, index int) string {
	if index < 0 || index >= len(options) {
		return ""
	}
	return options[index]
}

func parsePatternPrecision(option string) int {
	value, err := strconv.Atoi(strings.TrimSpace(option))
	if err != nil || value <= 0 {
		return 0
	}
	return value
}

func loggerNameWithPrecision(name string, precision int) string {
	if precision <= 0 || name == "" {
		return name
	}
	parts := strings.Split(name, ".")
	if precision >= len(parts) {
		return name
	}
	return strings.Join(parts[len(parts)-precision:], ".")
}

func formatChildPattern(layout *PatternLayout, event Event) string {
	if layout == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		return ""
	}
	return buf.String()
}

func newPatternUUID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	var out [36]byte
	hex := "0123456789abcdef"
	source := 0
	for index := 0; index < len(out); index++ {
		switch index {
		case 8, 13, 18, 23:
			out[index] = '-'
		default:
			if index == 14 {
				out[index] = '4'
				source++
				continue
			}
			b := value[source/2]
			if source%2 == 0 {
				out[index] = hex[b>>4]
			} else {
				out[index] = hex[b&0x0f]
			}
			source++
		}
	}
	return string(out[:])
}

type callerCache struct {
	loaded bool
	frame  callerFrame
}

type callerFrame struct {
	class  string
	method string
	file   string
	line   int
}

func (c *callerCache) resolve(event Event) callerFrame {
	if c == nil {
		return callerFrameFromPC(event.PC)
	}
	if !c.loaded {
		c.frame = callerFrameFromPC(event.PC)
		c.loaded = true
	}
	return c.frame
}

func callerFrameFromPC(pc uintptr) callerFrame {
	if pc == 0 {
		return callerFrame{}
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return callerFrame{}
	}
	file, line := fn.FileLine(pc)
	name := fn.Name()
	return callerFrame{
		class:  callerClassName(name),
		method: callerMethodName(name),
		file:   baseName(file),
		line:   line,
	}
}

func (f callerFrame) location() string {
	if f.method == "" && f.file == "" && f.line == 0 {
		return ""
	}
	if f.line == 0 {
		return f.method + "(" + f.file + ")"
	}
	return f.method + "(" + f.file + ":" + strconv.Itoa(f.line) + ")"
}

func callerClassName(name string) string {
	index := strings.LastIndexByte(name, '.')
	if index < 0 {
		return name
	}
	return name[:index]
}

func callerMethodName(name string) string {
	index := strings.LastIndexByte(name, '.')
	if index < 0 || index == len(name)-1 {
		return name
	}
	return name[index+1:]
}

func baseName(path string) string {
	index := strings.LastIndexAny(path, `/\`)
	if index < 0 || index == len(path)-1 {
		return path
	}
	return path[index+1:]
}

func normalizeTimePattern(format string) (string, timeUnixMode) {
	switch strings.ToUpper(strings.TrimSpace(format)) {
	case "", "DEFAULT", "ISO8601", "ISO8601_OFFSET_DATE_TIME":
		return defaultTimeFormat, timeUnixNone
	case "RFC3339":
		return time.RFC3339, timeUnixNone
	case "RFC3339NANO":
		return time.RFC3339Nano, timeUnixNone
	case "UNIX", "UNIX_SECONDS":
		return "", timeUnixSeconds
	case "UNIX_MILLIS", "UNIX_MS":
		return "", timeUnixMillis
	case "UNIX_MICROS", "UNIX_US":
		return "", timeUnixMicros
	case "UNIX_NANOS", "UNIX_NS":
		return "", timeUnixNanos
	default:
		return javaDatePatternToGo(format), timeUnixNone
	}
}

func javaDatePatternToGo(format string) string {
	replacer := strings.NewReplacer(
		"yyyy", "2006",
		"yy", "06",
		"MM", "01",
		"dd", "02",
		"HH", "15",
		"mm", "04",
		"ss", "05",
		"SSSSSS", "000000",
		"SSS", "000",
		"XXX", "Z07:00",
		"XX", "-0700",
		"X", "-07",
	)
	return replacer.Replace(format)
}

func eventErrorString(event Event) string {
	return eventErrorStringWithOption(event, "")
}

func eventErrorStringWithOption(event Event, option string) string {
	if event.Throwable != nil {
		if option == "short" {
			return event.Throwable.Message
		}
		return event.Throwable.String()
	}
	for _, key := range []string{"error", "err"} {
		value, ok := event.Attr(key)
		if ok {
			return attrValueString(value)
		}
	}
	return ""
}

func encodePatternValue(value string, mode string) string {
	switch mode {
	case "", "json":
		quoted := strconv.Quote(value)
		return strings.TrimSuffix(strings.TrimPrefix(quoted, `"`), `"`)
	case "html":
		return html.EscapeString(value)
	case "xml":
		return html.EscapeString(value)
	case "crlf":
		value = strings.ReplaceAll(value, "\r", `\r`)
		return strings.ReplaceAll(value, "\n", `\n`)
	default:
		return value
	}
}

func highlightStyle(level slog.Level) string {
	switch {
	case level >= LevelFatal:
		return "red,bold"
	case level >= slog.LevelError:
		return "red"
	case level >= slog.LevelWarn:
		return "yellow"
	case level >= slog.LevelInfo:
		return "green"
	case level >= slog.LevelDebug:
		return "cyan"
	default:
		return "faint"
	}
}

func applyANSIStyle(value string, style string) string {
	if value == "" {
		return ""
	}
	start := ansiStyleStart(style)
	if start == "" {
		return value
	}
	return start + value + "\x1b[0m"
}

func ansiStyleStart(style string) string {
	codes := ansiStyleCodes(style)
	if len(codes) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(codes, ";") + "m"
}

func ansiStyleCodes(style string) []string {
	parts := strings.FieldsFunc(style, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	codes := make([]string, 0, len(parts))
	for _, part := range parts {
		code, ok := ansiStyleCode(part)
		if ok {
			codes = append(codes, code)
		}
	}
	return codes
}

func ansiStyleCode(value string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, "_", "")
	switch normalized {
	case "bold":
		return "1", true
	case "dim", "faint":
		return "2", true
	case "underline":
		return "4", true
	case "blink":
		return "5", true
	case "reverse":
		return "7", true
	case "black":
		return "30", true
	case "red":
		return "31", true
	case "green":
		return "32", true
	case "yellow":
		return "33", true
	case "blue":
		return "34", true
	case "magenta", "purple":
		return "35", true
	case "cyan":
		return "36", true
	case "white":
		return "37", true
	case "brightblack", "gray", "grey":
		return "90", true
	case "brightred":
		return "91", true
	case "brightgreen":
		return "92", true
	case "brightyellow":
		return "93", true
	case "brightblue":
		return "94", true
	case "brightmagenta", "brightpurple":
		return "95", true
	case "brightcyan":
		return "96", true
	case "brightwhite":
		return "97", true
	case "bgblack", "backgroundblack":
		return "40", true
	case "bgred", "backgroundred":
		return "41", true
	case "bggreen", "backgroundgreen":
		return "42", true
	case "bgyellow", "backgroundyellow":
		return "43", true
	case "bgblue", "backgroundblue":
		return "44", true
	case "bgmagenta", "bgpurple", "backgroundmagenta", "backgroundpurple":
		return "45", true
	case "bgcyan", "backgroundcyan":
		return "46", true
	case "bgwhite", "backgroundwhite":
		return "47", true
	default:
		return "", false
	}
}

func maxPatternLength(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func eventMarkerString(event Event) string {
	if event.Marker != nil {
		return event.Marker.String()
	}
	for _, key := range []string{"marker", "goark.marker"} {
		value, ok := event.Attr(key)
		if ok {
			return attrValueString(value)
		}
	}
	return ""
}

func eventThreadName(event Event) string {
	if strings.TrimSpace(event.ThreadName) != "" {
		return strings.TrimSpace(event.ThreadName)
	}
	for _, key := range []string{"goark.thread", "thread", "goroutine"} {
		value, ok := event.Attr(key)
		if ok {
			name := strings.TrimSpace(attrValueString(value))
			if name != "" {
				return name
			}
		}
	}
	return defaultThreadName
}

func appendPadded(buf *bytes.Buffer, value string, minWidth int, maxWidth int, leftAlign bool) {
	if maxWidth > 0 && len(value) > maxWidth {
		value = value[:maxWidth]
	}
	if minWidth <= len(value) {
		buf.WriteString(value)
		return
	}
	padding := minWidth - len(value)
	if leftAlign {
		buf.WriteString(value)
		writeSpaces(buf, padding)
		return
	}
	writeSpaces(buf, padding)
	buf.WriteString(value)
}

func writeSpaces(buf *bytes.Buffer, count int) {
	for index := 0; index < count; index++ {
		buf.WriteByte(' ')
	}
}

func isPatternDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func isPatternLetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func isSpaceByte(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func appendPatternAttrs(buf *bytes.Buffer, attrs []slog.Attr) {
	for _, attr := range attrs {
		appendPatternKeyValueAttr(buf, attr.Key, attr.Value)
	}
}

func appendPatternKeyValueAttr(buf *bytes.Buffer, key string, value slog.Value) {
	if buf.Len() > 0 {
		data := buf.Bytes()
		if !isSpaceByte(data[len(data)-1]) {
			buf.WriteByte(' ')
		}
	}
	buf.WriteString(key)
	buf.WriteByte('=')
	appendTextValue(buf, value)
}

func appendKeyValue(buf *bytes.Buffer, key string, value string) {
	appendKey(buf, key)
	quoteValue(buf, value)
}

func appendKeyValueAttr(buf *bytes.Buffer, key string, value slog.Value) {
	appendKey(buf, key)
	appendTextValue(buf, value)
}

func appendKey(buf *bytes.Buffer, key string) {
	if buf.Len() > 0 {
		buf.WriteByte(' ')
	}
	buf.WriteString(key)
	buf.WriteByte('=')
}

func quoteValue(buf *bytes.Buffer, value string) {
	if value == "" || strings.ContainsAny(value, " \t\r\n\"=") {
		buf.Write(strconv.AppendQuote(buf.AvailableBuffer(), value))
		return
	}
	buf.WriteString(value)
}

func appendTextValue(buf *bytes.Buffer, value slog.Value) {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		quoteValue(buf, value.String())
	case slog.KindBool:
		if value.Bool() {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case slog.KindInt64:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), value.Int64(), 10))
	case slog.KindUint64:
		buf.Write(strconv.AppendUint(buf.AvailableBuffer(), value.Uint64(), 10))
	case slog.KindFloat64:
		buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), value.Float64(), 'g', -1, 64))
	case slog.KindDuration:
		quoteValue(buf, value.Duration().String())
	case slog.KindTime:
		buf.Write(value.Time().AppendFormat(buf.AvailableBuffer(), time.RFC3339Nano))
	case slog.KindGroup:
		quoteValue(buf, attrValueString(value))
	case slog.KindAny:
		appendTextAny(buf, value.Any())
	default:
		quoteValue(buf, attrValueString(value))
	}
}

func appendTextAny(buf *bytes.Buffer, value any) {
	switch typed := value.(type) {
	case nil:
		buf.WriteString("<nil>")
	case string:
		quoteValue(buf, typed)
	case error:
		quoteValue(buf, typed.Error())
	case fmt.Stringer:
		quoteValue(buf, typed.String())
	default:
		quoteValue(buf, fmt.Sprint(typed))
	}
}

func appendJSONFieldString(buf *bytes.Buffer, key string, value string, comma bool) {
	appendJSONKey(buf, key, comma)
	appendJSONString(buf, value)
}

func appendJSONFieldValue(buf *bytes.Buffer, key string, value slog.Value, comma bool) {
	appendJSONKey(buf, key, comma)
	appendJSONValue(buf, value)
}

func appendJSONFieldTime(buf *bytes.Buffer, key string, value time.Time, layout string, comma bool) {
	appendJSONKey(buf, key, comma)
	buf.WriteByte('"')
	buf.Write(value.AppendFormat(buf.AvailableBuffer(), layout))
	buf.WriteByte('"')
}

func appendJSONKey(buf *bytes.Buffer, key string, comma bool) {
	if comma {
		buf.WriteByte(',')
	}
	appendJSONString(buf, key)
	buf.WriteByte(':')
}

func appendJSONString(buf *bytes.Buffer, value string) {
	buf.Write(strconv.AppendQuote(buf.AvailableBuffer(), value))
}

func appendJSONValue(buf *bytes.Buffer, value slog.Value) {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		appendJSONString(buf, value.String())
	case slog.KindBool:
		if value.Bool() {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case slog.KindInt64:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), value.Int64(), 10))
	case slog.KindUint64:
		buf.Write(strconv.AppendUint(buf.AvailableBuffer(), value.Uint64(), 10))
	case slog.KindFloat64:
		buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), value.Float64(), 'g', -1, 64))
	case slog.KindDuration:
		appendJSONString(buf, value.Duration().String())
	case slog.KindTime:
		buf.WriteByte('"')
		buf.Write(value.Time().AppendFormat(buf.AvailableBuffer(), time.RFC3339Nano))
		buf.WriteByte('"')
	case slog.KindGroup:
		buf.WriteByte('{')
		for index, attr := range value.Group() {
			appendJSONFieldValue(buf, attr.Key, attr.Value, index > 0)
		}
		buf.WriteByte('}')
	case slog.KindAny:
		appendJSONAny(buf, value.Any())
	default:
		appendJSONString(buf, attrValueString(value))
	}
}

func appendJSONAny(buf *bytes.Buffer, value any) {
	switch typed := value.(type) {
	case nil:
		buf.WriteString("null")
	case string:
		appendJSONString(buf, typed)
	case error:
		appendJSONString(buf, typed.Error())
	case fmt.Stringer:
		appendJSONString(buf, typed.String())
	default:
		data, err := jsoncodec.Marshal(typed)
		if err != nil {
			appendJSONString(buf, fmt.Sprint(typed))
			return
		}
		buf.Write(data)
	}
}

func appendJSONAttrsListField(buf *bytes.Buffer, key string, attrs []slog.Attr, comma bool) {
	appendJSONKey(buf, key, comma)
	appendJSONAttrsList(buf, attrs)
}

func appendJSONAttrsList(buf *bytes.Buffer, attrs []slog.Attr) {
	buf.WriteByte('[')
	for index, attr := range attrs {
		if index > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('{')
		appendJSONFieldString(buf, "key", attr.Key, false)
		appendJSONKey(buf, "value", true)
		appendJSONValue(buf, attr.Value)
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
}

func attrValueString(value slog.Value) string {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		return value.String()
	case slog.KindBool:
		if value.Bool() {
			return "true"
		}
		return "false"
	case slog.KindInt64:
		return strconv.FormatInt(value.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(value.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(value.Float64(), 'g', -1, 64)
	case slog.KindTime:
		return value.Time().Format(time.RFC3339Nano)
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindGroup:
		var builder strings.Builder
		for index, attr := range value.Group() {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(attr.Key)
			builder.WriteByte('=')
			builder.WriteString(attrValueString(attr.Value))
		}
		return builder.String()
	default:
		return fmt.Sprint(value.Any())
	}
}

func throwableStackString(throwable *Throwable) string {
	if throwable == nil {
		return ""
	}
	var builder strings.Builder
	appendThrowableStackString(&builder, throwable)
	return builder.String()
}

func appendThrowableStackString(builder *strings.Builder, throwable *Throwable) {
	if throwable == nil {
		return
	}
	if throwable.Type != "" {
		builder.WriteString(throwable.Type)
		builder.WriteString(": ")
	}
	builder.WriteString(throwable.Message)
	for _, frame := range throwable.Stack {
		builder.WriteString("\n\tat ")
		builder.WriteString(frame)
	}
	if throwable.Cause != nil {
		builder.WriteString("\nCaused by: ")
		appendThrowableStackString(builder, throwable.Cause)
	}
}
