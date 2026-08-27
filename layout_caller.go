package goarklog

import (
	"runtime"
	"strconv"
	"strings"
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
