package goarklog

import (
	"strings"

	"goark.dev/log/internal/logvalue"
)

func eventMarkerString(event Event) string {
	if event.Marker != nil {
		return event.Marker.String()
	}
	for _, key := range []string{"marker", "goark.marker"} {
		value, ok := event.Attr(key)
		if ok {
			return logvalue.String(value)
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
			name := strings.TrimSpace(logvalue.String(value))
			if name != "" {
				return name
			}
		}
	}
	return defaultThreadName
}
