package rollingfile

import (
	"bytes"
	"io"
	"sync"

	internallayout "goark.dev/log/internal/layout"
)

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func acquireBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func releaseBuffer(buf *bytes.Buffer) {
	if buf.Cap() > 64*1024 {
		return
	}
	buf.Reset()
	bufferPool.Put(buf)
}

func (a *RollingFileAppender) flushLocked() error {
	if a == nil || a.writer == nil {
		return nil
	}
	return a.writer.Flush()
}

func (a *RollingFileAppender) writeHeaderLocked() (int, error) {
	writer := a.outputWriterLocked()
	if writer == nil {
		return 0, nil
	}
	return internallayout.WriteHeader(writer, a.layout)
}

func (a *RollingFileAppender) writeFooterLocked() (int, error) {
	writer := a.outputWriterLocked()
	if writer == nil {
		return 0, nil
	}
	return internallayout.WriteFooter(writer, a.layout)
}

func (a *RollingFileAppender) writeFooterErrorLocked() error {
	_, err := a.writeFooterLocked()
	return err
}

func (a *RollingFileAppender) outputWriterLocked() io.Writer {
	if a == nil {
		return nil
	}
	if a.writer != nil {
		return a.writer
	}
	return a.file
}
