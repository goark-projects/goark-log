package goarklog

import (
	"os"
	"strings"
	"time"
)

var hostNameString = resolveHostName()

func eventTime(when time.Time) time.Time {
	if when.IsZero() {
		return time.Now()
	}
	return when
}

func resolveHostName() string {
	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		return "localhost"
	}
	return strings.TrimSpace(name)
}
