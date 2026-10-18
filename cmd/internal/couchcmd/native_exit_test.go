package couchcmd

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
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

func newExitRig(t *testing.T) (*exitRig, couchmessage.Binding) {
	r, bindings := hubMessageRig(t)
	e := &exitRig{serviceRig: r, ordinal: 1, identity: "100:child-one"}
	r.s.authority.child = func(context.Context, couchmessage.Binding) (uint64, string, error) {
		e.mu.Lock()
		defer e.mu.Unlock()
		return e.ordinal, e.identity, nil
	}
	exited := func(_ couchmessage.Binding, report couchmessage.NativeExit) {
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
	e, worker := newExitRig(t)
	e.s.authority.child = nil
	if got := e.call(t, nativeExitRequest(worker, 1, "100:child-one", 0)); got.Code != "unsupported" || e.count() != 0 {
		t.Fatalf("no evidence seam: %+v, %d facts", got, e.count())
	}
}
