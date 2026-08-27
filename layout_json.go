package goarklog

import (
	"bytes"
	"strings"
	"sync/atomic"
)

// JSONLayout 输出 JSON 事件。
type JSONLayout struct {
	options LayoutOptions
	state   *jsonLayoutState
}

// NewJSONLayout 创建可配置 JSON 布局。
func NewJSONLayout(options LayoutOptions) JSONLayout {
	layout := JSONLayout{options: options}
	if options.Complete {
		layout.state = &jsonLayoutState{}
	}
	return layout
}

func (l JSONLayout) Format(buf *bytes.Buffer, event Event) error {
	appendJSONCompleteSeparator(buf, l.options, l.state)
	appendJSONLayoutEvent(buf, event, l.options)
	return nil
}

func (l JSONLayout) AppendHeader(buf *bytes.Buffer) error {
	appendJSONCompleteHeader(buf, l.options, l.state)
	return nil
}

func (l JSONLayout) AppendFooter(buf *bytes.Buffer) error {
	appendJSONCompleteFooter(buf, l.options)
	return nil
}

type jsonLayoutState struct {
	events atomic.Uint64
}

func appendJSONCompleteSeparator(buf *bytes.Buffer, options LayoutOptions, state *jsonLayoutState) {
	if !options.Complete || state == nil || state.events.Add(1) <= 1 {
		return
	}
	buf.WriteByte(',')
	if options.EventEOL || !options.Compact {
		buf.WriteByte('\n')
	}
}

func appendJSONCompleteHeader(buf *bytes.Buffer, options LayoutOptions, state *jsonLayoutState) {
	if state != nil {
		state.events.Store(0)
	}
	if !options.Complete {
		return
	}
	header := options.Header
	if strings.TrimSpace(header) == "" {
		header = "["
	}
	buf.WriteString(header)
	if options.EventEOL || !options.Compact {
		buf.WriteByte('\n')
	}
}

func appendJSONCompleteFooter(buf *bytes.Buffer, options LayoutOptions) {
	if !options.Complete {
		return
	}
	footer := options.Footer
	if strings.TrimSpace(footer) == "" {
		footer = "]"
	}
	buf.WriteString(footer)
}
