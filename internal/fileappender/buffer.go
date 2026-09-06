package fileappender

import (
	"bytes"
	"sync"

	internallayout "goark.dev/log/internal/layout"
	"goark.dev/log/internal/logevent"
)

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// Event 是文件输出端处理的事件快照。
type Event = logevent.Event

// Layout 是文件输出端依赖的布局接口。
type Layout = internallayout.Layout

func releaseBuffer(buf *bytes.Buffer) {
	if buf.Cap() > 64*1024 {
		return
	}
	buf.Reset()
	bufferPool.Put(buf)
}
