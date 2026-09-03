package layout

import (
	"bytes"
	"testing"
)

func TestLoggerAbbreviator_shouldFollowLog4j2PrecisionRules(t *testing.T) {
	tests := []struct {
		name      string
		logger    string
		precision string
		want      string
	}{
		{name: "full", logger: "org.apache.commons.Foo", want: "org.apache.commons.Foo"},
		{name: "retain one", logger: "org.apache.commons.Foo", precision: "1", want: "Foo"},
		{name: "retain two", logger: "org.apache.commons.Foo", precision: "2", want: "commons.Foo"},
		{name: "retain more than available", logger: "org.apache.commons.Foo", precision: "10", want: "org.apache.commons.Foo"},
		{name: "retain ignores trailing empty component", logger: "org.apache.commons.Foo.", precision: "1", want: "Foo."},
		{name: "drop one", logger: "org.apache.commons.Foo", precision: "-1", want: "apache.commons.Foo"},
		{name: "drop two", logger: "org.apache.commons.Foo", precision: "-2", want: "commons.Foo"},
		{name: "drop more than available", logger: "org.apache.commons.Foo", precision: "-10", want: "org.apache.commons.Foo"},
		{name: "abbreviate packages", logger: "org.apache.commons.Foo", precision: "1.", want: "o.a.c.Foo"},
		{name: "abbreviate with marker", logger: "org.apache.commons.test.Foo", precision: "1~.2~", want: "o~.ap~.co~.te~.Foo"},
		{name: "pattern keeps remaining components", logger: "org.apache.commons.test.Foo", precision: "1.1.1.*", want: "o.a.c.test.Foo"},
		{name: "dynamic keeps two rightmost components", logger: "org.apache.commons.test.Foo", precision: "1.2*", want: "o.a.c.test.Foo"},
		{name: "dynamic keeps one rightmost component", logger: "org.apache.commons.test.Foo", precision: "1.1*", want: "o.a.c.t.Foo"},
		{name: "unicode package", logger: "中国.软件.服务", precision: "1.", want: "中.软.服务"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newLoggerAbbreviator(tt.precision).format(tt.logger); got != tt.want {
				t.Fatalf("logger abbreviation = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPatternLayout_whenLoggerPrecisionUsed_shouldApplyLog4j2Rules(t *testing.T) {
	layout, err := NewPatternLayout("%logger|%logger{2}|%logger{-1}|%logger{1.}|%.8logger|%.-8logger")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	var buf bytes.Buffer
	if err := layout.Format(&buf, Event{Logger: "org.apache.commons.Foo"}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	const want = "org.apache.commons.Foo|commons.Foo|apache.commons.Foo|o.a.c.Foo|mons.Foo|org.apac"
	if got := buf.String(); got != want {
		t.Fatalf("formatted logger names = %q, want %q", got, want)
	}
}

func TestPatternLayout_whenDefaultLoggerPatternUsed_shouldAbbreviateDisplayOnly(t *testing.T) {
	layout, err := NewPatternLayout("%logger{1.2*}")
	if err != nil {
		t.Fatalf("NewPatternLayout() error = %v", err)
	}
	event := Event{Logger: "goark.dev.arkhos.hertz"}
	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if got, want := buf.String(), "g.d.arkhos.hertz"; got != want {
		t.Fatalf("formatted logger = %q, want %q", got, want)
	}
	if got, want := event.Logger, "goark.dev.arkhos.hertz"; got != want {
		t.Fatalf("event logger = %q, want full name %q", got, want)
	}
}

func BenchmarkLoggerAbbreviator(b *testing.B) {
	const name = "org.apache.logging.log4j.core.pattern.NameAbbreviator"
	for _, precision := range []string{"", "2", "1.", "1.2*"} {
		b.Run(precision, func(b *testing.B) {
			abbreviator := newLoggerAbbreviator(precision)
			var buf bytes.Buffer
			buf.Grow(len(name))
			b.ReportAllocs()
			for b.Loop() {
				buf.Reset()
				abbreviator.append(&buf, name)
			}
		})
	}
}
