package couchcmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type hubQuietEndpoint struct {
	serviceEndpointFake
	at time.Time
}

func (e hubQuietEndpoint) Observe(ctx context.Context) (couchmessage.Observation, error) {
	return couchmessage.Observation{LastActivity: e.at, Sequence: 2}, ctx.Err()
}

// Construct the service with its real ownership callback installed, then admit
// wrappers over the existing registry/session sockets. No process-global stores.
func hubMessageRig(t *testing.T) (*serviceRig, []couchmessage.Binding) {
	t.Helper()
	return hubMessageRigWithWorkerEndpoint(t, nil)
}

func hubMessageRigWithWorkerEndpoint(t *testing.T, endpoint couchmessage.DeliveryEndpoint) (*serviceRig, []couchmessage.Binding) {
	t.Helper()
	r := newServiceRig(t)
	r.s.Close()
	r.world.hub = "/repo0"
	authority := r.world.authority()
	authority.hubs = guardHubState(t)
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	authority.now = func() time.Time { return time.Unix(0, clock.Load()) }
	var err error
	r.s, err = newMessageService(context.Background(), r.brokerSock, r.regSock, filepath.Dir(r.brokerSock), authority, r.panes, func(string) (couchcore.SlotGitStatus, bool) { return couchcore.SlotGitStatus{}, false }, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.s.Close)
	var slots []couchmessage.Binding
	at := time.Now().Add(-time.Hour)
	for n := 0; n < 4; n++ {
		b := r.world.add(n)
		number := n
		r.world.set(b, func(s *worldSlot) {
			s.workspace.PrimaryRoot = "/repo"
			s.workspace.Slot = &number
			s.workspace.FleetRoot = "/"
		})
		r.world.mu.Lock()
		r.world.endpoints[b] = hubQuietEndpoint{at: at}
		if n == 2 && endpoint != nil {
			r.world.endpoints[b] = endpoint
		}
		r.world.mu.Unlock()
		r.attach(b, couchmessage.PaneHandle(fmt.Sprintf("h%d", n)))
		r.wrapper(b, couchmessage.Observation{LastActivity: at, Sequence: 2})
		r.waitConnected(b, true)
		slots = append(slots, b)
	}

	// Every wrapper is admitted before advancing beyond the quiet interval.
	clock.Add(int64(time.Hour))
	return r, slots
}

func hubSend(t *testing.T, r *serviceRig, from couchmessage.Binding, id, target string) couchmessage.Response {
	t.Helper()
	req := callerRequest(from, "send")
	req.ID, req.Target, req.Body = id, target, "report"
	var response couchmessage.Response
	if err := couchmessage.Call(context.Background(), r.brokerSock, req, &response); err != nil {
		t.Fatal(err)
	}
	return response
}
func requireHubRecipient(t *testing.T, response couchmessage.Response, want couchmessage.Binding) {
	t.Helper()
	if response.Code != "accepted" || response.Receipt == nil || response.Receipt.Message.To != want {
		t.Fatalf("want %s: %+v", want.Slot, response)
	}
}

func TestHubMessagesSenderRelativeAndReassignment(t *testing.T) {
	r, b := hubMessageRig(t)
	requireHubRecipient(t, hubSend(t, r, b[2], "extra-report", "hub"), b[1])
	requireHubRecipient(t, hubSend(t, r, b[3], "main-report", "hub"), b[0])
	h := r.s.authority.hubs
	h.mu.Lock()
	next, err := h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}, Caller: "/repo1", Target: "/repo0", Revision: h.state.Revision()})
	if err == nil {
		h.state = next
	}
	h.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	requireHubRecipient(t, hubSend(t, r, b[2], "moved-report", "hub"), b[0])
	// Receipt evidence stays with the destination actually admitted before move.
	receipt, err := r.s.broker.Status(b[2], "extra-report")
	if err != nil || receipt.Message.To != b[1] {
		t.Fatalf("old receipt relabeled: %+v %v", receipt, err)
	}
}

func TestHubMessagesFamilyStaysOwnHubAndExactCrosses(t *testing.T) {
	r, b := hubMessageRig(t)
	// Main's lower numbered slot would win the family route without filtering.
	requireHubRecipient(t, hubSend(t, r, b[1], "family", "pair"), b[2])
	requireHubRecipient(t, hubSend(t, r, b[1], "cross", "pair:3"), b[3])
}

func TestHubMessagesStaleCallerRefusedForEveryRoute(t *testing.T) {
	for _, target := range []string{"hub", "pair", "pair:3"} {
		t.Run(target, func(t *testing.T) {
			r, b := hubMessageRig(t)
			r.world.set(b[2], func(s *worldSlot) { s.pidFile++ })
			got := hubSend(t, r, b[2], "stale", target)
			if got.Code == "accepted" || got.Receipt != nil {
				t.Fatalf("stale caller admitted: %+v", got)
			}
		})
	}
}

func TestHubMessagesUnavailableNamesOwnHub(t *testing.T) {
	r, b := hubMessageRig(t)
	r.s.broker.Disconnect(b[1])
	r.s.mu.Lock()
	delete(r.s.workspaces, b[1])
	r.s.mu.Unlock()
	got := hubSend(t, r, b[2], "offline", "hub")
	if got.Code != "unavailable" || !strings.Contains(got.Error, "/repo1") || strings.Contains(got.Error, "/repo0") {
		t.Fatalf("wrong unavailable hub diagnostic: %+v", got)
	}
}
