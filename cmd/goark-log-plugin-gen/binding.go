package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type pluginBinding struct {
	Kind    string
	Factory string
}

type bindingFlag struct {
	values []pluginBinding
}

func (f *bindingFlag) String() string {
	if f == nil || len(f.values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(f.values))
	for _, value := range f.values {
		parts = append(parts, value.Kind+"="+value.Factory)
	}
	return strings.Join(parts, ",")
}

func (f *bindingFlag) Set(value string) error {
	kind, factory, ok := strings.Cut(value, "=")
	kind = strings.TrimSpace(kind)
	factory = strings.TrimSpace(factory)
	if !ok || kind == "" || factory == "" {
		return fmt.Errorf("plugin binding %q must use kind=factory", value)
	}
	if !isGoSelector(factory) {
		return fmt.Errorf("plugin binding factory %q is not a Go identifier or selector", factory)
	}
	f.values = append(f.values, pluginBinding{Kind: kind, Factory: factory})
	return nil
}

func (f *bindingFlag) Values() []pluginBinding {
	if f == nil || len(f.values) == 0 {
		return nil
	}
	return append([]pluginBinding(nil), f.values...)
}

func isPackageName(value string) bool {
	value = strings.TrimSpace(value)
	return isGoIdentifier(value) && !goKeywords[value]
}

func isGoSelector(value string) bool {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if !isGoIdentifier(part) || goKeywords[part] {
			return false
		}
	}
	return true
}

func isGoIdentifier(value string) bool {
	if value == "" {
		return false
	}
	r, size := utf8.DecodeRuneInString(value)
	if r == utf8.RuneError || !isIdentifierStart(r) {
		return false
	}
	for _, r := range value[size:] {
		if !isIdentifierPart(r) {
			return false
		}
	}
	return true
}

func isIdentifierStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentifierPart(r rune) bool {
	return isIdentifierStart(r) || unicode.IsDigit(r)
}

var goKeywords = map[string]bool{
	"break":       true,
	"default":     true,
	"func":        true,
	"interface":   true,
	"select":      true,
	"case":        true,
	"defer":       true,
	"go":          true,
	"map":         true,
	"struct":      true,
	"chan":        true,
	"else":        true,
	"goto":        true,
	"package":     true,
	"switch":      true,
	"const":       true,
	"fallthrough": true,
	"if":          true,
	"range":       true,
	"type":        true,
	"continue":    true,
	"for":         true,
	"import":      true,
	"return":      true,
	"var":         true,
}
