package goarklog

import "log/slog"

// New 创建默认命名 logger 和对应 Handler。
func New(options Options) (*slog.Logger, *Handler, error) {
	handler, err := NewHandler(options)
	if err != nil {
		return nil, nil, err
	}
	return NewLogger(handler, defaultLoggerName), handler, nil
}

// NewDefault 创建默认 stderr INFO logger。
func NewDefault() (*slog.Logger, *Handler) {
	handler := NewDefaultHandler()
	return NewLogger(handler, defaultLoggerName), handler
}

// NewLogger 基于 handler 创建命名 logger。
func NewLogger(handler slog.Handler, name string) *slog.Logger {
	return slog.New(handler).With(loggerNameKey, name)
}

// WithName 返回带有 goark-log logger 名称的 logger。
func WithName(logger *slog.Logger, name string) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return logger.With(loggerNameKey, name)
}
