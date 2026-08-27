package goarklog

import (
	"strings"
)

func throwableStackString(throwable *Throwable) string {
	if throwable == nil {
		return ""
	}
	var builder strings.Builder
	appendThrowableStackString(&builder, throwable)
	return builder.String()
}

func appendThrowableStackString(builder *strings.Builder, throwable *Throwable) {
	if throwable == nil {
		return
	}
	if throwable.Type != "" {
		builder.WriteString(throwable.Type)
		builder.WriteString(": ")
	}
	builder.WriteString(throwable.Message)
	for _, frame := range throwable.Stack {
		builder.WriteString("\n\tat ")
		builder.WriteString(frame)
	}
	if throwable.Cause != nil {
		builder.WriteString("\nCaused by: ")
		appendThrowableStackString(builder, throwable.Cause)
	}
}
