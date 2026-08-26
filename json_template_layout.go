package goarklog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

const defaultJSONEventTemplate = `{
  "timestamp": {"$resolver": "timestamp"},
  "level": {"$resolver": "level"},
  "loggerName": {"$resolver": "logger"},
  "message": {"$resolver": "message"},
  "thread": {"$resolver": "thread"},
  "marker": {"$resolver": "marker"},
  "thrown": {"$resolver": "throwable"},
  "contextStack": {"$resolver": "contextStack"},
  "endOfBatch": {"$resolver": "endOfBatch"},
  "contextMap": {"$resolver": "mdc"}
}`

// JSONTemplateLayout 按 Log4j2 JSON Template 风格输出事件。
type JSONTemplateLayout struct {
	fields []jsonTemplateField
}

type jsonTemplateField struct {
	key      string
	resolver jsonTemplateResolver
}

type jsonTemplateResolver interface {
	AppendJSON(buf *bytes.Buffer, event Event)
}

type jsonTemplateRawField struct {
	key string
	raw json.RawMessage
}

// NewJSONTemplateLayout 编译 JSON 事件模板。
func NewJSONTemplateLayout(template string) (*JSONTemplateLayout, error) {
	if strings.TrimSpace(template) == "" {
		template = defaultJSONEventTemplate
	}
	rawFields, err := decodeJSONTemplateRawFields(template)
	if err != nil {
		return nil, fmt.Errorf("goark-log: parse JSON template layout: %w", err)
	}
	if len(rawFields) == 0 {
		return nil, fmt.Errorf("goark-log: JSON template layout requires at least one field")
	}
	fields := make([]jsonTemplateField, 0, len(rawFields))
	for _, rawField := range rawFields {
		resolver, err := compileJSONTemplateResolver(rawField.raw)
		if err != nil {
			return nil, fmt.Errorf("goark-log: JSON template field %q: %w", rawField.key, err)
		}
		fields = append(fields, jsonTemplateField{key: rawField.key, resolver: resolver})
	}
	return &JSONTemplateLayout{fields: fields}, nil
}

func decodeJSONTemplateRawFields(template string) ([]jsonTemplateRawField, error) {
	decoder := json.NewDecoder(strings.NewReader(template))
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return nil, fmt.Errorf("event template must be a JSON object")
	}
	fields := make([]jsonTemplateRawField, 0, 8)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("event template field key must be a string")
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, err
		}
		fields = append(fields, jsonTemplateRawField{key: key, raw: append([]byte(nil), raw...)})
	}
	token, err = decoder.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '}' {
		return nil, fmt.Errorf("event template object is not closed")
	}
	if token, err = decoder.Token(); err != io.EOF {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("event template has trailing token %v", token)
	}
	return fields, nil
}

func (l *JSONTemplateLayout) Format(buf *bytes.Buffer, event Event) error {
	if l == nil {
		return JSONLayout{}.Format(buf, event)
	}
	buf.WriteByte('{')
	for index, field := range l.fields {
		appendJSONKey(buf, field.key, index > 0)
		field.resolver.AppendJSON(buf, event)
	}
	buf.WriteString("}\n")
	return nil
}

func compileJSONTemplateResolver(raw json.RawMessage) (jsonTemplateResolver, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err == nil {
		if resolverRaw, ok := object["$resolver"]; ok {
			var name string
			if err := json.Unmarshal(resolverRaw, &name); err != nil {
				return nil, fmt.Errorf("$resolver must be a string")
			}
			return newJSONTemplateResolver(name, object)
		}
	}
	return rawJSONResolver{raw: append([]byte(nil), raw...)}, nil
}

func newJSONTemplateResolver(name string, options map[string]json.RawMessage) (jsonTemplateResolver, error) {
	switch normalizeKind(name) {
	case "timestamp", "time":
		format := jsonTemplateStringOption(options, "format")
		layout, unix := normalizeTimePattern(format)
		return timestampJSONResolver{layout: layout, unix: unix}, nil
	case "level":
		return levelJSONResolver{}, nil
	case "logger", "loggername":
		return loggerJSONResolver{}, nil
	case "message", "msg":
		return messageJSONResolver{}, nil
	case "thread":
		return threadJSONResolver{}, nil
	case "threadname":
		return threadJSONResolver{}, nil
	case "marker":
		return markerJSONResolver{}, nil
	case "throwable", "exception", "thrown":
		return throwableJSONResolver{}, nil
	case "source", "location":
		return sourceJSONResolver{}, nil
	case "process":
		return processJSONResolver{}, nil
	case "contextstack", "ndc":
		return contextStackJSONResolver{}, nil
	case "mdc", "contextmap", "attrs":
		return attrsJSONResolver{}, nil
	case "attr":
		key := jsonTemplateStringOption(options, "key")
		if strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("attr resolver requires key")
		}
		return attrJSONResolver{key: key}, nil
	case "endofbatch":
		return endOfBatchJSONResolver{}, nil
	default:
		return nil, fmt.Errorf("unsupported resolver %q", name)
	}
}

func jsonTemplateStringOption(options map[string]json.RawMessage, key string) string {
	raw, ok := options[key]
	if !ok {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

type rawJSONResolver struct {
	raw []byte
}

func (r rawJSONResolver) AppendJSON(buf *bytes.Buffer, _ Event) {
	if len(r.raw) == 0 {
		buf.WriteString("null")
		return
	}
	buf.Write(r.raw)
}

type timestampJSONResolver struct {
	layout string
	unix   timeUnixMode
}

func (r timestampJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	when := event.Time
	if when.IsZero() {
		when = time.Now()
	}
	switch r.unix {
	case timeUnixSeconds:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.Unix(), 10))
	case timeUnixMillis:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMilli(), 10))
	case timeUnixMicros:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMicro(), 10))
	case timeUnixNanos:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixNano(), 10))
	default:
		buf.WriteByte('"')
		buf.Write(when.AppendFormat(buf.AvailableBuffer(), r.layout))
		buf.WriteByte('"')
	}
}

type levelJSONResolver struct{}

func (levelJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	appendJSONString(buf, levelName(event.Level))
}

type loggerJSONResolver struct{}

func (loggerJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	appendJSONString(buf, event.Logger)
}

type messageJSONResolver struct{}

func (messageJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	appendJSONString(buf, event.Message)
}

type threadJSONResolver struct{}

func (threadJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	appendJSONString(buf, eventThreadName(event))
}

type markerJSONResolver struct{}

func (markerJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	if event.Marker == nil {
		buf.WriteString("null")
		return
	}
	appendJSONString(buf, event.Marker.String())
}

type throwableJSONResolver struct{}

func (throwableJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	value := eventErrorString(event)
	if value == "" {
		buf.WriteString("null")
		return
	}
	appendJSONString(buf, value)
}

type sourceJSONResolver struct{}

func (sourceJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	frame := callerFrameFromPC(event.PC)
	if frame.method == "" && frame.file == "" && frame.line == 0 {
		buf.WriteString("null")
		return
	}
	buf.WriteByte('{')
	appendJSONFieldString(buf, "class", frame.class, false)
	appendJSONFieldString(buf, "method", frame.method, true)
	appendJSONFieldString(buf, "file", frame.file, true)
	appendJSONKey(buf, "line", true)
	buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(frame.line), 10))
	appendJSONFieldString(buf, "location", frame.location(), true)
	buf.WriteByte('}')
}

type processJSONResolver struct{}

func (processJSONResolver) AppendJSON(buf *bytes.Buffer, _ Event) {
	buf.WriteByte('{')
	appendJSONKey(buf, "pid", false)
	buf.WriteString(processIDString)
	buf.WriteByte('}')
}

type contextStackJSONResolver struct{}

func (contextStackJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	buf.WriteByte('[')
	for index, value := range event.ContextStack {
		if index > 0 {
			buf.WriteByte(',')
		}
		appendJSONString(buf, value)
	}
	buf.WriteByte(']')
}

type attrsJSONResolver struct{}

func (attrsJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	appendJSONAttrsObject(buf, event.Attrs)
}

type attrJSONResolver struct {
	key string
}

func (r attrJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	value, ok := event.Attr(r.key)
	if !ok {
		buf.WriteString("null")
		return
	}
	appendJSONValue(buf, value)
}

type endOfBatchJSONResolver struct{}

func (endOfBatchJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	if event.EndOfBatch {
		buf.WriteString("true")
		return
	}
	buf.WriteString("false")
}

func appendJSONAttrsObject(buf *bytes.Buffer, attrs []slog.Attr) {
	buf.WriteByte('{')
	for index, attr := range attrs {
		appendJSONFieldValue(buf, attr.Key, attr.Value, index > 0)
	}
	buf.WriteByte('}')
}
