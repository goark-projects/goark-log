package layout

import (
	"bytes"
	"log/slog"
	"strings"

	"goark.dev/log/internal/logvalue"
)

type structuredWriter struct {
	buf          *bytes.Buffer
	layout       *StructuredJSONLayout
	comma        bool
	keys         [32]string
	scopeStarts  [maxStructuredPathDepth]int
	scopeCounts  [maxStructuredPathDepth]int
	depth        int
	keyCursor    int
	overflowKeys map[int]map[string]struct{}
}

func (w *structuredWriter) Add(key string, value slog.Value) {
	w.addPath(key, key, value)
}

func (w *structuredWriter) addPath(path, key string, value slog.Value) {
	path = strings.TrimSpace(path)
	key = strings.TrimSpace(key)
	if path == "" || !w.layout.enabled(path) {
		return
	}
	if renamed := w.layout.rename[path]; renamed != "" {
		key = renamed
	}
	if !w.claimKey(key) {
		return
	}
	logvalue.AppendJSONFieldValue(w.buf, key, value, w.comma)
	w.comma = true
}

func (w *structuredWriter) addNonEmpty(key, value string) {
	if strings.TrimSpace(value) != "" {
		w.Add(key, slog.StringValue(value))
	}
}

func (w *structuredWriter) addNonEmptyPath(path, key, value string) {
	if strings.TrimSpace(value) != "" {
		w.addPath(path, key, slog.StringValue(value))
	}
}

func (w *structuredWriter) addStrings(path, key string, values []string) {
	if len(values) == 0 || !w.layout.enabled(path) {
		return
	}
	if renamed := w.layout.rename[path]; renamed != "" {
		key = renamed
	}
	if !w.claimKey(key) {
		return
	}
	logvalue.AppendJSONKey(w.buf, key, w.comma)
	w.buf.WriteByte('[')
	for index, value := range values {
		if index > 0 {
			w.buf.WriteByte(',')
		}
		logvalue.AppendJSONString(w.buf, value)
	}
	w.buf.WriteByte(']')
	w.comma = true
}

func (w *structuredWriter) beginObject(path, key string) bool {
	return w.beginObjectIf(true, path, key)
}

func (w *structuredWriter) beginObjectIf(condition bool, path, key string) bool {
	if !condition || !w.layout.objectEnabled(path) {
		return false
	}
	if renamed := w.layout.rename[path]; renamed != "" {
		key = renamed
	}
	if w.depth+1 >= len(w.scopeStarts) || !w.claimKey(key) {
		return false
	}
	logvalue.AppendJSONKey(w.buf, key, w.comma)
	w.buf.WriteByte('{')
	w.depth++
	w.scopeStarts[w.depth] = w.keyCursor
	w.scopeCounts[w.depth] = 0
	w.comma = false
	return true
}

func (w *structuredWriter) endObject() {
	w.buf.WriteByte('}')
	if w.overflowKeys != nil {
		delete(w.overflowKeys, w.depth)
	}
	w.keyCursor = w.scopeStarts[w.depth]
	w.scopeCounts[w.depth] = 0
	w.depth--
	w.comma = true
}

func (w *structuredWriter) claimKey(key string) bool {
	start := w.scopeStarts[w.depth]
	count := w.scopeCounts[w.depth]
	for index := start; index < start+count; index++ {
		if w.keys[index] == key {
			return false
		}
	}
	if overflow := w.overflowKeys[w.depth]; overflow != nil {
		if _, exists := overflow[key]; exists {
			return false
		}
	}
	if w.keyCursor < len(w.keys) {
		w.keys[w.keyCursor] = key
		w.keyCursor++
		w.scopeCounts[w.depth]++
		return true
	}
	if w.overflowKeys == nil {
		w.overflowKeys = make(map[int]map[string]struct{})
	}
	overflow := w.overflowKeys[w.depth]
	if overflow == nil {
		overflow = make(map[string]struct{})
		w.overflowKeys[w.depth] = overflow
	}
	overflow[key] = struct{}{}
	return true
}

var _ StructuredJSONFieldAppender = (*structuredWriter)(nil)
