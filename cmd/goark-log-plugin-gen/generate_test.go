package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestGeneratePluginRegistrar_whenBindingsConfigured_shouldEmitRegistrar(t *testing.T) {
	data, err := generatePluginRegistrar(generatorConfig{
		PackageName:   "httpappender",
		RegistrarName: "Registrar",
		Appenders: []pluginBinding{
			{Kind: "http", Factory: "buildHTTPAppender"},
		},
		Layouts: []pluginBinding{
			{Kind: "compactJson", Factory: "newCompactJSONLayout"},
		},
	})
	if err != nil {
		t.Fatalf("generatePluginRegistrar() error = %v", err)
	}
	output := string(data)
	for _, want := range []string{
		"package httpappender",
		"func Registrar() goarklog.PluginRegistrar",
		`goarklog.WithPluginAppender("http", buildHTTPAppender)`,
		`goarklog.WithPluginLayout("compactJson", newCompactJSONLayout)`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("generated output missing %q:\n%s", want, output)
		}
	}
}

func TestGeneratePluginRegistrar_whenNoBindingsConfigured_shouldReject(t *testing.T) {
	_, err := generatePluginRegistrar(generatorConfig{
		PackageName:   "empty",
		RegistrarName: "Registrar",
	})
	if err == nil || !strings.Contains(err.Error(), "at least one plugin binding") {
		t.Fatalf("generatePluginRegistrar() error = %v, want missing binding rejection", err)
	}
}

func TestRun_whenStdoutRequested_shouldGenerateRegistrar(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"-package", "kafkaappender",
		"-appender", "kafka=buildKafkaAppender",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run() exit = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `goarklog.WithPluginAppender("kafka", buildKafkaAppender)`) {
		t.Fatalf("stdout = %s, want generated appender binding", stdout.String())
	}
}
