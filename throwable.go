package goarklog

import (
	"errors"
	"log/slog"
	"reflect"
	"runtime"
	"strconv"
)

const (
	// ThrowableAttrKey 是 goark-log 标准异常属性键。
	ThrowableAttrKey = "goark.throwable"
)

// Throwable 是 Go error 的异常快照。
type Throwable struct {
	Type    string
	Message string
	Cause   *Throwable
	Stack   []string
}

// NewThrowable 把 error 转成轻量快照，不主动采集调用栈。
func NewThrowable(err error) *Throwable {
	return newThrowable(err, false, 0)
}

// NewThrowableWithStack 把 error 转成包含调用栈的快照。
func NewThrowableWithStack(err error) *Throwable {
	return newThrowable(err, true, 1)
}

func newThrowable(err error, withStack bool, skip int) *Throwable {
	if err == nil {
		return nil
	}
	throwable := &Throwable{
		Type:    reflect.TypeOf(err).String(),
		Message: err.Error(),
	}
	if withStack {
		throwable.Stack = captureThrowableStack(skip + 1)
	}
	if cause := errors.Unwrap(err); cause != nil && cause != err {
		throwable.Cause = NewThrowable(cause)
	}
	return throwable
}

func (t *Throwable) String() string {
	if t == nil {
		return ""
	}
	return t.Message
}

// ThrowableAttr 把 error 按标准异常属性键注入 slog 记录。
func ThrowableAttr(err error) slog.Attr {
	return slog.Any(ThrowableAttrKey, NewThrowable(err))
}

// ThrowableWithStackAttr 把 error 和当前调用栈注入 slog 记录。
func ThrowableWithStackAttr(err error) slog.Attr {
	return slog.Any(ThrowableAttrKey, NewThrowableWithStack(err))
}

func throwableFromAttrs(attrs []slog.Attr) *Throwable {
	for index := len(attrs) - 1; index >= 0; index-- {
		attr := attrs[index]
		switch attr.Key {
		case ThrowableAttrKey, "throwable", "error", "err":
			if throwable := throwableFromValue(attr.Value); throwable != nil {
				return throwable
			}
		}
	}
	return nil
}

func throwableFromValue(value slog.Value) *Throwable {
	value = value.Resolve()
	switch typed := value.Any().(type) {
	case nil:
		return nil
	case *Throwable:
		return typed
	case Throwable:
		return &typed
	case error:
		return NewThrowable(typed)
	default:
		text := attrValueString(value)
		if text == "" {
			return nil
		}
		return &Throwable{Type: "string", Message: text}
	}
}

func captureThrowableStack(skip int) []string {
	var pcs [32]uintptr
	count := runtime.Callers(skip+2, pcs[:])
	if count == 0 {
		return nil
	}
	frames := runtime.CallersFrames(pcs[:count])
	stack := make([]string, 0, count)
	for {
		frame, more := frames.Next()
		location := baseName(frame.File)
		if frame.Line > 0 {
			location += ":" + strconv.Itoa(frame.Line)
		}
		if frame.Function != "" {
			location = frame.Function + "(" + location + ")"
		}
		stack = append(stack, location)
		if !more {
			break
		}
	}
	return stack
}
