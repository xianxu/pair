package couchmessage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type batchEndpointUnderTest interface {
	DeliverBatch(context.Context, Message, func() ([]Message, error), func(Receipt)) (Receipt, error)
}

func TestEndpointBatchRequestShapes(t *testing.T) {
	m := endpointMessage()
	marshal := func(v any) json.RawMessage {
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	head := marshal(m)
	binding := marshal(m.To)
	for _, tc := range []struct {
		name, op string
		extra    string
		valid    bool
	}{
		{"prepare", "prepare-batch", `,"ID":"message","Message":` + string(head), true},
		{"status", "batch-status", `,"ID":"message"`, true},
		{"commit", "commit-batch", `,"ID":"message","Messages":[` + string(head) + `]`, true},
		{"prepare-members", "prepare-batch", `,"ID":"message","Message":` + string(head) + `,"Messages":[` + string(head) + `]`, false},
		{"status-head", "batch-status", `,"ID":"message","Message":` + string(head), false},
		{"commit-empty", "commit-batch", `,"ID":"message","Messages":[]`, false},
		{"commit-single-message", "commit-batch", `,"ID":"message","Message":` + string(head) + `,"Messages":[` + string(head) + `]`, false},
		{"legacy-members", "commit", `,"ID":"message","Message":` + string(head) + `,"Messages":[` + string(head) + `]`, false},
		{"wrong-head", "commit-batch", `,"ID":"other","Messages":[` + string(head) + `]`, false},
		{"duplicate-member", "commit-batch", `,"ID":"message","Messages":[` + string(head) + `,` + string(head) + `]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(`{"Op":` + string(marshal(tc.op)) + `,"Binding":` + string(binding) + tc.extra + `}`)
			var req EndpointRequest
			err := strictjson.Decode(raw, &req)
			if err == nil {
				err = ValidateEndpointRequest(req)
			}
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%t error=%v", tc.valid, err)
			}
		})
	}
}

// batchSocketRecipient owns preparation and finalization independently of the
// client's polling. Gates force arrivals before/after readiness without sleeps.
type batchSocketRecipient struct {
	mu                                    sync.Mutex
	head                                  Message
	prepared, ready, committed, confirmed bool
	prepares, commits, legacy             int
	members                               []Message
	mode                                  string
}

func (s *batchSocketRecipient) handle(_ context.Context, raw []byte) ([]byte, error) {
	var req struct {
		Op       string
		Binding  Binding
		ID       string
		Sequence uint64
		Message  *Message
		Messages []Message
	}
	if err := strictjson.Decode(raw, &req); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]any{}
	receipt := func(status Status) { out["Receipt"] = Receipt{Message: s.head, Status: status} }
	switch req.Op {
	case "prepare-batch":
		s.prepares++
		switch s.mode {
		case "legacy":
			out["Error"] = `unknown endpoint operation "prepare-batch"`
			return json.Marshal(out)
		case "near-legacy":
			out["Error"] = `unknown endpoint operation "prepare-batch": malformed`
			return json.Marshal(out)
		case "lost-prepare":
			return nil, errors.New("prepare response lost")
		case "bad-prepare-json":
			return []byte(`{"Receipt":`), nil
		}
		if s.prepared {
			return nil, errors.New("duplicate prepare")
		}
		s.prepared = true
		s.head = *req.Message
		if !s.head.AcceptedAt.IsZero() {
			return nil, errors.New("AcceptedAt leaked")
		}
		receipt(Held)
	case "batch-status":
		if !s.prepared {
			return nil, errors.New("no prepared owner")
		}
		receipt(Held)
		out["BatchReady"] = s.ready
		if s.mode == "wrong-receipt" {
			bad := s.head
			bad.Body = "changed"
			out["Receipt"] = Receipt{Message: bad, Status: Held}
		}
	case "commit-batch":
		s.commits++
		if !s.ready || s.committed {
			return nil, errors.New("unready or duplicate commit")
		}
		s.committed = true
		s.members = append([]Message(nil), req.Messages...)
		for _, m := range s.members {
			if !m.AcceptedAt.IsZero() {
				return nil, errors.New("AcceptedAt leaked")
			}
		}
		if s.mode == "lost-commit" {
			return nil, errors.New("commit response lost")
		}
		receipt(Delivering)
	case "commit":
		s.legacy++
		s.head = *req.Message
		receipt(Submitted)
	case "status":
		if !s.committed {
			return nil, errors.New("status without finalization")
		}
		if s.confirmed {
			receipt(Submitted)
		} else {
			receipt(Delivering)
		}
	default:
		return nil, fmt.Errorf("unexpected operation %s", req.Op)
	}
	return json.Marshal(out)
}
func startBatchSocket(t *testing.T, state *batchSocketRecipient) RemoteEndpoint {
	t.Helper()
	socket := transportSocket(t)
	server, err := StartServer(context.Background(), socket, state.handle)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return RemoteEndpoint{Binding: endpointMessage().To, socket: socket}
}
func batchMethod(t *testing.T, e RemoteEndpoint) batchEndpointUnderTest {
	t.Helper()
	b, ok := any(e).(batchEndpointUnderTest)
	if !ok {
		t.Fatal("RemoteEndpoint has no batch readiness/finalization seam")
	}
	return b
}
func TestRemoteEndpointBatchFreezesOnceAtReadiness(t *testing.T) {
	state := &batchSocketRecipient{}
	e := startBatchSocket(t, state)
	b := batchMethod(t, e)
	head := endpointMessage()
	head.AcceptedAt = time.Now()
	head.Deadline = time.Now().Add(time.Second)
	follower := head
	follower.ID = "second"
	follower.Body = "report two"
	originalMembers := []Message{head, follower}
	frozen := make(chan struct{})
	done := make(chan error, 1)
	calls := 0
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	go func() {
		r, err := b.DeliverBatch(ctx, head, func() ([]Message, error) { calls++; close(frozen); return originalMembers, nil }, func(r Receipt) {
			if r.Status == Held {
				expired, stop := context.WithDeadline(ctx, head.Deadline)
				<-expired.Done()
				stop()
				state.mu.Lock()
				state.ready = true
				state.mu.Unlock()
			}
			if r.Status == Delivering {
				state.mu.Lock()
				state.confirmed = true
				state.mu.Unlock()
			}
		})
		if err == nil && r.Status != Submitted {
			err = fmt.Errorf("outcome=%s", r.Status)
		}
		done <- err
	}()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	select {
	case <-frozen:
	default:
		t.Fatal("membership never frozen")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if originalMembers[0].AcceptedAt.IsZero() || originalMembers[1].AcceptedAt.IsZero() {
		t.Fatal("wire normalization mutated the frozen original messages")
	}
	if calls != 1 || state.prepares != 1 || state.commits != 1 || state.legacy != 0 || len(state.members) != 2 {
		t.Fatalf("calls=%d recipient=%+v", calls, state)
	}
}
func TestRemoteEndpointBatchNeverFallbacksOrReplaysUncertainExchange(t *testing.T) {
	for _, mode := range []string{"legacy", "near-legacy", "lost-prepare", "bad-prepare-json", "wrong-receipt", "lost-commit"} {
		t.Run(mode, func(t *testing.T) {
			state := &batchSocketRecipient{mode: mode, ready: true, confirmed: true}
			e := startBatchSocket(t, state)
			b := batchMethod(t, e)
			head := endpointMessage()
			freezes := 0
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			r, err := b.DeliverBatch(ctx, head, func() ([]Message, error) { freezes++; return []Message{head}, nil }, nil)
			state.mu.Lock()
			defer state.mu.Unlock()
			if mode == "legacy" {
				if err != nil || r.Status != Submitted || state.legacy != 1 || freezes != 0 {
					t.Fatalf("legacy r=%+v err=%v freezes=%d commits=%d", r, err, freezes, state.legacy)
				}
				return
			}
			if err == nil {
				t.Fatal("uncertain exchange accepted")
			}
			if state.legacy != 0 || state.prepares != 1 || state.commits > 1 || freezes > 1 {
				t.Fatalf("fallback/replay: recipient=%+v freezes=%d", state, freezes)
			}
		})
	}
}
func TestRemoteEndpointBatchFreezeErrorDoesNotCommit(t *testing.T) {
	state := &batchSocketRecipient{ready: true}
	b := batchMethod(t, startBatchSocket(t, state))
	sentinel := errors.New("head cancelled")
	_, err := b.DeliverBatch(context.Background(), endpointMessage(), func() ([]Message, error) { return nil, sentinel }, nil)
	if !errors.Is(err, sentinel) {
		t.Fatalf("got %v", err)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.commits != 0 {
		t.Fatal("committed after freeze failed")
	}
}

func TestEndpointBatchEmptyMembersRejectedOnLegacyOperations(t *testing.T) {
	m := endpointMessage()
	for _, op := range []string{"observe", "reserve", "status", "commit", "prepare-batch", "batch-status"} {
		req := EndpointRequest{Op: op, Binding: m.To, Messages: []Message{}}
		if op != "observe" {
			req.ID = m.ID
		}
		if op == "commit" || op == "prepare-batch" {
			req.Message = &m
		}
		if err := ValidateEndpointRequest(req); err == nil {
			t.Fatalf("%s accepted unrelated Messages field", op)
		}
	}
}
func TestRemoteEndpointBatchTransportTextCannotAuthorizeFallback(t *testing.T) {
	calls := 0
	e := RemoteEndpoint{Binding: endpointMessage().To, socket: transportSocket(t), transport: func(context.Context, string, any, any) error {
		calls++
		return errors.New(`unknown endpoint operation "prepare-batch"`)
	}}
	freezes := 0
	_, err := e.DeliverBatch(context.Background(), endpointMessage(), func() ([]Message, error) { freezes++; return nil, nil }, nil)
	if err == nil || calls != 1 || freezes != 0 {
		t.Fatalf("err=%v calls=%d freezes=%d", err, calls, freezes)
	}
}
func TestRemoteEndpointBatchRejectsChangedFrozenHead(t *testing.T) {
	for _, change := range []string{"body", "deadline", "binding", "empty"} {
		t.Run(change, func(t *testing.T) {
			state := &batchSocketRecipient{ready: true}
			b := batchMethod(t, startBatchSocket(t, state))
			head := endpointMessage()
			changed := head
			switch change {
			case "body":
				changed.Body = "changed"
			case "deadline":
				changed.Deadline = changed.Deadline.Add(time.Second)
			case "binding":
				changed.From.Nonce = "changed"
			}
			_, err := b.DeliverBatch(context.Background(), head, func() ([]Message, error) {
				if change == "empty" {
					return nil, nil
				}
				return []Message{changed}, nil
			}, nil)
			if err == nil {
				t.Fatal("changed head accepted")
			}
			state.mu.Lock()
			defer state.mu.Unlock()
			if state.commits != 0 {
				t.Fatal("changed head committed")
			}
		})
	}
}
func TestRemoteEndpointBatchHeldCancellationStopsBeforeFreeze(t *testing.T) {
	state := &batchSocketRecipient{}
	b := batchMethod(t, startBatchSocket(t, state))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	freezes := 0
	_, err := b.DeliverBatch(ctx, endpointMessage(), func() ([]Message, error) { freezes++; return nil, nil }, func(r Receipt) {
		if r.Status == Held {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) || freezes != 0 {
		t.Fatalf("err=%v freezes=%d", err, freezes)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.commits != 0 {
		t.Fatal("cancelled held message committed")
	}
}
func TestEndpointBatchFieldsOmittedFromLegacyWire(t *testing.T) {
	m := endpointMessage()
	req := EndpointRequest{Op: "commit", Binding: m.To, ID: m.ID, Message: &m}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var legacyReq struct {
		Drive    *DriveStart
		Op       string
		Binding  Binding
		ID       string
		Sequence uint64
		Message  *Message
		Lines    int
	}
	if err := strictjson.Decode(raw, &legacyReq); err != nil {
		t.Fatalf("old wrapper rejects request: %v", err)
	}
	raw, err = json.Marshal(EndpointResponse{Receipt: &Receipt{Message: m, Status: Held}})
	if err != nil {
		t.Fatal(err)
	}
	var legacyResp struct {
		Error       string
		Observation Observation
		Receipt     *Receipt
	}
	if err := strictjson.Decode(raw, &legacyResp); err != nil {
		t.Fatalf("old supervisor rejects reply: %v", err)
	}
}
func FuzzEndpointBatchPayloadIsolation(f *testing.F) {
	for _, op := range []string{"commit-batch", "prepare-batch", "batch-status", "commit", "status", "reserve", "release", "tail", "observe", "drive-start", "unknown"} {
		f.Add(op, false)
		f.Add(op, true)
	}
	f.Fuzz(func(t *testing.T, op string, empty bool) {
		m := endpointMessage()
		members := []Message{m}
		if empty {
			members = []Message{}
		}
		req := EndpointRequest{Op: op, Binding: m.To, ID: m.ID, Messages: members}
		err := ValidateEndpointRequest(req)
		if err == nil && (op != "commit-batch" || empty) {
			t.Fatalf("unrelated/empty members admitted for %q", op)
		}
	})
}
