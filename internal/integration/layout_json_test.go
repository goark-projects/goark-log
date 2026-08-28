package integration

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

func TestJSONLayout_whenAnyAttrIsMap_shouldEncodeStructuredJSON(t *testing.T) {
	event := Event{
		Time:    time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC),
		Level:   slog.LevelInfo,
		Logger:  "goark.test",
		Message: "event",
		Attrs: []slog.Attr{
			slog.Any("payload", map[string]any{
				"traceId": "abc-123",
				"attempt": 2,
				"ok":      true,
			}),
		},
	}

	var buf bytes.Buffer
	if err := (JSONLayout{}).Format(&buf, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var decoded struct {
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON output is invalid: %v\n%s", err, buf.String())
	}
	if decoded.Payload["traceId"] != "abc-123" {
		t.Fatalf("payload.traceId = %v, want abc-123", decoded.Payload["traceId"])
	}
	if decoded.Payload["attempt"] != float64(2) {
		t.Fatalf("payload.attempt = %v, want 2", decoded.Payload["attempt"])
	}
	if decoded.Payload["ok"] != true {
		t.Fatalf("payload.ok = %v, want true", decoded.Payload["ok"])
	}
}
