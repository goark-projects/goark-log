package goarklog

import (
	"strings"
	"testing"
)

func TestPluginRegistry_whenBlockedLookupRegistered_shouldReject(t *testing.T) {
	registry := NewPluginRegistry()
	err := registry.RegisterLookup("jndi", func(string) (string, bool) {
		return "unsafe", true
	})
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("RegisterLookup() error = %v, want blocked rejection", err)
	}
}

func TestLookupResolver_whenBlockedLookupRegisteredDirectly_shouldIgnore(t *testing.T) {
	resolver := NewLookupResolver()
	resolver.Register("ldap", func(string) (string, bool) {
		return "unsafe", true
	})
	_, err := resolver.Resolve("${ldap:name}")
	if err == nil || !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("Resolve() error = %v, want unregistered namespace", err)
	}
}
