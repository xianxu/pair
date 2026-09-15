package ptychild

import (
	"bytes"
	"context"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestChildHasOneTerminalAuthority(t *testing.T) {
	typ := reflect.TypeOf(Child{})
	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).Type == reflect.TypeOf((*Screen)(nil)) {
			t.Fatal("Child still retains duplicate Screen parser")
		}
	}
}

func TestEndpointRetainsFrameAndModesAfterDiagnosticRingEviction(t *testing.T) {
	child := NewFakeChild(nil)
	t.Cleanup(func() { child.Close() })
	child.ring = NewRing(32)
	child.Feed([]byte("\x1b[?1049h\x1b[?1002h\x1b[?1006h\x1b[HKEPT"))
	child.Feed([]byte(strings.Repeat("\x1b[2;1Hx", 64)))
	if bytes.Contains(child.Snapshot(), []byte("KEPT")) {
		t.Fatal("fixture did not evict original paint")
	}
	frame, err := child.Endpoint().Snapshot(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var row strings.Builder
	for _, c := range frame.Cells[:frame.Geometry.Cols] {
		row.WriteString(c.Content)
	}
	if !strings.HasPrefix(row.String(), "KEPT") {
		t.Fatalf("eviction lost frame: %q", row.String())
	}
	modes := child.Endpoint().Modes()
	if !modes.AltScreen || modes.Tracking != 1002 || !modes.SGR {
		t.Fatalf("eviction lost modes: %+v", modes)
	}
	if err := child.FlushOutput(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestChildNotificationIsTypedOnceIndependentOfRawRetention(t *testing.T) {
	envelope := notifyosc.Encode("review ready")
	stream := append(append([]byte("before"), envelope...), []byte("after")...)
	for capacity := 1; capacity <= len(stream); capacity++ {
		child := NewFakeChild(nil)
		child.ring = NewRing(capacity)
		var notifications []terminal.Effect
		child.SetSink(func(_ context.Context, b OutputBatch) error {
			for _, e := range b.Terminal.Effects {
				if e.Kind == terminal.NotificationEffect {
					notifications = append(notifications, e)
				}
			}
			return nil
		})
		for _, b := range stream {
			child.Feed([]byte{b})
		}
		if err := child.FlushOutput(context.Background()); err != nil {
			child.Close()
			t.Fatal(err)
		}
		frame, err := child.Endpoint().Snapshot(time.Now())
		if err != nil {
			child.Close()
			t.Fatal(err)
		}
		var row strings.Builder
		for _, c := range frame.Cells[:frame.Geometry.Cols] {
			row.WriteString(c.Content)
		}
		if row.String() != "beforeafter" || len(notifications) != 1 || notifications[0].Text != "review ready" {
			child.Close()
			t.Fatalf("capacity%d row%q notifications%+v", capacity, row.String(), notifications)
		}
		if !bytes.Equal(child.Snapshot(), stream[len(stream)-capacity:]) {
			child.Close()
			t.Fatalf("raw diagnostic retention changed at%d", capacity)
		}
		child.Close()
	}
}

func TestChildQueriesReplyOnlyToOriginEndpoint(t *testing.T) {
	child := NewFakeChild(nil)
	defer child.Close()
	child.Feed([]byte("\x1b[cvisible"))
	if err := child.Endpoint().Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := string(bytes.Join(child.Writes(), nil)); got != "\x1b[?1;2c" {
		t.Fatalf("origin query reply=%q", got)
	}
	frame, err := child.Endpoint().Snapshot(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var row strings.Builder
	for _, c := range frame.Cells[:frame.Geometry.Cols] {
		row.WriteString(c.Content)
	}
	if row.String() != "visible" {
		t.Fatalf("query leaked into display cells: %q", row.String())
	}
}
