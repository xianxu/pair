package broadcast

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

// TestManualViewerServer serves a sample broadcast on loopback for checking
// the viewer page in a real browser (CSP errors, rendering, font fit). Gated:
//
//	BROADCAST_MANUAL=1 BROADCAST_MANUAL_SECONDS=300 go test ./cmd/internal/broadcast -run TestManualViewerServer -v
//
// BROADCAST_MANUAL_THEME, a broadcast.Theme as JSON, sends that palette to
// viewers. The link is printed; a new frame arrives every second.
func TestManualViewerServer(t *testing.T) {
	if os.Getenv("BROADCAST_MANUAL") != "1" {
		t.Skip("set BROADCAST_MANUAL=1")
	}
	seconds, _ := strconv.Atoi(os.Getenv("BROADCAST_MANUAL_SECONDS"))
	if seconds <= 0 {
		seconds = 300
	}
	cfg := Config{Tunnel: LocalOnly{}}
	if raw := os.Getenv("BROADCAST_MANUAL_THEME"); raw != "" {
		var theme Theme
		if err := json.Unmarshal([]byte(raw), &theme); err != nil {
			t.Fatalf("BROADCAST_MANUAL_THEME: %v", err)
		}
		cfg.Theme = func() Theme { return theme }
	}
	s, err := Start(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop(nil)
	fmt.Printf("BROADCAST LINK: %s\n", s.Link())
	deadline := time.Now().Add(time.Duration(seconds) * time.Second)
	for i := 0; time.Now().Before(deadline); i++ {
		body := fmt.Sprintf("\x1b[1mCouch broadcast sample\x1b[0m  frame %d  \x1b[3mitalic\x1b[0m \x1b[1;3mbold italic\x1b[0m\n"+
			"\x1b[31mred\x1b[0m \x1b[32mgreen\x1b[0m \x1b[33myellow\x1b[0m \x1b[34mblue\x1b[0m \x1b[35mmagenta\x1b[0m \x1b[36mcyan\x1b[0m \x1b[90mbright black\x1b[0m \x1b[91mbright red\x1b[0m \x1b[44mblue bg\x1b[0m\n"+
			"wide: 界面  box: ┌─┐  ligature check: -> != === www\n"+
			"%s", i, time.Now().Format(time.TimeOnly))
		s.Offer(textFrame(t, 100, 30, body, liveChrome("REC 21%  ariadne:2  brain  pair")), terminal.FramePublic)
		time.Sleep(time.Second)
	}
}
