package terminal

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

func TestEndpointZellijNotificationMapping(t *testing.T) {
	for _, body := range []string{"ready; café", strings.Repeat("x", notifyosc.MaxMessageBytes), strings.Repeat("é", notifyosc.MaxMessageBytes/2)} {
		for _, terminator := range []string{"\a", "\x1b\\"} {
			e, _ := newEndpointTest(t, "origin")
			var effects []Effect
			for _, b := range []byte("\x1b]9;pair: " + body + terminator) {
				out, err := e.Feed([]byte{b}, time.Now())
				if err != nil {
					t.Fatal(err)
				}
				effects = append(effects, out.Effects...)
			}
			if len(effects) != 1 || effects[0].Kind != NotificationEffect || effects[0].Selection != "pair" || effects[0].Text != body || effects[0].EndpointID != "origin" {
				t.Fatalf("body bytes=%d: effects=%+v", len(body), effects)
			}
			e.Snapshot(time.Now())
			out, err := e.Feed(nil, time.Now())
			if err != nil || len(out.Effects) != 0 {
				t.Fatalf("replayed notification: %+v %v", out, err)
			}
		}
	}
}

func TestEndpointZellijMappingKeepsGenericNotificationsAndBounds(t *testing.T) {
	for _, tc := range []struct {
		body, title, text string
		count             int
	}{
		{"other: ready", "", "other: ready", 1},
		{"pair: ready", "pair", "ready", 1},
		{"pair: " + strings.Repeat("x", notifyosc.MaxMessageBytes+1), "", "", 0},
		{strings.Repeat("x", notifyosc.MaxMessageBytes+1), "", "", 0},
	} {
		e, _ := newEndpointTest(t, "one")
		out, err := e.Feed([]byte("\x1b]9;"+tc.body+"\a"), time.Now())
		if err != nil || len(out.Effects) != tc.count {
			t.Fatalf("body bytes=%d: %+v %v", len(tc.body), out, err)
		}
		if tc.count > 0 && (out.Effects[0].Selection != tc.title || out.Effects[0].Text != tc.text) {
			t.Fatalf("wrong mapping %+v", out.Effects)
		}
	}
}

// Captured #379 notification must remain an effect while the cursor is in nvim.
func TestEndpointClaudeFooterNotificationDoesNotPaintAtCursor(t *testing.T) {
	body := "✻ Crunched for 12s · done 7:57 PM"
	wire := []byte("\x1b[?2026h\x1b]9;pair: " + body + "\a\x1b[?2026l")
	for split := -1; split <= len(wire); split++ {
		t.Run(fmt.Sprint(split), func(t *testing.T) {
			e, err := NewEndpoint("brain", Geometry{191, 53}, ttyio.NewFake())
			if err != nil {
				t.Fatal(err)
			}
			defer e.Close()
			now := time.Now()
			if _, err = e.Feed([]byte("\x1b[40;98H74   \x1b[40;103H"), now); err != nil {
				t.Fatal(err)
			}
			before, _ := e.Snapshot(now)
			var effects []Effect
			feed := func(data []byte) {
				out, err := e.Feed(data, now)
				if err != nil {
					t.Fatal(err)
				}
				effects = append(effects, out.Effects...)
			}
			if split < 0 {
				for _, b := range wire {
					feed([]byte{b})
				}
			} else {
				feed(wire[:split])
				feed(wire[split:])
			}
			after, err := e.Snapshot(now)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before.Cells, after.Cells) {
				t.Fatal("notification changed terminal cells at nvim cursor")
			}
			if len(effects) != 1 || effects[0].Kind != NotificationEffect || effects[0].Text != body {
				t.Fatalf("notification effects=%+v", effects)
			}
		})
	}
}
