package socketappender

import (
	"bufio"
	"context"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	goarklog "goark.dev/goark-log"
)

func TestAppender_whenTCPConfigured_shouldWriteToSocket(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer listener.Close()
	received := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		line, _ := bufio.NewReader(conn).ReadString('\n')
		received <- line
	}()
	appender, err := New(listener.Addr().String(),
		WithLayout(goarklog.TextLayout{}),
		WithDialTimeout(time.Second),
		WithWriteTimeout(time.Second),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	event := goarklog.Event{
		Time:    time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC),
		Level:   slog.LevelInfo,
		Logger:  "goark.socket",
		Message: "socket event",
	}
	if err := appender.Append(context.Background(), event); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	select {
	case line := <-received:
		if !strings.Contains(line, "msg=\"socket event\"") {
			t.Fatalf("socket line = %q, want socket event", line)
		}
	case <-time.After(time.Second):
		t.Fatalf("socket server did not receive event")
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
