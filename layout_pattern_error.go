package goarklog

import (
	"strings"

	"goark.dev/log/internal/logvalue"
)

func eventErrorString(event Event) string {
	return eventErrorStringWithOption(event, "")
}

func eventErrorStringWithOption(event Event, option string) string {
	option = strings.ToLower(strings.TrimSpace(option))
	if option == "none" {
		return ""
	}
	if event.Throwable != nil {
		return throwableStringWithPatternOption(event.Throwable, option)
	}
	if throwable := throwableFromAttrs(event.Attrs); throwable != nil {
		return throwableStringWithPatternOption(throwable, option)
	}
	for _, key := range []string{"error", "err"} {
		value, ok := event.Attr(key)
		if ok {
			return logvalue.String(value)
		}
	}
	return ""
}

func throwableStringWithPatternOption(throwable *Throwable, option string) string {
	if throwable == nil {
		return ""
	}
	switch option {
	case "none":
		return ""
	case "short":
		return throwable.Message
	case "full":
		return throwableStackString(throwable)
	default:
		return throwable.String()
	}
}
