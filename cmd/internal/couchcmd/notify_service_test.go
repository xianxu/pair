package couchcmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestNotifyServiceHubAdmission(t *testing.T) {
	for _, fault := range []string{"none", "worker", "missing-hub", "stale-nonce", "stale-session", "replaced-wrapper", "disconnected", "no-console", "writer-closed"} {
		t.Run(fault, func(t *testing.T) {
			world := newMessageWorld()
			if fault != "missing-hub" {
				world.hub = "/repo0"
			}
			r := &serviceRig{t: t, world: world, slotGit: map[string]couchcore.SlotGitStatus{}}
			r.init()
			hub, worker := r.connect(0), r.connect(1)
			host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
			console := couchtty.New(host, bytes.NewReader(nil))
			// Run owns presenter teardown; entering it after Stop performs cleanup
			// without an input reader or a live operator terminal.
			t.Cleanup(func() { console.Stop(); console.Run() })
			if fault != "no-console" {
				r.s.SetDesktopNotify(console.NotifyDesktop)
			}
			caller := hub
			if fault == "worker" {
				caller = worker
			}
			req := callerRequest(caller, "notify")
			req.Title, req.Body = "Couch", "review ready"
			switch fault {
			case "stale-nonce":
				req.Nonce = "old"
			case "stale-session":
				req.Session = "old"
			case "replaced-wrapper":
				world.set(hub, func(s *worldSlot) { s.pidFile++ })
			case "disconnected":
				r.attach(hub, "")
				r.waitConnected(hub, false)
			case "writer-closed":
				_ = host.Close()
			}
			before := world.counts()
			var got couchmessage.Response
			if err := couchmessage.Call(context.Background(), r.brokerSock, req, &got); err != nil {
				t.Fatal(err)
			}
			if (got.Code == "ok") != (fault == "none") {
				t.Fatalf("response %+v", got)
			}
			if fault == "worker" && !strings.Contains(got.Error, "send to the hub instead") {
				t.Fatalf("advice %+v", got)
			}
			if fault == "writer-closed" && got.Code != "uncertain" {
				t.Fatalf("write failure %+v", got)
			}
			want := 0
			if fault == "none" {
				want = 1
			}
			if n := strings.Count(host.Written(), "\x1b]777;notify;Couch;review ready\x1b\\"); n != want {
				t.Fatalf("envelopes=%d want=%d output=%q", n, want, host.Written())
			}
			after := world.counts()
			if before[0] != after[0] || before[1] != after[1] || before[2] != after[2] {
				t.Fatalf("light command invoked probes: %v -> %v", before, after)
			}
		})
	}
}
