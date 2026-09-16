package terminal

import (
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
