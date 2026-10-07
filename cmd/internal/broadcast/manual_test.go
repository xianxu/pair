package broadcast

import (
	"context"
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
// The link is printed; a new frame arrives every second.
func TestManualViewerServer(t *testing.T) {
	if os.Getenv("BROADCAST_MANUAL") != "1" {
		t.Skip("set BROADCAST_MANUAL=1")
	}
	seconds, _ := strconv.Atoi(os.Getenv("BROADCAST_MANUAL_SECONDS"))
	if seconds <= 0 {
		seconds = 300
	}
	s, err := Start(context.Background(), Config{Tunnel: LocalOnly{}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop(nil)
	fmt.Printf("BROADCAST LINK: %s\n", s.Link())
	deadline := time.Now().Add(time.Duration(seconds) * time.Second)
	for i := 0; time.Now().Before(deadline); i++ {
		body := fmt.Sprintf("\x1b[1mCouch broadcast sample\x1b[0m  frame %d\n"+
			"\x1b[31mred\x1b[0m \x1b[32mgreen\x1b[0m \x1b[44mblue bg\x1b[0m  wide: 界面  box: ┌─┐\n"+
			"%s", i, time.Now().Format(time.TimeOnly))
		s.Offer(textFrame(t, 100, 30, body, liveChrome("REC 21%  ariadne:2  brain  pair")), terminal.FramePublic)
		time.Sleep(time.Second)
	}
}
