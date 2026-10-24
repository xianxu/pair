package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func nativeExitRequest(b couchmessage.Binding, ordinal uint64, identity string, code int) couchmessage.Request {
	r := callerRequest(b, "native-exit")
	r.ID = couchmessage.NativeExitID(b, ordinal)
	raw, _ := json.Marshal(couchmessage.NativeExit{LaunchOrdinal: ordinal, AgentIdentity: identity, ExitCode: code})
	r.Body = string(raw)
	return r
}

// exitRig is the hub rig with a recorded-child fake (the caller's ready
// record: launch ordinal and agent identity) and a recorder for the exited
// facts handed to the lifecycle notices.
type exitRig struct {
	*serviceRig
	mu       sync.Mutex
	ordinal  uint64
	identity string
	exits    []couchmessage.NativeExit
}

func newExitRig(t *testing.T, decorate ...func(messageAuthority) messageAuthority) (*exitRig, couchmessage.Binding) {
	e := &exitRig{ordinal: 1, identity: "100:child-one"}
	r, bindings := hubMessageRig(t, func(a messageAuthority) messageAuthority {
		a.child = func(context.Context, couchmessage.Binding) (uint64, string, error) {
			e.mu.Lock()
			defer e.mu.Unlock()
			return e.ordinal, e.identity, nil
		}
		if len(decorate) > 0 {
			a = decorate[0](a)
		}
		return a
	})
	e.serviceRig = r
	exited := func(_ couchmessage.Binding, _ couchcore.WorkspaceIdentity, report couchmessage.NativeExit) {
		e.mu.Lock()
		e.exits = append(e.exits, report)
		e.mu.Unlock()
	}
	r.s.exited.Store(&exited)
	return e, bindings[2]
}

func (e *exitRig) call(t *testing.T, req couchmessage.Request) couchmessage.Response {
	t.Helper()
	var response couchmessage.Response
	if err := couchmessage.Call(context.Background(), e.brokerSock, req, &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func (e *exitRig) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.exits)
}

// One confirmed exit of the current child posts exactly one exited fact; every
// report that is not about the caller's current child, from the current
// wrapper, while connected, posts none (pair#657).
func TestNativeExitAdmitsOnlyTheCurrentChild(t *testing.T) {
	for _, fault := range []string{"none", "stale-ordinal", "stale-identity", "replaced", "disconnected", "wrong-id", "stale-nonce"} {
		t.Run(fault, func(t *testing.T) {
			e, worker := newExitRig(t)
			req := nativeExitRequest(worker, 1, "100:child-one", 3)
			switch fault {
			case "stale-ordinal":
				e.ordinal = 2 // the wrapper restarted its child (SIGUSR2): same binding, next launch
			case "stale-identity":
				e.identity = "200:child-two"
			case "replaced":
				e.world.set(worker, func(s *worldSlot) { s.pidFile++ })
			case "disconnected":
				e.disconnect(worker)
			case "wrong-id":
				req.ID = "native-exit-other"
			case "stale-nonce":
				req.Nonce = "old"
			}
			got := e.call(t, req)
			if fault == "none" {
				if got.Code != "ok" || e.count() != 1 || e.exits[0].ExitCode != 3 {
					t.Fatalf("current exit: %+v, %d facts", got, e.count())
				}
				// A repeat of the same (binding, launch) posts nothing more.
				if again := e.call(t, req); again.Code != "ok" || e.count() != 1 {
					t.Fatalf("duplicate: %+v, %d facts", again, e.count())
				}
				return
			}
			if got.Code == "ok" || e.count() != 0 {
				t.Fatalf("%s accepted: %+v, %d facts", fault, got, e.count())
			}
		})
	}
}

// The next child's exit (after a restart moves the launch on) is a new fact;
// a report about the old launch after that is stale.
func TestNativeExitEachLaunchOnce(t *testing.T) {
	e, worker := newExitRig(t)
	if got := e.call(t, nativeExitRequest(worker, 1, "100:child-one", 0)); got.Code != "ok" {
		t.Fatal(got)
	}
	e.mu.Lock()
	e.ordinal, e.identity = 2, "200:child-two"
	e.mu.Unlock()
	if got := e.call(t, nativeExitRequest(worker, 2, "200:child-two", 1)); got.Code != "ok" || e.count() != 2 {
		t.Fatalf("next launch: %+v, %d facts", got, e.count())
	}
	if got := e.call(t, nativeExitRequest(worker, 1, "100:child-one", 0)); got.Code != "stale" || e.count() != 2 {
		t.Fatalf("old launch after the next: %+v, %d facts", got, e.count())
	}
}

// Detach, reattach and an attach-client exit are not native exits: without a
// wrapper report nothing is posted, and the wrapper's report stays valid
// across them because its session never closed.
func TestNativeExitNotInferredFromAttachment(t *testing.T) {
	e, worker := newExitRig(t)
	if err := e.s.post(couchmessage.RegistryEvent{Kind: couchmessage.PaneChanged, Thread: worker.Thread()}, true); err != nil {
		t.Fatal(err)
	}
	if e.count() != 0 {
		t.Fatal("an attachment change posted an exit")
	}
	if got := e.call(t, nativeExitRequest(worker, 1, "100:child-one", 0)); got.Code != "ok" || e.count() != 1 {
		t.Fatalf("report after a detach: %+v, %d facts", got, e.count())
	}
}

// A Couch without the recorded-child seam refuses rather than guessing.
func TestNativeExitWithoutChildEvidenceIsUnsupported(t *testing.T) {
	e, worker := newExitRig(t, func(a messageAuthority) messageAuthority {
		a.child = nil
		return a
	})
	if got := e.call(t, nativeExitRequest(worker, 1, "100:child-one", 0)); got.Code != "unsupported" || e.count() != 0 {
		t.Fatalf("no evidence seam: %+v, %d facts", got, e.count())
	}
}

// The whole chain the Done-when names (close review BR-2): a wrapper's
// native-exit report reaches the owning hub as one lifecycle system notice,
// through the production wiring (handler, Console sequence, publisher).
func TestNativeExitReachesTheHubAsOneLifecycleNotice(t *testing.T) {
	ep := newControlledEndpoint()
	r, publisher, _ := lifecyclePublisherRigWithAuthority(t, ep, func(a messageAuthority) messageAuthority {
		a.child = func(context.Context, couchmessage.Binding) (uint64, string, error) { return 5, "500:child", nil }
		return a
	})
	worker := r.world.add(2)
	r.attach(worker, "worker")
	r.wrapper(worker)
	r.waitConnected(worker, true)
	console := couchtty.New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), bytes.NewReader(nil))
	t.Cleanup(func() { console.Stop(); console.Run() })
	console.SetLifecycleObserver(publisher.observe)
	wireNativeExitNotices(r.s, console)

	var response couchmessage.Response
	if err := couchmessage.Call(context.Background(), r.brokerSock, nativeExitRequest(worker, 5, "500:child", 3), &response); err != nil || response.Code != "ok" {
		t.Fatalf("report: %+v %v", response, err)
	}
	m := lifecycleNextMessage(t, ep, publisher)
	if m.Origin != couchmessage.OriginSystem || !strings.Contains(m.Body, worker.Slot+": exited (code 3)") {
		t.Fatalf("hub notice: %+v", m)
	}
	ep.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	lifecycleCanonicalSubmitted(t, r, m)
	// The duplicate report posts nothing more.
	if err := couchmessage.Call(context.Background(), r.brokerSock, nativeExitRequest(worker, 5, "500:child", 3), &response); err != nil || response.Code != "ok" {
		t.Fatalf("duplicate: %+v %v", response, err)
	}
	select {
	case extra := <-ep.started:
		t.Fatalf("a duplicate report posted again: %+v", extra)
	case <-time.After(300 * time.Millisecond):
	}
}
