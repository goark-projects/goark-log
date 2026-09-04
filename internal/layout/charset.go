package layout

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
)

type charsetLayout struct {
	delegate Layout
	encoding encoding.Encoding
}

// NewCharsetLayout 使用指定字符集编码布局结果，UTF-8 不增加转换层。
func NewCharsetLayout(delegate Layout, name string) (Layout, error) {
	if delegate == nil {
		return nil, fmt.Errorf("goark-log: charset layout delegate is nil")
	}
	name = strings.TrimSpace(name)
	if name == "" || isUTF8(name) {
		return CloneLayout(delegate), nil
	}
	charset, err := htmlindex.Get(name)
	if err != nil {
		return nil, fmt.Errorf("goark-log: unsupported charset %q: %w", name, err)
	}
	return &charsetLayout{delegate: CloneLayout(delegate), encoding: charset}, nil
}

func (l *charsetLayout) Format(destination *bytes.Buffer, event Event) error {
	if l == nil || l.delegate == nil || l.encoding == nil {
		return fmt.Errorf("goark-log: charset layout is not initialized")
	}
	return l.transform(destination, func(source *bytes.Buffer) error {
		return l.delegate.Format(source, event)
	})
}

func (l *charsetLayout) AppendHeader(destination *bytes.Buffer) error {
	return l.transform(destination, func(source *bytes.Buffer) error {
		return appendDelegateHeader(source, l.delegate)
	})
}

func (l *charsetLayout) AppendFooter(destination *bytes.Buffer) error {
	return l.transform(destination, func(source *bytes.Buffer) error {
		return appendDelegateFooter(source, l.delegate)
	})
}

func (l *charsetLayout) CloneLayout() Layout {
	if l == nil {
		return l
	}
	return &charsetLayout{delegate: CloneLayout(l.delegate), encoding: l.encoding}
}

func (l *charsetLayout) RequiresSynchronizedFormatting() bool {
	return RequiresSynchronizedFormatting(l.delegate)
}

func (l *charsetLayout) transform(destination *bytes.Buffer, render func(*bytes.Buffer) error) error {
	source := bufferPool.Get().(*bytes.Buffer)
	source.Reset()
	defer releaseBuffer(source)
	if err := render(source); err != nil {
		return err
	}
	encoded, err := l.encoding.NewEncoder().Bytes(source.Bytes())
	if err != nil {
		return fmt.Errorf("goark-log: encode layout output: %w", err)
	}
	destination.Write(encoded)
	return nil
}

func appendDelegateHeader(destination *bytes.Buffer, layout Layout) error {
	if lifecycle, ok := layout.(lifecycleLayout); ok {
		return lifecycle.AppendHeader(destination)
	}
	return nil
}

func appendDelegateFooter(destination *bytes.Buffer, layout Layout) error {
	if lifecycle, ok := layout.(lifecycleLayout); ok {
		return lifecycle.AppendFooter(destination)
	}
	return nil
}

func isUTF8(name string) bool {
	normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(name))
	return normalized == "utf8"
}
