package syslogappender

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	goarklog "goark.dev/goark-log"
)

func TestAppender_whenUDPConfigured_shouldSendRFC5424StylePacket(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket() error = %v", err)
	}
	defer conn.Close()
	received := make(chan string, 1)
	go func() {
		buffer := make([]byte, 2048)
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		n, _, err := conn.ReadFrom(buffer)
		if err != nil {
			return
		}
		received <- string(buffer[:n])
	}()
	appender, err := New(conn.LocalAddr().String(),
		WithNetwork("udp"),
		WithFacility("local0"),
		WithAppName("goark"),
		WithLayout(goarklog.TextLayout{}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	event := goarklog.Event{
		Time:    time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC),
		Level:   slog.LevelError,
		Logger:  "goark.syslog",
		Message: "syslog event",
	}
	if err := appender.Append(context.Background(), event); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	select {
	case packet := <-received:
		if !strings.HasPrefix(packet, "<131>1 ") || !strings.Contains(packet, `msg="syslog event"`) {
			t.Fatalf("syslog packet = %q", packet)
		}
	case <-time.After(time.Second):
		t.Fatalf("syslog server did not receive packet")
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
