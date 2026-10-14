package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestNotifyServiceHubAdmission(t *testing.T) {
	for _, fault := range []string{"none", "worker", "missing-hub", "stale-nonce", "stale-session", "replaced-wrapper", "replaced-during-workspace-check", "disconnected", "no-console", "writer-closed"} {
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
			if fault == "replaced-during-workspace-check" {
				original := r.s.authority.thread
				reads := 0
				r.s.authority.thread = func(ctx context.Context, b couchmessage.Binding) (string, error) {
					root, err := original(ctx, b)
					reads++
					if reads == 2 {
						world.set(hub, func(s *worldSlot) { s.pidFile++ })
					}
					return root, err
				}
			}
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

func TestNotifyLostSocketReplyDoesNotRepeatWrittenNotification(t *testing.T) {
	world := newMessageWorld()
	world.hub = "/repo0"
	r := &serviceRig{t: t, world: world, slotGit: map[string]couchcore.SlotGitStatus{}}
	r.init()
	hub := r.connect(0)
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	console := couchtty.New(host, bytes.NewReader(nil))
	t.Cleanup(func() { console.Stop(); console.Run() })
	r.s.SetDesktopNotify(console.NotifyDesktop)
	// Execute the real service request, then lose its acknowledgment on the
	// socket. The caller cannot conclude that the terminal write failed.
	socket := filepath.Join(filepath.Dir(r.brokerSock), "lost-reply.sock")
	server, err := couchmessage.StartServer(context.Background(), socket, func(ctx context.Context, raw []byte) ([]byte, error) {
		var request couchmessage.Request
		if err := json.Unmarshal(raw, &request); err != nil {
			return nil, err
		}
		response := r.s.handle(ctx, request)
		if response.Code != "ok" {
			return json.Marshal(response)
		}
		return nil, errors.New("reply lost after write")
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	rt := testRT{env: map[string]string{"COUCH_STORE_DIR": "/test/namespace", "COUCH_THREAD_SCOPE": hub.Scope, "COUCH_THREAD_TAG": hub.Tag, "PAIR_SESSION_NAME": hub.Session, "PAIR_LAUNCH_NONCE": hub.Nonce}}
	inv, err := ParseCLI([]string{"--notify", "ready"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := runMessageCLIWithCall(inv, rt, &out, &stderr, func(ctx context.Context, _ string, request, response any) error {
		return couchmessage.Call(ctx, socket, request, response)
	})
	if code != 1 || !strings.Contains(stderr.String(), "do not retry automatically") {
		t.Fatalf("code=%d stderr=%s", code, &stderr)
	}
	if n := strings.Count(host.Written(), "\x1b]777;notify;Couch;ready\x1b\\"); n != 1 {
		t.Fatalf("notifications=%d", n)
	}
}

func TestHubNotifyNamesExtraHubAndBoundsCustomTitle(t *testing.T) {
	w, _ := multiHubWaitRig(t)
	var title string
	w.s.SetDesktopNotify(func(_ context.Context, heading, body string) error { title = heading; return nil })
	req := callerRequest(w.hub, "notify")
	req.Title, req.Body = strings.Repeat("x", 256), "ready"
	got := w.s.handleNotify(context.Background(), req)
	if got.Code != "ok" || !strings.Contains(title, "pair:1") || len(title) > 256 {
		t.Fatalf("response=%+v title=%q", got, title)
	}
}
