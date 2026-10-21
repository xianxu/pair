package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func stallRequest(b couchmessage.Binding, sequence uint64) couchmessage.Request {
	r := callerRequest(b, "automatic-input-stall")
	r.ID = couchmessage.AutomaticInputStallID(b, sequence)
	raw, _ := json.Marshal(couchmessage.AutomaticInputStall{Sequence: sequence, Cause: couchmessage.AutomaticInputStallNoRender, WaitMillis: 30000, Fence: 3, Epoch: 3, Released: true})
	r.Body = string(raw)
	return r
}
func stallCall(t *testing.T, r *serviceRig, req couchmessage.Request) couchmessage.Response {
	t.Helper()
	var response couchmessage.Response
	if err := couchmessage.Call(context.Background(), r.brokerSock, req, &response); err != nil {
		t.Fatal(err)
	}
	return response
}
func TestAutomaticInputStallSupervisorEffects(t *testing.T) {
	for _, fault := range []string{"none", "unfocused", "stale-nonce", "stale-session", "replaced", "disconnected", "wrong-id", "missing-hub", "closed-desktop", "hub-origin"} {
		t.Run(fault, func(t *testing.T) {
			r, bindings := hubMessageRig(t)
			hub, worker := bindings[1], bindings[2]
			host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
			console := couchtty.New(host, bytes.NewReader(nil))
			t.Cleanup(func() { console.Stop(); console.Run() })
			r.s.SetDesktopNotify(console.NotifyDesktop)
			var notices []string
			var mu sync.Mutex
			c := newSupervisionCoordinator(r.s, func(s string) { mu.Lock(); notices = append(notices, s); mu.Unlock() })
			r.s.supervision.Store(c)
			caller := worker
			if fault == "hub-origin" {
				caller = hub
			}
			req := stallRequest(caller, 2)
			switch fault {
			case "stale-nonce":
				req.Nonce = "old"
			case "stale-session":
				req.Session = "old"
			case "replaced":
				r.world.set(worker, func(s *worldSlot) { s.pidFile++ })
			case "unfocused":
				// Detaching a local client leaves the authenticated wrapper live.
				if err := r.s.post(couchmessage.RegistryEvent{Kind: couchmessage.PaneChanged, Thread: worker.Thread()}, true); err != nil {
					t.Fatal(err)
				}
			case "disconnected":
				r.disconnect(worker)
			case "wrong-id":
				req.ID = "other"
			case "missing-hub":
				r.disconnect(hub)
			case "closed-desktop":
				_ = host.Close()
			}
			got := stallCall(t, r, req)
			valid := fault == "none" || fault == "unfocused" || fault == "missing-hub" || fault == "closed-desktop" || fault == "hub-origin"
			if valid && got.Code != "ok" && got.Code != "uncertain" {
				t.Fatalf("response %+v", got)
			}
			if !valid && (got.Code == "ok" || got.Code == "uncertain") {
				t.Fatalf("accepted %+v", got)
			}
			// A repeated or reordered occurrence must have no further effects, including failed/uncertain attempts.
			_ = stallCall(t, r, req)
			old := stallRequest(caller, 1)
			if fault == "stale-nonce" {
				old.Nonce = "old"
			}
			if valid {
				_ = stallCall(t, r, old)
			}
			mu.Lock()
			defer mu.Unlock()
			want := 0
			if valid {
				want = 1
			}
			if len(notices) != want {
				t.Fatalf("notices=%q", notices)
			}
			n := strings.Count(host.Written(), "\x1b]777;notify;")
			if valid && fault != "closed-desktop" {
				if n != 1 {
					t.Fatalf("notifications=%d: %q", n, host.Written())
				}
			} else if n != 0 {
				t.Fatalf("notifications=%d", n)
			}
			if valid && fault != "missing-hub" && fault != "hub-origin" {
				receipt, err := r.s.broker.Status(caller, req.ID)
				if err != nil || receipt.Message.To != hub {
					t.Fatalf("hub receipt %+v %v", receipt, err)
				}
			}
		})
	}
}
func TestAutomaticInputStallReconnectAndCapacity(t *testing.T) {
	r, b := hubMessageRig(t)
	worker := b[2]
	calls := 0
	r.s.SetDesktopNotify(func(context.Context, string, string) error { calls++; return nil })
	r.s.supervision.Store(newSupervisionCoordinator(r.s, func(string) {}))
	if got := stallCall(t, r, stallRequest(worker, 2)); got.Code != "ok" {
		t.Fatal(got)
	}
	r.disconnect(worker)
	// The same wrapper incarnation opens a new registry connection; it does
	// not need a client pane to recover its previous stall sequence.
	r.wrapper(worker)
	r.waitConnected(worker, true)
	_ = stallCall(t, r, stallRequest(worker, 2))
	_ = stallCall(t, r, stallRequest(worker, 1))
	if calls != 1 {
		t.Fatalf("reconnect effects=%d", calls)
	}
	r.s.stalls.Lock()
	for i := 0; len(r.s.stalls.records) < couchmessage.MaxActors; i++ {
		r.s.stalls.records[fmt.Sprintf("filler:%d", i)] = automaticInputStallRecord{Binding: worker, Sequence: 2}
	}
	r.s.stalls.Unlock()
	got := stallCall(t, r, stallRequest(b[3], 1))
	if got.Code != "refused" {
		t.Fatalf("capacity %+v", got)
	}
	got = stallCall(t, r, stallRequest(worker, 3))
	if got.Code != "ok" || calls != 2 {
		t.Fatalf("retained slot %+v effects %d", got, calls)
	}
}
func TestAutomaticInputStallOwningHub(t *testing.T) {
	r, b := hubMessageRig(t)
	r.s.supervision.Store(newSupervisionCoordinator(r.s, func(string) {}))
	r.s.SetDesktopNotify(func(context.Context, string, string) error { return nil })
	for _, pair := range [][2]int{{2, 1}, {3, 0}} {
		got := stallCall(t, r, stallRequest(b[pair[0]], uint64(pair[0])))
		if got.Code != "ok" {
			t.Fatal(got)
		}
		receipt, err := r.s.broker.Status(b[pair[0]], stallRequest(b[pair[0]], uint64(pair[0])).ID)
		if err != nil || receipt.Message.To != b[pair[1]] {
			t.Fatalf("routing %+v %v", receipt, err)
		}
	}
}

func TestAutomaticInputStallReservesBeforeBlockedIO(t *testing.T) {
	r, b := hubMessageRig(t)
	worker := b[2]
	entered, release := make(chan struct{}), make(chan struct{})
	r.s.supervision.Store(newSupervisionCoordinator(r.s, func(string) {}))
	r.s.SetDesktopNotify(func(context.Context, string, string) error {
		close(entered)
		<-release
		return fmt.Errorf("write outcome unknown")
	})
	first := make(chan couchmessage.Response, 1)
	go func() {
		var got couchmessage.Response
		err := couchmessage.Call(context.Background(), r.brokerSock, stallRequest(worker, 2), &got)
		if err != nil {
			got.Error = err.Error()
		}
		first <- got
	}()
	<-entered
	// A duplicate completes while the first desktop write is still blocked.
	duplicate := make(chan couchmessage.Response, 1)
	go func() {
		var got couchmessage.Response
		_ = couchmessage.Call(context.Background(), r.brokerSock, stallRequest(worker, 2), &got)
		duplicate <- got
	}()
	select {
	case got := <-duplicate:
		if got.Code != "ok" {
			t.Errorf("duplicate %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Error("duplicate blocked behind IO")
	}
	close(release)
	if got := <-first; got.Code != "uncertain" {
		t.Fatalf("first %+v", got)
	}
	if got := stallCall(t, r, stallRequest(worker, 2)); got.Code != "ok" {
		t.Fatal(got)
	}
}

func TestAutomaticInputStallVerifiedReplacementResetsSequence(t *testing.T) {
	r := newServiceRig(t)
	worker := r.connect(2)
	calls := 0
	r.s.supervision.Store(newSupervisionCoordinator(r.s, func(string) {}))
	r.s.SetDesktopNotify(func(context.Context, string, string) error { calls++; return nil })
	_ = stallCall(t, r, stallRequest(worker, 20))
	r.disconnect(worker)
	replacement := worker
	replacement.Nonce = "replacement"
	r.world.set(worker, func(s *worldSlot) { s.binding = replacement })
	r.wrapper(replacement)
	r.waitConnected(replacement, true)
	if got := stallCall(t, r, stallRequest(worker, 21)); got.Code != "refused" {
		t.Fatalf("old incarnation %+v", got)
	}
	_ = stallCall(t, r, stallRequest(replacement, 1))
	if calls != 2 {
		t.Fatalf("replacement effects=%d", calls)
	}
}

func TestAutomaticInputStallPIDReuseAdmitsBothHubMessages(t *testing.T) {
	r, b := hubMessageRig(t)
	worker, hub := b[2], b[1]
	r.s.supervision.Store(newSupervisionCoordinator(r.s, func(string) {}))
	r.s.SetDesktopNotify(func(context.Context, string, string) error { return nil })
	first := stallRequest(worker, 1)
	if got := stallCall(t, r, first); got.Code != "ok" {
		t.Fatal(got)
	}
	r.disconnect(worker)
	replacement := worker
	replacement.Nonce = "new-launch"
	replacement.Start = "reused-pid-start"
	r.world.set(worker, func(s *worldSlot) { s.binding = replacement })
	r.wrapper(replacement)
	r.waitConnected(replacement, true)
	second := stallRequest(replacement, 1)
	if first.ID == second.ID {
		t.Fatal("PID reuse collided with prior occurrence ID")
	}
	if got := stallCall(t, r, second); got.Code != "ok" {
		t.Fatal(got)
	}
	for _, req := range []couchmessage.Request{first, second} {
		from := worker
		if req.ID == second.ID {
			from = replacement
		}
		receipt, known, _ := r.s.broker.ReceiptSnapshot(req.ID)
		if !known || receipt.Message.To != hub || receipt.Message.From != from {
			t.Fatalf("receipt %+v known=%v", receipt, known)
		}
	}
}

func TestAutomaticInputStallUnsubmittedComposerNotice(t *testing.T) {
	r, b := hubMessageRig(t)
	worker, hub := b[2], b[1]
	var mu sync.Mutex
	var notices, desktop []string
	r.s.supervision.Store(newSupervisionCoordinator(r.s, func(body string) { mu.Lock(); notices = append(notices, body); mu.Unlock() }))
	r.s.SetDesktopNotify(func(_ context.Context, _ string, body string) error {
		mu.Lock()
		desktop = append(desktop, body)
		mu.Unlock()
		return nil
	})
	req := stallRequest(worker, 2)
	raw, err := json.Marshal(couchmessage.AutomaticInputStall{Sequence: 2, Cause: couchmessage.AutomaticInputStallUnsubmittedComposer, WaitMillis: 30000, Fence: 3, Epoch: 4})
	if err != nil {
		t.Fatal(err)
	}
	req.Body = string(raw)
	// New causes retain the same caller authentication and sequence reservation.
	foreign := req
	foreign.Nonce = "foreign"
	if got := stallCall(t, r, foreign); got.Code != "refused" {
		t.Fatalf("foreign caller %+v", got)
	}
	if got := stallCall(t, r, req); got.Code != "ok" {
		t.Fatal(got)
	}
	if got := stallCall(t, r, req); got.Code != "ok" {
		t.Fatal(got)
	}
	older := req
	older.ID = couchmessage.AutomaticInputStallID(worker, 1)
	older.Body = strings.Replace(req.Body, `"Sequence":2`, `"Sequence":1`, 1)
	if got := stallCall(t, r, older); got.Code != "ok" {
		t.Fatal(got)
	}
	receipt, err := r.s.broker.Status(worker, req.ID)
	if err != nil || receipt.Message.To != hub {
		t.Fatalf("owning hub receipt %+v %v", receipt, err)
	}
	want := worker.Slot + "'s composer holds an unsubmitted automatic message; press Alt+Enter or clear it"
	mu.Lock()
	defer mu.Unlock()
	if len(notices) != 1 || len(desktop) != 1 {
		t.Fatalf("notices=%q desktop=%q", notices, desktop)
	}
	for _, body := range []string{notices[0], desktop[0], receipt.Message.Body} {
		if !strings.Contains(body, want) {
			t.Errorf("missing recovery instruction %q in %q", want, body)
		}
		if !strings.Contains(body, "unsubmitted-composer") || !strings.Contains(body, "sequence 2") {
			t.Errorf("lost typed diagnostic evidence: %q", body)
		}
	}
}

func TestAutomaticInputStallAutomationOffRetiresOccurrenceKeepsOutputs(t *testing.T) {
	r, b := hubMessageRig(t)
	worker := b[2]
	r.s.automation = newHubAutomationOwner(map[string]bool{"/repo0": true, "/repo1": false}, couchcore.StartupUnknown, nil)
	var status, desktop atomic.Int64
	r.s.supervision.Store(newSupervisionCoordinator(r.s, func(string) { status.Add(1) }))
	r.s.SetDesktopNotify(func(context.Context, string, string) error { desktop.Add(1); return nil })
	req := stallRequest(worker, 2)
	got := stallCall(t, r, req)
	if _, ok, _ := r.s.broker.ReceiptSnapshot(req.ID); ok {
		t.Fatal("automation off admitted generated hub input")
	}
	if got.Code != "ok" || got.Error != "" {
		t.Fatalf("intentional suppression manufactured a delivery failure: %+v", got)
	}
	if status.Load() != 1 || desktop.Load() != 1 {
		t.Fatalf("suppression hid local output: status=%d desktop=%d", status.Load(), desktop.Load())
	}
	if _, err := r.s.controlAutomation("/repo1", "on"); err != nil {
		t.Fatal(err)
	}
	for _, sequence := range []uint64{2, 1} {
		got = stallCall(t, r, stallRequest(worker, sequence))
		if got.Code != "ok" {
			t.Fatal(got)
		}
	}
	if _, ok, _ := r.s.broker.ReceiptSnapshot(req.ID); ok {
		t.Fatal("turning automation on replayed old occurrence")
	}
	if status.Load() != 1 || desktop.Load() != 1 {
		t.Fatal("reserved occurrence repeated desktop or status output")
	}
	// A distinct new stall remains eligible after explicit enabling.
	next := stallRequest(worker, 3)
	if got = stallCall(t, r, next); got.Code != "ok" {
		t.Fatal(got)
	}
	if receipt, ok, _ := r.s.broker.ReceiptSnapshot(next.ID); !ok || receipt.Message.To != b[1] {
		t.Fatalf("new occurrence lost hub delivery: %+v", receipt)
	}
	if status.Load() != 2 || desktop.Load() != 2 {
		t.Fatal("new occurrence lost independent output")
	}
}

func TestAutomaticInputStallPolicyEpochFencesBlockedStatusOutput(t *testing.T) {
	r, b := hubMessageRig(t)
	worker := b[2]
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	var statuses, desktop atomic.Int64
	r.s.supervision.Store(newSupervisionCoordinator(r.s, func(string) {
		if statuses.Add(1) == 1 {
			close(entered)
			<-release
		}
	}))
	r.s.SetDesktopNotify(func(context.Context, string, string) error { desktop.Add(1); return nil })
	req := stallRequest(worker, 2)
	done := make(chan couchmessage.Response, 1)
	go func() {
		var result couchmessage.Response
		if err := couchmessage.Call(context.Background(), r.brokerSock, req, &result); err != nil {
			result.Error = err.Error()
		}
		done <- result
	}()
	select {
	case <-entered:
	case <-testwait.After(t):
		t.Fatal("status output did not enter")
	}
	if _, err := r.s.controlAutomation("/repo1", "off"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.s.controlAutomation("/repo1", "on"); err != nil {
		t.Fatal(err)
	}
	unblock()
	select {
	case got := <-done:
		if got.Code != "ok" {
			t.Fatal(got)
		}
	case <-testwait.After(t):
		t.Fatal("stall notice did not finish")
	}
	if _, known, _ := r.s.broker.ReceiptSnapshot(req.ID); known {
		t.Fatal("stale reserved stall crossed off-on policy epoch")
	}
	if statuses.Load() != 1 || desktop.Load() != 1 {
		t.Fatal("policy change hid independent local outputs")
	}
	if got := stallCall(t, r, stallRequest(worker, 3)); got.Code != "ok" {
		t.Fatal(got)
	}
	if _, known, _ := r.s.broker.ReceiptSnapshot(stallRequest(worker, 3).ID); !known {
		t.Fatal("fresh stall did not qualify after on")
	}
}
