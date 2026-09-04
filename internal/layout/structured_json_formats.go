package layout

import (
	"log/slog"
	"os"
	"strings"

	"goark.dev/log/internal/layoutsupport"
)

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
