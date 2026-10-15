package couchcmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// workbenchMessageEndpoint retains the delivered message IDs. Socket requests
// must reach a real broker delivery before their zero-work measurement passes.
type workbenchMessageEndpoint struct {
	serviceEndpointFake
	mu        sync.Mutex
	delivered map[string]int
}

func (e *workbenchMessageEndpoint) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.delivered == nil {
		e.delivered = map[string]int{}
	}
	e.delivered[m.ID]++
	return couchmessage.Receipt{Status: couchmessage.Submitted}, ctx.Err()
}
func (e *workbenchMessageEndpoint) deliveries(id string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.delivered[id]
}

func TestWorkbenchPerformanceMessageUnchangedEvidence(t *testing.T) {
	const slots = 20
	f := newMessageEvidenceFixture(t, slots, 30_000_000)
	var threadChecks, pidChecks, readyChecks atomic.Int64
	r := newServiceRig(t, func(a messageAuthority) messageAuthority {
		a = f.authority(a)
		thread, pid, recorded := a.thread, a.wrapperPID, a.recorded
		a.thread = func(ctx context.Context, b couchmessage.Binding) (string, error) {
			threadChecks.Add(1)
			return thread(ctx, b)
		}
		a.wrapperPID = func(b couchmessage.Binding) (int, error) { pidChecks.Add(1); return pid(b) }
		a.recorded = func(ctx context.Context, b couchmessage.Binding) error { readyChecks.Add(1); return recorded(ctx, b) }
		return a
	})
	endpoints := make([]*workbenchMessageEndpoint, slots)
	for i := range slots {
		b := f.add(t, r.world, i)
		endpoints[i] = &workbenchMessageEndpoint{}
		r.world.mu.Lock()
		r.world.endpoints[b] = endpoints[i]
		r.world.mu.Unlock()
		r.attach(b, couchmessage.PaneHandle(fmt.Sprintf("h%d", i+1)))
		r.wrapper(b)
		r.waitConnected(b, true)
	}
	cold := f.Counts.Snapshot()
	if cold.ReadFamily("index") == 0 || cold.DecodeFamily("index") == 0 {
		t.Fatalf("cold service counters disconnected: %+v", cold)
	}
	for _, p := range f.indexes {
		var read, decoded int64
		for k, n := range cold.Read {
			if k.Path == p {
				read += n
			}
		}
		for k, n := range cold.Decoded {
			if k.Path == p {
				decoded += n
			}
		}
		if read == 0 || decoded == 0 {
			t.Fatalf("cold owner was never read/decoded: %s", p)
		}
	}
	call := func(request couchmessage.Request) couchmessage.Response {
		t.Helper()
		var resp couchmessage.Response
		if e := couchmessage.Call(context.Background(), r.brokerSock, request, &resp); e != nil {
			t.Fatal(e)
		}
		return resp
	}
	sequence := 0
	send := func(from, to int) {
		t.Helper()
		sequence++
		request := callerRequest(f.bindings[from], "send")
		request.ID, request.Target, request.Body = fmt.Sprintf("workbench-%d", sequence), f.bindings[to].Slot, "count this completed peer message"
		beforeThread, beforePID, beforeReady := threadChecks.Load(), pidChecks.Load(), readyChecks.Load()
		resp := call(request)
		if resp.Code != "accepted" || resp.Receipt == nil {
			t.Fatalf("send %s: %+v", request.ID, resp)
		}
		status := callerRequest(f.bindings[from], "status")
		status.ID = request.ID
		testwait.Until(t, "submitted socket message "+request.ID, func() bool {
			resp = call(status)
			if resp.Code != "ok" || resp.Receipt == nil {
				t.Fatalf("status %s: %+v", request.ID, resp)
			}
			switch resp.Receipt.Status {
			case couchmessage.Submitted:
				return true
			case couchmessage.Queued, couchmessage.Delivering:
				return false
			default:
				t.Fatalf("message %s did not submit: %+v", request.ID, resp.Receipt)
				return false
			}
		})
		if n := endpoints[to].deliveries(request.ID); n != 1 {
			t.Fatalf("message %s delivered %d times", request.ID, n)
		}
		// The sender and recipient must still consult live thread, PID and ready
		// evidence. A shortcut that avoids authority cannot satisfy zero work.
		if threadChecks.Load()-beforeThread < 2 || pidChecks.Load()-beforePID < 2 || readyChecks.Load()-beforeReady < 2 {
			t.Fatal("send skipped fresh sender/recipient authority")
		}
	}
	for range 3 {
		for i := range slots {
			before := f.Counts.Snapshot()
			send(i, (i+1)%slots)
			f.Counts.Snapshot().Delta(before).RequireZero(t)
		}
	}
	// A same-size index rewrite preserves the binding but must invalidate only
	// this owner's parsed index. The following socket message reuses it.
	original, e := os.ReadFile(f.indexes[7])
	if e != nil {
		t.Fatal(e)
	}
	before := f.Counts.Snapshot()
	if e = os.WriteFile(f.indexes[7], []byte(strings.Replace(string(original), "project", "PROJECT", 1)), 0600); e != nil {
		t.Fatal(e)
	}
	send(7, 8)
	requireMessageIndexMutation(t, f.Counts.Snapshot().Delta(before), f.indexes[7])
	before = f.Counts.Snapshot()
	send(7, 8)
	f.Counts.Snapshot().Delta(before).RequireZero(t)

	refused := func(i int) {
		t.Helper()
		sequence++
		request := callerRequest(f.bindings[i], "send")
		request.ID, request.Target, request.Body = fmt.Sprintf("refused-%d", sequence), f.bindings[(i+1)%slots].Slot, "must not deliver"
		resp := call(request)
		if resp.Code != "unavailable" {
			t.Fatalf("stale caller retained authority: %+v", resp)
		}
		if n := endpoints[(i+1)%slots].deliveries(request.ID); n != 0 {
			t.Fatalf("refused message delivered %d times", n)
		}
	}
	// Replacement while the old wrapper survives must revoke its authority;
	// repeated refusal still decodes no unchanged evidence.
	replacement := f.indexes[7] + ".new"
	if e = os.WriteFile(replacement, []byte(messageIndexLine(t, f.bindings[7], "pair-replaced")), 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(replacement, f.indexes[7]); e != nil {
		t.Fatal(e)
	}
	refused(7)
	before = f.Counts.Snapshot()
	refused(7)
	f.Counts.Snapshot().Delta(before).RequireZero(t)
	// Ready and PID contents remain fresh even when historical indexes are warm.
	paths, e := artifactpath.Resolve(artifactpath.Address{DataDir: f.DataDir, RepoScope: f.bindings[0].Scope, Tag: f.bindings[0].Tag})
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(paths.PairWrapPID(), []byte("1"), 0600); e != nil {
		t.Fatal(e)
	}
	before = f.Counts.Snapshot()
	refused(0)
	f.Counts.Snapshot().Delta(before).RequireZero(t)
	b := f.bindings[1]
	paths, e = artifactpath.Resolve(artifactpath.Address{DataDir: f.DataDir, RepoScope: b.Scope, Tag: b.Tag})
	if e != nil {
		t.Fatal(e)
	}
	raw, e := readiness.Encode(readiness.ReadyRecord{Tag: b.Tag, Agent: b.Agent, Session: b.Session, Nonce: "replacement", PID: b.PID})
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(paths.AgentReady(b.Agent), []byte(raw), 0600); e != nil {
		t.Fatal(e)
	}
	before = f.Counts.Snapshot()
	refused(1)
	f.Counts.Snapshot().Delta(before).RequireZero(t)
	r.world.set(f.bindings[2], func(s *worldSlot) { s.live = false })
	before = f.Counts.Snapshot()
	refused(2)
	f.Counts.Snapshot().Delta(before).RequireZero(t)
	t.Logf("20 owners x30,000,000 ledger bytes; cold index read=%d decoded=%d; each completed warm message read=0 decoded=0", cold.ReadFamily("index"), cold.DecodeFamily("index"))
}
