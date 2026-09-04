package layout

import (
	"fmt"
	"log/slog"
	"strings"
)

const maxStructuredPathDepth = 16

type structuredPathNode struct {
	name     string
	path     string
	value    slog.Value
	hasValue bool
	children []*structuredPathNode
	index    map[string]*structuredPathNode
}

func compileStructuredPaths(attrs []slog.Attr, prefix string) ([]*structuredPathNode, error) {
	root := structuredPathNode{}
	for _, attr := range attrs {
		path := joinStructuredPath(prefix, attr.Key, ".")
		if path == "" {
			continue
		}
		parts := strings.Split(path, ".")
		if len(parts) > maxStructuredPathDepth {
			return nil, fmt.Errorf("goark-log: structured JSON path %q exceeds maximum depth %d", path, maxStructuredPathDepth)
		}
		if err := root.insert(parts, path, attr.Value); err != nil {
			return nil, err
		}
	}
	return root.children, nil
}

func requiresNestedStructuredPaths(attrs []slog.Attr, prefix string) bool {
	if strings.TrimSpace(prefix) != "" {
		return true
	}
	for _, attr := range attrs {
		if !isStructuredControlAttr(attr.Key) && strings.Contains(attr.Key, ".") {
			return true
		}
	}
	return false
}

func (n *structuredPathNode) insert(parts []string, path string, value slog.Value) error {
	current := n
	for index, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return fmt.Errorf("goark-log: structured JSON path %q contains an empty segment", path)
		}
		if current.index == nil {
			current.index = make(map[string]*structuredPathNode)
		}
		child := current.index[part]
		if child == nil {
			child = &structuredPathNode{name: part, path: strings.Join(parts[:index+1], ".")}
			current.index[part] = child
			current.children = append(current.children, child)
		}
		if (child.hasValue && index != len(parts)-1) || (len(child.children) > 0 && index == len(parts)-1) {
			return fmt.Errorf("goark-log: duplicate structured JSON path under %q", child.path)
		}
		current = child
	}
	if current.hasValue {
		return fmt.Errorf("goark-log: duplicate structured JSON path %q", path)
	}
	current.value = value
	current.hasValue = true
	return nil
}

func (w *structuredWriter) addStructuredPaths(nodes []*structuredPathNode) {
	for _, node := range nodes {
		if node.hasValue {
			w.addPath(node.path, node.name, node.value)
			continue
		}
		if w.beginObject(node.path, node.name) {
			w.addStructuredPaths(node.children)
			w.endObject()
		}
	}
}

func joinStructuredPath(prefix, name, delimiter string) string {
	prefix = strings.TrimSpace(prefix)
	name = strings.TrimSpace(name)
	if prefix == "" {
		return name
	}
	if name == "" {
		return prefix
	}
	if strings.HasSuffix(prefix, delimiter) || strings.HasPrefix(name, delimiter) {
		return prefix + name
	}
	return prefix + delimiter + name
}
