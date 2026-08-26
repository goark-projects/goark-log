package goarklog

import "context"

// AsyncErrorHandler 处理异步后台写入失败。
type AsyncErrorHandler interface {
	HandleAsyncError(ctx context.Context, err error, event Event)
}

// AsyncErrorHandlerFunc 把函数适配为 AsyncErrorHandler。
type AsyncErrorHandlerFunc func(ctx context.Context, err error, event Event)

// HandleAsyncError 执行异步错误处理函数。
func (f AsyncErrorHandlerFunc) HandleAsyncError(ctx context.Context, err error, event Event) {
	if f != nil {
		f(ctx, err, event)
	}
}
