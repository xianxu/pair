package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// transportDrainWrapper is a stateful wrapper-side protocol fake. It owns
// reservations and retained outcomes, while the real Server/RemoteEndpoint
// pair owns framing, peer credentials, request validation and delivery polling.
// Actual PTY/input ownership is covered separately by wrapcmd's drain tests.
type transportDrainWrapper struct {
	mu                 sync.Mutex
	binding            couchmessage.Binding
	endpoint           *drainServiceEndpoint
	reserved           map[string]couchmessage.DrainFence
	calls              map[string]int
	order              []string
	loseInterruptReply bool
}

func (w *transportDrainWrapper) handle(ctx context.Context, raw []byte) ([]byte, error) {
	var request couchmessage.EndpointRequest
	if err := strictjson.Decode(raw, &request); err != nil {
		return nil, err
	}
	if err := couchmessage.ValidateEndpointRequest(request); err != nil {
		return nil, err
	}
	if request.Binding != w.binding {
		return nil, errors.New("wrong exact wrapper binding")
	}
	w.mu.Lock()
	w.calls[request.Op]++
	w.mu.Unlock()
	response := couchmessage.EndpointResponse{}
	switch request.Op {
	case "observe":
		response.Observation, _ = w.endpoint.Observe(ctx)
	case "reserve":
		return nil, errors.New("ordinary reservation cannot authorize a drain")
	case "drain-reserve":
		response.DrainFence = request.DrainFence
		if err := w.endpoint.ReserveDrain(ctx, request.ID, request.Sequence, *request.DrainFence); err != nil {
			return nil, err
		}
		w.mu.Lock()
		for id, fence := range w.reserved {
			if fence.Epoch < request.DrainFence.Epoch {
				delete(w.reserved, id)
			}
		}
		w.reserved[request.ID] = *request.DrainFence
		w.mu.Unlock()
	case "release":
		return nil, errors.New("ordinary release cannot authorize a drain")
	case "drain-release":
		response.DrainFence = request.DrainFence
		if err := w.endpoint.ReleaseDrain(ctx, request.ID, *request.DrainFence); err != nil {
			return nil, err
		}
		w.mu.Lock()
		delete(w.reserved, request.ID)
		w.mu.Unlock()
	case "commit":
		w.mu.Lock()
		reserved, ok := w.reserved[request.ID]
		w.mu.Unlock()
		if !ok || reserved != request.Message.DrainFence {
			return nil, errors.New("commit without input reservation")
		}
		observed, err := w.endpoint.ObserveDrain(ctx)
		if err != nil {
			return nil, err
		}
		if request.Message.DrainFence.Instance != observed.Instance {
			return nil, errors.New("commit for replaced wrapper instance")
		}
		receipt, err := w.endpoint.Deliver(ctx, *request.Message, func(couchmessage.Receipt) {})
		if err != nil {
			return nil, err
		}
		response.Receipt = &receipt
		w.mu.Lock()
		w.order = append(w.order, "notice")
		w.mu.Unlock()
	case "status":
		w.endpoint.mu.Lock()
		receipt, ok := w.endpoint.receipts[request.ID]
		w.endpoint.mu.Unlock()
		if !ok {
			return nil, errors.New("unknown receipt")
		}
		response.Receipt = &receipt
	case "drain-observe":
		observed, err := w.endpoint.ObserveDrain(ctx)
		if err != nil {
			return nil, err
		}
		response.DrainObservation = &observed
	case "drain-interrupt":
		result, err := w.endpoint.InterruptDrain(ctx, request.ID, *request.DrainFence, *request.DrainObserved)
		if err != nil {
			return nil, err
		}
		response.DrainInterrupt = &result
		w.mu.Lock()
		w.order = append(w.order, "interrupt")
		lose := w.loseInterruptReply
		w.loseInterruptReply = false
		w.mu.Unlock()
		if lose {
			return nil, errors.New("transport closes after retained interrupt effect")
		}
	case "drain-interrupt-status":
		result, err := w.endpoint.DrainInterruptStatus(ctx, request.ID, *request.DrainFence)
		if err != nil {
			return nil, err
		}
		response.DrainInterrupt = &result
	case "drain-revoke":
		result, err := w.endpoint.RevokeDrain(ctx, request.ID, *request.DrainFence)
		if err != nil {
			return nil, err
		}
		response.DrainRevoke = &result
	default:
		return nil, fmt.Errorf("unexpected protocol operation %s", request.Op)
	}
	return json.Marshal(response)
}
func (w *transportDrainWrapper) callCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, v := range w.calls {
		n += v
	}
	return n
}

func TestDrainTransportThreeMembers(t *testing.T) {
	for _, fast := range []bool{false, true} {
		name := "graceful"
		if fast {
			name = "fast-lost-interrupt-reply"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			namespace := t.TempDir()
			namespaceKey, err := couchmessage.DrainNamespaceKey(namespace)
			if err != nil {
				t.Fatal(err)
			}
			broker := couchmessage.NewBroker(ctx, time.Now, nil)
			defer broker.Close()
			fleet := &drainServiceFleet{members: map[string]drainLiveMember{}, endpoints: map[couchmessage.Binding]*drainServiceEndpoint{}}
			wrappers := make(map[couchmessage.Binding]*transportDrainWrapper)
			remotes := make(map[couchmessage.Binding]couchmessage.RemoteEndpoint)
			var servers []*couchmessage.Server
			var sockets []string
			defer func() {
				for _, server := range servers {
					_ = server.Close()
				}
				for _, socket := range sockets {
					if _, err := os.Lstat(socket); !errors.Is(err, os.ErrNotExist) {
						t.Errorf("owned transport socket survived close: %s: %v", socket, err)
					}
				}
			}()
			for i := 0; i < 3; i++ {
				binding := couchmessage.Binding{Slot: fmt.Sprintf("pair:%d", i), Repository: filepath.Join(namespace, "repo.git"), Scope: "scope", Tag: fmt.Sprint(i), Session: "session", Nonce: fmt.Sprintf("nonce-%d", i), Agent: "codex", Version: "test", PID: os.Getpid(), Start: "start"}
				wrapper := &transportDrainWrapper{binding: binding, endpoint: &drainServiceEndpoint{namespace: namespaceKey}, reserved: map[string]couchmessage.DrainFence{}, calls: map[string]int{}, loseInterruptReply: fast && i == 1}
				socket, err := couchmessage.EndpointSocket(namespace, binding)
				if err != nil {
					t.Fatal(err)
				}
				server, err := couchmessage.StartServer(ctx, socket, wrapper.handle)
				if err != nil {
					t.Fatal(err)
				}
				servers = append(servers, server)
				sockets = append(sockets, socket)
				remote := couchmessage.RemoteEndpoint{Namespace: namespace, Binding: binding}
				if err := broker.Register(binding, remote); err != nil {
					t.Fatal(err)
				}
				wrappers[binding] = wrapper
				remotes[binding] = remote
				fleet.endpoints[binding] = wrapper.endpoint
				fleet.members[binding.Slot] = drainLiveMember{Binding: binding, Session: "admitted", Observation: couchcore.DrainObservation{Known: true, Connected: true}}
			}
			queue := NewFakeDrainQueue()
			for i, kind := range []string{"suite", "merge", "landing"} {
				queue.Enqueue(QueueActivity{ID: fmt.Sprint(i), Workspace: fmt.Sprintf("pair:%d", i), Kind: kind, State: "running"})
				if !queue.AdmitNext() {
					t.Fatal("fake queue did not admit initial work")
				}
			}
			coordinator, err := newDrainCoordinator(ctx, drainCoordinatorOptions{Broker: broker, Queue: queue, Store: newDrainStore(namespace), Now: time.Now, Snapshot: fleet.snapshot, Endpoint: func(binding couchmessage.Binding) couchmessage.DrainEndpoint { return remotes[binding] }})
			if err != nil {
				t.Fatal(err)
			}
			defer coordinator.close()
			board, err := coordinator.start(ctx, "00000000-0000-4000-8000-000000000001", "headless transport verification", fast, time.Minute, fleet.snapshot()["pair:0"].Binding)
			if err != nil {
				t.Fatal(err)
			}
			waitDrainService(t, coordinator, func(b couchcore.DrainBoard) bool {
				for _, r := range b.Rows {
					if r.Reason != "active turn" {
						return false
					}
				}
				return true
			})
			for workspace, member := range fleet.snapshot() {
				if _, err := coordinator.ack(ctx, board.ID, workspace, member.Identity()); err != nil {
					t.Fatal(err)
				}
			}
			fleet.idle()
			coordinator.reconcile(ctx)
			if coordinator.view().State == couchcore.DrainDrained {
				t.Fatal("drained while queue still owns work")
			}
			for i := 0; i < 3; i++ {
				queue.Finish(fmt.Sprint(i))
			}
			waitDrainService(t, coordinator, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainDrained })
			// Status is a cached immutable projection: it does not touch the sockets.
			func() {
				coordinator.pass.Lock()
				defer coordinator.pass.Unlock()
				before := 0
				for _, wrapper := range wrappers {
					before += wrapper.callCount()
				}
				for i := 0; i < 25; i++ {
					copy := coordinator.view()
					if copy.State != couchcore.DrainDrained {
						t.Fatal(copy)
					}
				}
				after := 0
				for _, wrapper := range wrappers {
					after += wrapper.callCount()
				}
				if after != before {
					t.Fatal("status initiated endpoint probes")
				}
			}()
			for binding, wrapper := range wrappers {
				wrapper.endpoint.mu.Lock()
				notices, interrupts := wrapper.endpoint.notices, wrapper.endpoint.interrupts
				wrapper.endpoint.mu.Unlock()
				expectedInterrupts := 0
				if fast {
					expectedInterrupts = 1
				}
				if notices != 1 || interrupts != expectedInterrupts {
					t.Fatalf("%s input counts: notices=%d interrupts=%d", binding.Slot, notices, interrupts)
				}
				wrapper.mu.Lock()
				calls := make(map[string]int)
				for key, value := range wrapper.calls {
					calls[key] = value
				}
				order := append([]string(nil), wrapper.order...)
				wrapper.mu.Unlock()
				if calls["observe"] == 0 || calls["drain-reserve"] != 1 || calls["commit"] != 1 || calls["drain-observe"] == 0 {
					t.Fatalf("transport bypassed for %s: %v", binding.Slot, calls)
				}
				if fast && (len(order) != 2 || order[0] != "interrupt" || order[1] != "notice") {
					t.Fatalf("notice preceded original-turn interrupt on %s: %v", binding.Slot, order)
				}
				if fast && binding.Slot == "pair:1" && calls["drain-interrupt-status"] == 0 {
					t.Fatal("lost response not reconciled via retained status")
				}
			}
			_ = queue.Pause(ctx, "other-owner", "manual", false)
			if _, err := coordinator.cancel(ctx); err != nil {
				t.Fatal(err)
			}
			waitDrainService(t, coordinator, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainCancelled })
			final, err := queue.Status(ctx)
			if err != nil || len(final.PauseOwners) != 1 || final.PauseOwners[0].Owner != "other-owner" {
				t.Fatal("cancel released another owner", final, err)
			}
			// Reuse the exact wrapper incarnation for a second epoch, switch
			// mode, then recover the supervisor before owner-only cancellation.
			fleet.mu.Lock()
			for workspace, member := range fleet.members {
				member.Observation.Idle = false
				fleet.members[workspace] = member
				endpoint := fleet.endpoints[member.Binding]
				endpoint.mu.Lock()
				endpoint.idle = false
				endpoint.mu.Unlock()
			}
			fleet.mu.Unlock()
			secondID := "00000000-0000-4000-8000-000000000002"
			if _, err := coordinator.start(ctx, secondID, "second transport epoch", !fast, time.Minute, fleet.snapshot()["pair:0"].Binding); err != nil {
				t.Fatal(err)
			}
			waitDrainService(t, coordinator, func(b couchcore.DrainBoard) bool {
				for _, row := range b.Rows {
					if row.Reason != "active turn" {
						return false
					}
				}
				return true
			})
			if drainServiceRecordEpoch(t, coordinator) != 2 {
				t.Fatal("transport did not allocate second epoch")
			}
			coordinator.close()
			recoveredBroker := couchmessage.NewBroker(ctx, time.Now, nil)
			defer recoveredBroker.Close()
			for binding, endpoint := range remotes {
				if err := recoveredBroker.Register(binding, endpoint); err != nil {
					t.Fatal(err)
				}
			}
			recovered, err := newDrainCoordinator(ctx, drainCoordinatorOptions{Broker: recoveredBroker, Queue: queue, Store: newDrainStore(namespace), Now: time.Now, Snapshot: fleet.snapshot, Endpoint: func(binding couchmessage.Binding) couchmessage.DrainEndpoint { return remotes[binding] }})
			if err != nil {
				t.Fatal(err)
			}
			defer recovered.close()
			if recovered.view().State != couchcore.DrainStuck || drainServiceRecordEpoch(t, recovered) != 2 {
				t.Fatal("transport recovery changed epoch or reused live completion")
			}
			if _, err := recovered.cancel(ctx); err != nil {
				t.Fatal(err)
			}
			waitDrainService(t, recovered, func(b couchcore.DrainBoard) bool { return b.State == couchcore.DrainCancelled })
			for binding, wrapper := range wrappers {
				wrapper.endpoint.mu.Lock()
				notices, interrupts := wrapper.endpoint.notices, wrapper.endpoint.interrupts
				residue := len(wrapper.endpoint.receipts) + len(wrapper.endpoint.results) + len(wrapper.endpoint.revoked)
				watermark := wrapper.endpoint.authority.View()
				wrapper.endpoint.mu.Unlock()
				if notices != 2 || interrupts != 1 || residue > 4 || watermark.Epoch != 2 {
					t.Fatalf("%s repeated transport residue/input: notices%d interrupts%d residue%d watermark%+v", binding.Slot, notices, interrupts, residue, watermark)
				}
			}
			final, err = queue.Status(ctx)
			if err != nil || len(final.PauseOwners) != 1 || final.PauseOwners[0].Owner != "other-owner" {
				t.Fatal("recovered cancellation released another queue owner", final, err)
			}

		})
	}
}
