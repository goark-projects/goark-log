package integration

import (
	"bytes"
	"testing"
)

func TestDefaultPluginHelpers_whenLayoutRegistered_shouldUseDefaultRegistry(t *testing.T) {
	if err := RegisterLayout("helperPrefix", func(config LayoutBuildConfig) (Layout, error) {
		return prefixLayout{prefix: config.Pattern}, nil
	}); err != nil {
		t.Fatalf("RegisterLayout() error = %v", err)
	}
	layout, err := buildLayout(layoutConfig{Type: "helperPrefix", Pattern: "HELPER"}, DefaultPluginRegistry())
	if err != nil {
		t.Fatalf("buildLayout() error = %v", err)
	}

	var buf bytes.Buffer
	if err := layout.Format(&buf, testEvent("event", fixedTestTime())); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if buf.String() != "HELPER:event\n" {
		t.Fatalf("output = %q, want helper layout output", buf.String())
	}
}

func TestPluginRegistry_whenRegistrarUsed_shouldRegisterPlugins(t *testing.T) {
	registry := NewPluginRegistry()
	registrar := PluginRegistrarFunc(func(registry *PluginRegistry) error {
		return registry.RegisterLayout("registrarPrefix", func(config LayoutBuildConfig) (Layout, error) {
			return prefixLayout{prefix: config.Pattern}, nil
		})
	})
	if err := registry.RegisterPlugins(registrar); err != nil {
		t.Fatalf("RegisterPlugins() error = %v", err)
	}
	layout, err := buildLayout(layoutConfig{Type: "registrarPrefix", Pattern: "REG"}, registry)
	if err != nil {
		t.Fatalf("buildLayout() error = %v", err)
	}

	var buf bytes.Buffer
	if err := layout.Format(&buf, testEvent("event", fixedTestTime())); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if buf.String() != "REG:event\n" {
		t.Fatalf("output = %q, want registrar layout output", buf.String())
	}
}
