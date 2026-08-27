package goarklog

import (
	"strings"
)

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
