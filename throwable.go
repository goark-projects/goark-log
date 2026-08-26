package goarklog

import (
	"errors"
	"log/slog"
	"reflect"
)

const (
	// ThrowableAttrKey 是 goark-log 标准异常属性键。
	ThrowableAttrKey = "goark.throwable"
)

// Throwable 是 Go error 的 Log4j2 Throwable 快照。
type Throwable struct {
	Type    string
	Message string
	Cause   *Throwable
}

// NewThrowable 把 error 转成轻量快照，不主动采集调用栈。
func NewThrowable(err error) *Throwable {
	if err == nil {
		return nil
	}
	throwable := &Throwable{
		Type:    reflect.TypeOf(err).String(),
		Message: err.Error(),
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
