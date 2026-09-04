package layout

import (
	"log/slog"
	"os"
	"sort"
	"strings"

	"goark.dev/log/internal/layoutsupport"
	"goark.dev/log/internal/logcontext"
)

func (l *StructuredJSONLayout) appendECS(writer *structuredWriter, event Event) {
	writer.Add("@timestamp", slog.TimeValue(layoutsupport.EventTime(event.Time).UTC()))
	if writer.beginObject("log", "log") {
		writer.addPath("log.level", "level", slog.StringValue(levelName(event.Level)))
		writer.addPath("log.logger", "logger", slog.StringValue(event.Logger))
		writer.endObject()
	}
	if writer.beginObject("process", "process") {
		writer.addPath("process.pid", "pid", slog.IntValue(os.Getpid()))
		if writer.beginObject("process.thread", "thread") {
			writer.addPath("process.thread.name", "name", slog.StringValue(eventThreadName(event)))
			writer.endObject()
		}
		writer.endObject()
	}
	serviceConfigured := strings.TrimSpace(l.ecs.ServiceName) != "" || strings.TrimSpace(l.ecs.ServiceVersion) != "" ||
		strings.TrimSpace(l.ecs.ServiceEnvironment) != "" || strings.TrimSpace(l.ecs.ServiceNodeName) != ""
	if writer.beginObjectIf(serviceConfigured, "service", "service") {
		writer.addNonEmptyPath("service.name", "name", l.ecs.ServiceName)
		writer.addNonEmptyPath("service.version", "version", l.ecs.ServiceVersion)
		writer.addNonEmptyPath("service.environment", "environment", l.ecs.ServiceEnvironment)
		if writer.beginObjectIf(strings.TrimSpace(l.ecs.ServiceNodeName) != "", "service.node", "node") {
			writer.addNonEmptyPath("service.node.name", "name", l.ecs.ServiceNodeName)
			writer.endObject()
		}
		writer.endObject()
	}
	writer.Add("message", slog.StringValue(event.Message))
	if writer.beginObject("ecs", "ecs") {
		writer.addPath("ecs.version", "version", slog.StringValue("8.11"))
		writer.endObject()
	}
	if event.Throwable != nil {
		if writer.beginObject("error", "error") {
			writer.addNonEmptyPath("error.type", "type", event.Throwable.Type)
			writer.addNonEmptyPath("error.message", "message", event.Throwable.Message)
			writer.addPath("error.stack_trace", "stack_trace", slog.StringValue(formatStructuredStacktrace(event.Throwable, l.stacktrace)))
			writer.endObject()
		}
	}
	writer.addMarkerTags(event)
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
		fullMessage := event.Message + "\n\n" + formatStructuredStacktrace(event.Throwable, l.stacktrace)
		writer.Add("full_message", slog.StringValue(fullMessage))
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
	writer.addMarkerTags(event)
	if event.Throwable != nil {
		writer.Add("stack_trace", slog.StringValue(formatStructuredStacktrace(event.Throwable, l.stacktrace)))
	}
}

func (w *structuredWriter) addMarkerTags(event Event) {
	if event.Marker == nil {
		return
	}
	if len(event.Marker.Parents) == 0 {
		if event.Marker.Name != "" {
			w.addStrings("tags", "tags", []string{event.Marker.Name})
		}
		return
	}
	names := make([]string, 0, 1+len(event.Marker.Parents))
	seen := make(map[string]struct{}, 1+len(event.Marker.Parents))
	appendMarkerNames(&names, seen, *event.Marker)
	if len(names) > 0 {
		sort.Strings(names)
		w.addStrings("tags", "tags", names)
	}
}

func appendMarkerNames(names *[]string, seen map[string]struct{}, marker logcontext.Marker) {
	if marker.Name != "" {
		if _, exists := seen[marker.Name]; !exists {
			seen[marker.Name] = struct{}{}
			*names = append(*names, marker.Name)
		}
	}
	for _, parent := range marker.Parents {
		appendMarkerNames(names, seen, parent)
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
