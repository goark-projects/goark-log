package integration

import (
	"context"
	"testing"
)

func TestLoadOptions_whenConfigDataProvided_shouldDecodeWithoutFilesystem(t *testing.T) {
	options, result, err := LoadOptions(context.Background(), WithConfigData("classpath:logging/goark-log.yml", []byte(`
appenders:
  console:
    type: console
root:
  level: warn
  appenderRefs: [console]
`)))
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	defer closeAppendersForTest(t, options.Appenders)
	if result.Source != ConfigSourceExplicit || result.Path != "classpath:logging/goark-log.yml" {
		t.Fatalf("ConfigResult = %#v", result)
	}
	if options.Root.Level.String() != "WARN" {
		t.Fatalf("root level = %s", options.Root.Level)
	}
}

func TestLoadOptions_whenConfigDataNameHasUnsupportedExtension_shouldReject(t *testing.T) {
	if _, _, err := LoadOptions(context.Background(), WithConfigData("logging.conf", []byte("root.level=info"))); err == nil {
		t.Fatal("unsupported config data extension should fail")
	}
}

func TestLoadOptions_whenBootResolverAndConfigDataProvided_shouldResolveProperties(t *testing.T) {
	options, _, err := LoadOptions(context.Background(),
		WithBootPropertyResolver(PropertyMap{"logging.level.root": "error"}),
		WithConfigData("goark-log.yml", []byte(`
appenders:
  console:
    type: console
root:
  level: ${logging.level.root}
  appenderRefs: [console]
`)),
	)
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	defer closeAppendersForTest(t, options.Appenders)
	if options.Root.Level.String() != "ERROR" {
		t.Fatalf("root level = %s", options.Root.Level)
	}
}

func closeAppendersForTest(t *testing.T, appenders []Appender) {
	t.Helper()
	for _, appender := range appenders {
		if err := appender.Close(); err != nil {
			t.Errorf("Close(%q) error = %v", appender.Name(), err)
		}
	}
}
