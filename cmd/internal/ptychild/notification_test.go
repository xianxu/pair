package ptychild

import (
	"bytes"
	"testing"

	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

func TestScreenNotificationEverySplit(t *testing.T) {
	input := append(append([]byte("before"), notifyosc.Encode("ready")...), []byte("after")...)
	for split := 0; split <= len(input); split++ {
		var screen Screen
		screen.Feed(input[:split])
		first := screen.TakeOutputParts()
		screen.Feed(input[split:])
		parts := append(first, screen.TakeOutputParts()...)
		var ordinary []byte
		var notifications []NotificationObservation
		for _, part := range parts {
			ordinary = append(ordinary, part.Bytes...)
			if part.Notification != nil {
				notifications = append(notifications, *part.Notification)
			}
		}
		if string(ordinary) != "beforeafter" || len(notifications) != 1 || notifications[0].Message != "ready" || !bytes.Equal(notifications[0].Raw, notifyosc.Encode("ready")) {
			t.Fatalf("split %d: ordinary=%q notifications=%+v", split, ordinary, notifications)
		}
	}
}

func TestScreenNotificationPrefixMismatchFlushes(t *testing.T) {
	var screen Screen
	screen.Feed([]byte("a\x1b]777;notify;other;x\x07b"))
	parts := screen.TakeOutputParts()
	var got []byte
	for _, part := range parts {
		if part.Notification != nil {
			t.Fatalf("unexpected notification: %+v", part.Notification)
		}
		got = append(got, part.Bytes...)
	}
	if string(got) != "a\x1b]777;notify;other;x\x07b" {
		t.Fatalf("passthrough = %q", got)
	}
}

func TestScreenNotificationCandidateWithholdsUntilComplete(t *testing.T) {
	var screen Screen
	prefix := []byte(notifyosc.Prefix + "wait")
	screen.Feed(prefix)
	if parts := screen.TakeOutputParts(); len(parts) != 0 {
		t.Fatalf("partial candidate emitted %+v", parts)
	}
	if got := screen.ReplaySafeEnd(); got != 0 {
		t.Fatalf("ReplaySafeEnd() = %d, want 0", got)
	}
	screen.Feed([]byte{0x07})
	parts := screen.TakeOutputParts()
	if len(parts) != 1 || parts[0].Notification == nil || parts[0].Notification.Message != "wait" {
		t.Fatalf("completion parts = %+v", parts)
	}
	if got := screen.ReplaySafeEnd(); got != uint64(len(prefix)+1) {
		t.Fatalf("ReplaySafeEnd() = %d", got)
	}
}

func TestScreenNotificationPrefixInsideAnotherOSCIsOrdinaryPayload(t *testing.T) {
	inner := notifyosc.Encode("not a nested event")
	outer := append([]byte("\x1b]0;title "), inner...)
	outer = append(outer, 0x07)
	var screen Screen
	screen.Feed(outer)
	parts := screen.TakeOutputParts()
	var got []byte
	for _, part := range parts {
		if part.Notification != nil {
			t.Fatalf("nested payload became notification: %+v", part.Notification)
		}
		got = append(got, part.Bytes...)
	}
	if !bytes.Equal(got, outer) {
		t.Fatalf("ordinary OSC changed: got %q want %q", got, outer)
	}
}
