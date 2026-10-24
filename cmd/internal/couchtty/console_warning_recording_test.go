package couchtty

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Exercise the actual recording-end check, not a direct call to a log callback:
// healthy fidelity is feedback; an unavailable check requires attention.
func TestRecordedCastFidelityAttentionDistinguishesSuccess(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "healthy", true: "failed"}[failed], func(t *testing.T) {
			c, _ := continuationConsole(t)
			browser := webcast.NewFakeBrowser()
			var options *webcast.RecordOptions
			c.SetWebcast(webcast.CastConfig{
				Launch: func(_ context.Context, opts *webcast.RecordOptions) (webcast.Browser, error) {
					options = opts
					return browser, nil
				},
				Host:          broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}, ProbeTimeout: 2 * time.Second}),
				RecordingsDir: t.TempDir(),
				Check: func(context.Context, string) (string, error) {
					if failed {
						return "", errors.New("fidelity unavailable")
					}
					return "healthy fidelity summary", nil
				},
			})
			cast, err := webcast.StartCast(context.Background(), *c.webcastCfg)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(cast.End)
			out, err := options.Out()
			if err != nil {
				t.Fatal(err)
			}
			if err := out.Sink.Put(recording.ContractRecording(0)[0]); err != nil {
				t.Fatal(err)
			}
			if err := out.Frames.PutFrame(&recording.Frame{File: "frames/0.png", W: 1, H: 1, Scale: 0.5}, []byte("png")); err != nil {
				t.Fatal(err)
			}
			if err := out.Sink.Close(); err != nil {
				t.Fatal(err)
			}
			browser.QuitByHand()
			<-cast.Done()
			want := "healthy fidelity summary"
			if failed {
				want = "fidelity unavailable"
			}
			waitFor(t, "fidelity outcome retained", func() bool { return strings.Contains(c.menuSnapshot().LatestFeed, want) })
			if count := c.attentionCount(); count != map[bool]int{false: 0, true: 1}[failed] {
				t.Fatalf("failed=%t attention=%d", failed, count)
			}
		})
	}
}
