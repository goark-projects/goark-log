package asyncruntime

import "testing"

func TestNormalizeLoggerOptions_whenAliasesUsed_shouldNormalize(t *testing.T) {
	options, err := NormalizeLoggerOptions(LoggerOptions{
		Enabled:          true,
		QueueSize:        7,
		BatchSize:        16,
		OverflowStrategy: "discard-debug",
		WaitStrategy:     "busy-spin",
		IncludeLocation:  true,
	})
	if err != nil {
		t.Fatalf("NormalizeLoggerOptions() error = %v", err)
	}
	if options.QueueSize != 8 {
		t.Fatalf("QueueSize = %d, want normalized power of two 8", options.QueueSize)
	}
	if options.BatchSize != 8 {
		t.Fatalf("BatchSize = %d, want capped queue size 8", options.BatchSize)
	}
	if options.OverflowStrategy != OverflowDropDebug {
		t.Fatalf("OverflowStrategy = %q, want %q", options.OverflowStrategy, OverflowDropDebug)
	}
	if options.WaitStrategy != WaitSpin {
		t.Fatalf("WaitStrategy = %q, want %q", options.WaitStrategy, WaitSpin)
	}
	if !options.IncludeLocation {
		t.Fatalf("IncludeLocation = false, want true")
	}
}
