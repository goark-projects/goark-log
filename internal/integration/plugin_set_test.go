package integration

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPluginSet_whenRegistered_shouldExposeCoreExtensionPoints(t *testing.T) {
	registry := NewPluginRegistry()
	registrar := NewPluginSet(
		WithPluginLayout("setPrefix", func(config LayoutBuildConfig) (Layout, error) {
			return prefixLayout{prefix: config.Pattern}, nil
		}),
		WithPluginLookup("tenant", func(key string) (string, bool) {
			return "tenant-" + key, true
		}),
		WithPluginJSONTemplateResolver("setConstant", func(config JSONTemplateResolverBuildConfig) (JSONTemplateResolver, error) {
			var value string
			if err := json.Unmarshal(config.Options["value"], &value); err != nil {
				return nil, err
			}
			return constantJSONResolver(value), nil
		}),
	)
	if err := registry.RegisterPlugins(registrar); err != nil {
		t.Fatalf("RegisterPlugins() error = %v", err)
	}
	resolved, err := registry.LookupResolver().Resolve("${tenant:acme}")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved != "tenant-acme" {
		t.Fatalf("resolved lookup = %q, want tenant-acme", resolved)
	}
	layout, err := buildLayout(layoutConfig{Type: "setPrefix", Pattern: "SET"}, registry)
	if err != nil {
		t.Fatalf("buildLayout() error = %v", err)
	}
	var buf bytes.Buffer
	if err := layout.Format(&buf, testEvent("event", fixedTestTime())); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if !strings.HasPrefix(buf.String(), "SET:event") {
		t.Fatalf("layout output = %q, want registered layout prefix", buf.String())
	}

	template, err := NewJSONTemplateLayout(`{"value":{"$resolver":"setConstant","value":"ok"}}`, WithJSONTemplateResolverRegistry(registry))
	if err != nil {
		t.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}
	buf.Reset()
	if err := template.Format(&buf, testEvent("event", fixedTestTime())); err != nil {
		t.Fatalf("Format(template) error = %v", err)
	}
	if strings.TrimSpace(buf.String()) != `{"value":"ok"}` {
		t.Fatalf("template output = %s, want custom resolver value", buf.String())
	}
}

func TestPluginSet_whenInvalidPluginDeclared_shouldReturnContextualError(t *testing.T) {
	registrar := NewPluginSet(WithPluginAppender("", func(AppenderBuildConfig) (Appender, error) {
		return NewConsoleAppender(), nil
	}))
	err := NewPluginRegistry().RegisterPlugins(registrar)
	if err == nil || !strings.Contains(err.Error(), "register appender plugin") {
		t.Fatalf("RegisterPlugins() error = %v, want contextual registration error", err)
	}
}
