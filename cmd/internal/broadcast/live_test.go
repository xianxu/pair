package broadcast

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

// TestCloudflaredLive checks the fake's model of cloudflared against the real
// binary and Cloudflare: banner parsing, readiness, a named tunnel's unix
// socket origin, the probe, and teardown. Network; run by hand, and again
// after any cloudflared upgrade:
//
//	BROADCAST_LIVE_CLOUDFLARED=1 go test ./cmd/internal/broadcast -run TestCloudflaredLive -v
//	BROADCAST_LIVE_NAMED=<tunnel>@<hostname> go test ./cmd/internal/broadcast -run TestCloudflaredLive -v
func TestCloudflaredLive(t *testing.T) {
	quick := os.Getenv("BROADCAST_LIVE_CLOUDFLARED") == "1"
	named := os.Getenv("BROADCAST_LIVE_NAMED")
	if !quick && named == "" {
		t.Skip("set BROADCAST_LIVE_CLOUDFLARED=1 and/or BROADCAST_LIVE_NAMED=<tunnel>@<hostname>")
	}
	if quick {
		t.Run("quick", func(t *testing.T) { liveRoundTrip(t, nil) })
	}
	if named != "" {
		name, host, ok := strings.Cut(named, "@")
		if !ok {
			t.Fatalf("BROADCAST_LIVE_NAMED=%q, want <tunnel>@<hostname>", named)
		}
		t.Run("named", func(t *testing.T) { liveRoundTrip(t, &NamedTunnel{Name: name, Hostname: host}) })
	}
}

func liveRoundTrip(t *testing.T, named *NamedTunnel) {
	tunnel := Cloudflared{Named: named, Guard: guardArgv(t), RunDir: shortDir(t), Records: shortDir(t)}
	start := time.Now()
	s, err := Start(context.Background(), Config{Tunnel: tunnel, ProbeTimeout: 60 * time.Second})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Logf("link answered after %v (%s)", time.Since(start).Round(time.Millisecond), strings.SplitN(s.Link(), "/", 4)[2])
	h := s.handle.(*cloudflaredHandle)
	child := h.childPID()
	s.Offer(live(t, "live conformance"), terminal.FramePublic)
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{DialContext: resolvingDialer(PublicResolve)}}
	resp, err := client.Get(s.Link())
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("page through the tunnel, right after the probe: %v %v", resp, err)
	}
	resp.Body.Close()
	resp, err = client.Get(s.Link() + "events")
	if err != nil || resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("event stream through the tunnel: %v %v", resp, err)
	}
	resp.Body.Close()
	s.Stop(nil)
	select {
	case <-s.Done():
	case <-time.After(15 * time.Second):
		t.Fatal("teardown did not finish")
	}
	waitDead(t, strconv.Itoa(child), 5*time.Second)
	if resp, err := client.Get(s.Link()); err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Fatal("the link still serves after Stop")
		}
		t.Logf("after Stop the link answers %d", resp.StatusCode)
	}
}
