package jsoncodec

import "testing"

func BenchmarkMarshal(b *testing.B) {
	payload := map[string]any{
		"traceId": "abc-123",
		"attempt": 3,
		"ok":      true,
		"tags":    []string{"core", "json", "fallback"},
		"nested": map[string]any{
			"latency": 12.5,
			"status":  "ready",
		},
	}
	b.Run("sonic", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := Marshal(payload); err != nil {
				b.Fatalf("Marshal() error = %v", err)
			}
		}
	})
}
