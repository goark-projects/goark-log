package goarklog

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

func TestHandlerMetrics_whenAppenderFails_shouldCountWritesAndFailures(t *testing.T) {
	success := newRecordingAppender("success")
	handler, err := NewHandler(Options{
		Appenders: []Appender{success, failingAppender{name: "failure"}},
		Root: RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"success", "failure"},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	defer handler.Close()

	logger, err := NewNativeLogger(handler, "goark.metrics")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if err := logger.Info("counted"); err == nil {
		t.Fatalf("Info() error = nil, want appender failure")
	}

	metrics := handler.Metrics()
	if metrics.Events != 1 || metrics.AppenderWrites != 1 || metrics.AppenderFailures != 1 {
		t.Fatalf("metrics = %+v", metrics)
	}
}

func TestHandlerMetrics_whenFilterDenies_shouldCountFiltered(t *testing.T) {
	appender := newRecordingAppender("memory")
	handler, err := NewHandler(Options{
		Appenders: []Appender{appender},
		Root: RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"memory"},
			Filters:      []Filter{NewDenyFilter()},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	defer handler.Close()

	logger, err := NewNativeLogger(handler, "goark.metrics")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if err := logger.Info("filtered"); err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	metrics := handler.Metrics()
	if metrics.Filtered != 1 || metrics.Events != 0 || len(appender.Events()) != 0 {
		t.Fatalf("metrics = %+v, events=%+v", metrics, appender.Events())
	}
}

func TestHandlerMetrics_whenAppenderRefSkips_shouldNotCountWrite(t *testing.T) {
	allAppender := newRecordingAppender("all")
	errorAppender := newRecordingAppender("errors")
	handler, err := NewHandler(Options{
		Appenders: []Appender{allAppender, errorAppender},
		Root: RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"all"},
			AppenderRefControls: []AppenderRef{
				NewAppenderRef("errors", WithAppenderRefLevel(slog.LevelError)),
			},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	defer handler.Close()

	logger, err := NewNativeLogger(handler, "goark.metrics")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if err := logger.Info("info event"); err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	metrics := handler.Metrics()
	if metrics.Events != 1 || metrics.AppenderWrites != 1 || metrics.AppenderFailures != 0 {
		t.Fatalf("metrics = %+v", metrics)
	}
	if len(errorAppender.Events()) != 0 {
		t.Fatalf("error appender events = %+v, want none", errorAppender.Events())
	}
}

func TestHandlerExportMetrics_whenExporterProvided_shouldExportSnapshot(t *testing.T) {
	handler, err := NewHandler(Options{
		Appenders: []Appender{newRecordingAppender("memory")},
		Root: RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"memory"},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	defer handler.Close()

	exporter := &recordingMetricsExporter{}
	if err := handler.ExportMetrics(context.Background(), exporter); err != nil {
		t.Fatalf("ExportMetrics() error = %v", err)
	}
	if !exporter.called {
		t.Fatalf("exporter was not called")
	}
	if err := handler.ExportMetrics(context.Background(), nil); err == nil {
		t.Fatalf("ExportMetrics(nil) error = nil")
	}
}

type recordingMetricsExporter struct {
	called bool
}

func (e *recordingMetricsExporter) ExportLogMetrics(context.Context, MetricsSnapshot) error {
	e.called = true
	return nil
}

func TestHandlerExportMetrics_whenExporterFails_shouldReturnError(t *testing.T) {
	handler := NewDefaultHandler()
	defer handler.Close()
	wantErr := errors.New("export failed")
	err := handler.ExportMetrics(context.Background(), MetricsExporterFunc(func(context.Context, MetricsSnapshot) error {
		return wantErr
	}))
	if !errors.Is(err, wantErr) {
		t.Fatalf("ExportMetrics() error = %v, want %v", err, wantErr)
	}
}
