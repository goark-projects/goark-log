package httpappender

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	goarklog "goark.dev/log"
)

func TestAppender_whenAppendCalled_shouldPostFormattedEvent(t *testing.T) {
	body := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		content, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		body <- string(content)
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	appender, err := New(server.URL, WithLayout(goarklog.JSONLayout{}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	event := goarklog.Event{
		Time:    time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC),
		Level:   slog.LevelInfo,
		Logger:  "goark.http",
		Message: "http event",
	}
	if err := appender.Append(context.Background(), event); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if got := <-body; !strings.Contains(got, `"msg":"http event"`) {
		t.Fatalf("posted body = %q, want event json", got)
	}
}

func TestRegister_whenConfiguredThroughYaml_shouldBuildHTTPAppender(t *testing.T) {
	body := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		content, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		body <- string(content)
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	registry := goarklog.NewPluginRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	configPath := writeConfig(t, `
appenders:
  http:
    type: http
    url: "`+server.URL+`"
    writeTimeout: 2s
    layout:
      type: json
root:
  level: info
  appenderRefs: [http]
`)

	logger, handler, _, err := goarklog.NewConfigured(context.Background(), goarklog.WithConfigPath(configPath), goarklog.WithPluginRegistry(registry))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Info("configured http")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got := <-body; !strings.Contains(got, `"msg":"configured http"`) {
		t.Fatalf("posted body = %q, want configured event json", got)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "goark-log.yml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
