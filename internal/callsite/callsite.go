package callsite

import (
	"runtime"
	"strconv"
	"strings"
)

// Cache 缓存单个事件的调用点解析结果，避免同一布局多次解析 PC。
type Cache struct {
	loaded bool
	frame  Frame
}

// Frame 描述从 runtime PC 解析出的调用点。
type Frame struct {
	Class  string
	Method string
	File   string
	Line   int
}

// ResolvePC 返回缓存后的调用点。
func (c *Cache) ResolvePC(pc uintptr) Frame {
	if c == nil {
		return FrameFromPC(pc)
	}
	if !c.loaded {
		c.frame = FrameFromPC(pc)
		c.loaded = true
	}
	return c.frame
}

// FrameFromPC 从 runtime PC 解析调用点。
func FrameFromPC(pc uintptr) Frame {
	if pc == 0 {
		return Frame{}
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return Frame{}
	}
	file, line := fn.FileLine(pc)
	name := fn.Name()
	return Frame{
		Class:  className(name),
		Method: methodName(name),
		File:   BaseName(file),
		Line:   line,
	}
}

// IsZero 判断调用点是否为空。
func (f Frame) IsZero() bool {
	return f.Method == "" && f.File == "" && f.Line == 0
}

// Location 返回 Log4j 风格的 method(file:line) 调用位置。
func (f Frame) Location() string {
	if f.IsZero() {
		return ""
	}
	if f.Line == 0 {
		return f.Method + "(" + f.File + ")"
	}
	return f.Method + "(" + f.File + ":" + strconv.Itoa(f.Line) + ")"
}

func className(name string) string {
	index := strings.LastIndexByte(name, '.')
	if index < 0 {
		return name
	}
	return name[:index]
}

func methodName(name string) string {
	index := strings.LastIndexByte(name, '.')
	if index < 0 || index == len(name)-1 {
		return name
	}
	return name[index+1:]
}

// BaseName 返回路径最后一段，同时兼容 Unix 和 Windows 分隔符。
func BaseName(path string) string {
	index := strings.LastIndexAny(path, `/\`)
	if index < 0 || index == len(path)-1 {
		return path
	}
	return path[index+1:]
}
