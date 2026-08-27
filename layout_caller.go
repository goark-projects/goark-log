package goarklog

import (
	"runtime"
	"strconv"
	"strings"
	"time"
)

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
